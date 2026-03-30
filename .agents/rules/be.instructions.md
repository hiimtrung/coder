---
applyTo: "**/*.{ts,js,java,go,py,rs,c,dart}"
---

# Backend Development Instructions

## Language-to-Skill Mapping

Before working in any language, run `coder skill search` with the relevant skill:

| Language / Framework | Skill to search |
| -------------------- | --------------- |
| TypeScript (NestJS)  | `nestjs`        |
| Java (Spring/Quarkus)| `java`          |
| Go                   | `golang`        |
| Python               | `python`        |
| Rust                 | `rust`          |
| C                    | `c`             |
| Dart                 | `dart`          |
| Any backend          | `architecture`, `database`, `development`, `general-patterns`, `testing` |

## Knowledge Gates — MANDATORY

Every backend task starts and ends with memory gates:

```bash
# GATE 1 (START) — before any code
coder skill search "<language or topic>"
coder memory search "<feature or module name>"

# GATE 2 (END) — after completing work
coder memory store "<Title>" "<What, why, which files>" --tags "<project,language,topic>"
```

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

| Prefix | HTTP Status | Category              | Example                  |
| ------ | ----------- | --------------------- | ------------------------ |
| AUTH_* | 401, 403    | Authentication/Authz  | AUTH_TOKEN_EXPIRED       |
| VAL_*  | 400         | Input validation      | VAL_INVALID_EMAIL        |
| BIZ_*  | 400, 404, 409 | Business logic      | BIZ_USER_NOT_FOUND       |
| INF_*  | 500, 502, 503 | Infrastructure      | INF_DB_CONNECTION_FAILED |
| SYS_*  | 500         | System/Configuration  | SYS_CONFIG_MISSING       |

### Error Structure

```typescript
throw new AppError({
  code: 'BIZ_USER_NOT_FOUND',
  message: 'User with this ID does not exist',
  httpStatus: 404,
  action: 'Verify the user ID and try again',
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
- Run `coder skill search "testing"` for project-specific test patterns
