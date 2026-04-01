package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
	result := memdomain.RecallResult{
		Task:     "auth flow",
		Trigger:  "execution",
		Budget:   2,
		Limit:    6,
		Coverage: "strong",
		Keep:     []string{"auth-token"},
		Add:      []string{"incident-summary"},
		Drop:     []string{"stale-key"},
		Memories: []memdomain.RecalledMemory{
			{Result: memdomain.SearchResult{Knowledge: memdomain.Knowledge{ID: "1", Title: "Auth token ADR", CanonicalKey: "auth-token"}, Score: 0.91}, Reason: "kept active: still relevant to the current task"},
			{Result: memdomain.SearchResult{Knowledge: memdomain.Knowledge{ID: "2", Title: "Incident summary", CanonicalKey: "incident-summary"}, Score: 0.57}, Reason: "added: recalled for the current task"},
		},
	}

	output := buildMemoryRecallOutput(result)
	state := buildActiveMemoryRecallState(result, "repo", "decision", "", "", "", false, false)

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

func TestBuildMemorySearchOutput(t *testing.T) {
	results := []memdomain.SearchResult{{Knowledge: memdomain.Knowledge{ID: "1", Title: "Auth ADR"}, Score: 0.8}}
	output := buildMemorySearchOutput("auth", "repo", "decision", 5, "active", "decision:auth", "2026-04-01T00:00:00Z", false, false, results)

	if output.Query != "auth" || output.Scope != "repo" || output.Type != "decision" {
		t.Fatalf("unexpected output header: %+v", output)
	}
	if len(output.Results) != 1 || output.Results[0].ID != "1" {
		t.Fatalf("unexpected results: %+v", output.Results)
	}
}

func TestRenderMemoryRecallText(t *testing.T) {
	output := memoryRecallOutput{
		Task:     "auth flow",
		Trigger:  "execution",
		Coverage: "adequate",
		Keep:     []string{"auth-token"},
		Add:      []string{"grpc-retry"},
		Drop:     []string{"stale-key"},
		Memories: []activeMemoryEntry{{ID: "12345678", Title: "Auth token ADR", Score: 0.91, Reason: "kept active", Status: "active", CanonicalKey: "auth-token"}},
	}

	text := renderMemoryRecallText(output)
	for _, snippet := range []string{"Recalled 1 active memory item(s)", "Auth token ADR", "Coverage: adequate", "Keep:     auth-token"} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("expected %q in output %q", snippet, text)
		}
	}
}

func TestSaveActiveStateSyncsContextState(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir() error: %v", err)
	}

	skillState := &activeSkillState{Task: "backend task", Trigger: "execution", Budget: 3, Skills: []activeSkillEntry{{Name: "architecture"}}}
	if err := saveActiveSkillState(skillState); err != nil {
		t.Fatalf("saveActiveSkillState() error: %v", err)
	}
	memoryState := &activeMemoryState{Query: "auth", Results: []activeMemoryEntry{{ID: "mem-1", Title: "Auth ADR"}}}
	if err := saveActiveMemoryState(memoryState); err != nil {
		t.Fatalf("saveActiveMemoryState() error: %v", err)
	}

	state, err := loadContextState()
	if err != nil {
		t.Fatalf("loadContextState() error: %v", err)
	}
	if state == nil || state.Skills == nil || state.Memory == nil {
		t.Fatalf("expected both skills and memory in context state: %+v", state)
	}
	if state.Skills.Task != "backend task" || state.Memory.Query != "auth" {
		t.Fatalf("unexpected context state payload: %+v", state)
	}
	if _, err := os.Stat(filepath.Join(tempDir, ".coder", contextStateFile)); err != nil {
		t.Fatalf("expected context-state file to exist: %v", err)
	}
}
