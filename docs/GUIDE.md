# Coder CLI — Complete Usage Guide

> Last updated: 2026-03-23
> Version: v0.4.5+

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Quick Start](#quick-start)
3. [Two Usage Modes](#two-usage-modes)
4. [Mode A — Quick AI Workflows](#mode-a--quick-ai-workflows)
5. [Mode B — Full Project Lifecycle](#mode-b--full-project-lifecycle)
6. [Project Utilities](#project-utilities)
7. [Directory Structure Reference](#directory-structure-reference)
8. [STATE.md Format](#statemd-format)
9. [XML Plan Format](#xml-plan-format)
10. [Flags Quick Reference](#flags-quick-reference)
11. [Tips & Troubleshooting](#tips--troubleshooting)

---

## Prerequisites

```bash
# 1. coder-node must be running
coder-node start          # or however your team starts it

# 2. Connect coder CLI to coder-node
coder login               # prompts for URL (default: http://localhost:8080)

# 3. Verify connection
coder version             # prints version + server status

# 4. (Optional) Ingest local skills into vector DB
coder skill ingest --source local
```

---

## Quick Start

### New project (greenfield)

```bash
mkdir my-project && cd my-project
git init
coder new-project "build a REST API for a task manager"
```

### Existing codebase

```bash
cd my-existing-project
coder map-codebase        # analyse codebase → .coder/codebase/
coder chat "what is the main architectural concern here?"
```

---

## Two Usage Modes

| Mode | When to use | Commands |
|------|-------------|---------|
| **A — Quick AI** | Day-to-day: review, debug, Q&A | `chat`, `review`, `debug`, `plan`, `qa`, `session`, `workflow` |
| **B — Full Lifecycle** | Building a feature/project end-to-end | `new-project`, `map-codebase`, `discuss-phase`, `plan-phase`, `execute-phase`, `ship`, `milestone` |

Both modes share the same `coder-node` backend and `coder memory` / `coder skill` systems.

---

## Mode A — Quick AI Workflows

These commands work on any existing project without a `.coder/` directory.

### `coder chat`

Interactive Q&A with memory + skill context injection.

```bash
coder chat                                    # interactive REPL
coder chat "explain the auth middleware"      # single question
coder chat --file src/auth.ts "review this"  # include file in context
coder chat --session my-session              # continue a named session
coder chat --resume                           # resume last session
coder chat --no-memory                        # skip memory injection
```

### `coder review`

Structured code review of the current git diff, specific files, or a PR.

```bash
coder review                        # review git diff (staged + unstaged)
coder review --file src/auth.ts     # review specific file
coder review --pr 42                # review GitHub PR #42
coder review --focus security       # focus: security | performance | style
coder review --format json          # machine-readable output
```

### `coder debug`

Root cause analysis for errors, logs, or panics.

```bash
coder debug "panic: nil pointer dereference at auth.go:45"
coder debug --file error.log                    # analyse log file
coder debug --context src/auth.go "JWT error"  # include source context
coder debug --diff HEAD~1                       # debug regression from last commit
coder debug --interactive                       # REPL mode for multi-turn investigation
```

### `coder plan`

3-stage planning: Q&A → research → PLAN.md. For smaller tasks, not full phases.

```bash
coder plan "add rate limiting to the API"      # interactive Q&A + generate plan
coder plan --auto "add JWT refresh tokens"     # skip Q&A, generate directly
coder plan --prd path/to/feature.md            # generate plan from PRD file
coder plan --list                              # list existing plans in .coder/plans/
```

Output: `.coder/plans/PLAN-<slug>-<date>.md`

### `coder qa`

UAT verification workflow — walks through acceptance criteria from a PLAN.md.

```bash
coder qa --plan .coder/plans/PLAN-auth-2026.md    # verify plan criteria
coder qa --resume                                  # resume interrupted QA session
coder qa --list                                    # list QA sessions
coder qa --report                                  # export QA report
```

### `coder session`

Save and restore working context across sessions.

```bash
coder session save                    # save current task + next steps
coder session save --name "auth-jwt"  # save with a name
coder session resume                  # restore last saved session
coder session list                    # list all sessions
coder session export --format md      # export to markdown
```

### `coder workflow`

Auto-chains: `plan → review → implement hints → qa → fix → done`.

```bash
coder workflow "add email verification"    # full chain
coder workflow --prd feature.md            # start from PRD
coder workflow --dry-run "add search"      # preview steps only
coder workflow --resume                    # continue interrupted workflow
coder workflow --list                      # list past workflows
coder workflow --steps plan,review,qa      # run specific steps only
```

---

## Mode B — Full Project Lifecycle

This is the structured delivery loop. All state is tracked in `.coder/STATE.md`.
Every command updates STATE.md so `coder next` always knows where you are.

### The lifecycle loop

```
new-project ──► map-codebase
                    │
              discuss-phase N
                    │
               plan-phase N
                    │
             execute-phase N
                    │
                 ship N
                    │
           milestone complete N
                    │
             milestone next ──► discuss-phase N+1 ...
```

---

### `coder new-project` — Phase 9

Initialize a project with AI-guided requirements and roadmap.

```bash
coder new-project "build a CLI task manager in Go"
coder new-project --auto "REST API for a blog with auth"    # skip Q&A
coder new-project --prd path/to/feature.md                  # load from PRD
coder new-project --resume                                   # continue interrupted init
```

**Flow:**
1. Guards against re-initializing (checks `.coder/PROJECT.md`)
2. Deep interactive Q&A to understand scope, tech stack, constraints
3. Streams `REQUIREMENTS.md` → user approves `[Y/n/edit]`
4. Streams `ROADMAP.md` → user approves `[Y/n/edit]`
5. Writes: `.coder/PROJECT.md`, `.coder/REQUIREMENTS.md`, `.coder/ROADMAP.md`, `.coder/STATE.md`

**Output files:**
```
.coder/
  PROJECT.md        # project name, description, tech stack
  REQUIREMENTS.md   # full requirements list
  ROADMAP.md        # phases with goals
  STATE.md          # current_phase=1, step=discuss
```

---

### `coder map-codebase` — Phase 10

Analyse an existing codebase and generate structured documentation.

```bash
coder map-codebase              # analyse whole codebase
coder map-codebase auth         # focus on the auth area only
coder map-codebase --refresh    # force re-analysis even if .coder/codebase/ exists
```

**Flow:** 4 sequential AI analysis passes:

| Pass | Focus | Output file |
|------|-------|-------------|
| `tech` | Language, frameworks, libraries, versions | `STACK.md` + `INTEGRATIONS.md` |
| `arch` | Architecture pattern, layers, data flow | `ARCHITECTURE.md` + `STRUCTURE.md` |
| `quality` | Naming conventions, error handling, tests | `CONVENTIONS.md` + `TESTING.md` |
| `concerns` | Security, debt, missing tests, perf | `CONCERNS.md` |

**Output:** `.coder/codebase/*.md` — auto-committed to git as `chore: map codebase`

---

### `coder discuss-phase N` — Phase 11

Identify gray areas for phase N and capture decisions via Q&A → CONTEXT.md.

```bash
coder discuss-phase 1               # interactive one-by-one Q&A
coder discuss-phase 2 --auto        # AI picks defaults, no questions
coder discuss-phase 3 --batch       # ask all questions at once
```

**Flow:**
1. Loads ROADMAP.md to find phase N name
2. Streams 3-5 gray-area questions (API format, auth, error handling, etc.)
3. You answer interactively (or `--auto` mode skips Q&A)
4. Streams `CONTEXT.md` from your answers → user approves `[Y/n]`
5. Writes: `.coder/phases/NN-CONTEXT.md`
6. Updates STATE.md: `step=plan`

**Output:** `.coder/phases/01-CONTEXT.md` (numbered by phase)

**Example CONTEXT.md sections:**
```markdown
## Error Response Format
Decision: Use RFC 7807 problem+json
Rationale: Standard across industry, easier for client parsing
Impact: All endpoints must return Content-Type: application/problem+json

## Auth Strategy
Decision: JWT with 15min access + 7d refresh tokens
...
```

---

### `coder plan-phase N` — Phase 12

Research + generate XML implementation plans for phase N.

```bash
coder plan-phase 1                           # full flow: research → plans → verify
coder plan-phase 2 --skip-research           # skip if RESEARCH.md exists
coder plan-phase 3 --skip-verify             # skip verification loop
coder plan-phase 4 --gaps                    # re-plan only items flagged in VERIFICATION.md
coder plan-phase 5 --prd path/to/feature.md  # use PRD instead of CONTEXT.md
```

**Flow:**
1. Requires `NN-CONTEXT.md` to exist (or `--prd` to bypass)
2. **Step A — Research**: streams RESEARCH.md (approach, libraries, pitfalls)
3. **Step B — Planning**: generates 2-4 XML `<plan>` blocks covering all requirements
4. **Step C — Verification loop** (up to 3 iterations):
   - Checker: PASS or FAIL with specific issues
   - If FAIL: streams fixes and re-verifies
5. User confirms `[Y/n]`
6. Writes: `.coder/phases/NN-RR-PLAN.md` (one file per plan)
7. Updates STATE.md: `step=execute`

**Output files:**
```
.coder/phases/
  01-RESEARCH.md
  01-01-PLAN.md
  01-02-PLAN.md
  01-VERIFICATION.md
```

---

### `coder execute-phase N` — Phase 13

Execute phase N plans with atomic git commits per task.

```bash
coder execute-phase 1                          # execute all plans
coder execute-phase 2 --interactive            # pause for user checkpoint between plans
coder execute-phase 3 --gaps-only              # skip plans with existing SUMMARY.md
coder execute-phase 4 --plan "01-02"           # execute only a specific plan
```

**Flow:**
1. Loads all `NN-*-PLAN.md` files from `.coder/phases/`
2. Parses XML `<plan>` blocks with `parsePlanXML()`
3. Groups plans into dependency waves (plans with no deps go first)
4. For each plan in wave order:
   - Streams each `<task>` action through AI
   - `git add -A && git commit` after each task (conventional commits)
   - Writes `NN-planid-SUMMARY.md`
5. Verifier pass: checks all requirements are covered → writes `NN-VERIFICATION.md`
6. Updates STATE.md: `step=qa`

**Git commits per task:**
```
feat(01-01): implement JWT middleware
feat(01-01): add token refresh endpoint
test(01-01): write unit tests for auth service
feat(01-02): implement rate limiter
```

---

### `coder ship N` — Phase 14

Create a pull request for phase N using `gh` CLI.

```bash
coder ship                                        # ship current phase
coder ship 3                                      # ship phase 3 explicitly
coder ship --draft                                # open as draft PR
coder ship --base develop                         # use develop as base branch
coder ship --title "feat: phase 3 auth module"   # custom PR title
coder ship --skip-push                            # don't run git push first
```

**Flow:**
1. Resolves phase number (from arg or STATE.md)
2. `git push --set-upstream origin HEAD`
3. Collects all `NN-*-SUMMARY.md` files + `NN-VERIFICATION.md`
4. AI generates PR body (Summary / Changes / Test plan / Notes)
5. Shows title + base branch → user confirms `[Y/n]`
6. Runs `gh pr create --title "..." --body "..."`
7. Saves PR URL to STATE.md under `## PRs`
8. Updates STATE.md: `step=ship`

---

### `coder progress` / `coder next` — Phase 15

See where you are and what to do next.

```bash
coder progress              # full project status (phases, blockers, PRs, backlog)
coder progress --short      # one-line: phase=1 step=plan last=...
coder next                  # prints the next recommended command
```

**`coder next` output examples:**

| STATE.md step | Output |
|---------------|--------|
| (empty) | `coder map-codebase` |
| `discuss` | `coder plan-phase 1` |
| `plan` | `coder execute-phase 1` |
| `execute` | `coder execute-phase 1 --gaps-only` |
| `qa` | `coder qa --phase 1` |
| `ship` | `coder milestone complete 1` |
| `done` | `coder discuss-phase 2` |

---

### `coder milestone` — Phase 16

Manage phase lifecycle: audit, complete, archive, next.

```bash
coder milestone audit 3       # show completion status for phase 3
coder milestone complete 3    # mark phase 3 done, record in STATE.md
coder milestone archive 2     # move phase 2 files to .coder/archive/02/
coder milestone next          # advance STATE.md to next phase
```

**`coder milestone audit` output:**
```
── Milestone Audit: Phase 3 — Auth Module ──

  ✓ CONTEXT.md
  ✓ VERIFICATION.md
  ✓ Plans   : 2 found
  ✓ Summaries: 2 / 2
  ✓ PR       : https://github.com/org/repo/pull/42
```

---

## Project Utilities

### `coder todo` — Manage backlog

```bash
coder todo                              # list backlog items
coder todo add "investigate rate limiting"
coder todo done "rate limiting"         # removes by substring match
coder todo clear                        # clear all items
```

### `coder stats` — Project statistics

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

### `coder health` — Project health check

```bash
coder health
# Output:
#   ✓ Project initialized
#   ✓ Roadmap: 5 phases
#   ✓ State: phase=3 step=execute
#   ⚠ 1 blocker recorded: waiting for API keys from client
```

### `coder note` — Record decisions and blockers

```bash
coder note "decided to use JWT with refresh tokens"
coder note --blocker "waiting for API credentials from client"
coder note --backlog "add rate limiting to backlog"
```

All notes are persisted to `.coder/STATE.md`.

### `coder do` — One-off AI task with project context

```bash
coder do "write unit tests for the auth service"
coder do "refactor the payment module to use dependency injection"
```

Injects `PROJECT.md` + `REQUIREMENTS.md` + current phase/step before the task.

---

## Directory Structure Reference

```
.coder/
├── PROJECT.md            # project name, description, tech stack
├── REQUIREMENTS.md       # full requirements list (from new-project)
├── ROADMAP.md            # phases with goals and status
├── STATE.md              # current phase, step, decisions, blockers, PRs
│
├── codebase/             # output from map-codebase
│   ├── STACK.md
│   ├── INTEGRATIONS.md
│   ├── ARCHITECTURE.md
│   ├── STRUCTURE.md
│   ├── CONVENTIONS.md
│   ├── TESTING.md
│   └── CONCERNS.md
│
├── phases/               # per-phase artifacts
│   ├── 01-CONTEXT.md         ← discuss-phase 1 output
│   ├── 01-RESEARCH.md        ← plan-phase 1 output (step A)
│   ├── 01-01-PLAN.md         ← plan-phase 1 output (step B)
│   ├── 01-02-PLAN.md
│   ├── 01-VERIFICATION.md    ← plan-phase verification + execute-phase verification
│   ├── 01-01-SUMMARY.md      ← execute-phase 1 output
│   ├── 01-02-SUMMARY.md
│   ├── 02-CONTEXT.md
│   └── ...
│
├── archive/              # milestone archive output
│   └── 01/               # archived phase 1 files
│
├── plans/                # coder plan output (Mode A)
│   └── PLAN-<slug>-<date>.md
│
├── qa/                   # coder qa output (Mode A)
├── sessions/             # coder session output (Mode A)
└── workflows/            # coder workflow output (Mode A)
    └── WF-<slug>-<date>.json
```

---

## STATE.md Format

STATE.md is the project's heartbeat — all lifecycle commands read and update it.

```markdown
project: My Task Manager CLI
current_phase: 2
step: execute
last_action: execute-phase 1 completed
updated: 2026-03-23T14:30:00+07:00

## Decisions
- [2026-03-20] decided to use JWT with refresh tokens
- [2026-03-21] Phase 1 (Auth) completed on 2026-03-21

## Blockers
- [2026-03-22] waiting for API credentials from client

## Backlog
- investigate rate limiting middleware
- add GraphQL layer in phase 5

## PRs
- phase 1: https://github.com/org/repo/pull/42
```

**Step values (lifecycle):**

| Step | Meaning | Next command |
|------|---------|-------------|
| (empty) | Just initialized | `coder map-codebase` |
| `discuss` | Ready for Q&A | `coder discuss-phase N` |
| `plan` | Context captured | `coder plan-phase N` |
| `execute` | Plans ready | `coder execute-phase N` |
| `qa` | Code done | `coder qa --phase N` |
| `ship` | PR opened | `coder milestone complete N` |
| `done` | Phase closed | `coder milestone next` |

---

## XML Plan Format

`plan-phase` generates plans in this XML format. `execute-phase` parses them:

```xml
<plan id="1-01" phase="1" name="JWT Middleware">
  <objective>Implement JWT authentication middleware with refresh token support</objective>
  <files>
    internal/middleware/auth.go
    internal/middleware/auth_test.go
    internal/domain/auth/token.go
  </files>
  <dependencies>none</dependencies>
  <estimated_time>1h</estimated_time>
  <tasks>
    <task type="create" name="JWT validator middleware">
      <action>
        Implement JWT validation middleware in internal/middleware/auth.go.
        Use github.com/golang-jwt/jwt/v5. Extract claims into request context.
        Return 401 with RFC 7807 error body on invalid/expired tokens.
      </action>
      <verify>go test ./internal/middleware/... passes</verify>
      <done>All protected routes return 401 without valid JWT</done>
    </task>
    <task type="create" name="Token refresh endpoint">
      <action>
        Add POST /auth/refresh in internal/handler/auth.go.
        Validate refresh token from cookie. Issue new access token.
        Rotate refresh token (one-time use). Update DB record.
      </action>
      <verify>curl -X POST /auth/refresh returns new access_token</verify>
      <done>Refresh endpoint issues new tokens and invalidates old ones</done>
    </task>
  </tasks>
</plan>
```

**Task types:** `create` | `modify` | `delete` | `test`
**Commit prefix mapping:** `create/modify` → `feat`, `test` → `test`, `delete` → `chore`, `fix` → `fix`

---

## Flags Quick Reference

| Command | Key flags |
|---------|-----------|
| `new-project` | `--auto`, `--prd <file>`, `--resume` |
| `map-codebase` | `[area]`, `--refresh` |
| `discuss-phase N` | `--auto`, `--batch` |
| `plan-phase N` | `--skip-research`, `--skip-verify`, `--gaps`, `--prd <file>` |
| `execute-phase N` | `--interactive`, `--gaps-only`, `--plan <id>` |
| `ship [N]` | `--draft`, `--base <branch>`, `--title <str>`, `--skip-push` |
| `progress` | `--short` |
| `milestone` | `audit [N]`, `complete [N]`, `archive [N]`, `next` |
| `todo` | `list`, `add <text>`, `done <text>`, `clear` |
| `note` | `--blocker`, `--backlog` |
| `chat` | `--file`, `--session`, `--resume`, `--no-memory` |
| `review` | `--file`, `--pr <N>`, `--focus`, `--format json` |
| `debug` | `--file`, `--context <file>`, `--diff`, `--interactive` |
| `plan` | `--auto`, `--prd`, `--list` |
| `qa` | `--plan <file>`, `--resume`, `--list`, `--report` |
| `session` | `save`, `resume`, `list`, `export` |
| `workflow` | `--steps`, `--dry-run`, `--resume`, `--prd`, `--list` |

---

## Tips & Troubleshooting

### "coder-node not running" / connection refused

```bash
# Check if coder-node is running
curl http://localhost:8080/health

# Re-configure connection
coder login
```

### Resume interrupted commands

Most lifecycle commands can be re-run safely:
- `new-project --resume` — continues Q&A from where it stopped
- `plan-phase N --skip-research` — skips if RESEARCH.md exists
- `execute-phase N --gaps-only` — skips plans that already have SUMMARY.md
- `workflow --resume` — continues the chained workflow

### Re-run from a specific step

```bash
# Context needs updating → re-discuss
coder discuss-phase 2

# Plans need more work → re-plan without research
coder plan-phase 2 --skip-research

# Only failed tasks → re-execute missing summaries
coder execute-phase 2 --gaps-only

# Re-plan only items flagged by verifier
coder plan-phase 2 --gaps
```

### Check project health before starting

```bash
coder health       # check all required artifacts
coder progress     # see full state
coder next         # get the next command
```

### Useful combo

```bash
# Morning standup — see where you left off
coder progress
coder next

# End of day — save context
coder session save
coder note "tomorrow: continue with execute-phase 3 --plan 03-02"
```

### Intel commands work without a project

```bash
# No .coder/ needed — works everywhere
coder chat "how does Go's context cancellation work?"
coder review
coder debug "panic: concurrent map writes"
coder skill search "nestjs error handling"
coder memory search "JWT refresh pattern"
```
