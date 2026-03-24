---
description: Update planning docs to reflect implementation progress — works with both PLAN.md (quick path) and STATE.md (lifecycle path).
---

Help me reconcile current implementation progress with the planning documentation.

1. **Gate In (MANDATORY)** — Run `coder skill search "<feature planning context>"` to retrieve project management and planning best practices, then run `coder memory search "<feature planning context>"` to retrieve previous milestones and delayed tasks.

2. **Check Current State** — If using the lifecycle path (`coder new-project` was run):
   - Run `coder progress` to see current phase, step, decisions, blockers, and PR history.
   - Run `coder next` to confirm the next recommended command.
   - Run `coder health` to surface any missing artifacts or stale state.
   - Skip to step 4 if STATE.md is the source of truth.

3. **Gather Context** — If using quick-path PLAN.md (not lifecycle):
   - Ask for: feature/branch name and brief status, tasks completed since last update, new tasks discovered, current blockers or risks, planning doc path (default `docs/ai/planning/feature-{name}.md`).

4. **Review & Reconcile** — Summarize existing milestones, task breakdowns, and dependencies from the planning doc or STATE.md. For each planned task: mark status (done / in progress / blocked / not started), note scope changes, record blockers, identify skipped or added tasks.
   - For lifecycle path: use `coder milestone audit N` to check phase N completeness checklist.
   - Record new decisions: `coder note "<decision text>"`.
   - Record new blockers: `coder note --blocker "<blocker text>"`.
   - Add deferred work: `coder todo add "<item>"`.

5. **Produce Updated Task List** — Generate an updated checklist grouped by: Done, In Progress, Blocked, Newly Discovered Work — with short notes per task.

6. **Next Steps & Summary** — Suggest the next 2–3 actionable tasks and highlight risky areas. Prepare a summary covering: current state, major risks/blockers, upcoming focus, and scope/timeline changes.
   - If lifecycle path: confirm with `coder next` that the suggested action matches STATE.md.

7. **Gate Out (MANDATORY)** — Run `coder memory store "Planning Update: <Feature Name>" "<Summary of Progress, Risks, and Scope Changes>"` to update semantic memory.
