# CLI Command Reference

> Run `coder --help` or `coder <command> --help` for inline help at any time.

---

## Project setup

### `coder install <profile>`

Scaffolds the **agent engine** (workflows, rules, agent definitions) into the current project.

```bash
coder install fullstack          # backend + frontend profiles
coder install be                 # backend only
coder install fe                 # frontend only
coder install all                # every available profile
```

> Skills (NestJS, Go, Java, …) are **not** installed as local files. They live in the vector DB on coder-node and are queried via `coder skill search`. This command installs only the local `.agents/` scaffolding that tells agents *how* to use those skills.

**Flags**

| Flag | Description |
|------|-------------|
| `-t, --target <dir>` | Target directory (default: `.`) |
| `-f, --force` | Overwrite existing files |
| `--dry-run` | Preview changes without writing |

---

### `coder update [profile]`

Re-syncs local workflows and rules, then triggers a skill ingestion to keep the vector DB current.

```bash
coder update             # update everything
coder update be          # update backend profile only
coder update global      # install/update agent files in global user directories
                         # (~/.claude/agents/, ~/.config/github-copilot/, etc.)
```

---

### `coder list [profile]`

```bash
coder list               # summary of all profiles
coder list be            # skills inside the backend profile
```

---

## Skill RAG

### `coder skill ingest`

Imports skill knowledge into the vector database.

```bash
coder skill ingest --source local                        # 20+ built-in skills
coder skill ingest --source github --repo org/repo       # from a GitHub repo
coder skill ingest --source local --filter nestjs,go     # specific skills only
```

### `coder skill search <query>`

Hybrid semantic + full-text search (RRF fusion) across all ingested skills.

```bash
coder skill search "NestJS error handling"
coder skill search "database migration patterns" --limit 10
```

**Flags**: `--limit <n>` (default: 5)

### `coder skill list`

Lists all skills currently in the vector DB with chunk counts and metadata.

### `coder skill info <name>`

Detailed metadata and chunk breakdown for a single skill.

### `coder skill delete <name>`

Removes a skill and all its chunks from the DB.

---

## Semantic memory

### `coder memory store <title> <content>`

Stores a knowledge snippet in the semantic memory.

```bash
coder memory store "Auth pattern" "We use SHA-256 hashed tokens in coder_clients table"
coder memory store "DB migration rule" "Always use reversible migrations" \
  --tags "database,migrations" --type rule
```

**Flags**

| Flag | Description |
|------|-------------|
| `--tags <t1,t2>` | Comma-separated tags for categorisation |
| `--type <type>` | `rule`, `pattern`, `fact`, `decision`, … |

### `coder memory search <query>`

Hybrid semantic + full-text search across all stored memories.

```bash
coder memory search "how do we handle authentication"
coder memory search "postgres connection" --limit 3
```

### `coder memory list`

Shows the most recent memory entries.

```bash
coder memory list
coder memory list --limit 20 --offset 40
```

### `coder memory delete <id>`

Removes a single memory entry by ID.

---

## System commands

### `coder login`

Interactive wizard to connect the CLI to a coder-node instance and (optionally) register as an authenticated client.

**What it asks:**

1. **Protocol** — `gRPC` (recommended, lower overhead) or `HTTP` (required for initial registration on secure-mode servers)
2. **Server URL** — e.g. `localhost:50051` (gRPC) or `192.168.1.10:8080` (HTTP)
3. **Authentication** — `y` if the server runs `--secure`; enter the bootstrap token provided by your admin

**On success:** an access token is saved to `~/.coder/config.json`. All future `coder memory` and `coder skill` commands attach the token automatically — injected into gRPC metadata (`authorization: Bearer …`) and HTTP headers (`Authorization: Bearer …`).

**Error recovery:** if the connection test fails, `coder login` presents a menu:
```
  1) Retry with a different URL / protocol
  2) Re-enter authentication token
  3) Skip verification and continue anyway
  4) Exit
```

> **Tip**: register using HTTP on port 8080 first (token exchange is HTTP-only), then re-run `coder login` and switch to gRPC for daily use.

---

### `coder version`

Displays CLI version, Git commit hash, and build timestamp.

### `coder check-update`

Checks GitHub Releases for a newer version.

### `coder self-update`

Downloads and replaces the current binary with the latest release.

---

## Authentication reference

| Scenario | Action |
|----------|--------|
| Open-mode server | No token needed — leave auth prompt as `N` |
| Secure-mode server (first time) | `coder login` → `y` → enter bootstrap token |
| Lost access token | Re-run `coder login` → `y` → enter bootstrap token again (creates a new client record) |
| Bootstrap token lost | Admin must clear `coder_server_config` in DB and restart the node |
| Switch protocol (gRPC ↔ HTTP) | Re-run `coder login`, answer `N` to auth (token is already saved) |
