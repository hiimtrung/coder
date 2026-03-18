package memory

import "context"

// EmbeddingProvider generates vector embeddings.
type EmbeddingProvider interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}

// MemoryRepository is the persistence port for the knowledge store.
// Search accepts both a pre-computed embedding vector (for semantic similarity)
// and the raw query text (for full-text keyword search). Implementations that
// support hybrid search (e.g. PostgreSQL with pgvector + tsvector) use both;
// others may ignore queryText and fall back to pure semantic search.
type MemoryRepository interface {
	Store(ctx context.Context, k *Knowledge) error
	Search(ctx context.Context, queryVector []float32, queryText string, scope string, tags []string, memType MemoryType, metaFilters map[string]any, limit int) ([]SearchResult, error)
	List(ctx context.Context, limit int, offset int) ([]Knowledge, error)
	Delete(ctx context.Context, id string) error
	Close() error
}

// MemoryManager is the application service interface.
// Implemented by usecase/memory.Manager AND by gRPC/HTTP remote clients.
type MemoryManager interface {
	Store(ctx context.Context, title, content string, memType MemoryType, metadata map[string]any, scope string, tags []string) (string, error)
	Search(ctx context.Context, query string, scope string, tags []string, memType MemoryType, metaFilters map[string]any, limit int) ([]SearchResult, error)
	List(ctx context.Context, limit, offset int) ([]Knowledge, error)
	Delete(ctx context.Context, id string) error
	Compact(ctx context.Context, threshold float32) (int, error)
	Revector(ctx context.Context) error
	Close() error
}
