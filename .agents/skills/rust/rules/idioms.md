---
name: rust-development-rules
description: Rust safety, ownership, and Clean Architecture traits
applyTo: "**/*.rs"
---

# Rust Development Standards

## 1. Clean Architecture via Traits

- **Domain**: Define `traits` for repositories.
- **Application**: Use Cases are structs that take generic `T: RepositoryTrait`.
- **Infrastructure**: Implement the traits for specific drivers (SQLx, Diesel, etc.).

## 2. Safety & Error Handling

- **Result & Option**: NO `unwrap()` or `expect()`. Always use match patterns or `?` operator.
- **Zero-Cost Abstractions**: Use `Cow`, `References`, and `Slices` to minimize cloning.
- **Async**: Use `tokio` or `async-std` consistently.

## 3. Data Integrity

- **Newtype Pattern**: Use specific types (e.g., `struct CompanyId(String)`) to prevent ID mixing.
- **Validation**: Use `validator` crate at the boundary (DTOs).

## 4. Multi-Tenancy

- Implement a `TenantContext` middleware that injects the ID into the request state.
- Use `tower` layers or `actix/axum` extractors to ensure the tenant ID is present in every transaction.

## 5. Violation Checklist

- [ ] Use of `unsafe` block without extreme justification? (FAIL)
- [ ] Using `unwrap()` in production-ready business logic? (FAIL)
- [ ] Leakage of SQL crates into domain crate? (FAIL)
