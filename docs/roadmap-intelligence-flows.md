# Coder Intelligence Flows — Roadmap

> **Inspired by:** [get-shit-done](https://github.com/glittercowboy/get-shit-done) — context engineering + spec-driven development system
> **Goal:** Evolve coder from a RAG/memory CLI into a full **AI development workflow engine** — with Q&A, review, planning, QA, and debug capabilities on par with a senior engineer AI pair.
> **Last updated:** 2026-03-20

---

## Architecture Overview

```
Developer / AI Agent
      │
      │  coder CLI
      │  ├─ coder chat          ← Q&A with context injection   (Phase 2)
      │  ├─ coder review        ← Multi-model code review      (Phase 3)
      │  ├─ coder plan          ← Planning workflow            (Phase 4)
      │  ├─ coder qa            ← QA / UAT verification        (Phase 5)
      │  ├─ coder debug         ← Root cause diagnosis         (Phase 6)
      │  ├─ coder session       ← State management             (Phase 7)
      │  └─ coder workflow      ← Auto-chain orchestration     (Phase 8)
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
  "model":  "llama3.2:latest",
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
    "model":          "llama3.2:latest",
    "stream":         true,
    "inject_memory":  true,
    "inject_skills":  true,
    "memory_limit":   5,
    "skill_limit":    3,
    "history_limit":  20
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

| Command | Action |
|---|---|
| `/help` | Show available commands |
| `/sessions` | List recent sessions |
| `/resume <id>` | Load a session |
| `/clear` | Clear conversation history (keep session ID) |
| `/context` | Show currently injected context |
| `/model <name>` | Switch model for this session |
| `/save <note>` | Save session with a custom title |
| `/exit` or Ctrl+C | Exit and auto-save session |

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
  Model: llama3.2:latest · Context: auth patterns (memory)
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

  ⟳ Reviewing with llama3.2:latest...  done
  ⟳ Reviewing with gemma2:9b...        done

Synthesizing consensus...

══════════════════════════════════════════
  MULTI-MODEL REVIEW CONSENSUS
══════════════════════════════════════════

AGREED CONCERNS (raised by 2+ models)
  ● [HIGH] Token refresh expiry not checked  ← llama3.2 + gemma2

UNIQUE CONCERNS — llama3.2 only
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
status: in_progress    # new | in_progress | complete
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
  Confidence: HIGH · Model: llama3.2 · Context: 2 memory hits
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
status: qa         # plan | review | implement | qa | fix | done
created: 2026-03-20T09:00Z
updated: 2026-03-20T14:00Z

steps:
  plan:      { status: done, artifact: .coder/plans/PLAN-auth-jwt.md }
  review:    { status: done, concerns: 2, approved: true }
  implement: { status: done }
  qa:        { status: in_progress, session: qa-abc123 }
  fix:       { status: pending }
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

## Implementation Priority Summary

| Phase | Feature | Priority | Effort | Value |
|-------|---------|----------|--------|-------|
| 1 | LLM Backbone | P0 | 5 days | Foundation |
| 2 | coder chat | P1 | 3 days | High — daily use |
| 3 | coder review | P1 | 4 days | High — code quality |
| 4 | coder plan | P2 | 6 days | High — workflow |
| 5 | coder qa | P2 | 5 days | High — quality gate |
| 6 | coder debug | P2 | 4 days | High — debugging |
| 7 | coder session | P3 | 3 days | Medium — UX |
| 8 | coder workflow | P3 | 5 days | High — automation |

**Total estimated effort:** ~35 engineer-days

**Recommended execution order:**
1. Phase 1 (blocker for everything else)
2. Phase 2 + 3 in parallel (independent after Phase 1)
3. Phase 4 + 6 in parallel
4. Phase 5 (depends on 4)
5. Phase 7 + 8 last

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
