package skill

import (
	"context"
	"time"
)

// Skill represents a registered skill in the vector database.
type Skill struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Source      string                 `json:"source"`      // "local", "antigravity", "ui-ux-pro-max", "custom"
	SourceRepo  string                 `json:"source_repo"` // GitHub URL or "embedded"
	Risk        string                 `json:"risk"`        // "safe", "critical", "unknown"
	Version     string                 `json:"version"`
	Tags        []string               `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
	ChunkCount  int                    `json:"chunk_count,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// SkillChunk represents a vectorized chunk of a skill (rule, description, example, etc.)
type SkillChunk struct {
	ID          string    `json:"id"`
	SkillID     string    `json:"skill_id"`
	SectionID   string    `json:"section_id"`  // links all parts split from the same logical section
	ChunkType   string    `json:"chunk_type"`  // "description", "rule", "example", "workflow"
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	ChunkIndex  int       `json:"chunk_index"`
	ContentHash string    `json:"content_hash"`
	Vector      []float32 `json:"vector,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// SkillSearchResult represents a skill with its matched chunks and scores.
type SkillSearchResult struct {
	Skill  Skill              `json:"skill"`
	Chunks []SkillChunkResult `json:"chunks"`
	Score  float32            `json:"score"` // Max score across chunks
}

// SkillChunkResult represents a single matching chunk with score.
type SkillChunkResult struct {
	SkillChunk
	Score float32 `json:"score"`
}

// RuleFile represents a parsed rule file from a skill's rules directory.
type RuleFile struct {
	Path    string
	Content string
}

// SkillFile represents a raw script/data file associated with a skill.
// Stored as bytea in PostgreSQL; extracted to ~/.coder/cache/<skill>/ at runtime.
type SkillFile struct {
	ID          string    `json:"id"`
	SkillID     string    `json:"skill_id"`
	RelPath     string    `json:"rel_path"`      // e.g. "scripts/search.py", "data/colors.csv"
	ContentType string    `json:"content_type"`  // "text/x-python", "text/csv", etc.
	Content     []byte    `json:"content"`
	ContentHash string    `json:"content_hash"`
	SizeBytes   int       `json:"size_bytes"`
	CreatedAt   time.Time `json:"created_at"`
}

// SkillService defines the interface for skill CRUD + RAG operations.
type SkillService interface {
	// CRUD
	UpsertSkill(ctx context.Context, s *Skill) error
	StoreChunk(ctx context.Context, c *SkillChunk) error
	GetSkill(ctx context.Context, name string) (*Skill, []SkillChunk, error)
	ListSkills(ctx context.Context, source string, category string, limit int, offset int) ([]Skill, error)
	DeleteSkill(ctx context.Context, name string) error

	// RAG Search
	SearchChunks(ctx context.Context, queryVector []float32, limit int) ([]SkillChunkResult, error)

	// Retrieval helpers for context expansion
	GetSkillByID(ctx context.Context, id string) (*Skill, error)
	GetAllChunksBySkillID(ctx context.Context, skillID string) ([]SkillChunk, error)
	GetChunksBySectionID(ctx context.Context, sectionID string) ([]SkillChunk, error)

	// Deduplication
	GetChunkHashes(ctx context.Context, skillID string) (map[string]bool, error)

	// Cleanup
	DeleteChunksBySkillID(ctx context.Context, skillID string) error

	// File storage (scripts, data, references)
	StoreFile(ctx context.Context, f *SkillFile) error
	GetFiles(ctx context.Context, skillID string) ([]SkillFile, error)
	DeleteFilesBySkillID(ctx context.Context, skillID string) error

	Close() error
}
