---
description: Technical Writer process — API documentation, runbooks, CHANGELOG entries, and README updates for completed features.
---

# Workflow: Write Documentation

This workflow produces the documentation artifacts required before a feature is released: API docs, operational runbook, CHANGELOG entry, and README updates.

## When to Use

- A feature implementation is complete and QA has approved it
- Existing documentation is stale relative to recent changes
- A new API endpoint or CLI command needs documentation
- Preparing a release and documentation must be current

## Step 1 — Context Load (MANDATORY)

```bash
coder skill search "technical writing documentation"
coder memory search "<feature or component name>"
```

Then read:
- `docs/requirements/<feature>.md` — user stories for documentation framing
- `docs/design/<feature>.md` — technical details, API contracts, data flows
- `CHANGELOG.md` — existing format and conventions

## Step 2 — API Documentation

For each new or changed endpoint, write or update the API reference.

**Output path**: `docs/api/<resource>.md` (or update existing Swagger/OpenAPI annotations in source)

```markdown
# API Reference: <Resource Name>

## POST /api/v1/<resource>

Creates a new <resource>.

**Authentication**: Required. Bearer JWT token. `company_id` extracted from token.

**Request Body**:

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| `name` | string | Yes | max 255 chars | Display name of the resource |

**Request Example**:
```json
{
  "name": "Example Resource"
}
```

**Response** `201 Created`:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Example Resource",
  "createdAt": "2026-01-15T10:30:00Z"
}
```

**Error Responses**:

| Code | HTTP Status | Trigger |
|------|-------------|---------|
| `VAL_INVALID_NAME` | 400 | Name is empty or too long |
| `AUTH_UNAUTHORIZED` | 401 | Token missing or expired |
| `AUTH_FORBIDDEN` | 403 | Token valid but insufficient permissions |
| `INF_DATABASE_ERROR` | 500 | Persistence failure |

---

## GET /api/v1/<resource>/:id

Returns a single <resource> by ID.

... (follow same format)
```

## Step 3 — Runbook

Write an operations guide for teams managing this feature in production.

**Output path**: `docs/runbooks/<feature-name>.md`

```markdown
# Runbook: <Feature Name>

**Last Updated**: YYYY-MM-DD
**On-Call Escalation**: <team or channel>

---

## Overview

Brief description of what this feature does operationally and which systems it interacts with.

---

## Configuration

| Environment Variable | Required | Default | Description |
|----------------------|----------|---------|-------------|
| `FEATURE_ENABLED` | No | `true` | Feature flag |
| `FEATURE_TIMEOUT_MS` | No | `5000` | External call timeout |

---

## Health Checks

```bash
# Verify the feature is operational
curl -H "Authorization: Bearer $TOKEN" https://api.example.com/api/v1/<resource>?limit=1

# Expected: 200 OK with empty or populated array
```

---

## Common Issues and Remediation

### Issue: <symptom>

**Symptoms**: <what you see in logs or monitoring>

**Likely Cause**: <technical explanation>

**Remediation**:
1. Check <log location> for error pattern `<pattern>`
2. Verify <service or config>
3. If persists: <escalation step>

---

## Rollback Procedure

If this feature must be disabled:

1. Set `FEATURE_ENABLED=false` in environment configuration
2. Restart the application: `kubectl rollout restart deployment/<service>`
3. Verify rollback: `curl .../health` returns 200
4. Notify <channel> that the feature has been disabled
5. Revert database migrations if schema changes were included:
   ```bash
   <migration rollback command>
   ```

---

## Monitoring and Alerts

| Metric | Alert Threshold | Dashboard Link |
|--------|-----------------|----------------|
| Error rate | > 1% over 5 min | <link> |
| P99 latency | > 500ms | <link> |
```

## Step 4 — CHANGELOG Entry

Append to `CHANGELOG.md` using Keep a Changelog format:

```markdown
## [Unreleased]

### Added
- <Feature name>: <one sentence describing what users can now do>
  - <sub-bullet for significant detail>

### Changed
- <What changed in existing behavior and why>

### Fixed
- <Bug fix description — link to issue if applicable>

### Deprecated
- <What is now deprecated and what to use instead>
```

Rules:
- Use present tense, active voice: "Add X", not "Added X" or "X was added"
- Write for the user, not the implementer: focus on impact, not implementation detail
- Group by type: Added, Changed, Fixed, Deprecated, Removed, Security

## Step 5 — README Update

If the feature changes setup, configuration, or public interface:

1. Update the Quick Start section if new setup steps are required
2. Update the command reference if new CLI commands are added
3. Update the features table if a new capability exists
4. Update environment variable documentation

## Step 6 — Gate Out (MANDATORY)

```bash
coder memory store "Documentation: <Feature Name>" "Docs written: API reference, runbook, CHANGELOG. Key terminology: <important terms defined>. Runbook location: <path>." --tags "documentation,<feature>,technical-writing"
```

---

## Checklist

- [ ] `coder skill search` run
- [ ] `coder memory search` run
- [ ] API documentation written for each new/changed endpoint
- [ ] Runbook written with health checks and remediation steps
- [ ] CHANGELOG entry added in correct format
- [ ] README updated if setup or interface changed
- [ ] All documents cross-reference each other where relevant
- [ ] `coder memory store` run
