---
name: general-patterns
description: Universal backend development patterns, error handling, and architecture principles applicable across all languages (NestJS, Java, Go, Rust).
---

# Skill: General Backend Patterns

## Overview

Language-agnostic patterns for backend development applicable across NestJS, Java, Go, and Rust.

## Rule Categories

| Priority | Category | Impact |
| :--- | :--- | :--- |
| 1 | [Universal Patterns](rules/patterns.md) | HIGH |
| 2 | [Error Handling](rules/errors.md) | HIGH |

## Core Expertise

- **Architecture Layers**: Presentation, Application, Domain, and Infrastructure isolation.
- **Multi-Tenant Isolation**: Context extraction and mandatory tenant validation.
- **Event-Driven Architecture**: Cross-module communication via domain events.
- **Persistence Strategy**: Relational (PostgreSQL), Append-Only (MongoDB), and Ephemeral (Redis).

## Instructions for the Agent

- **Consult Internal Rules**: Always refer to the `rules/` directory within this skill for universal business logic and error handling standards.
- **Layer Integrity**: Never skip layers; Presentation must go through Application which uses Domain.
- **Cross-Module Calls**: Never call repositories directly across module boundaries; use events.

