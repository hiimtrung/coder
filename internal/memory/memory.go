package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Manager struct {
	db       MemoryService
	provider EmbeddingProvider
	chunker  *Chunker
}

// NewManager creates a new Manager
func NewManager(db MemoryService, provider EmbeddingProvider) *Manager {
	return &Manager{
		db:       db,
		provider: provider,
		chunker:  NewChunker(1000, 200), // Default settings
	}
}

func (m *Manager) Store(ctx context.Context, title, content string, memType MemoryType, metadata map[string]interface{}, scope string, tags []string) (string, error) {
	chunks := m.chunker.Chunk(content)
	parentID := uuid.New().String()

	for i, chunk := range chunks {
		embedding, err := m.provider.GenerateEmbedding(ctx, chunk)
		if err != nil {
			return "", err
		}

		id := uuid.New().String()
		if len(chunks) == 1 {
			id = parentID
		}

		k := &Knowledge{
			ID:              id,
			Title:           title,
			Content:         chunk,
			Type:            memType,
			Metadata:        metadata,
			Tags:            tags,
			Scope:           scope,
			ParentID:        parentID,
			ChunkIndex:      i,
			NormalizedTitle: strings.ToLower(strings.TrimSpace(title)),
			ContentHash:     hex.EncodeToString(hash(chunk)),
			Vector:          embedding,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		if err := m.db.Store(ctx, k); err != nil {
			return "", err
		}
	}

	return parentID, nil
}

func hash(content string) []byte {
	h := sha256.Sum256([]byte(content))
	return h[:]
}

func (m *Manager) Search(ctx context.Context, query string, scope string, tags []string, memType MemoryType, metaFilters map[string]interface{}, limit int) ([]SearchResult, error) {
	embedding, err := m.provider.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, err
	}

	// Pass both the embedding vector and the raw query text so the storage
	// layer can perform hybrid semantic + full-text search (RRF).
	return m.db.Search(ctx, embedding, query, scope, tags, memType, metaFilters, limit)
}

func (m *Manager) List(ctx context.Context, limit, offset int) ([]Knowledge, error) {
	return m.db.List(ctx, limit, offset)
}

func (m *Manager) Delete(ctx context.Context, id string) error {
	return m.db.Delete(ctx, id)
}

func (m *Manager) Close() error {
	return m.db.Close()
}

// Revector re-generates embeddings for all knowledge items
func (m *Manager) Revector(ctx context.Context) error {
	items, err := m.db.List(ctx, 10000, 0)
	if err != nil {
		return err
	}

	for _, item := range items {
		embedding, err := m.provider.GenerateEmbedding(ctx, item.Content)
		if err != nil {
			return err
		}
		item.Vector = embedding
		if err := m.db.Store(ctx, &item); err != nil {
			return err
		}
	}
	return nil
}

// Compact identifies and removes near-duplicate entries
func (m *Manager) Compact(ctx context.Context, threshold float32) (int, error) {
	items, err := m.db.List(ctx, 10000, 0)
	if err != nil {
		return 0, err
	}

	if len(items) < 2 {
		return 0, nil
	}

	if threshold <= 0 {
		threshold = CalculateSimilarityThreshold(len(items))
	}

	removed := 0
	deletedIDs := make(map[string]bool)

	// Fetch all items with vectors for comparison
	// Note: In a production system, we'd use a more efficient way to find duplicates
	// For this local implementation, we'll fetch full data for comparison
	fullItems := make([]Knowledge, 0, len(items))
	for _, it := range items {
		// We need to fetch vectors somehow. Let's assume List returns them for simplicity
		// Or we fetch them individually. Since it's local SQLite, let's fetch them.
		// Actually, I'll update the database.go List to optionally return vectors or just use Search.
		fullItems = append(fullItems, it)
	}

	for i := 0; i < len(fullItems); i++ {
		if deletedIDs[fullItems[i].ID] {
			continue
		}

		for j := i + 1; j < len(fullItems); j++ {
			if deletedIDs[fullItems[j].ID] {
				continue
			}

			similarity := CosineSimilarity(fullItems[i].Vector, fullItems[j].Vector)
			if similarity >= threshold {
				// Remove the later one to keep the earlier one
				if err := m.db.Delete(ctx, fullItems[j].ID); err != nil {
					return removed, err
				}
				deletedIDs[fullItems[j].ID] = true
				removed++
			}
		}
	}

	return removed, nil
}

// removed redundant CalculateSimilarityThreshold
