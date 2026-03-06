---
description: End-to-end professional delivery lifecycle orchestrating BA, Dev, and QA phases.
---

Follow this master workflow for any new feature or project to ensure enterprise-grade quality and Agile compliance.

### 🎭 Skill Transition Overview
As you move through this workflow, your primary expertise should shift to match the goals of each phase:
1. **Phase 1 (BA)**: Load `docs-analysis` + `architecture`.
2. **Phase 2 (Dev)**: Load language skills (e.g., `nestjs`) + `development`.
3. **Phase 3 (QA)**: Load `testing`.
4. **Phase 4 (Review)**: Load `docs-analysis` (to finalize docs).

### 🛠 Workflow Orchestration
This master workflow coordinates specialized sub-workflows. You can switch between them as needed or call them individually for specific tasks.

### Phase 1: Planning & Analysis (Business Analysis)
*Objective: Discover context, map user stories, and define architectural boundaries.*
0. **Gate In (MANDATORY)** — Run `coder memory search "<overall project/feature goal>"` at the very start of the lifecycle.
1. **Task Visibility** — Create/update `task.md` in `@brain/`. Mark tasks as `[ ]`, `[/]`, or `[x]`.
2. **Context Discovery** — Run `coder memory search "<topic>"` to find existing patterns. Run `/new-requirement` to scaffold documentation.
3. **Requirement Mapping** — Decompose requirements into actionable stories. Run `/capture-knowledge` for complex components.
4. **Implementation Plan** — Create an `implementation_plan.md`. Success is an approved plan and populated `task.md`.

### Phase 2: Iterative Implementation (Development)
*Objective: Deliver functional increments through TDD and clean architecture.*
1. **Task Execution** — Run `/execute-plan` for each User Story. 
2. **Design Compliance** — Run `/check-implementation` or `/review-design` to ensure alignment with documentation.
3. **Refinement** — Run `/simplify-implementation` for complex logic. Run `/remember` to store new patterns.

### Phase 3: Quality Assurance (Testing)
*Objective: Systematically verify requirements and ensure regression safety.*
1. **Verification** — Run `/qa-testing` for the delivered increment.
2. **Automated Testing** — Run `/writing-test` for missing coverage.
3. **Technical Review** — Run `/code-review` and `/technical-writer-review` to polish documentation and code.
4. **Memory Capture** — Run `coder memory store` to save significant patterns or decisions.

### Phase 4: Lifecycle Closure & Review
*Objective: Collect evidence and finalize the delivery.*
1. **Evidence** — Finalize `walkthrough.md`. Update all docs in `docs/` via `/update-planning`.
2. **Gate Out (MANDATORY)** — Run `coder memory store "Project Lifecycle: <Feature Name>" "<Complete Delivery Summary and Lessons Learned>"` to capture the end-to-end knowledge.
3. **Sign-off** — Present results to the user and mark the sprint as closed.