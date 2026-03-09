package memory

import (
	"context"
	"time"
)

type MemoryType string

const (
	TypeFact       MemoryType = "fact"
	TypeRule       MemoryType = "rule"
	TypePreference MemoryType = "preference"
	TypeSkill      MemoryType = "skill"
	TypeEvent      MemoryType = "event"
	TypeDocument   MemoryType = "document"
)

// Knowledge represents a single piece of stored information
type Knowledge struct {
	ID              string                 `json:"id"`
	Title           string                 `json:"title"`
	Content         string                 `json:"content"`
	Type            MemoryType             `json:"type"`
	Metadata        map[string]interface{} `json:"metadata"`
	Tags            []string               `json:"tags"`
	Scope           string                 `json:"scope"`
	ParentID        string
	ChunkIndex      int
	NormalizedTitle string    `json:"normalized_title"`
	ContentHash     string    `json:"content_hash"`
	Vector          []float32 `json:"vector,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// SearchResult represents a knowledge item with its similarity score
type SearchResult struct {
	Knowledge
	Score float32 `json:"score"`
}

// MemoryService defines the interface for memory operations
type MemoryService interface {
	Store(ctx context.Context, k *Knowledge) error
	Search(ctx context.Context, queryVector []float32, scope string, tags []string, memType MemoryType, metaFilters map[string]interface{}, limit int) ([]SearchResult, error)
	List(ctx context.Context, limit int, offset int) ([]Knowledge, error)
	Delete(ctx context.Context, id string) error
	Close() error
}

type MemoryManager interface {
	Store(ctx context.Context, title, content string, memType MemoryType, metadata map[string]interface{}, scope string, tags []string) (string, error)
	Search(ctx context.Context, query string, scope string, tags []string, memType MemoryType, metaFilters map[string]interface{}, limit int) ([]SearchResult, error)
	List(ctx context.Context, limit, offset int) ([]Knowledge, error)
	Delete(ctx context.Context, id string) error
	Compact(ctx context.Context, threshold float32) (int, error)
	Revector(ctx context.Context) error
	Close() error
}
