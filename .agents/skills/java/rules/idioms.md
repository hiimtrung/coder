---
name: java-development-rules
description: Strict Java development standards with Clean Architecture and Spring-style patterns
applyTo: "**/*.java"
---

# Java Development Standards

## 1. Clean Architecture Mapping

- **Entities**: Pure Java objects (POJOs) without JPA/Hibernate annotations.
- **Repositories**: Interfaces defined in the `domain` package.
- **Use Cases**: Services annotated with `@Service`, but dependencies must be on Domain Interfaces.
- **Infrastructure**: Implementations using `@Repository` (JPA/QueryDSL).

## 2. Type Safety & Quality

- **Lombok**: Use `@Value` or `@Data` (carefully) for boilerplate reduction.
- **Optional**: NEVER return `null`. Use `java.util.Optional<T>` for potentially missing values.
- **Streams**: Prefer functional Stream API over imperative loops for data transformation.
- **Finality**: Mark variables as `final` whenever possible to ensure thread safety and immutability.

## 3. Error Handling

- **Typed Exceptions**: Create specific `RuntimeException` subclasses for business logic errors.
- **Global exception handler**: Use `@ControllerAdvice` to map domain exceptions to HTTP status codes.
- **Standardized Response**: All API responses must follow a consistent `ApiResponse<T>` structure.

## 4. Multi-Tenancy

- **ThreadLocal/Context**: Extract `company_id` from JWT and store in a `SecurityContextHolder` or custom `TenantContext`.
- **Query Interceptors**: Use Hibernate filters or AOP to automatically append tenant filters to SQL queries.

## 5. Violation Checklist

- [ ] JPA annotations in the Domain layer? (FAIL)
- [ ] Using `null` instead of `Optional` in return types? (FAIL)
- [ ] Business logic inside `@Controller`? (FAIL)
