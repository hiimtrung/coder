---
name: coder-cli
description: Expert knowledge of the coder CLI and coder-node architecture. Covers all AI workflow commands, full project lifecycle commands, HTTP API endpoints, domain structure, .coder/ directory layout, STATE.md format, and XML plan format.
---

# Skill: coder CLI Architecture

## Overview

`coder` is a Go CLI + local server system for AI-powered development workflows.
It has two binaries:

| Binary | Role |
|--------|------|
| `coder` | CLI (developer-facing) — thin client, no LLM calls |
| `coder-node` | Local HTTP+gRPC server — all LLM calls, memory, skills |

The CLI **never** calls Ollama directly. All intelligence lives in `coder-node`.

---

## Command Groups

### Group A — Quick AI Workflow Commands

Work on any project without a `.coder/` directory.

| Command | Purpose | Key flags |
|---------|---------|-----------|
| `coder chat` | Q&A with memory+skill context injection | `--resume`, `--session`, `--file`, `--no-memory` |
| `coder review` | Structured AI code review | `--pr`, `--file`, `--focus`, `--format json` |
| `coder debug` | Root cause analysis for errors | `--file`, `--context`, `--diff`, `--interactive` |
| `coder plan` | 3-stage planning: Q&A → research → PLAN.md | `--auto`, `--prd`, `--list` |
| `coder qa` | UAT verification from PLAN.md criteria | `--plan`, `--resume`, `--list`, `--report` |
| `coder session` | Save/restore working context | `save`, `resume`, `list`, `export` |
| `coder workflow` | Auto-chain: plan→review→implement→qa→fix | `--steps`, `--dry-run`, `--resume` |

### Group B — Project Lifecycle Commands

Require `coder new-project` to initialize. All state tracked in `.coder/STATE.md`.

| Command | Purpose | Output |
|---------|---------|--------|
| `coder new-project "idea"` | Q&A → REQUIREMENTS.md, ROADMAP.md, STATE.md | `.coder/PROJECT.md` etc. |
| `coder map-codebase` | 4-pass codebase analysis → structured docs | `.coder/codebase/*.md` |
| `coder discuss-phase N` | Gray-area Q&A → CONTEXT.md | `.coder/phases/NN-CONTEXT.md` |
| `coder plan-phase N` | Research + XML plans + verification loop | `.coder/phases/NN-*-PLAN.md` |
| `coder execute-phase N` | Wave-based execution + atomic git commits | `.coder/phases/NN-*-SUMMARY.md` |
| `coder ship [N]` | `gh pr create` with AI-generated PR body | PR URL saved to STATE.md |
| `coder progress` | Show phases, step, blockers, PRs | — |
| `coder next` | Print next recommended command | one-liner |
| `coder milestone <action>` | audit / complete / archive / next | STATE.md updated |

### Group C — Project Utilities

| Command | Purpose |
|---------|---------|
| `coder todo` | Manage backlog (list / add / done / clear) |
| `coder stats` | Project statistics (phases, commits, plans, files) |
| `coder health` | Health check (artifacts, blockers, stale state) |
| `coder note <text>` | Record decision, blocker, or backlog item |
| `coder do "<task>"` | One-off AI task with full project context injected |

### Memory Lifecycle Commands

Use these when working with semantic memory quality, not just capture:

| Command | Purpose |
|---------|---------|
| `coder memory store` | Create a new memory entry; supports lifecycle metadata and `--replace-active` |
| `coder memory search` | Lifecycle-aware retrieval; defaults to active-only and may return a conflict summary |
| `coder memory verify` | Refresh `last_verified_at`, `verified_by`, `confidence`, and `source_ref` for a memory version |
| `coder memory supersede` | Mark one version group as replaced by another and link the chain |
| `coder memory audit` | Report active conflicts, expired active memories, long-unverified memories, and missing lifecycle columns |

---

## Full Lifecycle Flow

```
coder new-project "idea"
       │  writes: PROJECT.md, REQUIREMENTS.md, ROADMAP.md, STATE.md
       ▼
coder map-codebase
       │  writes: .coder/codebase/STACK.md + ARCHITECTURE.md + CONVENTIONS.md + CONCERNS.md
       ▼
coder discuss-phase N
       │  interactive Q&A → writes: .coder/phases/NN-CONTEXT.md
       │  STATE.md: step=plan
       ▼
coder plan-phase N
       │  research → XML plans → verification loop
       │  writes: .coder/phases/NN-RESEARCH.md + NN-01-PLAN.md + NN-VERIFICATION.md
       │  STATE.md: step=execute
       ▼
coder execute-phase N
       │  wave-based task execution + git commit per task
       │  writes: .coder/phases/NN-*-SUMMARY.md + NN-VERIFICATION.md
       │  STATE.md: step=qa
       ▼
coder ship N
       │  git push + gh pr create + AI-generated body
       │  STATE.md: step=ship, PRs[N]=url
       ▼
coder milestone complete N
       │  records decision → STATE.md: step=done
       ▼
coder milestone next
       │  advances current_phase → N+1, step=discuss
       ▼
coder discuss-phase N+1  ...repeat...
```

Use `coder next` at any point to get the recommended next command.

---

## .coder/ Directory Layout

```
.coder/
├── PROJECT.md            # project name, description, tech stack
├── REQUIREMENTS.md       # full requirements (from new-project)
├── ROADMAP.md            # phases with goals and status
├── STATE.md              # current phase, step, decisions, blockers, PRs
│
├── codebase/             # map-codebase output
│   ├── STACK.md          # language, frameworks, libraries
│   ├── INTEGRATIONS.md   # external APIs + events
│   ├── ARCHITECTURE.md   # pattern, layers, data flow
│   ├── STRUCTURE.md      # annotated directory tree
│   ├── CONVENTIONS.md    # naming, error handling, imports
│   ├── TESTING.md        # test locations, coverage, how to run
│   └── CONCERNS.md       # security, debt, missing tests, perf
│
├── phases/               # per-phase artifacts (N = phase number)
│   ├── NN-CONTEXT.md         ← discuss-phase output
│   ├── NN-RESEARCH.md        ← plan-phase step A output
│   ├── NN-01-PLAN.md         ← plan-phase step B output (one per plan)
│   ├── NN-02-PLAN.md
│   ├── NN-VERIFICATION.md    ← plan-phase checker + execute-phase verifier
│   ├── NN-01-SUMMARY.md      ← execute-phase output (one per plan)
│   └── NN-02-SUMMARY.md
│
├── archive/NN/           ← milestone archive output
│
├── plans/                ← coder plan output (Mode A)
├── qa/                   ← coder qa output (Mode A)
├── sessions/             ← coder session output (Mode A)
└── workflows/            ← coder workflow output (Mode A)
```

---

## STATE.md Format

```markdown
project: My Task Manager CLI
current_phase: 2
step: execute
last_action: plan-phase 1 completed
updated: 2026-03-23T14:30:00+07:00

## Decisions
- [2026-03-20] decided to use JWT with refresh tokens

## Blockers
- [2026-03-22] waiting for API credentials from client

## Backlog
- investigate rate limiting middleware

## PRs
- phase 1: https://github.com/org/repo/pull/42
```

**Step → next command mapping:**

| step | next command |
|------|-------------|
| (empty) | `coder map-codebase` |
| `discuss` | `coder plan-phase N` |
| `plan` | `coder execute-phase N` |
| `execute` | `coder execute-phase N --gaps-only` |
| `qa` | `coder qa --phase N` |
| `ship` | `coder milestone complete N` |
| `done` | `coder milestone next` (→ `discuss-phase N+1`) |

---

## XML Plan Format

`plan-phase` generates plans; `execute-phase` parses them with `parsePlanXML()`:

```xml
<plan id="1-01" phase="1" name="JWT Middleware">
  <objective>Implement JWT authentication middleware with refresh token support</objective>
  <files>
    internal/middleware/auth.go
    internal/middleware/auth_test.go
  </files>
  <dependencies>none</dependencies>
  <estimated_time>1h</estimated_time>
  <tasks>
    <task type="create" name="JWT validator middleware">
      <action>
        Implement JWT validation in internal/middleware/auth.go.
        Use github.com/golang-jwt/jwt/v5. Return 401 on invalid tokens.
      </action>
      <verify>go test ./internal/middleware/... passes</verify>
      <done>All protected routes return 401 without valid JWT</done>
    </task>
  </tasks>
</plan>
```

**Task type → git commit prefix:**
- `create` / `modify` → `feat`
- `test` → `test`
- `delete` → `chore`
- `fix` → `fix`

**Wave-based execution:** Plans with `<dependencies>none</dependencies>` execute in wave 1.
Plans that depend on other plan IDs execute in later waves (sequential).

---

## coder-node HTTP API Endpoints

```
POST /v1/chat              — LLM completion (blocking)
POST /v1/chat/stream       — LLM completion (SSE streaming)
GET  /v1/sessions          — List chat sessions
GET  /v1/sessions/:id      — Get session messages
DELETE /v1/sessions/:id    — Delete session

POST /v1/review            — Structured code review
POST /v1/debug             — Root cause analysis

POST /v1/memory/store      — Store memory entry
POST /v1/memory/search     — Semantic memory search
POST /v1/memory/verify     — Verify an existing memory version
POST /v1/memory/supersede  — Supersede one memory version with another
POST /v1/memory/audit      — Audit lifecycle issues in memory

POST /skill/search         — Vector skill search
POST /skill/ingest         — Ingest skill files

POST /auth/login           — Login (secure mode)
POST /auth/token/rotate    — Rotate token
GET  /health               — Health check + secure_mode flag
```

---

## Project Structure

```
cmd/
  coder/              ← CLI binary
    main.go           — command dispatch
    state.go          — ProjectState, loadState(), saveState(), loadRoadmap()
    config.go         — ~/.coder/config.json loader
    cmd_chat.go       — coder chat
    cmd_review.go     — coder review
    cmd_debug.go      — coder debug
    cmd_plan.go       — coder plan
    cmd_qa.go         — coder qa
    cmd_session.go    — coder session
    cmd_workflow.go   — coder workflow
    cmd_new_project.go    — coder new-project
    cmd_map_codebase.go   — coder map-codebase
    cmd_discuss_phase.go  — coder discuss-phase (+ helpers)
    cmd_plan_phase.go     — coder plan-phase (+ parsePlanXML, XML extraction)
    cmd_execute_phase.go  — coder execute-phase (+ executePlan, groupPlansIntoWaves)
    cmd_ship.go           — coder ship
    cmd_progress.go       — coder progress + coder next
    cmd_milestone.go      — coder milestone
    cmd_todo.go           — coder todo + stats + health + note + do

  coder-node/         ← Server binary
    main.go           — wires all dependencies + starts HTTP + gRPC

internal/
  domain/
    chat/             — Session, Message, ChatRequest/Response, interfaces
    review/           — ReviewRequest, ReviewResult, ReviewConcern
    debug/            — DebugRequest, DebugResult, DebugContext
    skill/            — Skill, SkillChunk, SkillSearchResult
    memory/           — Knowledge entities, lifecycle metadata, MemoryManager interfaces
    auth/             — AuthManager, Token interfaces

  usecase/
    chat/manager.go   — context injection pipeline (parallel memory+skill search)
    review/manager.go — review prompt builder + JSON parser
    debug/manager.go  — debug prompt builder + JSON parser
    skill/            — ingestor, facade
    memory/           — manager
    auth/             — manager (secure/open mode)

  infra/
    llm/ollama.go     — Ollama /api/chat client (stream + blocking)
    postgres/
      chat.go         — coder_sessions + coder_messages tables
      memory.go       — pgvector memory storage
      skill.go        — pgvector skill storage
      auth.go         — tokens table

  transport/
    http/
      server/         — HTTP handlers (chat, review, debug, memory, skill, auth)
      client/         — HTTP clients used by the CLI
      middleware/     — auth middleware
    grpc/             — gRPC server for memory + skill
```

---

## Context Injection Pipeline

Every chat/review/debug request runs this pipeline in `usecase/chat/manager.go`:

```
User message
     │
     ├─ goroutine 1: memory.Search(message, limit=5)  ─┐
     ├─ goroutine 2: skill.Search(message, limit=3)   ─┤  300ms timeout
     │                                                  ┘
     ▼
Build system prompt:
  [base system prompt]
  + [skill context chunks]
  + [memory context entries or conflict summaries]
  + [session history (last N messages)]
     │
     ▼
POST Ollama /api/chat
     │
     ▼
Persist user + assistant messages → coder_sessions / coder_messages
```

---

## Key Patterns

### SSE Streaming
- Server: `Content-Type: text/event-stream`, write `data: {...}\n\n`, flush each chunk
- Client: `bufio.Scanner`, strip `data: ` prefix, parse JSON delta

### STATE.md mutations
All lifecycle commands call `loadState()` at start and `saveState()` at end.
Never modify STATE.md manually during a workflow — use `coder note` instead.

### Memory lifecycle
- Default `coder memory search` is active-only and validity-aware unless `--include-stale` is passed.
- Conflicting active versions collapse into a synthesized summary unless `--history` is requested.
- Prefer `coder memory verify` over `coder memory store` when you are only reconfirming an existing memory.
- Prefer `coder memory supersede` or `coder memory store --replace-active` when replacing existing guidance.

### Config
`~/.coder/config.json` — connection settings (baseURL, accessToken):
```json
{
  "memory": { "base_url": "http://localhost:8080" },
  "auth":   { "access_token": "..." }
}
```

### Auth Modes
- **Open mode** (default): no token required, `SECURE_MODE=false`
- **Secure mode**: token required, `SECURE_MODE=true`, bootstrap token printed on first start

---

## Rules Reference

| Priority | Rule file | Topic |
|----------|-----------|-------|
| 1 | [Architecture](rules/architecture.md) | Layer structure, naming, separation |
| 2 | [Commands](rules/commands.md) | Adding new CLI commands |
| 3 | [API](rules/api.md) | Adding new coder-node endpoints |
