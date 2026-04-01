package ucmemory

import (
	"context"
	"sort"
	"strings"

	memdomain "github.com/trungtran/coder/internal/domain/memory"
)

func (m *Manager) Recall(ctx context.Context, opts memdomain.RecallOptions) (memdomain.RecallResult, error) {
	budget := opts.Budget
	if budget <= 0 {
		budget = 5
	}

	searchLimit := opts.Limit
	if searchLimit <= 0 {
		searchLimit = max(budget*3, 8)
	}

	results, err := m.Search(ctx, opts.Task, opts.Scope, opts.Tags, opts.Type, opts.MetaFilters, searchLimit)
	if err != nil {
		return memdomain.RecallResult{}, err
	}

	selected, keep, add, drop, conflicts := selectRecalledMemory(results, opts.Current, budget)
	memories := buildRecalledMemories(selected, opts.Current)

	return memdomain.RecallResult{
		Task:      opts.Task,
		Trigger:   opts.Trigger,
		Budget:    budget,
		Limit:     searchLimit,
		Coverage:  classifyMemoryCoverage(selected),
		Keep:      keep,
		Add:       add,
		Drop:      drop,
		Conflicts: conflicts,
		Memories:  memories,
	}, nil
}

func selectRecalledMemory(results []memdomain.SearchResult, current []string, budget int) ([]memdomain.SearchResult, []string, []string, []string, []string) {
	if budget <= 0 {
		budget = 5
	}

	ranked := append([]memdomain.SearchResult(nil), results...)
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Score == ranked[j].Score {
			return ranked[i].Title < ranked[j].Title
		}
		return ranked[i].Score > ranked[j].Score
	})
	if len(ranked) > budget {
		ranked = ranked[:budget]
	}

	currentSet := make(map[string]bool, len(current))
	for _, item := range current {
		currentSet[strings.ToLower(strings.TrimSpace(item))] = true
	}

	selectedSet := make(map[string]bool, len(ranked))
	keep := make([]string, 0, len(ranked))
	add := make([]string, 0, len(ranked))
	conflicts := make([]string, 0)
	for _, result := range ranked {
		identity := memoryIdentity(memdomain.CanonicalKeyForKnowledge(result.Knowledge), result.ID)
		selectedSet[identity] = true
		label := memoryLabel(result)
		if currentSet[identity] {
			keep = append(keep, label)
		} else {
			add = append(add, label)
		}
		if memdomain.MetadataBool(result.Metadata, memdomain.MetadataKeyConflictDetected) {
			conflicts = append(conflicts, label)
		}
	}

	drop := make([]string, 0)
	for _, item := range current {
		if !selectedSet[item] {
			drop = append(drop, item)
		}
	}

	return ranked, keep, add, drop, conflicts
}

func buildRecalledMemories(results []memdomain.SearchResult, current []string) []memdomain.RecalledMemory {
	currentSet := make(map[string]bool, len(current))
	for _, item := range current {
		currentSet[strings.ToLower(strings.TrimSpace(item))] = true
	}

	memories := make([]memdomain.RecalledMemory, 0, len(results))
	for _, res := range results {
		memories = append(memories, memdomain.RecalledMemory{
			Result: res,
			Reason: resolveMemoryReason(res, currentSet[memoryIdentity(memdomain.CanonicalKeyForKnowledge(res.Knowledge), res.ID)]),
		})
	}
	return memories
}

func classifyMemoryCoverage(results []memdomain.SearchResult) string {
	if len(results) == 0 {
		return "none"
	}
	best := results[0].Score
	switch {
	case best >= 0.80:
		return "strong"
	case best >= 0.55:
		return "adequate"
	default:
		return "weak"
	}
}

func resolveMemoryReason(result memdomain.SearchResult, alreadyActive bool) string {
	base := "added: recalled for the current task"
	if alreadyActive {
		base = "kept active: still relevant to the current task"
	}
	if memdomain.MetadataBool(result.Metadata, memdomain.MetadataKeyConflictDetected) {
		return base + " with conflict warning"
	}
	return base
}

func memoryIdentity(canonicalKey, id string) string {
	if strings.TrimSpace(canonicalKey) != "" {
		return strings.ToLower(strings.TrimSpace(canonicalKey))
	}
	return strings.ToLower(strings.TrimSpace(id))
}

func memoryLabel(result memdomain.SearchResult) string {
	if key := memdomain.CanonicalKeyForKnowledge(result.Knowledge); key != "" {
		return key
	}
	return result.ID
}
