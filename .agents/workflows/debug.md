---
description: Debug an issue with structured root-cause analysis before changing code.
---

Help me debug an issue. Clarify expectations, identify gaps, and agree on a fix plan before changing code.

1. **Gate In (MANDATORY)** — Run `coder skill search "<error message or symptoms>"` to retrieve debugging best practices and known error patterns, then run `coder memory search "<error message or symptoms>"` to retrieve historical fixes and root cause analysis.
2. **Gather Context** — If not already provided, ask for: issue description (what is happening vs what should happen), error messages/logs/screenshots, recent related changes or deployments, and scope of impact.
3. **Clarify Reality vs Expectation** — Restate observed vs expected behavior. Confirm relevant requirements or docs that define the expectation. Define acceptance criteria for the fix.
4. **Reproduce & Isolate** — Determine reproducibility (always, intermittent, environment-specific). Capture reproduction steps. List suspected components or modules.
5. **Analyze Potential Causes** — Brainstorm root causes (data, config, code regressions, external dependencies). Apply relevant skill patterns from Gate In. Gather supporting evidence (logs, metrics, traces). Highlight unknowns needing investigation.
6. **Resolve** — Present resolution options (quick fix, refactor, rollback, etc.) with pros/cons and risks. Ask which option to pursue. Summarize chosen approach, pre-work, success criteria, and validation steps.
7. **Gate Out (MANDATORY)** — Run `coder memory store "Root Cause & Fix: <Issue Name>" "<Root Cause Analysis and Resolution Details>"` to prevent similar bugs in the future.
