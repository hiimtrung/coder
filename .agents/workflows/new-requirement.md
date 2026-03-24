---
description: Scaffold feature documentation from requirements through planning — supports quick-path docs and lifecycle-path project init.
---

Guide me through adding a new feature or project, from requirements documentation to implementation readiness.

1. **Gate In (MANDATORY)** — Run `coder skill search "<feature name>"` to retrieve relevant architectural and design patterns, then run `coder memory search "<feature name>"` to identify existing related requirements or project-specific context.

2. **Identify Scope**
   - **New standalone feature** (days): follow the Quick Path (steps 3–8).
   - **New project or large multi-phase feature** (weeks): run `coder new-project "<idea>"` to generate REQUIREMENTS.md, ROADMAP.md, STATE.md via AI-guided Q&A, then switch to the Lifecycle Path in `/full-lifecycle-delivery`.

3. **Capture Requirement** — If not already provided, ask for:
   - Feature name (kebab-case, e.g., `user-authentication`)
   - What problem it solves and who will use it
   - Key user stories (3–5 minimum)

4. **Create Feature Documentation Structure** — Copy each template into feature-specific files:
   - `docs/ai/requirements/README.md` → `docs/ai/requirements/feature-{name}.md`
   - `docs/ai/design/README.md` → `docs/ai/design/feature-{name}.md`
   - `docs/ai/planning/README.md` → `docs/ai/planning/feature-{name}.md`
   - `docs/ai/implementation/README.md` → `docs/ai/implementation/feature-{name}.md`
   - `docs/ai/testing/README.md` → `docs/ai/testing/feature-{name}.md`

5. **Requirements Phase** — Fill out `docs/ai/requirements/feature-{name}.md`:
   - Problem statement, goals/non-goals, user stories, success criteria, constraints, open questions.
   - Apply any relevant best practices from Gate In skill results.

6. **Design Phase** — Fill out `docs/ai/design/feature-{name}.md`:
   - Architecture changes, data models, API/interfaces, components, design decisions, security and performance considerations.

7. **Planning Phase** — Fill out `docs/ai/planning/feature-{name}.md`:
   - Task breakdown with subtasks, dependencies, effort estimates, implementation order, risks.
   - Run `coder plan "<feature>" --auto` to generate a PLAN.md with AI-estimated tasks, then cross-reference with the manual planning doc.

8. **Documentation Review** — Run `/review-requirements` and `/review-design` to validate the drafted docs.

9. **Gate Out (MANDATORY)** — Run `coder memory store "Feature Design: <Name>" "<Key Architectural Decisions and Requirements Summary>"` to capture the state of documentation into memory.

10. **Next Steps**:
    - Small feature ready to implement: use `/execute-plan`.
    - Large feature or project: use `coder discuss-phase 1` → `coder plan-phase 1` → `coder execute-phase 1`.
    - Generate a PR description covering: summary, requirements doc link, key changes, test status, readiness checklist.
