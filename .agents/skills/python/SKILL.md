---
name: python
description: Expert in modern Python development with Type Hints, Pydantic, and Clean Architecture.
---

# Skill: Python Development

## Overview

Expert in modern Python development using type hints, strict validation, and Clean Architecture principles.

## Rule Categories

| Priority | Category | Impact |
| :--- | :--- | :--- |
| 1 | [Python Idioms](rules/idioms.md) | HIGH |
| 2 | [Error Handling](rules/errors.md) | MEDIUM |

## Core Expertise

- **Type Safety**: Mandatory type hints and static analysis (mypy/pyright).
- **Validation**: Pydantic v2 for robust DTOs and configuration.
- **Clean Architecture**: Interface isolation (ABC) and dataclass-based entities.
- **Modern Idioms**: Ruff/Black formatting and PEP 8 strict compliance.

## Instructions for the Agent

- **Consult Internal Rules**: Always refer to the `rules/` directory within this skill for Python-specific idioms and error codes.
- **No Globals**: Avoid logic in `__init__.py`; keep modules clean and focused.
- **Async Safety**: Use `contextvars` for managing tenant state in async environments.
