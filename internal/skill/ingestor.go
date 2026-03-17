package skill

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/trungtran/coder/internal/memory"
)

// maxChunkChars is the safe character limit per chunk for mxbai-embed-large (~512 tokens).
// ~1800 chars ≈ 450 tokens with typical English text (4 chars/token average).
const maxChunkChars = 1800

// splitSectionIfNeeded splits a ParsedSection into multiple smaller sections
// if its content exceeds maxChunkChars. Splits on paragraph boundaries (\n\n)
// when possible, otherwise on newlines, otherwise hard-cuts.
func splitSectionIfNeeded(s ParsedSection) []ParsedSection {
	if len(s.Content) <= maxChunkChars {
		return []ParsedSection{s}
	}

	var result []ParsedSection
	remaining := s.Content
	partNum := 0

	for len(remaining) > 0 {
		if len(remaining) <= maxChunkChars {
			title := s.Title
			if partNum > 0 {
				title = fmt.Sprintf("%s (part %d)", s.Title, partNum+1)
			}
			result = append(result, ParsedSection{
				Title:   title,
				Content: remaining,
				Type:    s.Type,
			})
			break
		}

		// Try to cut at a paragraph boundary (\n\n) within the limit
		cutAt := maxChunkChars
		if idx := strings.LastIndex(remaining[:cutAt], "\n\n"); idx > maxChunkChars/2 {
			cutAt = idx
		} else if idx := strings.LastIndex(remaining[:cutAt], "\n"); idx > maxChunkChars/2 {
			cutAt = idx
		}

		title := s.Title
		if partNum > 0 {
			title = fmt.Sprintf("%s (part %d)", s.Title, partNum+1)
		} else {
			title = fmt.Sprintf("%s (part 1)", s.Title)
		}

		result = append(result, ParsedSection{
			Title:   title,
			Content: strings.TrimSpace(remaining[:cutAt]),
			Type:    s.Type,
		})
		remaining = strings.TrimSpace(remaining[cutAt:])
		partNum++
	}

	return result
}

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
// It uses content-hash deduplication to avoid re-embedding unchanged chunks.
func (ing *Ingestor) IngestSkill(ctx context.Context, name string, skillMD string, rules []RuleFile, source string, sourceRepo string, category string) (*IngestResult, error) {
	result := &IngestResult{SkillName: name}

	// 1. Parse SKILL.md
	parsed := ParseSkillMD(name, skillMD)

	// 2. Collect all sections from SKILL.md body + rule files; split long sections
	var allSections []ParsedSection
	for _, s := range parsed.Sections {
		allSections = append(allSections, splitSectionIfNeeded(s)...)
	}

	for _, rule := range rules {
		section := ParseRuleFile(rule.Path, rule.Content)
		allSections = append(allSections, splitSectionIfNeeded(section)...)
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

	// 4. Fetch existing content hashes for deduplication
	existingHashes, err := ing.store.GetChunkHashes(ctx, skillID)
	if err != nil {
		// If we can't get hashes (e.g., first time), proceed with full ingest
		existingHashes = make(map[string]bool)
	}

	// 5. Compute new hashes and determine which chunks need (re-)ingestion
	newHashes := make(map[string]bool)
	var sectionsToIngest []struct {
		index   int
		section ParsedSection
		hash    string
	}

	for i, section := range allSections {
		if section.Content == "" {
			continue
		}
		contentHash := hashContent(section.Content)
		newHashes[contentHash] = true
		result.ChunksTotal++

		if existingHashes[contentHash] {
			// Content hasn't changed — skip re-embedding
			result.ChunksSkip++
		} else {
			sectionsToIngest = append(sectionsToIngest, struct {
				index   int
				section ParsedSection
				hash    string
			}{index: i, section: section, hash: contentHash})
		}
	}

	// 6. If all chunks are unchanged, skip entirely
	if len(sectionsToIngest) == 0 && len(existingHashes) == len(newHashes) {
		return result, nil
	}

	// 7. Delete old chunks that no longer exist in the new version
	if len(sectionsToIngest) > 0 || len(existingHashes) != len(newHashes) {
		if err := ing.store.DeleteChunksBySkillID(ctx, skillID); err != nil {
			return nil, fmt.Errorf("failed to delete old chunks for %q: %w", name, err)
		}
		// Reset: we need to re-ingest ALL chunks since we deleted them
		// (PostgreSQL CASCADE or bulk delete removes everything)
		sectionsToIngest = nil
		result.ChunksNew = 0
		result.ChunksSkip = 0

		for i, section := range allSections {
			if section.Content == "" {
				continue
			}
			contentHash := hashContent(section.Content)
			sectionsToIngest = append(sectionsToIngest, struct {
				index   int
				section ParsedSection
				hash    string
			}{index: i, section: section, hash: contentHash})
		}
	}

	// 8. Process changed/new sections → rewrite paths → generate embedding → store chunk
	for _, item := range sectionsToIngest {
		// Rewrite script paths to point to ~/.coder/cache/<skill>/ before embedding
		rewrittenContent := RewriteSkillPaths(item.section.Content, name)

		// Generate embedding on rewritten content so semantic search reflects cache paths
		embeddingText := item.section.Title + "\n" + rewrittenContent
		embedding, err := ing.provider.GenerateEmbedding(ctx, embeddingText)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding for chunk %d of %q: %w", item.index, name, err)
		}

		chunk := &SkillChunk{
			ID:          uuid.New().String(),
			SkillID:     skillID,
			ChunkType:   item.section.Type,
			Title:       item.section.Title,
			Content:     rewrittenContent,
			ChunkIndex:  item.index,
			ContentHash: item.hash,
			Vector:      embedding,
			CreatedAt:   now,
		}

		if err := ing.store.StoreChunk(ctx, chunk); err != nil {
			return nil, fmt.Errorf("failed to store chunk %d of %q: %w", item.index, name, err)
		}

		result.ChunksNew++
	}

	return result, nil
}

// IngestFiles stores raw skill files (scripts, data, references) into the DB.
// Called separately from IngestSkill; files are keyed by (skillID, relPath).
func (ing *Ingestor) IngestFiles(ctx context.Context, skillName string, files []SkillFile) (int, error) {
	sk, _, err := ing.store.GetSkill(ctx, skillName)
	if err != nil {
		return 0, fmt.Errorf("skill %q must be ingested before its files: %w", skillName, err)
	}

	// Clear old files first so removed files don't linger
	if err := ing.store.DeleteFilesBySkillID(ctx, sk.ID); err != nil {
		return 0, fmt.Errorf("failed to clear old files for %q: %w", skillName, err)
	}

	stored := 0
	for _, f := range files {
		f.SkillID = sk.ID
		if f.ID == "" {
			f.ID = uuid.New().String()
		}
		if err := ing.store.StoreFile(ctx, &f); err != nil {
			return stored, fmt.Errorf("failed to store file %q for skill %q: %w", f.RelPath, skillName, err)
		}
		stored++
	}
	return stored, nil
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
