---
name: coder-cli
description: Expert knowledge of the current coder CLI and coder-node architecture. Use for questions about implemented commands, memory and skill flows, auth, release flow, and .coder state handling. Distinguishes current commands from roadmap-only ideas.
---

# Skill: coder CLI Architecture

## Overview

`coder` is currently a Go CLI focused on:

- memory retrieval and storage
- skill retrieval and ingestion
- local session checkpointing
- project state helpers around `.coder/STATE.md`
- install, auth, update, and release-adjacent maintenance

It is **not** currently a built-in LLM chat or workflow runner. Any document that describes `coder chat`, `coder review`, `coder debug`, or phase-execution commands should be treated as roadmap unless those commands are registered in [`cmd/coder/main.go`](/Users/trungtran/ai-agents/coder/cmd/coder/main.go).

`coder-node` is the backend service. It owns memory, skills, auth, and storage. The CLI is a thin client except for some local session and `.coder/` state commands.

---

## Current Top-Level Commands

The current binary registers only these commands:

```text
install
update
list
version
check-update
self-update
login
token
skill
memory
remove
session
progress
next
milestone
```

Always anchor command explanations to [`cmd/coder/main.go`](/Users/trungtran/ai-agents/coder/cmd/coder/main.go).

---

## Command Groups

### Setup And Maintenance

| Command               | Purpose                                          |
| --------------------- | ------------------------------------------------ |
| `coder install`       | Install project or global profiles               |
| `coder update`        | Refresh installed profile files                  |
| `coder list`          | Show available profiles                          |
| `coder remove global` | Remove globally installed coder-managed files    |
| `coder version`       | Print CLI build information                      |
| `coder check-update`  | Check GitHub for a newer release                 |
| `coder self-update`   | Replace the local binary with the latest release |

### Connection And Auth

| Command              | Purpose                                  |
| -------------------- | ---------------------------------------- |
| `coder login`        | Configure connection to `coder-node`     |
| `coder token show`   | Show current auth identity/token summary |
| `coder token rotate` | Rotate saved access token                |

### Knowledge Commands

| Command                  | Purpose                                              |
| ------------------------ | ---------------------------------------------------- |
| `coder skill search`     | Search ingested skills                               |
| `coder skill ingest`     | Ingest local, embedded, or GitHub skills             |
| `coder skill list`       | List stored skills                                   |
| `coder skill info`       | Show a skill and its chunks                          |
| `coder skill delete`     | Remove a skill                                       |
| `coder skill cache`      | Pull/list/clear cached skill files                   |
| `coder skill index`      | Generate local `skills_index.json`                   |
| `coder memory store`     | Store memory with lifecycle metadata                 |
| `coder memory search`    | Search lifecycle-aware memory                        |
| `coder memory recall`    | Re-recall memory and compute keep/add/drop decisions |
| `coder memory active`    | Inspect the current active memory recall state       |
| `coder memory verify`    | Refresh verification metadata                        |
| `coder memory supersede` | Link old and replacement versions                    |
| `coder memory audit`     | Report lifecycle issues                              |
| `coder memory list`      | List recent memories                                 |
| `coder memory delete`    | Delete one memory                                    |
| `coder memory compact`   | Compact and optionally re-vector memory              |

### Local Context And Project State

| Command                                             | Purpose                                              |
| --------------------------------------------------- | ---------------------------------------------------- |
| `coder session save/resume/list/show/delete/export` | Manage local working sessions                        |
| `coder progress`                                    | Read `.coder/STATE.md` and `.coder/ROADMAP.md`       |
| `coder next`                                        | Suggest the next workflow command from project state |
| `coder milestone`                                   | Audit, complete, archive, or advance a phase         |

---

## What Is Roadmap Only

These commands are frequently referenced in older docs but are not currently wired in the binary:

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

When explaining them:

- describe them as roadmap or historical design only
- do not imply they are callable today
- prefer the current `memory`, `skill`, `session`, `progress`, `next`, and `milestone` commands instead

---

## Runtime Architecture

### CLI side

- `cmd/coder/main.go`: top-level dispatch
- `cmd/coder/config.go`: loads `~/.coder/config.json`
- `cmd/coder/cmd_skill.go`: skill command handlers
- `cmd/coder/cmd_memory.go`: memory command handlers
- `cmd/coder/cmd_session.go`: local session management
- `cmd/coder/cmd_progress.go`: `.coder/` state readers
- `cmd/coder/cmd_milestone.go`: phase lifecycle helpers

### Server side

- `cmd/coder-node/main.go`: wires dependencies and starts transports
- `internal/usecase/skill`: ingest and search logic
- `internal/usecase/memory`: store, search, lifecycle, compact
- `internal/usecase/auth`: registration, token, activity
- `internal/infra/postgres`: storage for memory, skill, auth
- `internal/transport/grpc` and `internal/transport/http`: remote APIs

---

## Important Flows

### Skill flow

`coder skill search` does not parse local files itself. It calls `coder-node`, which searches already ingested skill chunks.

`coder skill ingest`:

1. reads skills from local FS, embedded FS, or GitHub
2. sends `SKILL.md` plus rule files to the server
3. server parses sections, optionally embeds them, and stores chunks

### Memory flow

`coder memory store/search/recall/verify/supersede/audit` are lifecycle-aware and operate through the server unless local Postgres mode is configured.
`coder memory recall` now uses a shared recall contract at the usecase and transport layers rather than a CLI-only decision loop.

`coder memory search` and `coder memory recall` also refresh the local `.coder/active-memory.json` snapshot so the latest recalled context can be inspected with `coder memory active`.
Both commands also update `.coder/context-state.json` so active skills and memory can be recovered together.

### Session flow

`coder session` is local-only. It writes to:

```text
.coder/session.md
.coder/sessions/*.md
```

### Project-state flow

`coder progress`, `coder next`, and `coder milestone` read or mutate:

```text
.coder/STATE.md
.coder/ROADMAP.md
.coder/phases/
.coder/archive/
```

`coder next` may recommend roadmap commands that are not implemented yet. It reflects intended workflow state, not guaranteed command availability.

---

## Guidance For Agents

When asked about coder commands:

1. check `cmd/coder/main.go` first
2. prefer implemented command behavior over old docs
3. call out explicitly when a document is describing roadmap rather than current behavior
4. do not tell users that `coder` itself runs LLM chat flows unless the code actually wires those commands
