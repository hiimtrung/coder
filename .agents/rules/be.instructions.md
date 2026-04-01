---
applyTo: "**/*.{ts,js,java,go,py,rs,c,dart}"
---

# Backend Development Instructions

## Language-to-Skill Mapping

Before working in any language, run `coder skill search` with the relevant skill:

| Language / Framework  | Skill to search                                                          |
| --------------------- | ------------------------------------------------------------------------ |
| TypeScript (NestJS)   | `nestjs`                                                                 |
| Java (Spring/Quarkus) | `java`                                                                   |
| Go                    | `golang`                                                                 |
| Python                | `python`                                                                 |
| Rust                  | `rust`                                                                   |
| C                     | `c`                                                                      |
| Dart                  | `dart`                                                                   |
| Any backend           | `architecture`, `database`, `development`, `general-patterns`, `testing` |

## Knowledge Gates — MANDATORY

Every backend task starts and ends with memory gates:

```bash
# GATE 1 (START) — before any code
coder skill resolve "<language or topic>" --trigger initial --budget 3
coder memory search "<feature or module name>"

# GATE 2 (END) — after completing work
coder memory store "<Title>" "<What, why, which files>" --tags "<project,language,topic>"
```

Dynamic retrieval is mandatory:

- Re-run `coder skill resolve` with a more precise query after clarification, before switching phase, when a new language/framework appears, after repeated errors, and before review/release.
- Use `coder memory recall "<topic>"` to narrow the active memory working set and `coder memory active` to inspect what is currently pinned for the task.

## API Design Standards

### REST APIs

- Use resource-oriented URLs: `/api/v1/users/{id}/orders`
- HTTP methods: GET (read), POST (create), PUT (full update), PATCH (partial update), DELETE
- Return appropriate status codes: 200, 201, 400, 401, 403, 404, 409, 500
- Always version APIs: `/api/v1/...`
- Paginate list endpoints: `{ data: [], total, page, limit }`

### gRPC Services

- Define proto files in `proto/` directory
- Use `snake_case` for field names in proto definitions
- Version your packages: `package myservice.v1;`
- Always include request/response message wrappers (not raw types)

### Error Response Format

```json
{
  "error": {
    "code": "VAL_INVALID_EMAIL",
    "message": "Email format is invalid",
    "field": "email",
    "action": "Provide a valid email address"
  }
}
```

## Database Patterns

### Repository Pattern

- All database access goes through repository interfaces
- Never query the DB directly from use cases or controllers
- Repositories return domain objects, not raw DB rows
- Use transactions at the use case level, not repository level

### Migrations

- Use sequential numbered migrations: `001_create_users.sql`, `002_add_email_index.sql`
- Never modify existing migrations — always create new ones
- Include both UP and DOWN migration scripts
- Test migrations on staging before production

### Multi-Database Orchestration

- PostgreSQL: primary relational data, transactions, reporting
- MongoDB: document storage, flexible schemas, event logs
- Redis: caching, sessions, pub/sub, rate limiting
- Each database has its own repository interface

### Query Guidelines

- Use parameterized queries — never string concatenation
- Add indexes for all foreign keys and frequent filter columns
- Use `EXPLAIN ANALYZE` to verify query plans for critical paths
- Set query timeouts for all external-facing endpoints

## Multi-Tenancy Patterns

### Company ID in Every Query

Every data query MUST include `company_id` filter:

```typescript
// ✅ Correct
await this.repo.find({ where: { company_id: ctx.companyId, id: userId } });

// ❌ Wrong — missing company_id
await this.repo.find({ where: { id: userId } });
```

### Tenant Context Propagation

- Extract tenant context from JWT token in auth middleware
- Pass tenant context through request context / DI container
- Never derive tenant from request body (security risk)

### Data Isolation Enforcement

- Row-level security in PostgreSQL where possible
- Application-level company_id checks as defense-in-depth
- Audit logs must include company_id for all mutations

## Error Code Standards

Use standardized error codes across all services:

| Prefix   | HTTP Status   | Category             | Example                  |
| -------- | ------------- | -------------------- | ------------------------ |
| AUTH\_\* | 401, 403      | Authentication/Authz | AUTH_TOKEN_EXPIRED       |
| VAL\_\*  | 400           | Input validation     | VAL_INVALID_EMAIL        |
| BIZ\_\*  | 400, 404, 409 | Business logic       | BIZ_USER_NOT_FOUND       |
| INF\_\*  | 500, 502, 503 | Infrastructure       | INF_DB_CONNECTION_FAILED |
| SYS\_\*  | 500           | System/Configuration | SYS_CONFIG_MISSING       |

### Error Structure

```typescript
throw new AppError({
  code: "BIZ_USER_NOT_FOUND",
  message: "User with this ID does not exist",
  httpStatus: 404,
  action: "Verify the user ID and try again",
});
```

## Event-Driven Architecture Rules

### Event Publishing

- Publish domain events AFTER successful transaction commit
- Use outbox pattern for guaranteed delivery
- Event names: past tense, `user.created`, `order.shipped`
- Include `aggregate_id`, `company_id`, `timestamp`, `version` in all events

### Event Consuming

- Idempotent handlers — processing same event twice must be safe
- Dead letter queue for failed events after 3 retries
- Log all consumed events with correlation ID

### Module Communication

- Modules communicate ONLY via events or well-defined service interfaces
- No direct repository access across module boundaries
- Cross-module queries go through the owning module's service

## Clean Architecture Layers

```
Controller/Handler (HTTP/gRPC)
    ↓ DTOs
Use Case / Application Service
    ↓ Domain interfaces
Domain (entities, value objects, domain services)
    ↑ implements
Infrastructure (repositories, external APIs, DB)
```

- Dependencies point INWARD only
- Domain layer has zero framework dependencies
- Use cases orchestrate domain objects and infrastructure interfaces

## Testing Standards

- Unit test all business logic (use cases, domain services)
- Integration test all repository implementations
- E2E test all API endpoints with realistic data
- Test coverage minimum: 80% for use cases, 70% overall
- Run `coder skill resolve "testing" --trigger execution --budget 3` for project-specific test patterns

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

## 🏢 Professional Delivery Pipeline

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

**Last Updated**: March 2026
**System**: AI-Agents Backend Development Guidance
**Status**: Production Ready
