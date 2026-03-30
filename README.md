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
Distribute skills, enforce architecture, preserve memory — and orchestrate full project delivery from requirements to pull request.

[![Build & Release](https://github.com/hiimtrung/coder/actions/workflows/release.yml/badge.svg)](https://github.com/hiimtrung/coder/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/hiimtrung/coder)](https://goreportcard.com/report/github.com/hiimtrung/coder)
[![Latest Release](https://img.shields.io/github/v/release/hiimtrung/coder)](https://github.com/hiimtrung/coder/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

</div>

---

## What is coder?

Most AI agents operate in a vacuum — no memory, no standards, no institutional knowledge. **coder** fixes that.

It gives every agent in your team access to the same centralized brain: a vector-powered knowledge base holding your architecture rules, your senior engineers' patterns, and the project history that made those decisions meaningful.

Beyond knowledge retrieval, **coder** provides a complete **project lifecycle orchestration** layer — from writing requirements all the way to opening a pull request, with structured planning, wave-based execution, and atomic git commits at every step.

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

| Feature | Description |
|---------|-------------|
| **Hybrid RAG Search** | pgvector cosine similarity fused with full-text search via Reciprocal Rank Fusion |
| **Semantic Memory** | Store and retrieve cross-project decisions, patterns, and post-mortems |
| **20+ Built-in Skills** | NestJS, Go, Java, Rust, Python, React, architecture, testing, and more |
| **AI Workflow Commands** | `chat`, `review`, `debug`, `plan`, `qa`, `session`, `workflow` — context-aware dev tools |
| **Project Lifecycle** | `new-project` → `discuss-phase` → `plan-phase` → `execute-phase` → `ship` — end-to-end delivery |
| **Dual Transport** | gRPC (performance) + HTTP (compatibility) — both support Bearer token auth |
| **Secure Mode** | Bootstrap token registration, SHA-256 hashed storage, per-client access tokens |
| **Activity Tracking** | Fire-and-forget telemetry: command + repo + branch, logged per developer |
| **Web Dashboard** | Embedded HTMX dashboard for monitoring clients, memory, and activity |
| **Self-Hosted** | One Docker command — Postgres + pgvector + Ollama + coder-node |
| **Single Binary** | ~7MB CLI, zero runtime dependencies, cross-platform |

---

## 📗 Documentation

| Document | Description |
|----------|-------------|
| [**Usage Guide**](docs/GUIDE.md) | Complete guide: all commands, lifecycle flow, flags, examples |
| [**CLI Reference**](docs/cli.md) | Every command with flags and examples |
| [**Installation**](docs/installation.md) | CLI + coder-node setup, secure mode, env vars |
| [**Architecture**](docs/architecture.md) | System design, data flows, layer structure |
| [**Skill System**](docs/skill_system.md) | How the vector RAG works |
| [**Memory System**](docs/memory_system.md) | Semantic memory internals |
| [**Memory Lifecycle Plan**](docs/memory_lifecycle_plan.md) | Detailed rollout plan for freshness, validity, and superseded memory handling |
| [**Skill Files**](docs/skill_files.md) | Bundling and executing binary assets |
| [**Secure Mode**](docs/secure_mode.md) | Node-level security and client registration |
| [**Web Dashboard**](docs/dashboard.md) | HTMX-powered visual management console |
| [**Intelligence Flows Roadmap**](docs/roadmap-intelligence-flows.md) | Implementation roadmap for all AI phases |
| [**Development**](docs/development.md) | Building from source, release process |
| [**Changelog**](CHANGELOG.md) | Release history |

---

## 🚀 Quick Start

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
coder install fullstack             # scaffold .agents/ into the project
coder skill ingest --source local   # load 20+ built-in skills into the vector DB
```

### 5 — Use it

```bash
# Quick AI tools (work anywhere)
coder chat "explain the auth middleware"
coder review                             # AI review of your git diff
coder debug "panic: nil pointer at auth.go:45"
coder plan "add rate limiting"           # generate PLAN.md
coder qa --plan .coder/plans/PLAN-*.md  # UAT against acceptance criteria

# Full project lifecycle
coder new-project "build a task manager API in Go"
coder map-codebase
coder discuss-phase 1
coder plan-phase 1
coder execute-phase 1
coder ship 1
```

---

## How it works

```
┌─────────────────────────────────────────────────────────┐
│                    Developer Machine                     │
│                                                         │
│  AI Agent (Claude / Copilot / any)                      │
│       │ coder skill search "topic"      ← GATE 1        │
│       │ coder memory search "topic"     ← GATE 2        │
│       │        ... does work ...                        │
│       │ coder memory store "title" ...  ← GATE 3        │
│       │                                                  │
│  coder CLI  ──── Bearer token ────▶  coder-node         │
│                    (gRPC / HTTP)        │                │
└─────────────────────────────────────────────────────────┘
                                          │
                              ┌───────────┴──────────┐
                              │    coder-node         │
                              │                      │
                              │  Auth interceptors   │
                              │  Context injection   │
                              │  Hybrid search (RRF) │
                              │  Skill ingestor      │
                              │  Memory manager      │
                              └──────────┬───────────┘
                                         │
                              ┌──────────┴──────────┐
                              │  PostgreSQL + pgvec  │
                              │  Ollama embeddings   │
                              └─────────────────────┘
```

The **3-Gate Loop** enforced by agent workflows:

1. **GATE 1 — Skill retrieval** `coder skill search "<topic>"` — retrieves architecture rules and best practices before any coding starts
2. **GATE 2 — Memory retrieval** `coder memory search "<topic>"` — loads project-specific history and past decisions
3. **GATE 3 — Knowledge capture** `coder memory store "<title>" "<content>"` — persists new patterns so the next agent benefits

---

## Key Commands

### Intelligence Gates (always run these)

```bash
coder skill search "NestJS error handling"       # GATE 1 — retrieve best practices
coder memory search "auth pattern"               # GATE 2 — retrieve project decisions
coder memory store "Auth decision" "content..."  # GATE 3 — capture new knowledge
```

### Quick AI Workflows

```bash
coder chat "explain the auth middleware"         # context-enriched Q&A
coder review                                     # AI code review of git diff
coder review --pr 42                             # review a GitHub PR
coder debug "panic: nil pointer at auth.go:45"  # root cause analysis
coder plan "add rate limiting"                   # generate PLAN.md
coder qa --plan .coder/plans/PLAN-*.md          # walk acceptance criteria
coder session save                               # save working context
coder workflow "add email verification"          # full auto-chain
```

### Project Lifecycle

```bash
coder new-project "build a REST API"   # initialize: requirements + roadmap
coder map-codebase                     # analyse codebase → STACK/ARCH/CONCERNS
coder discuss-phase 1                  # Q&A → CONTEXT.md
coder plan-phase 1                     # research + XML plans + verification
coder execute-phase 1                  # execute plans, atomic git commits
coder ship 1                           # gh pr create with AI-generated body
coder progress                         # see where you are
coder next                             # get the next recommended command
coder milestone complete 1             # close phase, advance to next
```

### System

```bash
coder login                             # connect to coder-node
coder install fullstack                 # scaffold agent engine into project
coder skill ingest --source local       # load built-in skills into vector DB
coder self-update                       # update the CLI binary
coder health                            # project health check
coder stats                             # project statistics
```

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
- Raw token generated with `crypto/rand`, **never stored**
- Only the SHA-256 hash lives in the database
- Bootstrap token shown **once** in server logs
- Access tokens sent via `authorization` metadata on gRPC; `Authorization: Bearer` on HTTP

---

## Contributing

Issues and pull requests are welcome. See the [Development Guide](docs/development.md) for build instructions, project structure, and the release process.

---

<div align="center">

Built in Go · Self-hosted · MIT License

</div>
