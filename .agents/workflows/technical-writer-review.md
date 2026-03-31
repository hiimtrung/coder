---
description: Review and improve documentation from a professional technical writer's perspective — clarity, completeness, actionability, and structure.
---

# Workflow: Technical Writer Review

Review documentation files to ensure they are clear, complete, and useful for the intended audience — whether that is a developer onboarding to the project, an operator following a runbook, or an API consumer building an integration.

## When to Use

- Documentation was written by an engineer and needs a quality pass
- Docs are stale relative to recent code changes
- Preparing documentation for a public release
- Onboarding feedback indicates docs are hard to follow

## Step 1 — Context Load (MANDATORY)

```bash
coder skill resolve "technical writing documentation" --trigger review --budget 3
coder memory search "<document topic or component>"
```

## Step 2 — Identify Target Documents

Confirm which documents are being reviewed:
- API reference (`docs/api/`)
- Runbook (`docs/runbooks/`)
- Requirements or design doc (`docs/requirements/`, `docs/design/`)
- README, CHANGELOG, or other project-level docs

Read each document from start to finish as its target audience would — as a developer seeing this for the first time.

## Step 3 — Evaluate by Criteria

### 1. Clarity

- Does it explain concepts before using them (no undefined jargon)?
- Is each sentence unambiguous — could it be interpreted more than one way?
- Are examples concrete and realistic (not "foo" and "bar")?
- Is the vocabulary consistent (same term used for the same thing throughout)?

### 2. Completeness

- Does it explain *what* something is before explaining *how* to use it?
- Are prerequisites explicitly stated?
- Are there quick start examples for common tasks?
- Are edge cases and error scenarios addressed?
- Is the "why" (motivation) explained for non-obvious choices?

### 3. Actionability

- Are commands copy-paste ready (correct tool, flags, and quoting)?
- Is expected output shown after commands?
- Are "when to use this" hints provided?
- Are links to related docs present?
- Can a reader follow the document without leaving to search for missing context?

### 4. Structure

- Does the order make sense for a first-time reader?
- Is there a clear hierarchy (headings, subheadings)?
- Is there a logical flow from simple to complex?
- Are related items grouped together?
- Is the document an appropriate length (not padded, not truncated)?

## Step 4 — Produce Review Report

For each document:

```markdown
## [Document Name]

**Audience**: <developer | operator | API consumer>
**Current State**: Good | Needs Work | Major Revision Required

| Aspect | Rating (1-5) | Notes |
|--------|--------------|-------|
| Clarity | N/5 | <specific observations> |
| Completeness | N/5 | <what is missing> |
| Actionability | N/5 | <what blocks a reader from acting> |
| Structure | N/5 | <organizational issues> |

**Issues by Priority**:

**High** (blocks the reader from succeeding):
1. <issue> — Line <N>: <specific fix>

**Medium** (causes confusion but reader can work through it):
1. <issue> — Section <X>: <suggestion>

**Low** (polish):
1. <observation> — <suggestion>

**Suggested Fixes**:
```markdown
[Before]
<original text>

[After]
<improved text>
```
```

## Step 5 — Common Patterns to Fix

| Issue | Fix |
|-------|-----|
| No introduction | Add opening paragraph: what this is, what it does, who it's for |
| No prerequisites | Add prerequisites section before step 1 |
| Undefined jargon | Add inline explanation on first use |
| No quick start | Add a minimal working example before the full reference |
| Flat structure | Organize into logical sections with descriptive headings |
| No cross-references | Add "See also" or "Next steps" links |
| Missing expected output | Show what success looks like after each command |
| Vague language | Replace "configure appropriately" with the exact values or options |
| Wrong audience level | Adjust technicality for the intended reader |

## Step 6 — Gate Out (MANDATORY)

```bash
coder memory store "Doc Review: <Document Name>" "Audience: <audience>. Overall quality: <N/5>. Critical issues: <count and summary>. Key terminology decisions: <terms standardized>. Structural changes made: <if applied>." --tags "documentation,technical-writing,<component>"
```

---

## Checklist

- [ ] `coder skill resolve` run
- [ ] `coder memory search` run
- [ ] Each document read as target audience
- [ ] All four criteria evaluated (clarity, completeness, actionability, structure)
- [ ] Issues classified by priority (high / medium / low)
- [ ] Concrete before/after suggestions provided for high-priority issues
- [ ] `coder memory store` run
