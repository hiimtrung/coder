package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
)

type postgresMemory struct {
	db *sql.DB
}

func NewPostgresMemory(dsn string) (MemoryService, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	p := &postgresMemory{db: db}
	if err := p.init(); err != nil {
		db.Close()
		return nil, err
	}

	return p, nil
}

// NewPostgresMemoryWithDB creates a new PostgresMemory and also returns the raw *sql.DB
// for shared use by other services (e.g., skill store).
func NewPostgresMemoryWithDB(dsn string) (MemoryService, *sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, nil, err
	}

	p := &postgresMemory{db: db}
	if err := p.init(); err != nil {
		db.Close()
		return nil, nil, err
	}

	return p, db, nil
}

func (p *postgresMemory) init() error {
	_, err := p.db.Exec(`CREATE EXTENSION IF NOT EXISTS vector;`)
	if err != nil {
		return fmt.Errorf("failed to create vector extension: %w", err)
	}

	query := `
	CREATE TABLE IF NOT EXISTS knowledge (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		type VARCHAR(50) DEFAULT 'document',
		metadata JSONB DEFAULT '{}'::jsonb,
		tags JSONB,
		scope TEXT,
		parent_id TEXT,
		chunk_index INTEGER,
		normalized_title TEXT,
		content_hash TEXT,
		vector vector(1024),
		created_at TIMESTAMP,
		updated_at TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_knowledge_scope ON knowledge(scope);
	CREATE INDEX IF NOT EXISTS idx_knowledge_hash ON knowledge(content_hash);
	CREATE INDEX IF NOT EXISTS idx_knowledge_parent ON knowledge(parent_id);
	CREATE INDEX IF NOT EXISTS idx_knowledge_metadata ON knowledge USING GIN (metadata);
	`
	_, err = p.db.Exec(query)
	return err
}

func (p *postgresMemory) Store(ctx context.Context, k *Knowledge) error {
	tagsJSON, err := json.Marshal(k.Tags)
	if err != nil {
		return err
	}

	metaJSON, _ := json.Marshal(k.Metadata)

	vec := pgvector.NewVector(k.Vector)

	query := `
	INSERT INTO knowledge (id, title, content, type, metadata, tags, scope, parent_id, chunk_index, normalized_title, content_hash, vector, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	ON CONFLICT (id) DO UPDATE SET
		title = EXCLUDED.title,
		content = EXCLUDED.content,
		type = EXCLUDED.type,
		metadata = EXCLUDED.metadata,
		tags = EXCLUDED.tags,
		scope = EXCLUDED.scope,
		parent_id = EXCLUDED.parent_id,
		chunk_index = EXCLUDED.chunk_index,
		normalized_title = EXCLUDED.normalized_title,
		content_hash = EXCLUDED.content_hash,
		vector = EXCLUDED.vector,
		updated_at = EXCLUDED.updated_at
	`
	_, err = p.db.ExecContext(ctx, query,
		k.ID, k.Title, k.Content, string(k.Type), string(metaJSON), string(tagsJSON), k.Scope, k.ParentID, k.ChunkIndex,
		k.NormalizedTitle, k.ContentHash, vec, k.CreatedAt, k.UpdatedAt,
	)
	return err
}

func (p *postgresMemory) Search(ctx context.Context, queryVector []float32, scope string, tags []string, memType MemoryType, metaFilters map[string]interface{}, limit int) ([]SearchResult, error) {
	vec := pgvector.NewVector(queryVector)

	sqlQuery := `
		SELECT id, title, content, type, metadata, tags, scope, parent_id, chunk_index, normalized_title, content_hash, created_at, updated_at,
		       1 - (vector <=> $1) AS score
		FROM knowledge
	`
	var args []interface{}
	args = append(args, vec)
	argCounts := 2

	var conditions []string

	if scope != "" {
		conditions = append(conditions, fmt.Sprintf("scope = $%d", argCounts))
		args = append(args, scope)
		argCounts++
	}

	if string(memType) != "" {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argCounts))
		args = append(args, string(memType))
		argCounts++
	}

	if len(tags) > 0 {
		tagsJSON, _ := json.Marshal(tags)
		conditions = append(conditions, fmt.Sprintf("tags @> $%d", argCounts))
		args = append(args, string(tagsJSON))
		argCounts++
	}

	if len(metaFilters) > 0 {
		metaJSON, _ := json.Marshal(metaFilters)
		conditions = append(conditions, fmt.Sprintf("metadata @> $%d", argCounts))
		args = append(args, string(metaJSON))
		argCounts++
	}

	if len(conditions) > 0 {
		sqlQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	sqlQuery += fmt.Sprintf(" ORDER BY vector <=> $1 LIMIT $%d", argCounts)
	args = append(args, limit)

	rows, err := p.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var sr SearchResult
		var tagsStr []byte
		var metaStr []byte
		var mType string
		if err := rows.Scan(
			&sr.ID, &sr.Title, &sr.Content, &mType, &metaStr, &tagsStr, &sr.Scope, &sr.ParentID,
			&sr.ChunkIndex, &sr.NormalizedTitle, &sr.ContentHash, &sr.CreatedAt, &sr.UpdatedAt,
			&sr.Score,
		); err != nil {
			return nil, err
		}

		sr.Type = MemoryType(mType)
		if len(tagsStr) > 0 {
			json.Unmarshal(tagsStr, &sr.Tags)
		}
		if len(metaStr) > 0 {
			json.Unmarshal(metaStr, &sr.Metadata)
		}
		results = append(results, sr)
	}

	return results, nil
}

func (p *postgresMemory) List(ctx context.Context, limit int, offset int) ([]Knowledge, error) {
	query := `SELECT id, title, content, type, metadata, tags, scope, parent_id, chunk_index, normalized_title, content_hash, created_at, updated_at FROM knowledge LIMIT $1 OFFSET $2`
	rows, err := p.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Knowledge
	for rows.Next() {
		var k Knowledge
		var tagsStr []byte
		var metaStr []byte
		var mType string
		if err := rows.Scan(
			&k.ID, &k.Title, &k.Content, &mType, &metaStr, &tagsStr, &k.Scope, &k.ParentID,
			&k.ChunkIndex, &k.NormalizedTitle, &k.ContentHash, &k.CreatedAt, &k.UpdatedAt,
		); err != nil {
			return nil, err
		}

		k.Type = MemoryType(mType)
		if len(tagsStr) > 0 {
			json.Unmarshal(tagsStr, &k.Tags)
		}
		if len(metaStr) > 0 {
			json.Unmarshal(metaStr, &k.Metadata)
		}
		results = append(results, k)
	}
	return results, nil
}

func (p *postgresMemory) Delete(ctx context.Context, id string) error {
	query := "DELETE FROM knowledge WHERE id = $1 OR id LIKE $2"
	_, err := p.db.ExecContext(ctx, query, id, id+"%")
	return err
}

func (p *postgresMemory) Close() error {
	return p.db.Close()
}
