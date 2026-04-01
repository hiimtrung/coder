---
description: Review feature design documentation for completeness, accuracy, and architectural soundness.
---

# Workflow: Review Design

Review a feature's design document before implementation begins. Catches gaps, inconsistencies, and architectural violations early — when they are cheapest to fix.

## When to Use

- A design document has been written and needs approval before implementation
- Implementation is underway but design feels incomplete
- Reviewing an existing design before a major refactor

## Step 1 — Context Load (MANDATORY)

```bash
coder skill resolve "architecture <design context>" --trigger review --budget 3
coder memory search "<feature or component name>"
```

Use `coder memory recall "<feature or component name>"` when the design history is broad and you need a focused active working set.
Use `coder memory active` or `.coder/context-state.json` to inspect the current local context before reviewing the design.

## Step 2 — Read the Design Document

Read `docs/design/<feature>.md` in full. Also read:

- The linked requirements doc to understand what the design must achieve
- Any referenced existing modules to verify integration points

## Step 3 — Structured Review

Evaluate the design against each criterion below:

### Architecture

- [ ] Mermaid component diagram is present and accurately represents the structure
- [ ] Sequence diagram covers the primary data flow
- [ ] Dependencies point inward only (no domain → infrastructure in diagrams)
- [ ] Cross-module communication is event-driven, not direct repository access
- [ ] New modules are placed in the correct layer

### API Contract

- [ ] All endpoints from the requirements are present
- [ ] Request and response DTOs are fully specified
- [ ] All error responses are listed with correct error codes (`VAL_*`, `BIZ_*`, etc.)
- [ ] Authentication requirements are stated
- [ ] Pagination specified for list endpoints

### Data Model

- [ ] All required fields are present in the schema
- [ ] Foreign keys and indexes are defined
- [ ] Migration approach is described
- [ ] Multi-tenant isolation (`company_id`) is included on all relevant tables
- [ ] Data types are appropriate (UUIDs for IDs, TIMESTAMPTZ for timestamps)

### Error Handling

- [ ] Error scenarios from the requirements edge cases section are addressed
- [ ] Correct error code categories used for each scenario
- [ ] Error responses do not expose internal implementation details

### Security

- [ ] Auth requirements stated explicitly
- [ ] PII fields identified and handling described
- [ ] Rate limiting mentioned for public endpoints
- [ ] `company_id` sourced from JWT, not request body

### Non-Functional Requirements

- [ ] Performance requirements from requirements doc are addressed
- [ ] Scalability considerations noted
- [ ] Observability (logging, metrics, tracing) mentioned

### ADR

- [ ] At least one ADR present for each non-obvious decision
- [ ] Alternatives considered section is populated
- [ ] Consequences section is honest about trade-offs

## Step 4 — Summarize Findings

Produce a review summary:

```markdown
## Design Review: <Feature Name>

**Reviewer**: <name>
**Date**: YYYY-MM-DD
**Verdict**: APPROVED | APPROVED WITH COMMENTS | REQUEST REVISION

### Findings

#### Blocking (must fix before implementation)

1. <finding>: <specific section and what needs to change>

#### Recommended

1. <finding>: <suggestion>

#### Minor

1. <observation>

### Summary

<2-3 sentences on overall design quality and most important finding>
```

## Step 5 — Gate Out (MANDATORY)

```bash
coder memory store "Design Review: <Feature Name>" "Verdict: <verdict>. Key decisions confirmed: <list>. Issues found: <count and summary>. Architecture compliance: <notes>." --tags "design-review,<feature>,architecture"
```

---

## Checklist

- [ ] `coder skill resolve` run
- [ ] `coder memory search` run
- [ ] Design doc read in full
- [ ] Requirements doc cross-referenced
- [ ] All review categories evaluated
- [ ] Findings classified as blocking / recommended / minor
- [ ] Verdict stated clearly
- [ ] `coder memory store` run
