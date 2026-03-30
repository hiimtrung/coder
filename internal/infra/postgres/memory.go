package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
	memdomain "github.com/trungtran/coder/internal/domain/memory"
)

type postgresMemory struct {
	db *sql.DB
}

// NewPostgresMemory creates a new PostgreSQL-backed memory repository.
func NewPostgresMemory(dsn string) (memdomain.MemoryRepository, error) {
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
func NewPostgresMemoryWithDB(dsn string) (memdomain.MemoryRepository, *sql.DB, error) {
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
		status VARCHAR(32) DEFAULT 'active',
		canonical_key TEXT,
		supersedes_id TEXT,
		superseded_by_id TEXT,
		valid_from TIMESTAMPTZ,
		valid_to TIMESTAMPTZ,
		last_verified_at TIMESTAMPTZ,
		confidence DOUBLE PRECISION,
		source_ref TEXT,
		verified_by TEXT,
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
		`ALTER TABLE knowledge ADD COLUMN IF NOT EXISTS status VARCHAR(32)`,
		`ALTER TABLE knowledge ADD COLUMN IF NOT EXISTS canonical_key TEXT`,
		`ALTER TABLE knowledge ADD COLUMN IF NOT EXISTS supersedes_id TEXT`,
		`ALTER TABLE knowledge ADD COLUMN IF NOT EXISTS superseded_by_id TEXT`,
		`ALTER TABLE knowledge ADD COLUMN IF NOT EXISTS valid_from TIMESTAMPTZ`,
		`ALTER TABLE knowledge ADD COLUMN IF NOT EXISTS valid_to TIMESTAMPTZ`,
		`ALTER TABLE knowledge ADD COLUMN IF NOT EXISTS last_verified_at TIMESTAMPTZ`,
		`ALTER TABLE knowledge ADD COLUMN IF NOT EXISTS confidence DOUBLE PRECISION`,
		`ALTER TABLE knowledge ADD COLUMN IF NOT EXISTS source_ref TEXT`,
		`ALTER TABLE knowledge ADD COLUMN IF NOT EXISTS verified_by TEXT`,
		`ALTER TABLE knowledge ALTER COLUMN status SET DEFAULT 'active'`,
		`CREATE INDEX IF NOT EXISTS idx_knowledge_status ON knowledge(status)`,
		`CREATE INDEX IF NOT EXISTS idx_knowledge_canonical_key ON knowledge(canonical_key)`,
		`CREATE INDEX IF NOT EXISTS idx_knowledge_supersedes_id ON knowledge(supersedes_id)`,
		`CREATE INDEX IF NOT EXISTS idx_knowledge_superseded_by_id ON knowledge(superseded_by_id)`,
		`CREATE INDEX IF NOT EXISTS idx_knowledge_valid_to ON knowledge(valid_to)`,
		`CREATE INDEX IF NOT EXISTS idx_knowledge_last_verified_at ON knowledge(last_verified_at)`,
		`CREATE INDEX IF NOT EXISTS idx_knowledge_active_key_scope ON knowledge(canonical_key, scope) WHERE status = 'active'`,
	}
	for _, m := range migrations {
		if _, err := p.db.Exec(m); err != nil {
			return fmt.Errorf("memory migration failed: %w", err)
		}
	}
	return p.backfillLifecycleColumns(context.Background())
}

func (p *postgresMemory) Store(ctx context.Context, k *memdomain.Knowledge) error {
	memdomain.HydrateKnowledgeLifecycle(k)

	tagsJSON, err := json.Marshal(k.Tags)
	if err != nil {
		return err
	}

	metaJSON, _ := json.Marshal(k.Metadata)

	vec := pgvector.NewVector(k.Vector)

	query := `
	INSERT INTO knowledge (
		id, title, content, type, metadata, tags, scope, status, canonical_key,
		supersedes_id, superseded_by_id, valid_from, valid_to, last_verified_at,
		confidence, source_ref, verified_by, parent_id, chunk_index, normalized_title,
		content_hash, vector, created_at, updated_at
	)
	VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9,
		$10, $11, $12, $13, $14,
		$15, $16, $17, $18, $19, $20,
		$21, $22, $23, $24
	)
	ON CONFLICT (id) DO UPDATE SET
		title = EXCLUDED.title,
		content = EXCLUDED.content,
		type = EXCLUDED.type,
		metadata = EXCLUDED.metadata,
		tags = EXCLUDED.tags,
		scope = EXCLUDED.scope,
		status = EXCLUDED.status,
		canonical_key = EXCLUDED.canonical_key,
		supersedes_id = EXCLUDED.supersedes_id,
		superseded_by_id = EXCLUDED.superseded_by_id,
		valid_from = EXCLUDED.valid_from,
		valid_to = EXCLUDED.valid_to,
		last_verified_at = EXCLUDED.last_verified_at,
		confidence = EXCLUDED.confidence,
		source_ref = EXCLUDED.source_ref,
		verified_by = EXCLUDED.verified_by,
		parent_id = EXCLUDED.parent_id,
		chunk_index = EXCLUDED.chunk_index,
		normalized_title = EXCLUDED.normalized_title,
		content_hash = EXCLUDED.content_hash,
		vector = EXCLUDED.vector,
		updated_at = EXCLUDED.updated_at
	`
	_, err = p.db.ExecContext(ctx, query,
		k.ID, k.Title, k.Content, string(k.Type), string(metaJSON), string(tagsJSON), k.Scope, string(k.Status), k.CanonicalKey,
		nullIfEmpty(k.SupersedesID), nullIfEmpty(k.SupersededByID), nullableTime(k.ValidFrom), nullableTime(k.ValidTo), nullableTime(k.LastVerifiedAt),
		nullableFloat64(k.Confidence), nullIfEmpty(k.SourceRef), nullIfEmpty(k.VerifiedBy), k.ParentID, k.ChunkIndex,
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
func (p *postgresMemory) Search(ctx context.Context, queryVector []float32, queryText string, scope string, tags []string, memType memdomain.MemoryType, metaFilters map[string]any, limit int) ([]memdomain.SearchResult, error) {
	if strings.TrimSpace(queryText) != "" {
		return p.hybridSearch(ctx, queryVector, queryText, scope, tags, memType, metaFilters, limit)
	}
	return p.semanticSearch(ctx, queryVector, scope, tags, memType, metaFilters, limit)
}

// semanticSearch is the original pure-vector search (used as fallback when
// no query text is available or FTS adds no value).
func (p *postgresMemory) semanticSearch(ctx context.Context, queryVector []float32, scope string, tags []string, memType memdomain.MemoryType, metaFilters map[string]any, limit int) ([]memdomain.SearchResult, error) {
	vec := pgvector.NewVector(queryVector)
	lifecycle, remainingFilters, err := extractLifecycleFilters(metaFilters)
	if err != nil {
		return nil, err
	}

	sqlQuery := `
		SELECT id, title, content, type, metadata, tags, scope, status, canonical_key,
		       supersedes_id, superseded_by_id, valid_from, valid_to, last_verified_at,
		       confidence, source_ref, verified_by, parent_id, chunk_index,
		       normalized_title, content_hash, created_at, updated_at,
		       1 - (vector <=> $1) AS score
		FROM knowledge
	`
	var args []any
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
	if lifecycle.status != "" {
		conditions = append(conditions, fmt.Sprintf("COALESCE(status, 'active') = $%d", argIdx))
		args = append(args, lifecycle.status)
		argIdx++
	}
	if lifecycle.canonicalKey != "" {
		conditions = append(conditions, fmt.Sprintf("COALESCE(NULLIF(canonical_key, ''), type || ':' || normalized_title) = $%d", argIdx))
		args = append(args, lifecycle.canonicalKey)
		argIdx++
	}
	if lifecycle.hasAsOf {
		conditions = append(
			conditions,
			fmt.Sprintf("(valid_from IS NULL OR valid_from <= $%d)", argIdx),
			fmt.Sprintf("(valid_to IS NULL OR valid_to > $%d)", argIdx),
		)
		args = append(args, lifecycle.asOf)
		argIdx++
	}
	if len(tags) > 0 {
		tagsJSON, _ := json.Marshal(tags)
		conditions = append(conditions, fmt.Sprintf("tags @> $%d", argIdx))
		args = append(args, string(tagsJSON))
		argIdx++
	}
	if len(remainingFilters) > 0 {
		metaJSON, _ := json.Marshal(remainingFilters)
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
func (p *postgresMemory) hybridSearch(ctx context.Context, queryVector []float32, queryText string, scope string, tags []string, memType memdomain.MemoryType, metaFilters map[string]any, limit int) ([]memdomain.SearchResult, error) {
	vec := pgvector.NewVector(queryVector)
	lifecycle, remainingFilters, err := extractLifecycleFilters(metaFilters)
	if err != nil {
		return nil, err
	}

	// $1 = embedding vector, $2 = query text for plainto_tsquery
	args := []any{vec, queryText}
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
	if lifecycle.status != "" {
		conditions = append(conditions, fmt.Sprintf("COALESCE(status, 'active') = $%d", argIdx))
		args = append(args, lifecycle.status)
		argIdx++
	}
	if lifecycle.canonicalKey != "" {
		conditions = append(conditions, fmt.Sprintf("COALESCE(NULLIF(canonical_key, ''), type || ':' || normalized_title) = $%d", argIdx))
		args = append(args, lifecycle.canonicalKey)
		argIdx++
	}
	if lifecycle.hasAsOf {
		conditions = append(
			conditions,
			fmt.Sprintf("(valid_from IS NULL OR valid_from <= $%d)", argIdx),
			fmt.Sprintf("(valid_to IS NULL OR valid_to > $%d)", argIdx),
		)
		args = append(args, lifecycle.asOf)
		argIdx++
	}
	if len(tags) > 0 {
		tagsJSON, _ := json.Marshal(tags)
		conditions = append(conditions, fmt.Sprintf("tags @> $%d", argIdx))
		args = append(args, string(tagsJSON))
		argIdx++
	}
	if len(remainingFilters) > 0 {
		metaJSON, _ := json.Marshal(remainingFilters)
		conditions = append(conditions, fmt.Sprintf("metadata @> $%d", argIdx))
		args = append(args, string(metaJSON))
		argIdx++
	}

	filterClause := ""
	if len(conditions) > 0 {
		filterClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Candidate list is 3× the final limit so RRF has enough diversity.
	candidateLimit := max(limit*3, 30)

	sqlQuery := fmt.Sprintf(`
	WITH
	  filtered AS (
	    SELECT id, title, content, type, metadata, tags, scope, status, canonical_key,
	           supersedes_id, superseded_by_id, valid_from, valid_to, last_verified_at,
	           confidence, source_ref, verified_by, parent_id, chunk_index,
	           normalized_title, content_hash, created_at, updated_at,
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
	  f.id, f.title, f.content, f.type, f.metadata, f.tags, f.scope, f.status, f.canonical_key,
	  f.supersedes_id, f.superseded_by_id, f.valid_from, f.valid_to, f.last_verified_at,
	  f.confidence, f.source_ref, f.verified_by, f.parent_id,
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
func (p *postgresMemory) runSearchQuery(ctx context.Context, sqlQuery string, args []any) ([]memdomain.SearchResult, error) {
	rows, err := p.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []memdomain.SearchResult
	for rows.Next() {
		knowledge, score, err := scanSearchResultRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, memdomain.SearchResult{Knowledge: knowledge, Score: score})
	}
	return results, nil
}

func (p *postgresMemory) List(ctx context.Context, limit int, offset int) ([]memdomain.Knowledge, error) {
	query := `SELECT id, title, content, type, metadata, tags, scope, status, canonical_key, supersedes_id, superseded_by_id, valid_from, valid_to, last_verified_at, confidence, source_ref, verified_by, parent_id, chunk_index, normalized_title, content_hash, created_at, updated_at FROM knowledge LIMIT $1 OFFSET $2`
	rows, err := p.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKnowledgeRows(rows)
}

func (p *postgresMemory) Delete(ctx context.Context, id string) error {
	query := "DELETE FROM knowledge WHERE id = $1 OR id LIKE $2"
	_, err := p.db.ExecContext(ctx, query, id, id+"%")
	return err
}

func (p *postgresMemory) Get(ctx context.Context, id string) (*memdomain.Knowledge, error) {
	row := p.db.QueryRowContext(
		ctx,
		`SELECT id, title, content, type, metadata, tags, scope, status, canonical_key, supersedes_id, superseded_by_id, valid_from, valid_to, last_verified_at, confidence, source_ref, verified_by, parent_id, chunk_index, normalized_title, content_hash, created_at, updated_at
		 FROM knowledge WHERE id = $1`,
		id,
	)

	knowledge, err := scanKnowledgeRow(row)
	if err != nil {
		return nil, err
	}
	return &knowledge, nil
}

func (p *postgresMemory) ListByParentID(ctx context.Context, parentID string) ([]memdomain.Knowledge, error) {
	rows, err := p.db.QueryContext(
		ctx,
		`SELECT id, title, content, type, metadata, tags, scope, status, canonical_key, supersedes_id, superseded_by_id, valid_from, valid_to, last_verified_at, confidence, source_ref, verified_by, parent_id, chunk_index, normalized_title, content_hash, created_at, updated_at
		 FROM knowledge
		 WHERE parent_id = $1 OR id = $1
		 ORDER BY chunk_index ASC, created_at ASC`,
		parentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanKnowledgeRows(rows)
}

func (p *postgresMemory) ListActiveByCanonicalKey(ctx context.Context, canonicalKey string, scope string) ([]memdomain.Knowledge, error) {
	query := `
		SELECT id, title, content, type, metadata, tags, scope, status, canonical_key, supersedes_id, superseded_by_id, valid_from, valid_to, last_verified_at, confidence, source_ref, verified_by, parent_id, chunk_index, normalized_title, content_hash, created_at, updated_at
		FROM knowledge
		WHERE COALESCE(NULLIF(canonical_key, ''), type || ':' || normalized_title) = $1
		  AND COALESCE(status, 'active') = 'active'
	`
	args := []any{canonicalKey}
	if strings.TrimSpace(scope) != "" {
		query += " AND scope = $2"
		args = append(args, scope)
	}
	query += " ORDER BY created_at ASC, chunk_index ASC"

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanKnowledgeRows(rows)
}

func (p *postgresMemory) UpdateMetadata(ctx context.Context, id string, metadata map[string]any, updatedAt time.Time) error {
	metadata = memdomain.CloneMetadata(metadata)
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	status := string(memdomain.StatusFromMetadata(metadata))
	canonicalKey := nullIfEmpty(memdomain.MetadataString(metadata, memdomain.MetadataKeyCanonicalKey))
	supersedesID := nullIfEmpty(memdomain.MetadataString(metadata, memdomain.MetadataKeySupersedesID))
	supersededByID := nullIfEmpty(memdomain.MetadataString(metadata, memdomain.MetadataKeySupersededByID))
	var validFrom any
	if at, ok := memdomain.MetadataTime(metadata, memdomain.MetadataKeyValidFrom); ok {
		validFrom = at.UTC()
	}
	var validTo any
	if at, ok := memdomain.MetadataTime(metadata, memdomain.MetadataKeyValidTo); ok {
		validTo = at.UTC()
	}
	var lastVerifiedAt any
	if at, ok := memdomain.MetadataTime(metadata, memdomain.MetadataKeyLastVerifiedAt); ok {
		lastVerifiedAt = at.UTC()
	}
	var confidence any
	if value, ok := memdomain.MetadataFloat(metadata, memdomain.MetadataKeyConfidence); ok {
		confidence = value
	}
	sourceRef := nullIfEmpty(memdomain.MetadataString(metadata, memdomain.MetadataKeySourceRef))
	verifiedBy := nullIfEmpty(memdomain.MetadataString(metadata, memdomain.MetadataKeyVerifiedBy))

	_, err = p.db.ExecContext(
		ctx,
		`UPDATE knowledge SET
			metadata = $2,
			status = COALESCE(NULLIF($3, ''), 'active'),
			canonical_key = COALESCE(NULLIF($4, ''), type || ':' || normalized_title),
			supersedes_id = NULLIF($5, ''),
			superseded_by_id = NULLIF($6, ''),
			valid_from = $7,
			valid_to = $8,
			last_verified_at = $9,
			confidence = $10,
			source_ref = NULLIF($11, ''),
			verified_by = NULLIF($12, ''),
			updated_at = $13
		WHERE id = $1`,
		id,
		string(metaJSON),
		nullIfEmpty(status),
		canonicalKey,
		supersedesID,
		supersededByID,
		validFrom,
		validTo,
		lastVerifiedAt,
		confidence,
		sourceRef,
		verifiedBy,
		updatedAt,
	)
	return err
}

func (p *postgresMemory) Audit(ctx context.Context, opts memdomain.AuditOptions) (memdomain.AuditReport, error) {
	if opts.UnverifiedDays <= 0 {
		opts.UnverifiedDays = 180
	}

	now := time.Now().UTC()
	findings := make([]memdomain.AuditFinding, 0)

	const versionHeads = `
WITH version_heads AS (
	SELECT DISTINCT ON (COALESCE(NULLIF(parent_id, ''), id))
		COALESCE(NULLIF(parent_id, ''), id) AS version_id,
		title,
		content,
		type,
		scope,
		status,
		canonical_key,
		normalized_title,
		content_hash,
		valid_to,
		last_verified_at,
		created_at,
		updated_at
	FROM knowledge
	WHERE ($1 = '' OR scope = $1)
	ORDER BY COALESCE(NULLIF(parent_id, ''), id), chunk_index ASC, created_at ASC
)`

	conflictRows, err := p.db.QueryContext(
		ctx,
		versionHeads+`
SELECT
	COALESCE(NULLIF(canonical_key, ''), type || ':' || normalized_title) AS canonical_key,
	scope,
	array_agg(version_id ORDER BY updated_at DESC, created_at DESC),
	array_agg(title ORDER BY updated_at DESC, created_at DESC),
	COUNT(*)::INT
FROM (
	SELECT *,
		COALESCE(NULLIF(content_hash, ''), md5(lower(trim(title)) || '|' || lower(trim(content)))) AS signature
	FROM version_heads
	WHERE COALESCE(status, 'active') = 'active'
) active_versions
GROUP BY canonical_key, scope
HAVING COUNT(*) > 1 AND COUNT(DISTINCT signature) > 1
ORDER BY COUNT(*) DESC, canonical_key ASC`,
		opts.Scope,
	)
	if err != nil {
		return memdomain.AuditReport{}, err
	}
	defer conflictRows.Close()

	for conflictRows.Next() {
		var canonicalKey string
		var scope string
		var versionIDs []string
		var titles []string
		var count int
		if err := conflictRows.Scan(&canonicalKey, &scope, pq.Array(&versionIDs), pq.Array(&titles), &count); err != nil {
			return memdomain.AuditReport{}, err
		}
		findings = append(findings, memdomain.AuditFinding{
			Type:         memdomain.AuditFindingActiveConflict,
			CanonicalKey: canonicalKey,
			Scope:        scope,
			VersionIDs:   versionIDs,
			Titles:       titles,
			Details:      "Multiple active versions disagree for the same canonical key and should be resolved with supersede or verification.",
			Count:        count,
		})
	}
	if err := conflictRows.Err(); err != nil {
		return memdomain.AuditReport{}, err
	}

	expiredRows, err := p.db.QueryContext(
		ctx,
		versionHeads+`
SELECT
	COALESCE(NULLIF(canonical_key, ''), type || ':' || normalized_title) AS canonical_key,
	scope,
	version_id,
	title
FROM version_heads
WHERE COALESCE(status, 'active') = 'active'
  AND valid_to IS NOT NULL
  AND valid_to <= $2
ORDER BY valid_to ASC, canonical_key ASC`,
		opts.Scope,
		now,
	)
	if err != nil {
		return memdomain.AuditReport{}, err
	}
	defer expiredRows.Close()

	for expiredRows.Next() {
		var canonicalKey string
		var scope string
		var versionID string
		var title string
		if err := expiredRows.Scan(&canonicalKey, &scope, &versionID, &title); err != nil {
			return memdomain.AuditReport{}, err
		}
		findings = append(findings, memdomain.AuditFinding{
			Type:         memdomain.AuditFindingExpiredActive,
			CanonicalKey: canonicalKey,
			Scope:        scope,
			VersionIDs:   []string{versionID},
			Titles:       []string{title},
			Details:      "This memory is still active even though its validity window has ended.",
			Count:        1,
		})
	}
	if err := expiredRows.Err(); err != nil {
		return memdomain.AuditReport{}, err
	}

	cutoff := now.Add(-time.Duration(opts.UnverifiedDays) * 24 * time.Hour)
	unverifiedRows, err := p.db.QueryContext(
		ctx,
		versionHeads+`
SELECT
	COALESCE(NULLIF(canonical_key, ''), type || ':' || normalized_title) AS canonical_key,
	scope,
	version_id,
	title
FROM version_heads
WHERE COALESCE(status, 'active') = 'active'
  AND (last_verified_at IS NULL OR last_verified_at < $2)
ORDER BY COALESCE(last_verified_at, created_at) ASC, canonical_key ASC`,
		opts.Scope,
		cutoff,
	)
	if err != nil {
		return memdomain.AuditReport{}, err
	}
	defer unverifiedRows.Close()

	for unverifiedRows.Next() {
		var canonicalKey string
		var scope string
		var versionID string
		var title string
		if err := unverifiedRows.Scan(&canonicalKey, &scope, &versionID, &title); err != nil {
			return memdomain.AuditReport{}, err
		}
		findings = append(findings, memdomain.AuditFinding{
			Type:         memdomain.AuditFindingActiveUnverified,
			CanonicalKey: canonicalKey,
			Scope:        scope,
			VersionIDs:   []string{versionID},
			Titles:       []string{title},
			Details:      fmt.Sprintf("This active memory has not been verified within the last %d days.", opts.UnverifiedDays),
			Count:        1,
		})
	}
	if err := unverifiedRows.Err(); err != nil {
		return memdomain.AuditReport{}, err
	}

	missingRows, err := p.db.QueryContext(
		ctx,
		versionHeads+`
SELECT
	COALESCE(NULLIF(canonical_key, ''), type || ':' || normalized_title) AS canonical_key,
	scope,
	version_id,
	title,
	CASE
		WHEN COALESCE(status, '') = '' AND COALESCE(canonical_key, '') = '' THEN 'status and canonical_key'
		WHEN COALESCE(status, '') = '' THEN 'status'
		ELSE 'canonical_key'
	END AS missing_field
FROM version_heads
WHERE COALESCE(status, '') = ''
   OR COALESCE(canonical_key, '') = ''
ORDER BY canonical_key ASC, version_id ASC`,
		opts.Scope,
	)
	if err != nil {
		return memdomain.AuditReport{}, err
	}
	defer missingRows.Close()

	for missingRows.Next() {
		var canonicalKey string
		var scope string
		var versionID string
		var title string
		var missingField string
		if err := missingRows.Scan(&canonicalKey, &scope, &versionID, &title, &missingField); err != nil {
			return memdomain.AuditReport{}, err
		}
		findings = append(findings, memdomain.AuditFinding{
			Type:         memdomain.AuditFindingMissingLifecycle,
			CanonicalKey: canonicalKey,
			Scope:        scope,
			VersionIDs:   []string{versionID},
			Titles:       []string{title},
			Details:      fmt.Sprintf("This memory version is missing %s and should be backfilled.", missingField),
			Count:        1,
		})
	}
	if err := missingRows.Err(); err != nil {
		return memdomain.AuditReport{}, err
	}

	return memdomain.AuditReport{
		GeneratedAt: now,
		Findings:    findings,
	}, nil
}

func (p *postgresMemory) Close() error {
	return p.db.Close()
}

type lifecycleSearchFilters struct {
	status       string
	canonicalKey string
	asOf         time.Time
	hasAsOf      bool
}

type knowledgeScanner interface {
	Scan(dest ...any) error
}

func (p *postgresMemory) backfillLifecycleColumns(ctx context.Context) error {
	rows, err := p.db.QueryContext(
		ctx,
		`SELECT id, title, content, type, metadata, tags, scope, status, canonical_key, supersedes_id, superseded_by_id, valid_from, valid_to, last_verified_at, confidence, source_ref, verified_by, parent_id, chunk_index, normalized_title, content_hash, created_at, updated_at
		 FROM knowledge
		 WHERE status IS NULL
		    OR canonical_key IS NULL
		    OR canonical_key = ''
		    OR (metadata ? 'supersedes_id' AND supersedes_id IS NULL)
		    OR (metadata ? 'superseded_by_id' AND superseded_by_id IS NULL)
		    OR (metadata ? 'valid_from' AND valid_from IS NULL)
		    OR (metadata ? 'valid_to' AND valid_to IS NULL)
		    OR (metadata ? 'last_verified_at' AND last_verified_at IS NULL)
		    OR (metadata ? 'confidence' AND confidence IS NULL)
		    OR (metadata ? 'source_ref' AND source_ref IS NULL)
		    OR (metadata ? 'verified_by' AND verified_by IS NULL)`,
	)
	if err != nil {
		return fmt.Errorf("memory lifecycle backfill query failed: %w", err)
	}
	defer rows.Close()

	items, err := scanKnowledgeRows(rows)
	if err != nil {
		return fmt.Errorf("memory lifecycle backfill scan failed: %w", err)
	}

	for _, item := range items {
		if err := p.UpdateMetadata(ctx, item.ID, item.Metadata, item.UpdatedAt); err != nil {
			return fmt.Errorf("memory lifecycle backfill update failed for %s: %w", item.ID, err)
		}
	}
	return nil
}

func extractLifecycleFilters(metaFilters map[string]any) (lifecycleSearchFilters, map[string]any, error) {
	filters := memdomain.CloneMetadata(metaFilters)
	lifecycle := lifecycleSearchFilters{
		status:       memdomain.MetadataString(filters, memdomain.FilterKeyStatus),
		canonicalKey: memdomain.MetadataString(filters, memdomain.FilterKeyCanonicalKey),
	}

	if asOf := memdomain.MetadataString(filters, memdomain.FilterKeyAsOf); asOf != "" {
		parsed, err := time.Parse(time.RFC3339, asOf)
		if err != nil {
			return lifecycleSearchFilters{}, nil, fmt.Errorf("invalid %s value: %w", memdomain.FilterKeyAsOf, err)
		}
		lifecycle.asOf = parsed.UTC()
		lifecycle.hasAsOf = true
	}

	delete(filters, memdomain.FilterKeyStatus)
	delete(filters, memdomain.FilterKeyCanonicalKey)
	delete(filters, memdomain.FilterKeyAsOf)
	delete(filters, memdomain.FilterKeyHistory)
	delete(filters, memdomain.FilterKeyIncludeStale)
	return lifecycle, filters, nil
}

func scanKnowledgeRows(rows *sql.Rows) ([]memdomain.Knowledge, error) {
	var results []memdomain.Knowledge
	for rows.Next() {
		knowledge, err := scanKnowledgeRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, knowledge)
	}
	return results, rows.Err()
}

func scanKnowledgeRow(scanner knowledgeScanner) (memdomain.Knowledge, error) {
	var knowledge memdomain.Knowledge
	var tagsStr []byte
	var metaStr []byte
	var memoryType string
	var status sql.NullString
	var canonicalKey sql.NullString
	var supersedesID sql.NullString
	var supersededByID sql.NullString
	var validFrom sql.NullTime
	var validTo sql.NullTime
	var lastVerifiedAt sql.NullTime
	var confidence sql.NullFloat64
	var sourceRef sql.NullString
	var verifiedBy sql.NullString
	if err := scanner.Scan(
		&knowledge.ID, &knowledge.Title, &knowledge.Content, &memoryType, &metaStr, &tagsStr,
		&knowledge.Scope, &status, &canonicalKey, &supersedesID, &supersededByID,
		&validFrom, &validTo, &lastVerifiedAt, &confidence, &sourceRef, &verifiedBy,
		&knowledge.ParentID, &knowledge.ChunkIndex, &knowledge.NormalizedTitle,
		&knowledge.ContentHash, &knowledge.CreatedAt, &knowledge.UpdatedAt,
	); err != nil {
		return memdomain.Knowledge{}, err
	}

	knowledge.Type = memdomain.MemoryType(memoryType)
	if len(tagsStr) > 0 {
		json.Unmarshal(tagsStr, &knowledge.Tags)
	}
	if len(metaStr) > 0 {
		json.Unmarshal(metaStr, &knowledge.Metadata)
	}
	if status.Valid {
		knowledge.Status = memdomain.LifecycleStatus(status.String)
	}
	if canonicalKey.Valid {
		knowledge.CanonicalKey = canonicalKey.String
	}
	if supersedesID.Valid {
		knowledge.SupersedesID = supersedesID.String
	}
	if supersededByID.Valid {
		knowledge.SupersededByID = supersededByID.String
	}
	if validFrom.Valid {
		knowledge.ValidFrom = timePtrLocal(validFrom.Time)
	}
	if validTo.Valid {
		knowledge.ValidTo = timePtrLocal(validTo.Time)
	}
	if lastVerifiedAt.Valid {
		knowledge.LastVerifiedAt = timePtrLocal(lastVerifiedAt.Time)
	}
	if confidence.Valid {
		knowledge.Confidence = float64PtrLocal(confidence.Float64)
	}
	if sourceRef.Valid {
		knowledge.SourceRef = sourceRef.String
	}
	if verifiedBy.Valid {
		knowledge.VerifiedBy = verifiedBy.String
	}
	memdomain.HydrateKnowledgeLifecycle(&knowledge)
	return knowledge, nil
}

func scanSearchResultRow(scanner knowledgeScanner) (memdomain.Knowledge, float32, error) {
	var knowledge memdomain.Knowledge
	var score float32
	var tagsStr []byte
	var metaStr []byte
	var memoryType string
	var status sql.NullString
	var canonicalKey sql.NullString
	var supersedesID sql.NullString
	var supersededByID sql.NullString
	var validFrom sql.NullTime
	var validTo sql.NullTime
	var lastVerifiedAt sql.NullTime
	var confidence sql.NullFloat64
	var sourceRef sql.NullString
	var verifiedBy sql.NullString
	if err := scanner.Scan(
		&knowledge.ID, &knowledge.Title, &knowledge.Content, &memoryType, &metaStr, &tagsStr,
		&knowledge.Scope, &status, &canonicalKey, &supersedesID, &supersededByID,
		&validFrom, &validTo, &lastVerifiedAt, &confidence, &sourceRef, &verifiedBy,
		&knowledge.ParentID, &knowledge.ChunkIndex, &knowledge.NormalizedTitle,
		&knowledge.ContentHash, &knowledge.CreatedAt, &knowledge.UpdatedAt, &score,
	); err != nil {
		return memdomain.Knowledge{}, 0, err
	}

	knowledge.Type = memdomain.MemoryType(memoryType)
	if len(tagsStr) > 0 {
		json.Unmarshal(tagsStr, &knowledge.Tags)
	}
	if len(metaStr) > 0 {
		json.Unmarshal(metaStr, &knowledge.Metadata)
	}
	if status.Valid {
		knowledge.Status = memdomain.LifecycleStatus(status.String)
	}
	if canonicalKey.Valid {
		knowledge.CanonicalKey = canonicalKey.String
	}
	if supersedesID.Valid {
		knowledge.SupersedesID = supersedesID.String
	}
	if supersededByID.Valid {
		knowledge.SupersededByID = supersededByID.String
	}
	if validFrom.Valid {
		knowledge.ValidFrom = timePtrLocal(validFrom.Time)
	}
	if validTo.Valid {
		knowledge.ValidTo = timePtrLocal(validTo.Time)
	}
	if lastVerifiedAt.Valid {
		knowledge.LastVerifiedAt = timePtrLocal(lastVerifiedAt.Time)
	}
	if confidence.Valid {
		knowledge.Confidence = float64PtrLocal(confidence.Float64)
	}
	if sourceRef.Valid {
		knowledge.SourceRef = sourceRef.String
	}
	if verifiedBy.Valid {
		knowledge.VerifiedBy = verifiedBy.String
	}
	memdomain.HydrateKnowledgeLifecycle(&knowledge)
	return knowledge, score, nil
}

func nullIfEmpty(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func nullableTime(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return value.UTC()
}

func nullableFloat64(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func timePtrLocal(value time.Time) *time.Time {
	v := value.UTC()
	return &v
}

func float64PtrLocal(value float64) *float64 {
	v := value
	return &v
}
