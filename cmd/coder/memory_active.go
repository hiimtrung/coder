package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	memdomain "github.com/trungtran/coder/internal/domain/memory"
)

const activeMemoryStateFile = "active-memory.json"

type activeMemoryState struct {
	Mode         string              `json:"mode,omitempty"`
	Query        string              `json:"query"`
	Trigger      string              `json:"trigger,omitempty"`
	Budget       int                 `json:"budget,omitempty"`
	Scope        string              `json:"scope,omitempty"`
	Type         string              `json:"type,omitempty"`
	Limit        int                 `json:"limit"`
	Status       string              `json:"status,omitempty"`
	CanonicalKey string              `json:"canonical_key,omitempty"`
	AsOf         string              `json:"as_of,omitempty"`
	IncludeStale bool                `json:"include_stale"`
	History      bool                `json:"history"`
	SearchedAt   time.Time           `json:"searched_at"`
	Keep         []string            `json:"keep,omitempty"`
	Add          []string            `json:"add,omitempty"`
	Drop         []string            `json:"drop,omitempty"`
	Coverage     string              `json:"coverage,omitempty"`
	Conflicts    []string            `json:"conflicts,omitempty"`
	Results      []activeMemoryEntry `json:"results"`
}

type activeMemoryEntry struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Type             string   `json:"type"`
	Scope            string   `json:"scope,omitempty"`
	Status           string   `json:"status"`
	CanonicalKey     string   `json:"canonical_key,omitempty"`
	Score            float32  `json:"score"`
	Tags             []string `json:"tags,omitempty"`
	LastVerifiedAt   string   `json:"last_verified_at,omitempty"`
	ConflictDetected bool     `json:"conflict_detected"`
	ConflictCount    int      `json:"conflict_count,omitempty"`
	Reason           string   `json:"reason,omitempty"`
	Content          string   `json:"content"`
}

func activeMemoryPath() string {
	return coderPath(activeMemoryStateFile)
}

func loadActiveMemoryState() (*activeMemoryState, error) {
	data, err := os.ReadFile(activeMemoryPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var state activeMemoryState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func saveActiveMemoryState(state *activeMemoryState) error {
	if err := os.MkdirAll(coderDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(activeMemoryPath(), data, 0644); err != nil {
		return err
	}
	return syncContextState(nil, state)
}

func buildActiveMemoryState(query, scope, memType string, limit int, status, canonicalKey, asOf string, includeStale, history bool, results []memdomain.SearchResult) *activeMemoryState {
	state := &activeMemoryState{
		Mode:         "search",
		Query:        query,
		Scope:        scope,
		Type:         memType,
		Limit:        limit,
		Status:       status,
		CanonicalKey: canonicalKey,
		AsOf:         asOf,
		IncludeStale: includeStale,
		History:      history,
		SearchedAt:   time.Now(),
		Results:      make([]activeMemoryEntry, 0, len(results)),
	}

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
		}
		if entry.ConflictDetected {
			entry.ConflictCount = metadataInt(res.Metadata[memdomain.MetadataKeyConflictCount])
		}
		if verifiedAt, ok := memdomain.LastVerifiedAtForKnowledge(res.Knowledge); ok {
			entry.LastVerifiedAt = verifiedAt.Format(time.RFC3339)
		}
		state.Results = append(state.Results, entry)
	}

	return state
}

func runMemoryActive(args []string) {
	logActivity("memory active")
	fs := flag.NewFlagSet("memory active", flag.ExitOnError)
	format := fs.String("format", "text", "Output format: text, json")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: coder memory active [flags]")
		fmt.Fprintln(os.Stderr, "\nFLAGS:")
		fs.PrintDefaults()
	}

	fs.Parse(args)

	state, err := loadActiveMemoryState()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load active memory state: %v\n", err)
		os.Exit(1)
	}
	if state == nil {
		empty := &activeMemoryState{Results: []activeMemoryEntry{}}
		if *format == "json" {
			writeJSON(empty)
			return
		}
		fmt.Println("No active memory state found. Run `coder memory search \"<query>\"` first.")
		return
	}

	switch *format {
	case "text":
		fmt.Printf("Active memory for %q\n", state.Query)
		if state.Mode != "" {
			fmt.Printf("Mode:     %s\n", state.Mode)
		}
		if state.Trigger != "" {
			fmt.Printf("Trigger:  %s\n", state.Trigger)
		}
		if !state.SearchedAt.IsZero() {
			fmt.Printf("Searched: %s\n", state.SearchedAt.Format("2006-01-02 15:04:05"))
		}
		fmt.Printf("Limit:    %d\n", state.Limit)
		if state.Budget > 0 {
			fmt.Printf("Budget:   %d\n", state.Budget)
		}
		if state.Scope != "" {
			fmt.Printf("Scope:    %s\n", state.Scope)
		}
		if state.Type != "" {
			fmt.Printf("Type:     %s\n", state.Type)
		}
		if state.Status != "" {
			fmt.Printf("Status:   %s\n", state.Status)
		}
		if state.CanonicalKey != "" {
			fmt.Printf("Key:      %s\n", state.CanonicalKey)
		}
		fmt.Printf("History:  %t\n", state.History)
		fmt.Printf("Stale:    %t\n", state.IncludeStale)
		if state.Coverage != "" {
			fmt.Printf("Coverage: %s\n", state.Coverage)
		}
		if len(state.Keep) > 0 || len(state.Add) > 0 || len(state.Drop) > 0 {
			fmt.Printf("Keep:     %s\n", fallbackList(state.Keep))
			fmt.Printf("Add:      %s\n", fallbackList(state.Add))
			fmt.Printf("Drop:     %s\n", fallbackList(state.Drop))
		}
		if len(state.Conflicts) > 0 {
			fmt.Printf("Conflicts:%s\n", formatIndentedList(state.Conflicts))
		}
		fmt.Printf("\n")
		for _, item := range state.Results {
			fmt.Printf("[%s] %s (Score: %.4f)\n", shortMemoryID(item.ID), item.Title, item.Score)
			fmt.Printf("  Type: %s | Status: %s | Key: %s\n", fallbackString(item.Type, "(unknown)"), fallbackString(item.Status, "(unknown)"), fallbackString(item.CanonicalKey, "(none)"))
			if item.Reason != "" {
				fmt.Printf("  %s\n", item.Reason)
			}
			if item.LastVerifiedAt != "" {
				fmt.Printf("  Verified: %s\n", item.LastVerifiedAt)
			}
			if item.ConflictDetected {
				fmt.Printf("  Conflict: %d active versions\n", item.ConflictCount)
			}
			fmt.Printf("  Tags: %s\n\n", strings.Join(item.Tags, ", "))
		}
	case "json":
		writeJSON(state)
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported format %q (supported: text, json)\n", *format)
		os.Exit(1)
	}
}

func renderRawMemoryContext(query string, results []memdomain.SearchResult) string {
	var sb strings.Builder
	for i, res := range results {
		if i > 0 {
			sb.WriteString("\n")
		}
		status := memdomain.StatusForKnowledge(res.Knowledge)
		key := memdomain.CanonicalKeyForKnowledge(res.Knowledge)
		sb.WriteString(fmt.Sprintf("<!-- coder-memory query=%q id=%q score=%.4f status=%q canonical_key=%q -->\n", query, res.ID, res.Score, status, key))
		sb.WriteString(fmt.Sprintf("# Memory: %s\n\n", res.Title))
		sb.WriteString(fmt.Sprintf("- id: %s\n", res.ID))
		sb.WriteString(fmt.Sprintf("- type: %s\n", res.Type))
		sb.WriteString(fmt.Sprintf("- scope: %s\n", res.Scope))
		sb.WriteString(fmt.Sprintf("- status: %s\n", status))
		if key != "" {
			sb.WriteString(fmt.Sprintf("- canonical_key: %s\n", key))
		}
		if verifiedAt, ok := memdomain.LastVerifiedAtForKnowledge(res.Knowledge); ok {
			sb.WriteString(fmt.Sprintf("- last_verified_at: %s\n", verifiedAt.Format(time.RFC3339)))
		}
		if len(res.Tags) > 0 {
			sb.WriteString(fmt.Sprintf("- tags: %s\n", strings.Join(res.Tags, ", ")))
		}
		if memdomain.MetadataBool(res.Metadata, memdomain.MetadataKeyConflictDetected) {
			sb.WriteString(fmt.Sprintf("- conflict_detected: true\n"))
			sb.WriteString(fmt.Sprintf("- conflict_count: %d\n", metadataInt(res.Metadata[memdomain.MetadataKeyConflictCount])))
		}
		sb.WriteString("\n")
		sb.WriteString(res.Content)
		if !strings.HasSuffix(res.Content, "\n") {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func shortMemoryID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

func metadataInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float32:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func fallbackString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func formatIndentedList(items []string) string {
	if len(items) == 0 {
		return " (none)"
	}
	return "\n  - " + strings.Join(items, "\n  - ")
}
