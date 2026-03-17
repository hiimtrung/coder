package skill

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/trungtran/coder/internal/memory"
)

// Confidence thresholds and retrieval tuning for SearchSkills.
const (
	scoreHighConfidence   float32 = 0.80 // return ALL chunks of the skill
	scoreMediumConfidence float32 = 0.55 // return matched chunks + section siblings + adjacent

	// adjacentRadius is how many chunks before/after a matched chunk to include.
	// 1 means ±1 (prev + next). Increase to 2 for wider context windows.
	adjacentRadius = 1
)

// maxEmbedChars is the safe character limit for the combined embedding text sent to the model.
// mxbai-embed-large has a 512-token context window. Code/markdown content averages ~2 chars/token
// (much denser than English prose at ~4 chars/token). We budget 460 tokens × 2 chars = 920 chars,
// then subtract ~50 chars for the section title that is prepended before embedding.
const maxEmbedChars = 800

// splitSectionIfNeeded splits a ParsedSection into multiple smaller sections so that
// title + "\n" + content stays within maxEmbedChars. Splits at paragraph (\n\n) or
// line (\n) boundaries; falls back to hard-cut only when necessary.
func splitSectionIfNeeded(s ParsedSection) []ParsedSection {
	// Check against the combined embedding text length
	combined := s.Title + "\n" + s.Content
	if len(combined) <= maxEmbedChars {
		return []ParsedSection{s}
	}

	// Content budget after reserving space for title overhead
	titleOverhead := len(s.Title) + 1 // title + "\n"
	contentBudget := maxEmbedChars - titleOverhead
	if contentBudget < 100 {
		contentBudget = 100 // floor: always take at least 100 chars of content
	}

	var result []ParsedSection
	remaining := s.Content
	partNum := 0

	for len(remaining) > 0 {
		if len(remaining) <= contentBudget {
			title := s.Title
			if partNum > 0 {
				title = fmt.Sprintf("%s (part %d)", s.Title, partNum+1)
			}
			result = append(result, ParsedSection{
				Title:     title,
				Content:   remaining,
				Type:      s.Type,
				SectionID: s.SectionID, // all parts share the same section ID
			})
			break
		}

		// Try to cut at a paragraph boundary (\n\n) in the second half of the budget
		cutAt := contentBudget
		if idx := strings.LastIndex(remaining[:cutAt], "\n\n"); idx > contentBudget/2 {
			cutAt = idx
		} else if idx := strings.LastIndex(remaining[:cutAt], "\n"); idx > contentBudget/2 {
			cutAt = idx
		}
		// Ensure we always make forward progress
		if cutAt == 0 {
			cutAt = contentBudget
		}

		title := s.Title
		if partNum == 0 {
			title = fmt.Sprintf("%s (part 1)", s.Title)
		} else {
			title = fmt.Sprintf("%s (part %d)", s.Title, partNum+1)
		}

		result = append(result, ParsedSection{
			Title:     title,
			Content:   strings.TrimSpace(remaining[:cutAt]),
			Type:      s.Type,
			SectionID: s.SectionID, // all parts share the same section ID
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

	// 2. Collect all sections from SKILL.md body + rule files; split long sections.
	//    Each section gets a SectionID before splitting so all parts stay linked.
	var allSections []ParsedSection
	for _, s := range parsed.Sections {
		s.SectionID = generateSectionID(name, s.Title)
		allSections = append(allSections, splitSectionIfNeeded(s)...)
	}

	for _, rule := range rules {
		ruleSections := ParseRuleFile(rule.Path, rule.Content)
		for _, section := range ruleSections {
			section.SectionID = generateSectionID(name, section.Title)
			allSections = append(allSections, splitSectionIfNeeded(section)...)
		}
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
			SectionID:   item.section.SectionID,
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

// SearchSkills performs semantic search with confidence-based context expansion:
//   - score ≥ 0.80 (high):   return ALL chunks of the matched skill
//   - score ≥ 0.55 (medium): return matched chunks + all split parts of the same section
//   - score < 0.55 (low):    return only the matched chunks
func (ing *Ingestor) SearchSkills(ctx context.Context, query string, limit int) ([]SkillSearchResult, error) {
	// 1. Embed query
	queryVec, err := ing.provider.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// 2. Fetch top-k candidate chunks (overfetch to ensure enough distinct skills)
	rawChunks, err := ing.store.SearchChunks(ctx, queryVec, limit*5)
	if err != nil {
		return nil, err
	}

	// 3. Group by skill_id — track best score, matched chunks, and matched section IDs
	type skillEntry struct {
		bestScore  float32
		chunks     []SkillChunkResult
		sectionIDs map[string]bool
	}
	skillMap := make(map[string]*skillEntry)
	var orderedIDs []string

	for _, cr := range rawChunks {
		e, ok := skillMap[cr.SkillID]
		if !ok {
			e = &skillEntry{sectionIDs: make(map[string]bool)}
			skillMap[cr.SkillID] = e
			orderedIDs = append(orderedIDs, cr.SkillID)
		}
		e.chunks = append(e.chunks, cr)
		if cr.Score > e.bestScore {
			e.bestScore = cr.Score
		}
		if cr.SectionID != "" {
			e.sectionIDs[cr.SectionID] = true
		}
	}

	// 4. For each matched skill, apply context-expansion based on confidence tier
	var results []SkillSearchResult
	count := 0

	for _, skillID := range orderedIDs {
		if count >= limit {
			break
		}
		e := skillMap[skillID]

		// Fetch skill metadata (single lookup by ID)
		sk, err := ing.store.GetSkillByID(ctx, skillID)
		if err != nil {
			continue // skill deleted between ingest and search — skip
		}

		var finalChunks []SkillChunk

		switch {
		case e.bestScore >= scoreHighConfidence:
			// High confidence: return the complete skill so the agent has all rules
			allChunks, err := ing.store.GetAllChunksBySkillID(ctx, skillID)
			if err == nil {
				finalChunks = allChunks
			} else {
				// fallback to matched chunks on error
				for _, cr := range e.chunks {
					finalChunks = append(finalChunks, cr.SkillChunk)
				}
			}

		case e.bestScore >= scoreMediumConfidence:
			// Medium confidence: matched chunks + section siblings + adjacent neighbours.
			// Layer order:
			//   1. matched chunk itself
			//   2. all parts split from the same original section (section_id match)
			//   3. chunks at index ±adjacentRadius from each matched chunk
			seen := make(map[string]bool)

			addChunk := func(c SkillChunk) {
				if !seen[c.ID] {
					seen[c.ID] = true
					finalChunks = append(finalChunks, c)
				}
			}

			for _, cr := range e.chunks {
				addChunk(cr.SkillChunk)

				// Layer 2: reassemble split section parts
				if cr.SectionID != "" {
					if siblings, err := ing.store.GetChunksBySectionID(ctx, cr.SectionID); err == nil {
						for _, s := range siblings {
							addChunk(s)
						}
					}
				}

				// Layer 3: adjacent chunks (prev + next) for narrative continuity
				if neighbours, err := ing.store.GetAdjacentChunks(ctx, cr.SkillID, cr.ChunkIndex, adjacentRadius); err == nil {
					for _, n := range neighbours {
						addChunk(n)
					}
				}
			}

			// Re-sort by chunk_index so content reads in logical order
			sort.Slice(finalChunks, func(i, j int) bool {
				return finalChunks[i].ChunkIndex < finalChunks[j].ChunkIndex
			})

		default:
			// Low confidence: matched chunks only — let caller decide if useful
			for _, cr := range e.chunks {
				finalChunks = append(finalChunks, cr.SkillChunk)
			}
		}

		// Wrap in SkillChunkResult with the skill-level score
		chunkResults := make([]SkillChunkResult, 0, len(finalChunks))
		for _, c := range finalChunks {
			chunkResults = append(chunkResults, SkillChunkResult{SkillChunk: c, Score: e.bestScore})
		}

		results = append(results, SkillSearchResult{
			Skill:  *sk,
			Chunks: chunkResults,
			Score:  e.bestScore,
		})
		count++
	}

	return results, nil
}

func generateSkillID(name string) string {
	h := sha256.Sum256([]byte("skill:" + name))
	return hex.EncodeToString(h[:16])
}

// generateSectionID creates a stable ID for a logical section within a skill.
// All chunks split from the same section share this ID, enabling section-level retrieval.
func generateSectionID(skillName, sectionTitle string) string {
	h := sha256.Sum256([]byte("section:" + skillName + ":" + sectionTitle))
	return hex.EncodeToString(h[:16])
}

func hashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}
