---
description: Compare current implementation against design and requirements documents to verify alignment and catch deviations before merge.
---

# Workflow: Check Implementation

Verifies that what was built matches what was designed and what was required. Run this before code review or before marking a story as done.

## When to Use

- Implementation is complete and you want to verify correctness before code review
- A story has been in development and a mid-sprint alignment check is needed
- After a refactor to confirm the design intent was preserved

## Step 1 — Context Load (MANDATORY)

```bash
coder skill resolve "architecture <implementation context>" --trigger review --budget 3
coder memory search "<feature or module name>"
```

## Step 2 — Gather Context

Identify the relevant documents and files:

- `docs/requirements/<feature>.md` — user stories and acceptance criteria
- `docs/design/<feature>.md` — architecture, API contract, data model
- Modified source files — the actual implementation

```bash
git diff --name-only main...HEAD
```

## Step 3 — Acceptance Criteria Verification

For each user story in the requirements doc, trace the implementation:

| Story | Acceptance Criterion | Implementation Location | Status |
|-------|---------------------|------------------------|--------|
| Story 1 | Given X, When Y, Then Z | `src/.../feature.spec.ts` test "..." | VERIFIED / MISSING |
| Story 1 | Error case: VAL_INVALID_NAME | `src/.../feature.spec.ts` test "..." | VERIFIED / MISSING |

For each criterion marked MISSING: note the gap and the action required.

## Step 4 — Design Alignment Review

For each component in the design doc, verify the implementation:

### Architecture Layers

- [ ] Domain layer: entities and domain logic in `domain/` — no framework imports
- [ ] Application layer: use cases in `application/` — no DB imports
- [ ] Infrastructure layer: repositories in `infrastructure/` — implements domain interfaces
- [ ] Controller: in `presentation/` — only validates DTOs and calls use case

### API Contract

- [ ] All endpoints from design doc are implemented
- [ ] Request DTOs match the spec (field names, types, validation)
- [ ] Response DTOs match the spec (all fields present, correct types)
- [ ] Error codes match the spec (`VAL_*`, `BIZ_*`, etc.)
- [ ] HTTP status codes correct (201 for create, 200 for update, etc.)

### Data Model

- [ ] Migration file exists and matches the schema in design doc
- [ ] All indexes defined
- [ ] `company_id` column present on all multi-tenant tables

### Event Publishing

- [ ] Events from design doc are implemented
- [ ] Events published AFTER transaction commits, not before
- [ ] Event payload matches the contract

## Step 5 — Code Quality Check

For each modified file:

- [ ] No `any` types (TypeScript) / no raw `Object` (Java)
- [ ] Error paths handled — no silent swallowing of exceptions
- [ ] Company ID included in all repository queries
- [ ] No hardcoded values that belong in configuration
- [ ] No dead code left from refactoring

## Step 6 — Test Coverage

- [ ] Unit tests exist for domain logic
- [ ] Integration tests exist for repository implementations
- [ ] E2E / request-level tests exist for controller endpoints
- [ ] All error paths have tests, not just happy path
- [ ] Test names clearly describe the scenario

## Step 7 — Summarize Findings

```markdown
## Implementation Check: <Feature Name>

**Date**: YYYY-MM-DD
**Status**: ALIGNED | DEVIATIONS FOUND

### Acceptance Criteria Coverage
<N of N criteria verified. Missing: <list>

### Design Deviations
1. <deviation>: <what was designed vs what was built> — Impact: <low/medium/high>

### Missing Tests
1. <scenario not covered>

### Recommended Actions
1. <action with priority>

### Summary
<2-3 sentences on overall alignment quality>
```

## Step 8 — Gate Out (MANDATORY)

```bash
coder memory store "Implementation Check: <Feature Name>" "Status: <ALIGNED/DEVIATIONS>. AC coverage: <N/N>. Deviations: <count and summary>. Tests missing: <list>." --tags "implementation-check,<feature>,<module>"
```

---

## Checklist

- [ ] `coder skill resolve` run
- [ ] `coder memory search` run
- [ ] Requirements doc read — all acceptance criteria listed
- [ ] Design doc read — all components and contracts listed
- [ ] Implementation traced for each acceptance criterion
- [ ] Architecture layer compliance verified
- [ ] API contract alignment verified
- [ ] Test coverage assessed
- [ ] Summary written with verdict
- [ ] `coder memory store` run
