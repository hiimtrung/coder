<div align="center">

<img src="logo.png" width="300" alt="coder logo">

```
   ██████╗ ██████╗ ██████╗ ███████╗██████╗
  ██╔════╝██╔═══██╗██╔══██╗██╔════╝██╔══██╗
  ██║     ██║   ██║██║  ██║█████╗  ██████╔╝
  ██║     ██║   ██║██║  ██║██╔══╝  ██╔══██╗
  ╚██████╗╚██████╔╝██████╔╝███████╗██║  ██║
   ╚═════╝ ╚═════╝ ╚═════╝ ╚══════╝╚═╝  ╚═╝
```

**Universal engineering intelligence for AI agents.**
Distribute skills, enforce architecture, and preserve memory — so every agent on your team operates with the same institutional knowledge.

[![Build & Release](https://github.com/hiimtrung/coder/actions/workflows/release.yml/badge.svg)](https://github.com/hiimtrung/coder/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/hiimtrung/coder)](https://goreportcard.com/report/github.com/hiimtrung/coder)
[![Latest Release](https://img.shields.io/github/v/release/hiimtrung/coder)](https://github.com/hiimtrung/coder/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

</div>

---

## What is coder?

Most AI agents operate in a vacuum — no memory, no standards, no institutional knowledge. **coder** fixes that.

It gives every AI agent in your team access to the same centralized brain: a vector-powered knowledge base holding your architecture rules, your senior engineers' patterns, and the project history that made those decisions meaningful.

**coder is a pure memory and skill service.** It does not run LLMs. All reasoning, planning, and code generation is done by your AI agent (Claude, GitHub Copilot, or any MCP client). coder provides the knowledge retrieval and storage infrastructure that makes those agents consistently good.

```
  Your Team's Knowledge                  AI Agents Anywhere
  ┌──────────────────────┐               ┌──────────────────┐
  │  Architecture rules  │               │  Claude Code     │
  │  NestJS patterns     │  ─── coder ─▶ │  GitHub Copilot  │
  │  Past decisions      │               │  Any MCP client  │
  │  Bug post-mortems    │               └──────────────────┘
  └──────────────────────┘
```

---

## Features

| Feature                 | Description                                                                       |
| ----------------------- | --------------------------------------------------------------------------------- |
| **Hybrid RAG Search**   | pgvector cosine similarity fused with full-text search via Reciprocal Rank Fusion |
| **Semantic Memory**     | Store and retrieve cross-project decisions, patterns, and post-mortems            |
| **20+ Built-in Skills** | NestJS, Go, Java, Rust, Python, React, architecture, testing, and more            |
| **Dual Transport**      | gRPC (performance) + HTTP (compatibility) — both support Bearer token auth        |
| **Secure Mode**         | Bootstrap token registration, SHA-256 hashed storage, per-client access tokens    |
| **Activity Tracking**   | Fire-and-forget telemetry: command + repo + branch, logged per developer          |
| **Web Dashboard**       | Embedded HTMX dashboard for monitoring clients, memory, and activity              |
| **Self-Hosted**         | One Docker command — Postgres + pgvector + coder-node                             |
| **Single Binary**       | ~7MB CLI, zero runtime dependencies, cross-platform                               |

---

## Documentation

| Document                                                                 | Description                                                       |
| ------------------------------------------------------------------------ | ----------------------------------------------------------------- |
| [**Usage Guide**](docs/GUIDE.md)                                         | Quick guide for the current working CLI                           |
| [**CLI Reference**](docs/cli.md)                                         | Current implemented commands and examples                         |
| [**Installation**](docs/installation.md)                                 | CLI + coder-node setup, secure mode, env vars                     |
| [**Architecture**](docs/architecture.md)                                 | System design, data flows, layer structure                        |
| [**Skill System**](docs/skill_system.md)                                 | How the vector RAG works                                          |
| [**Memory System**](docs/memory_system.md)                               | Semantic memory internals                                         |
| [**Memory Lifecycle Plan**](docs/memory_lifecycle_plan.md)               | Freshness, validity, and superseded memory handling               |
| [**Dynamic Skill Retrieval Plan**](docs/dynamic_skill_retrieval_plan.md) | Proposed upgrade for mid-task skill loading and context budgeting |
| [**Skill Files**](docs/skill_files.md)                                   | Bundling and executing binary assets                              |
| [**Secure Mode**](docs/secure_mode.md)                                   | Node-level security and client registration                       |
| [**Web Dashboard**](docs/dashboard.md)                                   | HTMX-powered visual management console                            |
| [**Development**](docs/development.md)                                   | Building from source, release process                             |
| [**Intelligence Roadmap**](docs/roadmap-intelligence-flows.md)           | Future design ideas; not the current command surface              |
| [**Changelog**](CHANGELOG.md)                                            | Release history                                                   |

---

## Quick Start

### 1 — Install the CLI

```bash
# macOS / Linux
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/install.sh)"

# Windows (PowerShell)
irm https://raw.githubusercontent.com/hiimtrung/coder/main/install.ps1 | iex
```

### 2 — Start coder-node

```bash
# Open mode — no auth required
curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/install-node.sh | sh

# Secure mode — restrict to registered developers
curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/install-node.sh | sh -s -- --secure
```

> **Secure mode**: On first startup the server prints a one-time bootstrap token.
> Each developer runs `coder login` and enters the token to register their machine.
> All subsequent API calls carry a `Bearer` token automatically — over both gRPC and HTTP.

### 3 — Connect

```bash
coder login
# Prompts for protocol, URL, and auth token (if secure mode)
```

### 4 — Apply to a project

```bash
cd my-project
coder install fullstack             # scaffold .agents/ and .claude/agents/ into the project
coder skill ingest --source local   # load 20+ built-in skills into the vector DB
```

### 5 — Use it with your AI agent

Open Claude Code, GitHub Copilot, or any MCP-compatible AI agent and instruct it to use `coder skill resolve`, `coder memory search`, `coder memory recall`, and `coder memory store` as part of its workflow.

The current binary is centered on knowledge retrieval and storage. Roadmap commands documented elsewhere are not automatically available unless they are wired in `cmd/coder/main.go`.

---

## How it works with AI Agents

coder does not run language models. Your AI agent (Claude, Copilot, etc.) is the reasoning engine. coder provides the knowledge retrieval layer that makes every agent consistently informed.

```
┌──────────────────────────────────────────────────────────────┐
│                     Developer Machine                        │
│                                                              │
│  AI Agent (Claude Code / GitHub Copilot / any MCP client)   │
│       │                                                      │
│       │  1. coder skill resolve "implement auth middleware"  │
│       │     → Returns: active skill set for the task         │
│       │                                                      │
│       │  2. coder memory search "auth middleware"            │
│       │     → Returns: past decisions, known issues          │
│       │                                                      │
│       │  3. Agent reasons, plans, and writes code            │
│       │                                                      │
│       │  4. coder memory store "Auth decision" "content..."  │
│       │     → Persists new knowledge for next agent/session  │
│       │                                                      │
│  coder CLI  ──── Bearer token ────▶  coder-node              │
│                    (gRPC / HTTP)                             │
└──────────────────────────────────────────────────────────────┘
                              │
              ┌───────────────┴───────────────┐
              │           coder-node           │
              │                               │
              │  Auth interceptors            │
              │  Hybrid search (RRF)          │
              │  Skill ingestor               │
              │  Memory manager               │
              └───────────────┬───────────────┘
                              │
              ┌───────────────┴───────────────┐
              │    PostgreSQL + pgvector       │
              │    Ollama / OpenAI / FTS-only  │
              └───────────────────────────────┘
```

### The 3-Gate Loop

Agent workflows enforce a consistent knowledge gate pattern:

1. **Gate 1 — Active skill retrieval**: `coder skill resolve "<topic>" --trigger initial --budget 3` — selects the active skill set for the current task
2. **Gate 2 — Memory retrieval**: `coder memory search "<topic>"` — loads project-specific history and past decisions
3. **Gate 3 — Knowledge capture**: `coder memory store "<title>" "<content>"` — persists new patterns so every future agent benefits

Use `coder skill search` for ad hoc discovery and `coder memory recall` when a long-running task needs to refresh the active memory set without restarting the workflow.

### Specialized Sub-Agents

The agent system simulates a professional delivery team with specialized roles:

| Agent               | Role                   | When to Use                                     |
| ------------------- | ---------------------- | ----------------------------------------------- |
| `coder`             | Fullstack orchestrator | End-to-end delivery, coordinates all phases     |
| `coder-ba`          | Business Analyst       | Elicit and document requirements before design  |
| `coder-architect`   | System Architect       | Design system, write ADRs, define API contracts |
| `coder-be`          | Backend Developer      | Implement APIs, services, repositories          |
| `coder-fe`          | Frontend Developer     | Build components, pages, design system          |
| `coder-reviewer`    | Code Reviewer          | Enforce quality before merge                    |
| `coder-qa`          | QA Engineer            | Acceptance testing, regression, QA report       |
| `coder-tech-writer` | Technical Writer       | API docs, runbooks, CHANGELOG                   |
| `coder-debugger`    | Debugger               | Root cause analysis, post-mortems               |

---

## Key Commands

### Intelligence Gates (always run these)

```bash
coder skill resolve "implement grpc auth flow" --trigger initial --budget 3
coder memory search "auth pattern"
coder memory store "Auth decision" "content..."
```

### Memory Management

```bash
coder memory search "<query>"                    # Search semantic memory
coder memory recall "<task>"                     # Refresh active memory for a running task
coder memory active --format json                # Inspect the local active memory snapshot
coder memory store "<title>" "<content>"         # Store new knowledge
coder memory verify <id>                         # Refresh verification metadata
coder memory supersede <old-id> <new-id>         # Mark one version as replaced
coder memory audit                               # Report lifecycle conflicts or stale truths
coder memory list                                # List recent entries
coder memory compact --revector                  # Clean and re-embed memory
```

### Skill Management

```bash
coder skill resolve "<task>" --trigger execution --budget 3
coder skill active --format json
coder skill search "<topic>"                     # Ad hoc skill search
coder skill list                                 # List all ingested skills
coder skill info <name>                          # Detailed skill info
coder skill ingest --source local                # Load built-in skills
coder skill ingest --source github --repo hiimtrung/coder
```

### Session and Progress

```bash
coder session save                               # Save working context
coder progress                                   # See current project state
coder next                                       # Get next recommended action
coder milestone complete N                       # Close a milestone
```

### System

```bash
coder login                                      # Connect to coder-node
coder install fullstack                          # Scaffold agent engine into project
coder self-update                                # Update the CLI binary
```

### Current Command Surface

The current CLI exposes:

```text
install, update, list, version, check-update, self-update,
login, token, skill, memory, remove, session, progress, next, milestone
```

If another document mentions `chat`, `review`, `debug`, `plan`, `workflow`, `new-project`, or similar commands, treat those as roadmap concepts unless they are present in `cmd/coder/main.go`.

---

## Authentication (Secure Mode)

When `coder-node` runs with `--secure`, every API call requires a valid Bearer token.

```
Server admin                        Developer
     │                                   │
     │  ./install-node.sh --secure       │
     │  docker logs | grep BOOTSTRAP     │
     │  ─── shares token ──────────────▶ │
     │                                   │  coder login
     │                                   │  > auth? y
     │                                   │  > token: <bootstrap>
     │                                   │  ✓ registered, token saved
     │                                   │
     │                                   │  coder skill search ...
     │                    Bearer <token> │  (automatic, every call)
```

Token lifecycle:

- Raw token generated with `crypto/rand`, never stored
- Only the SHA-256 hash lives in the database
- Bootstrap token shown once in server logs
- Access tokens sent via `authorization` metadata on gRPC; `Authorization: Bearer` on HTTP

---

## Infrastructure

coder-node requires PostgreSQL with the pgvector extension plus an embedding provider. The default local setup uses Ollama for embeddings, but the system can also run with OpenAI embeddings or FTS-only mode depending on configuration. coder itself still does not run chat LLMs.

```yaml
# docker-compose.yml
services:
  postgres:
    image: pgvector/pgvector:pg16
    environment:
      POSTGRES_DB: coder
      POSTGRES_USER: coder
      POSTGRES_PASSWORD: coder
    volumes:
      - pgdata:/var/lib/postgresql/data

  coder-node:
    image: ghcr.io/hiimtrung/coder-node:latest
    ports:
      - "8080:8080" # HTTP
      - "9090:9090" # gRPC
    environment:
      DATABASE_URL: postgres://coder:coder@postgres:5432/coder
    depends_on:
      - postgres

volumes:
  pgdata:
```

---

## Contributing

Issues and pull requests are welcome. See the [Development Guide](docs/development.md) for build instructions, project structure, and the release process.

---

<div align="center">

Built in Go · Self-hosted · MIT License

</div>
