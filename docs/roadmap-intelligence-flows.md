# Coder Intelligence Flows — Roadmap

> **Inspired by:** [get-shit-done](https://github.com/glittercowboy/get-shit-done) — context engineering + spec-driven development system
> **Goal:** Evolve coder from a RAG/memory CLI into a full **AI development workflow engine** — with Q&A, review, planning, QA, and debug capabilities on par with a senior engineer AI pair.
> **Last updated:** 2026-03-23

## Implementation Status

| Phase | Feature                                      | Status     |
| ----- | -------------------------------------------- | ---------- |
| 1     | LLM Backbone (coder-node)                    | ✅ Done    |
| 2     | coder chat                                   | ✅ Done    |
| 3     | coder review                                 | ✅ Done    |
| 4     | coder plan                                   | ✅ Done    |
| 5     | coder qa                                     | ✅ Done    |
| 6     | coder debug                                  | ✅ Done    |
| 7     | coder session                                | ✅ Done    |
| 8     | coder workflow                               | ✅ Done    |
| 9     | coder new-project                            | 🔲 Planned |
| 10    | coder map-codebase                           | 🔲 Planned |
| 11    | coder discuss-phase                          | 🔲 Planned |
| 12    | coder plan-phase (upgrade)                   | 🔲 Planned |
| 13    | coder execute-phase (subagent execution)     | 🔲 Planned |
| 14    | coder ship                                   | 🔲 Planned |
| 15    | coder progress / coder next                  | 🔲 Planned |
| 16    | coder milestone                              | 🔲 Planned |
| 17    | Utilities (todo, stats, health, do, note)    | 🔲 Planned |
| 18    | Subagent agent definitions (.claude/agents/) | 🔲 Planned |

---

## Architecture Overview

```
Developer / AI Agent
      │
      │  coder CLI
      │  ├─ coder chat           ← Q&A with context injection      (Phase 2)  ✅
      │  ├─ coder review         ← Multi-model code review         (Phase 3)  ✅
      │  ├─ coder plan           ← Planning workflow               (Phase 4)  ✅
      │  ├─ coder qa             ← QA / UAT verification           (Phase 5)  ✅
      │  ├─ coder debug          ← Root cause diagnosis            (Phase 6)  ✅
      │  ├─ coder session        ← State management                (Phase 7)  ✅
      │  ├─ coder workflow       ← Auto-chain orchestration        (Phase 8)  ✅
      │  │
      │  ├─ coder new-project    ← Project init + docs scaffold    (Phase 9)  🔲
      │  ├─ coder map-codebase   ← Parallel codebase analysis      (Phase 10) 🔲
      │  ├─ coder discuss-phase  ← Gray-area Q&A → CONTEXT.md      (Phase 11) 🔲
      │  ├─ coder plan-phase     ← Research + plan + verify loop   (Phase 12) 🔲
      │  ├─ coder execute-phase  ← Wave-based subagent execution   (Phase 13) 🔲
      │  ├─ coder ship           ← Push branch + create PR         (Phase 14) 🔲
      │  ├─ coder progress/next  ← State-aware navigation          (Phase 15) 🔲
      │  ├─ coder milestone      ← Lifecycle: complete/new/audit   (Phase 16) 🔲
      │  └─ coder todo/stats/... ← Utilities                       (Phase 17) 🔲
      │
      ▼
┌──────────────────────────────────────────────────────┐
│                    coder-node                        │
│                                                      │
│  NEW: POST /v1/chat            (LLM completion)      │
│  NEW: POST /v1/chat/stream     (SSE streaming)       │
│  NEW: POST /v1/review          (code review)         │
│  NEW: GET  /v1/sessions        (list sessions)       │
│  NEW: GET  /v1/sessions/:id    (session history)     │
│                                                      │
│  Existing: gRPC :50051 / HTTP :8080                  │
│  Existing: Memory · Skill · Auth · Activity          │
│                                                      │
│  ┌──────────────────────────────────────────────┐    │
│  │         Context Injection Pipeline           │    │
│  │  request → parallel(memory, skill) search    │    │
│  │          → build enriched system prompt      │    │
│  │          → forward to Ollama /api/chat       │    │
│  │          → persist session + log activity    │    │
│  └──────────────────────────────────────────────┘    │
│                                                      │
│  PostgreSQL + pgvector · Ollama · (OpenAI-compat)    │
└──────────────────────────────────────────────────────┘
```

---

## Phase 1 — LLM Backbone (coder-node)

> **Priority:** P0 — Foundation for all subsequent phases
> **Effort:** ~5 days
> **Depends on:** nothing (extend existing)

### Objective

Turn coder-node into an **intelligent LLM proxy** — not just embed/search but also generate, automatically injecting memory + skill context into every request. This is the only layer that calls Ollama to generate text; the CLI never calls Ollama directly.

### 1.1 — New API endpoints

#### `POST /v1/chat`

```
Request:
{
  "message":    "How should I implement JWT refresh tokens?",
  "session_id": "abc123",          // optional — resume conversation
  "context": {                     // optional overrides
    "inject_memory": true,         // default: true
    "inject_skills":  true,        // default: true
    "memory_limit":  5,            // top-N memory results injected
    "skill_limit":   3,            // top-N skill results injected
    "extra_system":  "..."         // append to system prompt
  }
}

Response:
{
  "reply":      "For JWT refresh tokens, the recommended pattern is...",
  "session_id": "abc123",
  "context_used": {
    "memory_hits": ["JWT auth pattern (2025-11)", "Token rotation fix"],
    "skill_hits":  ["nestjs:auth", "general-patterns:security"]
  },
  "model":  "qwen3.5:0.8b",
  "tokens": { "prompt": 1240, "completion": 380 }
}
```

#### `POST /v1/chat/stream`

Same as `/v1/chat` but Server-Sent Events:

```
data: {"delta": "For JWT"}
data: {"delta": " refresh tokens, the recommended"}
data: {"delta": " pattern is..."}
data: {"done": true, "session_id": "abc123", "tokens": {...}}
```

#### `GET /v1/sessions`

```
Response:
{
  "sessions": [
    {
      "id": "abc123",
      "title": "JWT refresh tokens",   // auto-generated from first message
      "message_count": 4,
      "updated_at": "2026-03-20T10:00Z"
    }
  ]
}
```

#### `GET /v1/sessions/:id`

Returns the full conversation history (messages array).

#### `DELETE /v1/sessions/:id`

Deletes the session.

### 1.2 — Context Injection Pipeline

```
POST /v1/chat receives request
        │
        ▼
┌───────────────────────────────────────────────────────┐
│ Step 1: Extract search keywords                       │
│   Take first 15 words + noun phrases from message     │
│   No NLP needed — simple heuristic is sufficient      │
├───────────────────────────────────────────────────────┤
│ Step 2: Parallel context search (goroutine)           │
│   a. memory.Search(keywords, limit=5)                 │
│   b. skill.Search(keywords, limit=3)                  │
│   Timeout: 300ms — return empty context if exceeded   │
├───────────────────────────────────────────────────────┤
│ Step 3: Build enriched system prompt                  │
│                                                       │
│   [BASE SYSTEM PROMPT]                                │
│   You are a senior software engineer AI assistant.   │
│   Answer concisely and precisely.                     │
│                                                       │
│   [SKILL CONTEXT — if hits found]                     │
│   ## Relevant patterns and rules:                     │
│   {skill_chunk_1_content}                             │
│   {skill_chunk_2_content}                             │
│                                                       │
│   [MEMORY CONTEXT — if hits found]                    │
│   ## Past decisions and learnings:                    │
│   {memory_1_content}                                  │
│   {memory_2_content}                                  │
├───────────────────────────────────────────────────────┤
│ Step 4: Build messages array                          │
│   [system: enriched prompt]                           │
│   [user/assistant: session history (last 20 msgs)]    │
│   [user: current message]                             │
├───────────────────────────────────────────────────────┤
│ Step 5: POST to Ollama /api/chat                      │
│   stream=true  → forward SSE to client                │
│   stream=false → wait for full response               │
├───────────────────────────────────────────────────────┤
│ Step 6: Persist + log                                 │
│   - upsert session (coder_sessions)                   │
│   - append user + assistant messages (coder_messages) │
│   - log activity "chat" (existing activity system)    │
└───────────────────────────────────────────────────────┘
```

### 1.3 — New PostgreSQL schema

```sql
CREATE TABLE coder_sessions (
    id          TEXT PRIMARY KEY,
    client_id   TEXT NOT NULL REFERENCES coder_clients(id) ON DELETE CASCADE,
    title       TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP NOT NULL
);

CREATE TABLE coder_messages (
    id          TEXT PRIMARY KEY,
    session_id  TEXT NOT NULL REFERENCES coder_sessions(id) ON DELETE CASCADE,
    role        TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
    content     TEXT NOT NULL,
    tokens_in   INT  DEFAULT 0,
    tokens_out  INT  DEFAULT 0,
    created_at  TIMESTAMP NOT NULL
);

CREATE INDEX idx_sessions_client  ON coder_sessions(client_id, updated_at DESC);
CREATE INDEX idx_messages_session ON coder_messages(session_id, created_at ASC);
```

### 1.4 — New code structure

```
internal/domain/chat/
  entity.go         — Session, Message, ChatRequest, ChatResponse types
  port.go           — ChatRepository, ChatManager, LLMProvider interfaces

internal/usecase/chat/
  manager.go        — orchestrate: inject context → call LLM → persist

internal/infra/llm/
  ollama.go         — Ollama /api/chat client (stream + non-stream)
  openai.go         — OpenAI-compatible endpoint (optional fallback)

internal/infra/postgres/
  chat.go           — ChatRepository impl (sessions + messages CRUD)

internal/transport/http/server/
  chat.go           — handleChat, handleChatStream, handleSessions
```

### 1.5 — Config extension

```json
// ~/.coder/config.json — new section added
{
  "chat": {
    "model": "qwen3.5:0.8b",
    "stream": true,
    "inject_memory": true,
    "inject_skills": true,
    "memory_limit": 5,
    "skill_limit": 3,
    "history_limit": 20
  }
}
```

### 1.6 — Acceptance criteria

- [ ] `POST /v1/chat` returns a reply with context from memory + skills injected
- [ ] `POST /v1/chat/stream` streams SSE in the correct format; client receives deltas
- [ ] Session persisted to DB; `GET /v1/sessions/:id` retrieves history correctly
- [ ] Context injection: 2 search queries run in parallel, timeout 300ms
- [ ] Response includes `context_used` for debugging/verifying context injection
- [ ] Ollama unavailable → clear error with actionable fix instructions
- [ ] Activity "chat" logged to the activity table
- [ ] Unit tests for context injection pipeline
- [ ] Build + vet + tests pass

---

## Phase 2 — `coder chat` (Q&A Flow)

> **Priority:** P1
> **Effort:** ~3 days
> **Depends on:** Phase 1

### Objective

Interactive Q&A CLI with AI, automatically injecting memory + skill context. Equivalent to the `discuss-phase` of GSD but general-purpose. Developers can ask anything — coder responds with full project context.

### 2.1 — CLI interface

```sh
# Interactive REPL (default)
coder chat

# Single question, non-interactive
coder chat "what is the best way to handle DB migrations in NestJS?"

# Resume last session
coder chat --resume

# Resume specific session
coder chat --session abc123

# Load extra file context
coder chat --file path/to/error.log "what is causing this error?"
coder chat --file src/auth/service.ts "review this file"

# Pipe input
echo "explain: $(cat error.log)" | coder chat

# Disable context injection (raw mode)
coder chat --no-memory --no-skills "explain what a goroutine is"

# List sessions
coder chat --list

# Delete session
coder chat --delete abc123
```

### 2.2 — Interactive REPL

```
$ coder chat

╔══════════════════════════════════════════╗
║  coder chat  ·  session: new             ║
║  /help · /sessions · /clear · /exit      ║
╚══════════════════════════════════════════╝

You › how should I structure error handling in NestJS?

  ⟳ Searching context... memory(2) skill(1)

Assistant › Based on your project patterns and NestJS best practices:

  Use a global exception filter with standardized error codes...
  [full response]

  ──────────────────────────────────────────
  Context: nestjs:error-handling · VAL_*/BIZ_* patterns from memory

You › can you show me a concrete example with AUTH errors?

  ⟳ Searching context... memory(1) skill(2)

Assistant › Here's the AUTH error pattern matching your codebase...
  [response continues]

You › /exit

Session saved: abc123 — "error handling in NestJS" (2 messages)
```

### 2.3 — Slash commands in REPL

| Command           | Action                                       |
| ----------------- | -------------------------------------------- |
| `/help`           | Show available commands                      |
| `/sessions`       | List recent sessions                         |
| `/resume <id>`    | Load a session                               |
| `/clear`          | Clear conversation history (keep session ID) |
| `/context`        | Show currently injected context              |
| `/model <name>`   | Switch model for this session                |
| `/save <note>`    | Save session with a custom title             |
| `/exit` or Ctrl+C | Exit and auto-save session                   |

### 2.4 — Internal CLI flow

```
coder chat "question"
      │
      ▼
1. loadConfig()                   — read ~/.coder/config.json
2. loadSession() / newSession()   — resume or create new
3. POST /v1/chat/stream           — send message to coder-node
      │                             (coder-node auto-injects context)
      ▼
4. Stream display                 — print delta tokens in real-time
      │
      ▼
5. Show context used              — "Context: [memory hits] [skill hits]"
6. Wait for next input (REPL)     — or exit if single-question mode
```

### 2.5 — File: `cmd/coder/cmd_chat.go`

Key functions:

```go
func runChat(args []string)
func runChatREPL(cfg Config, sessionID string)
func runChatSingle(cfg Config, message string, sessionID string)
func streamResponse(resp *http.Response) string
func printDelta(delta string)
func loadOrCreateSession(cfg Config) (sessionID string)
func listSessions(cfg Config)
```

### 2.6 — Acceptance criteria

- [ ] `coder chat "question"` returns a context-aware answer in < 3s
- [ ] Interactive REPL works with stdin/stdout
- [ ] Stream output renders in real-time (delta by delta)
- [ ] Session auto-saved on exit
- [ ] `--resume` and `--session <id>` load the correct history
- [ ] `--file` injects file content into context
- [ ] Slash commands `/clear`, `/sessions`, `/exit` work correctly
- [ ] `coder chat --list` displays most recent sessions
- [ ] Clear error message if coder-node is unreachable

---

## Phase 3 — `coder review` (Code Review Flow)

> **Priority:** P1
> **Effort:** ~4 days
> **Depends on:** Phase 1

### Objective

Structured AI code review — reads a git diff or specific files and returns structured feedback (Summary, Strengths, Concerns with severity, Suggestions). Inspired by the `review.md` pattern in GSD: adversarial multi-model review.

### 3.1 — CLI interface

```sh
# Review git diff (staged + unstaged)
coder review

# Review only staged changes
coder review --staged

# Review specific file(s)
coder review src/auth/service.go
coder review src/auth/service.go src/auth/handler.go

# Review a GitHub PR
coder review --pr 123
coder review --pr https://github.com/org/repo/pull/123

# Review with a specific focus
coder review --focus security
coder review --focus performance
coder review --focus "error handling"

# Multi-model review (if multiple providers configured)
coder review --all-models
coder review --model gpt-4o

# Output format
coder review --format json      # machine-readable
coder review --format markdown  # save to file
coder review -o review.md       # save output

# Severity filter
coder review --min-severity high  # show only HIGH concerns
```

### 3.2 — Review output format

```
══════════════════════════════════════════════════════════
  CODE REVIEW  ·  src/auth/service.go  ·  2026-03-20
══════════════════════════════════════════════════════════

SUMMARY
  The authentication service looks well-structured overall.
  Main concern is missing rate limiting on the login endpoint
  and token expiry not being validated on refresh.

STRENGTHS
  ✓ Clean separation of concerns — business logic in service,
    no DB queries leaking into handler
  ✓ Error codes follow project standard (AUTH_*/BIZ_*)
  ✓ Token hash stored correctly with SHA-256

CONCERNS
  ● [HIGH]   Token refresh doesn't validate expiry on the old token.
             An expired token could be used to get a new one indefinitely.
             File: src/auth/service.go:142

  ● [HIGH]   No rate limiting on POST /v1/auth/login.
             Brute-force attacks possible.
             Suggestion: add middleware or use golang.org/x/time/rate

  ● [MEDIUM] Missing test for the token rotation edge case
             (concurrent rotation requests).
             File: src/auth/service_test.go

  ● [LOW]    Variable name `t` is ambiguous on line 89.
             Prefer `token` or `rawToken` for clarity.

SUGGESTIONS
  1. Add expiry check before allowing refresh (line 142)
  2. Implement rate limiting middleware — see existing pattern
     in internal/transport/http/middleware/
  3. Add concurrent token rotation test
  4. Rename variables for clarity

──────────────────────────────────────────────────────────
  3 files reviewed · 4 concerns (2 HIGH, 1 MEDIUM, 1 LOW)
  Model: qwen3.5:0.8b · Context: auth patterns (memory)
══════════════════════════════════════════════════════════
```

### 3.3 — Server endpoint: `POST /v1/review`

```
Request:
{
  "type":    "diff" | "file" | "pr",
  "content": "--- a/src/auth/service.go\n+++ b/...",  // diff or file content
  "focus":   "security",                               // optional focus area
  "context": {
    "inject_memory": true,
    "inject_skills":  true
  }
}

Response:
{
  "summary":   "The authentication service...",
  "strengths": ["Clean separation...", "Error codes follow..."],
  "concerns": [
    {
      "severity":    "HIGH",
      "description": "Token refresh doesn't validate expiry...",
      "location":    "src/auth/service.go:142",
      "suggestion":  "Add expiry check before allowing refresh"
    }
  ],
  "suggestions": ["Add expiry check...", "Implement rate limiting..."],
  "stats": {
    "files_reviewed": 3,
    "concerns_high":   2,
    "concerns_medium": 1,
    "concerns_low":    1
  },
  "context_used": { "memory_hits": [...], "skill_hits": [...] }
}
```

### 3.4 — Review system prompt

```
You are a senior code reviewer. Analyze the following code changes and provide structured feedback.

## Project context (from memory)
{memory_context}

## Relevant patterns and standards (from skills)
{skill_context}

## Focus area
{focus_area if set, else "general quality, security, performance, maintainability"}

## Instructions
Return a JSON object with this exact structure:
{
  "summary": "one paragraph overall assessment",
  "strengths": ["strength 1", "strength 2"],
  "concerns": [
    {
      "severity": "HIGH|MEDIUM|LOW",
      "description": "clear description",
      "location": "file:line if known",
      "suggestion": "concrete fix"
    }
  ],
  "suggestions": ["improvement 1", "improvement 2"]
}

Severity guide:
  HIGH   — security issue, data loss risk, production bug
  MEDIUM — correctness issue, missing test, bad pattern
  LOW    — style, naming, minor clarity

## Code to review:
{diff_or_file_content}
```

### 3.5 — Multi-model review (adversarial)

When multiple models are configured or `--all-models` is passed:

```
coder review --all-models

  ⟳ Reviewing with qwen3.5:0.8b...  done
  ⟳ Reviewing with gemma2:9b...        done

Synthesizing consensus...

══════════════════════════════════════════
  MULTI-MODEL REVIEW CONSENSUS
══════════════════════════════════════════

AGREED CONCERNS (raised by 2+ models)
  ● [HIGH] Token refresh expiry not checked  ← qwen3.5 + gemma2

UNIQUE CONCERNS — qwen3.5 only
  ● [LOW] Variable naming on line 89

UNIQUE CONCERNS — gemma2 only
  ● [MEDIUM] Missing idempotency key on retry logic

DIVERGENT VIEWS
  Both models disagree on the error handling approach.
  Worth investigating manually.
```

### 3.6 — Acceptance criteria

- [ ] `coder review` diffs git correctly and displays structured output
- [ ] `coder review file.go` reviews a single file
- [ ] `--pr` fetches the GitHub PR diff via the `gh` CLI
- [ ] Response JSON is parsed and displayed cleanly
- [ ] `--focus security` narrows review to security concerns only
- [ ] Memory + skill context injected correctly into the review prompt
- [ ] `--format json` outputs machine-readable JSON
- [ ] `-o file.md` saves output to file
- [ ] Activity "review" logged
- [ ] Build + tests pass

---

## Phase 4 — `coder plan` (Planning Flow)

> **Priority:** P2
> **Effort:** ~6 days
> **Depends on:** Phase 1, Phase 2

### Objective

Planning workflow: receive a feature description or PRD → ask clarifying questions (Q&A) → research → generate a structured PLAN.md. Equivalent to the `discuss-phase` + `plan-phase` of GSD.

### 4.1 — CLI interface

```sh
# Interactive planning session
coder plan "implement user authentication with JWT"

# From a PRD document
coder plan --prd path/to/prd.md

# Skip Q&A, auto-generate plan
coder plan --auto "implement caching layer with Redis"

# Plan for a specific file/module
coder plan --file src/auth/service.go "refactor this to use the new token manager"

# Output file
coder plan -o .coder/plans/PLAN-auth.md "implement auth"

# List existing plans
coder plan --list
```

### 4.2 — Planning flow (3 stages)

```
Stage 1: Q&A (discuss)
──────────────────────
Receive feature description
      │
      ▼
Analyze → identify gray areas (ambiguous decisions)
      │
      ▼
Interactive Q&A loop:
  - Ask one question at a time
  - Offer concrete options (no open-ended questions)
  - Capture decisions into CONTEXT
  - Detect scope creep → defer to backlog
      │
      ▼
Write CONTEXT.md (decisions captured)

Stage 2: Research
─────────────────
Read CONTEXT.md decisions
      │
      ▼
Search memory + skills for relevant patterns
      │
      ▼
Synthesize research findings → RESEARCH.md

Stage 3: Plan generation
────────────────────────
Read CONTEXT.md + RESEARCH.md
      │
      ▼
Generate structured PLAN.md
      │
      ▼
Self-verify plan (check completeness, edge cases)
      │
      ▼
Present plan → user approves / requests changes
      │
      ▼
Save final PLAN.md
```

### 4.3 — Q&A interaction example

```
$ coder plan "implement user authentication with JWT"

╔════════════════════════════════════════════════════════╗
║  coder plan  ·  User Authentication                   ║
╚════════════════════════════════════════════════════════╝

Analyzing feature scope...

I found 4 areas that need your input before I can plan:
  1. Token storage strategy
  2. Refresh token behavior
  3. Multi-device support
  4. Session invalidation

Let's discuss each.

──────────────────────────────────────
  TOKEN STORAGE
──────────────────────────────────────

How should access tokens be stored on the client?

  [1] httpOnly cookie (recommended — XSS-safe, automatic expiry)
  [2] localStorage (simpler, but XSS risk)
  [3] Memory only (most secure, lost on refresh)
  [4] Other (describe)

Your choice › 1

──────────────────────────────────────
  REFRESH TOKEN BEHAVIOR
──────────────────────────────────────

How long should sessions stay valid?

  [1] 15min access + 7d refresh with rotation (recommended)
  [2] 1h access + 30d refresh without rotation
  [3] 24h access, no refresh
  [4] Custom (describe)

Your choice › 1

[... 2 more questions ...]

Decisions captured. Researching implementation patterns...

● Searching memory... 3 hits
● Searching skills... 2 hits (nestjs:auth, general-patterns:security)

Generating plan...

══════════════════════════════════════════════════════════
  PLAN: User Authentication with JWT
  Generated: 2026-03-20
══════════════════════════════════════════════════════════

## Overview
Implement JWT authentication with httpOnly cookies, 15-minute
access tokens, and 7-day rotating refresh tokens.

## Tasks

### Task 1: Auth domain entities (30 min)
- Create Token value object with expiry validation
- Create AuthSession entity
- Add TokenRepository interface to domain/auth/port.go

### Task 2: JWT service (45 min)
- Install golang-jwt/jwt/v5
- Implement JWTService.Sign(claims) → token string
- Implement JWTService.Verify(token) → claims
- Unit tests for sign + verify + expiry

### Task 3: Refresh token rotation (1h)
- PostgreSQL table: coder_refresh_tokens
  (id, client_id, token_hash, expires_at, used_at)
- Implement rotation: validate old → invalidate → issue new
- Handle concurrent rotation (optimistic lock)
- Test: normal rotation, expired token, already-used token

### Task 4: HTTP handlers (45 min)
- POST /v1/auth/login → set httpOnly cookie
- POST /v1/auth/refresh → rotate token
- POST /v1/auth/logout → invalidate + clear cookie
- Update auth middleware to read cookie

### Task 5: Integration tests (30 min)
- Full login → refresh → logout flow
- Concurrent refresh race condition test

## Files to create/modify
- internal/domain/auth/token.go (new)
- internal/infra/postgres/token.go (new)
- internal/usecase/auth/jwt.go (new)
- internal/transport/http/server/auth.go (modify)

## Risks
- [MEDIUM] Concurrent refresh: use DB row-level lock
- [LOW] Cookie SameSite setting on mobile browsers

## Estimated total: ~3.5h

══════════════════════════════════════════════════════════

Accept this plan? [Y/n/edit] › Y

Plan saved: .coder/plans/PLAN-auth-jwt.md
```

### 4.4 — PLAN.md output format

```markdown
---
feature: "User Authentication with JWT"
created: "2026-03-20"
status: "approved"
estimated_hours: 3.5
---

# Plan: User Authentication with JWT

## Context (decisions from Q&A)

- Token storage: httpOnly cookie
- Session length: 15min access + 7d refresh with rotation
- Multi-device: yes, each device gets own refresh token
- Invalidation: logout clears all sessions for the device

## Research findings

- Existing auth patterns: SHA-256 token hash (already in codebase)
- JWT library: golang-jwt/jwt/v5 (community standard for Go)
- Refresh rotation: recommended by OWASP session mgmt guide

## Tasks

[... tasks with time estimates ...]

## Files

[... create/modify list ...]

## Risks

[... with severity ...]

## Deferred (out of scope — noted for later)

- OAuth2/social login (separate feature)
- 2FA/TOTP (separate feature)
```

### 4.5 — Acceptance criteria

- [ ] Q&A flow: automatically identifies gray areas, asks sequentially
- [ ] Options are concrete and include a "recommended" label
- [ ] Scope creep detection: if user mentions a new feature → defer + note it
- [ ] Research: searches memory + skills before generating the plan
- [ ] PLAN.md output includes complete tasks, time estimates, files, and risks
- [ ] `--auto` skips Q&A and uses sensible defaults
- [ ] `--prd` reads a PRD file and extracts requirements automatically
- [ ] Plan is saved and listable with `coder plan --list`
- [ ] Activity "plan" logged
- [ ] Build + tests pass

---

## Phase 5 — `coder qa` (QA / Verification Flow)

> **Priority:** P2
> **Effort:** ~5 days
> **Depends on:** Phase 1, Phase 4 (optional — can run standalone)

### Objective

UAT verification workflow: load acceptance criteria from a plan → present expected behavior for each test → user confirms pass or reports an issue → if issues found: auto-diagnose + generate fix plan. Persistent state across sessions — no progress lost on Ctrl+C. Equivalent to `verify-work` in GSD.

### 5.1 — CLI interface

```sh
# Start QA session from a plan
coder qa --plan .coder/plans/PLAN-auth-jwt.md

# Start with a feature description (auto-generate test cases)
coder qa "user authentication feature"

# Resume an in-progress session
coder qa --resume
coder qa --session qa-abc123

# List QA sessions
coder qa --list

# Run a specific test only
coder qa --test "3"

# Skip a test
coder qa --skip "cold start"

# Export report
coder qa --report -o qa-report.md
```

### 5.2 — QA session flow

```
coder qa --plan PLAN-auth-jwt.md
      │
      ▼
1. Parse plan → extract acceptance criteria + tasks
2. Generate test cases (one per task/acceptance criterion)
3. Save UAT.md (persistent state)
4. Present tests one by one

For each test:
  ┌─────────────────────────────────────────────────────┐
  │  TEST 3/8: Token Refresh                            │
  │                                                     │
  │  Expected:                                          │
  │  POST /v1/auth/refresh with valid refresh token     │
  │  → returns new access token in cookie               │
  │  → old refresh token is invalidated                 │
  │  → new refresh token returned                       │
  │                                                     │
  │  → Type "pass", describe the issue, or "skip"       │
  └─────────────────────────────────────────────────────┘

User: "the old refresh token is NOT being invalidated"

→ Logged as MAJOR issue
→ Severity inferred from description (no severity question)
→ Continue to next test

[After all tests]

══════════════════════════════════════════
  QA COMPLETE — 7 passed, 1 issue
══════════════════════════════════════════

Issues:
  ● [MAJOR] Token Refresh: old token not invalidated
            "the old refresh token is NOT being invalidated"

Diagnosing root cause...
  ⟳ Searching relevant code...

Root cause: usecase/auth/manager.go:156
  The UpdateAccessTokenHash call updates the hash but doesn't
  delete the old refresh token entry. Missing:
  repo.DeleteRefreshToken(ctx, oldTokenID)

Fix plan generated: .coder/plans/PLAN-auth-jwt-fix-1.md

Ready to fix:
  coder plan --list    # see fix plan
```

### 5.3 — UAT.md (persistent state file)

```markdown
---
id: qa-abc123
plan: .coder/plans/PLAN-auth-jwt.md
status: in_progress # new | in_progress | complete
started: 2026-03-20T10:00Z
updated: 2026-03-20T10:45Z
---

## Progress

total: 8 · passed: 7 · issues: 1 · skipped: 0 · pending: 0

## Current Test

number: 3
status: complete

## Tests

### 1. Login flow

expected: POST /v1/auth/login with valid credentials sets httpOnly cookie
result: pass

### 2. Invalid credentials

expected: POST /v1/auth/login with wrong password returns 401 AUTH_INVALID_CREDENTIALS
result: pass

### 3. Token refresh

expected: POST /v1/auth/refresh returns new tokens, invalidates old refresh token
result: issue
reported: "the old refresh token is NOT being invalidated"
severity: major
root_cause: "Missing DeleteRefreshToken call in manager.go:156"

[...]

## Issues

- id: issue-1
  test: 3
  severity: major
  description: "old refresh token not invalidated"
  root_cause: "Missing DeleteRefreshToken call in manager.go:156"
  fix_plan: .coder/plans/PLAN-auth-jwt-fix-1.md
```

### 5.4 — Auto-diagnosis

When the user reports an issue:

```
1. Extract keywords from the issue description
2. Search memory + skills for relevant patterns
3. Read related source files (from PLAN.md → files section)
4. Ask LLM: "Given this implementation and this reported issue,
             what is the most likely root cause?"
5. Present root cause with file:line where possible
6. Generate a minimal fix plan
7. Append root_cause + fix_plan to UAT.md
```

### 5.5 — Acceptance criteria

- [ ] Loads test cases from PLAN.md acceptance criteria
- [ ] Presents tests one at a time with clear expected behavior
- [ ] User response "pass" / description / "skip" handled correctly
- [ ] Severity inferred from description (not asked explicitly)
- [ ] UAT.md saved after each test (no progress lost on crash)
- [ ] `--resume` continues correctly from the last in-progress test
- [ ] Auto-diagnosis identifies root cause and suggests a fix
- [ ] Fix plan generated and saved
- [ ] `--report` exports a full QA report
- [ ] Activity "qa" logged
- [ ] Build + tests pass

---

## Phase 6 — `coder debug` (Debug Flow)

> **Priority:** P2
> **Effort:** ~4 days
> **Depends on:** Phase 1

### Objective

Debug assistant: receive an error message / stack trace / log → search relevant context → analyze root cause → suggest a fix. Faster than QA — no plan needed, just the error. Equivalent to `debug` + `diagnose-issues` in GSD.

### 6.1 — CLI interface

```sh
# Debug from an error message
coder debug "panic: runtime error: index out of range [3] with length 3"

# From a log file
coder debug --file error.log

# From stdin
cat crash.log | coder debug

# Debug with file context
coder debug --context src/auth/manager.go "nil pointer dereference on line 89"

# Debug git diff (what did I break?)
coder debug --diff HEAD~1

# Verbose: show reasoning steps
coder debug --verbose "connection refused on port 5432"

# Interactive debug session
coder debug --interactive
```

### 6.2 — Debug output

```
$ coder debug "panic: runtime error: nil pointer dereference"

══════════════════════════════════════════════════════════
  DEBUG ANALYSIS
══════════════════════════════════════════════════════════

INPUT
  panic: runtime error: nil pointer dereference
  goroutine 1 [running]:
  github.com/trungtran/coder/internal/usecase/auth.(*Manager).RotateToken(...)
    /internal/usecase/auth/manager.go:189

ROOT CAUSE (confidence: HIGH)
  The RotateToken method calls m.repo.UpdateAccessTokenHash() but
  m.repo is nil because NewManager() was called without a repository.

  This happens when secure_mode=false — the manager is created with
  repo=nil (intentional) but RotateToken doesn't check for this case
  the way other methods do.

  Compare: ValidateToken (line 90) has: if m.repo == nil { ... }
           RotateToken  (line 182) is: MISSING this nil check

LOCATION
  File: internal/usecase/auth/manager.go
  Line: 182–196

SUGGESTED FIX
  Add nil check at the start of RotateToken:

  func (m *Manager) RotateToken(ctx context.Context, clientID string) (string, error) {
    if !m.secureMode {
      return "", fmt.Errorf("server is not in secure mode")
    }
    if m.repo == nil {                    // ← ADD THIS
      return "", fmt.Errorf("auth repository not configured")
    }
    ...
  }

SIMILAR PAST ISSUES
  ● "nil repo check missing in RegenerateBootstrapToken" — fixed 2026-01-15
    (same pattern, same file)

──────────────────────────────────────────────────────────
  Confidence: HIGH · Model: qwen3.5 · Context: 2 memory hits
══════════════════════════════════════════════════════════
```

### 6.3 — Debug system prompt

```
You are a senior software engineer debugging a reported error.

## Project context (from memory)
{memory_context}

## Relevant patterns (from skills)
{skill_context}

## File context (if provided)
{file_content}

## Error to debug:
{error_message}

Analyze and respond with JSON:
{
  "root_cause": "clear explanation of what is wrong and why",
  "location": "file:line if determinable",
  "confidence": "HIGH|MEDIUM|LOW",
  "suggested_fix": "concrete code fix or step-by-step instructions",
  "similar_issues": ["past similar bugs if found in context"],
  "follow_up": "what to check if this fix doesn't work"
}
```

### 6.4 — Interactive debug mode

```sh
$ coder debug --interactive

coder debug — interactive mode
Describe the bug or paste an error message. Type /done when fixed.

You › I'm getting a 401 on /v1/auth/rotate but the token is valid

  ⟳ Searching context...

Assistant › Let me check the auth middleware flow...

  The 401 on /v1/auth/token/rotate suggests the token isn't
  being passed correctly. Check:

  1. Is the Authorization header set? Expected: "Bearer <token>"
  2. Is the token the raw token (not hashed)?
     The middleware hashes it — passing a pre-hashed token will fail.

  Can you share the request you're making?

You › curl -H "Authorization: Bearer abc123" http://localhost:8080/v1/auth/token/rotate

Assistant › The issue is that /v1/auth/token/rotate requires the client
  to be in context (set by auth middleware), but this endpoint
  might not be behind the auth middleware.

  Check: internal/transport/http/server/server.go
  Look for where /v1/auth/token/rotate is registered —
  is it wrapped with authMiddleware?

You › yes that was it! /done

Session saved. Storing fix to memory...
✓ Stored: "token/rotate endpoint missing auth middleware registration"
```

### 6.5 — Acceptance criteria

- [ ] `coder debug "error"` analyzes and returns root cause with confidence level
- [ ] `--file` reads a log file and debugs it
- [ ] `--context file.go` injects file content into the analysis
- [ ] `--diff` debugs from a git diff
- [ ] Output includes: root cause, location, suggested fix, similar past issues
- [ ] `--interactive` REPL supports follow-up questions
- [ ] `/done` in interactive mode → auto-saves lesson to memory
- [ ] Activity "debug" logged
- [ ] Build + tests pass

---

## Phase 7 — `coder session` (State Management)

> **Priority:** P3
> **Effort:** ~3 days
> **Depends on:** Phase 1

### Objective

Save and restore working context — current task, open files, recent decisions, next steps. Solves the context rot problem: AI loses context after a restart. Equivalent to `pause-work` / `resume-work` in GSD.

### 7.1 — CLI interface

```sh
# Save current session
coder session save "implementing JWT refresh tokens — need to add rotation logic"
coder session save  # interactive: prompts for description

# Resume last session
coder session resume

# Resume specific session
coder session resume ses-abc123

# List sessions
coder session list

# Show session detail
coder session show ses-abc123

# Delete session
coder session delete ses-abc123

# Export session as a context file (for pasting into any AI)
coder session export ses-abc123 -o context.md
```

### 7.2 — Session format: `.coder/session.md`

```markdown
---
id: ses-abc123
saved: 2026-03-20T14:30Z
status: active
---

# Session: JWT Refresh Token Implementation

## Current Task

Implementing refresh token rotation — need to add the missing
DeleteRefreshToken call after UpdateAccessTokenHash.

## Next Steps

1. Fix manager.go:182 — add nil check + DeleteRefreshToken call
2. Add integration test for concurrent rotation
3. Run: go test ./internal/usecase/auth/...

## Open Files

- internal/usecase/auth/manager.go (line 182 — main fix)
- internal/infra/postgres/auth.go (need DeleteRefreshToken impl)
- internal/domain/auth/port.go (add DeleteRefreshToken to interface)

## Recent Decisions

- Using optimistic locking for concurrent rotation (not mutex)
- Old refresh token deleted immediately after new one issued
- No grace period for old tokens (security > convenience)

## Context

- Started from: coder qa issue report "old token not invalidated"
- Root cause: missing DeleteRefreshToken in RotateToken flow
- Related PR: #42 (auth refactor) merged 2026-01-15

## Blockers

- None currently
```

### 7.3 — Auto-context inject

When an active session exists, all commands automatically inject session context:

```sh
coder chat "how do I implement DeleteRefreshToken?"
# → Automatically injects session.md context → AI knows what you're working on
# → "Based on your current task implementing token rotation..."
```

### 7.4 — Acceptance criteria

- [ ] `coder session save` creates `.coder/session.md` with all required fields
- [ ] Interactive save: prompts user for current task, next steps, and decisions
- [ ] `coder session resume` displays context and offers to continue
- [ ] `coder session list` displays sessions with a summary
- [ ] Session automatically injected into `coder chat`, `coder debug`, `coder review`
- [ ] `coder session export` creates a file that can be pasted into any AI
- [ ] Build + tests pass

---

## Phase 8 — `coder workflow` (Auto-Chain Orchestration)

> **Priority:** P3
> **Effort:** ~5 days
> **Depends on:** Phase 2, 3, 4, 5, 6

### Objective

Automated chain: plan → review → qa → fix → done. Developer only needs to describe the feature; coder handles the rest. Equivalent to the `autonomous` + `--auto` chain in GSD.

### 8.1 — CLI interface

```sh
# Full auto: plan → implement hints → review → qa
coder workflow "implement Redis caching for skill search"

# Run only plan + review (skip QA)
coder workflow --steps plan,review "refactor auth service"

# Resume an in-progress workflow
coder workflow --resume

# Dry run — show plan only, do not execute
coder workflow --dry-run "add rate limiting"

# From a PRD file
coder workflow --prd path/to/feature.md
```

### 8.2 — Workflow chain

```
coder workflow "feature description"
      │
      ▼
Step 1: PLAN
  coder plan --auto "feature description"
  → .coder/plans/PLAN-{slug}.md
      │
      ▼
Step 2: REVIEW PLAN
  AI self-reviews the plan for completeness
  → highlights risks before implementation
      │
      ▼
Step 3: CHECKPOINT
  Show plan + risks → user approves/adjusts
  [Y to continue / E to edit / Q to quit]
      │
      ▼
Step 4: IMPLEMENT (hints mode)
  Generate implementation checklist for developer
  "Here's what to build, in order, with file references"
  (coder does not write code itself — that is the AI agent's job)
      │
      ▼
Step 5: QA
  coder qa --plan PLAN-{slug}.md
  → user walks through tests
      │
      ▼
Step 6: FIX (if issues)
  coder debug → diagnose → fix plan
  → loop back to QA
      │
      ▼
Step 7: DONE
  Summary: feature name, tests passed, issues resolved
  Activity log entry with full workflow summary
```

### 8.3 — Workflow state file

```yaml
# .coder/workflows/WF-auth-jwt-2026-03-20.yaml
id: wf-abc123
feature: "implement JWT refresh tokens"
status: qa # plan | review | implement | qa | fix | done
created: 2026-03-20T09:00Z
updated: 2026-03-20T14:00Z

steps:
  plan: { status: done, artifact: .coder/plans/PLAN-auth-jwt.md }
  review: { status: done, concerns: 2, approved: true }
  implement: { status: done }
  qa: { status: in_progress, session: qa-abc123 }
  fix: { status: pending }
```

### 8.4 — Acceptance criteria

- [ ] Full chain: plan → review checkpoint → QA works end-to-end
- [ ] Workflow state saved to YAML, resumable after Ctrl+C
- [ ] `--steps` runs only the selected steps
- [ ] Checkpoint: user must approve/edit before proceeding
- [ ] Activity log entry created for the entire workflow
- [ ] `--dry-run` shows plan only, does not execute
- [ ] Build + tests pass

---

## Dashboard Updates

### Phase 2+ — Chat Dashboard Page

Add `/dashboard/chat` page displaying:

- Recent sessions (title, message count, last active)
- Click a session → view conversation history
- Stats: total sessions, total messages, average session length
- Top topics (from session titles, word frequency)

### Phase 3+ — Review History Page

Add `/dashboard/reviews` page displaying:

- Recent reviews (file/PR, concern counts by severity)
- Trend: HIGH concerns over time
- Top recurring issues

---

---

## Phase 9 — `coder new-project` (Project Initialization)

> **Priority:** P0 — Foundation for phases 10–16
> **Effort:** ~4 days
> **Depends on:** Phase 1 (LLM Backbone)
> **GSD equivalent:** `/gsd:new-project`

### Objective

Initialize a new project through a unified flow: questioning → research → requirements → roadmap.
Creates the core document set that all downstream phases reference. This is the most leveraged
moment — deep questioning here means better plans, better execution, better outcomes.

### 9.1 — CLI interface

```sh
coder new-project "build a multi-tenant SaaS with JWT auth"
coder new-project --auto @docs/prd.md   # skip Q&A, extract from PRD
coder new-project --resume              # continue interrupted init
```

### 9.2 — Flow

```
Step 1: Detect context
  - Check if .coder/PROJECT.md already exists → error if so
  - Check git init → offer to run git init if missing
  - Check .coder/codebase/ → prompt to run coder map-codebase first (brownfield)

Step 2: Deep questioning (interactive)
  - Ask until idea is fully understood:
    goals, constraints, tech preferences, must-haves, nice-to-haves
  - Detect feature type to ask domain-appropriate questions:
    API/CLI → auth, versioning, error format, flags
    UI → layout, interactions, empty states, mobile
    Data → schema, access patterns, retention
  - Scope creep guard: "that sounds like v2 — add to backlog?"

Step 3: Research (spawns parallel agents — see Phase 18)
  - Agent 1: stack research (libraries, frameworks, ecosystem)
  - Agent 2: feature research (patterns, edge cases)
  - Agent 3: architecture research (structure, conventions)
  - Agent 4: pitfalls research (known issues, gotchas)
  - Results synthesized into .coder/research/

Step 4: Requirements extraction
  - v1 (must-have) vs v2 (nice-to-have) vs out-of-scope
  - Phase traceability: each requirement linked to roadmap phase
  - User approves requirements list before proceeding

Step 5: Roadmap generation
  - Phases mapped to requirements
  - Dependencies identified (phase N requires phase M)
  - Estimates per phase
  - User approves roadmap

Step 6: Write artifacts
  .coder/PROJECT.md       — vision, goals, tech stack, constraints
  .coder/REQUIREMENTS.md  — v1/v2 requirements with phase links
  .coder/ROADMAP.md       — phases with estimates + dependencies
  .coder/STATE.md         — current phase, decisions, blockers
```

### 9.3 — STATE.md format

```markdown
---
project: "multi-tenant SaaS"
current_phase: 1
updated: 2026-03-23T10:00Z
---

## Current Position

Phase: 1 — Auth foundation
Step: plan
Last action: discuss-phase completed

## Decisions

- JWT with httpOnly cookies (not localStorage)
- PostgreSQL row-level security for tenancy
- No refresh token rotation grace period

## Blockers

- None

## Backlog (deferred ideas)

- OAuth2 social login (v2)
- Multi-region support (v3)
```

### 9.4 — Acceptance criteria

- [ ] Creates all 5 core documents in .coder/
- [ ] `--auto` mode reads PRD and skips interactive Q&A
- [ ] Scope creep detection redirects to backlog (not expands scope)
- [ ] STATE.md auto-updated by all subsequent commands
- [ ] Activity "new-project" logged
- [ ] Build + tests pass

---

## Phase 10 — `coder map-codebase` (Codebase Analysis)

> **Priority:** P0 — Required for brownfield projects
> **Effort:** ~3 days
> **Depends on:** Phase 18 (subagent definitions)
> **GSD equivalent:** `/gsd:map-codebase`

### Objective

Analyze an existing codebase before starting new work. Spawns parallel agents to explore
different aspects. Each agent writes its document directly — orchestrator only collects
confirmations, keeping context minimal.

### 10.1 — CLI interface

```sh
coder map-codebase              # full analysis
coder map-codebase auth         # focus on specific subsystem
coder map-codebase --refresh    # re-analyze after major changes
```

### 10.2 — Parallel agent execution (Claude Code + Copilot)

```
Orchestrator (coder map-codebase)
      │
      ├─ spawn: coder-codebase-mapper --focus=tech
      │         → writes .coder/codebase/STACK.md
      │         → writes .coder/codebase/INTEGRATIONS.md
      │
      ├─ spawn: coder-codebase-mapper --focus=arch
      │         → writes .coder/codebase/ARCHITECTURE.md
      │         → writes .coder/codebase/STRUCTURE.md
      │
      ├─ spawn: coder-codebase-mapper --focus=quality
      │         → writes .coder/codebase/CONVENTIONS.md
      │         → writes .coder/codebase/TESTING.md
      │
      └─ spawn: coder-codebase-mapper --focus=concerns
                → writes .coder/codebase/CONCERNS.md

Wait for all 4 → verify 7 documents exist → commit map
```

**Runtime compatibility:**

- **Claude Code**: `Agent(subagent_type="coder-codebase-mapper", ...)` — true parallel
- **Copilot**: `@coder-codebase-mapper` — parallel with sequential fallback
- **Standalone CLI (no agent runtime)**: sequential inline execution, same output

### 10.3 — Acceptance criteria

- [ ] All 7 codebase documents written to .coder/codebase/
- [ ] Parallel agents complete without context bleed between them
- [ ] `--focus` limits analysis to a subsystem
- [ ] Result committed with message `chore: map codebase`
- [ ] STATE.md updated with codebase map reference
- [ ] Build + tests pass

---

## Phase 11 — `coder discuss-phase` (Context Capture)

> **Priority:** P1
> **Effort:** ~2 days
> **Depends on:** Phase 9
> **GSD equivalent:** `/gsd:discuss-phase N`

### Objective

Separate the "what do you want" conversation from planning. Outputs a CONTEXT.md that
downstream research and planning agents read to know what decisions are locked.
This is what prevents AI from making reasonable-but-wrong assumptions.

### 11.1 — CLI interface

```sh
coder discuss-phase 1              # interactive Q&A for phase 1
coder discuss-phase 1 --auto       # AI picks recommended defaults
coder discuss-phase 1 --batch      # grouped questions, answer in bulk
```

### 11.2 — Flow

```
1. Load prior context
   - PROJECT.md + REQUIREMENTS.md + STATE.md
   - All existing CONTEXT.md files from prior phases
   - Codebase map if available

2. Scout codebase for reusable assets
   - Existing patterns, components, utilities relevant to this phase

3. Analyze phase from ROADMAP.md
   - What's being built? (API / UI / CLI / data / org task)
   - What gray areas exist for this specific type?
   - Skip areas already decided in prior CONTEXT.md files

4. Present gray areas (multi-select)
   - Domain-aware: UI phase → layout/interactions/states
                   API phase → response format/auth/versioning
                   CLI phase → flags/output/error handling
   - User picks which to discuss (3-4 areas max)

5. Deep-dive each selected area
   - 4 focused questions per area
   - Concrete options (numbered), not open-ended
   - "More questions on this, or move on?"

6. Scope creep guard
   - "That's a new capability → adding to backlog, not this phase"

7. Write CONTEXT.md
   .coder/phases/{N}-CONTEXT.md
```

### 11.3 — CONTEXT.md format

```markdown
# Phase 1 Context — Auth Foundation

## Token Storage

Decision: httpOnly cookie (not localStorage)
Rationale: security, XSS protection
Impact: all auth endpoints must set cookie header

## Refresh Token Behavior

Decision: rotate on every use, no grace period
Rationale: security > convenience
Impact: DeleteRefreshToken called in RotateToken flow

## Multi-Device Support

Decision: NOT in v1 — each login invalidates all previous sessions
Deferred to: v2
```

### 11.4 — Acceptance criteria

- [ ] Prior decisions from earlier phases are NOT re-asked
- [ ] Scope creep redirected to backlog, not expanded
- [ ] CONTEXT.md decisions are specific enough for agents to act without re-asking
- [ ] `--auto` picks recommended defaults for all gray areas
- [ ] Build + tests pass

---

## Phase 12 — `coder plan-phase` (Upgraded Planning)

> **Priority:** P1
> **Effort:** ~3 days
> **Depends on:** Phase 11
> **GSD equivalent:** `/gsd:plan-phase N`
> **Upgrades:** existing `coder plan` command

### Objective

Upgrade `coder plan` into a full 3-document pipeline: research → plan → verify loop.
The plan checker catches incomplete plans before a single line of code is written.

### 12.1 — Flow (upgrade to current coder plan)

```
Current: coder plan → Q&A → single LLM call → PLAN.md

Upgraded: coder plan-phase N
  │
  ├─ Step 1: Load context
  │    Read: CONTEXT.md + REQUIREMENTS.md + codebase map
  │
  ├─ Step 2: Research (spawns coder-phase-researcher agent)
  │    Investigates: implementation approaches, library options, pitfalls
  │    Writes: .coder/phases/{N}-RESEARCH.md
  │
  ├─ Step 3: Plan generation (spawns coder-planner agent)
  │    Reads: CONTEXT.md + RESEARCH.md
  │    Generates: 2-4 atomic task plans with XML structure
  │    Each plan fits in a fresh context window (~50-100 lines)
  │    Writes: .coder/phases/{N}-{1,2,3,...}-PLAN.md
  │
  └─ Step 4: Verification loop (spawns coder-plan-checker agent)
       Checks: are plans complete? cover all requirements? no contradictions?
       Loop: max 3 iterations → fail → present to user
       Writes: PLAN.md updated if issues fixed
```

### 12.2 — XML Plan format (upgrade from markdown)

```xml
<plan id="1-01" phase="1" name="JWT token infrastructure">
  <objective>Create the token generation, validation, and rotation foundation</objective>
  <files>
    internal/domain/auth/entity.go
    internal/usecase/auth/manager.go
    internal/infra/postgres/auth.go
  </files>
  <tasks>
    <task type="create">
      <name>Token domain types</name>
      <action>
        Create Token, RefreshToken, Claims structs in entity.go.
        Use jose v4 for JWT (not golang-jwt — license issues).
        Include: sub, iat, exp, jti claims.
      </action>
      <verify>go test ./internal/domain/auth/... passes</verify>
      <done>Token struct created, marshals to valid JWT</done>
    </task>
    <task type="modify">
      <name>Manager.RotateToken</name>
      <action>
        Add nil check for m.repo at line 182.
        Call repo.DeleteRefreshToken after UpdateAccessTokenHash.
      </action>
      <verify>TestRotateToken_InvalidatesOldToken passes</verify>
      <done>Old token deleted, new token returned</done>
    </task>
  </tasks>
  <dependencies>none</dependencies>
  <estimated_time>2h</estimated_time>
</plan>
```

### 12.3 — Flags

```sh
coder plan-phase 1                   # full: research + plan + verify
coder plan-phase 1 --skip-research   # skip if RESEARCH.md exists
coder plan-phase 1 --skip-verify     # skip verification loop
coder plan-phase 1 --gaps            # re-plan only failing items from verify-work
coder plan-phase 1 --prd prd.md      # skip discuss-phase, extract from PRD
```

### 12.4 — Acceptance criteria

- [ ] RESEARCH.md generated before planning
- [ ] Plans use XML format with per-task `<verify>` steps
- [ ] Plan checker catches missing requirements coverage
- [ ] Verification loop max 3 iterations before surfacing to user
- [ ] `--gaps` mode generates fix plans from UAT issues
- [ ] Backward compatible: existing `coder plan` still works unchanged
- [ ] Build + tests pass

---

## Phase 13 — `coder execute-phase` (Subagent Execution)

> **Priority:** P0 — Core differentiator vs simple "generate hints"
> **Effort:** ~6 days
> **Depends on:** Phase 12, Phase 18
> **GSD equivalent:** `/gsd:execute-phase N`

### Objective

Execute PLAN.md files using wave-based parallel subagent execution. Each subagent gets a
fresh context window (zero accumulated garbage) and handles one plan end-to-end.
Orchestrator stays at ~15% context; subagents do the heavy lifting.

### 13.1 — CLI interface

```sh
coder execute-phase 1               # execute all plans, parallel waves
coder execute-phase 1 --interactive # sequential, user checkpoint per plan
coder execute-phase 1 --gaps-only   # only fix plans from verify-work issues
coder execute-phase 1 --plan 1-02   # execute single plan
```

### 13.2 — Wave-based parallel execution

```
Orchestrator reads phase plans → analyzes dependencies → groups into waves

Example: Phase 1 has 4 plans
  Plan 1-01: token domain types   (no deps)  ─┐ WAVE 1 (parallel)
  Plan 1-02: postgres schema       (no deps)  ─┘
  Plan 1-03: auth manager          (needs 1-01, 1-02) ─┐ WAVE 2 (parallel)
  Plan 1-04: HTTP handlers         (needs 1-03)        ─┘ WAVE 3

WAVE 1: spawn coder-executor for plan 1-01 AND 1-02 simultaneously
        wait for both to complete
WAVE 2: spawn coder-executor for plan 1-03
        wait
WAVE 3: spawn coder-executor for plan 1-04
        wait

Each executor:
  1. Reads plan XML (fresh context, no history)
  2. Executes tasks in order
  3. Runs verify step after each task
  4. Commits: git commit -m "feat(1-01): token domain types"
  5. Writes .coder/phases/{N}-{plan}-SUMMARY.md
  6. Returns: done | failed | partial
```

### 13.3 — Runtime compatibility

```
Claude Code (coder.md agent):
  Agent(subagent_type="coder-executor", prompt=plan_context)
  → true parallel via Agent tool
  → orchestrator blocks per wave, not per plan

GitHub Copilot:
  @coder-executor agent reference
  → parallel with sequential fallback if spawning fails

Standalone CLI (no agent runtime):
  Sequential inline execution
  Reads and follows plan XML directly
  Same output, slower

Fallback rule:
  If spawned agent commits are visible + SUMMARY.md exists
  but orchestrator never receives completion signal
  → treat as success, continue to next wave
```

### 13.4 — Per-task atomic commits

```
feat(1-01): create Token and RefreshToken domain types
feat(1-01): add postgres token schema and migration
feat(1-02): implement Manager.RotateToken with nil check
...

Format: {type}({plan-id}): {task name}
Types: feat | fix | refactor | test | docs | chore
```

### 13.5 — Post-execution verification

After all waves complete, orchestrator spawns `coder-verifier`:

```
coder-verifier reads:
  - REQUIREMENTS.md (phase requirements)
  - All SUMMARY.md files from this phase
  - Runs: go test ./... (or equivalent)

Writes: .coder/phases/{N}-VERIFICATION.md
  - Requirements: covered / partial / missing
  - Test results: pass / fail
  - Recommendation: ready for verify-work | needs fix plans
```

### 13.6 — Acceptance criteria

- [ ] Wave dependency analysis correct (parallel vs sequential)
- [ ] Each plan runs in fresh context (no cross-contamination)
- [ ] Atomic git commit after every task
- [ ] SUMMARY.md created per plan
- [ ] VERIFICATION.md checks requirements coverage
- [ ] `--interactive` mode works without subagents (sequential inline)
- [ ] Runtime fallback: if agent spawning unavailable → sequential
- [ ] Activity "execute-phase" logged
- [ ] Build + tests pass

---

## Phase 14 — `coder ship` (PR Creation)

> **Priority:** P1
> **Effort:** ~1 day
> **Depends on:** Phase 13
> **GSD equivalent:** `/gsd:ship N`

### Objective

Bridge local completion → merged PR. After `coder qa` passes, ship the work:
push branch, create PR with auto-generated body from SUMMARY.md files.

### 14.1 — CLI interface

```sh
coder ship 1              # ship phase 1 work
coder ship 1 --draft      # create draft PR
coder ship                # ship current phase (reads STATE.md)
```

### 14.2 — Flow

```
1. Read STATE.md → current phase, phase name
2. Read all .coder/phases/{N}-*-SUMMARY.md files
3. git push --set-upstream origin <branch>
4. gh pr create \
     --title "feat: phase {N} — {phase_name}" \
     --body "<generated from SUMMARY.md>"    \
     --label "phase-{N}"
5. Update STATE.md: current_phase status = "shipped", pr_url = <url>
6. Print: PR URL + next steps
```

### 14.3 — Auto-generated PR body format

```markdown
## Summary

Phase 1 — Auth Foundation

Implements JWT-based authentication with refresh token rotation.

## Changes

- Token domain types (entity.go)
- Postgres schema: tokens + refresh_tokens tables
- Manager.RotateToken with nil guard + DeleteRefreshToken
- HTTP handlers: /v1/auth/login, /v1/auth/refresh, /v1/auth/logout

## Tests

- 12 unit tests added
- 2 integration tests added
- All passing: go test ./...

## Verification

- [x] Login flow works
- [x] Token refresh invalidates old token
- [x] Invalid credentials return 401 AUTH_INVALID_CREDENTIALS

🤖 Generated by coder ship
```

### 14.4 — Acceptance criteria

- [ ] Requires `gh` CLI installed and authenticated
- [ ] PR body generated from SUMMARY.md files
- [ ] STATE.md updated with PR URL
- [ ] `--draft` creates draft PR
- [ ] Build + tests pass

---

## Phase 15 — `coder progress` / `coder next` (Navigation)

> **Priority:** P1
> **Effort:** ~2 days
> **Depends on:** Phase 9
> **GSD equivalent:** `/gsd:progress`, `/gsd:next`

### Objective

State-aware navigation. Developer never needs to remember "where am I?" —
`coder progress` shows it. `coder next` auto-invokes the logical next step.

### 15.1 — CLI interface

```sh
coder progress            # show current state, phase, what's done, what's next
coder next                # auto-detect and run next step
coder next --dry-run      # show what next would do without running it
```

### 15.2 — coder progress output

```
══════════════════════════════════════════════════════════
  PROJECT: multi-tenant SaaS with JWT auth
  Milestone: v1.0 — Auth + Core API
══════════════════════════════════════════════════════════

  ROADMAP
  ✅ Phase 1 — Auth foundation      (shipped: PR #12)
  ▶  Phase 2 — User management      (in progress)
     └─ discuss: ✅  plan: ✅  execute: 🔄 (wave 2/3)  qa: ⬜  ship: ⬜
  ⬜ Phase 3 — Organization tenancy  (not started)
  ⬜ Phase 4 — API rate limiting     (not started)

  CURRENT STEP
  execute-phase 2 — wave 2/3 running

  NEXT STEP (when current completes)
  coder qa --plan .coder/phases/2-*-PLAN.md

  BLOCKERS
  None
══════════════════════════════════════════════════════════
```

### 15.3 — coder next logic

```
Read STATE.md + ROADMAP.md + phase directories

Decision tree:
  No PROJECT.md?             → coder new-project
  No codebase map?           → coder map-codebase (if brownfield)
  Phase N has no CONTEXT.md? → coder discuss-phase N
  Phase N has no PLAN.md?    → coder plan-phase N
  Phase N not executed?      → coder execute-phase N
  Phase N not verified?      → coder qa --phase N
  Phase N not shipped?       → coder ship N
  All phases done?           → coder milestone complete
```

### 15.4 — Acceptance criteria

- [ ] Reads STATE.md + ROADMAP.md accurately
- [ ] Shows per-phase status (discuss/plan/execute/qa/ship)
- [ ] `coder next` invokes the correct command automatically
- [ ] `--dry-run` shows command without running
- [ ] Works even if STATE.md is missing (reconstructs from filesystem)
- [ ] Build + tests pass

---

## Phase 16 — `coder milestone` (Lifecycle Management)

> **Priority:** P2
> **Effort:** ~2 days
> **Depends on:** Phase 15
> **GSD equivalent:** `/gsd:complete-milestone`, `/gsd:new-milestone`, `/gsd:audit-milestone`

### Objective

Milestone lifecycle: audit → complete → archive → start next.
One milestone = one shippable version (e.g. v1.0, v1.1).

### 16.1 — CLI interface

```sh
coder milestone audit          # verify all phases DoD complete
coder milestone complete       # archive current milestone, tag release
coder milestone new "v1.1"     # start next milestone cycle
coder milestone list           # list milestones with status
```

### 16.2 — complete flow

```
1. Run milestone audit (all phases shipped? all PRs merged?)
2. Prompt: squash merge or keep history?
3. git tag v1.0 -m "$(cat .coder/ROADMAP.md | head -20)"
4. Archive: move .coder/phases/ → .coder/milestones/v1.0/
5. Archive: move .coder/ROADMAP.md → .coder/milestones/v1.0/ROADMAP.md
6. Update STATE.md: milestone = "v1.0", status = "complete"
7. Print summary: phases completed, PRs merged, tests passing
```

### 16.3 — Acceptance criteria

- [ ] `audit` checks all phases have: SUMMARY.md + UAT.md + PR merged
- [ ] `complete` creates git tag from milestone name
- [ ] Archives .coder/phases/ cleanly
- [ ] `new` resets for next milestone cycle (keeps PROJECT.md, STATE.md)
- [ ] Build + tests pass

---

## Phase 17 — Utilities (`todo`, `stats`, `health`, `do`, `note`)

> **Priority:** P2
> **Effort:** ~3 days
> **Depends on:** Phase 9
> **GSD equivalent:** various utility commands

### 17.1 — coder todo

```sh
coder todo add "implement rate limiting for /v1/chat"  # capture idea
coder todo list                                         # list pending
coder todo done <id>                                    # mark complete
coder todo promote <id>                                 # promote to phase
```

Saved to `.coder/TODO.md`. Auto-linked to current milestone.

### 17.2 — coder stats

```sh
coder stats
```

Output:

```
PROJECT: multi-tenant SaaS
  Phases:       4 total  |  2 complete  |  1 in-progress  |  1 planned
  Plans:        8 total  |  6 executed  |  2 pending
  Requirements: 12 v1    |  8 complete  |  4 pending
  Tests:        47 total |  47 passing
  PRs merged:   2
  Git commits:  34 (28 feat, 4 fix, 2 chore)
  Todo items:   3 pending
```

### 17.3 — coder health

```sh
coder health          # validate .coder/ integrity
coder health --repair # auto-repair missing/corrupted files
```

Checks:

- STATE.md parseable and consistent with filesystem
- All PLAN.md files referenced in ROADMAP.md exist
- All SUMMARY.md files exist for executed plans
- No orphaned files

### 17.4 — coder do (smart routing)

```sh
coder do "add dark mode toggle to settings"
# → Analyzes intent → routes to: coder quick "add dark mode toggle to settings"

coder do "the login flow is broken"
# → Analyzes intent → routes to: coder debug "the login flow is broken"

coder do "we need rate limiting for the API"
# → Analyzes intent → routes to: coder todo add "rate limiting for API" (backlog)
```

### 17.5 — coder note

```sh
coder note "remember: use optimistic locking not mutex for rotation"
coder note list
coder note promote 3    # promote note to todo
```

Zero-friction capture. Appends to `.coder/NOTES.md`.

### 17.6 — Acceptance criteria

- [ ] `coder todo` CRUD works, saved to TODO.md
- [ ] `coder stats` reads from filesystem (no DB needed)
- [ ] `coder health` detects STATE.md inconsistencies
- [ ] `coder do` correctly routes 80%+ of freeform commands
- [ ] `coder note` appends without disrupting workflow
- [ ] Build + tests pass

---

## Phase 18 — Subagent Definitions (`.claude/agents/` + `.github/agents/`)

> **Priority:** P0 — Required by phases 10, 12, 13
> **Effort:** ~4 days
> **Depends on:** Phase 1 (LLM Backbone)
> **GSD equivalent:** `agents/gsd-*.md` definitions

### Objective

Define specialized subagents that orchestrator commands (map-codebase, plan-phase,
execute-phase) spawn to do focused work in fresh context windows.
Each agent is a markdown file describing role, tools, and behavior.

### 18.1 — Agent definitions to create

| Agent file                                 | Role                                                      | Used by       |
| ------------------------------------------ | --------------------------------------------------------- | ------------- |
| `.claude/agents/coder-executor.md`         | Executes one PLAN.md, commits per task, writes SUMMARY.md | execute-phase |
| `.claude/agents/coder-planner.md`          | Creates XML plan from CONTEXT.md + RESEARCH.md            | plan-phase    |
| `.claude/agents/coder-plan-checker.md`     | Verifies plan covers requirements, no gaps                | plan-phase    |
| `.claude/agents/coder-phase-researcher.md` | Researches implementation approaches for a phase          | plan-phase    |
| `.claude/agents/coder-verifier.md`         | Checks codebase against requirements after execution      | execute-phase |
| `.claude/agents/coder-codebase-mapper.md`  | Analyzes one aspect of codebase, writes structured doc    | map-codebase  |
| `.claude/agents/coder-debugger.md`         | Deep root cause analysis with persistent state            | debug, qa     |

### 18.2 — Agent spawning (runtime-specific)

```
Claude Code:
  Agent(
    subagent_type = "coder-executor",
    prompt = "<plan file content + tools context>",
    isolation = "worktree"   ← optional: isolated git worktree
  )

GitHub Copilot:
  @coder-executor <plan context>
  → fallback: sequential inline if spawning fails

Standalone CLI:
  Sequential inline: agent logic embedded in command
```

### 18.3 — coder-executor agent design

```markdown
---
name: coder-executor
description: Execute one PLAN.md — read tasks, implement in order,
  commit after each task, write SUMMARY.md when done.
tools: Read, Write, Edit, Bash, Glob, Grep
---

## Role

You are a focused implementer. You receive exactly one PLAN.md and
execute it completely. You do not ask questions — decisions are in the plan.

## Process

1. Read the full plan XML
2. For each <task>:
   a. Read referenced files (from <files>)
   b. Implement changes
   c. Run <verify> step — fix if failing
   d. git commit -m "{type}({plan-id}): {task name}"
3. Write SUMMARY.md when all tasks complete
4. Return: done | failed (with reason)

## Rules

- Never expand scope beyond the plan
- If a <verify> fails twice, write failure to SUMMARY.md and stop
- Commit message format: feat|fix|refactor|test({plan_id}): {name}
- Never skip a task silently — either do it or document why
```

### 18.4 — Acceptance criteria

- [ ] All 7 agent .md files created with correct frontmatter
- [ ] Each agent tested manually via `/coder-executor` (Claude Code)
- [ ] Orchestrator correctly spawns agents and collects results
- [ ] Sequential fallback works when Task tool unavailable
- [ ] Agent isolation: no context bleed between parallel agents
- [ ] Build + tests pass

---

## Implementation Priority Summary

| Phase  | Feature                                   | Priority | Effort | Status  |
| ------ | ----------------------------------------- | -------- | ------ | ------- |
| 1      | LLM Backbone                              | P0       | 5d     | ✅ Done |
| 2      | coder chat                                | P1       | 3d     | ✅ Done |
| 3      | coder review                              | P1       | 4d     | ✅ Done |
| 4      | coder plan                                | P2       | 6d     | ✅ Done |
| 5      | coder qa                                  | P2       | 5d     | ✅ Done |
| 6      | coder debug                               | P2       | 4d     | ✅ Done |
| 7      | coder session                             | P3       | 3d     | ✅ Done |
| 8      | coder workflow                            | P3       | 5d     | ✅ Done |
| **9**  | **coder new-project**                     | **P0**   | **4d** | 🔲      |
| **10** | **coder map-codebase**                    | **P0**   | **3d** | 🔲      |
| **11** | **coder discuss-phase**                   | **P1**   | **2d** | 🔲      |
| **12** | **coder plan-phase (upgrade)**            | **P1**   | **3d** | 🔲      |
| **13** | **coder execute-phase**                   | **P0**   | **6d** | 🔲      |
| **14** | **coder ship**                            | **P1**   | **1d** | 🔲      |
| **15** | **coder progress / next**                 | **P1**   | **2d** | 🔲      |
| **16** | **coder milestone**                       | **P2**   | **2d** | 🔲      |
| **17** | **Utilities (todo/stats/health/do/note)** | **P2**   | **3d** | 🔲      |
| **18** | **Subagent definitions**                  | **P0**   | **4d** | 🔲      |

**Total remaining effort:** ~30 engineer-days

**Recommended execution order (next sprint):**

1. Phase 18 (subagent definitions) — unblocks 10, 12, 13
2. Phase 9 (new-project + STATE.md) — unblocks everything else
3. Phase 10 + 11 in parallel (independent after 9 + 18)
4. Phase 12 (upgrades coder plan) — needs 11 + 18
5. Phase 13 (execute-phase) — needs 12 + 18 — **the core differentiator**
6. Phase 14 + 15 in parallel (ship + progress/next)
7. Phase 16 + 17 last (milestone + utilities)

---

## Key design principles (from GSD lessons)

1. **Context engineering over prompt engineering** — Automatically inject the right context instead of asking the user to provide it
2. **State persistence** — Every workflow has a state file, resumable after a crash
3. **Concrete options, not open questions** — Always offer numbered options instead of blank input
4. **Scope creep detection** — Recognize when the user is describing a new feature → defer, do not expand
5. **Severity inference** — Infer severity from language, never ask "how bad is this?"
6. **Verification loops** — Plan → check → revise before presenting to the user
7. **Activity logging** — Every command is logged for dashboard tracking
8. **Fail gracefully** — LLM unavailable, network error → clear, actionable error messages
9. **Orchestrator stays lean** — Spawns agents, collects results; never does heavy lifting itself (15% context budget)
10. **Fresh context per task** — Each subagent gets 200k clean tokens; no accumulated garbage from prior work
11. **Runtime-aware** — Agent spawning adapts: Claude Code `Agent` tool → Copilot `@agent` → sequential inline fallback
12. **Atomic commits** — Every task gets its own commit immediately; clean bisectable history
