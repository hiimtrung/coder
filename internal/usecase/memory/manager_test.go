package ucmemory

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	memdomain "github.com/trungtran/coder/internal/domain/memory"
)

type fakeEmbeddingProvider struct {
	vector []float32
}

func (p fakeEmbeddingProvider) GenerateEmbedding(_ context.Context, _ string) ([]float32, error) {
	return append([]float32(nil), p.vector...), nil
}

type fakeMemoryRepo struct {
	searchResults []memdomain.SearchResult
	listItems     []memdomain.Knowledge
	storeCalls    []*memdomain.Knowledge
	lastFilters   map[string]any
	activeByKey   map[string][]memdomain.Knowledge
	byID          map[string]memdomain.Knowledge
	updated       map[string]map[string]any
}

func newFakeMemoryRepo() *fakeMemoryRepo {
	return &fakeMemoryRepo{
		activeByKey: make(map[string][]memdomain.Knowledge),
		byID:        make(map[string]memdomain.Knowledge),
		updated:     make(map[string]map[string]any),
	}
}

func (r *fakeMemoryRepo) Store(_ context.Context, k *memdomain.Knowledge) error {
	copyKnowledge := *k
	copyKnowledge.Metadata = memdomain.CloneMetadata(k.Metadata)
	r.storeCalls = append(r.storeCalls, &copyKnowledge)
	r.byID[k.ID] = copyKnowledge
	return nil
}

func (r *fakeMemoryRepo) Search(_ context.Context, _ []float32, _ string, _ string, _ []string, _ memdomain.MemoryType, metaFilters map[string]any, _ int) ([]memdomain.SearchResult, error) {
	r.lastFilters = memdomain.CloneMetadata(metaFilters)
	return append([]memdomain.SearchResult(nil), r.searchResults...), nil
}

func (r *fakeMemoryRepo) List(_ context.Context, _ int, _ int) ([]memdomain.Knowledge, error) {
	items := make([]memdomain.Knowledge, 0, len(r.listItems))
	for _, item := range r.listItems {
		item.Metadata = memdomain.CloneMetadata(item.Metadata)
		items = append(items, item)
	}
	return items, nil
}

func (r *fakeMemoryRepo) Delete(_ context.Context, _ string) error {
	return nil
}

func (r *fakeMemoryRepo) Close() error {
	return nil
}

func (r *fakeMemoryRepo) Get(_ context.Context, id string) (*memdomain.Knowledge, error) {
	knowledge, ok := r.byID[id]
	if !ok {
		return nil, fmt.Errorf("memory %s not found", id)
	}
	return &knowledge, nil
}

func (r *fakeMemoryRepo) ListByParentID(_ context.Context, parentID string) ([]memdomain.Knowledge, error) {
	var results []memdomain.Knowledge
	for _, knowledge := range r.byID {
		if knowledge.ParentID == parentID || knowledge.ID == parentID {
			results = append(results, knowledge)
		}
	}
	return results, nil
}

func (r *fakeMemoryRepo) ListActiveByCanonicalKey(_ context.Context, canonicalKey string, _ string) ([]memdomain.Knowledge, error) {
	rows := r.activeByKey[canonicalKey]
	cloned := make([]memdomain.Knowledge, 0, len(rows))
	for _, row := range rows {
		row.Metadata = memdomain.CloneMetadata(row.Metadata)
		cloned = append(cloned, row)
	}
	return cloned, nil
}

func (r *fakeMemoryRepo) UpdateMetadata(_ context.Context, id string, metadata map[string]any, _ time.Time) error {
	r.updated[id] = memdomain.CloneMetadata(metadata)
	return nil
}

func TestStoreAssignsLifecycleDefaults(t *testing.T) {
	repo := newFakeMemoryRepo()
	manager := NewManager(repo, fakeEmbeddingProvider{vector: []float32{1, 0, 0}})

	_, err := manager.Store(context.Background(), "Auth decision", "Use rotating refresh tokens", memdomain.TypeDecision, nil, "backend", []string{"auth"})
	if err != nil {
		t.Fatalf("Store() returned error: %v", err)
	}
	if len(repo.storeCalls) != 1 {
		t.Fatalf("expected 1 store call, got %d", len(repo.storeCalls))
	}

	stored := repo.storeCalls[0]
	if got := memdomain.StatusFromMetadata(stored.Metadata); got != memdomain.StatusActive {
		t.Fatalf("expected active status, got %q", got)
	}
	wantKey := "decision:auth-decision"
	if got := memdomain.MetadataString(stored.Metadata, memdomain.MetadataKeyCanonicalKey); got != wantKey {
		t.Fatalf("expected canonical key %q, got %q", wantKey, got)
	}
}

func TestStoreReplaceActiveSupersedesExistingRows(t *testing.T) {
	repo := newFakeMemoryRepo()
	oldRows := []memdomain.Knowledge{
		{
			ID:       "old-chunk-1",
			ParentID: "old-parent",
			Type:     memdomain.TypeDecision,
			Metadata: map[string]any{
				memdomain.MetadataKeyStatus:       string(memdomain.StatusActive),
				memdomain.MetadataKeyCanonicalKey: "decision:auth-decision",
			},
		},
		{
			ID:       "old-chunk-2",
			ParentID: "old-parent",
			Type:     memdomain.TypeDecision,
			Metadata: map[string]any{
				memdomain.MetadataKeyStatus:       string(memdomain.StatusActive),
				memdomain.MetadataKeyCanonicalKey: "decision:auth-decision",
			},
		},
	}
	repo.activeByKey["decision:auth-decision"] = oldRows

	manager := NewManager(repo, fakeEmbeddingProvider{vector: []float32{1, 0, 0}})
	_, err := manager.Store(
		context.Background(),
		"Auth decision",
		"Use rotating refresh tokens",
		memdomain.TypeDecision,
		map[string]any{
			memdomain.ControlKeyReplaceActive: true,
		},
		"backend",
		nil,
	)
	if err != nil {
		t.Fatalf("Store() returned error: %v", err)
	}
	if len(repo.storeCalls) != 1 {
		t.Fatalf("expected 1 new store call, got %d", len(repo.storeCalls))
	}

	newParentID := repo.storeCalls[0].ID
	if got := memdomain.MetadataString(repo.storeCalls[0].Metadata, memdomain.MetadataKeySupersedesID); got != "old-parent" {
		t.Fatalf("expected supersedes id %q, got %q", "old-parent", got)
	}

	for _, row := range oldRows {
		metadata, ok := repo.updated[row.ID]
		if !ok {
			t.Fatalf("expected row %q to be updated", row.ID)
		}
		if got := memdomain.StatusFromMetadata(metadata); got != memdomain.StatusSuperseded {
			t.Fatalf("expected row %q to become superseded, got %q", row.ID, got)
		}
		if got := memdomain.MetadataString(metadata, memdomain.MetadataKeySupersededByID); got != newParentID {
			t.Fatalf("expected row %q superseded_by_id %q, got %q", row.ID, newParentID, got)
		}
		if _, ok := memdomain.MetadataTime(metadata, memdomain.MetadataKeyValidTo); !ok {
			t.Fatalf("expected row %q valid_to to be set", row.ID)
		}
	}
}

func TestSearchDefaultsToActiveOnlyAndCollapsesByCanonicalKey(t *testing.T) {
	repo := newFakeMemoryRepo()
	now := time.Now().UTC()
	repo.searchResults = []memdomain.SearchResult{
		{
			Score: 0.95,
			Knowledge: memdomain.Knowledge{
				ID:      "old",
				Title:   "Old auth decision",
				Type:    memdomain.TypeDecision,
				Content: "Do not rotate refresh tokens",
				Metadata: map[string]any{
					memdomain.MetadataKeyStatus:       string(memdomain.StatusSuperseded),
					memdomain.MetadataKeyCanonicalKey: "decision:auth-decision",
				},
				UpdatedAt: now.Add(-24 * time.Hour),
			},
		},
		{
			Score: 0.92,
			Knowledge: memdomain.Knowledge{
				ID:      "new",
				Title:   "Current auth decision",
				Type:    memdomain.TypeDecision,
				Content: "Rotate refresh tokens",
				Metadata: map[string]any{
					memdomain.MetadataKeyStatus:         string(memdomain.StatusActive),
					memdomain.MetadataKeyCanonicalKey:   "decision:auth-decision",
					memdomain.MetadataKeyLastVerifiedAt: now.Format(time.RFC3339),
					memdomain.MetadataKeyConfidence:     1.0,
				},
				UpdatedAt: now,
			},
		},
		{
			Score: 0.5,
			Knowledge: memdomain.Knowledge{
				ID:      "other",
				Title:   "Queue pattern",
				Type:    memdomain.TypePattern,
				Content: "Use a worker queue",
				Metadata: map[string]any{
					memdomain.MetadataKeyStatus:       string(memdomain.StatusActive),
					memdomain.MetadataKeyCanonicalKey: "pattern:queue-worker",
				},
				UpdatedAt: now,
			},
		},
	}

	manager := NewManager(repo, fakeEmbeddingProvider{vector: []float32{1, 0, 0}})
	results, err := manager.Search(context.Background(), "auth", "", nil, "", nil, 5)
	if err != nil {
		t.Fatalf("Search() returned error: %v", err)
	}

	if got := memdomain.MetadataString(repo.lastFilters, memdomain.FilterKeyStatus); got != string(memdomain.StatusActive) {
		t.Fatalf("expected default status filter %q, got %q", memdomain.StatusActive, got)
	}
	if memdomain.MetadataString(repo.lastFilters, memdomain.FilterKeyAsOf) == "" {
		t.Fatalf("expected default as-of filter to be set")
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 collapsed results, got %d", len(results))
	}
	if results[0].ID != "new" {
		t.Fatalf("expected active result to rank first, got %q", results[0].ID)
	}
}

func TestSearchSummarizesConflictingActiveVersions(t *testing.T) {
	repo := newFakeMemoryRepo()
	now := time.Now().UTC()
	repo.searchResults = []memdomain.SearchResult{
		{
			Score: 0.9,
			Knowledge: memdomain.Knowledge{
				ID:       "version-a",
				ParentID: "version-a",
				Title:    "Current auth decision",
				Type:     memdomain.TypeDecision,
				Content:  "Rotate refresh tokens for every device.",
				Metadata: map[string]any{
					memdomain.MetadataKeyStatus:       string(memdomain.StatusActive),
					memdomain.MetadataKeyCanonicalKey: "decision:auth-decision",
				},
				ContentHash: "hash-a",
				UpdatedAt:   now,
			},
		},
		{
			Score: 0.89,
			Knowledge: memdomain.Knowledge{
				ID:       "version-b",
				ParentID: "version-b",
				Title:    "Current auth decision",
				Type:     memdomain.TypeDecision,
				Content:  "Use non-rotating refresh tokens.",
				Metadata: map[string]any{
					memdomain.MetadataKeyStatus:       string(memdomain.StatusActive),
					memdomain.MetadataKeyCanonicalKey: "decision:auth-decision",
				},
				ContentHash: "hash-b",
				UpdatedAt:   now.Add(-time.Hour),
			},
		},
	}

	manager := NewManager(repo, fakeEmbeddingProvider{vector: []float32{1, 0, 0}})
	results, err := manager.Search(context.Background(), "auth", "", nil, "", nil, 5)
	if err != nil {
		t.Fatalf("Search() returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 summarized result, got %d", len(results))
	}
	if !memdomain.MetadataBool(results[0].Metadata, memdomain.MetadataKeyConflictDetected) {
		t.Fatalf("expected conflict_detected metadata to be set")
	}
	if !strings.Contains(results[0].Content, "Multiple active memories disagree") {
		t.Fatalf("expected conflict summary content, got %q", results[0].Content)
	}
	if titles := memdomain.MetadataStringSlice(results[0].Metadata, memdomain.MetadataKeyConflictTitles); len(titles) != 2 {
		t.Fatalf("expected 2 conflict titles, got %v", titles)
	}
}

func TestVerifyUpdatesVersionGroupLifecycleMetadata(t *testing.T) {
	repo := newFakeMemoryRepo()
	repo.byID["group"] = memdomain.Knowledge{
		ID:       "group",
		ParentID: "group",
		Metadata: map[string]any{
			memdomain.MetadataKeyStatus:       string(memdomain.StatusActive),
			memdomain.MetadataKeyCanonicalKey: "decision:auth-decision",
		},
	}
	repo.byID["chunk-1"] = memdomain.Knowledge{
		ID:       "chunk-1",
		ParentID: "group",
		Metadata: map[string]any{
			memdomain.MetadataKeyStatus:       string(memdomain.StatusActive),
			memdomain.MetadataKeyCanonicalKey: "decision:auth-decision",
		},
	}

	manager := NewManager(repo, fakeEmbeddingProvider{vector: []float32{1, 0, 0}})
	confidence := 0.85
	updated, err := manager.Verify(context.Background(), "group", memdomain.VerifyOptions{
		VerifiedBy: "phase-3",
		Confidence: &confidence,
		SourceRef:  "docs/memory_lifecycle_plan.md",
	})
	if err != nil {
		t.Fatalf("Verify() returned error: %v", err)
	}
	if updated != 2 {
		t.Fatalf("expected 2 updated rows, got %d", updated)
	}
	for _, id := range []string{"group", "chunk-1"} {
		metadata := repo.updated[id]
		if metadata == nil {
			t.Fatalf("expected metadata update for %q", id)
		}
		if _, ok := memdomain.MetadataTime(metadata, memdomain.MetadataKeyLastVerifiedAt); !ok {
			t.Fatalf("expected last_verified_at for %q", id)
		}
		if got := memdomain.MetadataString(metadata, memdomain.MetadataKeyVerifiedBy); got != "phase-3" {
			t.Fatalf("expected verified_by for %q, got %q", id, got)
		}
	}
}

func TestSupersedeUpdatesVersionChains(t *testing.T) {
	repo := newFakeMemoryRepo()
	repo.byID["old"] = memdomain.Knowledge{
		ID:       "old",
		ParentID: "old",
		Metadata: map[string]any{
			memdomain.MetadataKeyStatus:       string(memdomain.StatusActive),
			memdomain.MetadataKeyCanonicalKey: "decision:auth-decision",
		},
	}
	repo.byID["new"] = memdomain.Knowledge{
		ID:       "new",
		ParentID: "new",
		Metadata: map[string]any{
			memdomain.MetadataKeyStatus:       string(memdomain.StatusDraft),
			memdomain.MetadataKeyCanonicalKey: "decision:auth-decision",
			memdomain.MetadataKeyValidTo:      time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339),
		},
	}

	manager := NewManager(repo, fakeEmbeddingProvider{vector: []float32{1, 0, 0}})
	updated, err := manager.Supersede(context.Background(), "old", "new")
	if err != nil {
		t.Fatalf("Supersede() returned error: %v", err)
	}
	if updated != 2 {
		t.Fatalf("expected 2 updated rows, got %d", updated)
	}
	if got := memdomain.StatusFromMetadata(repo.updated["old"]); got != memdomain.StatusSuperseded {
		t.Fatalf("expected old version to become superseded, got %q", got)
	}
	if got := memdomain.StatusFromMetadata(repo.updated["new"]); got != memdomain.StatusActive {
		t.Fatalf("expected new version to become active, got %q", got)
	}
	if _, ok := memdomain.MetadataTime(repo.updated["new"], memdomain.MetadataKeyValidTo); ok {
		t.Fatalf("expected replacement valid_to to be cleared")
	}
}

func TestAuditReportsConflictsExpiredAndUnverified(t *testing.T) {
	repo := newFakeMemoryRepo()
	now := time.Now().UTC()
	repo.listItems = []memdomain.Knowledge{
		{
			ID:       "conflict-a",
			ParentID: "conflict-a",
			Title:    "Auth decision A",
			Type:     memdomain.TypeDecision,
			Content:  "Rotate refresh tokens",
			Metadata: map[string]any{
				memdomain.MetadataKeyStatus:       string(memdomain.StatusActive),
				memdomain.MetadataKeyCanonicalKey: "decision:auth-decision",
			},
			ContentHash: "hash-a",
			UpdatedAt:   now,
		},
		{
			ID:       "conflict-b",
			ParentID: "conflict-b",
			Title:    "Auth decision B",
			Type:     memdomain.TypeDecision,
			Content:  "Do not rotate refresh tokens",
			Metadata: map[string]any{
				memdomain.MetadataKeyStatus:       string(memdomain.StatusActive),
				memdomain.MetadataKeyCanonicalKey: "decision:auth-decision",
			},
			ContentHash: "hash-b",
			UpdatedAt:   now.Add(-time.Hour),
		},
		{
			ID:       "expired",
			ParentID: "expired",
			Title:    "Old incident",
			Type:     memdomain.TypeEvent,
			Content:  "Temporary outage",
			Metadata: map[string]any{
				memdomain.MetadataKeyStatus:       string(memdomain.StatusActive),
				memdomain.MetadataKeyCanonicalKey: "event:old-incident",
				memdomain.MetadataKeyValidTo:      now.Add(-time.Hour).Format(time.RFC3339),
			},
			UpdatedAt: now,
		},
		{
			ID:       "unverified",
			ParentID: "unverified",
			Title:    "Queue pattern",
			Type:     memdomain.TypePattern,
			Content:  "Use a worker queue",
			Metadata: map[string]any{
				memdomain.MetadataKeyStatus:         string(memdomain.StatusActive),
				memdomain.MetadataKeyCanonicalKey:   "pattern:queue-worker",
				memdomain.MetadataKeyLastVerifiedAt: now.Add(-400 * 24 * time.Hour).Format(time.RFC3339),
			},
			UpdatedAt: now,
		},
	}

	manager := NewManager(repo, fakeEmbeddingProvider{vector: []float32{1, 0, 0}})
	report, err := manager.Audit(context.Background(), memdomain.AuditOptions{UnverifiedDays: 180})
	if err != nil {
		t.Fatalf("Audit() returned error: %v", err)
	}
	if len(report.Findings) < 3 {
		t.Fatalf("expected at least 3 findings, got %d", len(report.Findings))
	}

	var foundConflict bool
	var foundExpired bool
	var foundUnverified bool
	for _, finding := range report.Findings {
		switch finding.Type {
		case memdomain.AuditFindingActiveConflict:
			foundConflict = true
		case memdomain.AuditFindingExpiredActive:
			foundExpired = true
		case memdomain.AuditFindingActiveUnverified:
			foundUnverified = true
		}
	}
	if !foundConflict || !foundExpired || !foundUnverified {
		t.Fatalf("expected conflict, expired, and unverified findings, got %+v", report.Findings)
	}
}

func TestRecallComputesKeepAddDropAndCoverage(t *testing.T) {
	repo := newFakeMemoryRepo()
	now := time.Now().UTC()
	repo.searchResults = []memdomain.SearchResult{
		{
			Score: 0.91,
			Knowledge: memdomain.Knowledge{
				ID:      "auth-memory",
				Title:   "Auth token ADR",
				Type:    memdomain.TypeDecision,
				Content: "Rotate refresh tokens.",
				Metadata: map[string]any{
					memdomain.MetadataKeyStatus:       string(memdomain.StatusActive),
					memdomain.MetadataKeyCanonicalKey: "decision:auth-token",
				},
				UpdatedAt: now,
			},
		},
		{
			Score: 0.66,
			Knowledge: memdomain.Knowledge{
				ID:      "grpc-memory",
				Title:   "gRPC retry incident",
				Type:    memdomain.TypeEvent,
				Content: "Retries are capped at 3.",
				Metadata: map[string]any{
					memdomain.MetadataKeyStatus:           string(memdomain.StatusActive),
					memdomain.MetadataKeyCanonicalKey:     "event:grpc-retry-incident",
					memdomain.MetadataKeyConflictDetected: true,
					memdomain.MetadataKeyConflictCount:    2,
				},
				UpdatedAt: now.Add(-time.Minute),
			},
		},
	}

	manager := NewManager(repo, fakeEmbeddingProvider{vector: []float32{1, 0, 0}})
	result, err := manager.Recall(context.Background(), memdomain.RecallOptions{
		Task:    "auth grpc flow",
		Current: []string{"decision:auth-token", "stale:key"},
		Trigger: "execution",
		Budget:  2,
	})
	if err != nil {
		t.Fatalf("Recall() returned error: %v", err)
	}
	if result.Coverage != "strong" {
		t.Fatalf("expected strong coverage, got %q", result.Coverage)
	}
	if !reflect.DeepEqual(result.Keep, []string{"decision:auth-token"}) {
		t.Fatalf("keep = %v", result.Keep)
	}
	if !reflect.DeepEqual(result.Add, []string{"event:grpc-retry-incident"}) {
		t.Fatalf("add = %v", result.Add)
	}
	if !reflect.DeepEqual(result.Drop, []string{"stale:key"}) {
		t.Fatalf("drop = %v", result.Drop)
	}
	if !reflect.DeepEqual(result.Conflicts, []string{"event:grpc-retry-incident"}) {
		t.Fatalf("conflicts = %v", result.Conflicts)
	}
	if len(result.Memories) != 2 {
		t.Fatalf("expected 2 memories, got %d", len(result.Memories))
	}
	if result.Memories[0].Reason == "" {
		t.Fatal("expected recall reason to be populated")
	}
}
