---
description: Pre-push code review against design docs.
---

Perform a local code review **before** pushing changes.

1. **Gate In (MANDATORY)** — Run `coder skill search "<topic or area of review>"` to retrieve coding standards and review checklists, then run `coder memory search "<topic or area of review>"` to retrieve historical review patterns and common issues in this module.
2. **Gather Context** — If not already provided, ask for: feature/branch description, list of modified files, relevant design doc(s) (e.g., `docs/ai/design/feature-{name}.md`), known constraints or risky areas, and which tests have been run. Also review the latest diff via `git status` and `git diff --stat`.
2. **Understand Design Alignment** — For each design doc, summarize architectural intent and critical constraints.
3. **File-by-File Review** — For every modified file: check alignment with design/requirements and flag deviations, apply skill-specific rules from Gate In (e.g., error handling, architecture patterns), spot logic issues/edge cases/redundant code, flag security concerns (input validation, secrets, auth, data handling), check error handling/performance/observability, and identify missing or outdated tests.
4. **Cross-Cutting Concerns** — Verify naming consistency and project conventions. Confirm docs/comments updated where behavior changed. Identify missing tests (unit, integration, E2E). Check for needed configuration/migration updates.
6. **Gate Out (MANDATORY)** — Run `coder memory store "Code Review: <Feature Name>" "<Key Patterns and Lessons Learned>"` to capture review findings for future automation.
