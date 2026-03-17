package skill

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pgvector/pgvector-go"
)

type postgresSkillStore struct {
	db *sql.DB
}

// NewPostgresSkillStore creates a new PostgreSQL-backed skill store.
// It re-uses an existing *sql.DB connection (shared with memory store).
func NewPostgresSkillStore(db *sql.DB) (SkillService, error) {
	s := &postgresSkillStore{db: db}
	if err := s.init(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *postgresSkillStore) init() error {
	query := `
	CREATE TABLE IF NOT EXISTS skills (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		description TEXT,
		category TEXT DEFAULT 'uncategorized',
		source TEXT DEFAULT 'local',
		source_repo TEXT,
		risk TEXT DEFAULT 'unknown',
		version TEXT,
		tags JSONB DEFAULT '[]'::jsonb,
		metadata JSONB DEFAULT '{}'::jsonb,
		created_at TIMESTAMP,
		updated_at TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_skills_name ON skills(name);
	CREATE INDEX IF NOT EXISTS idx_skills_category ON skills(category);
	CREATE INDEX IF NOT EXISTS idx_skills_source ON skills(source);

	CREATE TABLE IF NOT EXISTS skill_chunks (
		id TEXT PRIMARY KEY,
		skill_id TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
		section_id TEXT,
		chunk_type TEXT DEFAULT 'rule',
		title TEXT,
		content TEXT NOT NULL,
		chunk_index INTEGER,
		content_hash TEXT,
		vector vector(1024),
		created_at TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_skill_chunks_skill_id ON skill_chunks(skill_id);
	CREATE INDEX IF NOT EXISTS idx_skill_chunks_hash ON skill_chunks(content_hash);

	CREATE TABLE IF NOT EXISTS skill_files (
		id TEXT PRIMARY KEY,
		skill_id TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
		rel_path TEXT NOT NULL,
		content_type TEXT NOT NULL DEFAULT 'application/octet-stream',
		content BYTEA NOT NULL,
		content_hash TEXT NOT NULL,
		size_bytes INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		UNIQUE(skill_id, rel_path)
	);
	CREATE INDEX IF NOT EXISTS idx_skill_files_skill_id ON skill_files(skill_id);
	`
	if _, err := s.db.Exec(query); err != nil {
		return err
	}

	// Idempotent migration: add section_id to existing skill_chunks tables
	migrations := []string{
		`ALTER TABLE skill_chunks ADD COLUMN IF NOT EXISTS section_id TEXT`,
		`CREATE INDEX IF NOT EXISTS idx_skill_chunks_section_id ON skill_chunks(section_id)`,
	}
	for _, m := range migrations {
		if _, err := s.db.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}

func (s *postgresSkillStore) UpsertSkill(ctx context.Context, sk *Skill) error {
	tagsJSON, _ := json.Marshal(sk.Tags)
	metaJSON, _ := json.Marshal(sk.Metadata)

	query := `
	INSERT INTO skills (id, name, description, category, source, source_repo, risk, version, tags, metadata, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	ON CONFLICT (name) DO UPDATE SET
		description = EXCLUDED.description,
		category = EXCLUDED.category,
		source = EXCLUDED.source,
		source_repo = EXCLUDED.source_repo,
		risk = EXCLUDED.risk,
		version = EXCLUDED.version,
		tags = EXCLUDED.tags,
		metadata = EXCLUDED.metadata,
		updated_at = EXCLUDED.updated_at
	`
	_, err := s.db.ExecContext(ctx, query,
		sk.ID, sk.Name, sk.Description, sk.Category, sk.Source, sk.SourceRepo,
		sk.Risk, sk.Version, string(tagsJSON), string(metaJSON),
		sk.CreatedAt, sk.UpdatedAt,
	)
	return err
}

func (s *postgresSkillStore) StoreChunk(ctx context.Context, c *SkillChunk) error {
	vec := pgvector.NewVector(c.Vector)

	query := `
	INSERT INTO skill_chunks (id, skill_id, section_id, chunk_type, title, content, chunk_index, content_hash, vector, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	ON CONFLICT (id) DO UPDATE SET
		section_id = EXCLUDED.section_id,
		content = EXCLUDED.content,
		chunk_type = EXCLUDED.chunk_type,
		title = EXCLUDED.title,
		chunk_index = EXCLUDED.chunk_index,
		content_hash = EXCLUDED.content_hash,
		vector = EXCLUDED.vector
	`
	_, err := s.db.ExecContext(ctx, query,
		c.ID, c.SkillID, c.SectionID, c.ChunkType, c.Title, c.Content, c.ChunkIndex, c.ContentHash, vec, c.CreatedAt,
	)
	return err
}

func (s *postgresSkillStore) GetSkill(ctx context.Context, name string) (*Skill, []SkillChunk, error) {
	var sk Skill
	var tagsStr, metaStr []byte

	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, category, source, source_repo, risk, version, tags, metadata, created_at, updated_at FROM skills WHERE name = $1`, name,
	).Scan(&sk.ID, &sk.Name, &sk.Description, &sk.Category, &sk.Source, &sk.SourceRepo,
		&sk.Risk, &sk.Version, &tagsStr, &metaStr, &sk.CreatedAt, &sk.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, fmt.Errorf("skill %q not found", name)
		}
		return nil, nil, err
	}

	json.Unmarshal(tagsStr, &sk.Tags)
	json.Unmarshal(metaStr, &sk.Metadata)

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, skill_id, section_id, chunk_type, title, content, chunk_index, content_hash, created_at FROM skill_chunks WHERE skill_id = $1 ORDER BY chunk_index`, sk.ID,
	)
	if err != nil {
		return &sk, nil, err
	}
	defer rows.Close()

	var chunks []SkillChunk
	for rows.Next() {
		var c SkillChunk
		if err := rows.Scan(&c.ID, &c.SkillID, &c.SectionID, &c.ChunkType, &c.Title, &c.Content, &c.ChunkIndex, &c.ContentHash, &c.CreatedAt); err != nil {
			return &sk, nil, err
		}
		chunks = append(chunks, c)
	}

	return &sk, chunks, nil
}

func (s *postgresSkillStore) ListSkills(ctx context.Context, source string, category string, limit int, offset int) ([]Skill, error) {
	sqlQuery := `SELECT s.id, s.name, s.description, s.category, s.source, s.source_repo, s.risk, s.version, s.tags, s.metadata, s.created_at, s.updated_at,
		(SELECT COUNT(*) FROM skill_chunks sc WHERE sc.skill_id = s.id) as chunk_count
		FROM skills s`

	var args []interface{}
	argCount := 1
	var conditions []string

	if source != "" {
		conditions = append(conditions, fmt.Sprintf("s.source = $%d", argCount))
		args = append(args, source)
		argCount++
	}
	if category != "" {
		conditions = append(conditions, fmt.Sprintf("s.category = $%d", argCount))
		args = append(args, category)
		argCount++
	}

	if len(conditions) > 0 {
		sqlQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	sqlQuery += " ORDER BY s.name"

	if limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, limit)
		argCount++
	}
	if offset > 0 {
		sqlQuery += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)
	}

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []Skill
	for rows.Next() {
		var sk Skill
		var tagsStr, metaStr []byte
		if err := rows.Scan(&sk.ID, &sk.Name, &sk.Description, &sk.Category, &sk.Source, &sk.SourceRepo,
			&sk.Risk, &sk.Version, &tagsStr, &metaStr, &sk.CreatedAt, &sk.UpdatedAt, &sk.ChunkCount); err != nil {
			return nil, err
		}
		json.Unmarshal(tagsStr, &sk.Tags)
		json.Unmarshal(metaStr, &sk.Metadata)
		skills = append(skills, sk)
	}

	return skills, nil
}

func (s *postgresSkillStore) DeleteSkill(ctx context.Context, name string) error {
	// CASCADE will delete chunks
	_, err := s.db.ExecContext(ctx, `DELETE FROM skills WHERE name = $1`, name)
	return err
}

func (s *postgresSkillStore) SearchChunks(ctx context.Context, queryVector []float32, limit int) ([]SkillChunkResult, error) {
	vec := pgvector.NewVector(queryVector)

	query := `
		SELECT sc.id, sc.skill_id, sc.section_id, sc.chunk_type, sc.title, sc.content, sc.chunk_index, sc.content_hash, sc.created_at,
		       1 - (sc.vector <=> $1) AS score
		FROM skill_chunks sc
		ORDER BY sc.vector <=> $1
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, vec, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SkillChunkResult
	for rows.Next() {
		var r SkillChunkResult
		if err := rows.Scan(&r.ID, &r.SkillID, &r.SectionID, &r.ChunkType, &r.Title, &r.Content, &r.ChunkIndex, &r.ContentHash, &r.CreatedAt, &r.Score); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, nil
}

func (s *postgresSkillStore) DeleteChunksBySkillID(ctx context.Context, skillID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM skill_chunks WHERE skill_id = $1`, skillID)
	return err
}

// GetSkillByID fetches a single skill record by its internal ID (no chunks).
// Used by SearchSkills to avoid the N×ListSkills anti-pattern.
func (s *postgresSkillStore) GetSkillByID(ctx context.Context, id string) (*Skill, error) {
	var sk Skill
	var tagsStr, metaStr []byte

	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, category, source, source_repo, risk, version, tags, metadata, created_at, updated_at FROM skills WHERE id = $1`, id,
	).Scan(&sk.ID, &sk.Name, &sk.Description, &sk.Category, &sk.Source, &sk.SourceRepo,
		&sk.Risk, &sk.Version, &tagsStr, &metaStr, &sk.CreatedAt, &sk.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(tagsStr, &sk.Tags)
	json.Unmarshal(metaStr, &sk.Metadata)
	return &sk, nil
}

// GetAllChunksBySkillID returns every chunk for a skill sorted by chunk_index.
// Used for high-confidence retrieval so the agent receives complete skill content.
func (s *postgresSkillStore) GetAllChunksBySkillID(ctx context.Context, skillID string) ([]SkillChunk, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, skill_id, section_id, chunk_type, title, content, chunk_index, content_hash, created_at
		 FROM skill_chunks WHERE skill_id = $1 ORDER BY chunk_index`, skillID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []SkillChunk
	for rows.Next() {
		var c SkillChunk
		if err := rows.Scan(&c.ID, &c.SkillID, &c.SectionID, &c.ChunkType, &c.Title, &c.Content, &c.ChunkIndex, &c.ContentHash, &c.CreatedAt); err != nil {
			return nil, err
		}
		chunks = append(chunks, c)
	}
	return chunks, nil
}

// GetChunksBySectionID returns all chunks that share a section_id, sorted by chunk_index.
// Used for medium-confidence retrieval to reassemble split sections into complete context.
func (s *postgresSkillStore) GetChunksBySectionID(ctx context.Context, sectionID string) ([]SkillChunk, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, skill_id, section_id, chunk_type, title, content, chunk_index, content_hash, created_at
		 FROM skill_chunks WHERE section_id = $1 ORDER BY chunk_index`, sectionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []SkillChunk
	for rows.Next() {
		var c SkillChunk
		if err := rows.Scan(&c.ID, &c.SkillID, &c.SectionID, &c.ChunkType, &c.Title, &c.Content, &c.ChunkIndex, &c.ContentHash, &c.CreatedAt); err != nil {
			return nil, err
		}
		chunks = append(chunks, c)
	}
	return chunks, nil
}

// GetAdjacentChunks returns chunks within ±radius of chunkIndex for a given skill.
// Used for medium-confidence retrieval to include neighboring context around a matched chunk.
func (s *postgresSkillStore) GetAdjacentChunks(ctx context.Context, skillID string, chunkIndex int, radius int) ([]SkillChunk, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, skill_id, section_id, chunk_type, title, content, chunk_index, content_hash, created_at
		 FROM skill_chunks
		 WHERE skill_id = $1 AND chunk_index BETWEEN $2 AND $3
		 ORDER BY chunk_index`,
		skillID, chunkIndex-radius, chunkIndex+radius,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []SkillChunk
	for rows.Next() {
		var c SkillChunk
		if err := rows.Scan(&c.ID, &c.SkillID, &c.SectionID, &c.ChunkType, &c.Title, &c.Content, &c.ChunkIndex, &c.ContentHash, &c.CreatedAt); err != nil {
			return nil, err
		}
		chunks = append(chunks, c)
	}
	return chunks, nil
}

func (s *postgresSkillStore) Close() error {
	return nil // DB connection is shared, don't close here
}

func (s *postgresSkillStore) StoreFile(ctx context.Context, f *SkillFile) error {
	query := `
	INSERT INTO skill_files (id, skill_id, rel_path, content_type, content, content_hash, size_bytes, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	ON CONFLICT (skill_id, rel_path) DO UPDATE SET
		content_type = EXCLUDED.content_type,
		content      = EXCLUDED.content,
		content_hash = EXCLUDED.content_hash,
		size_bytes   = EXCLUDED.size_bytes,
		created_at   = EXCLUDED.created_at
	`
	_, err := s.db.ExecContext(ctx, query,
		f.ID, f.SkillID, f.RelPath, f.ContentType, f.Content, f.ContentHash, f.SizeBytes, f.CreatedAt,
	)
	return err
}

func (s *postgresSkillStore) GetFiles(ctx context.Context, skillID string) ([]SkillFile, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, skill_id, rel_path, content_type, content, content_hash, size_bytes, created_at
		 FROM skill_files WHERE skill_id = $1 ORDER BY rel_path`, skillID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []SkillFile
	for rows.Next() {
		var f SkillFile
		if err := rows.Scan(&f.ID, &f.SkillID, &f.RelPath, &f.ContentType, &f.Content, &f.ContentHash, &f.SizeBytes, &f.CreatedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}

func (s *postgresSkillStore) DeleteFilesBySkillID(ctx context.Context, skillID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM skill_files WHERE skill_id = $1`, skillID)
	return err
}

func (s *postgresSkillStore) GetChunkHashes(ctx context.Context, skillID string) (map[string]bool, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT content_hash FROM skill_chunks WHERE skill_id = $1`, skillID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hashes := make(map[string]bool)
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			return nil, err
		}
		hashes[hash] = true
	}
	return hashes, nil
}
