---
description: QA Engineer process — test plan creation, acceptance testing against user stories, regression testing, and QA report production.
---

# Workflow: QA Test

This workflow verifies that a completed feature meets all acceptance criteria, passes regression checks, and is ready for release.

## When to Use

- Implementation is complete and all code is committed
- Before a pull request is merged to main
- After a hotfix to verify no regressions
- As part of release readiness verification

## Step 1 — Context Load (MANDATORY)

```bash
coder skill resolve "testing <language or framework>" --trigger review --budget 3
coder memory search "<feature name>"
```

Use `coder memory recall "<feature name>"` when the QA scope is large and you need the active working set focused on the current release.
Use `coder memory active` or `.coder/context-state.json` to inspect the current local context before executing the plan.

Then read:

- `docs/requirements/<feature>.md` — user stories and acceptance criteria
- `docs/design/<feature>.md` — data flows and error handling contracts

## Step 2 — Create Test Plan

**Output path**: `docs/qa/<feature-name>-test-plan.md`

```markdown
# Test Plan: <Feature Name>

**Date**: YYYY-MM-DD
**Feature**: <name>
**Requirements**: [link]
**Tester**: <name>

---

## Scope

### In Scope

- <what this test plan covers>

### Out of Scope

- <what is not tested here and why>

---

## Test Environment

- Environment: local | staging | production-like
- Database state: fresh seed | existing data with migrations
- Dependencies: <list any external services needed>

---

## Test Cases by User Story

### Story 1: <Title>

| Test ID | Scenario         | Input              | Expected Output        | Type       |
| ------- | ---------------- | ------------------ | ---------------------- | ---------- |
| TC-001  | Happy path       | valid payload      | 201, resource created  | Acceptance |
| TC-002  | Empty name field | `name: ""`         | 400, VAL_INVALID_NAME  | Negative   |
| TC-003  | Unauthorized     | no JWT token       | 401, AUTH_UNAUTHORIZED | Security   |
| TC-004  | Wrong company    | JWT from company B | 404 or empty           | Isolation  |

### Story 2: <Title>

... (repeat for each story)

---

## Regression Test Suite

| Test ID | Area              | Scenario            | Expected Result    |
| ------- | ----------------- | ------------------- | ------------------ |
| REG-001 | Auth              | Existing login flow | Unchanged behavior |
| REG-002 | <adjacent module> | Core operation      | Unchanged behavior |

---

## Performance Checks

| Check                       | Metric                  | Threshold             |
| --------------------------- | ----------------------- | --------------------- |
| List endpoint response time | p95 latency             | < 200ms               |
| Create under load           | 100 concurrent requests | < 500ms p99, 0 errors |
```

## Step 3 — Execute Test Plan

Run each test case systematically. For each test:

1. Set up the test state (seed data, auth token, etc.)
2. Execute the action (API call, UI action, CLI command)
3. Compare actual result to expected result
4. Record PASS / FAIL / BLOCKED

Run the full automated test suite:

```bash
yarn test
yarn test:e2e
# or: gradle test integrationTest
# or: go test ./...
```

Also verify with the running application for acceptance tests that require it.

## Step 4 — Produce QA Report

**Output path**: `docs/qa/<feature-name>-qa-report.md`

```markdown
# QA Report: <Feature Name>

**Date**: YYYY-MM-DD
**Tester**: <name>
**Verdict**: PASS | FAIL | CONDITIONAL PASS

---

## Summary

| Category         | Total | Pass  | Fail  | Blocked |
| ---------------- | ----- | ----- | ----- | ------- |
| Acceptance tests | N     | N     | N     | N       |
| Negative tests   | N     | N     | N     | N       |
| Security tests   | N     | N     | N     | N       |
| Regression tests | N     | N     | N     | N       |
| **Total**        | **N** | **N** | **N** | **N**   |

---

## Test Results by Story

### Story 1: <Title> — PASS / FAIL

| Test ID | Scenario     | Result | Notes                      |
| ------- | ------------ | ------ | -------------------------- |
| TC-001  | Happy path   | PASS   |                            |
| TC-002  | Empty name   | PASS   |                            |
| TC-003  | Unauthorized | FAIL   | Returns 500 instead of 401 |

---

## Defects Found

### DEF-001: <Title>

- **Severity**: Critical | Major | Minor
- **Test Case**: TC-003
- **Steps to Reproduce**: <step-by-step>
- **Expected**: <expected behavior>
- **Actual**: <actual behavior>
- **Recommendation**: Block merge | Fix before release | Create follow-up ticket

---

## Regression Status

All regression tests: PASS / N failures found

---

## Verdict Rationale

<2-3 sentences explaining the verdict>

## Conditions for Approval (if CONDITIONAL PASS)

- [ ] DEF-001 fixed and re-verified
```

## Step 5 — Gate Out (MANDATORY)

```bash
coder memory store "QA: <Feature Name>" "Verdict: <PASS/FAIL>. Test count: <N>. Defects: <count and severity>. Common failure modes: <if any>." --tags "qa,testing,<feature>,<domain>"
```

---

## Checklist

- [ ] `coder skill resolve` run
- [ ] `coder memory search` run
- [ ] Requirements doc read — all user stories identified
- [ ] Test plan written with test cases for every story
- [ ] Negative and security test cases included
- [ ] Regression test cases included
- [ ] All test cases executed
- [ ] QA report written with results for each test case
- [ ] Defects documented with severity and reproduction steps
- [ ] Verdict stated clearly
- [ ] `coder memory store` run
