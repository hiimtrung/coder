---
description: Debug memory leaks and resource exhaustion issues.
---

Help me identify and fix memory leaks or resource exhaustion issues.

1. **Gate In (MANDATORY)** — Run `coder skill search "memory leak <language/framework>"` to retrieve profiling techniques and known leak patterns, then run `coder memory search "memory leak <language/framework>"` to retrieve previous leak investigations.
2. **Symptom Collection** — Gather logs, heap dumps, or monitoring data. Identify the resource being leaked (memory, file handles, connections, goroutines).
3. **Analyze & Fix** — Apply skill-specific profiling patterns from Gate In. Identify the root cause (missing close/dispose, circular references, event listener accumulation, unbounded caches). Implement a fix with proper resource cleanup.
4. **Gate Out (MANDATORY)** — Run `coder memory store "Leak Analysis: <Context>" "<Root Cause and Preventive Measures>"` to save knowledge.