---
description: End-to-end professional delivery lifecycle orchestrating BA, Dev, and QA phases.
---

Follow this master workflow for any new feature or project to ensure enterprise-grade quality.

### 🎭 Skill Transition Overview

As you move through this workflow, your primary expertise should shift to match the goals of each phase:
1. **Phase 1 (BA)**: Load `docs-analysis` + `architecture`.
2. **Phase 2 (Dev)**: Load language skills (e.g., `nestjs`) + `development`.
3. **Phase 3 (QA)**: Load `testing`.
4. **Phase 4 (Review)**: Load `docs-analysis` (to finalize docs).

---

### 🛠 Workflow Orchestration

This master workflow has two paths depending on scope:

- **Small feature (days)** — use the Quick Path (Phases 1–4 below)
- **Full project / large feature (weeks)** — use the Lifecycle Path (`coder new-project` → `coder execute-phase`)

---

## Quick Path — Feature delivery

### Phase 1: Planning & Analysis

*Objective: Discover context, map user stories, and define architectural boundaries.*

0. **Gate In (MANDATORY)** — Run `coder skill search "<feature goal>"` then `coder memory search "<feature goal>"` to load relevant patterns and project history.
1. **Task Visibility** — Create/update `task.md`. Mark tasks as `[ ]`, `[/]`, or `[x]`.
2. **Context Discovery** — Apply patterns from Gate In results. Run `/new-requirement` to scaffold documentation.
3. **Requirement Mapping** — Decompose requirements into actionable stories. Run `/capture-knowledge` for complex components.
4. **Implementation Plan** — Run `coder plan "<feature>" --auto` to generate PLAN.md. Save to `.coder/plans/`. Success: approved plan and populated `task.md`.

### Phase 2: Iterative Implementation

*Objective: Deliver functional increments through TDD and clean architecture.*

1. **Task Execution** — Run `/execute-plan` for each User Story.
2. **Continuous Review** — After each meaningful change run `coder review` before committing.
3. **Debug as you go** — Run `coder debug "<error>"` for structured root cause analysis when hitting errors.
4. **Design Compliance** — Run `/check-implementation` or `/review-design` to ensure alignment.
5. **Refinement** — Run `/simplify-implementation` for complex logic. Run `/remember` to store new patterns.

### Phase 3: Quality Assurance

*Objective: Verify requirements and ensure regression safety.*

1. **UAT Verification** — Run `coder qa --plan <PLAN.md>` to walk through acceptance criteria. Issues auto-diagnosed.
2. **Automated Testing** — Run `/writing-test` for missing coverage.
3. **AI Code Review** — Run `coder review` on the full feature diff or `coder review --pr <number>` for PR review.
4. **Technical Review** — Run `/code-review` and `/technical-writer-review` to polish docs and code.
5. **Memory Capture** — Run `coder memory store` to save significant patterns or decisions.

### Phase 4: Lifecycle Closure

*Objective: Collect evidence and finalize the delivery.*

1. **Evidence** — Finalize `walkthrough.md`. Update all docs via `/update-planning`.
2. **Gate Out (MANDATORY)** — Run `coder memory store "Project Lifecycle: <Feature Name>" "<Complete Delivery Summary>"`.
3. **Sign-off** — Present results and mark the sprint as closed.

---

## Lifecycle Path — Project delivery

Use this path when building a full project or a large multi-phase feature.

### Phase 1: Project Init

0. **Gate In (MANDATORY)** — Run `coder skill search "<project type>"` then `coder memory search "<project type>"`.
1. **Initialize** — Run `coder new-project "<idea>"` for a new project, or `coder map-codebase` for an existing one.
   - `coder new-project` produces: `.coder/PROJECT.md`, `REQUIREMENTS.md`, `ROADMAP.md`, `STATE.md`
   - `coder map-codebase` produces: `.coder/codebase/STACK.md`, `ARCHITECTURE.md`, `CONVENTIONS.md`, `CONCERNS.md`
2. **Check** — Run `coder health` to verify all artifacts are in place.

### Phase 2: Per-Phase Delivery Loop

Repeat for each phase N in the roadmap:

1. **Discuss** — Run `coder discuss-phase N` to identify gray areas and capture decisions → `NN-CONTEXT.md`.
   - Use `--auto` to skip Q&A in auto mode, `--batch` to answer all at once.
2. **Plan** — Run `coder plan-phase N` to research + generate XML plans + verify.
   - Produces: `NN-RESEARCH.md`, `NN-01-PLAN.md` … `NN-VERIFICATION.md`
   - Use `--skip-research` if RESEARCH.md already exists.
   - Use `--gaps` to re-plan only items flagged by the verifier.
3. **Execute** — Run `coder execute-phase N` to execute plans with atomic git commits per task.
   - Use `--gaps-only` to skip plans that already have `SUMMARY.md`.
   - Use `--interactive` to checkpoint between plans.
4. **QA** — Run `coder qa --plan .coder/phases/NN-01-PLAN.md` to walk acceptance criteria.
   - For each failure, run `coder debug "<issue>"` before fixing.
5. **Ship** — Run `coder ship N` to create a PR with AI-generated body via `gh pr create`.
   - Use `--draft` to open as draft.
6. **Close** — Run `coder milestone audit N` to check completeness, then `coder milestone complete N`.
7. **Advance** — Run `coder milestone next` to move to the next phase.

At any time:
- `coder progress` — see full project state (phases, step, blockers, PRs)
- `coder next` — print the next recommended command
- `coder note "decision"` — record decisions or blockers to STATE.md
- `coder todo add "..."` — add backlog items

### Phase 3: Gate Out

**Gate Out (MANDATORY)** — Run:
```bash
coder memory store "Project Delivery: <Name>" "<Summary: phases delivered, key decisions, patterns used>"
```
