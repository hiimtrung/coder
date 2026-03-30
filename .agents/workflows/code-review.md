---
description: Code review process — systematic quality gate before merging any feature branch. Covers security, architecture, tests, performance, and documentation.
---

# Workflow: Code Review

A structured code review that enforces production quality standards. This workflow runs before any branch is merged.

## When to Use

- Any feature branch ready for merge
- A hotfix or patch before deployment
- Periodic quality audit of an existing module

## Step 1 — Context Load (MANDATORY)

```bash
coder skill search "<language or module being reviewed>"
coder memory search "<feature or module name>"
```

## Step 2 — Gather Context

If not already provided, collect:
- Feature description and requirements doc path
- List of modified files (`git diff --name-only main...HEAD`)
- Design doc path (if applicable)
- Test coverage report (if available)
- Any known risky areas or edge cases

```bash
git status
git diff --stat main...HEAD
```

## Step 3 — Systematic Review by Category

Work through each category below. Mark findings as:
- **BLOCKING** — must fix before merge (security, correctness, architecture violation)
- **RECOMMENDED** — should fix, but merge can proceed (readability, performance, style)
- **SUGGESTION** — optional improvement for future iteration

---

### Category 1: Security

- [ ] No secrets, tokens, or credentials in source code
- [ ] All user inputs validated before use (DTO validation decorators / bean validation)
- [ ] No SQL string concatenation — parameterized queries only
- [ ] `company_id` extracted from JWT, never from request body
- [ ] No PII logged in plaintext
- [ ] Auth guards applied to all protected endpoints
- [ ] No path traversal risks in file operations
- [ ] Dependency versions reviewed (no known CVEs in new packages)

---

### Category 2: Architecture Compliance

- [ ] Dependencies point inward only (no domain → infrastructure imports)
- [ ] Domain layer contains zero framework imports (no NestJS, Spring, etc.)
- [ ] Use cases do not import from repositories directly — only through interfaces
- [ ] No direct repository access across module boundaries (cross-module = events only)
- [ ] Events published AFTER transaction commits, not before
- [ ] Error codes follow standard prefixes: `AUTH_*`, `VAL_*`, `BIZ_*`, `INF_*`, `SYS_*`
- [ ] No `any` types in TypeScript; no raw `Object` in Java
- [ ] All new files placed in correct layer directory

---

### Category 3: Test Coverage

- [ ] Unit tests exist for all domain logic (entities, value objects, use cases)
- [ ] Repository implementations have integration tests
- [ ] Controller endpoints have E2E or request-level tests
- [ ] All acceptance criteria from the requirements doc are tested
- [ ] Error paths and edge cases are tested, not just happy path
- [ ] No disabled or skipped tests without explanation
- [ ] Test names clearly describe the scenario (not `test1`, `it should work`)

---

### Category 4: Code Correctness

- [ ] No obvious logic errors or off-by-one conditions
- [ ] Null/undefined/empty cases handled correctly
- [ ] Async operations properly awaited — no floating promises
- [ ] Error propagation correct — errors not silently swallowed
- [ ] Idempotent operations are actually idempotent
- [ ] Race conditions considered for concurrent operations
- [ ] Resource cleanup in finally blocks or `using` statements where needed

---

### Category 5: Performance

- [ ] No N+1 query patterns (use joins or batched queries)
- [ ] Database queries use indexed columns in WHERE clauses
- [ ] Large result sets are paginated — no unbounded SELECTs
- [ ] No blocking synchronous I/O in async context
- [ ] Cache invalidation is correct and not over-broad
- [ ] No memory leaks (event listeners removed, connections closed)

---

### Category 6: Documentation

- [ ] Public methods and interfaces have meaningful comments where behavior is non-obvious
- [ ] API endpoints are reflected in OpenAPI/Swagger annotations
- [ ] `CHANGELOG.md` entry added for user-visible changes
- [ ] Design doc updated if implementation deviated from original design
- [ ] Migration scripts present for schema changes
- [ ] README updated if setup or usage changed

---

## Step 4 — Produce Review Report

Write a structured summary:

```markdown
## Code Review: <Feature Name>

**Branch**: <branch-name>
**Reviewer**: <name>
**Date**: YYYY-MM-DD
**Verdict**: APPROVED | APPROVED WITH COMMENTS | REQUEST CHANGES

---

### Blocking Issues

1. **[BLOCKING] <Category>**: <description>
   - File: `path/to/file.ts`, line N
   - Fix: <specific action required>

### Recommended Changes

1. **[RECOMMENDED] <Category>**: <description>
   - File: `path/to/file.ts`, line N
   - Suggestion: <what to do>

### Suggestions

1. **[SUGGESTION]**: <description>

---

### Summary

<2-3 sentences on overall code quality and the most important finding>
```

## Step 5 — Gate Out (MANDATORY)

```bash
coder memory store "Code Review: <Feature Name>" "Verdict: <APPROVED/CHANGES>. Blocking issues: <count and summary>. Patterns found: <new patterns>. Common issues in this module: <if any>." --tags "code-review,<feature>,<module>"
```

---

## Checklist

- [ ] `coder skill search` run
- [ ] `coder memory search` run
- [ ] All 6 review categories completed
- [ ] Each finding classified as BLOCKING / RECOMMENDED / SUGGESTION
- [ ] Review report written
- [ ] Verdict stated clearly
- [ ] `coder memory store` run with key findings
