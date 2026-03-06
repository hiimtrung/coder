---
name: dart-development-rules
description: Dart/Flutter coding standards with sound null safety and Clean Architecture
applyTo: "**/*.dart"
---

# Dart Development Standards

## 1. Clean Architecture Patterns

- **Features folder**: Organise by `feature/data`, `feature/domain`, `feature/presentation`.
- **Domain**: Contain Entities and Abstract Repositories.
- **Data**: Implement Repositories and Data Sources (MDS/RDS).
- **Presentation**: BLoC, Provider, or Riverpod for state management.

## 2. Language Features

- **Null Safety**: Always use sound null safety. No `!`.
- **Async/Await**: Use `Future` and `Stream` correctly. Prefer `async*` for streams.
- **Records**: Use Dart 3+ Records for multiple return values instead of utility classes.

## 3. Immutability

- Use `equatable` or `freezed` for immutable domain entities and states.
- All classes in Domain and Application layers should ideally be `immutable`.

## 4. Multi-Tenancy

- Manage Tenant/Company context via a Provider or global Service Locator (GetIt).
- Inject the context into data repositories to ensure restricted data access.

## 5. Violation Checklist

- [ ] Using `dart:html` in non-web projects? (FAIL)
- [ ] UI logic in the Data layer? (FAIL)
- [ ] Direct API calls without a Repository abstraction? (FAIL)
