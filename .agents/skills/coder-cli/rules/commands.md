# Rule: Using coder CLI Commands

## Decision Guide — Which Command to Use?

### Quick AI Workflows (no project setup needed)

```
Starting a new feature?
  → coder plan "<feature>"            generate PLAN.md with tasks + estimates
  → coder plan --auto "<feature>"     skip Q&A, generate directly

Quick question about the codebase?
  → coder chat "<question>"           context-enriched Q&A (memory + skills injected)

Just wrote code, want review?
  → coder review                      reviews current git diff
  → coder review --file path.go       reviews a specific file
  → coder review --pr 42              reviews a GitHub PR

Hit an error / bug?
  → coder debug "<error message>"     structured root cause analysis
  → coder debug --file error.log      from a log file
  → coder debug --interactive         REPL for multi-turn debugging

Done implementing, need UAT?
  → coder qa --plan PLAN-xxx.md       walks through acceptance criteria

Ending the day / switching context?
  → coder session save

Running a complete feature end-to-end?
  → coder workflow "<feature>"        plan → review → implement → qa → fix
```

### Full Project Lifecycle (requires `coder new-project` first)

```
Starting a new project from scratch?
  → coder new-project "idea"          AI-guided Q&A → requirements + roadmap + STATE.md
  → coder new-project --auto "idea"   skip Q&A
  → coder new-project --prd file.md   load from PRD file

Analyzing existing codebase?
  → coder map-codebase                4 AI passes → STACK/ARCH/CONVENTIONS/CONCERNS
  → coder map-codebase --refresh      force re-analysis

Clarifying a phase before planning?
  → coder discuss-phase N             interactive Q&A → CONTEXT.md
  → coder discuss-phase N --auto      AI picks defaults, no Q&A

Generating implementation plans?
  → coder plan-phase N                research → XML plans → verification loop
  → coder plan-phase N --skip-research  if RESEARCH.md exists
  → coder plan-phase N --gaps         re-plan only items flagged in VERIFICATION.md

Executing plans?
  → coder execute-phase N             all plans, atomic git commits per task
  → coder execute-phase N --gaps-only skip plans with existing SUMMARY.md
  → coder execute-phase N --interactive  pause between each plan for review

Shipping a phase?
  → coder ship N                      gh pr create with AI-generated body
  → coder ship N --draft              open as draft PR

Checking progress?
  → coder progress                    full status (phases, step, blockers, PRs)
  → coder progress --short            one-line summary
  → coder next                        print next recommended command

Closing out a phase?
  → coder milestone audit N           show completion checklist
  → coder milestone complete N        mark done
  → coder milestone archive N         move files to .coder/archive/NN/
  → coder milestone next              advance to next phase

Daily utilities?
  → coder todo add "..."              add backlog item
  → coder note "decision"             record decision to STATE.md
  → coder note --blocker "..."        record blocker
  → coder health                      check artifacts + blockers
  → coder stats                       project statistics
  → coder do "task description"       one-off AI task with project context
```

---

## Output Locations

```
.coder/
  PROJECT.md          ← coder new-project
  REQUIREMENTS.md     ← coder new-project
  ROADMAP.md          ← coder new-project
  STATE.md            ← all lifecycle commands update this

  codebase/           ← coder map-codebase
    STACK.md
    ARCHITECTURE.md
    CONVENTIONS.md
    CONCERNS.md
    (+ INTEGRATIONS.md, STRUCTURE.md, TESTING.md)

  phases/             ← discuss/plan/execute/ship outputs
    NN-CONTEXT.md
    NN-RESEARCH.md
    NN-01-PLAN.md
    NN-01-SUMMARY.md
    NN-VERIFICATION.md

  archive/NN/         ← milestone archive output

  plans/              ← coder plan (Mode A)
  qa/                 ← coder qa (Mode A)
  sessions/           ← coder session (Mode A)
  workflows/          ← coder workflow (Mode A)
```

---

## Resuming Interrupted Commands

| Command | Resume approach |
|---------|----------------|
| `coder new-project` | `--resume` |
| `coder plan-phase N` | `--skip-research` (RESEARCH.md exists) |
| `coder plan-phase N` | `--gaps` (re-plan only flagged items) |
| `coder execute-phase N` | `--gaps-only` (skips plans with SUMMARY.md) |
| `coder workflow` | `--resume` |
| `coder qa` | `--resume` |

---

## Chat Stream Pattern

All streaming commands use `ChatStream` under the hood:

```go
chatClient.ChatStream(ctx, prompt, sessionID, injectMemory, injectSkills, func(delta string) {
    fmt.Print(delta)   // print token as it arrives
    buffer.WriteString(delta)
})
```

---

## Config Setup

The CLI reads `~/.coder/config.json`:
```json
{
  "memory": { "base_url": "http://localhost:8080" },
  "auth":   { "access_token": "your-token-here" }
}
```

Run `coder login` to set this up interactively.

---

## Session Context Auto-Injection

If `.coder/session.md` exists, `coder chat` / `coder debug` / `coder review` automatically
inject it as extra system context. The AI "knows" what you're working on without being told.

---

## coder plan → coder qa Chain (Mode A)

```bash
# 1. Generate plan with acceptance criteria
coder plan "implement JWT refresh tokens"
# → saves to .coder/plans/PLAN-implement-jwt-refresh-<date>.md

# 2. After implementing, run QA against that plan
coder qa --plan .coder/plans/PLAN-implement-jwt-refresh-<date>.md
# → walks through each criterion, auto-diagnoses failures via coder debug
```

## coder plan-phase → coder execute-phase Chain (Mode B)

```bash
# 1. Discuss → plan → execute chain
coder discuss-phase 1           # Q&A → .coder/phases/01-CONTEXT.md
coder plan-phase 1              # → .coder/phases/01-01-PLAN.md (XML)
coder execute-phase 1           # → executes tasks, git commit per task
coder ship 1                    # → gh pr create
coder milestone complete 1      # → STATE.md: step=done
coder milestone next            # → STATE.md: current_phase=2, step=discuss
```

---

## Skill Ingest After Changes

After updating `.agents/skills/` files, re-ingest so vector search picks them up:
```bash
coder skill ingest --source local
```
