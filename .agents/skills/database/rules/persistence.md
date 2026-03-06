---
name: infrastructure-integration
description: Infrastructure implementation rules, stability patterns, and data mapping
applyTo: "**/infrastructure/**/*, **/adapters/**/*"
---

# Infrastructure & Integration Rules

## 1. Interface Implementation (Repositories)

Infrastructure is the "slave" of the Domain. It implements the interfaces defined in the Domain layer.

### Rules

- **Encapsulation**: Never expose database-specific types (e.g., TypeORM MongoEntityManager, SQL ResultSets) to the outside world.
- **Transactional Integrity**: Handle transactions at the Application layer using a `UnitOfWork` or infrastructure-provided abstraction, but the _intent_ is defined by the use case.
- **Fail-Fast**: Validate configuration (API URLs, keys) on startup, not when the first request arrives.

## 2. Mapping Layer (Anti-Corruption)

Mappers are mandatory to prevent "Persistence Leakage".

- **Persistence Mapper**: `Domain Entity ↔ Persistence Database Entity`.
- **API Mapper**: `External API Payload ↔ Domain Request/Event`.
- **Response Mapper**: `Domain Entity 1 → Partial DTO`.

### Constraints

- Mappers should be stateless and pure.
- Avoid "Automated Mappers" (like AutoMapper) if they hide complex transformations or break type safety.

## 3. External Service Resilience

Integrations with 3rd party APIs (Facebook, LINE, Stripe) must be resilient.

- **Wrapping**: Centralize 3rd party SDKs behind your own "Clients".
- **Timeout/Retry**: Every external call must have a timeout and a defined retry strategy (Exponential Backoff).
- **Circuit Breaker**: Use patterns to stop calling failing services.
- **Standardized Errors**: Translate 3rd party error codes (e.g., FB Error 190) into internal **DomainExceptions**.

## 4. Database Persistence Patterns

**Importance**: 🔴 CRITICAL

### Naming Conventions:
| Object           | Convention                   | Example                                |
| ---------------- | ---------------------------- | -------------------------------------- |
| **Table names**  | Plural, snake_case           | `channels`, `channel_contacts`         |
| **Column names** | camelCase                    | `createdAt`, `externalId`              |
| **Entity files** | `{entity}.orm-entity.ts`     | `channel.orm-entity.ts`                |
| **Migrations**   | `{timestamp}-{Description}.ts` | `1708800000000-CreateChannelsTable.ts` |

### Migration Workflow (MANDATORY):
1. **Never use `synchronize: true`** in production or staging.
2. Update the ORM entity file (`.orm-entity.ts`).
3. Generate the migration using the tool (e.g., `yarn migration:generate`).
4. Review the generated SQL manually before applying.
5. Apply migrations using `migration:run`.

### Multi-Database Strategy:
- **PostgreSQL**: Metadata, ACID transactions, relationships, company/channel config.
- **MongoDB**: High-volume, append-only data like messages or audit logs.
- **Redis**: Ephemeral state, sessions, presence, and rate limiting.

---

## 5. Generic Persistence Guidelines

- **Soft Deletes**: Use `deleted_at` instead of hard deletes where audit trails are required.
- **Versioning**: Use optimistic locking (`version` column) for high-concurrency entities.
- **Audit Logging**: Mandatory `created_at` and `updated_at` on every table/collection.

## 6. Security & Stability

- **Parameterized Queries**: No string concatenation for queries.
- **Sensitive Data**: Encryption at rest for secrets (AES-256 for provider tokens).
- **Connection Pool**: Keep transactions short (< 1s). Avoid external API calls inside DB transactions.
- **Fail-Fast**: Validate configuration (API URLs, keys) on startup.
