# Coder CLI Guide

> Last updated: 2026-03-31
> Source of truth for command availability: `cmd/coder/main.go`

This guide covers the current working CLI, not the long-term roadmap.

For the detailed command list, use [docs/cli.md](/Users/trungtran/ai-agents/coder/docs/cli.md).

---

## What coder Is Today

`coder` currently provides:

- project installation and updates
- memory retrieval and storage
- skill retrieval and ingestion
- local session checkpointing
- `.coder/` project-state helpers such as `progress`, `next`, and `milestone`
- connection, auth, and self-update utilities

It does not currently expose built-in commands like `chat`, `review`, `debug`, `plan`, or `workflow` in the binary.

Those ideas still exist in roadmap documents, but they are not part of the current command surface.

---

## Quick Start

### 1. Start `coder-node`

Run the server with your normal local or team setup.

### 2. Connect the CLI

```bash
coder login
coder version
```

### 3. Install project guidance

```bash
coder install fullstack
```

### 4. Ingest local skills

```bash
coder skill ingest --source local
```

### 5. Use the knowledge loop with your agent

```bash
coder skill search "auth middleware"
coder memory search "token rotation"
# your AI agent reasons and writes code
coder memory store "Auth decision" "Use rotating refresh tokens" --tags "auth,backend"
```

---

## Common Workflows

### Daily knowledge workflow

```bash
coder skill search "<topic>"
coder memory search "<topic>"
```

Use this before a non-trivial task so your AI agent starts from team patterns instead of a blank slate.

### Save work context

```bash
coder session save "implementing release flow cleanup"
coder session resume
```

Use sessions when switching tasks or before compacting context.

### Check project state

```bash
coder progress
coder next
coder milestone audit
```

These commands are for `.coder/`-managed projects and state files.

### Maintain the local install

```bash
coder update
coder self-update
coder check-update
```

---

## Important Scope Note

The following commands are mentioned in some older documents but are not implemented in the current binary:

```text
chat
review
debug
plan
qa
workflow
new-project
map-codebase
discuss-phase
plan-phase
execute-phase
ship
todo
stats
health
note
do
```

Treat them as roadmap concepts unless they are added to `cmd/coder/main.go`.
