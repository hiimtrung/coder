# Rule: Using coder CLI Commands

## Decision Guide — Which Command to Use?

### Quick Command Choices

```
Need general implementation guidance?
  → coder skill search "<topic>"
  → coder skill resolve "<task>" --trigger initial --budget 3

Need project-specific truth or historical context?
  → coder memory search "<topic>"
  → coder memory recall "<task>" --trigger execution --budget 5
  → coder memory active --format json

Need to save or resume local context?
  → coder session save
  → coder session resume

Need current project state?
  → coder progress
  → coder next
  → coder milestone audit N
```

### Current Command Surface

```
Memory and skills?
  → coder skill search "topic"
  → coder skill resolve "task" --trigger execution --budget 3
  → coder skill active --format json
  → coder memory search "topic" --format json
  → coder memory recall "task" --trigger execution --budget 5
  → coder memory active --format json

Project state?
  → coder progress                    full status (phases, step, blockers, PRs)
  → coder progress --short            one-line summary
  → coder next                        print the next implemented state command
  → coder milestone audit N           show completion checklist
  → coder milestone complete N        mark done
  → coder milestone archive N         move files to .coder/archive/NN/
  → coder milestone next              advance to next phase

Session and setup?
  → coder session save "task"
  → coder session resume
  → coder install <profile>
  → coder update [profile]
  → coder login
  → coder token show
```

Legacy commands such as `coder new-project`, `coder map-codebase`, `coder discuss-phase`, `coder plan-phase`, `coder execute-phase`, `coder ship`, `coder todo`, and `coder note` are roadmap or historical only. Do not describe them as live commands.

---

## Output Locations

```
.coder/
  STATE.md               ← coder progress / next / milestone read and update this
  ROADMAP.md             ← coder progress reads roadmap phases from here
  session.md             ← coder session resume default source
  sessions/*.md          ← coder session save history
  active-skills.json     ← coder skill resolve / active
  active-memory.json     ← coder memory search / recall / active
  phases/                ← milestone audit/archive examines files here
  archive/NN/            ← milestone archive output
```

---

## Local State Refresh

| Command               | Local state refreshed                       |
| --------------------- | ------------------------------------------- |
| `coder skill resolve` | `.coder/active-skills.json`                 |
| `coder memory search` | `.coder/active-memory.json`                 |
| `coder memory recall` | `.coder/active-memory.json`                 |
| `coder session save`  | `.coder/session.md`, `.coder/sessions/*.md` |

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
  "auth": { "access_token": "your-token-here" }
}
```

Run `coder login` to set this up interactively.

---

## Guidance For Agents

For long-running work, treat memory the same way skills are treated:

- use `coder memory search` for plain recall
- use `coder memory recall` when you need a keep/add/drop decision against current active memory
- use `coder memory active --format json` to inspect the last local recall state

---

## Skill Ingest After Changes

After updating `.agents/skills/` files, re-ingest so vector search picks them up:

```bash
coder skill ingest --source local
```
