---
description: Refactor complex code to reduce complexity while preserving behavior — with structured analysis, before/after comparison, and prioritized action plan.
---

# Workflow: Simplify Implementation

Systematically reduce complexity in existing code. The goal is to make code easier to understand, maintain, and extend — without changing observable behavior.

## When to Use

- A module is difficult to understand or modify safely
- Code review flagged excessive complexity
- Onboarding is slow because the codebase is hard to follow
- A piece of code has accumulated too many responsibilities
- Preparing to add a significant new feature to an existing module

## Step 1 — Context Load (MANDATORY)

```bash
coder skill search "refactoring <language or module>"
coder memory search "<module or feature to simplify>"
```

## Step 2 — Gather Context

If the target is not specified, ask for:
- Target file(s) or component(s)
- Current pain points (hard to understand, extend, or test?)
- Performance or scalability concerns
- Constraints: backward compatibility, API stability, deadlines

## Step 3 — Complexity Analysis

For each target file or component, identify complexity sources:

### Structural Complexity

| Issue | Indicator | Recommended Fix |
|-------|-----------|-----------------|
| Deep nesting | More than 3 levels of if/for/try | Extract guard clauses, extract method |
| Long method | More than 30 lines | Extract sub-methods with descriptive names |
| God class | More than 300 lines | Extract by responsibility |
| Long parameter list | More than 4 parameters | Introduce parameter object / command |
| Duplicate code | Same logic in 2+ places | Extract shared utility or base class |

### Cognitive Complexity

Apply the **30-second test**: can a new team member understand what a function does in 30 seconds?

- Too many responsibilities in one function → extract
- Magic numbers or strings → extract as named constants
- Negative boolean logic (`!isNotActive`) → rename variable

### Architecture Violations

- Business logic in controllers → move to use case
- DB queries in use cases → move to repository
- Framework imports in domain entities → remove
- Direct cross-module repository calls → replace with events

### Tight Coupling

- Concrete class injected where interface should be used
- Infrastructure detail (DB column name) leaking into domain
- Hard-coded external URLs or configs → move to environment config

## Step 4 — Propose Simplifications

For each identified issue, provide a before/after example:

```typescript
// BEFORE: deep nesting, multiple responsibilities
async function processOrder(order: Order) {
  if (order.status === 'pending') {
    if (order.items.length > 0) {
      for (const item of order.items) {
        if (item.stock > 0) {
          await this.db.query(`UPDATE stock SET quantity = quantity - 1 WHERE id = ${item.id}`);
        }
      }
      order.status = 'processed';
      await this.db.query(`UPDATE orders SET status = 'processed' WHERE id = ${order.id}`);
    }
  }
}

// AFTER: guard clauses, extracted methods, parameterized queries, correct layer
async processOrder(command: ProcessOrderCommand): Promise<void> {
  const order = await this.orderRepo.findByIdOrThrow(command.orderId, command.companyId);
  order.process(); // domain method handles status logic
  await this.stockRepo.decrementForOrder(order.items);
  await this.orderRepo.save(order);
  await this.events.publish(new OrderProcessedEvent(order.id));
}
```

## Step 5 — Prioritize Changes

Rank proposed changes by impact and risk:

| Priority | Impact | Risk | Action |
|----------|--------|------|--------|
| 1 — Do first | High | Low | Extract guard clauses, rename variables |
| 2 — Plan carefully | High | Medium | Extract sub-modules, introduce interfaces |
| 3 — Quick wins | Low | Low | Remove dead code, add constants |
| 4 — Defer | Low | High | Major architecture changes |

For each high-risk change, specify:
- What tests must exist before refactoring (to catch regressions)
- How to validate the change preserves behavior

## Step 6 — Execute Refactoring

For each change:

1. Ensure tests exist that cover the current behavior
2. Make the change
3. Run the full test suite — behavior must not change
4. Commit as a separate commit from feature work

```bash
git add <specific files>
git commit -m "refactor(<scope>): <what was simplified>

- <specific change made>
- Tests: all N passing, behavior unchanged"
```

Keep refactor commits separate from feature commits. This makes git bisect and rollback safer.

## Step 7 — Gate Out (MANDATORY)

```bash
coder memory store "Simplification: <Module or Feature Name>" "Complexity sources found: <list>. Changes made: <list of improvements>. Patterns extracted: <reusable patterns>. Test count before/after: <N/N>. Files changed: <list>." --tags "refactor,simplification,<module>,<language>"
```

---

## Checklist

- [ ] `coder skill search` run
- [ ] `coder memory search` run
- [ ] Target files identified
- [ ] Complexity sources categorized (structural / cognitive / architecture / coupling)
- [ ] Before/after examples written for each significant change
- [ ] Changes prioritized by impact and risk
- [ ] Tests exist before refactoring begins
- [ ] Each change committed separately
- [ ] Full test suite passes after refactoring
- [ ] `coder memory store` run with patterns discovered
