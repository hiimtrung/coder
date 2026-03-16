---
description: Compare implementation with design and requirements docs to ensure alignment.
---

Compare the current implementation with the design in docs/ai/design/ and requirements in docs/ai/requirements/.

1. **Gate In (MANDATORY)** — Run `coder skill search "<feature or implementation context>"` to retrieve architecture and design review patterns, then run `coder memory search "<feature or implementation context>"` to retrieve related design patterns and previous implementation checks.
2. **Gather Context** — If not already provided, ask for: feature/branch description, list of modified files, relevant design doc(s), and any known constraints or assumptions.
2. For each design doc: summarize key architectural decisions and constraints, highlight components, interfaces, and data flows that must be respected.
3. File-by-file comparison: confirm implementation matches design intent, apply skill-specific architecture rules from Gate In, note deviations or missing pieces, flag logic gaps, edge cases, or security issues, suggest simplifications or refactors, and identify missing tests or documentation updates.
4. Summarize findings with recommended next steps.
5. **Gate Out (MANDATORY)** — Run `coder memory store "Implementation Check: <Feature Name>" "<Summary of Deviations and Fixes>"` to ensure implementation memory is updated.
