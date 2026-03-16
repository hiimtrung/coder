---
description: Store reusable guidance in the knowledge memory service.
---

When I say "remember this" or want to save a reusable rule, help me store it in the knowledge memory service.

1. **Gate In (MANDATORY)** — Run `coder skill search "<topic>"` to check if similar best practices already exist in the skill DB, then run `coder memory search "<topic>"` to check if similar knowledge already exists in memory before adding a new item.
2. **Capture Knowledge** — If not already provided, ask for: a short explicit title (5-12 words), detailed content (markdown, examples encouraged), optional tags (keywords like "api", "testing"), and optional scope (`global`, `project:<name>`, `repo:<name>`). If vague, ask follow-ups to make it specific and actionable.
3. **Validate Quality** — Ensure it is specific and reusable (not generic advice). Avoid storing secrets or sensitive data. Cross-reference with skill results from Gate In to avoid duplicating existing skill knowledge.
4. **Gate Out (MANDATORY)** — Call `coder memory store` with title, content, tags, scope.
5. **Confirm** — Summarize what was saved and offer to store more knowledge if needed.
