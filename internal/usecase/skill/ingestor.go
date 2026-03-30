package ucskill

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	memdomain "github.com/trungtran/coder/internal/domain/memory"
	skilldomain "github.com/trungtran/coder/internal/domain/skill"
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
func splitSectionIfNeeded(s skilldomain.ParsedSection) []skilldomain.ParsedSection {
	// Check against the combined embedding text length
	combined := s.Title + "\n" + s.Content
	if len(combined) <= maxEmbedChars {
		return []skilldomain.ParsedSection{s}
	}

	// Content budget after reserving space for title overhead
	titleOverhead := len(s.Title) + 1 // title + "\n"
	contentBudget := maxEmbedChars - titleOverhead
	contentBudget = max(contentBudget, 100) // floor: always take at least 100 chars of content

	var result []skilldomain.ParsedSection
	remaining := s.Content
	partNum := 0

	for len(remaining) > 0 {
		if len(remaining) <= contentBudget {
			title := s.Title
			if partNum > 0 {
				title = fmt.Sprintf("%s (part %d)", s.Title, partNum+1)
			}
			result = append(result, skilldomain.ParsedSection{
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

		result = append(result, skilldomain.ParsedSection{
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
	store    skilldomain.SkillRepository
	provider memdomain.EmbeddingProvider
}

// NewIngestor creates a new skill ingestor.
func NewIngestor(store skilldomain.SkillRepository, provider memdomain.EmbeddingProvider) *Ingestor {
	return &Ingestor{
		store:    store,
		provider: provider,
	}
}

// IngestSkill ingests a single skill (SKILL.md content + rule files) into the vector DB.
// It uses content-hash deduplication to avoid re-embedding unchanged chunks.
func (ing *Ingestor) IngestSkill(ctx context.Context, name string, skillMD string, rules []skilldomain.RuleFile, source string, sourceRepo string, category string) (*skilldomain.IngestResult, error) {
	result := &skilldomain.IngestResult{SkillName: name}

	// 1. Parse SKILL.md
	parsed := skilldomain.ParseSkillMD(name, skillMD)

	// 2. Collect all sections from SKILL.md body + rule files; split long sections.
	//    Each section gets a SectionID before splitting so all parts stay linked.
	var allSections []skilldomain.ParsedSection
	for _, s := range parsed.Sections {
		s.SectionID = generateSectionID(name, s.Title)
		allSections = append(allSections, splitSectionIfNeeded(s)...)
	}

	for _, rule := range rules {
		ruleSections := skilldomain.ParseRuleFile(rule.Path, rule.Content)
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
	if finalCategory == "" || finalCategory == "uncategorized" {
		finalCategory = inferCategory(name, parsed.Description)
	}

	sk := &skilldomain.Skill{
		ID:          skillID,
		Name:        name,
		Description: parsed.Description,
		Category:    finalCategory,
		Source:      source,
		SourceRepo:  sourceRepo,
		Risk:        "unknown",
		Version:     now.Format("2006-01-02"),
		Tags:        parsed.Tags,
		Metadata:    map[string]any{},
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
		section skilldomain.ParsedSection
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
				section skilldomain.ParsedSection
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
				section skilldomain.ParsedSection
				hash    string
			}{index: i, section: section, hash: contentHash})
		}
	}

	// 8. Process changed/new sections → rewrite paths → optionally embed → store chunk
	for _, item := range sectionsToIngest {
		// Rewrite script paths to point to ~/.coder/cache/<skill>/ before embedding
		rewrittenContent := skilldomain.RewriteSkillPaths(item.section.Content, name)

		var vec []float32
		if ing.provider != nil {
			embeddingText := item.section.Title + "\n" + rewrittenContent
			v, err := ing.provider.GenerateEmbedding(ctx, embeddingText)
			if err != nil {
				return nil, fmt.Errorf("failed to generate embedding for chunk %d of %q: %w", item.index, name, err)
			}
			if len(v) > 0 {
				vec = v
			}
		}

		chunk := &skilldomain.SkillChunk{
			ID:          uuid.New().String(),
			SkillID:     skillID,
			SectionID:   item.section.SectionID,
			ChunkType:   item.section.Type,
			Title:       item.section.Title,
			Content:     rewrittenContent,
			ChunkIndex:  item.index,
			ContentHash: item.hash,
			Vector:      vec,
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
func (ing *Ingestor) IngestFiles(ctx context.Context, skillName string, files []skilldomain.SkillFile) (int, error) {
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
func (ing *Ingestor) SearchSkills(ctx context.Context, query string, limit int) ([]skilldomain.SkillSearchResult, error) {
	// 1. Optionally embed query (nil vector → FTS-only mode in store)
	var queryVec []float32
	if ing.provider != nil {
		v, err := ing.provider.GenerateEmbedding(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("failed to generate query embedding: %w", err)
		}
		if len(v) > 0 {
			queryVec = v
		}
	}

	// 2. Fetch top-k candidate chunks via hybrid/FTS search.
	// Overfetch so there are enough distinct skills after grouping.
	rawChunks, err := ing.store.SearchChunks(ctx, queryVec, query, limit*5)
	if err != nil {
		return nil, err
	}

	// 3. Group by skill_id — track best score, matched chunks, and matched section IDs
	type skillEntry struct {
		bestScore  float32
		chunks     []skilldomain.SkillChunkResult
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
	var results []skilldomain.SkillSearchResult
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

		var finalChunks []skilldomain.SkillChunk

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

			addChunk := func(c skilldomain.SkillChunk) {
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
		chunkResults := make([]skilldomain.SkillChunkResult, 0, len(finalChunks))
		for _, c := range finalChunks {
			chunkResults = append(chunkResults, skilldomain.SkillChunkResult{SkillChunk: c, Score: e.bestScore})
		}

		results = append(results, skilldomain.SkillSearchResult{
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
// inferCategory derives a category from the skill name and description when
// the SKILL.md frontmatter does not specify one explicitly.
func inferCategory(name, description string) string {
	lower := strings.ToLower(name + " " + description)

	frontendKW := []string{"react", "vue", "angular", "svelte", "next", "nuxt", "css", "html",
		"frontend", "ui", "ux", "design", "banner", "brand", "slides", "figma",
		"tailwind", "sass", "web-design", "stylesheet", "component"}
	backendKW := []string{"golang", "rust", "java", "nestjs", "django", "fastapi", "spring",
		"grpc", "database", "sql", "postgres", "redis", "backend", "api", "server",
		"microservice", "architecture", "docker", "kubernetes", "devops", "c language",
		"c development", "dart", "flutter", "mobile"}
	generalKW := []string{"testing", "test", "pattern", "composition", "general", "docs",
		"documentation", "analysis", "development", "refactor", "clean code", "python"}

	score := func(kws []string) int {
		n := 0
		for _, kw := range kws {
			if strings.Contains(lower, kw) {
				n++
			}
		}
		return n
	}

	fe, be, gen := score(frontendKW), score(backendKW), score(generalKW)
	switch {
	case fe > be && fe > gen:
		return "frontend"
	case be > fe && be > gen:
		return "backend"
	case gen > 0:
		return "general"
	default:
		return "general"
	}
}

func generateSectionID(skillName, sectionTitle string) string {
	h := sha256.Sum256([]byte("section:" + skillName + ":" + sectionTitle))
	return hex.EncodeToString(h[:16])
}

func hashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}
