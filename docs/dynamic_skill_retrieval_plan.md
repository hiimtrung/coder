# Dynamic Context Retrieval Plan

> Status: active plan
> Last updated: 2026-04-01

This document replaces the earlier skill-only plan with a unified context retrieval plan for both skills and memory. The goal is to make retrieval re-entrant during work: if the LLM loses context, narrows scope, hits conflicts, or enters a new domain, it must be able to call `coder skill ...` and `coder memory ...` again without restarting the workflow.

---

## Why This Plan Exists

The old model assumed a mostly one-shot context load at task start:

1. user submits a task
2. agent runs an initial retrieval
3. agent keeps that context for too long

That is no longer sufficient. Real work shifts while the agent is operating:

- clarification changes the actual task
- the task moves into a different file area, language, or protocol
- prior decisions and incidents from memory become more important than generic skills
- the model notices missing context only after errors or contradictory search results

The failure mode is the same across both systems:

- stale skill context remains active too long
- relevant memories are never re-queried when the task changes
- memory retrieval is available, but not yet modeled as an anytime recall loop
- terminal-friendly output is mixed with LLM-facing context instead of using structured or raw payloads
- token budget is consumed by weak or outdated context

---

## Goals

- allow agents to retrieve more skills and more memory after clarification, during execution, and after errors
- preserve raw markdown structure or machine-readable payloads for injected context
- keep active context small, relevant, and explainable
- make context loading explicit, inspectable, and debuggable
- keep the design compatible with the current `coder` CLI and `coder-node` architecture

## Non-Goals

- turning `coder` into a full chat product in this phase
- auto-installing arbitrary skills from unknown sources during execution
- stuffing the full skill library into the system prompt
- turning memory retrieval into hidden background magic the agent cannot inspect or re-run explicitly

---

## Roadmap Gap Snapshot

This section combines the roadmap gap review with the retrieval upgrade plan so the project has one source of truth for retrieval work.

| Area                           | Current state       | Gap                                                                                                                                         |
| ------------------------------ | ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------- |
| Skill structured output        | Implemented         | `coder skill search --format json` and `coder skill info --format raw                                                                       | json` exist.                             |
| Skill dynamic re-resolve       | Implemented locally | `coder skill resolve` and `coder skill active` exist, but the resolver is still CLI-side and not yet a server-level orchestration contract. |
| Skill active state             | Implemented locally | `.coder/active-skills.json` exists, but no broader `.coder` task/run coordination depends on it yet.                                        |
| Memory lifecycle retrieval     | Implemented         | Active-only filtering, history, verify, supersede, and audit exist.                                                                         |
| Memory anytime recall loop     | Implemented         | `coder memory recall` now runs through shared usecase plus HTTP/gRPC contracts and still refreshes `.coder/active-memory.json` locally.     |
| Memory machine-readable output | Implemented         | `coder memory search --format json                                                                                                          | raw` now supports prompt-safe injection. |
| Unified context state          | Implemented locally | Skill and memory refreshes now also update `.coder/context-state.json` as a combined active context snapshot.                               |
| File-backed recovery           | Partial             | `.coder/session.md`, `STATE.md`, and `ROADMAP.md` exist, but task/run/subagent state is still largely roadmap-only.                         |

---

## Current Baseline

Current command surface:

- `coder skill search`
- `coder skill resolve`
- `coder skill active`
- `coder skill ingest`
- `coder skill list`
- `coder skill info`
- `coder skill delete`
- `coder skill cache`
- `coder skill index`
- `coder memory search`
- `coder memory active`
- `coder memory recall`
- `coder memory store`
- `coder memory verify`
- `coder memory supersede`
- `coder memory audit`

Current technical constraints:

- skills already support a local resolver and session-scoped state in `.coder/active-skills.json`
- memory retrieval is lifecycle-aware and now supports `text`, `json`, and `raw` output modes
- successful memory searches now refresh `.coder/active-memory.json`
- memory recall now uses a shared usecase plus HTTP/gRPC transport contract, while the CLI remains responsible for local active-state persistence
- active skills and active memory now also roll up into `.coder/context-state.json` for local recovery and inspection

---

## Target Model

Dynamic context retrieval should behave like a loop, not a single gate.

```text
task submitted
  -> initial skill resolve
  -> initial memory recall
  -> clarification
  -> refined skill resolve
  -> refined memory recall
  -> work starts
  -> domain shift / error / new file area / contradiction detected
  -> resolve/recall again
  -> add/drop skills and refresh memory within a context budget
```

The important shift is:

- `skill search` finds candidates
- `skill resolve` decides what should be active now
- `memory search` recalls relevant project truths and incidents now
- both systems must support explicit re-entry whenever context is insufficient

---

## Proposed Architecture

### 1. Separate Human Output From LLM Output

Keep terminal-friendly output for humans, but add structured output for agents.

Proposed changes:

- keep `coder skill search --format json`
- keep `coder skill info --format raw|json`
- add `coder memory search --format json`
- add `coder memory search --format raw`
- keep `coder memory recall` as the explicit task-time refresh command, but route its decisions through shared contracts instead of CLI-only logic

Requirements:

- preserve original markdown formatting
- return chunk/result metadata such as `skill`, `section_id`, `chunk_type`, `score`, `memory_id`, `canonical_key`, `status`, `last_verified_at`, `conflict_detected`
- avoid using terminal-rendered stdout as LLM context for either system

### 2. Treat Skills And Memory As Separate Retrieval Roles

The agent should not use skills and memory interchangeably.

- skills answer: what general pattern or practice applies here?
- memory answers: what did this project decide, fix, or learn before?

Retrieval order during work:

1. resolve skills for the current domain
2. recall memory for the current repository/task state
3. execute
4. re-run either or both when confidence drops

### 3. Keep Skill Resolve, Add Memory Recall Protocol

Skill resolve already exists as a user-facing CLI behavior. Memory recall is now standardized as an equally explicit protocol across usecase, HTTP, gRPC, and CLI layers.

Proposed API:

- `ResolveSkills(task, current_skills, phase, budget, trigger)`
- `RecallMemory(task, current_memory, phase, budget, trigger)`

Where:

- `task`: current clarified task or subtask
- `current_skills`: currently loaded skills
- `current_memory`: currently active memory keys or IDs
- `phase`: `initial`, `clarified`, `execution`, `error-recovery`, `review`
- `budget`: max active skills / chunk budget
- `trigger`: why the resolver was called

Expected output:

- `keep`: skills that should stay active
- `add`: skills to fetch now
- `drop`: skills to remove from active context
- `reason`: concise explanation for each change
- `context_blocks`: raw markdown or structured blocks selected for injection

Expected memory recall output:

- `keep`: memory records or keys that are still relevant
- `add`: newly recalled memory records or summaries
- `drop`: stale or irrelevant memory records currently in active context
- `conflicts`: conflicting active memories that need a warning or summary
- `coverage`: whether current recall looks sufficient, weak, or missing

### 4. Track Active Context Per Session

Skill state should remain local-first, and memory recall should gain an equivalent session record.

Recommended local files:

- `.coder/active-skills.json`
- `.coder/active-memory.json`
- `.coder/context-state.json`

Suggested `active-memory` fields:

- current task summary
- active memory IDs
- active canonical keys
- last recall query
- trigger history
- conflict summary
- dropped memories and why

### 5. Add Re-Resolve And Re-Recall Triggers

The agent should be allowed to call `coder memory` at any time when context is weak. This must be explicit in the operating model, not only implied.

Recommended triggers for both skills and memory:

- after clarification changes the task meaning
- before switching from analysis to implementation
- when entering a new file area or language
- after repeated tool errors
- when a new library, framework, protocol, or subsystem appears
- when the current search confidence is low
- when top results conflict or feel underspecified
- before review or release steps

Memory-specific triggers:

- the agent needs project-specific truth, not generic best practice
- the agent sees multiple plausible implementations and needs prior decisions
- the task references a bug, migration, incident, rollout, or historical constraint
- the LLM notices it is answering from generic knowledge instead of repository memory

### 6. Add Context Budgeting

The retrieval layer must keep context small.

Rules:

- default to 1 to 3 active skills
- default to 3 to 5 active memory items or summaries
- allow more only for clearly multi-domain tasks
- deduplicate overlapping results
- prefer high-confidence and high-verification items first
- downgrade or drop low-utility context after the task narrows

### 7. Add Coverage Feedback

The agent should be able to say, in machine-readable form:

- current skills are sufficient
- current skills are weakly matched
- current memory recall is sufficient
- current memory recall is weak or conflicting
- a new domain has appeared and another skill is needed
- no local memory is a good fit

---

## Proposed CLI And API Changes

### CLI

Phase 1:

- keep `coder skill search --format json`
- keep `coder skill info <name> --format raw|json`
- add `coder memory search --format json`
- add `coder memory search --format raw`

Phase 2:

- `coder skill resolve "<task>" --current a,b,c --trigger clarified --budget 3`
- keep memory recall guidance aligned with the shared contract shape:
  - `coder memory search "<task>" --limit 5`
  - `coder memory search "<task>" --as-of <time>`
  - `coder memory search "<task>" --history`

Phase 3:

- add `coder memory active`
- `coder memory recall "<task>" --current a,b,c --trigger execution --budget 5`

Optional later:

- `coder skill drop <name>`
- `coder memory drop <id>`

### Domain / Use Case

Add new types under `internal/domain/skill`:

- `ResolveRequest`
- `ResolveDecision`
- `ResolvedSkillContext`

Add matching memory types under `internal/domain/memory`:

- `RecallRequest`
- `RecallDecision`
- `RecalledMemoryContext`

Keep or promote resolver-style use cases under:

- `internal/usecase/skill`
- `internal/usecase/memory`

### Transport

Add matching endpoints over gRPC and HTTP so the CLI remains a thin client.

---

## Suggested Rollout

### Phase 0: Documentation cleanup

Done first.

Definition:

- README and core docs reflect current commands only
- roadmap docs are clearly labeled as future design
- retrieval docs stop treating context loading as start-only behavior

### Phase 1: Structured skill output

Deliverables:

- validate `coder skill search --format json`
- validate `coder skill info --format raw|json`
- add `coder memory search --format json|raw`
- tests that verify markdown and lifecycle metadata are preserved

Success criteria:

- no prompt injection path depends on terminal-formatted output

### Phase 2: Dynamic re-resolve in the agent loop

Deliverables:

- a client-visible `skill resolve` flow
- explicit re-resolve and re-recall triggers
- session-scoped active skill tracking
- session-scoped active memory tracking

Success criteria:

- an agent can add a new skill after clarification without restarting the task
- an agent can drop stale skills when the task narrows
- an agent can call `coder memory` at any point during execution when context is weak
- an agent can refresh project-specific context without restarting the task

### Phase 3: Budgeting and observability

Deliverables:

- active skill budget policy
- active memory budget policy
- retrieval reason codes
- activity or telemetry fields for resolve and recall events
- evaluation fixtures for false-positive and false-negative skill and memory loading

Success criteria:

- active skill count remains small
- active memory count remains small
- resolve and recall decisions can be audited after a run

### Phase 4: Advanced routing

Optional later work:

- fallback to a curated external catalog when local skills are insufficient
- skill-combination memory for recurring task patterns
- resolver and recall heuristics based on file paths, stack, state files, or recent tool failures

---

## File Impact

Expected code areas:

- `cmd/coder/cmd_skill.go`
- `cmd/coder/cmd_memory.go`
- `internal/domain/skill/*`
- `internal/domain/memory/*`
- `internal/usecase/skill/*`
- `internal/usecase/memory/*`
- `internal/transport/grpc/server/skill.go`
- `internal/transport/grpc/client/skill.go`
- `internal/transport/grpc/server/memory.go`
- `internal/transport/grpc/client/memory.go`
- `internal/transport/http/server/skill.go`
- `internal/transport/http/client/skill.go`
- `internal/transport/http/server/memory.go`
- `internal/transport/http/client/memory.go`
- `cmd/coder/cmd_session.go` or a new local state helper for active context

Expected docs impact:

- `README.md`
- `docs/cli.md`
- `docs/architecture.md`
- `docs/skill_system.md`
- `docs/memory_system.md`
- `docs/memory_lifecycle_plan.md`

---

## Definition Of Done

This upgrade is complete when:

- agents can fetch additional skills mid-task without restarting the workflow
- agents can recall additional memory mid-task whenever context is insufficient
- skill and memory context are injected from raw markdown or structured payloads, not terminal text
- active skills can be inspected and explained
- active memory can be inspected and explained
- stale skills can be dropped from context
- stale or conflicting memory can be flagged, summarized, or dropped from active context
- docs describe the dynamic loop clearly
- retrieval quality is covered by tests and example scenarios

---

## Practical Memory Query Protocol For Agents

With a first-class `memory recall` command now available across the stack, the operating rule should be explicit:

The LLM may call `coder memory search` again at any time.

Recommended usage:

1. Initial recall after skill resolve:

- `coder memory search "<task or module>" --limit 5`

2. When the task narrows:

- `coder memory search "<subtask or file area>" --limit 5`

3. When verifying historical truth:

- `coder memory search "<decision or incident>" --history`

4. When time-sensitive behavior matters:

- `coder memory search "<topic>" --as-of <time>`

5. When the model suspects stale or conflicting context:

- `coder memory audit`
- `coder memory search "<topic>" --history`

This protocol is the minimum upgrade needed to ensure the model is never blocked by missing context simply because retrieval happened too early.
