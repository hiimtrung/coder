---
name: dart
description: Expert in Flutter/Dart development with sound null safety and Clean Architecture.
---

# Skill: Dart & Flutter Development

## Overview

Expert in building performant cross-platform applications using Dart and Flutter with sound null safety.

## Rule Categories

| Priority | Category | Impact |
| :--- | :--- | :--- |
| 1 | [Dart Idioms](rules/idioms.md) | HIGH |
| 2 | [Error Handling](rules/errors.md) | MEDIUM |

## Core Expertise

- **Clean Architecture**: Feature-based organization (Data, Domain, Presentation).
- **Deep Null Safety**: Strict avoidance of `!` and leveraging sound null safety.
- **State Management**: BLoC, Provider, and Riverpod expertise.
- **Immutability**: Leveraging `freezed` and `equatable` for predictable state.

## Instructions for the Agent

- **Consult Internal Rules**: Always refer to the `rules/` directory within this skill for Dart-specific idioms and error codes.
- **Follow Layering**: Keep UI logic out of the Data layer; use Repositories as abstractions.
- **Native Bridges**: Be cautious with `dart:html` in non-web projects.
