# Memory Lifecycle Plan

This document describes the implementation plan for fixing stale or superseded results in `coder memory search` and in every workflow that injects memory into agent context.

It complements the current [Memory System](memory_system.md) and [Architecture](architecture.md) docs by defining lifecycle semantics, retrieval policy, rollout phases, and acceptance criteria.

## Implementation Status

- Phase 0 completed: taxonomy aligned in docs and code for `decision` and `pattern`, with legacy types retained for backward compatibility.
- Phase 1 completed: metadata-first lifecycle behavior shipped for active-only search, canonical keys, and `--replace-active`.
- Phase 2 completed: lifecycle fields are now promoted into first-class PostgreSQL columns with backfill and indexing, while metadata remains mirrored for compatibility.
- Phase 3 completed: lifecycle-aware reranking now collapses by `canonical_key` and returns conflict summaries when multiple active versions disagree.
- Phase 4 partially completed ahead of plan: manual `memory verify` and `memory supersede` commands now update version groups without requiring a new write.
- Phase 6 partially completed ahead of plan: `memory audit` now reports active conflicts, expired active memories, long-unverified active memories, and missing lifecycle columns.

## Problem Statement

The current memory system is strong at semantic retrieval, but weak at temporal correctness:

- Memory records are effectively append-only.
- Search ranks by hybrid semantic/full-text relevance only.
- There is no first-class notion of `active`, `expired`, `superseded`, or `verified`.
- `compact` handles duplicates, but not conflicting or obsolete memories.
- The documented taxonomy and the implemented taxonomy are not aligned.

This leads to a predictable failure mode: an older memory that is still semantically similar can outrank the newer, correct memory and be injected into the agent context.

## Current State

### What exists today

- Hybrid RRF ranking is documented in [architecture.md](architecture.md#hybrid-search).
- Memory is documented as a long-term semantic store in [memory_system.md](memory_system.md).
- The current `Knowledge` model now includes explicit lifecycle fields alongside mirrored metadata in [../internal/domain/memory/entity.go](../internal/domain/memory/entity.go).
- The PostgreSQL table now stores first-class lifecycle columns with backfill and indexing in [../internal/infra/postgres/memory.go](../internal/infra/postgres/memory.go).
- The write path supports lifecycle-aware defaults and replace-active superseding in [../internal/usecase/memory/manager.go](../internal/usecase/memory/manager.go).
- The search path now reranks with lifecycle signals, collapses by `canonical_key`, and emits conflict summaries for materially conflicting active versions in [../internal/usecase/memory/manager.go](../internal/usecase/memory/manager.go).
- The CLI now exposes lifecycle-aware `store`, `search`, `verify`, `supersede`, and `audit` flows in [../cmd/coder/cmd_memory.go](../cmd/coder/cmd_memory.go).

### Gaps that must be closed

1. Taxonomy mismatch:
   - [memory_system.md](memory_system.md) describes `fact`, `rule`, `decision`, `pattern`, `document`.
   - [entity.go](../internal/domain/memory/entity.go) implements `fact`, `rule`, `preference`, `skill`, `event`, `document`.
2. Lifecycle mismatch:
   - The system stores history, but has no way to mark whether a memory is still valid.
3. Workflow mismatch:
   - Higher-level context injection paths still need to guarantee they only consume filtered and conflict-aware summaries.
4. Operational rollout mismatch:
   - Existing historical data still needs ongoing audit and cleanup so old parallel truths do not remain active forever.

## Goals

- Default retrieval must prefer memories that are valid now.
- Superseded or expired memories must not appear in default search results.
- Historical memories must remain queryable when explicitly requested.
- The rollout must be backward-compatible with existing stored data.
- The plan must improve both direct `coder memory search` and all workflow-level context injection.

## Non-Goals

- Replacing the current vector + full-text architecture with a graph memory system.
- Automatically inferring truth without any human or workflow verification.
- Deleting all obsolete memories permanently by default.
- Solving every retrieval-quality issue in the same phase as lifecycle correctness.

## Design Principles

1. Lifecycle before ranking.
   Retrieval must first eliminate obviously invalid memories before tuning score formulas.
2. Default-safe behavior.
   A plain `coder memory search` should return the best active answer, not the most similar old answer.
3. Preserve history without polluting the present.
   Historical memory remains valuable, but it must be opt-in.
4. Metadata-first rollout.
   Start with backward-compatible fields and filters, then optimize with stronger schema/indexing where needed.
5. Conflict visibility over silent ambiguity.
   If two active memories conflict, the system should surface the conflict rather than silently pick one.

## Canonical Memory Taxonomy

Before lifecycle logic is added, memory types must be normalized.

### Proposed canonical types

| Type | Meaning | Expected lifetime | Freshness behavior |
|------|---------|-------------------|--------------------|
| `fact` | Project fact or implementation truth | medium | decays if unverified for too long |
| `rule` | Constraint or standard that should be followed | long | low age penalty, strong verification weight |
| `decision` | Chosen architecture/product decision with rationale | long | low age penalty, supersede-aware |
| `pattern` | Reusable implementation pattern discovered from work | medium | moderate verification weight |
| `event` | Incident, bug, migration, or one-time occurrence | short/medium | strongest time decay |
| `document` | General documentation-like knowledge | medium | moderate time decay |

### Deprecation plan

- `preference` should be folded into `rule` or represented by metadata such as `audience`, `scope`, or `owner`.
- `skill` should not be a memory type in long-term semantic memory; skills already belong in the Skill System.

## Lifecycle Model

Each memory should gain explicit lifecycle semantics.

### Required fields

| Field | Type | Purpose |
|------|------|---------|
| `status` | enum | `active`, `superseded`, `expired`, `archived`, `draft` |
| `canonical_key` | text | Stable identity for multiple versions of the same memory |
| `supersedes_id` | text nullable | Previous memory replaced by this one |
| `superseded_by_id` | text nullable | Forward link for traversal |
| `valid_from` | timestamp nullable | Start of validity window |
| `valid_to` | timestamp nullable | End of validity window |
| `last_verified_at` | timestamp nullable | Most recent verification timestamp |
| `confidence` | float nullable | Confidence in correctness or applicability |
| `source_ref` | text nullable | PR, commit, doc, issue, or external source |
| `verified_by` | text nullable | Actor or workflow that verified the memory |

### Rollout strategy for fields

Phase 1 should store these values in `metadata` for compatibility.

Phase 2 should promote the highest-value fields into first-class columns:

- `status`
- `canonical_key`
- `supersedes_id`
- `superseded_by_id`
- `valid_from`
- `valid_to`
- `last_verified_at`
- `confidence`

This preserves flexibility while making search filters and indexes efficient.

## Retrieval Policy

### Default search behavior

By default, `coder memory search` should:

1. Filter to `status = active`.
2. Exclude records where `valid_to < now()`.
3. Prefer records with newer `last_verified_at`.
4. Collapse multiple versions of the same `canonical_key` to the best active candidate.
5. Surface a conflict marker when multiple active records with the same `canonical_key` disagree materially.

### Explicit historical search behavior

Historical retrieval should be opt-in via flags such as:

- `--include-stale`
- `--status superseded`
- `--as-of 2026-03-01T00:00:00Z`
- `--history`

### Proposed ranking model

The system should keep RRF candidate generation and then apply lifecycle-aware reranking:

`final_score = rrf_score * freshness_multiplier * verification_multiplier * confidence_multiplier`

Guidance:

- `event` gets the strongest freshness decay.
- `fact`, `pattern`, and `document` get moderate freshness decay.
- `rule` and `decision` get minimal age decay, but strong penalty when long-unverified.
- `superseded`, `expired`, and `archived` should not participate in default ranking.

### Context injection policy

All higher-level workflows using memory injection should switch from "raw top N" to "filtered top N":

- `chat`
- `review`
- `debug`
- `plan`
- `qa`
- `workflow`
- lifecycle commands that auto-inject memory

If the top results contain conflicting active memories, the system should inject a short conflict summary instead of multiple contradictory snippets.

## Write Path Semantics

### New store behavior

`coder memory store` should support lifecycle-aware writes.

Recommended new flags:

| Flag | Purpose |
|------|---------|
| `--key` | Set `canonical_key` |
| `--status` | Set lifecycle status explicitly |
| `--supersedes <id>` | Replace a previous memory |
| `--valid-from` | Set validity start |
| `--valid-until` | Set validity end |
| `--verified-at` | Set verification time |
| `--confidence` | Set confidence score |
| `--source` | Attach source reference |
| `--replace-active` | Atomically supersede current active memory with same key |

### Store rules

1. A write with `--replace-active --key <k>` should:
   - find the current active record for `<k>`
   - write the new record
   - mark the previous active record as `superseded`
   - connect both records using `supersedes_id` and `superseded_by_id`
2. A write without lifecycle flags should remain supported and default to:
   - `status = active`
   - no validity window
   - `last_verified_at = created_at` only for trusted workflow-generated memories
3. Conflicting writes for the same key must not silently create multiple active rows unless explicitly allowed.

## Maintenance Workflows

`compact` should remain focused on duplicate or chunk-level consolidation. Lifecycle management needs separate workflows.

### New command: `coder memory verify`

Purpose:

- mark a memory as verified
- refresh `last_verified_at`
- optionally update `confidence`
- attach a `source_ref`

### New command: `coder memory supersede`

Purpose:

- mark one memory as replaced by another
- enforce version chain integrity

### New command: `coder memory audit`

Purpose:

- find active conflicts by `canonical_key`
- find expired-but-active memories
- find long-unverified active memories
- find legacy records missing lifecycle fields
- report superseded memories still leaking into search

## API and Transport Changes

The lifecycle model must exist across all layers, not only in the CLI.

### Affected areas

- [../internal/domain/memory/port.go](../internal/domain/memory/port.go)
- gRPC memory protobufs under `api/proto`
- gRPC server/client under `internal/transport/grpc`
- HTTP server/client under `internal/transport/http`

### Required contract updates

- Add lifecycle fields to store and search request/response payloads.
- Add lifecycle filters to search APIs.
- Return lifecycle metadata in list and search results.
- Preserve compatibility for older clients by making new fields optional.

## Detailed Implementation Phases

### Phase 0: Taxonomy and docs alignment

Deliverables:

- Align docs and code on canonical memory types.
- Document lifecycle vocabulary.
- Document the default retrieval policy.

Files likely affected:

- [memory_system.md](memory_system.md)
- [cli.md](cli.md)
- [architecture.md](architecture.md)
- [../internal/domain/memory/entity.go](../internal/domain/memory/entity.go)

Exit criteria:

- One canonical taxonomy exists in both docs and code.

### Phase 1: Metadata-first lifecycle support

Deliverables:

- Allow lifecycle fields to be written and read via `metadata`.
- Add search filters based on lifecycle metadata.
- Add CLI flags for lifecycle-aware store and search.

Files likely affected:

- [../cmd/coder/cmd_memory.go](../cmd/coder/cmd_memory.go)
- [../internal/usecase/memory/manager.go](../internal/usecase/memory/manager.go)
- [../internal/domain/memory/port.go](../internal/domain/memory/port.go)
- [../internal/transport/grpc/server/memory.go](../internal/transport/grpc/server/memory.go)
- [../internal/transport/http/server/memory.go](../internal/transport/http/server/memory.go)

Exit criteria:

- Users can mark memory as superseded or expired without schema migration dependencies.

### Phase 2: Schema promotion and indexing

Deliverables:

- Add first-class lifecycle columns to `knowledge`.
- Backfill columns from `metadata`.
- Add indexes for active-only and validity-window retrieval.

Files likely affected:

- [../internal/infra/postgres/memory.go](../internal/infra/postgres/memory.go)
- database migration files if introduced later

Recommended indexes:

- `(status)`
- `(canonical_key, status)`
- `(valid_to)`
- `(last_verified_at)`
- partial index on active rows if query volume justifies it

Exit criteria:

- Active-only retrieval does not require JSONB-only scans for the common path.

### Phase 3: Retrieval reranking and conflict handling

Deliverables:

- Apply lifecycle filtering before ranking.
- Apply freshness and verification reranking after candidate generation.
- Collapse results by `canonical_key`.
- Add conflict detection.

Files likely affected:

- [../internal/infra/postgres/memory.go](../internal/infra/postgres/memory.go)
- [../internal/usecase/memory/manager.go](../internal/usecase/memory/manager.go)

Exit criteria:

- Superseded memories no longer appear in default search.
- Results are stable when both old and new versions are semantically similar.

### Phase 4: Write-path versioning

Deliverables:

- Add transactional supersede behavior.
- Prevent multiple active rows for a key in the common path.
- Introduce `memory supersede` and `memory verify`.

Files likely affected:

- [../cmd/coder/cmd_memory.go](../cmd/coder/cmd_memory.go)
- [../internal/usecase/memory/manager.go](../internal/usecase/memory/manager.go)
- [../internal/infra/postgres/memory.go](../internal/infra/postgres/memory.go)

Exit criteria:

- Updating a memory produces a clean version chain instead of ambiguous parallel truth.

### Phase 5: Workflow-level injection safety

Deliverables:

- Update memory injection in all context-aware workflows.
- Inject filtered and conflict-aware memory summaries only.
- Add debug output or explain mode showing why memories were selected.

Files likely affected:

- chat/review/plan/debug workflow entry points
- any internal context assembly code introduced by `coder-node`

Exit criteria:

- Agent workflows stop rehydrating obviously stale knowledge by default.

### Phase 6: Audit, backfill, and rollout

Deliverables:

- Add `memory audit`.
- Backfill `canonical_key`, `status`, and `last_verified_at` for existing data where possible.
- Run audit and manually review high-risk conflicts.

Backfill rules:

- If a key has one obvious latest record and older records with same normalized title, mark older ones as `superseded`.
- If a record is an old incident or one-time event, mark with shorter verification SLA.
- If no reliable backfill decision exists, keep the record active but flag it for audit.

Exit criteria:

- Legacy data no longer dominates default search because of missing lifecycle metadata.

## Testing Plan

### Unit tests

- lifecycle metadata parsing
- taxonomy validation
- freshness multiplier rules by memory type
- supersede chain updates

### Integration tests

- active-only search excludes superseded rows
- `--include-stale` returns historical rows
- `--as-of` returns time-correct rows
- conflicting active rows are detected
- `--replace-active` updates version chain atomically

### Regression tests

- old memory and new memory with same semantics: default search returns new active memory
- expired event memories do not leak into context injection
- decision/rule memories stay retrievable when still active and verified

## Observability

Add metrics before broad rollout.

Recommended metrics:

- `memory_search_active_hit_rate`
- `memory_search_superseded_result_rate`
- `memory_search_conflict_count`
- `memory_verify_backlog_count`
- `memory_active_unverified_over_sla_count`
- `memory_search_latency_ms`

Recommended structured logs:

- query
- filters used
- active/stale candidate counts
- canonical collapse count
- conflict count
- final selected IDs

## Risks and Open Questions

1. Taxonomy migration:
   existing `preference` and `skill` rows need a deterministic mapping plan.
2. Backfill safety:
   automatically superseding records from historical data can be wrong if titles are ambiguous.
3. Score tuning:
   freshness multipliers can over-penalize useful older rules if tuned too aggressively.
4. Compatibility:
   older clients should continue to work even if they do not pass lifecycle fields.

## Definition of Done

This plan is complete when all of the following are true:

- Default `coder memory search` returns only active, currently valid memories.
- Superseded or expired memories require explicit opt-in to appear.
- Memory updates can supersede previous versions without manual delete workflows.
- Agent context injection uses lifecycle-aware filtered retrieval.
- Audit tooling exists for long-unverified and conflicting memories.
- Docs, CLI help text, and transport contracts all describe the same lifecycle model.
