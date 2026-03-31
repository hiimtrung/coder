---
description: Pre-release checklist — verify that all quality gates, documentation, and operational requirements are met before deploying to production.
---

# Workflow: Release Readiness

A structured gate review that confirms a feature or release is safe to deploy. Nothing goes to production without passing this checklist.

## When to Use

- A feature branch is approved and ready for production deployment
- A hotfix is ready to deploy
- A scheduled release containing multiple features

## Step 1 — Context Load (MANDATORY)

```bash
coder skill resolve "release deployment" --trigger review --budget 3
coder memory search "<feature or release name>"
```

## Step 2 — Code Quality Gates

Run all quality checks and confirm they pass. A single failure blocks the release.

```bash
# Lint — zero warnings on new code
yarn lint
# or: go vet ./... && golangci-lint run
# or: gradle checkstyleMain

# Build — must compile cleanly
yarn build
# or: go build ./...
# or: gradle build

# Unit tests — all must pass
yarn test
# or: go test ./...
# or: gradle test

# Integration tests — all must pass
yarn test:e2e
# or: go test -tags=integration ./...
# or: gradle integrationTest
```

Record results:

| Gate | Status | Notes |
|------|--------|-------|
| Lint | PASS / FAIL | |
| Build | PASS / FAIL | |
| Unit tests | PASS / FAIL | N tests, N failures |
| Integration tests | PASS / FAIL | N tests, N failures |

## Step 3 — Acceptance Criteria Verification

Cross-reference the requirements document. Every acceptance criterion must be satisfied:

```
Requirements doc: docs/requirements/<feature>.md

[ ] Story 1 — AC-1: <criterion> — VERIFIED by test TC-001
[ ] Story 1 — AC-2: <criterion> — VERIFIED by test TC-002
[ ] Story 2 — AC-1: <criterion> — VERIFIED by test TC-003
```

If any criterion is unverified, the release is blocked.

## Step 4 — Documentation Checklist

| Document | Status | Location |
|----------|--------|----------|
| API documentation | Complete / Missing / Stale | `docs/api/` |
| Runbook | Complete / Missing / Stale | `docs/runbooks/` |
| CHANGELOG entry | Added / Missing | `CHANGELOG.md` |
| README updated | Yes / No (if needed) | `README.md` |
| Design doc finalized | Yes / N/A | `docs/design/` |

## Step 5 — Database and Infrastructure Changes

If schema or infrastructure changes are included:

- [ ] Migration script exists at the correct path and follows naming convention
- [ ] Migration has been tested on a staging database
- [ ] Migration is reversible (DOWN migration exists or rollback procedure is documented)
- [ ] Any new indexes are validated against query plans
- [ ] New environment variables are documented and set in all target environments

```bash
# Verify migration runs cleanly
<migration tool> migrate up --dry-run
```

## Step 6 — Security Review

- [ ] No secrets or credentials committed to source control
- [ ] New dependencies reviewed for known CVEs
- [ ] New endpoints protected by appropriate auth guards
- [ ] Input validation applied to all user-supplied data
- [ ] `company_id` isolation verified in new queries
- [ ] Rate limiting applied to new public endpoints (if applicable)

## Step 7 — Rollback Plan

Document the rollback procedure before deploying:

```markdown
## Rollback Plan: <Release Name>

**Deployment**: <service name>, version <X.Y.Z>

### Triggers for rollback
- Error rate exceeds 2% within 10 minutes of deployment
- P99 latency exceeds 1 second
- Any data corruption detected

### Rollback steps

1. Revert deployment:
   ```bash
   kubectl rollout undo deployment/<service>
   # or: docker-compose up -d --scale service=1 (previous image tag)
   ```

2. Verify previous version is running:
   ```bash
   kubectl rollout status deployment/<service>
   curl https://api.example.com/health
   ```

3. If migration was applied, run the DOWN migration:
   ```bash
   <migration tool> migrate down 1
   ```

4. Notify: post in #deployments that rollback was executed and why

### Time to rollback
Estimated: < 5 minutes
```

## Step 8 — Deployment Steps

```markdown
## Deployment Steps: <Release Name>

**Target**: production | staging
**Scheduled**: YYYY-MM-DD HH:MM UTC

### Pre-deployment
- [ ] Maintenance window communicated (if required)
- [ ] Team on standby in #deployments channel
- [ ] Monitoring dashboards open

### Deployment sequence
1. Deploy database migrations (if any):
   ```bash
   <migration command>
   ```
2. Deploy application:
   ```bash
   <deployment command>
   ```
3. Verify health endpoint returns 200:
   ```bash
   curl https://api.example.com/health
   ```
4. Run smoke tests against production:
   ```bash
   <smoke test command>
   ```

### Post-deployment
- [ ] Monitor error rate for 15 minutes
- [ ] Verify key metrics in dashboard
- [ ] Post deployment note in #deployments
```

## Step 9 — Release Sign-off

Record sign-off from each required role:

| Role | Name | Status |
|------|------|--------|
| Developer | | Approved |
| Code Reviewer | | Approved |
| QA Engineer | | Approved |
| Product / Feature Owner | | Approved |

**Release verdict**: APPROVED TO DEPLOY / BLOCKED (reason: <reason>)

## Step 10 — Gate Out (MANDATORY)

```bash
coder memory store "Release: <Name>" "Version: <X.Y.Z>. Gates: all passed. Deployment date: <date>. Rollback plan: documented at <path>. Notable items: <anything non-standard>." --tags "release,deployment,<feature>"
```

---

## Quick Blockers Reference

| Finding | Action |
|---------|--------|
| Any test failure | Fix before deploy — no exceptions |
| Missing acceptance criterion test | Write test, get QA approval |
| No rollback plan | Write rollback plan before proceeding |
| Secret in source code | Remove, rotate secret, then proceed |
| Missing migration | Write migration, test on staging |
| Documentation missing | Write docs, get tech writer review |
