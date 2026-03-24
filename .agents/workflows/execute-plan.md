---
description: Execute a feature plan task by task — supports both PLAN.md (quick path) and XML phase plans (lifecycle path).
---

Help me work through a feature plan one task at a time.

1. **Gate In (MANDATORY)** — Run `coder skill search "<task context>"` to retrieve relevant best practices and implementation patterns, then run `coder memory search "<task context>"` to retrieve project-specific history and previous task context.

2. **Identify Execution Mode**
   - **Quick path** (PLAN.md from `coder plan`): follow steps 3–6 below.
   - **Lifecycle path** (XML plans from `coder plan-phase`): run `coder execute-phase N` instead — it handles wave-based execution and atomic git commits automatically. Use `--gaps-only` to resume, `--interactive` for checkpoints. Skip to step 6 after it completes.

3. **Gather Context** — If not already provided, ask for: feature name (kebab-case), brief description, planning doc path (default `.coder/plans/PLAN-<name>-<date>.md`), and supporting docs (design, requirements).

4. **Load & Present Plan** — Read the planning doc and parse task lists. Present an ordered task queue grouped by section with status: `todo`, `in-progress`, `done`, `blocked`.

5. **Interactive Task Execution** — For each task in order:
   - Display context and full task text.
   - Reference relevant design/requirements docs and apply skill rules from Gate In.
   - Offer to outline sub-steps before starting.
   - After work: run `coder review` to get AI feedback on the diff before committing.
   - If an error occurs: run `coder debug "<error>"` for structured root cause analysis before fixing.
   - Prompt for status update (`done`, `in-progress`, `blocked`, `skipped`) with short notes.
   - If blocked: record the blocker with `coder note --blocker "<reason>"` and move to a "Blocked" list.
   - After each task completion: `git add` changed files and commit with a conventional commit message.

6. **Update Planning Doc** — After each status change, generate a markdown snippet to paste back into the planning doc. After each section, ask if new tasks were discovered.

7. **Session Summary** — Produce a summary: Completed, In Progress (with next steps), Blocked (with blockers), Skipped/Deferred, and New Tasks discovered.
   - If using lifecycle path: run `coder milestone audit N` to verify phase completeness.
   - Run `coder next` to see the recommended next command.

8. **Gate Out (MANDATORY)** — Run `coder memory store "Implementation Detail: <Task Name>" "<Key Patterns and Code Decisions>"` to capture technical execution into semantic memory.
