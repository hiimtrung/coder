---
description: Add tests for a new feature.
---

Review `docs/ai/testing/feature-{name}.md` and ensure it mirrors the base template before writing tests.

1. **Gate In (MANDATORY)** — Run `coder skill search "<testing context or target module>"` to retrieve testing best practices, mocking strategies, and coverage standards, then run `coder memory search "<testing context or target module>"` to retrieve past test patterns and common failure modes.
2. **Gather Context** — If not already provided, ask for: feature name/branch, summary of changes (link to design & requirements docs), target environment, existing test suites, and any flaky/slow tests to avoid.
2. **Analyze Testing Template** — Identify required sections from `docs/ai/testing/feature-{name}.md`. Confirm success criteria and edge cases from requirements & design docs. Note available mocks/stubs/fixtures. Apply testing skill patterns from Gate In.
3. **Unit Tests (aim for 100% coverage)** — For each module/function: list behavior scenarios (happy path, edge cases, error handling), generate test cases with assertions using existing utilities/mocks, and highlight missing branches preventing full coverage.
4. **Integration Tests** — Identify critical cross-component flows. Define setup/teardown steps and test cases for interaction boundaries, data contracts, and failure modes.
5. **Coverage Strategy** — Recommend coverage tooling commands. Call out files/functions still needing coverage and suggest additional tests if <100%.
6. **Update Documentation** — Summarize tests added or still missing. Update `docs/ai/testing/feature-{name}.md` with links to test files and results. Flag deferred tests as follow-up tasks.
7. **Gate Out (MANDATORY)** — Run `coder memory store "Testing Details: <Feature Name>" "<Summary of Test Scenarios, Coverage, and Mocking Strategies>"` to capture testing knowledge.
