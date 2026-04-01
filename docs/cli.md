# CLI Command Reference

This document describes the commands currently wired in [`cmd/coder/main.go`](/Users/trungtran/ai-agents/coder/cmd/coder/main.go).

If a command appears in roadmap documents but not here, treat it as planned or historical, not implemented.

Use `coder --help` or `coder <command> --help` for the exact inline flags.

---

## Implemented Top-Level Commands

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

---

## Project Setup

### `coder install <profile>`

Install a profile into the current project.

Examples:

```bash
coder install fullstack
coder install be
coder install global fullstack
coder install be --dry-run
```

Use this to scaffold `.agents/`, workflow files, and agent guidance into a repo.

### `coder update [profile]`

Refresh an existing install.

Examples:

```bash
coder update
coder update be
coder update global
```

Project updates may also trigger local skill ingestion to keep the skill database aligned with bundled assets.

### `coder list [profile]`

Inspect available profiles or show details for a specific one.

Examples:

```bash
coder list
coder list be
```

### `coder remove global`

Remove globally installed coder-managed files.

Example:

```bash
coder remove global
```

---

## Connection And Auth

### `coder login`

Configure the CLI to talk to `coder-node`.

Examples:

```bash
coder login
```

The resulting config is stored in `~/.coder/config.json`.

### `coder token`

Manage the saved access token.

Subcommands:

```text
show
rotate
```

Examples:

```bash
coder token show
coder token rotate
```

### `coder version`

Print CLI build metadata.

### `coder check-update`

Check GitHub releases for a newer CLI version.

### `coder self-update`

Download and replace the local CLI binary with the latest release.

---

## Skill Commands

### `coder skill search <query>`

Search ingested skills using hybrid semantic plus full-text retrieval.

Example:

```bash
coder skill search "error handling in golang" --limit 5
```

### `coder skill ingest`

Ingest skills into the skill database.

Supported sources:

```text
auto
local
github
```

Examples:

```bash
coder skill ingest
coder skill ingest --source local
coder skill ingest --source local --include-files
coder skill ingest --source github --repo hiimtrung/coder
```

### `coder skill list`

List ingested skills, with optional filtering.

Example:

```bash
coder skill list --category core
```

### `coder skill info <name>`

Show metadata and stored chunks for one skill.

### `coder skill delete <name>`

Delete one skill from the database.

### `coder skill cache`

Manage cached skill files.

Subcommands:

```text
pull
list
clear
```

Examples:

```bash
coder skill cache pull ui-ux-pro-max
coder skill cache pull --all
coder skill cache list
coder skill cache clear ui-ux-pro-max
```

### `coder skill index`

Generate `skills_index.json` from local `.agents/skills/`.

---

## Memory Commands

### `coder memory store <title> <content>`

Store semantic memory with lifecycle metadata.

Examples:

```bash
coder memory store "Go Interfaces" "Context on interfaces..." --tags "go,pattern"
coder memory store "Auth decision" "Use rotating refresh tokens" --type decision --replace-active
```

### `coder memory search <query>`

Search memory with lifecycle-aware filtering.

Supports machine-readable output for agent-safe context injection:

```text
--format text|json|raw
```

Examples:

```bash
coder memory search "auth middleware" --limit 5
coder memory search "token rotation" --status active
coder memory search "release process" --history
coder memory search "grpc auth" --format json
coder memory search "jwt rotation" --format raw
```

Every successful search also refreshes `.coder/active-memory.json` so the latest recalled memory context can be inspected locally.

### `coder memory recall <task>`

Re-recall memory for the current task and compute a decision diff against the current active memory set.

This command is intended for long-running agent work where memory must be refreshed without restarting the task.

Examples:

```bash
coder memory recall "grpc auth flow" --trigger execution --budget 5
coder memory recall "token rotation" --current auth-token,release-notes --format json
coder memory recall "migration incident" --format raw
```

The recall result reports:

- `keep`: memory that should remain active
- `add`: newly recalled memory
- `drop`: stale memory no longer needed in active context
- `coverage`: `strong`, `adequate`, `weak`, or `none`
- `conflicts`: recalled items that still represent active-memory conflicts

### `coder memory active`

Show the current active memory recall state stored in `.coder/active-memory.json`.

Examples:

```bash
coder memory active
coder memory active --format json
```

### `coder memory verify <id>`

Refresh verification metadata for a memory or version group.

Example:

```bash
coder memory verify 7f9c4c1e --verified-by phase-3 --confidence 0.9
```

### `coder memory supersede <id> <replacement-id>`

Mark one memory/version group as replaced by another.

### `coder memory audit`

Report lifecycle conflicts, expired active memories, and stale entries.

### `coder memory list`

List recent memory entries.

### `coder memory delete <id>`

Delete one memory entry.

### `coder memory compact`

Run memory maintenance such as duplicate cleanup and optional re-vectoring.

---

## Session And State

### `coder session`

Save and restore working context locally.

Subcommands:

```text
save
resume
list
show
delete
export
```

Examples:

```bash
coder session save "implementing JWT tokens"
coder session resume
coder session list
coder session show ses-123
coder session export ses-123 -o context.md
```

### `coder progress`

Show project state derived from `.coder/STATE.md` and `.coder/ROADMAP.md`.

### `coder next`

Print the next recommended command from the current project state.

`next` now stays within the currently implemented state-management surface, primarily `coder milestone audit`, `coder milestone complete`, and `coder milestone next`.

### `coder milestone`

Manage phase lifecycle for `.coder/`-based projects.

Actions:

```text
audit
complete
archive
next
```

Examples:

```bash
coder milestone audit 2
coder milestone complete 2
coder milestone archive 1
coder milestone next
```

---

## Current Scope

`coder` is currently a memory, skill, session, and project-state CLI.

These commands are often referenced in old docs or roadmap files but are not wired in the current binary:

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

Keep implementation docs aligned with [`cmd/coder/main.go`](/Users/trungtran/ai-agents/coder/cmd/coder/main.go) first, then treat roadmap docs as future design.
