package main

import (
	"reflect"
	"testing"
	"time"

	memdomain "github.com/trungtran/coder/internal/domain/memory"
)

func TestNormalizeMemoryIdentifiers(t *testing.T) {
	got := normalizeMemoryIdentifiers(" auth-token, Decision-Key ,auth-token,, MEMORY-1 ")
	want := []string{"auth-token", "decision-key", "memory-1"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeMemoryIdentifiers() = %v, want %v", got, want)
	}
}

func TestSelectRecalledMemory(t *testing.T) {
	results := []memdomain.SearchResult{
		{Knowledge: memdomain.Knowledge{ID: "1", Title: "Auth token ADR", CanonicalKey: "auth-token"}, Score: 0.91},
		{Knowledge: memdomain.Knowledge{ID: "2", Title: "GRPC retries", CanonicalKey: "grpc-retries"}, Score: 0.82},
		{Knowledge: memdomain.Knowledge{ID: "3", Title: "Legacy note", CanonicalKey: "legacy-note"}, Score: 0.41},
	}

	selected, keep, add, drop, conflicts := selectRecalledMemory(results, []string{"auth-token", "stale-key"}, 2)

	if len(selected) != 2 {
		t.Fatalf("len(selected) = %d, want 2", len(selected))
	}
	if !reflect.DeepEqual(keep, []string{"auth-token"}) {
		t.Fatalf("keep = %v, want %v", keep, []string{"auth-token"})
	}
	if !reflect.DeepEqual(add, []string{"grpc-retries"}) {
		t.Fatalf("add = %v, want %v", add, []string{"grpc-retries"})
	}
	if !reflect.DeepEqual(drop, []string{"stale-key"}) {
		t.Fatalf("drop = %v, want %v", drop, []string{"stale-key"})
	}
	if len(conflicts) != 0 {
		t.Fatalf("conflicts = %v, want empty", conflicts)
	}
}

func TestBuildActiveMemoryStateCapturesConflictAndVerification(t *testing.T) {
	verifiedAt := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	results := []memdomain.SearchResult{
		{
			Knowledge: memdomain.Knowledge{
				ID:             "memory-12345678",
				Title:          "Auth rollout incident",
				Type:           memdomain.TypeEvent,
				Scope:          "repo",
				CanonicalKey:   "auth-rollout-incident",
				LastVerifiedAt: &verifiedAt,
				Metadata: map[string]any{
					memdomain.MetadataKeyConflictDetected: true,
					memdomain.MetadataKeyConflictCount:    2,
				},
			},
			Score: 0.77,
		},
	}

	state := buildActiveMemoryState("auth rollout", "repo", string(memdomain.TypeEvent), 5, "active", "auth-rollout-incident", "", false, false, results)

	if state.Mode != "search" {
		t.Fatalf("state.Mode = %q, want %q", state.Mode, "search")
	}
	if len(state.Results) != 1 {
		t.Fatalf("len(state.Results) = %d, want 1", len(state.Results))
	}
	entry := state.Results[0]
	if !entry.ConflictDetected || entry.ConflictCount != 2 {
		t.Fatalf("conflict state = (%v, %d), want (true, 2)", entry.ConflictDetected, entry.ConflictCount)
	}
	if entry.LastVerifiedAt != verifiedAt.Format(time.RFC3339) {
		t.Fatalf("LastVerifiedAt = %q, want %q", entry.LastVerifiedAt, verifiedAt.Format(time.RFC3339))
	}
}

func TestBuildMemoryRecallOutput(t *testing.T) {
	results := []memdomain.SearchResult{
		{Knowledge: memdomain.Knowledge{ID: "1", Title: "Auth token ADR", CanonicalKey: "auth-token"}, Score: 0.91},
		{Knowledge: memdomain.Knowledge{ID: "2", Title: "Incident summary", CanonicalKey: "incident-summary"}, Score: 0.57},
	}

	output, state := buildMemoryRecallOutput("auth flow", "execution", 2, "repo", "decision", "", "", "", false, false, []string{"auth-token", "stale-key"}, results)

	if output.Coverage != "strong" {
		t.Fatalf("output.Coverage = %q, want %q", output.Coverage, "strong")
	}
	if !reflect.DeepEqual(output.Keep, []string{"auth-token"}) {
		t.Fatalf("output.Keep = %v, want %v", output.Keep, []string{"auth-token"})
	}
	if !reflect.DeepEqual(output.Add, []string{"incident-summary"}) {
		t.Fatalf("output.Add = %v, want %v", output.Add, []string{"incident-summary"})
	}
	if !reflect.DeepEqual(output.Drop, []string{"stale-key"}) {
		t.Fatalf("output.Drop = %v, want %v", output.Drop, []string{"stale-key"})
	}
	if state.Mode != "recall" || state.Trigger != "execution" {
		t.Fatalf("state = (%q, %q), want (%q, %q)", state.Mode, state.Trigger, "recall", "execution")
	}
	if len(state.Results) != 2 {
		t.Fatalf("len(state.Results) = %d, want 2", len(state.Results))
	}
	if state.Results[0].Reason == "" {
		t.Fatal("expected recall result reason to be populated")
	}
}
