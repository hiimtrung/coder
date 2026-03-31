# Dynamic Skill Retrieval Plan

> Status: proposed
> Last updated: 2026-03-31

This document defines the upgrade path from the current one-shot `coder skill search` pattern to a dynamic skill retrieval loop that can fetch additional skills while an agent is working.

---

## Why This Plan Exists

Today the common flow is:

1. user submits a task
2. agent runs `coder skill search "<topic>"`
3. agent keeps that result for the rest of the task

This breaks down in real work because:

- the true task often becomes clear only after clarification
- the task can shift domains during execution
- the CLI search output is optimized for human terminal reading, not LLM prompt injection
- the agent has no explicit mechanism to add, replace, or drop skills mid-flight

The result is predictable:

- wrong initial skills stay in context too long
- missing skills are discovered too late
- markdown structure is flattened when terminal output is reused as prompt context
- token budget is wasted on stale or weakly relevant skills

---

## Goals

- allow agents to retrieve more skills after clarification, during execution, and after errors
- preserve raw markdown structure for injected skill content
- keep the number of active skills small and relevant
- make skill loading explicit, inspectable, and debuggable
- keep the design compatible with the current `coder` CLI and `coder-node` architecture

## Non-Goals

- turning `coder` into a full chat product in this phase
- auto-installing arbitrary skills from unknown sources during execution
- stuffing the full skill library into the system prompt

---

## Current Baseline

Current command surface:

- `coder skill search`
- `coder skill ingest`
- `coder skill list`
- `coder skill info`
- `coder skill delete`
- `coder skill cache`
- `coder skill index`

Current technical constraints:

- search results are chunk-based and confidence-tiered in `internal/usecase/skill/ingestor.go`
- CLI rendering in `cmd/coder/cmd_skill.go` is human-friendly text, not machine-oriented context payload
- there is no session-scoped notion of `active skills`
- there is no first-class resolver that decides `keep`, `add`, or `drop`

---

## Target Model

Dynamic skill retrieval should behave like a loop, not a single gate.

```text
task submitted
  -> initial skill hint search
  -> clarification
  -> refined skill resolve
  -> work starts
  -> domain shift / error / new file area detected
  -> resolve again
  -> add/drop skills within a context budget
```

The important shift is:

- `skill search` finds candidates
- `skill resolve` decides what should be active now
- `skill fetch` returns raw content suitable for prompt injection

---

## Proposed Architecture

### 1. Separate Human Output From LLM Output

Keep terminal-friendly output for humans, but add structured output for agents.

Proposed changes:

- add `coder skill search --format json`
- add `coder skill info --format raw`
- or add a dedicated `coder skill fetch <name>` command that returns raw markdown chunks

Requirements:

- preserve original markdown formatting
- return chunk metadata such as `skill`, `section_id`, `chunk_type`, `score`
- avoid using terminal-rendered stdout as LLM context

### 2. Introduce A Skill Resolver

Add a new server-side use case that works above plain search.

Proposed API:

- `ResolveSkills(task, current_skills, phase, budget, trigger)`

Where:

- `task`: current clarified task or subtask
- `current_skills`: currently loaded skills
- `phase`: `initial`, `clarified`, `execution`, `error-recovery`, `review`
- `budget`: max active skills / chunk budget
- `trigger`: why the resolver was called

Expected output:

- `keep`: skills that should stay active
- `add`: skills to fetch now
- `drop`: skills to remove from active context
- `reason`: concise explanation for each change
- `context_blocks`: raw markdown chunks selected for injection

### 3. Track Active Skills Per Session

Introduce session-scoped skill state.

Possible storage options:

- lightweight local state in `.coder/session.md`
- separate `.coder/active-skills.json`
- server-side session state later, if chat/workflow features return

Suggested first step:

- local state file, because current session handling is local already

Stored fields:

- current task summary
- loaded skill names
- last resolve query
- trigger history
- dropped skills and why

### 4. Add Re-Resolve Triggers

The agent should not resolve skills on every token. It should re-resolve on explicit events.

Recommended triggers:

- after clarification changes the task meaning
- before switching from analysis to implementation
- when entering a new file area or language
- after repeated tool errors
- when a new library, framework, or protocol appears
- when current search confidence is low
- before review or release steps

### 5. Add Context Budgeting

The resolver must keep context small.

Rules:

- default to 1 to 3 active skills
- allow more only for clearly multi-domain tasks
- deduplicate overlapping chunks
- prefer high-confidence chunks first
- downgrade or drop low-utility skills after the task narrows

### 6. Add Skill Coverage Feedback

The agent should be able to say:

- current skills are sufficient
- current skills are weakly matched
- a new domain has appeared and another skill is needed
- no local skill is a good fit

This should be machine-readable, not only implicit in model prose.

---

## Proposed CLI And API Changes

### CLI

Phase 1:

- `coder skill search --format json`
- `coder skill info <name> --format raw`

Phase 2:

- `coder skill resolve "<task>" --current a,b,c --trigger clarified --budget 3`

Optional later:

- `coder skill active`
- `coder skill drop <name>`

### Domain / Use Case

Add new types under `internal/domain/skill`:

- `ResolveRequest`
- `ResolveDecision`
- `ResolvedSkillContext`

Add a new use case under `internal/usecase/skill`:

- `Resolver`

### Transport

Add matching endpoints over gRPC and HTTP so the CLI remains a thin client.

---

## Suggested Rollout

### Phase 0: Documentation cleanup

Done first.

Definition:

- README and core docs reflect current commands only
- roadmap docs are clearly labeled as future design
- agent skill docs stop claiming removed commands are live

### Phase 1: Structured skill output

Deliverables:

- `coder skill search --format json`
- raw markdown skill content path for LLM injection
- tests that verify markdown is preserved

Success criteria:

- no prompt injection path depends on terminal-formatted search output

### Phase 2: Dynamic re-resolve in the agent loop

Deliverables:

- a client-visible `skill resolve` flow
- explicit re-resolve triggers
- session-scoped active skill tracking

Success criteria:

- an agent can add a new skill after clarification without restarting the task
- an agent can drop stale skills when the task narrows

### Phase 3: Budgeting and observability

Deliverables:

- active skill budget policy
- retrieval reason codes
- activity or telemetry fields for resolve events
- evaluation fixtures for false-positive and false-negative skill loading

Success criteria:

- active skill count remains small
- resolve decisions can be audited after a run

### Phase 4: Advanced routing

Optional later work:

- fallback to a curated external catalog when local skills are insufficient
- skill-combination memory for recurring task patterns
- resolver heuristics based on file paths, stack, or recent tool failures

---

## File Impact

Expected code areas:

- `cmd/coder/cmd_skill.go`
- `internal/domain/skill/*`
- `internal/usecase/skill/*`
- `internal/transport/grpc/server/skill.go`
- `internal/transport/grpc/client/skill.go`
- `internal/transport/http/server/skill.go`
- `internal/transport/http/client/skill.go`
- `cmd/coder/cmd_session.go` or a new local state helper for active skills

Expected docs impact:

- `README.md`
- `docs/cli.md`
- `docs/architecture.md`
- `docs/skill_system.md`

---

## Definition Of Done

This upgrade is complete when:

- agents can fetch additional skills mid-task without restarting the workflow
- skill context is injected from raw markdown or structured payloads, not terminal text
- active skills can be inspected and explained
- stale skills can be dropped from context
- docs describe the dynamic loop clearly
- retrieval quality is covered by tests and example scenarios
