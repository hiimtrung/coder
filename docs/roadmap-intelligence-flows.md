# Coder Intelligence Flows — Roadmap

> Status: active roadmap
> Last updated: 2026-03-31
> This file describes the target agent workflow around `coder`, not a plan to make `coder-node` run its own LLM chat product.

---

## Direction Change

Earlier roadmap iterations assumed `coder-node` would evolve into an LLM-serving backend with commands such as:

- `coder chat`
- `coder review`
- `coder debug`
- `coder plan`
- `coder qa`
- `coder workflow`

That direction is intentionally dropped.

Reason:

- `coder-node` runs in Docker on infrastructure without GPU
- CPU-only local inference is too slow for interactive agent workflows
- response quality and latency would be worse than using the agent's own model
- this duplicates capabilities the agent already has

`coder-node` should remain a knowledge and state backend:

- memory
- skills
- auth
- activity
- storage

The reasoning engine stays in the external agent.

---

## Core Architecture

### What owns reasoning

The agent owns:

- clarification
- planning
- implementation reasoning
- review reasoning
- deciding when to call subagents

### What `coder` owns

`coder` owns:

- retrieving memory
- retrieving skills
- saving memory
- saving local session context
- reading and updating `.coder/` project state

### What `coder-node` owns

`coder-node` owns:

- remote memory and skill APIs
- storage and retrieval
- auth and activity logging

It should not become a conversational LLM server.

---

## Target Workflow

When the user submits a problem, the agent should follow this loop:

```text
1. Read project context
   - coder skill search
   - coder memory search
   - docs/
   - .coder/

2. Determine current phase and task state
   - .coder/STATE.md
   - .coder/ROADMAP.md
   - .coder/phases/*
   - .coder/session.md

3. Clarify and refine the task
   - update or create the right .coder artifact

4. Execute work
   - do locally when simple
   - spawn subagent(s) when a task is document-defined and parallelizable

5. Persist progress continuously
   - update status
   - update task ownership
   - write summaries/checkpoints back into .coder

6. Close the loop
   - save session
   - store memory
```

The important change is that the workflow is **agent-driven and file-backed**, not **server-LLM-driven**.

---

## Source Of Truth Before Acting

Before any substantial work, the agent must inspect:

### 1. Semantic context

```bash
coder skill search "<topic>"
coder memory search "<topic>"
```

### 2. Project documentation

- `docs/`
- any feature-specific design or requirement docs

### 3. Project execution state

- `.coder/STATE.md`
- `.coder/ROADMAP.md`
- `.coder/session.md`
- `.coder/phases/*`

The agent must infer:

- what phase the project is in
- what task is currently active
- what artifacts already exist
- what is missing
- whether the work is a continuation, a replanning step, or a new task

---

## `.coder/` As The Execution Backbone

The `.coder/` directory should become the primary state and coordination layer.

### Required roles of `.coder/`

- persistent task state
- execution checkpoints
- plan and verification artifacts
- subagent coordination state
- recovery after context loss or `/compact`

### Key files

```text
.coder/STATE.md
.coder/ROADMAP.md
.coder/session.md
.coder/phases/
```

### Recommended additions

```text
.coder/tasks/
  task-001.md
  task-002.md

.coder/runs/
  run-2026-03-31-01.md

.coder/subagents/
  subagent-task-001.md
  subagent-task-002.md
```

These files do not need to be implemented as CLI commands first. The agent can start using the structure before wrappers exist.

---

## Subagent Model

Subagents should be used only when a task is:

- already defined in `.coder` documents
- sufficiently scoped
- independently executable
- worth parallelizing or isolating

### Subagent responsibilities

Each subagent must:

1. read the task document it owns
2. mark the task as `in_progress`
3. record its ownership
4. update checkpoints while working
5. mark the task `done` or `blocked`
6. write a short completion summary back into `.coder`

### Main agent responsibilities

The main agent must:

- decide when a subagent is appropriate
- create or refine the task document before dispatch
- avoid overlapping ownership across subagents
- reconcile outputs back into the parent phase/run state

### Status ownership rule

The agent or subagent doing the work is responsible for updating the status of that work.

No task should complete without its state being written back into `.coder`.

---

## Proposed State Model

### Project-level state

`STATE.md` should keep:

- current phase
- current step
- active run
- blockers
- decisions
- open subagent tasks

### Task-level state

Each task file should keep:

- task id
- phase
- title
- status: `todo | in_progress | blocked | done`
- owner: `main-agent | subagent:<name>`
- inputs
- expected outputs
- checkpoints
- final summary

### Run-level state

Each run file should keep:

- trigger or user request
- initial context used
- task decomposition
- subagents spawned
- final outcome

---

## Roadmap Phases

### Phase 1: Documentation And State Cleanup

Goal:

- remove stale assumptions that `coder-node` will run LLM workflows
- align docs with the current command surface
- make `.coder/` the visible execution backbone

Definition of done:

- roadmap no longer plans `/chat`, `/review`, `/debug`, `/plan`, `/qa`, `/workflow` on `coder-node`
- docs clearly say the agent does reasoning, `coder` does retrieval and state

### Phase 2: Context Discovery Protocol

Goal:

- standardize how the agent reads memory, docs, and `.coder/` before acting

Definition of done:

- every substantial workflow begins by scanning:
  - `coder skill search`
  - `coder memory search`
  - `docs/`
  - `.coder/`
- current phase can be inferred reliably from files

### Phase 3: File-Backed Execution Tracking

Goal:

- persist plan, progress, and status into `.coder` continuously

Definition of done:

- tasks have explicit status files or sections
- work can resume after interruption from `.coder` alone

### Phase 4: Subagent-Oriented Execution

Goal:

- let the main agent delegate documented tasks to subagents

Definition of done:

- subagent work units are defined in `.coder/tasks/` or equivalent phase files
- each subagent updates its own status and summary
- ownership collisions are avoided

### Phase 5: Dynamic Skill Retrieval During Work

Goal:

- let the agent fetch additional skills after clarification or during execution

This phase aligns with [docs/dynamic_skill_retrieval_plan.md](/Users/trungtran/ai-agents/coder/docs/dynamic_skill_retrieval_plan.md).

Definition of done:

- skill retrieval is not locked to the initial user prompt
- active skill context can be updated mid-task
- prompt injection uses structured or raw skill content, not terminal-formatted output

### Phase 6: Recovery And Auditability

Goal:

- make every run reconstructable from `.coder`

Definition of done:

- phase, task, owner, and summary history are visible in files
- interrupted work can restart from state without depending on chat history

---

## Explicitly Dropped From This Roadmap

The following are removed as roadmap targets for `coder-node`:

- built-in chat serving
- built-in review serving
- built-in debug serving
- built-in planning serving
- built-in QA serving
- built-in workflow orchestration via local LLM endpoints
- any design that depends on `coder-node` being a fast, accurate conversational model host

If thin CLI wrappers are added later, they should orchestrate agent behavior and `.coder` state, not host inference on `coder-node`.

---

## Practical Operating Model

The intended operating model is:

1. user gives a task
2. agent scans memory, docs, and `.coder`
3. agent identifies current phase and required artifacts
4. agent writes or updates the plan/state in `.coder`
5. agent executes locally or spawns subagent(s)
6. each worker updates its own task state
7. main agent reconciles and closes the loop with session save and memory store

This keeps `coder` focused, fast, and reliable:

- the agent uses its strongest available model
- `coder-node` stays lightweight
- `.coder` becomes the durable operational memory of the project
