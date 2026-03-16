---
description: Review feature design for completeness.
---

1. **Gate In (MANDATORY)** — Run `coder skill search "<design context>"` to retrieve architectural best practices and design review checklists, then run `coder memory search "<design context>"` to retrieve historical design decisions and similar architectures.
2. **Review Design** — Review the design documentation in docs/ai/design/feature-{name}.md (and the project-level README if relevant). Apply skill guidelines from Gate In. Summarize:

- Architecture overview (ensure mermaid diagram is present and accurate)
- Key components and their responsibilities
- Technology choices and rationale
- Data models and relationships
- API/interface contracts (inputs, outputs, auth)
- Major design decisions and trade-offs
- Non-functional requirements that must be preserved

Highlight any inconsistencies, missing sections, or diagrams that need updates.
3. **Gate Out (MANDATORY)** — Run `coder memory store "Design Review: <Name>" "<Summary of Key Architecture & Decisions>"` to store knowledge.
