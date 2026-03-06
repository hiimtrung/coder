---
description: Review feature requirements for completeness.
---

1. **Gate In (MANDATORY)** — Run `coder memory search "<requirement context>"` to retrieve related business rules and previous requirement reviews.
2. **Review Requirements** — Review `docs/ai/requirements/feature-{name}.md` and the project-level template `docs/ai/requirements/README.md` to ensure structure and content alignment. Summarize:

- Core problem statement and affected users
- Goals, non-goals, and success criteria
- Primary user stories & critical flows
- Constraints, assumptions, open questions
- Any missing sections or deviations from the template

Identify gaps or contradictions and suggest clarifications.
3. **Gate Out (MANDATORY)** — Run `coder memory store "Requirement Review: <Name>" "<Summary of Key Patterns and Identified Gaps>"` to store knowledge.
