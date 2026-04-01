package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	memdomain "github.com/trungtran/coder/internal/domain/memory"
)

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

	result, err := mgr.Recall(context.Background(), memdomain.RecallOptions{
		Task:        task,
		Current:     currentItems,
		Trigger:     *trigger,
		Budget:      *budget,
		Limit:       searchLimit,
		Scope:       *scope,
		Type:        memdomain.MemoryType(*memType),
		MetaFilters: metaFilters,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	output := buildMemoryRecallOutput(result)
	state := buildActiveMemoryRecallState(result, *scope, *memType, *status, *key, *asOf, *includeStale, *history)
	if !*noSave {
		if err := saveActiveMemoryState(state); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to save active memory state: %v\n", err)
			os.Exit(1)
		}
	}

	switch *format {
	case "text":
		fmt.Print(renderMemoryRecallText(output))
	case "json":
		writeJSON(output)
	case "raw":
		selected := recalledMemoriesToSearchResults(result.Memories)
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
		identity := strings.ToLower(strings.TrimSpace(result.ID))
		if strings.TrimSpace(result.CanonicalKey) != "" {
			identity = strings.ToLower(strings.TrimSpace(result.CanonicalKey))
		}
		items = append(items, identity)
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
