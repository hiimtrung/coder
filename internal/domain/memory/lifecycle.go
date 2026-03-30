package memory

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type LifecycleStatus string

const (
	StatusActive     LifecycleStatus = "active"
	StatusSuperseded LifecycleStatus = "superseded"
	StatusExpired    LifecycleStatus = "expired"
	StatusArchived   LifecycleStatus = "archived"
	StatusDraft      LifecycleStatus = "draft"
)

const (
	MetadataKeyStatus         = "status"
	MetadataKeyCanonicalKey   = "canonical_key"
	MetadataKeySupersedesID   = "supersedes_id"
	MetadataKeySupersededByID = "superseded_by_id"
	MetadataKeyValidFrom      = "valid_from"
	MetadataKeyValidTo        = "valid_to"
	MetadataKeyLastVerifiedAt = "last_verified_at"
	MetadataKeyConfidence     = "confidence"
	MetadataKeySourceRef      = "source_ref"
	MetadataKeyVerifiedBy     = "verified_by"
	ControlKeyReplaceActive   = "__replace_active"
	FilterKeyStatus           = "__status"
	FilterKeyIncludeStale     = "__include_stale"
	FilterKeyAsOf             = "__as_of"
	FilterKeyCanonicalKey     = "__canonical_key"
	FilterKeyHistory          = "__history"
)

func EnsureMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return make(map[string]any)
	}
	return metadata
}

func CloneMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return make(map[string]any)
	}

	cloned := make(map[string]any, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}

func NormalizeCanonicalKey(memType MemoryType, title string) string {
	kind := strings.TrimSpace(string(memType))
	if kind == "" {
		kind = string(TypeDocument)
	}

	title = strings.ToLower(strings.TrimSpace(title))
	var b strings.Builder
	lastDash := false
	for _, r := range title {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}

	normalizedTitle := strings.Trim(b.String(), "-")
	if normalizedTitle == "" {
		normalizedTitle = "untitled"
	}

	return kind + ":" + normalizedTitle
}

func CanonicalKeyForKnowledge(k Knowledge) string {
	if strings.TrimSpace(k.CanonicalKey) != "" {
		return strings.TrimSpace(k.CanonicalKey)
	}
	if key := MetadataString(k.Metadata, MetadataKeyCanonicalKey); key != "" {
		return key
	}
	return NormalizeCanonicalKey(k.Type, firstNonEmpty(k.NormalizedTitle, k.Title))
}

func VersionGroupID(k Knowledge) string {
	if strings.TrimSpace(k.ParentID) != "" {
		return k.ParentID
	}
	return k.ID
}

func StatusFromMetadata(metadata map[string]any) LifecycleStatus {
	return normalizeStatus(LifecycleStatus(strings.ToLower(strings.TrimSpace(MetadataString(metadata, MetadataKeyStatus)))))
}

func StatusForKnowledge(k Knowledge) LifecycleStatus {
	if k.Status != "" {
		return normalizeStatus(k.Status)
	}
	status := strings.ToLower(strings.TrimSpace(MetadataString(k.Metadata, MetadataKeyStatus)))
	return normalizeStatus(LifecycleStatus(status))
}

func SetStatus(metadata map[string]any, status LifecycleStatus) {
	EnsureMetadata(metadata)[MetadataKeyStatus] = string(normalizeStatus(status))
}

func MetadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}

	switch value := metadata[key].(type) {
	case string:
		return strings.TrimSpace(value)
	case fmt.Stringer:
		return strings.TrimSpace(value.String())
	case []byte:
		return strings.TrimSpace(string(value))
	default:
		return ""
	}
}

func MetadataBool(metadata map[string]any, key string) bool {
	if metadata == nil {
		return false
	}

	switch value := metadata[key].(type) {
	case bool:
		return value
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(value))
		return err == nil && parsed
	default:
		return false
	}
}

func MetadataStringSlice(metadata map[string]any, key string) []string {
	if metadata == nil {
		return nil
	}

	switch value := metadata[key].(type) {
	case []string:
		out := make([]string, 0, len(value))
		for _, item := range value {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(value))
		for _, item := range value {
			if str, ok := item.(string); ok {
				str = strings.TrimSpace(str)
				if str != "" {
					out = append(out, str)
				}
			}
		}
		return out
	case string:
		value = strings.TrimSpace(value)
		if value == "" {
			return nil
		}
		parts := strings.Split(value, ",")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
		return out
	default:
		return nil
	}
}

func MetadataFloat(metadata map[string]any, key string) (float64, bool) {
	if metadata == nil {
		return 0, false
	}

	switch value := metadata[key].(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case int:
		return float64(value), true
	case int32:
		return float64(value), true
	case int64:
		return float64(value), true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func MetadataTime(metadata map[string]any, key string) (time.Time, bool) {
	if metadata == nil {
		return time.Time{}, false
	}

	switch value := metadata[key].(type) {
	case time.Time:
		if value.IsZero() {
			return time.Time{}, false
		}
		return value.UTC(), true
	case string:
		if strings.TrimSpace(value) == "" {
			return time.Time{}, false
		}
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return time.Time{}, false
		}
		return parsed.UTC(), true
	default:
		return time.Time{}, false
	}
}

func SetMetadataTime(metadata map[string]any, key string, at time.Time) {
	if at.IsZero() {
		delete(metadata, key)
		return
	}
	EnsureMetadata(metadata)[key] = at.UTC().Format(time.RFC3339)
}

func NormalizeSearchFilters(metaFilters map[string]any, now time.Time) map[string]any {
	filters := CloneMetadata(metaFilters)
	if !MetadataBool(filters, FilterKeyIncludeStale) {
		if MetadataString(filters, FilterKeyStatus) == "" {
			filters[FilterKeyStatus] = string(StatusActive)
		}
		if MetadataString(filters, FilterKeyAsOf) == "" {
			filters[FilterKeyAsOf] = now.UTC().Format(time.RFC3339)
		}
	}
	delete(filters, FilterKeyIncludeStale)
	return filters
}

func LastVerifiedAtForKnowledge(k Knowledge) (time.Time, bool) {
	if k.LastVerifiedAt != nil && !k.LastVerifiedAt.IsZero() {
		return k.LastVerifiedAt.UTC(), true
	}
	return MetadataTime(k.Metadata, MetadataKeyLastVerifiedAt)
}

func ValidFromForKnowledge(k Knowledge) (time.Time, bool) {
	if k.ValidFrom != nil && !k.ValidFrom.IsZero() {
		return k.ValidFrom.UTC(), true
	}
	return MetadataTime(k.Metadata, MetadataKeyValidFrom)
}

func ValidToForKnowledge(k Knowledge) (time.Time, bool) {
	if k.ValidTo != nil && !k.ValidTo.IsZero() {
		return k.ValidTo.UTC(), true
	}
	return MetadataTime(k.Metadata, MetadataKeyValidTo)
}

func ConfidenceForKnowledge(k Knowledge) (float64, bool) {
	if k.Confidence != nil {
		return *k.Confidence, true
	}
	return MetadataFloat(k.Metadata, MetadataKeyConfidence)
}

func HydrateKnowledgeLifecycle(k *Knowledge) {
	if k == nil {
		return
	}

	k.Metadata = EnsureMetadata(k.Metadata)
	k.Status = StatusForKnowledge(*k)
	k.CanonicalKey = firstNonEmpty(k.CanonicalKey, MetadataString(k.Metadata, MetadataKeyCanonicalKey))
	if k.CanonicalKey == "" {
		k.CanonicalKey = NormalizeCanonicalKey(k.Type, firstNonEmpty(k.NormalizedTitle, k.Title))
	}

	k.SupersedesID = firstNonEmpty(k.SupersedesID, MetadataString(k.Metadata, MetadataKeySupersedesID))
	k.SupersededByID = firstNonEmpty(k.SupersededByID, MetadataString(k.Metadata, MetadataKeySupersededByID))
	if k.ValidFrom == nil {
		if at, ok := MetadataTime(k.Metadata, MetadataKeyValidFrom); ok {
			k.ValidFrom = timePtr(at)
		}
	}
	if k.ValidTo == nil {
		if at, ok := MetadataTime(k.Metadata, MetadataKeyValidTo); ok {
			k.ValidTo = timePtr(at)
		}
	}
	if k.LastVerifiedAt == nil {
		if at, ok := MetadataTime(k.Metadata, MetadataKeyLastVerifiedAt); ok {
			k.LastVerifiedAt = timePtr(at)
		}
	}
	if k.Confidence == nil {
		if confidence, ok := MetadataFloat(k.Metadata, MetadataKeyConfidence); ok {
			k.Confidence = float64Ptr(confidence)
		}
	}
	k.SourceRef = firstNonEmpty(k.SourceRef, MetadataString(k.Metadata, MetadataKeySourceRef))
	k.VerifiedBy = firstNonEmpty(k.VerifiedBy, MetadataString(k.Metadata, MetadataKeyVerifiedBy))

	SyncKnowledgeLifecycleMetadata(k)
}

func SyncKnowledgeLifecycleMetadata(k *Knowledge) {
	if k == nil {
		return
	}

	metadata := EnsureMetadata(k.Metadata)
	k.Status = StatusForKnowledge(*k)
	if k.Status == "" {
		k.Status = StatusActive
	}
	if strings.TrimSpace(k.CanonicalKey) == "" {
		k.CanonicalKey = CanonicalKeyForKnowledge(*k)
	}

	metadata[MetadataKeyStatus] = string(k.Status)
	metadata[MetadataKeyCanonicalKey] = k.CanonicalKey

	setOptionalMetadataString(metadata, MetadataKeySupersedesID, k.SupersedesID)
	setOptionalMetadataString(metadata, MetadataKeySupersededByID, k.SupersededByID)
	if k.ValidFrom != nil {
		SetMetadataTime(metadata, MetadataKeyValidFrom, k.ValidFrom.UTC())
	} else {
		delete(metadata, MetadataKeyValidFrom)
	}
	if k.ValidTo != nil {
		SetMetadataTime(metadata, MetadataKeyValidTo, k.ValidTo.UTC())
	} else {
		delete(metadata, MetadataKeyValidTo)
	}
	if k.LastVerifiedAt != nil {
		SetMetadataTime(metadata, MetadataKeyLastVerifiedAt, k.LastVerifiedAt.UTC())
	} else {
		delete(metadata, MetadataKeyLastVerifiedAt)
	}
	if k.Confidence != nil {
		metadata[MetadataKeyConfidence] = *k.Confidence
	} else {
		delete(metadata, MetadataKeyConfidence)
	}
	setOptionalMetadataString(metadata, MetadataKeySourceRef, k.SourceRef)
	setOptionalMetadataString(metadata, MetadataKeyVerifiedBy, k.VerifiedBy)

	k.Metadata = metadata
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func normalizeStatus(status LifecycleStatus) LifecycleStatus {
	switch LifecycleStatus(strings.ToLower(strings.TrimSpace(string(status)))) {
	case StatusSuperseded:
		return StatusSuperseded
	case StatusExpired:
		return StatusExpired
	case StatusArchived:
		return StatusArchived
	case StatusDraft:
		return StatusDraft
	case StatusActive, "":
		return StatusActive
	default:
		return LifecycleStatus(strings.ToLower(strings.TrimSpace(string(status))))
	}
}

func setOptionalMetadataString(metadata map[string]any, key string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		delete(metadata, key)
		return
	}
	metadata[key] = value
}

func timePtr(value time.Time) *time.Time {
	v := value.UTC()
	return &v
}

func float64Ptr(value float64) *float64 {
	v := value
	return &v
}
