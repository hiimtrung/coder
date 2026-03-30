package memory

import (
	"time"
)

type MemoryType string

const (
	TypeFact       MemoryType = "fact"
	TypeRule       MemoryType = "rule"
	TypeDecision   MemoryType = "decision"
	TypePattern    MemoryType = "pattern"
	TypePreference MemoryType = "preference"
	TypeSkill      MemoryType = "skill"
	TypeEvent      MemoryType = "event"
	TypeDocument   MemoryType = "document"
)

// Knowledge represents a single piece of stored information
type Knowledge struct {
	ID              string          `json:"id"`
	Title           string          `json:"title"`
	Content         string          `json:"content"`
	Type            MemoryType      `json:"type"`
	Metadata        map[string]any  `json:"metadata"`
	Tags            []string        `json:"tags"`
	Scope           string          `json:"scope"`
	Status          LifecycleStatus `json:"status,omitempty"`
	CanonicalKey    string          `json:"canonical_key,omitempty"`
	SupersedesID    string          `json:"supersedes_id,omitempty"`
	SupersededByID  string          `json:"superseded_by_id,omitempty"`
	ValidFrom       *time.Time      `json:"valid_from,omitempty"`
	ValidTo         *time.Time      `json:"valid_to,omitempty"`
	LastVerifiedAt  *time.Time      `json:"last_verified_at,omitempty"`
	Confidence      *float64        `json:"confidence,omitempty"`
	SourceRef       string          `json:"source_ref,omitempty"`
	VerifiedBy      string          `json:"verified_by,omitempty"`
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
