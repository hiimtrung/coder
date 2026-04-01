---
name: general-instructions
description: Universal rules for all AI agents — architecture standards, knowledge gates, error handling, and professional delivery process.
applyTo: "**/*"
---

# General Instructions — AI Agent Development System

All projects follow **Clean Architecture + Event-Driven Design** with standardized error codes, semantic memory, and a professional multi-role delivery pipeline.

---

## 🏗️ System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  YOUR AI AGENT (Claude / Copilot)                           │
│  All reasoning, planning, code generation — done HERE       │
│  No LLM configuration needed on coder-node                  │
└────────────────────┬────────────────────────────────────────┘
                     │ coder CLI (gRPC)
                     ▼
┌─────────────────────────────────────────────────────────────┐
│  coder-node  (memory + skill + semantic search service)     │
│  ├── PostgreSQL + pgvector  (vector(1024) storage)          │
│  └── Ollama mxbai-embed-large  (embedding only, dim 1024)   │
│      ↳ Converts text → vectors for semantic similarity      │
│      ↳ NO chat LLM — Ollama is embedding-only here         │
└─────────────────────────────────────────────────────────────┘
```

**Embedding provider** (`EMBEDDING_PROVIDER`):
| Value | Description |
|-------|-------------|
| `ollama` (default) | Local Ollama `mxbai-embed-large` — best for air-gapped / offline |
| `openai` | OpenAI `text-embedding-3-small` — requires `EMBEDDING_API_KEY` |
| `none` | FTS-only mode (keyword search, no vector similarity) |

---

## 🔐 Knowledge Gates — MANDATORY for every task

```
┌──────────────────────────────────────────────────────────────┐
│  GATE 1 (START) — Before touching any code                   │
│  coder skill resolve "<topic>" --trigger initial --budget 3                                │
│  coder memory search "<topic>"                               │
├──────────────────────────────────────────────────────────────┤
│  EXECUTE (implement, test, commit)                           │
├──────────────────────────────────────────────────────────────┤
│  GATE 2 (END) — After completing any non-trivial work        │
│  coder memory store "<Title>" "<Content>" --tags "<tags>"    │
└──────────────────────────────────────────────────────────────┘
```

Both gates are **blocking**. Skipping either is a workflow violation.

Dynamic retrieval is mandatory:

- Run `coder skill resolve "<topic>" --trigger initial --budget 3` at task start.
- Re-run `coder skill resolve` with a more precise query after clarification, before switching phase, when a new language/framework appears, after repeated errors, and before review/release.
- Use `coder skill resolve "<topic>" --trigger execution --budget 3 --format raw` when you need markdown-preserving skill context for prompt injection.
- Inspect the current session skill set with `coder skill active --format json`.
- Use `coder memory recall "<topic>"` to narrow the active memory working set and `coder memory active` to inspect what is currently pinned for the task.
- Treat `.coder/active-skills.json` as the current active-skill record for the session.
- Treat `.coder/context-state.json` as the combined local snapshot of active skills and active memory.

**Todo list rule** — every task list MUST be:

```
1. [GATE 1] coder skill resolve "<topic>" --trigger initial --budget 3
2. [GATE 1] coder memory search "<topic>"
   ... implementation tasks ...
N. [GATE 2] coder memory store "<title>"
```

---

## 🏢 Professional Delivery Pipeline

Every feature follows a **multi-role pipeline** that mirrors a professional tech company:

```
BA (clarify-requirements)
  ↓  confirmed requirements doc
Architect (architecture-design)
  ↓  ADR + system design
Backend Dev (implement-feature)
  ↓  implementation + unit tests
Frontend Dev (implement-feature)
  ↓  UI + integration
Code Reviewer (code-review)
  ↓  review checklist
QA Engineer (qa-test)
  ↓  test plan + execution
Tech Writer (write-documentation / technical-writer-review)
  ↓  updated docs
Release (release-readiness)
  ↓  production-ready artifact
```

Available workflow slash commands:

- `/clarify-requirements` — BA phase: ask questions → write requirements doc
- `/architecture-design` — Architect phase: ADR + design decisions
- `/implement-feature` — Dev phase: implement + unit tests
- `/code-review` — Review phase: structured code review checklist
- `/qa-test` — QA phase: test plan + execution report
- `/write-documentation` — Tech Writer: generate or update docs
- `/technical-writer-review` — Review existing docs for quality
- `/debug-issue` — Root cause analysis + fix plan
- `/debug-leak` — Memory / resource leak investigation
- `/writing-test` — Generate test cases and test suites
- `/check-implementation` — Verify implementation matches requirements
- `/review-design` — Review UI/UX design decisions
- `/review-requirements` — BA review of requirements completeness
- `/simplify-implementation` — Refactor for clarity/maintainability
- `/release-readiness` — Pre-release checklist
- `/knowledge-capture` — Manually capture patterns and decisions

---

## 🏗️ Clean Architecture (All Languages)

```
Presentation Layer  (Controllers / Handlers / gRPC)
    ↓ DTOs
Application Layer   (Use Cases / Application Services)
    ↓ Domain interfaces
Domain Layer        (Entities, Value Objects, Domain Events, Exceptions)
    ↑ implements
Infrastructure Layer (Repositories, External APIs, DB adapters)
```

**Rules:**

- Dependencies point **inward only** — infrastructure depends on domain, never the reverse
- Domain layer has **zero framework dependencies**
- Use cases orchestrate domain objects through infrastructure interfaces
- Cross-module communication: **events only**, never direct repository calls
- Multi-tenancy: `company_id` on **every** query, extracted from JWT — never from request body

---

## ⚠️ Error Code Standard (ALL Languages)

**Format**: `{CATEGORY}_{DESCRIPTIVE_NAME}`

| Prefix   | HTTP Status   | Category                       | Example                    |
| -------- | ------------- | ------------------------------ | -------------------------- |
| `AUTH_*` | 401, 403      | Authentication / Authorization | `AUTH_TOKEN_EXPIRED`       |
| `VAL_*`  | 400           | Input validation               | `VAL_INVALID_EMAIL`        |
| `BIZ_*`  | 400, 404, 409 | Business logic                 | `BIZ_USER_NOT_FOUND`       |
| `INF_*`  | 500, 502, 503 | Infrastructure / External      | `INF_DB_CONNECTION_FAILED` |
| `SYS_*`  | 500           | System / Configuration         | `SYS_CONFIG_MISSING`       |

**Error response shape** (consistent across all APIs):

```json
{
  "error": {
    "code": "BIZ_USER_NOT_FOUND",
    "message": "User with this ID does not exist",
    "action": "Verify the user ID and try again"
  }
}
```

---

## ⚠️ Critical Rules (ALL Languages)

1. **Type Safety** — NO `any` types; explicit typing everywhere
2. **Error Handling** — Throw domain exceptions using the error code standard above
3. **Multi-Tenancy** — ALWAYS include `company_id` in every data query
4. **Events** — Publish domain events AFTER successful persistence, never before
5. **Knowledge Gates** — Both gates are blocking (see above)
6. **Build** — Run `lint → build → test` before every commit
7. **Package Manager** — `yarn` for NestJS; `gradle` for Java; never `npm` on Node projects

---

## 🛠️ Available coder CLI Commands

```bash
# Memory — semantic storage and retrieval
coder memory search "<query>"
coder memory recall "<query>"
coder memory active
coder memory store "<title>" "<content>" --tags "<tag1,tag2>"
coder memory list
coder memory compact --revector

# Skills — knowledge base retrieval
coder skill resolve "<topic>" --trigger initial --budget 3
coder skill resolve "<topic>" --trigger execution --budget 3 --format raw
coder skill active --format json
coder skill search "<topic>" --format json
coder skill list
coder skill info <name> --format raw

# Session — checkpointing
coder session save
coder progress
coder next

# Project lifecycle
coder install [profile]        # install rules + workflows + agent files
coder login                    # authenticate with coder-node
coder token                    # manage API tokens
coder milestone complete N     # mark milestone done
coder version                  # show version
```

**DO NOT call**: `coder chat`, `coder debug`, `coder review`, `coder qa`, `coder workflow`, `coder plan-phase`, `coder execute-phase` — these have been removed. All reasoning is handled by your AI agent (Claude / Copilot).

## 🤖 Subagents And `.coder`

- When handing a bounded task to a subagent, the subagent must run its own `coder skill resolve` for that subtask instead of inheriting stale skills blindly.
- Subagents must update the task file or checkpoint they own under `.coder/` before handing control back.
- Phase, plan, run status, and task ownership live in `.coder/`; do not treat them as optional notes.

---

## 📋 Language-Specific Rules

| Language            | Skill Reference | Primary Projects                               |
| ------------------- | --------------- | ---------------------------------------------- |
| TypeScript (NestJS) | `nestjs`        | omi-channel-be, findtourgoUI, packageTourAdmin |
| Java (Spring Boot)  | `java`          | crm_be, packageTourApi                         |
| Go                  | `golang`        | Future services                                |
| Rust                | `rust`          | Future services                                |
| Python              | `python`        | Scripts / utilities                            |
| Dart                | `dart`          | Mobile / Flutter                               |
| C                   | `c`             | Embedded / system                              |

**Architecture Skills:**

| Topic                      | Skill              |
| -------------------------- | ------------------ |
| Clean Architecture, DDD    | `architecture`     |
| Use cases, domain events   | `development`      |
| Repositories, integrations | `database`         |
| Error codes, exceptions    | `general-patterns` |
| UI/UX design               | `ui-ux-pro-max`    |

---

## 🔄 Cross-Module Communication

```typescript
// ✅ RIGHT — event-driven
await this.repository.save(entity);
await this.eventPublisher.publish(new UserCreatedEvent(...));

// ❌ WRONG — cross-module repository access
private readonly orderRepository: IOrderRepository; // forbidden in UserModule
```

---

**Last Updated**: March 2026
**System**: AI-Agents Unified Development Guidance
**Status**: Production Ready
