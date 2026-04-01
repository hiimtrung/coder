---
description: Knowledge management — decide what to store, write high-quality memory entries, and keep the knowledge base clean and actionable.
---

# Workflow: Knowledge Capture

This workflow governs when and how to store knowledge in the semantic memory system. Good memory hygiene ensures agents consistently benefit from past work. Poor memory hygiene means repeating the same mistakes.

## When to Use

- After completing any non-trivial implementation
- After resolving a non-obvious bug
- After making a significant architectural decision
- After integrating with an external system or API
- After discovering a pattern worth reusing
- Periodically to review and clean stale memory

## Decision Tree: Should I Store This?

```
Did this involve non-obvious reasoning or discovery?
├─ YES → Store it
└─ NO  → Was it a single-line fix or obvious typo?
          ├─ YES → Skip
          └─ NO  → Does a future agent benefit from knowing this?
                    ├─ YES → Store it
                    └─ NO  → Skip
```

### Store (Yes)

| Situation                            | Memory Type  | Tags                       |
| ------------------------------------ | ------------ | -------------------------- |
| New feature implemented              | Pattern      | feature, module, language  |
| Architectural decision made          | Decision     | architecture, decision     |
| Non-obvious bug fixed                | Bug fix      | bug, component, error-type |
| External API integration figured out | Integration  | integration, api, vendor   |
| Refactor pattern discovered          | Pattern      | refactor, module           |
| New testing strategy established     | Testing      | testing, coverage          |
| Performance optimization found       | Optimization | performance, module        |

### Skip (No)

- Single-line typo fix
- Variable rename
- Comment update
- Obvious syntax correction
- Change already documented in a PR description

## Memory Entry Templates

### Pattern Entry

```bash
coder memory store "Pattern: <Descriptive Title>" \
  "Context: <when this pattern applies>. Implementation: <how it works>. Example: <code or pseudocode snippet>. Pitfalls: <what to avoid>. Files: <relevant file paths>." \
  --tags "pattern,<module>,<language>"
```

Example:

```bash
coder memory store "Pattern: NestJS multi-tenant repository base class" \
  "Context: All repositories in omi-channel-be must filter by company_id. Implementation: Extend BaseRepository which injects company_id from request context automatically. Pitfall: Never call base.findAll() without scope — always use findByCompany(). Files: src/shared/repositories/base.repository.ts." \
  --tags "pattern,nestjs,multi-tenancy,repository"
```

### Architectural Decision Entry

```bash
coder memory store "Decision: <What was decided>" \
  "Context: <what problem prompted the decision>. Decision: <what was chosen>. Rationale: <why>. Alternatives rejected: <what else was considered and why rejected>. Consequences: <trade-offs accepted>. Date: YYYY-MM-DD." \
  --tags "decision,architecture,<module>"
```

Example:

```bash
coder memory store "Decision: Use outbox pattern for domain event publishing" \
  "Context: We needed guaranteed event delivery without dual-write failures. Decision: All domain events written to outbox table in same transaction as aggregate save, then published async by outbox worker. Rationale: Eliminates possibility of event loss on app crash. Alternatives: Direct Kafka publish (rejected: possible loss on crash), Saga (rejected: overkill for current scale). Date: 2026-01-10." \
  --tags "decision,architecture,events,outbox"
```

### Bug Fix Entry

```bash
coder memory store "Bug: <Issue Description>" \
  "Symptoms: <observable behavior>. Root cause: <precise technical cause>. Fix: <what was changed>. Location: <file:line>. Regression test: <test name>. Prevention: <how to avoid in future>." \
  --tags "bug,<component>,<error-type>"
```

Example:

```bash
coder memory store "Bug: JWT company_id missing after token refresh" \
  "Symptoms: 404 on all requests after token refresh despite valid token. Root cause: Refresh endpoint issued new token without copying company_id claim from old token. Fix: Added company_id to refresh token claims in auth.service.ts:refreshToken(). Location: src/auth/auth.service.ts line 87. Regression test: 'should include company_id in refreshed token'. Prevention: Always include company_id in token claims checklist." \
  --tags "bug,auth,jwt,company_id"
```

### Integration Entry

```bash
coder memory store "Integration: <System Name>" \
  "Purpose: <what this integration does>. Auth: <how authentication works>. Endpoints used: <list>. Rate limits: <if any>. Gotchas: <non-obvious behaviors>. Error handling: <how errors are handled>. Files: <relevant files>." \
  --tags "integration,<vendor>,<module>"
```

## Memory Quality Standards

A high-quality memory entry:

- Has a clear, searchable title that describes the topic (not "fix for issue #123")
- Is written as if explaining to a new team member
- Includes enough context to understand when it applies
- Contains actionable information (code patterns, commands, file paths)
- Identifies pitfalls or anti-patterns to avoid
- Is concise — if it needs more than 3 sentences per section, consider splitting

A low-quality memory entry (do not store these):

- Generic: "Use clean architecture" (too obvious)
- Incomplete: "Fixed the auth bug" (no root cause, no location)
- Stale: describes a pattern that was subsequently changed
- Duplicate: exact same content already in memory

## Periodic Memory Review

Run this quarterly or when the memory feels noisy:

```bash
# List all stored memories
coder memory list

# Remove outdated entries
coder memory compact --revector
```

During review, evaluate each entry:

- Is it still accurate given recent code changes?
- Is it specific enough to be actionable?
- Has it been superseded by a better pattern?

## Step-by-Step Procedure

1. `coder skill resolve "knowledge management" --trigger review --budget 3` — retrieve any memory standards
2. `coder memory search "<topic>"` — check what already exists before storing
3. `coder memory recall "<topic>"` or `coder memory active` — inspect and narrow the active working set before deciding what to pin next
4. Choose the correct template (Pattern / Decision / Bug / Integration)
5. Write the entry following quality standards above
6. Run `coder memory store` with appropriate tags
7. Verify the entry is findable: `coder memory search "<key term from entry>"`

---

## Checklist

- [ ] Decision tree applied — confirmed this is worth storing
- [ ] Existing memories checked to avoid duplicates
- [ ] Correct template used for the entry type
- [ ] Entry title is specific and searchable
- [ ] Content includes context, implementation, and pitfalls
- [ ] Tags are accurate and useful for future search
- [ ] Entry verified as findable after storage
