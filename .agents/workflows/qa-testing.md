---
description: Iterative Quality Assurance focused on rapid feedback and acceptance testing for incremental changes.
---

Follow this workflow to verify each delivered increment against its specific acceptance criteria.

1. **Gate In (MANDATORY)** — Run `coder skill search "<feature context>"` to retrieve testing best practices and quality standards, then run `coder memory search "<feature context>"` to retrieve previous QA results and test patterns.
2. **UAT via coder qa** — If a PLAN.md was generated: run `coder qa --plan <path/to/PLAN.md>` to walk through acceptance criteria one by one with auto-diagnosis of failures. Resume with `coder qa --resume` if interrupted. Export report with `coder qa --report`.
3. **Acceptance Testing** — For criteria not in a plan: systematically verify each AC manually. Use factories or mocks to generate the specific data needed for the story's scenarios. Apply testing patterns from Gate In skill results.
4. **Automated Regression Safety** — Run targeted integration tests to verify that the new increment correctly integrates with existing modules without breakages. Perform a focused request-based test (E2E) of the new feature increment.
5. **Feedback & Correction** — Identify any defects introduced. For each issue: run `coder debug "<issue description>"` to get a structured root cause before fixing. Fix immediately within the same iteration. Briefly assess if the increment caused obvious performance regressions (e.g., slow queries).
6. **Evidence & Sync** — Record successful completion of criteria. Update test documentation and README/docs to reflect the final "as-built" implementation.
7. **Final DoD Sign-off** — Confirm the story is "Done" and ready for deployment or the next iteration.
8. **Gate Out (MANDATORY)** — Run `coder memory store "QA Summary: <Feature Name>" "<Summary of Test Results, Coverage, and Found Bugs>"` to update the quality history.
