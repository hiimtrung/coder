---
name: strict-type-safety
description: Mandatory type safety and code quality standards for strongly typed languages
applyTo: "**/*.{ts,tsx,java,go}"
---

# Strict Type Safety & Quality Standards

## 1. Type Enforcement (CRITICAL)

### ❌ FORBIDDEN: Anti-Type Patterns

- **No `any` or equivalent**: Escape hatches from the type system are strictly forbidden. Use `unknown` or specific interfaces if the type is truly dynamic.
- **No Implicit Casts**: Always use proper type guards or structure validation (e.g., Zod, Class-Validator).
- **No `null` as default**: Use Optionals, Unions (e.g., `string | undefined`), or Default Objects to represent absence of data.

### ✅ REQUIRED: Strong Typing

- **Explicit Returns**: All public functions/methods MUST have explicit return types.
- **Generic Constraints**: All generics must have an `extends` constraint (no unbounded `<T>`).
- **Discriminated Unions**: Use Tagged Unions for polymorphic state (e.g., `type Status = Success | Error`).

## 2. Immutability

- Prefer `readonly` fields and `const` variables.
- Domain Entities should be updated via methods that return new state or update internal private state, never by direct public property mutation.

## 3. Error Handling Philosophy

- **Exceptions for Unrecoverable**: Use exceptions for truly exceptional/unexpected errors.
- **Domain Results for Business Logic**: For expected failures (e.g., "User Not Found"), prefer returning a Result/Either type or throwing a typed **DomainException**.

## 4. Code Quality

- **Self-Documenting Code**: Variable names must describe _intent_, not _type_ (e.g., `pendingOrders` instead of `orderList`).
- **Function Size**: Methods should ideally follow the Single Responsibility Principle and fit within one screen (approx. 20-30 lines).
- **No Magic Values**: Use Enums or Constants for all literals.

## 5. Technology Specifics (TypeScript)

- `strict: true` in `tsconfig.json` is non-negotiable.
- No `as any` or `!`.
- Use `Satisfies` operator for inferred types with constraints.
- Documentation: JSDoc required for all exported interfaces and classes.
