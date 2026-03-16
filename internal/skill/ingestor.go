package skill

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/trungtran/coder/internal/memory"
)

// Ingestor handles parsing + embedding + storing skills into vector DB.
type Ingestor struct {
	store    SkillService
	provider memory.EmbeddingProvider
}

// NewIngestor creates a new skill ingestor.
func NewIngestor(store SkillService, provider memory.EmbeddingProvider) *Ingestor {
	return &Ingestor{
		store:    store,
		provider: provider,
	}
}

// IngestResult holds the outcome of an ingestion operation.
type IngestResult struct {
	SkillName   string
	ChunksTotal int
	ChunksNew   int
	ChunksSkip  int
	Error       error
}

// IngestSkill ingests a single skill (SKILL.md content + rule files) into the vector DB.
func (ing *Ingestor) IngestSkill(ctx context.Context, name string, skillMD string, rules []RuleFile, source string, sourceRepo string, category string) (*IngestResult, error) {
	result := &IngestResult{SkillName: name}

	// 1. Parse SKILL.md
	parsed := ParseSkillMD(name, skillMD)

	// 2. Collect all sections from SKILL.md body + rule files
	var allSections []ParsedSection
	allSections = append(allSections, parsed.Sections...)

	for _, rule := range rules {
		section := ParseRuleFile(rule.Path, rule.Content)
		allSections = append(allSections, section)
	}

	// 3. Create/Update skill record
	skillID := generateSkillID(name)
	now := time.Now()

	finalCategory := category
	if parsed.Category != "" {
		finalCategory = parsed.Category
	}

	sk := &Skill{
		ID:          skillID,
		Name:        name,
		Description: parsed.Description,
		Category:    finalCategory,
		Source:      source,
		SourceRepo:  sourceRepo,
		Risk:        "unknown",
		Version:     now.Format("2006-01-02"),
		Tags:        parsed.Tags,
		Metadata:    map[string]interface{}{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := ing.store.UpsertSkill(ctx, sk); err != nil {
		return nil, fmt.Errorf("failed to upsert skill %q: %w", name, err)
	}

	// 4. Delete existing chunks (re-ingest)
	if err := ing.store.DeleteChunksBySkillID(ctx, skillID); err != nil {
		return nil, fmt.Errorf("failed to delete old chunks for %q: %w", name, err)
	}

	// 5. Process each section → generate embedding → store chunk
	for i, section := range allSections {
		if section.Content == "" {
			continue
		}

		contentHash := hashContent(section.Content)

		// Generate embedding
		embeddingText := section.Title + "\n" + section.Content
		embedding, err := ing.provider.GenerateEmbedding(ctx, embeddingText)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding for chunk %d of %q: %w", i, name, err)
		}

		chunk := &SkillChunk{
			ID:          uuid.New().String(),
			SkillID:     skillID,
			ChunkType:   section.Type,
			Title:       section.Title,
			Content:     section.Content,
			ChunkIndex:  i,
			ContentHash: contentHash,
			Vector:      embedding,
			CreatedAt:   now,
		}

		if err := ing.store.StoreChunk(ctx, chunk); err != nil {
			return nil, fmt.Errorf("failed to store chunk %d of %q: %w", i, name, err)
		}

		result.ChunksNew++
		result.ChunksTotal++
	}

	return result, nil
}

// SearchSkills performs semantic search across all skill chunks.
func (ing *Ingestor) SearchSkills(ctx context.Context, query string, limit int) ([]SkillSearchResult, error) {
	// Generate query embedding
	queryVec, err := ing.provider.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search chunks
	chunkResults, err := ing.store.SearchChunks(ctx, queryVec, limit*3) // Fetch more to group by skill
	if err != nil {
		return nil, err
	}

	// Group by skill_id
	skillMap := make(map[string]*SkillSearchResult)
	var orderedIDs []string

	for _, cr := range chunkResults {
		existing, ok := skillMap[cr.SkillID]
		if !ok {
			existing = &SkillSearchResult{
				Score: cr.Score,
			}
			skillMap[cr.SkillID] = existing
			orderedIDs = append(orderedIDs, cr.SkillID)
		}
		existing.Chunks = append(existing.Chunks, cr)
		if cr.Score > existing.Score {
			existing.Score = cr.Score
		}
	}

	// Fetch skill details for each unique skill
	var results []SkillSearchResult
	count := 0
	for _, skillID := range orderedIDs {
		if count >= limit {
			break
		}

		sr := skillMap[skillID]

		// We need to fetch the skill name from any chunk's skill_id
		// Use the first chunk's skill_id to fetch skill details
		skills, err := ing.store.ListSkills(ctx, "", "", 1000, 0)
		if err == nil {
			for _, sk := range skills {
				if sk.ID == skillID {
					sr.Skill = sk
					break
				}
			}
		}

		results = append(results, *sr)
		count++
	}

	return results, nil
}

func generateSkillID(name string) string {
	h := sha256.Sum256([]byte("skill:" + name))
	return hex.EncodeToString(h[:16])
}

func hashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}
