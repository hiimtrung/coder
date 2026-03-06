---
name: golang-development-rules
description: Go-specific idioms for Clean Architecture, error handling, and concurrency
applyTo: "**/*.go"
---

# Golang Development Standards

## 1. Project Layout (Standard Go)

- `/cmd`: Entry points (main.go).
- `/internal`: Private code; contains layers:
  - `/domain`: Business logic and interfaces.
  - `/usecase`: Application logic.
  - `/infrastructure`: Adapters (DB, External API).
- `/pkg`: Public libraries (reusable code).

## 2. Idiomatic Code

- **Composition**: Use embedding for composition (no inheritance).
- **Interfaces**: Define interfaces where they are _consumed_, not where they are implemented.
- **Error Handling**: Explicitly check `if err != nil`. Wrap errors with `fmt.Errorf("context: %w", err)` to preserve stack trace/cause.
- **Panic**: Never use `panic` for business logic. Use `error` return values.

## 3. Concurrency

- **Channels**: Use channels for communication, not shared memory (mutexes) where possible.
- **Context**: Always propagate `context.Context` for cancellation and tenant tracing.

## 4. Multi-Tenancy

- Store `TenantID` in the `context.Value`.
- Every repository method must take `ctx context.Context` and extract the tenant identifier for queries.

## 5. Violation Checklist

- [ ] Direct DB calls in the main handlers? (FAIL)
- [ ] Global variables for state? (FAIL)
- [ ] Ignoring errors? (FAIL - use `_` only when absolutely safe)
- [ ] Third-party framework leakage into the `internal/domain` package? (FAIL)
