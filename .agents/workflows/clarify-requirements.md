---
description: Business Analyst process — elicit, clarify, and document feature requirements before any design or implementation begins.
---

# Workflow: Clarify Requirements

This workflow is the first step in the delivery pipeline. No design or code begins until requirements are documented and confirmed.

## When to Use

- Starting any new feature, module, or significant change
- Requirements are ambiguous, incomplete, or undocumented
- Stakeholder intent is unclear
- Existing requirements need revision before the next delivery cycle

## Step 1 — Context Load (MANDATORY)

Before engaging the stakeholder, silently load existing knowledge:

```bash
coder skill search "requirements analysis"
coder memory search "<feature name or domain>"
```

Also read:
- `docs/requirements/` — existing requirement docs
- `docs/design/` — any prior design context
- `ROADMAP.md` or `README.md` — product direction

## Step 2 — Structured Elicitation

Ask the following 7 questions. Present as a numbered list. Do NOT start writing docs or code until you have answers.

```
1. Goal — What problem does this solve, and for whom?
          What does success look like in one sentence?

2. Users — Who are the primary users of this feature?
           Are there different user roles with different needs?

3. Scope (v1) — What is the minimum set of functionality for v1?
                What is explicitly OUT of scope for now?

4. User Stories — Walk me through the primary workflow step by step.
                  What does the user do, see, and get back?

5. Edge Cases — What happens when input is invalid, missing, or malformed?
                What happens if a downstream service is unavailable?
                What are the boundary conditions?

6. Acceptance Criteria — How will we know the feature is done?
                         What tests or checks must pass?

7. Integrations — What existing systems, APIs, or modules does this touch?
                  Are there upstream dependencies or downstream consumers?
```

## Step 3 — Write Requirements Document

After receiving answers, immediately write the document:

**Output path**: `docs/requirements/<feature-name>.md`

```markdown
# Requirements: <Feature Name>

**Date**: YYYY-MM-DD
**Status**: Draft | Confirmed
**Author**: <name or team>

---

## Goal

<One sentence: what this feature does and why it exists>

## Users

| Role | Need |
|------|------|
| <role> | <what they need from this feature> |

---

## User Stories

### Story 1: <Title>

**As a** <role>
**I want to** <action>
**So that** <outcome>

**Acceptance Criteria** (BDD format):

```gherkin
Given <initial context>
When <user action>
Then <expected outcome>
And <additional outcome>
```

### Story 2: <Title>
... (repeat for each story)

---

## Scope

### In Scope (v1)
- ...

### Out of Scope
- ...
- (deferred to v2 or future iterations)

---

## Edge Cases and Error Handling

| Scenario | Expected Behavior |
|----------|-------------------|
| <edge case> | <what the system must do> |

---

## Integration Points

| System / Module | Dependency Type | Notes |
|-----------------|-----------------|-------|
| <system> | upstream / downstream | <detail> |

---

## Non-Functional Requirements

- **Performance**: <response time, throughput, concurrency expectations>
- **Security**: <auth requirements, data sensitivity, PII handling>
- **Reliability**: <uptime, retry behavior, idempotency>
- **Scalability**: <data volume, growth projections>

---

## Open Questions

- [ ] <question that needs stakeholder answer>
- [ ] <assumption that needs validation>

---

## Decisions Log

| Date | Decision | Rationale |
|------|----------|-----------|
| YYYY-MM-DD | <decision> | <why> |
```

## Step 4 — Confirm

Present the document summary to the stakeholder:

> "I've documented the requirements for [Feature]. Here is the summary:
> - Goal: [one sentence]
> - Stories: [N stories covering X, Y, Z]
> - Out of scope: [list]
> - Open questions: [list]
>
> Does this capture everything correctly? Any corrections before design begins?"

Only proceed to `architecture-design.md` after explicit confirmation.

## Step 5 — Gate Out (MANDATORY)

```bash
coder memory store "Requirements: <Feature Name>" "Goal: <goal>. Key constraints: <constraints>. Out of scope: <out-of-scope items>. Open questions: <questions>." --tags "requirements,<feature>,<domain>"
```

---

## Checklist

- [ ] `coder skill search` run before starting
- [ ] `coder memory search` run before starting
- [ ] All 7 elicitation questions asked and answered
- [ ] `docs/requirements/<feature>.md` written with all sections
- [ ] Document confirmed by stakeholder
- [ ] `coder memory store` run with key decisions
