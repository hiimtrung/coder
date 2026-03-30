# CLI Command Reference

> Run `coder --help` or `coder <command> --help` for inline help at any time.

---

## Table of Contents

- [Project Setup](#project-setup)
- [AI Workflow Commands](#ai-workflow-commands)
- [Project Lifecycle Commands](#project-lifecycle-commands)
- [Project Utilities](#project-utilities)
- [Skill RAG](#skill-rag)
- [Semantic Memory](#semantic-memory)
- [System Commands](#system-commands)
- [Authentication Reference](#authentication-reference)

---

## Project Setup

### `coder install <profile>`

Scaffolds the **agent engine** (workflows, rules, agent definitions) into the current project.

```bash
coder install fullstack          # backend + frontend profiles
coder install be                 # backend only
coder install fe                 # frontend only
coder install all                # every available profile
coder install global be          # install globally for the current user
```

> Skills (NestJS, Go, Java, …) live in the vector DB and are queried via `coder skill search`. This command installs only the local `.agents/` scaffolding that tells agents *how* to use those skills.

**Flags**

| Flag | Description |
|------|-------------|
| `-t, --target <dir>` | Target directory (default: `.`) |
| `-f, --force` | Overwrite existing files |
| `--dry-run` | Preview changes without writing |

---

### `coder update [profile]`

Re-syncs local workflows and rules, then triggers skill ingestion to keep the vector DB current.

```bash
coder update             # update everything
coder update be          # update backend profile only
coder update global      # install/update agent files in global user directories
```

---

### `coder list [profile]`

```bash
coder list               # summary of all profiles
coder list be            # skills inside the backend profile
```

---

## AI Workflow Commands

These commands work on any project without any prior setup. They all inject memory + skill context automatically.

---

### `coder chat`

Interactive Q&A with memory + skill context injection. Supports multi-turn sessions.

```bash
coder chat                                     # interactive REPL
coder chat "explain the auth middleware"       # single question
coder chat --file src/auth.ts "review this"   # include file in context
coder chat --session my-session               # continue a named session
coder chat --resume                            # resume last session
coder chat --no-memory                         # skip memory injection
```

**Flags**

| Flag | Description |
|------|-------------|
| `--file <path>` | Include file content in the prompt |
| `--session <id>` | Use or create a named session |
| `--resume` | Resume the most recent session |
| `--no-memory` | Skip memory + skill context injection |

---

### `coder review`

Structured AI code review of the current git diff, specific files, or a GitHub PR.

```bash
coder review                        # review git diff (staged + unstaged)
coder review --file src/auth.ts     # review specific file
coder review --pr 42                # review GitHub PR #42
coder review --focus security       # focus: security | performance | style
coder review --format json          # machine-readable output
```

**Flags**

| Flag | Description |
|------|-------------|
| `--file <path>` | Review a specific file |
| `--pr <N>` | Review a GitHub PR by number |
| `--focus <area>` | Focus area: `security`, `performance`, `style` |
| `--format json` | Output as JSON |

---

### `coder debug`

Root cause analysis for errors, logs, or stack traces.

```bash
coder debug "panic: nil pointer dereference at auth.go:45"
coder debug --file error.log                     # analyse a log file
coder debug --context src/auth.go "JWT error"    # include source context
coder debug --diff HEAD~1                         # debug a regression
coder debug --interactive                         # REPL for multi-turn investigation
```

**Flags**

| Flag | Description |
|------|-------------|
| `--file <path>` | Analyse a log file |
| `--context <path>` | Include source file for context |
| `--diff <ref>` | Compare git diff against ref for regression debugging |
| `--interactive` | Multi-turn REPL mode |

---

### `coder plan`

3-stage planning workflow: Q&A → research → PLAN.md. For smaller tasks or single features.

```bash
coder plan "add rate limiting to the API"       # interactive Q&A → PLAN.md
coder plan --auto "add JWT refresh tokens"      # skip Q&A, generate directly
coder plan --prd path/to/feature.md             # generate plan from PRD file
coder plan --list                               # list plans in .coder/plans/
```

Output: `.coder/plans/PLAN-<slug>-<date>.md`

**Flags**

| Flag | Description |
|------|-------------|
| `--auto` | Skip Q&A, generate plan directly |
| `--prd <path>` | Read feature description from PRD file |
| `--list` | List existing plans |

---

### `coder qa`

UAT verification workflow — walks through acceptance criteria from a PLAN.md.

```bash
coder qa --plan .coder/plans/PLAN-auth-2026.md    # verify plan criteria
coder qa --resume                                   # resume interrupted QA session
coder qa --list                                     # list QA sessions
coder qa --report                                   # export QA report
```

**Flags**

| Flag | Description |
|------|-------------|
| `--plan <path>` | Path to PLAN.md to verify against |
| `--resume` | Resume the most recent QA session |
| `--list` | List all QA sessions |
| `--report` | Export QA report |

---

### `coder session`

Save and restore working context across sessions.

```bash
coder session save                      # save current task + next steps
coder session save --name "auth-jwt"    # save with a name
coder session resume                    # restore last saved session
coder session list                      # list all sessions
coder session export --format md        # export to markdown
```

---

### `coder workflow`

Auto-chains: `plan → review → implement hints → qa → fix → done`.

```bash
coder workflow "add email verification"    # full auto-chain
coder workflow --prd feature.md            # start from PRD
coder workflow --dry-run "add search"      # preview steps only
coder workflow --resume                    # continue interrupted workflow
coder workflow --list                      # list past workflows
coder workflow --steps plan,review,qa      # run specific steps only
```

**Flags**

| Flag | Description |
|------|-------------|
| `--prd <path>` | Start from a PRD file |
| `--dry-run` | Preview steps without executing |
| `--resume` | Continue an interrupted workflow |
| `--list` | List past workflows |
| `--steps <list>` | Comma-separated steps: `plan,review,qa` |

---

## Project Lifecycle Commands

These commands implement a structured delivery loop tracked in `.coder/STATE.md`.
Always run `coder new-project` first to initialize the project.

The lifecycle loop:
```
new-project → map-codebase → discuss-phase N → plan-phase N → execute-phase N → ship N → milestone complete N → milestone next → discuss-phase N+1 ...
```

Use `coder next` at any point to see the recommended next command.

---

### `coder new-project`

Initialize a project with AI-guided requirements and roadmap generation.

```bash
coder new-project "build a CLI task manager in Go"
coder new-project --auto "REST API for a blog with auth"    # skip Q&A
coder new-project --prd path/to/feature.md                  # load from PRD
coder new-project --resume                                   # continue interrupted init
```

**What it creates:**

| File | Content |
|------|---------|
| `.coder/PROJECT.md` | Project name, description, tech stack |
| `.coder/REQUIREMENTS.md` | Full requirements list |
| `.coder/ROADMAP.md` | Phases with goals |
| `.coder/STATE.md` | `current_phase=1, step=discuss` |

**Flags**

| Flag | Description |
|------|-------------|
| `--auto` | Skip interactive Q&A |
| `--prd <path>` | Load project description from PRD file |
| `--resume` | Continue an interrupted init session |

---

### `coder map-codebase`

Analyse an existing codebase and generate structured documentation in `.coder/codebase/`.

```bash
coder map-codebase              # analyse whole codebase
coder map-codebase auth         # focus on a specific area
coder map-codebase --refresh    # force re-analysis
```

**What it creates (4 passes):**

| Pass | Output |
|------|--------|
| `tech` | `STACK.md` + `INTEGRATIONS.md` |
| `arch` | `ARCHITECTURE.md` + `STRUCTURE.md` |
| `quality` | `CONVENTIONS.md` + `TESTING.md` |
| `concerns` | `CONCERNS.md` |

Auto-commits output as `chore: map codebase`.

**Flags**

| Flag | Description |
|------|-------------|
| `[area]` | Optional focus area (e.g. `auth`, `payments`) |
| `--refresh` | Force re-analysis even if `.coder/codebase/` exists |

---

### `coder discuss-phase N`

Identify gray areas for phase N via Q&A → CONTEXT.md.

```bash
coder discuss-phase 1               # interactive Q&A
coder discuss-phase 2 --auto        # AI picks defaults, no questions
coder discuss-phase 3 --batch       # answer all questions at once
```

Requires `.coder/ROADMAP.md` and `.coder/PROJECT.md` to exist.

Output: `.coder/phases/NN-CONTEXT.md` — decisions, rationale, and implementation impact.

**Flags**

| Flag | Description |
|------|-------------|
| `--auto` | AI selects recommended defaults, no Q&A |
| `--batch` | Collect all answers in one prompt |

---

### `coder plan-phase N`

Research + generate XML implementation plans + verification loop for phase N.

```bash
coder plan-phase 1                            # full flow
coder plan-phase 2 --skip-research            # skip if RESEARCH.md exists
coder plan-phase 3 --skip-verify              # skip verification loop
coder plan-phase 4 --gaps                     # re-plan only items flagged in VERIFICATION.md
coder plan-phase 5 --prd path/to/feature.md  # use PRD instead of CONTEXT.md
```

Requires `NN-CONTEXT.md` to exist (or `--prd`).

**3-step flow:**
1. **Research** → `NN-RESEARCH.md` (approach, libraries, pitfalls)
2. **Planning** → `NN-01-PLAN.md`, `NN-02-PLAN.md` … (XML `<plan>` blocks)
3. **Verification loop** → `NN-VERIFICATION.md` (up to 3 iterations: PASS or FAIL + auto-fix)

**Flags**

| Flag | Description |
|------|-------------|
| `--skip-research` | Skip if `NN-RESEARCH.md` already exists |
| `--skip-verify` | Skip verification loop |
| `--gaps` | Re-plan only items flagged in VERIFICATION.md |
| `--prd <path>` | Use PRD file instead of CONTEXT.md |

---

### `coder execute-phase N`

Execute phase N plans with atomic git commits per task.

```bash
coder execute-phase 1                          # execute all plans
coder execute-phase 2 --interactive            # pause for review between plans
coder execute-phase 3 --gaps-only              # skip plans with existing SUMMARY.md
coder execute-phase 4 --plan "01-02"           # execute only a specific plan
```

**What it does:**
- Groups plans into dependency waves (plans with `<dependencies>none</dependencies>` run first)
- For each task: streams AI execution, then `git add -A && git commit`
- Commit format: `feat(01-01): task name`, `test(01-01): task name`, etc.
- After all plans: runs a verifier pass → `NN-VERIFICATION.md`

**Flags**

| Flag | Description |
|------|-------------|
| `--interactive` | Pause for user checkpoint between plans |
| `--gaps-only` | Skip plans that already have a `SUMMARY.md` |
| `--plan <id>` | Execute only a specific plan (e.g. `01-02`) |

---

### `coder ship [N]`

Create a pull request for phase N using `gh` CLI with an AI-generated PR body.

```bash
coder ship                                          # ship current phase
coder ship 3                                        # ship phase 3 explicitly
coder ship --draft                                  # open as draft PR
coder ship --base develop                           # target branch
coder ship --title "feat: phase 3 auth module"     # custom PR title
coder ship --skip-push                              # don't run git push first
```

Reads phase summaries + verification → AI generates Summary / Changes / Test plan / Notes.

**Flags**

| Flag | Description |
|------|-------------|
| `[N]` | Phase number (default: current phase from STATE.md) |
| `--draft` | Open as draft PR |
| `--base <branch>` | Base branch (default: `main`) |
| `--title <str>` | Custom PR title |
| `--skip-push` | Skip `git push` before PR creation |

---

### `coder progress`

Show current project progress — phases, step, decisions, blockers, PRs, backlog.

```bash
coder progress              # full status view
coder progress --short      # one-line summary: phase=1 step=plan last=...
```

---

### `coder next`

Print the next recommended command based on `STATE.md`.

```bash
coder next
# → coder execute-phase 1   (when step=plan)
```

| STATE.md step | Output |
|---------------|--------|
| (empty) | `coder map-codebase` |
| `discuss` | `coder plan-phase N` |
| `plan` | `coder execute-phase N` |
| `execute` | `coder execute-phase N --gaps-only` |
| `qa` | `coder qa --phase N` |
| `ship` | `coder milestone complete N` |
| `done` | `coder milestone next` |

---

### `coder milestone <action>`

Manage the phase lifecycle: audit, complete, archive, and advance.

```bash
coder milestone audit 3       # show completion checklist for phase 3
coder milestone complete 3    # mark phase 3 done → STATE.md: step=done
coder milestone archive 2     # move phase 2 files to .coder/archive/02/
coder milestone next          # advance to next phase → step=discuss
```

**Actions:**

| Action | What it does |
|--------|-------------|
| `audit [N]` | Shows CONTEXT.md, VERIFICATION.md, plans, summaries, PR status |
| `complete [N]` | Records completion in STATE.md |
| `archive [N]` | Moves `.coder/phases/NN-*` to `.coder/archive/NN/` |
| `next` | Advances `current_phase` and resets `step=discuss` |

---

## Project Utilities

### `coder todo`

Manage the project backlog stored in `.coder/STATE.md`.

```bash
coder todo                              # list backlog items
coder todo add "investigate rate limiting"
coder todo done "rate limiting"         # remove by substring match
coder todo clear                        # clear all items
```

---

### `coder stats`

Show project statistics.

```bash
coder stats
# Output:
#   Phases     : 5 total, 2 done
#   Plans      : 8
#   Summaries  : 6 / 8
#   Git commits: 47
#   Source files: 132
#   Current phase: 3  step: execute
```

---

### `coder health`

Check project health — missing artifacts, blockers, stale state.

```bash
coder health
# Output:
#   ✓ Project initialized
#   ✓ Roadmap: 5 phases
#   ⚠ 1 blocker(s) recorded: waiting for API credentials
```

---

### `coder note`

Record a decision, blocker, or backlog item to `.coder/STATE.md`.

```bash
coder note "decided to use JWT with refresh tokens"       # decision
coder note --blocker "waiting for API credentials"        # blocker
coder note --backlog "add rate limiting in phase 4"       # backlog
```

**Flags**

| Flag | Description |
|------|-------------|
| `--blocker` | Record as a blocker (shows in `coder health`) |
| `--backlog` | Record as a backlog item (shows in `coder todo`) |

---

### `coder do`

Run a one-off AI task with full project context injected (PROJECT.md + REQUIREMENTS.md + current state).

```bash
coder do "write unit tests for the auth service"
coder do "refactor the payment module to use dependency injection"
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

### `coder skill list`

Lists all skills currently in the vector DB with chunk counts and metadata.

### `coder skill info <name>`

Detailed metadata and chunk breakdown for a single skill.

### `coder skill delete <name>`

Removes a skill and all its chunks from the DB.

---

## Semantic Memory

### `coder memory store <title> <content>`

Stores a knowledge snippet in the semantic memory. Supports lifecycle metadata for versioned memories, validity windows, and superseding older active entries.

```bash
coder memory store "Auth pattern" "We use SHA-256 hashed tokens in coder_clients table"
coder memory store "DB migration rule" "Always use reversible migrations" \
  --tags "database,migrations" --type rule
coder memory store "Auth decision" "Use rotating refresh tokens" \
  --type decision --replace-active --key "decision:auth-refresh"
```

**Flags**

| Flag | Description |
|------|-------------|
| `--tags <t1,t2>` | Comma-separated tags |
| `--type <type>` | `fact`, `rule`, `decision`, `pattern`, `event`, `document` |
| `--scope <scope>` | Memory scope |
| `--meta <json>` | Raw JSON metadata |
| `--status <status>` | Lifecycle status: `active`, `superseded`, `expired`, `archived`, `draft` |
| `--key <canonical_key>` | Stable key for multiple versions of the same memory |
| `--supersedes <id>` | Explicitly mark which memory/version this entry replaces |
| `--valid-from <RFC3339>` | Validity window start |
| `--valid-until <RFC3339>` | Validity window end |
| `--verified-at <RFC3339>` | Last verification timestamp |
| `--verified-by <actor>` | Actor or workflow that verified this memory |
| `--confidence <0..1>` | Confidence score used during reranking |
| `--source <ref>` | Source reference such as PR, commit, or doc |
| `--replace-active` | Supersede the current active memory with the same canonical key |

### `coder memory search <query>`

Hybrid semantic + full-text search across all stored memories. By default, search is lifecycle-aware: it prefers active memories, applies validity-window filtering, collapses multiple versions of the same canonical key, and emits a conflict summary when multiple active versions disagree materially.

```bash
coder memory search "how do we handle authentication"
coder memory search "postgres connection" --limit 3
coder memory search "refresh token behavior" --include-stale --history
coder memory search "auth decision" --key "decision:auth-refresh" --as-of 2026-03-27T00:00:00Z
```

**Flags**

| Flag | Description |
|------|-------------|
| `--limit <n>` | Number of results to return |
| `--scope <scope>` | Memory scope |
| `--type <type>` | Filter by memory type |
| `--meta <json>` | Additional JSON metadata filters |
| `--status <status>` | Lifecycle status filter |
| `--key <canonical_key>` | Canonical key filter |
| `--as-of <RFC3339>` | Evaluate validity at a specific point in time |
| `--include-stale` | Include superseded, expired, or archived memories |
| `--history` | Return multiple versions for the same canonical key instead of collapsing to the best active hit |

### `coder memory verify <id>`

Refreshes verification metadata for a memory version group. This updates `last_verified_at` on all chunks in the target version and can also capture verifier identity, confidence, and source reference.

```bash
coder memory verify 7f9c4c1e
coder memory verify 7f9c4c1e --verified-by phase-3 --confidence 0.9 --source docs/memory_lifecycle_plan.md
```

**Flags**

| Flag | Description |
|------|-------------|
| `--verified-at <RFC3339>` | Verification timestamp, defaults to now |
| `--verified-by <actor>` | Actor or workflow verifying this memory |
| `--confidence <0..1>` | Updated confidence score |
| `--source <ref>` | Source reference used for verification |

### `coder memory supersede <id> <replacement-id>`

Marks one memory version group as superseded by another. The source version becomes `superseded`, the replacement becomes `active`, and the version chain is linked with `supersedes_id` / `superseded_by_id`.

```bash
coder memory supersede old-parent new-parent
```

### `coder memory audit`

Reports lifecycle issues in the current memory store so stale or conflicting memories can be resolved before they leak into retrieval.

```bash
coder memory audit
coder memory audit --scope backend --unverified-days 90
coder memory audit --json
```

**Flags**

| Flag | Description |
|------|-------------|
| `--scope <scope>` | Restrict the audit to a specific memory scope |
| `--unverified-days <n>` | Flag active memories not verified within this many days |
| `--json` | Print the full audit report as JSON |

### `coder memory list`

Shows the most recent memory entries, including lifecycle status.

```bash
coder memory list
coder memory list --limit 20 --offset 40
```

### `coder memory delete <id>`

Removes a single memory entry by ID.

---

## System Commands

### `coder login`

Interactive wizard to connect the CLI to a coder-node instance and register as an authenticated client.

**What it asks:**
1. **Protocol** — `gRPC` (recommended) or `HTTP`
2. **Server URL** — e.g. `localhost:50051` (gRPC) or `192.168.1.10:8080` (HTTP)
3. **Authentication** — `y` if the server runs `--secure`; enter the bootstrap token

**On success:** an access token is saved to `~/.coder/config.json`. All future `coder memory` and `coder skill` calls attach the token automatically.

---

### `coder token show`

Display the current access token and client identity.

### `coder token rotate`

Rotate your access token (generates a new one, old one becomes invalid).

---

### `coder version`

Displays CLI version, Git commit hash, and build timestamp.

### `coder check-update`

Checks GitHub Releases for a newer version.

### `coder self-update`

Downloads and replaces the current binary with the latest release.

---

## Authentication Reference

| Scenario | Action |
|----------|--------|
| Open-mode server | No token needed — leave auth prompt as `N` |
| Secure-mode server (first time) | `coder login` → `y` → enter bootstrap token |
| Lost access token | Re-run `coder login` → `y` → enter bootstrap token again |
| Bootstrap token lost | Admin must clear `coder_server_config` in DB and restart |
| Switch protocol (gRPC ↔ HTTP) | Re-run `coder login`, answer `N` to auth |
| Rotate token | `coder token rotate` |
