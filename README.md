<div align="center">

```
   ██████╗ ██████╗ ██████╗ ███████╗██████╗
  ██╔════╝██╔═══██╗██╔══██╗██╔════╝██╔══██╗
  ██║     ██║   ██║██║  ██║█████╗  ██████╔╝
  ██║     ██║   ██║██║  ██║██╔══╝  ██╔══██╗
  ╚██████╗╚██████╔╝██████╔╝███████╗██║  ██║
   ╚═════╝ ╚═════╝ ╚═════╝ ╚══════╝╚═╝  ╚═╝
```

**Universal engineering intelligence for AI agents.**
Distribute skills, enforce architecture, and preserve memory — across every project and every developer.

[![Build & Release](https://github.com/hiimtrung/coder/actions/workflows/release.yml/badge.svg)](https://github.com/hiimtrung/coder/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/hiimtrung/coder)](https://goreportcard.com/report/github.com/hiimtrung/coder)
[![Latest Release](https://img.shields.io/github/v/release/hiimtrung/coder)](https://github.com/hiimtrung/coder/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

</div>

---

## What is coder?

Most AI agents operate in a vacuum — no memory, no standards, no institutional knowledge. **coder** fixes that.

It gives every agent in your team access to the same centralized brain: a vector-powered knowledge base holding your architecture rules, your senior engineers' patterns, and the project history that made those decisions meaningful.

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

| | |
|---|---|
| **Hybrid RAG Search** | pgvector cosine similarity fused with full-text search via Reciprocal Rank Fusion |
| **Semantic Memory** | Store and retrieve cross-project decisions, patterns, and post-mortems |
| **20+ Built-in Skills** | NestJS, Go, Java, Rust, Python, React, architecture, testing, and more |
| **Dual Transport** | gRPC (performance) + HTTP (compatibility) — both support Bearer token auth |
| **Secure Mode** | Bootstrap token registration, SHA-256 hashed storage, per-client access tokens |
| **Activity Tracking** | Fire-and-forget telemetry: command + repo + branch, logged per developer |
| **Self-Hosted** | One Docker command — Postgres + pgvector + Ollama + coder-node |
| **Single Binary** | ~7MB CLI, zero runtime dependencies, cross-platform |

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
> All subsequent `coder memory` / `coder skill` calls carry a `Bearer` token automatically — over both gRPC and HTTP.

### 3 — Connect

```bash
coder login
# Prompts for protocol, URL, and auth token (if secure mode)
```

### 4 — Apply to a project

```bash
cd my-project
coder install fullstack        # scaffold .agents/ into the project
coder skill ingest --source local   # load 20+ built-in skills into the vector DB
```

### 5 — Let your agents use it

```bash
coder skill search "NestJS error handling"
coder memory store "Auth pattern" "We use SHA-256 hashed tokens stored in coder_clients"
coder memory search "how do we handle authentication"
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
                              │  Hybrid search (RRF) │
                              │  Skill ingestor      │
                              │  Memory manager      │
                              └──────────┬───────────┘
                                         │
                              ┌──────────┴──────────┐
                              │  PostgreSQL + pgvec │
                              │  Ollama embeddings  │
                              └─────────────────────┘
```

The **3-Gate Loop** enforced by agent workflows:

1. **GATE 1 — Skill retrieval** `coder skill search "<topic>"` — retrieves architecture rules and best practices from the vector DB before any coding starts
2. **GATE 2 — Memory retrieval** `coder memory search "<topic>"` — loads project-specific history and past decisions
3. **GATE 3 — Knowledge capture** `coder memory store "<title>" "<content>"` — persists new patterns so the next agent (or the same agent tomorrow) benefits

---

## Key Commands

```bash
coder login                              # connect to coder-node (handles auth)
coder install fullstack                  # scaffold agent engine into a project
coder skill ingest --source local        # load built-in skills into vector DB
coder skill ingest --source github \
  --repo your-org/your-skills           # ingest skills from a GitHub repo
coder skill search "error handling"     # semantic search across skills
coder memory store "Title" "Content"    # save a knowledge snippet
coder memory search "auth pattern"      # retrieve relevant memory
coder self-update                        # update the CLI binary
```

---

## Authentication (Secure Mode)

When `coder-node` runs with `--secure`, every API call (gRPC and HTTP) requires a valid Bearer token.

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
- Raw token generated once with `crypto/rand`, **never stored**
- Only the SHA-256 hash lives in the database
- Bootstrap token shown **once** in server logs — regenerate by clearing `coder_server_config`
- Access tokens sent via `authorization` metadata on gRPC; `Authorization: Bearer` header on HTTP

---

## Documentation

| Document | Description |
|----------|-------------|
| [Installation](docs/installation.md) | CLI + coder-node setup, secure mode, env vars |
| [CLI Reference](docs/cli.md) | All commands and flags |
| [Architecture](docs/architecture.md) | System design, data flows, Mermaid diagrams |
| [Skill System](docs/skill_system.md) | How the vector RAG works |
| [Memory System](docs/memory_system.md) | Semantic memory internals |
| [Development](docs/development.md) | Building from source, release process |
| [Changelog](CHANGELOG.md) | Release history |

---

## Contributing

Issues and pull requests are welcome. See the [Development Guide](docs/development.md) for build instructions, project structure, and the release process.

---

<div align="center">

Built in Go · Self-hosted · MIT License

</div>
