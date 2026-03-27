package memory

import "time"

type VerifyOptions struct {
	VerifiedAt time.Time `json:"verified_at"`
	VerifiedBy string    `json:"verified_by,omitempty"`
	Confidence *float64  `json:"confidence,omitempty"`
	SourceRef  string    `json:"source_ref,omitempty"`
}

type AuditFindingType string

const (
	AuditFindingActiveConflict   AuditFindingType = "active_conflict"
	AuditFindingExpiredActive    AuditFindingType = "expired_active"
	AuditFindingActiveUnverified AuditFindingType = "active_unverified"
	AuditFindingMissingLifecycle AuditFindingType = "missing_lifecycle"
)

type AuditOptions struct {
	Scope          string `json:"scope,omitempty"`
	UnverifiedDays int    `json:"unverified_days,omitempty"`
}

type AuditFinding struct {
	Type         AuditFindingType `json:"type"`
	CanonicalKey string           `json:"canonical_key,omitempty"`
	Scope        string           `json:"scope,omitempty"`
	VersionIDs   []string         `json:"version_ids,omitempty"`
	Titles       []string         `json:"titles,omitempty"`
	Details      string           `json:"details,omitempty"`
	Count        int              `json:"count"`
}

type AuditReport struct {
	GeneratedAt time.Time      `json:"generated_at"`
	Findings    []AuditFinding `json:"findings"`
}
