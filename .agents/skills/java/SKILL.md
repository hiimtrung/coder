---
name: java
description: Expert in implementing enterprise Java applications (Spring/Quarkus) with Clean Architecture, strict typing, and multi-tenant design. Covers Gradle, Maven, and modern Java patterns.
---

# Skill: Java Backend Development

## Overview

Expert in implementing enterprise Java applications (Spring Boot, Quarkus) with Clean Architecture and strict typing.

## Rule Categories

| Priority | Category | Impact |
| :--- | :--- | :--- |
| 1 | [Java Idioms](rules/idioms.md) | HIGH |
| 2 | [Error Handling](rules/errors.md) | MEDIUM |

## Core Expertise

- **Clean Architecture**: Domain → Application → Presentation.
- **Strict Typing**: Generics, Records (Java 14+), and Sealed classes.
- **Exception-Based Error Handling**: Domain exceptions and global handlers.

## Instructions for the Agent

- **Consult Internal Rules**: Always refer to the `rules/` directory within this skill for Java-specific idioms and error codes.
- **Dependency Injection**: Use constructor-based injection (no @Autowired on fields).
- **Type Safety**: Use proper generics; avoid Object casts and raw types.

