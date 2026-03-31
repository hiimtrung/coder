---
description: Review feature requirements for completeness, clarity, and testability before design or implementation begins.
---

# Workflow: Review Requirements

A structured review of a requirements document before it is used to drive design or implementation. Catching gaps at this stage is far cheaper than catching them during code review.

## When to Use

- A requirements document has been written and needs approval
- Requirements feel incomplete or ambiguous before design begins
- Reviewing requirements as part of a backlog refinement session

## Step 1 — Context Load (MANDATORY)

```bash
coder skill resolve "requirements analysis <domain>" --trigger review --budget 3
coder memory search "<feature or product area>"
```

## Step 2 — Read the Document

Read `docs/requirements/<feature>.md` in full. Also review:
- The project `ROADMAP.md` or product direction for context
- Related existing requirement docs for consistency

## Step 3 — Structured Review

Evaluate each section against the criteria below:

### Goal and Problem Statement

- [ ] Goal is stated in one clear sentence
- [ ] The problem being solved is named explicitly
- [ ] The target user or role is identified
- [ ] Success is measurable — not "improve the experience" but "reduce checkout time by 30%"

### User Stories

- [ ] Each story follows: "As a [role], I want to [action], so that [outcome]"
- [ ] Each story is small enough to implement independently
- [ ] Stories are not implementation steps (no "System calls the DB" stories)
- [ ] All personas from the Users section have at least one story
- [ ] Stories do not contradict each other

### Acceptance Criteria

- [ ] Every story has at least one acceptance criterion
- [ ] Criteria are written in BDD format: Given / When / Then
- [ ] Criteria are testable — a QA engineer can verify each one
- [ ] Both success and failure cases are covered
- [ ] Numeric thresholds specified where applicable (not "fast" but "< 200ms")

### Scope

- [ ] In-scope items are specific, not vague
- [ ] Out-of-scope section exists and is populated
- [ ] v1 / v2 split is clear — what is deferred is stated explicitly

### Edge Cases and Error Handling

- [ ] Invalid input scenarios are addressed
- [ ] Missing or null data scenarios are addressed
- [ ] Downstream service failure scenarios are addressed
- [ ] Concurrent access scenarios are addressed (if relevant)

### Integrations

- [ ] All systems this feature interacts with are listed
- [ ] Dependency direction is clear (upstream vs downstream)
- [ ] Breaking changes to existing integrations are noted

### Non-Functional Requirements

- [ ] Performance expectations are stated (or explicitly deferred)
- [ ] Security requirements are stated (auth, data sensitivity)
- [ ] Data retention or compliance requirements noted (if relevant)

### Open Questions

- [ ] All unresolved items are listed
- [ ] Each open question has a clear owner and due date (or is marked "accepted risk")

## Step 4 — Summarize Findings

```markdown
## Requirements Review: <Feature Name>

**Reviewer**: <name>
**Date**: YYYY-MM-DD
**Verdict**: APPROVED | APPROVED WITH COMMENTS | REQUEST REVISION

### Core Problem and Users
<summary of what the requirements say the feature does and for whom>

### User Stories Coverage
<N stories covering: X, Y, Z workflows>

### Gaps Found

#### Blocking (must address before design)
1. <gap>: <specific section and what is missing>

#### Recommended
1. <gap>: <suggestion>

### Inconsistencies or Contradictions
1. <description>: <which sections conflict>

### Summary
<2-3 sentences on overall requirement quality and most important finding>
```

## Step 5 — Gate Out (MANDATORY)

```bash
coder memory store "Requirements Review: <Feature Name>" "Verdict: <verdict>. User stories: <N>. Gaps found: <count and summary>. Key business rules confirmed: <rules>." --tags "requirements-review,<feature>,<domain>"
```

---

## Checklist

- [ ] `coder skill resolve` run
- [ ] `coder memory search` run
- [ ] Requirements doc read in full
- [ ] All review categories evaluated
- [ ] Gaps classified as blocking / recommended
- [ ] Inconsistencies identified
- [ ] Verdict stated clearly
- [ ] `coder memory store` run
