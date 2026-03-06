---
name: c
description: Expert in procedural Clean Architecture, memory safety, and modular C development.
---

# Skill: C Systems Development

## Overview

Expert in building safe, modular, and performant systems using C with a focus on procedural Clean Architecture.

## Rule Categories

| Priority | Category | Impact |
| :--- | :--- | :--- |
| 1 | [C Idioms](rules/idioms.md) | HIGH |
| 2 | [Error Handling](rules/errors.md) | MEDIUM |

## Core Expertise

- **Modular Architecture**: Opaque pointers, header encapsulation, and clear layering.
- **Memory Safety**: Strict ownership rules, buffer safety, and mandatory NULL checks.
- **Procedural Patterns**: Struct-based "methods" and explicit return codes.

## Instructions for the Agent

- **Consult Internal Rules**: Always refer to the `rules/` directory within this skill for C-specific idioms and error codes.
- **Modularize**: Use headers to define interfaces and source files for private implementations.
- **Buffer Safety**: Strictly avoid unsafe string/buffer functions; use length-bounded alternatives.
