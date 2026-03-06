---
name: python-development-rules
description: Modern Python coding standards with Type Hints and Clean Architecture
applyTo: "**/*.py"
---

# Python Development Standards

## 1. Type Safety

- **Type Hints**: Mandatory for all function signatures (parameters and returns).
- **Static Analysis**: Must pass `mypy --strict` or `pyright`.
- **Validation**: Use `Pydantic` v2 for all DTOs and configuration models.

## 2. Layering

- **Entities**: Use `@dataclass(frozen=True)` or `Pydantic` models for domain entities.
- **Interfaces**: Use `abc.ABC` and `@abstractmethod` to define repository interfaces.
- **Dependency Injection**: Use a library like `pinject` or simple constructor injection (Manual DI).

## 3. Idioms (PEP 8+)

- Follow PEP 8 strictly. Use `ruff` or `black` for formatting.
- Avoid dynamic attribute injection (no `setattr` on business objects).
- Prefer list comprehensions or generators over complex loops.

## 4. Multi-Tenancy

- Use `contextvars` to manage tenant global state across async tasks.
- Ensure all repository methods filter by `tenant_id` automatically or via explicit parameters.

## 5. Violation Checklist

- [ ] Missing type hints? (FAIL)
- [ ] Logic inside `__init__.py` files? (FAIL)
- [ ] Raw SQL in service files? (FAIL)
