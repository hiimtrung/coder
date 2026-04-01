package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	memdomain "github.com/trungtran/coder/internal/domain/memory"
)

type memoryRecallOutput struct {
	Task      string              `json:"task"`
	Trigger   string              `json:"trigger"`
	Budget    int                 `json:"budget"`
	Coverage  string              `json:"coverage"`
	Keep      []string            `json:"keep"`
	Add       []string            `json:"add"`
	Drop      []string            `json:"drop"`
	Conflicts []string            `json:"conflicts,omitempty"`
	Memories  []activeMemoryEntry `json:"memories"`
}

func runMemoryRecall(args []string) {
	logActivity("memory recall")
	fs := flag.NewFlagSet("memory recall", flag.ExitOnError)
	trigger := fs.String("trigger", "execution", "Recall trigger: initial, clarified, execution, error-recovery, review")
	current := fs.String("current", "", "Comma-separated current memory IDs or canonical keys (default: load from .coder/active-memory.json)")
	budget := fs.Int("budget", 5, "Maximum number of active memory items to keep")
	limit := fs.Int("limit", 0, "Search candidate limit before recall (default: max(budget*3, 8))")
	format := fs.String("format", "text", "Output format: text, json, raw")
	noSave := fs.Bool("no-save", false, "Do not update .coder/active-memory.json")
	scope := fs.String("scope", "", "Memory scope")
	memType := fs.String("type", "", "Filter by Memory type")
	status := fs.String("status", "", "Lifecycle status filter")
	key := fs.String("key", "", "Canonical key filter")
	asOf := fs.String("as-of", "", "RFC3339 timestamp to evaluate validity at a point in time")
	includeStale := fs.Bool("include-stale", false, "Include stale, expired, or superseded memories")
	history := fs.Bool("history", false, "Return multiple versions for the same canonical key")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: coder memory recall <task> [flags]")
		fmt.Fprintln(os.Stderr, "\nFLAGS:")
		fs.PrintDefaults()
	}

	fs.Parse(args)
	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}
	task := fs.Arg(0)

	currentItems, err := currentMemoryFromFlagOrState(*current)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load active memory state: %v\n", err)
		os.Exit(1)
	}

	searchLimit := *limit
	if searchLimit <= 0 {
		searchLimit = max(*budget*3, 8)
	}

	var metaFilters map[string]any
	metaFilters = memdomain.EnsureMetadata(metaFilters)
	if *status != "" {
		metaFilters[memdomain.FilterKeyStatus] = *status
	}
	if *key != "" {
		metaFilters[memdomain.FilterKeyCanonicalKey] = *key
	}
	if *includeStale {
		metaFilters[memdomain.FilterKeyIncludeStale] = true
	}
	if *history {
		metaFilters[memdomain.FilterKeyHistory] = true
	}
	if *asOf != "" {
		at, err := parseRFC3339Flag("as-of", *asOf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		metaFilters[memdomain.FilterKeyAsOf] = at.UTC().Format(time.RFC3339)
	}

	mgr := getMemoryManager()
	defer mgr.Close()

	results, err := mgr.Search(context.Background(), task, *scope, nil, memdomain.MemoryType(*memType), metaFilters, searchLimit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	output, state := buildMemoryRecallOutput(task, *trigger, *budget, *scope, *memType, *status, *key, *asOf, *includeStale, *history, currentItems, results)
	if !*noSave {
		if err := saveActiveMemoryState(state); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to save active memory state: %v\n", err)
			os.Exit(1)
		}
	}

	switch *format {
	case "text":
		fmt.Printf("Recalled %d active memory item(s) for %q [%s]:\n\n", len(output.Memories), task, output.Trigger)
		for _, memory := range output.Memories {
			fmt.Printf("[%s] %s (Score: %.4f)\n", shortMemoryID(memory.ID), memory.Title, memory.Score)
			fmt.Printf("  %s\n", memory.Reason)
			fmt.Printf("  Status: %s | Key: %s\n\n", fallbackString(memory.Status, "(unknown)"), fallbackString(memory.CanonicalKey, "(none)"))
		}
		fmt.Printf("Coverage: %s\n", output.Coverage)
		fmt.Printf("Keep:     %s\n", fallbackList(output.Keep))
		fmt.Printf("Add:      %s\n", fallbackList(output.Add))
		fmt.Printf("Drop:     %s\n", fallbackList(output.Drop))
		if len(output.Conflicts) > 0 {
			fmt.Printf("Conflicts:%s\n", formatIndentedList(output.Conflicts))
		}
	case "json":
		writeJSON(output)
	case "raw":
		selected := recalledEntriesToSearchResults(output.Memories, results)
		fmt.Print(renderRawMemoryContext(task, selected))
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported format %q (supported: text, json, raw)\n", *format)
		os.Exit(1)
	}
}

func currentMemoryFromFlagOrState(raw string) ([]string, error) {
	if items := normalizeMemoryIdentifiers(raw); len(items) > 0 {
		return items, nil
	}

	state, err := loadActiveMemoryState()
	if err != nil || state == nil {
		return nil, err
	}

	items := make([]string, 0, len(state.Results))
	for _, result := range state.Results {
		items = append(items, memoryIdentity(result.CanonicalKey, result.ID))
	}
	return items, nil
}

func normalizeMemoryIdentifiers(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	var out []string
	seen := make(map[string]bool)
	for _, item := range strings.Split(raw, ",") {
		normalized := strings.ToLower(strings.TrimSpace(item))
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		out = append(out, normalized)
	}
	return out
}

func buildMemoryRecallOutput(task, trigger string, budget int, scope, memType, status, canonicalKey, asOf string, includeStale, history bool, current []string, results []memdomain.SearchResult) (*memoryRecallOutput, *activeMemoryState) {
	if budget <= 0 {
		budget = 5
	}

	selected, keep, add, drop, conflicts := selectRecalledMemory(results, current, budget)
	coverage := classifyMemoryCoverage(selected)
	entries := buildActiveMemoryEntries(selected, current)

	output := &memoryRecallOutput{
		Task:      task,
		Trigger:   trigger,
		Budget:    budget,
		Coverage:  coverage,
		Keep:      keep,
		Add:       add,
		Drop:      drop,
		Conflicts: conflicts,
		Memories:  entries,
	}
	state := &activeMemoryState{
		Mode:         "recall",
		Query:        task,
		Trigger:      trigger,
		Budget:       budget,
		Scope:        scope,
		Type:         memType,
		Limit:        budget,
		Status:       status,
		CanonicalKey: canonicalKey,
		AsOf:         asOf,
		IncludeStale: includeStale,
		History:      history,
		SearchedAt:   time.Now(),
		Keep:         keep,
		Add:          add,
		Drop:         drop,
		Coverage:     coverage,
		Conflicts:    conflicts,
		Results:      entries,
	}
	return output, state
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

func buildActiveMemoryEntries(results []memdomain.SearchResult, current []string) []activeMemoryEntry {
	currentSet := make(map[string]bool, len(current))
	for _, item := range current {
		currentSet[strings.ToLower(strings.TrimSpace(item))] = true
	}

	entries := make([]activeMemoryEntry, 0, len(results))
	for _, res := range results {
		entry := activeMemoryEntry{
			ID:               res.ID,
			Title:            res.Title,
			Type:             string(res.Type),
			Scope:            res.Scope,
			Status:           string(memdomain.StatusForKnowledge(res.Knowledge)),
			CanonicalKey:     memdomain.CanonicalKeyForKnowledge(res.Knowledge),
			Score:            res.Score,
			Tags:             append([]string(nil), res.Tags...),
			ConflictDetected: memdomain.MetadataBool(res.Metadata, memdomain.MetadataKeyConflictDetected),
			Content:          res.Content,
			Reason:           resolveMemoryReason(res, currentSet[memoryIdentity(memdomain.CanonicalKeyForKnowledge(res.Knowledge), res.ID)]),
		}
		if entry.ConflictDetected {
			entry.ConflictCount = metadataInt(res.Metadata[memdomain.MetadataKeyConflictCount])
		}
		if verifiedAt, ok := memdomain.LastVerifiedAtForKnowledge(res.Knowledge); ok {
			entry.LastVerifiedAt = verifiedAt.Format(time.RFC3339)
		}
		entries = append(entries, entry)
	}
	return entries
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

func recalledEntriesToSearchResults(entries []activeMemoryEntry, results []memdomain.SearchResult) []memdomain.SearchResult {
	byID := make(map[string]memdomain.SearchResult, len(results))
	for _, result := range results {
		byID[result.ID] = result
	}
	selected := make([]memdomain.SearchResult, 0, len(entries))
	for _, entry := range entries {
		if result, ok := byID[entry.ID]; ok {
			selected = append(selected, result)
		}
	}
	return selected
}
