package memory

import (
	"testing"
	"time"
)

func TestNormalizeCanonicalKey(t *testing.T) {
	got := NormalizeCanonicalKey(TypeDecision, "Auth decision: rotate refresh tokens")
	want := "decision:auth-decision-rotate-refresh-tokens"
	if got != want {
		t.Fatalf("NormalizeCanonicalKey() = %q, want %q", got, want)
	}
}

func TestNormalizeSearchFiltersDefaultsToActiveAtCurrentTime(t *testing.T) {
	now := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
	filters := NormalizeSearchFilters(nil, now)

	if MetadataString(filters, FilterKeyStatus) != string(StatusActive) {
		t.Fatalf("expected default status %q, got %q", StatusActive, MetadataString(filters, FilterKeyStatus))
	}

	if MetadataString(filters, FilterKeyAsOf) != now.Format(time.RFC3339) {
		t.Fatalf("expected default as-of %q, got %q", now.Format(time.RFC3339), MetadataString(filters, FilterKeyAsOf))
	}
}

func TestNormalizeSearchFiltersKeepsExplicitStaleSearch(t *testing.T) {
	now := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
	filters := NormalizeSearchFilters(map[string]any{
		FilterKeyIncludeStale: true,
	}, now)

	if MetadataString(filters, FilterKeyStatus) != "" {
		t.Fatalf("expected no default status for stale search, got %q", MetadataString(filters, FilterKeyStatus))
	}
	if MetadataString(filters, FilterKeyAsOf) != "" {
		t.Fatalf("expected no default as-of for stale search, got %q", MetadataString(filters, FilterKeyAsOf))
	}
}

func TestHydrateKnowledgeLifecyclePromotesMetadataToFields(t *testing.T) {
	validTo := "2026-03-28T00:00:00Z"
	knowledge := Knowledge{
		Title: "Auth decision",
		Type:  TypeDecision,
		Metadata: map[string]any{
			MetadataKeyStatus:         string(StatusSuperseded),
			MetadataKeyCanonicalKey:   "decision:auth-refresh",
			MetadataKeySupersedesID:   "old-parent",
			MetadataKeyLastVerifiedAt: validTo,
			MetadataKeyConfidence:     0.85,
		},
	}

	HydrateKnowledgeLifecycle(&knowledge)

	if knowledge.Status != StatusSuperseded {
		t.Fatalf("expected status %q, got %q", StatusSuperseded, knowledge.Status)
	}
	if knowledge.CanonicalKey != "decision:auth-refresh" {
		t.Fatalf("expected canonical key to be hydrated, got %q", knowledge.CanonicalKey)
	}
	if knowledge.SupersedesID != "old-parent" {
		t.Fatalf("expected supersedes id to be hydrated, got %q", knowledge.SupersedesID)
	}
	if knowledge.LastVerifiedAt == nil || knowledge.LastVerifiedAt.Format(time.RFC3339) != validTo {
		t.Fatalf("expected last_verified_at to be hydrated, got %+v", knowledge.LastVerifiedAt)
	}
	if knowledge.Confidence == nil || *knowledge.Confidence != 0.85 {
		t.Fatalf("expected confidence to be hydrated, got %+v", knowledge.Confidence)
	}
}

func TestHydrateKnowledgeLifecycleSyncsFieldsBackToMetadata(t *testing.T) {
	validFrom := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
	confidence := 0.9
	knowledge := Knowledge{
		Title:           "Queue pattern",
		Type:            TypePattern,
		Status:          StatusActive,
		CanonicalKey:    "pattern:queue-worker",
		ValidFrom:       &validFrom,
		Confidence:      &confidence,
		VerifiedBy:      "phase2-backfill",
		Metadata:        map[string]any{},
		NormalizedTitle: "queue-pattern",
	}

	HydrateKnowledgeLifecycle(&knowledge)

	if got := MetadataString(knowledge.Metadata, MetadataKeyCanonicalKey); got != "pattern:queue-worker" {
		t.Fatalf("expected canonical key metadata to be synced, got %q", got)
	}
	if got := MetadataString(knowledge.Metadata, MetadataKeyVerifiedBy); got != "phase2-backfill" {
		t.Fatalf("expected verified_by metadata to be synced, got %q", got)
	}
	if got := MetadataString(knowledge.Metadata, MetadataKeyValidFrom); got != validFrom.Format(time.RFC3339) {
		t.Fatalf("expected valid_from metadata to be synced, got %q", got)
	}
}
