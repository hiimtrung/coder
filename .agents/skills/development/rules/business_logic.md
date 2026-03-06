---
name: business-logic-patterns
description: Guidelines for Use Case execution, Event-Driven Architecture, and Multi-Tenancy
applyTo: "**/application/**/*, **/domain/**/*"
---

# Business Logic & Patterns

## 1. Atomic Use Cases

### Execution Flow
Every Use Case must follow a standardized pipeline:
1. **Context Extraction**: Retrieve identity and tenant (CompanyId) from context.
2. **Authorization**: Verify permissions for the specific action.
3. **Input Validation**: Check DTOs; throw `ValidationException` (e.g., `VAL_MISSING_FIELD`).
4. **Data Retrieval**: Fetch required Domain Entities via Repositories.
5. **Business Rule Validation**: Call domain methods; throw domain-specific exceptions (e.g., `BIZ_ENTITY_NOT_FOUND`).
6. **Execution/Transformation**: Logic processing, calling external services (wrapped in try-catch).
7. **Persistence**: Save modified entities.
8. **Side Effects**: Publish domain events AFTER successful persistence.

### Constraints
- **Exception-Based**: Use domain exceptions for control flow. **NEVER use "Result Object" patterns** to return success/failure.
- **Independent**: One use case should not directly call another use case.
- **No Side Effects in Domain**: Entities record events; Application layer publishes them.

## 2. Type Safety & Strict Typing (MANDATORY)

- ❌ **NEVER use `any`** or `as any`.
- ✅ **Explicit Typing**: All variables, parameters, and return values must be explicitly typed.
- ✅ **Typed API Responses**: Always define interfaces for external API payloads.
- ✅ **Custom Type Guards**: Use type guards for safe casting of `unknown` types.

## 3. External API Integration Patterns

- **Isolation**: Centralize external API logic in dedicated `Clients` in the Infrastructure layer.
- **Wrapping**: Use cases should call these clients and wrap any infrastructure errors into domain exceptions (`INF_EXTERNAL_API_ERROR`).
- **Data Mapping**: Convert external vendor types to your internal Domain Entities immediately.

---

## 4. Multi-Tenancy Strategy (Dual-ID)

- **External Identity**: Use opaque IDs (number/long) for API routes/JWTs.
- **Internal Identity**: Use UUIDs for database primary keys.
- **Isolation**: Every query MUST be scoped by `company_id`.

## 5. Anti-Patterns
- ❌ Putting business logic into Database Stored Procedures.
- ❌ Using "Result Object" patterns to hide system exceptions.
- ❌ Cross-module transaction coupling.
- ❌ Mixing External IDs into the Domain Logic layer.
- ❌ Using `any` or skipping type definitions for speed.
