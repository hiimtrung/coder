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
	if _, err = p.db.Exec(query); err != nil {
		return err
	}

	// Idempotent migrations: add full-text search column and index for hybrid search.
	migrations := []string{
		// Generated stored tsvector column: concatenates title + content for full-text indexing.
		// GENERATED ALWAYS AS ... STORED is supported in PostgreSQL 12+.
		`ALTER TABLE knowledge ADD COLUMN IF NOT EXISTS fts tsvector
		 GENERATED ALWAYS AS (
		     to_tsvector('english', coalesce(title, '') || ' ' || coalesce(content, ''))
		 ) STORED`,
		// GIN index for fast full-text search.
		`CREATE INDEX IF NOT EXISTS idx_knowledge_fts ON knowledge USING GIN(fts)`,
	}
	for _, m := range migrations {
		if _, err := p.db.Exec(m); err != nil {
			return fmt.Errorf("memory migration failed: %w", err)
		}
	}
	return nil
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

// Search performs hybrid search combining pgvector cosine similarity with
// PostgreSQL full-text search, fused via Reciprocal Rank Fusion (RRF).
//
// When queryText is non-empty, both semantic and keyword candidate lists are
// built independently, then merged with:
//
//	rrf_score = 1/(k + semantic_rank) + 1/(k + keyword_rank)   where k = 60
//
// When queryText is empty the method falls back to pure semantic search.
func (p *postgresMemory) Search(ctx context.Context, queryVector []float32, queryText string, scope string, tags []string, memType MemoryType, metaFilters map[string]interface{}, limit int) ([]SearchResult, error) {
	if strings.TrimSpace(queryText) != "" {
		return p.hybridSearch(ctx, queryVector, queryText, scope, tags, memType, metaFilters, limit)
	}
	return p.semanticSearch(ctx, queryVector, scope, tags, memType, metaFilters, limit)
}

// semanticSearch is the original pure-vector search (used as fallback when
// no query text is available or FTS adds no value).
func (p *postgresMemory) semanticSearch(ctx context.Context, queryVector []float32, scope string, tags []string, memType MemoryType, metaFilters map[string]interface{}, limit int) ([]SearchResult, error) {
	vec := pgvector.NewVector(queryVector)

	sqlQuery := `
		SELECT id, title, content, type, metadata, tags, scope, parent_id, chunk_index,
		       normalized_title, content_hash, created_at, updated_at,
		       1 - (vector <=> $1) AS score
		FROM knowledge
	`
	var args []interface{}
	args = append(args, vec)
	argIdx := 2

	var conditions []string
	if scope != "" {
		conditions = append(conditions, fmt.Sprintf("scope = $%d", argIdx))
		args = append(args, scope)
		argIdx++
	}
	if string(memType) != "" {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, string(memType))
		argIdx++
	}
	if len(tags) > 0 {
		tagsJSON, _ := json.Marshal(tags)
		conditions = append(conditions, fmt.Sprintf("tags @> $%d", argIdx))
		args = append(args, string(tagsJSON))
		argIdx++
	}
	if len(metaFilters) > 0 {
		metaJSON, _ := json.Marshal(metaFilters)
		conditions = append(conditions, fmt.Sprintf("metadata @> $%d", argIdx))
		args = append(args, string(metaJSON))
		argIdx++
	}
	if len(conditions) > 0 {
		sqlQuery += " WHERE " + strings.Join(conditions, " AND ")
	}
	sqlQuery += fmt.Sprintf(" ORDER BY vector <=> $1 LIMIT $%d", argIdx)
	args = append(args, limit)

	return p.runSearchQuery(ctx, sqlQuery, args)
}

// hybridSearch uses Reciprocal Rank Fusion (RRF) to blend pgvector cosine
// similarity ranks with PostgreSQL full-text search ranks.
//
// Architecture:
//
//	filtered  CTE  — applies scope/type/tag/meta filters once
//	semantic  CTE  — top candidate_limit rows by vector distance, ranked 1…N
//	keyword   CTE  — top candidate_limit rows by ts_rank,        ranked 1…M
//	rrf       CTE  — FULL OUTER JOIN, score = Σ 1/(60 + rank_i)
//	final query    — join back to knowledge, ORDER BY rrf_score DESC, LIMIT
func (p *postgresMemory) hybridSearch(ctx context.Context, queryVector []float32, queryText string, scope string, tags []string, memType MemoryType, metaFilters map[string]interface{}, limit int) ([]SearchResult, error) {
	vec := pgvector.NewVector(queryVector)

	// $1 = embedding vector, $2 = query text for plainto_tsquery
	args := []interface{}{vec, queryText}
	argIdx := 3 // next dynamic parameter index

	// Build the WHERE clause for the "filtered" CTE.
	var conditions []string
	if scope != "" {
		conditions = append(conditions, fmt.Sprintf("scope = $%d", argIdx))
		args = append(args, scope)
		argIdx++
	}
	if string(memType) != "" {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, string(memType))
		argIdx++
	}
	if len(tags) > 0 {
		tagsJSON, _ := json.Marshal(tags)
		conditions = append(conditions, fmt.Sprintf("tags @> $%d", argIdx))
		args = append(args, string(tagsJSON))
		argIdx++
	}
	if len(metaFilters) > 0 {
		metaJSON, _ := json.Marshal(metaFilters)
		conditions = append(conditions, fmt.Sprintf("metadata @> $%d", argIdx))
		args = append(args, string(metaJSON))
		argIdx++
	}

	filterClause := ""
	if len(conditions) > 0 {
		filterClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Candidate list is 3× the final limit so RRF has enough diversity.
	candidateLimit := limit * 3
	if candidateLimit < 30 {
		candidateLimit = 30
	}

	sqlQuery := fmt.Sprintf(`
	WITH
	  filtered AS (
	    SELECT id, title, content, type, metadata, tags, scope, parent_id,
	           chunk_index, normalized_title, content_hash, created_at, updated_at,
	           vector, fts
	    FROM knowledge
	    %s
	  ),
	  semantic AS (
	    SELECT id,
	           ROW_NUMBER() OVER (ORDER BY vector <=> $1) AS rank
	    FROM filtered
	    ORDER BY vector <=> $1
	    LIMIT %d
	  ),
	  keyword AS (
	    SELECT id,
	           ROW_NUMBER() OVER (ORDER BY ts_rank(fts, plainto_tsquery('english', $2)) DESC) AS rank
	    FROM filtered
	    WHERE fts @@ plainto_tsquery('english', $2)
	    ORDER BY ts_rank(fts, plainto_tsquery('english', $2)) DESC
	    LIMIT %d
	  ),
	  rrf AS (
	    SELECT
	      COALESCE(s.id, k.id) AS id,
	      COALESCE(1.0 / (60.0 + s.rank), 0.0) + COALESCE(1.0 / (60.0 + k.rank), 0.0) AS rrf_score
	    FROM semantic s
	    FULL OUTER JOIN keyword k ON s.id = k.id
	  )
	SELECT
	  f.id, f.title, f.content, f.type, f.metadata, f.tags, f.scope, f.parent_id,
	  f.chunk_index, f.normalized_title, f.content_hash, f.created_at, f.updated_at,
	  r.rrf_score AS score
	FROM filtered f
	JOIN rrf r ON f.id = r.id
	ORDER BY r.rrf_score DESC
	LIMIT %d
	`, filterClause, candidateLimit, candidateLimit, limit)

	return p.runSearchQuery(ctx, sqlQuery, args)
}

// runSearchQuery executes a search SQL and scans the standard column set.
func (p *postgresMemory) runSearchQuery(ctx context.Context, sqlQuery string, args []interface{}) ([]SearchResult, error) {
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
