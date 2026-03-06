---
name: general-instructions
description: Global project rules and architecture patterns (Synced from .github/copilot-instructions.md)
applyTo: "**/*"
---

# Copilot Instructions - AI-Agents Development System

This is a **unified AI Agent system** for VS Code Copilot designed to guide development across **multiple languages and frameworks** with consistent, maintainable code standards.

## 🎯 System Overview

You are working in a **multi-codebase workspace** with:

- **omi-channel-be**: NestJS multi-tenant omnichannel backend (PostgreSQL + MongoDB + Redis)
- **crm_be**: Spring Boot CRM backend
- **packageTourApi**: Java REST API
- **packageTourAdmin** & **findtourgoUI**: Next.js/React frontends
- **ai-agents**: AI development guidance system (current)

## 🛡️ Workspace Safety & Sandbox

To ensure system stability and security, the following rules apply to all AI Agent operations:

1. **Sandbox Execution**: Strictly forbid reading or modifying files outside of the defined project repositories (`/Users/trungtran/ai-agents/` and its sub-packages).
2. **Forbidden Directories**: NEVER access or write to system directories, including but not limited to: `/tmp`, `/etc`, `/var`, `/usr`, `~/.ssh`, `~/.config`, or OS-specific sensitive paths.
3. **Path Validation**: Before any `write_to_file` or `replace_file_content` operation, verify that the `TargetFile` absolute path resides within the approved workspace.
4. **Tool usage**: Only use tools on files that you have explicitly identified as part of the project structure via `list_dir` or `find_by_name`.

---

All projects follow **Clean Architecture + Event-Driven Design** with standardized error codes and patterns.

---

## 📋 Language-Specific Rules

Detailed instructions for each language are in `.agents/rules/`:

### Backend Languages

| Language       | Skill Reference | Primary Projects                               |
| -------------- | --------------- | ---------------------------------------------- |
| **TypeScript** | `nestjs`        | omi-channel-be, findtourgoUI, packageTourAdmin |
| **Java**       | `java`          | crm_be, packageTourApi                         |
| **Go**         | `golang`        | (reference for future services)                |
| **Rust**       | `rust`          | (reference for future services)                |
| **Python**     | `python`        | (reference for utilities/scripts)              |
| **C**          | `c`             | (reference, if needed)                         |
| **Dart**       | `dart`          | (reference, if needed)                         |

### Architecture & Domain Rules

| Topic                             | Skill Reference    | Focus                                                                         |
| --------------------------------- | ------------------ | ----------------------------------------------------------------------------- |
| **Architecture Patterns**         | `architecture`     | Clean Architecture, layers, module structure, DDD                             |
| **Business Logic Patterns**       | `development`      | Use cases, domain events, exception handling                                  |
| **Infrastructure & Integrations** | `database`         | Repositories, external APIs, messaging, databases                             |
| **Error Codes & Exceptions**      | `general-patterns` | Error code standards, exception patterns, HTTP status codes, recovery actions |
| **UI/UX Design Intelligence**     | `ui-ux-pro-max`    | Design system generation, styles, accessibility, and professional UI rules    |
| **Semantic Memory**               | `coder memory` | Semantic search, pattern storage, and context retrieval                       |

---

## 🚀 Quick Start for AI Agents

### 1. Identify the Project & Language + Error Handling Rules

```bash
# Example: Working in omi-channel-be (TypeScript)
→ Load Skill: `nestjs` (primary)
→ Reference Skill: `general-patterns` (exceptions & error codes)
→ Reference Skill: `architecture`
→ Reference Skill: `development`

# Example: Working in crm_be (Java)
→ Load Skill: `java` (primary)
→ Reference Skill: `general-patterns` (exceptions & error codes)
→ Reference Skill: `architecture`
→ Reference Skill: `database`
```

### 2. Follow the Hierarchy

1. **Language-specific skills** (e.g., `nestjs`, `java`) - type safety, syntax, idioms
2. **General patterns & error handling** (`general-patterns`) - standardized error codes, HTTP status, recovery actions
3. **Architecture skills** (`architecture`) - folder structure, Clean Architecture layers
4. **Development skills** (`development`) - use cases, domain events, exception handling
5. **Database skills** (`database`) - database operations, external integrations

### 3. When in Doubt

- Check the relevant rule file for patterns
- Check error codes reference for exception patterns
- Look for examples in the codebase
- Ask for specific files that exemplify the pattern (e.g., "show me a use case implementation in omi-channel-be")

---

## 🏗️ Unified Architecture Principles

### Clean Architecture (All Languages)

```
Presentation Layer (Controllers/Handlers)
    ↓ (calls)
Application Layer (Use Cases, Services, DTOs)
    ↓ (uses)
Domain Layer (Entities, Exceptions, Interfaces)
    ↓ (implements)
Infrastructure Layer (Repositories, External APIs)
```

### Error Handling (Standardized Across All Projects)

**Error Code Format**: `{CATEGORY}_{DESCRIPTIVE_NAME}`

```
Categories:
- AUTH_*     : Authentication/Authorization (401, 403)
- VAL_*      : Validation errors (400)
- BIZ_*      : Business logic errors (400, 404, 409)
- INF_*      : Infrastructure errors (500, 502, 503)
- SYS_*      : System errors (500)

Examples:
- AUTH_INVALID_TOKEN
- VAL_MISSING_REQUIRED_FIELD
- BIZ_ENTITY_NOT_FOUND
- INF_DATABASE_ERROR
- SYS_INTERNAL_ERROR
```

### Multi-Tenancy Pattern

Every request includes tenant context (company_id from JWT):

- **Extract** in controller from JWT
- **Pass** through all use cases/services
- **Validate** at use case level (never in database)
- **Use** in all repository queries for data isolation

### Semantic Memory System (coder memory)

All AI Agent operations MUST utilize the local semantic memory via **Memory Gating**:
1. **Gate In (MANDATORY)**: EVERY workflow must start by running `coder memory search "<topic>"` to retrieve historical context, patterns, and decisions.
2. **Gate Out (MANDATORY)**: EVERY workflow must end by running `coder memory store "<Title>" "<Content>" --tags "<tags>"` to capture new patterns, root causes, or architectural decisions.
3. **Compact Regularly**: Run `coder memory compact --revector` periodically to keep the memory clean and up-to-date.

## ⚠️ Critical Rules (ALL Languages)

1. **Type Safety**: NO `any` types, explicit typing everywhere
2. **Error Handling**: Throw domain exceptions with standardized error codes (see `general-patterns` skill)
   - **AUTH\_\*** (401, 403): Authentication & authorization failures
   - **VAL\_\*** (400): Input validation failures
   - **BIZ\_\*** (400, 404, 409): Business logic violations
   - **INF\_\*** (500, 502, 503): Infrastructure/external failures
   - **SYS\_\*** (500): System/configuration errors
3. **Multi-Tenancy**: ALWAYS validate company_id on queries
4. **Events**: Publish AFTER persistence, NOT before (event-driven architecture)
5. **Memory** — TWO MANDATORY GATES:

   **GATE 1 (START)** — Run before touching any code:

   ```bash
   coder memory search "<task topic>"
   ```

   **GATE 2 (END)** — Run after completing any non-trivial work:

   ```bash
   coder memory store "<Title>" "<What was done, why, which files>" --tags "<project,topic>"
   ```

   Both gates are **blocking**. Skipping either is a workflow violation.

   **Todo List Rule**: Every todo list MUST be structured as:
   ```
   1. [GATE 1] Memory search: "<topic>"      ← always first
      ... actual work tasks ...
   N. [GATE 2] Memory store: "<title>"       ← always last
   ```
   A todo list without these two bookend tasks is **invalid**.

6. **Build**: Run lint → build → test before commit
7. **Package Manager**: Use `yarn` for NestJS, `gradle` for Java (never npm for Node projects)

---

## 🔄 Cross-Module Communication

### Right Way: Event-Driven

```typescript
// Module A: Publish event after persistence
await this.repository.save(entity);
await this.eventPublisher.publish(new DomainEvent(...));

// Module B: Subscribe to event in handler
@EventsHandler(DomainEvent)
export class DomainEventHandler { ... }
```

### Wrong Way: Direct Repository Calls

```typescript
// ❌ FORBIDDEN: Module A calling Module B's repository
private readonly moduleB_Repository: IRepository; // ← Cross-module!
```

---

## 📖 How to Use These Rules

**For Copilot/AI Agents**:

1. When asked to work on a file, check its extension (`.ts`, `.java`, etc.)
2. Load the corresponding language skill (e.g., `nestjs`, `java`)
3. Cross-reference `general-patterns` as needed
4. Reference `architecture`/`development` skills for patterns
5. Ask for specific code examples from the codebase if clarification needed

**For Developers**:

1. Keep `.agents/skills/` referenced when developing
2. Refer to the language-specific skill for your project
3. Check `general-patterns` for exception patterns
4. Reference `architecture` skill for folder structure questions
5. Reference `development` skill for use case/domain patterns

---

## 🔍 Complete Rules Reference Map

### Core Skills (All Languages)

1. **`general-patterns`** - Standardized error codes (AUTH*\*, VAL*\_, BIZ\_\_, INF*\*, SYS*\*), exception patterns, HTTP status codes, recovery actions
2. **`architecture`** - Clean Architecture, layer structure, module organization, design patterns
3. **`development`** - Use cases, domain events, exception handling, validation patterns
4. **`database`** - Repositories, external APIs, database operations, messaging
5. **`ui-ux-pro-max`** - Design system generation, styles, accessibility, and professional UI guidelines

### Language-Specific Skills

- **`nestjs`** - Type safety, NestJS patterns, module structure (for omi-channel-be, findtourgoUI, packageTourAdmin)
- **`java`** - Spring Boot patterns, annotations, dependency injection (for crm_be, packageTourApi)
- **`golang`** - Interfaces, error handling, concurrency (reference for future Go services)
- **`rust`** - Ownership, Result types, async/await (reference for future Rust services)
- **`python`** - Type hints, best practices (reference for scripts/utilities)
- **`dart`** - Classes, async patterns (reference for mobile/Flutter apps)
- **`c`** - Memory management, C conventions (reference if needed)

### Rule Selection Guide

| Scenario                                       | Load First         | Then Reference                                       |
| ---------------------------------------------- | ------------------ | ---------------------------------------------------- |
| Creating new TypeScript file in omi-channel-be | `nestjs`           | `general-patterns`, `architecture`, `development`    |
| Creating new Java file in crm_be               | `java`             | `general-patterns`, `architecture`, `database`       |
| Implementing new exception                     | `general-patterns` | `development`, language-specific skills              |
| Designing new module architecture              | `architecture`     | `general-patterns`, `development`, language-specific |
| Building cross-module integration              | `development`      | `architecture`, `general-patterns`, `database`       |
| Integrating external API                       | `database`         | `general-patterns`, language-specific, `development` |
| Designing a new UI/Landing Page                | `ui-ux-pro-max`    | `nestjs`, `architecture`, `general-patterns`         |
| Reviewing UI for professional quality          | `ui-ux-pro-max`    | `web-design-guidelines`, `general-patterns`          |

---

## 🎓 Learning Path for New Developers

1. **Day 1**: Read the project's main `README.md`
2. **Day 2-3**: Read language-specific skill (e.g., `nestjs`)
3. **Day 3**: Read `architecture` skill
4. **Day 4**: Read project docs (e.g., omi-channel-be/docs/ARCHITECTURE.md)
5. **Day 5+**: Reference `development` and `database` skills as needed

---

**Last Updated**: February 2026
**System**: AI-Agents Unified Development Guidance
**Status**: Production Ready
