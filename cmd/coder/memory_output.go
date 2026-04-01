package main

import (
	"fmt"
	"strings"
	"time"

	memdomain "github.com/trungtran/coder/internal/domain/memory"
)

type memorySearchOutput struct {
	Query        string                   `json:"query"`
	Scope        string                   `json:"scope,omitempty"`
	Type         string                   `json:"type,omitempty"`
	Limit        int                      `json:"limit"`
	Status       string                   `json:"status,omitempty"`
	Key          string                   `json:"key,omitempty"`
	AsOf         string                   `json:"as_of,omitempty"`
	IncludeStale bool                     `json:"include_stale"`
	History      bool                     `json:"history"`
	Results      []memdomain.SearchResult `json:"results"`
}

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

func buildMemorySearchOutput(query, scope, memType string, limit int, status, key, asOf string, includeStale, history bool, results []memdomain.SearchResult) memorySearchOutput {
	return memorySearchOutput{
		Query:        query,
		Scope:        scope,
		Type:         memType,
		Limit:        limit,
		Status:       status,
		Key:          key,
		AsOf:         asOf,
		IncludeStale: includeStale,
		History:      history,
		Results:      results,
	}
}

func buildMemoryRecallOutput(result memdomain.RecallResult) memoryRecallOutput {
	entries := make([]activeMemoryEntry, 0, len(result.Memories))
	for _, recalled := range result.Memories {
		entry := activeMemoryEntry{
			ID:               recalled.Result.ID,
			Title:            recalled.Result.Title,
			Type:             string(recalled.Result.Type),
			Scope:            recalled.Result.Scope,
			Status:           string(memdomain.StatusForKnowledge(recalled.Result.Knowledge)),
			CanonicalKey:     memdomain.CanonicalKeyForKnowledge(recalled.Result.Knowledge),
			Score:            recalled.Result.Score,
			Tags:             append([]string(nil), recalled.Result.Tags...),
			ConflictDetected: memdomain.MetadataBool(recalled.Result.Metadata, memdomain.MetadataKeyConflictDetected),
			Content:          recalled.Result.Content,
			Reason:           recalled.Reason,
		}
		if entry.ConflictDetected {
			entry.ConflictCount = metadataInt(recalled.Result.Metadata[memdomain.MetadataKeyConflictCount])
		}
		if verifiedAt, ok := memdomain.LastVerifiedAtForKnowledge(recalled.Result.Knowledge); ok {
			entry.LastVerifiedAt = verifiedAt.Format(time.RFC3339)
		}
		entries = append(entries, entry)
	}

	return memoryRecallOutput{
		Task:      result.Task,
		Trigger:   result.Trigger,
		Budget:    result.Budget,
		Coverage:  result.Coverage,
		Keep:      append([]string(nil), result.Keep...),
		Add:       append([]string(nil), result.Add...),
		Drop:      append([]string(nil), result.Drop...),
		Conflicts: append([]string(nil), result.Conflicts...),
		Memories:  entries,
	}
}

func buildActiveMemoryRecallState(result memdomain.RecallResult, scope, memType, status, canonicalKey, asOf string, includeStale, history bool) *activeMemoryState {
	output := buildMemoryRecallOutput(result)
	return &activeMemoryState{
		Mode:         "recall",
		Query:        result.Task,
		Trigger:      result.Trigger,
		Budget:       result.Budget,
		Scope:        scope,
		Type:         memType,
		Limit:        result.Limit,
		Status:       status,
		CanonicalKey: canonicalKey,
		AsOf:         asOf,
		IncludeStale: includeStale,
		History:      history,
		SearchedAt:   time.Now().UTC(),
		Keep:         append([]string(nil), result.Keep...),
		Add:          append([]string(nil), result.Add...),
		Drop:         append([]string(nil), result.Drop...),
		Coverage:     result.Coverage,
		Conflicts:    append([]string(nil), result.Conflicts...),
		Results:      output.Memories,
	}
}

func renderMemoryRecallText(output memoryRecallOutput) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Recalled %d active memory item(s) for %q [%s]:\n\n", len(output.Memories), output.Task, output.Trigger))
	for _, memory := range output.Memories {
		sb.WriteString(fmt.Sprintf("[%s] %s (Score: %.4f)\n", shortMemoryID(memory.ID), memory.Title, memory.Score))
		sb.WriteString(fmt.Sprintf("  %s\n", memory.Reason))
		sb.WriteString(fmt.Sprintf("  Status: %s | Key: %s\n\n", fallbackString(memory.Status, "(unknown)"), fallbackString(memory.CanonicalKey, "(none)")))
	}
	sb.WriteString(fmt.Sprintf("Coverage: %s\n", output.Coverage))
	sb.WriteString(fmt.Sprintf("Keep:     %s\n", fallbackList(output.Keep)))
	sb.WriteString(fmt.Sprintf("Add:      %s\n", fallbackList(output.Add)))
	sb.WriteString(fmt.Sprintf("Drop:     %s\n", fallbackList(output.Drop)))
	if len(output.Conflicts) > 0 {
		sb.WriteString(fmt.Sprintf("Conflicts:%s\n", formatIndentedList(output.Conflicts)))
	}
	return sb.String()
}

func recalledMemoriesToSearchResults(memories []memdomain.RecalledMemory) []memdomain.SearchResult {
	results := make([]memdomain.SearchResult, 0, len(memories))
	for _, memory := range memories {
		results = append(results, memory.Result)
	}
	return results
}
