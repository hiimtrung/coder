---
description: debug
---

Help me identify and fix memory leaks or resource exhaustion issues.

1. **Gate In (MANDATORY)** — Run `coder memory search "memory leak <language/framework>"` to retrieve known leak patterns.
2. **Symptom Collection** — Gather logs, heap dumps, or monitoring data.
3. **Analyze & Fix** — Identify the root cause and implement a fix.
4. **Gate Out (MANDATORY)** — Run `coder memory store "Leak Analysis: <Context>" "<Root Cause and Preventive Measures>"` to save knowledge.