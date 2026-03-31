---
description: Debug memory leaks and resource exhaustion — structured profiling, root cause identification, and fix with preventive measures.
---

# Workflow: Debug Memory Leak

A systematic approach to identifying and fixing memory leaks, resource exhaustion, goroutine leaks, connection pool saturation, and similar resource management failures.

## When to Use

- Application memory usage grows continuously and does not stabilize
- Connection pool exhaustion under load
- Goroutine, thread, or file descriptor count grows unbounded
- OOM (out of memory) kills in production
- Response latency degrades over time without traffic increase

## Step 1 — Context Load (MANDATORY)

```bash
coder skill resolve "memory leak <language or framework>" --trigger error-recovery --budget 3
coder memory search "memory leak <component or service>"
```

## Step 2 — Symptom Collection

Gather diagnostic data before forming a hypothesis:

```bash
# Process memory over time (check for monotonic growth)
# Node.js:
node --inspect app.js
# Then: chrome://inspect, take heap snapshots at T+0, T+5min, T+15min

# Go:
curl localhost:6060/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# Java:
jmap -dump:format=b,file=heap.hprof <pid>
# Then: open in VisualVM or Eclipse MAT

# Docker / general:
docker stats <container>
```

Document:
- Resource type: memory | file descriptors | goroutines | DB connections | event listeners
- Growth pattern: linear, step-function, or event-triggered
- First observed: after which deployment or traffic change
- Environment: local only, staging, production

## Step 3 — Classify the Leak Type

| Leak Type | Symptoms | Common Causes |
|-----------|----------|---------------|
| Memory (heap) | RSS grows, GC doesn't recover it | Unbounded caches, circular refs, large buffers not released |
| Goroutine / thread | goroutine/thread count grows | Missing channel close, blocking call without timeout, leaked worker |
| DB connection | connection pool saturation | Missing `defer rows.Close()`, unclosed transactions, no connection timeout |
| File descriptor | "too many open files" error | Missing `defer file.Close()`, socket not closed on error path |
| Event listener | memory grows after repeated subscribe calls | Listener not removed on component unmount / object destruction |

## Step 4 — Isolate the Source

For memory leaks — compare heap snapshots:
1. Take snapshot at startup (T+0)
2. Run the suspected operation N times
3. Take snapshot (T+N)
4. Diff: what objects grew and were not garbage collected?

For connection/goroutine leaks:
```bash
# Go goroutines:
curl localhost:6060/debug/pprof/goroutine?debug=1

# Node.js event emitters:
process.listenerCount('event')

# DB connections (PostgreSQL):
SELECT count(*), state FROM pg_stat_activity GROUP BY state;
```

Read the code path that runs during the growth period. Look for:
- Objects added to a collection but never removed
- `defer` or `finally` blocks missing on the error path
- Callbacks registered without a corresponding unregister
- Infinite retry loops without backoff or termination condition

## Step 5 — Implement Fix

Apply the minimum change to close the resource leak:

```typescript
// BEFORE: DB connection not closed on error path
async function getUser(id: string) {
  const conn = await pool.connect();
  const result = await conn.query('SELECT * FROM users WHERE id = $1', [id]);
  conn.release(); // not called if query throws
  return result.rows[0];
}

// AFTER: release guaranteed via try/finally
async function getUser(id: string) {
  const conn = await pool.connect();
  try {
    const result = await conn.query('SELECT * FROM users WHERE id = $1', [id]);
    return result.rows[0];
  } finally {
    conn.release(); // always called
  }
}
```

```go
// BEFORE: rows not closed on error
rows, err := db.Query("SELECT id FROM items WHERE company_id = $1", companyID)
for rows.Next() { ... }

// AFTER: defer close immediately after error check
rows, err := db.Query("SELECT id FROM items WHERE company_id = $1", companyID)
if err != nil { return err }
defer rows.Close()
for rows.Next() { ... }
```

Run the full test suite after fixing:
```bash
yarn test && yarn build
# or: go test ./... && go build ./...
```

Commit:
```bash
git commit -m "fix(<scope>): close resource in all code paths to prevent <resource> leak

Root cause: <description>
Fix: <what was changed>
Regression test: <test name>"
```

## Step 6 — Write Post-Mortem

**Output path**: `docs/post-mortems/<YYYY-MM-DD>-<slug>-leak.md`

```markdown
# Post-Mortem: <Resource> Leak in <Component>

**Date**: YYYY-MM-DD
**Severity**: P1 | P2 | P3
**Status**: Resolved

## Summary
<2-3 sentences: what leaked, what the impact was, how it was fixed>

## Root Cause
<Exact mechanism. Which resource, which code path, why it wasn't released.>

**Location**: `path/to/file`, line N

## Fix
<What was changed and why it prevents the leak>

## Prevention Measures
- <coding practice to prevent recurrence>
- <linter rule or review checklist item to add>
- <monitoring alert to add>
```

## Step 7 — Gate Out (MANDATORY)

```bash
coder memory store "Leak: <Resource> in <Component>" "Resource type: <type>. Root cause: <cause>. Fix: <what changed>. Prevention: <measures added>. Location: <file:line>." --tags "leak,memory,<language>,<component>"
```

---

## Checklist

- [ ] `coder skill resolve` run with language/framework context
- [ ] `coder memory search` run for similar past leaks
- [ ] Diagnostic data collected (heap snapshot, goroutine count, etc.)
- [ ] Leak type classified
- [ ] Source isolated to specific file and code path
- [ ] Fix applied — resource closed in ALL code paths (including error paths)
- [ ] Full test suite passes
- [ ] Fix committed
- [ ] Post-mortem written
- [ ] `coder memory store` run with root cause and prevention
