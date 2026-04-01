---
description: Debug and root cause analysis process — 5-step structured investigation ending in a confirmed fix and post-mortem document.
---

# Workflow: Debug Issue

A structured approach to diagnosing and resolving bugs, errors, and unexpected behaviors. Produces a confirmed root cause, a minimal fix, and a post-mortem for institutional memory.

## When to Use

- An error or exception is occurring in production or staging
- A test is failing without a clear reason
- Behavior is incorrect but no error is thrown
- Performance degradation with unclear origin

## Step 1 — Context Load (MANDATORY)

```bash
coder skill resolve "<error type or component>" --trigger error-recovery --budget 3
coder memory search "<error message or symptom keywords>"
```

Use `coder memory recall "<error message or symptom keywords>"` when the debugging history is broad and you need the closest incidents pinned.
Use `coder memory active` or `.coder/context-state.json` to inspect the current local context before tracing the root cause.

Check memory first — this issue may have been seen before.

## Step 2 — Symptom Collection

If the issue description is incomplete, ask for:

1. What is happening (observed behavior)?
2. What should happen (expected behavior)?
3. When did it start? After which deployment or change?
4. Is it reproducible? Always, intermittent, or environment-specific?
5. Error message, stack trace, or log excerpt (exact text)
6. Scope of impact: one user, all users, specific tenant?

Document collected symptoms:

```markdown
## Symptoms

- Observed: <exact observed behavior>
- Expected: <what should have happened>
- Frequency: always | intermittent (<N>% of requests)
- Started: <date/event>
- Impact: <scope>
- Error: <paste exact error/trace>
```

## Step 3 — Reproduce and Isolate

1. Reproduce the issue in a controlled environment
2. Identify the exact file and line where the failure manifests
3. Trace backwards from the symptom to find the trigger

```bash
# Run the specific failing test
yarn test --testNamePattern="<test name>"
# or: go test -run TestName ./path/to/...
# or: gradle test --tests "ClassName.testName"
```

Read the relevant source files:

- The file at the error location (±50 lines around the failure)
- The caller chain leading to the failure
- Any configuration or environment variables involved

## Step 4 — Root Cause Analysis

Work through these categories systematically:

| Category      | Questions                                                       |
| ------------- | --------------------------------------------------------------- |
| Data          | Is input data in an unexpected format, null, or missing?        |
| State         | Is an object in an unexpected state when this code runs?        |
| Logic         | Is there an off-by-one, wrong condition, or incorrect operator? |
| Concurrency   | Could a race condition or shared mutable state cause this?      |
| Integration   | Is a downstream service returning unexpected data?              |
| Configuration | Is an env var missing, wrong, or loaded too late?               |
| Dependency    | Did a library version change behavior?                          |

Confirm your hypothesis:

- Can you write a failing test that reproduces the exact bug?
- Does your proposed fix make that test pass?
- Does fixing it cause any other tests to fail?

## Step 5 — Implement Minimal Fix

Apply the smallest possible change that resolves the root cause. Avoid refactoring unrelated code during a bug fix.

```bash
# Before fixing: write the regression test
# After fixing: run full test suite
yarn test
yarn build
```

Commit the fix:

```bash
git add <specific files>
git commit -m "fix(<scope>): <what was fixed>

Root cause: <one sentence>
Fix: <one sentence>

Closes #<issue-number>"
```

## Step 6 — Write Post-Mortem Document

**Output path**: `docs/post-mortems/<YYYY-MM-DD>-<slug>.md`

```markdown
# Post-Mortem: <Issue Title>

**Date**: YYYY-MM-DD
**Severity**: P1-Critical | P2-Major | P3-Minor
**Status**: Resolved
**Duration**: <how long the issue was active>

---

## Summary

<2-3 sentences: what happened, what the impact was, how it was resolved>

---

## Timeline

| Time  | Event                    |
| ----- | ------------------------ |
| HH:MM | Issue first observed     |
| HH:MM | Investigation started    |
| HH:MM | Root cause identified    |
| HH:MM | Fix deployed             |
| HH:MM | Issue confirmed resolved |

---

## Root Cause

<Precise technical explanation of WHY this happened. Include the specific file, line, and code logic that caused the failure.>

**Location**: `path/to/file.ts`, line N

**Faulty code**:
```

<code snippet showing the bug>
```

**Fixed code**:

```
<code snippet showing the fix>
```

---

## Impact

- Users/tenants affected: <description>
- Data integrity: <was any data corrupted? how was it remediated?>
- Downstream systems: <any cascading effects?>

---

## What Went Well

- <detection, response, or fix that worked effectively>

---

## What Could Be Improved

- <monitoring, testing, or process gaps that allowed this to occur>

---

## Action Items

| Action                                   | Owner  | Due Date |
| ---------------------------------------- | ------ | -------- |
| Add regression test for this scenario    | <team> | <date>   |
| Add monitoring alert for <metric>        | <team> | <date>   |
| Review similar code paths for same issue | <team> | <date>   |

````

## Step 7 — Gate Out (MANDATORY)

```bash
coder memory store "Bug Fix: <Issue Title>" "Root cause: <precise cause>. Fix: <what was changed>. Location: <file:line>. Regression test: <test name>. Prevention: <action items>." --tags "bug,<component>,<error-type>"
````

---

## Checklist

- [ ] `coder skill resolve` run
- [ ] `coder memory search` run (check if issue was seen before)
- [ ] Symptoms fully documented
- [ ] Issue reproduced in controlled environment
- [ ] Root cause identified with specific file and line
- [ ] Hypothesis validated with a failing test
- [ ] Minimal fix implemented
- [ ] Full test suite passes after fix
- [ ] Fix committed with clear message
- [ ] Post-mortem written at `docs/post-mortems/`
- [ ] `coder memory store` run with root cause and fix
