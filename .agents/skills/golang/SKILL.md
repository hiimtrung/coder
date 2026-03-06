---
name: golang
description: Expert in building concurrent Go services with clean architecture, error handling, and idiomatic patterns. Covers Go modules, interfaces, and concurrency best practices.
---

# Skill: Go Backend Development

## Overview

Expert in building concurrent Go services with clean architecture and idiomatic patterns.

## Rule Categories

| Priority | Category | Impact |
| :--- | :--- | :--- |
| 1 | [Go Idioms](rules/idioms.md) | HIGH |
| 2 | [Error Handling](rules/errors.md) | MEDIUM |

## Core Expertise

- **Clean Architecture**: Domain → Application → Presentation.
- **Idiomatic Go**: Error returns, interface-based design, and concurrency patterns.
- **Dependency Management**: Go Modules (Standard).

## Instructions for the Agent

- **Consult Internal Rules**: Always refer to the `rules/` directory within this skill for Go-specific idioms and error codes.
- **No Panic**: Never panic in production code; use explicit error returns with context.
- **Interfaces**: Use small, focused interfaces for dependencies.

