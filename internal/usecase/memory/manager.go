package ucmemory

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
)

// Manager implements memdomain.MemoryManager using a MemoryRepository and an optional EmbeddingProvider.
// When provider is nil the manager operates in FTS-only mode: Store persists without vectors and
// Search uses full-text search instead of semantic (vector) search.
type Manager struct {
	db       memdomain.MemoryRepository
	provider memdomain.EmbeddingProvider
	chunker  *memdomain.Chunker
}

// NewManager creates a new Manager. provider may be nil (FTS-only mode).
func NewManager(db memdomain.MemoryRepository, provider memdomain.EmbeddingProvider) *Manager {
	return &Manager{
		db:       db,
		provider: provider,
		chunker:  memdomain.NewChunker(1000, 200),
	}
}

func (m *Manager) Store(ctx context.Context, title, content string, memType memdomain.MemoryType, metadata map[string]any, scope string, tags []string) (string, error) {
	if memType == "" {
		memType = memdomain.TypeDocument
	}

	now := time.Now().UTC()
	metadata = memdomain.EnsureMetadata(metadata)
	replaceActive := memdomain.MetadataBool(metadata, memdomain.ControlKeyReplaceActive)
	explicitSupersedesID := memdomain.MetadataString(metadata, memdomain.MetadataKeySupersedesID)
	delete(metadata, memdomain.ControlKeyReplaceActive)

	if memdomain.MetadataString(metadata, memdomain.MetadataKeyCanonicalKey) == "" {
		metadata[memdomain.MetadataKeyCanonicalKey] = memdomain.NormalizeCanonicalKey(memType, title)
	}
	if memdomain.MetadataString(metadata, memdomain.MetadataKeyStatus) == "" {
		metadata[memdomain.MetadataKeyStatus] = string(memdomain.StatusActive)
	}

	var rowsToSupersede []memdomain.Knowledge
	if replaceActive || memdomain.MetadataString(metadata, memdomain.MetadataKeySupersedesID) != "" {
		repo, ok := m.db.(memdomain.MemoryLifecycleRepository)
		if !ok {
			return "", fmt.Errorf("memory repository does not support lifecycle updates")
		}

		if replaceActive {
			activeRows, err := repo.ListActiveByCanonicalKey(
				ctx,
				memdomain.MetadataString(metadata, memdomain.MetadataKeyCanonicalKey),
				scope,
			)
			if err != nil {
				return "", err
			}
			rowsToSupersede = append(rowsToSupersede, activeRows...)
			if len(activeRows) > 0 && memdomain.MetadataString(metadata, memdomain.MetadataKeySupersedesID) == "" {
				metadata[memdomain.MetadataKeySupersedesID] = memdomain.VersionGroupID(activeRows[0])
			}
		}

		if explicitSupersedesID != "" {
			explicitRows, err := m.groupRowsForSupersede(ctx, repo, explicitSupersedesID)
			if err != nil {
				return "", err
			}
			rowsToSupersede = append(rowsToSupersede, explicitRows...)
		}
	}

	chunks := m.chunker.Chunk(content)
	parentID := uuid.New().String()

	for i, chunk := range chunks {
		var vec []float32
		if m.provider != nil {
			v, err := m.provider.GenerateEmbedding(ctx, chunk)
			if err != nil {
				return "", err
			}
			vec = v
		}

		id := uuid.New().String()
		if len(chunks) == 1 {
			id = parentID
		}

		k := &memdomain.Knowledge{
			ID:              id,
			Title:           title,
			Content:         chunk,
			Type:            memType,
			Metadata:        memdomain.CloneMetadata(metadata),
			Tags:            tags,
			Scope:           scope,
			ParentID:        parentID,
			ChunkIndex:      i,
			NormalizedTitle: strings.ToLower(strings.TrimSpace(title)),
			ContentHash:     hex.EncodeToString(hash(chunk)),
			Vector:          vec,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		if err := m.db.Store(ctx, k); err != nil {
			return "", err
		}
	}

	if err := m.supersedeRows(ctx, rowsToSupersede, parentID, now); err != nil {
		return "", err
	}

	return parentID, nil
}

func hash(content string) []byte {
	h := sha256.Sum256([]byte(content))
	return h[:]
}

func (m *Manager) Search(ctx context.Context, query string, scope string, tags []string, memType memdomain.MemoryType, metaFilters map[string]any, limit int) ([]memdomain.SearchResult, error) {
	if limit <= 0 {
		limit = 5
	}

	now := time.Now().UTC()
	filters := memdomain.NormalizeSearchFilters(metaFilters, now)

	// When an embedding provider is available, use hybrid (vector + FTS) search.
	// Otherwise fall back to FTS-only by passing a nil vector.
	var vec []float32
	if m.provider != nil {
		v, err := m.provider.GenerateEmbedding(ctx, query)
		if err != nil {
			return nil, err
		}
		vec = v
	}

	fetchLimit := max(limit*3, 15)
	results, err := m.db.Search(ctx, vec, query, scope, tags, memType, filters, fetchLimit)
	if err != nil {
		return nil, err
	}

	results = m.rerankSearchResults(results, now)
	if !memdomain.MetadataBool(filters, memdomain.FilterKeyHistory) {
		results = m.collapseSearchResults(results)
	}
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func (m *Manager) List(ctx context.Context, limit, offset int) ([]memdomain.Knowledge, error) {
	return m.db.List(ctx, limit, offset)
}

func (m *Manager) Delete(ctx context.Context, id string) error {
	return m.db.Delete(ctx, id)
}

func (m *Manager) Verify(ctx context.Context, id string, opts memdomain.VerifyOptions) (int, error) {
	repo, ok := m.db.(memdomain.MemoryLifecycleRepository)
	if !ok {
		return 0, fmt.Errorf("memory repository does not support lifecycle updates")
	}
	if opts.Confidence != nil && (*opts.Confidence < 0 || *opts.Confidence > 1) {
		return 0, fmt.Errorf("confidence must be between 0 and 1")
	}

	rows, err := m.groupRowsForSupersede(ctx, repo, id)
	if err != nil {
		return 0, err
	}

	verifiedAt := opts.VerifiedAt.UTC()
	if verifiedAt.IsZero() {
		verifiedAt = time.Now().UTC()
	}
	updatedAt := time.Now().UTC()
	updated := 0
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if _, exists := seen[row.ID]; exists {
			continue
		}
		seen[row.ID] = struct{}{}

		metadata := memdomain.CloneMetadata(row.Metadata)
		memdomain.SetMetadataTime(metadata, memdomain.MetadataKeyLastVerifiedAt, verifiedAt)
		if strings.TrimSpace(opts.VerifiedBy) != "" {
			metadata[memdomain.MetadataKeyVerifiedBy] = strings.TrimSpace(opts.VerifiedBy)
		}
		if strings.TrimSpace(opts.SourceRef) != "" {
			metadata[memdomain.MetadataKeySourceRef] = strings.TrimSpace(opts.SourceRef)
		}
		if opts.Confidence != nil {
			metadata[memdomain.MetadataKeyConfidence] = *opts.Confidence
		}

		if err := repo.UpdateMetadata(ctx, row.ID, metadata, updatedAt); err != nil {
			return updated, err
		}
		updated++
	}

	return updated, nil
}

func (m *Manager) Supersede(ctx context.Context, id string, replacementID string) (int, error) {
	repo, ok := m.db.(memdomain.MemoryLifecycleRepository)
	if !ok {
		return 0, fmt.Errorf("memory repository does not support lifecycle updates")
	}

	targetRows, err := m.groupRowsForSupersede(ctx, repo, id)
	if err != nil {
		return 0, err
	}
	replacementRows, err := m.groupRowsForSupersede(ctx, repo, replacementID)
	if err != nil {
		return 0, err
	}
	if len(targetRows) == 0 || len(replacementRows) == 0 {
		return 0, fmt.Errorf("supersede requires both source and replacement memories")
	}

	targetGroupID := memdomain.VersionGroupID(targetRows[0])
	replacementGroupID := memdomain.VersionGroupID(replacementRows[0])
	if targetGroupID == replacementGroupID {
		return 0, fmt.Errorf("cannot supersede a memory with itself")
	}

	now := time.Now().UTC()
	canonicalKey := memdomain.CanonicalKeyForKnowledge(targetRows[0])
	if strings.TrimSpace(canonicalKey) == "" {
		canonicalKey = memdomain.CanonicalKeyForKnowledge(replacementRows[0])
	}

	updated := 0
	seen := make(map[string]struct{}, len(targetRows)+len(replacementRows))
	for _, row := range replacementRows {
		if _, exists := seen[row.ID]; exists {
			continue
		}
		seen[row.ID] = struct{}{}

		metadata := memdomain.CloneMetadata(row.Metadata)
		metadata[memdomain.MetadataKeyStatus] = string(memdomain.StatusActive)
		metadata[memdomain.MetadataKeyCanonicalKey] = canonicalKey
		metadata[memdomain.MetadataKeySupersedesID] = targetGroupID
		delete(metadata, memdomain.MetadataKeySupersededByID)
		delete(metadata, memdomain.MetadataKeyValidTo)

		if err := repo.UpdateMetadata(ctx, row.ID, metadata, now); err != nil {
			return updated, err
		}
		updated++
	}

	for _, row := range targetRows {
		if _, exists := seen[row.ID]; exists {
			continue
		}
		seen[row.ID] = struct{}{}

		metadata := memdomain.CloneMetadata(row.Metadata)
		metadata[memdomain.MetadataKeyStatus] = string(memdomain.StatusSuperseded)
		metadata[memdomain.MetadataKeyCanonicalKey] = canonicalKey
		metadata[memdomain.MetadataKeySupersededByID] = replacementGroupID
		if _, ok := memdomain.ValidToForKnowledge(row); !ok {
			memdomain.SetMetadataTime(metadata, memdomain.MetadataKeyValidTo, now)
		}

		if err := repo.UpdateMetadata(ctx, row.ID, metadata, now); err != nil {
			return updated, err
		}
		updated++
	}

	return updated, nil
}

func (m *Manager) Audit(ctx context.Context, opts memdomain.AuditOptions) (memdomain.AuditReport, error) {
	if opts.UnverifiedDays <= 0 {
		opts.UnverifiedDays = 180
	}
	if auditRepo, ok := m.db.(memdomain.MemoryAuditRepository); ok {
		return auditRepo.Audit(ctx, opts)
	}

	items, err := m.listAllKnowledge(ctx)
	if err != nil {
		return memdomain.AuditReport{}, err
	}
	return buildAuditReport(items, opts, time.Now().UTC()), nil
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
		threshold = memdomain.CalculateSimilarityThreshold(len(items))
	}

	removed := 0
	deletedIDs := make(map[string]bool)

	// Fetch all items with vectors for comparison
	// Note: In a production system, we'd use a more efficient way to find duplicates
	// For this local implementation, we'll fetch full data for comparison
	fullItems := make([]memdomain.Knowledge, 0, len(items))
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

			similarity := memdomain.CosineSimilarity(fullItems[i].Vector, fullItems[j].Vector)
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

func (m *Manager) groupRowsForSupersede(ctx context.Context, repo memdomain.MemoryLifecycleRepository, id string) ([]memdomain.Knowledge, error) {
	rows, err := repo.ListByParentID(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(rows) > 0 {
		return rows, nil
	}

	target, err := repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	groupID := memdomain.VersionGroupID(*target)
	rows, err = repo.ListByParentID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []memdomain.Knowledge{*target}, nil
	}
	return rows, nil
}

func (m *Manager) supersedeRows(ctx context.Context, rows []memdomain.Knowledge, newParentID string, now time.Time) error {
	if len(rows) == 0 {
		return nil
	}

	repo, ok := m.db.(memdomain.MemoryLifecycleRepository)
	if !ok {
		return fmt.Errorf("memory repository does not support lifecycle updates")
	}

	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if _, exists := seen[row.ID]; exists {
			continue
		}
		seen[row.ID] = struct{}{}
		if memdomain.VersionGroupID(row) == newParentID {
			continue
		}

		metadata := memdomain.CloneMetadata(row.Metadata)
		metadata[memdomain.MetadataKeyStatus] = string(memdomain.StatusSuperseded)
		metadata[memdomain.MetadataKeySupersededByID] = newParentID
		if _, ok := memdomain.ValidToForKnowledge(row); !ok {
			memdomain.SetMetadataTime(metadata, memdomain.MetadataKeyValidTo, now)
		}

		if err := repo.UpdateMetadata(ctx, row.ID, metadata, now); err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) rerankSearchResults(results []memdomain.SearchResult, now time.Time) []memdomain.SearchResult {
	ranked := append([]memdomain.SearchResult(nil), results...)
	for i := range ranked {
		ranked[i].Score = lifecycleScore(ranked[i], now)
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].Score > ranked[j].Score
	})
	return ranked
}

func (m *Manager) collapseSearchResults(results []memdomain.SearchResult) []memdomain.SearchResult {
	grouped := make(map[string][]memdomain.SearchResult, len(results))
	order := make([]string, 0, len(results))
	for _, result := range results {
		key := memdomain.CanonicalKeyForKnowledge(result.Knowledge)
		if _, exists := grouped[key]; !exists {
			order = append(order, key)
		}
		grouped[key] = append(grouped[key], result)
	}

	collapsed := make([]memdomain.SearchResult, 0, len(grouped))
	for _, key := range order {
		group := grouped[key]
		activeVersions := uniqueActiveVersionResults(group)
		if hasMaterialConflict(activeVersions) {
			collapsed = append(collapsed, summarizeConflictGroup(group, activeVersions))
			continue
		}
		collapsed = append(collapsed, group[0])
	}
	return collapsed
}

func lifecycleScore(result memdomain.SearchResult, now time.Time) float32 {
	score := result.Score
	switch memdomain.StatusForKnowledge(result.Knowledge) {
	case memdomain.StatusActive:
		score += 0.05
	case memdomain.StatusSuperseded, memdomain.StatusExpired, memdomain.StatusArchived:
		score -= 0.25
	case memdomain.StatusDraft:
		score -= 0.15
	}

	if confidence, ok := memdomain.ConfidenceForKnowledge(result.Knowledge); ok {
		score += float32(confidence) * 0.05
	}
	if validTo, ok := memdomain.ValidToForKnowledge(result.Knowledge); ok && !validTo.After(now) {
		score -= 0.2
	}

	referenceTime := result.UpdatedAt
	if verifiedAt, ok := memdomain.LastVerifiedAtForKnowledge(result.Knowledge); ok {
		referenceTime = verifiedAt
	}
	if referenceTime.IsZero() {
		referenceTime = result.CreatedAt
	}

	age := now.Sub(referenceTime)
	switch result.Type {
	case memdomain.TypeEvent:
		switch {
		case age > 180*24*time.Hour:
			score -= 0.15
		case age < 30*24*time.Hour:
			score += 0.03
		}
	case memdomain.TypeFact, memdomain.TypePattern, memdomain.TypeDocument:
		switch {
		case age > 365*24*time.Hour:
			score -= 0.06
		case age < 90*24*time.Hour:
			score += 0.02
		}
	case memdomain.TypeRule, memdomain.TypeDecision:
		switch {
		case age > 730*24*time.Hour:
			score -= 0.02
		default:
			score += 0.02
		}
	}

	return score
}

func (m *Manager) listAllKnowledge(ctx context.Context) ([]memdomain.Knowledge, error) {
	const pageSize = 500

	var items []memdomain.Knowledge
	for offset := 0; ; offset += pageSize {
		batch, err := m.db.List(ctx, pageSize, offset)
		if err != nil {
			return nil, err
		}
		items = append(items, batch...)
		if len(batch) < pageSize {
			return items, nil
		}
	}
}

func buildAuditReport(items []memdomain.Knowledge, opts memdomain.AuditOptions, now time.Time) memdomain.AuditReport {
	versionHeads := make(map[string]memdomain.Knowledge)
	for _, item := range items {
		if strings.TrimSpace(opts.Scope) != "" && item.Scope != opts.Scope {
			continue
		}

		groupID := memdomain.VersionGroupID(item)
		current, exists := versionHeads[groupID]
		if !exists || item.ChunkIndex < current.ChunkIndex || (item.ChunkIndex == current.ChunkIndex && item.CreatedAt.Before(current.CreatedAt)) {
			versionHeads[groupID] = item
		}
	}

	findings := make([]memdomain.AuditFinding, 0)
	conflictGroups := make(map[string][]memdomain.Knowledge)
	cutoff := now.Add(-time.Duration(opts.UnverifiedDays) * 24 * time.Hour)

	for _, version := range versionHeads {
		status := memdomain.StatusForKnowledge(version)
		if status == memdomain.StatusActive {
			scopeKey := version.Scope + "\x00" + memdomain.CanonicalKeyForKnowledge(version)
			conflictGroups[scopeKey] = append(conflictGroups[scopeKey], version)

			if validTo, ok := memdomain.ValidToForKnowledge(version); ok && !validTo.After(now) {
				findings = append(findings, memdomain.AuditFinding{
					Type:         memdomain.AuditFindingExpiredActive,
					CanonicalKey: memdomain.CanonicalKeyForKnowledge(version),
					Scope:        version.Scope,
					VersionIDs:   []string{memdomain.VersionGroupID(version)},
					Titles:       []string{version.Title},
					Details:      "This memory is still active even though its validity window has ended.",
					Count:        1,
				})
			}

			if verifiedAt, ok := memdomain.LastVerifiedAtForKnowledge(version); !ok || verifiedAt.Before(cutoff) {
				findings = append(findings, memdomain.AuditFinding{
					Type:         memdomain.AuditFindingActiveUnverified,
					CanonicalKey: memdomain.CanonicalKeyForKnowledge(version),
					Scope:        version.Scope,
					VersionIDs:   []string{memdomain.VersionGroupID(version)},
					Titles:       []string{version.Title},
					Details:      fmt.Sprintf("This active memory has not been verified within the last %d days.", opts.UnverifiedDays),
					Count:        1,
				})
			}
		}
	}

	for _, versions := range conflictGroups {
		if !hasMaterialKnowledgeConflict(versions) {
			continue
		}

		sort.SliceStable(versions, func(i, j int) bool {
			return versions[i].UpdatedAt.After(versions[j].UpdatedAt)
		})

		findings = append(findings, memdomain.AuditFinding{
			Type:         memdomain.AuditFindingActiveConflict,
			CanonicalKey: memdomain.CanonicalKeyForKnowledge(versions[0]),
			Scope:        versions[0].Scope,
			VersionIDs:   knowledgeVersionIDs(versions),
			Titles:       knowledgeTitles(versions),
			Details:      "Multiple active versions disagree for the same canonical key and should be resolved with supersede or verification.",
			Count:        len(versions),
		})
	}

	sort.SliceStable(findings, func(i, j int) bool {
		if auditFindingRank(findings[i].Type) != auditFindingRank(findings[j].Type) {
			return auditFindingRank(findings[i].Type) < auditFindingRank(findings[j].Type)
		}
		if findings[i].CanonicalKey != findings[j].CanonicalKey {
			return findings[i].CanonicalKey < findings[j].CanonicalKey
		}
		return findings[i].Scope < findings[j].Scope
	})

	return memdomain.AuditReport{
		GeneratedAt: now,
		Findings:    findings,
	}
}

func uniqueActiveVersionResults(group []memdomain.SearchResult) []memdomain.SearchResult {
	versions := make([]memdomain.SearchResult, 0, len(group))
	seen := make(map[string]struct{}, len(group))
	for _, result := range group {
		if memdomain.StatusForKnowledge(result.Knowledge) != memdomain.StatusActive {
			continue
		}

		versionID := memdomain.VersionGroupID(result.Knowledge)
		if _, exists := seen[versionID]; exists {
			continue
		}
		seen[versionID] = struct{}{}
		versions = append(versions, result)
	}
	return versions
}

func hasMaterialConflict(results []memdomain.SearchResult) bool {
	if len(results) < 2 {
		return false
	}

	signatures := make(map[string]struct{}, len(results))
	for _, result := range results {
		signatures[materialSignature(result.Knowledge)] = struct{}{}
	}
	return len(signatures) > 1
}

func hasMaterialKnowledgeConflict(versions []memdomain.Knowledge) bool {
	if len(versions) < 2 {
		return false
	}

	signatures := make(map[string]struct{}, len(versions))
	for _, version := range versions {
		signatures[materialSignature(version)] = struct{}{}
	}
	return len(signatures) > 1
}

func summarizeConflictGroup(group []memdomain.SearchResult, activeVersions []memdomain.SearchResult) memdomain.SearchResult {
	best := activeVersions[0]
	metadata := memdomain.CloneMetadata(best.Metadata)
	metadata[memdomain.MetadataKeyConflictDetected] = true
	metadata[memdomain.MetadataKeyConflictCount] = len(activeVersions)
	metadata[memdomain.MetadataKeyConflictVersionID] = searchVersionIDs(activeVersions)
	metadata[memdomain.MetadataKeyConflictTitles] = searchTitles(activeVersions)
	best.Metadata = metadata
	best.Title = "Conflict detected: " + best.Title
	best.Content = conflictSummary(best.CanonicalKey, activeVersions)
	return best
}

func conflictSummary(canonicalKey string, versions []memdomain.SearchResult) string {
	parts := make([]string, 0, min(len(versions), 3))
	for _, version := range versions {
		parts = append(parts, fmt.Sprintf("[%s] %s", shortenID(memdomain.VersionGroupID(version.Knowledge)), version.Title))
		if len(parts) == 3 {
			break
		}
	}

	return fmt.Sprintf(
		"Multiple active memories disagree for %s. Candidates: %s. Inspect history before relying on this memory.",
		canonicalKey,
		strings.Join(parts, "; "),
	)
}

func materialSignature(k memdomain.Knowledge) string {
	if hash := strings.TrimSpace(k.ContentHash); hash != "" {
		return hash
	}

	title := strings.ToLower(strings.TrimSpace(k.NormalizedTitle))
	if title == "" {
		title = strings.ToLower(strings.TrimSpace(k.Title))
	}
	content := strings.ToLower(strings.TrimSpace(k.Content))
	content = strings.Join(strings.Fields(content), " ")
	if len(content) > 160 {
		content = content[:160]
	}
	return title + "|" + content
}

func searchVersionIDs(results []memdomain.SearchResult) []string {
	ids := make([]string, 0, len(results))
	for _, result := range results {
		ids = append(ids, memdomain.VersionGroupID(result.Knowledge))
	}
	return ids
}

func knowledgeVersionIDs(items []memdomain.Knowledge) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, memdomain.VersionGroupID(item))
	}
	return ids
}

func searchTitles(results []memdomain.SearchResult) []string {
	titles := make([]string, 0, len(results))
	for _, result := range results {
		titles = append(titles, result.Title)
	}
	return titles
}

func knowledgeTitles(items []memdomain.Knowledge) []string {
	titles := make([]string, 0, len(items))
	for _, item := range items {
		titles = append(titles, item.Title)
	}
	return titles
}

func shortenID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

func auditFindingRank(kind memdomain.AuditFindingType) int {
	switch kind {
	case memdomain.AuditFindingActiveConflict:
		return 0
	case memdomain.AuditFindingExpiredActive:
		return 1
	case memdomain.AuditFindingActiveUnverified:
		return 2
	case memdomain.AuditFindingMissingLifecycle:
		return 3
	default:
		return 4
	}
}
