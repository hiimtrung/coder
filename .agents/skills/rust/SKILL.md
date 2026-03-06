---
name: rust
description: Expert in building type-safe, concurrent Rust services with ownership patterns, error handling, and clean architecture. Covers Actix, Axum, Rocket, and memory-safe patterns.
---

# Skill: Rust Backend Development

## Overview

Expert in building type-safe, concurrent Rust services with ownership patterns and clean architecture.

## Rule Categories

| Priority | Category | Impact |
| :--- | :--- | :--- |
| 1 | [Rust Idioms](rules/idioms.md) | HIGH |
| 2 | [Error Handling](rules/errors.md) | MEDIUM |

## Core Expertise

- **Clean Architecture**: Domain → Application → Presentation.
- **Ownership & Borrowing**: No nulls, memory safe, and explicit Result/Option handling.
- **Trait-Based Design**: Composition over inheritance.

## Instructions for the Agent

- **Consult Internal Rules**: Always refer to the `rules/` directory within this skill for Rust-specific idioms and error codes.
- **No Unwraps**: Avoid `unwrap()` unless in debug or tests; use the `?` operator for error propagation.
- **Traits for Dependencies**: Depend on traits (Arc<dyn Trait>) rather than concrete types.

