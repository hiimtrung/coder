# ⌨️ CLI Command Reference

This document provides a detailed reference for all `coder` commands.

## 📦 Project Skills Management

### `coder install <profile|skill>`
Installs the **Agent Engine** (workflows, rules, and agent definitions) into a project.

> [!NOTE]
> **Skills** (NestJS, Go, etc.) are NOT installed as local files. They are managed via the **Skill RAG (Vector DB)**. Use this command to scaffold the local agent infrastructure so it can perform searches.

- **Arguments**:
    - `profile`: `be`, `fe`, `fullstack`, or `all`.
    - `skill`: Individual skill name (e.g., `nestjs`).
- **Flags**:
    - `-t, --target <dir>`: Target directory (default: `.`).
    - `-f, --force`: Overwrite existing files.
    - `--dry-run`: Preview changes without writing.

### `coder update [profile]`
Re-syncs local workflows/rules and **automatically triggers a skill ingestion** to ensure your Vector DB is up to date with the embedded knowledge.

### `coder list [profile]`
Lists available profiles and skills.
- `coder list`: Show summary of all profiles.
- `coder list be`: Show detailed skills inside the `be` profile.

---

## 🧠 Skill RAG (Vector DB)

Commands to manage the centralized intelligence database.

### `coder skill ingest`
Imports skill data into the vector database for semantic search.
- `--source local`: Ingests the 20+ built-in skills.
- `--source github --repo <user/repo>`: Ingests skills directly from a remote GitHub repository.

### `coder skill search <query>`
Performs a semantic similarity search across all ingested skills.
- `--limit <n>`: Number of results (default: 5).

### `coder skill list`
Shows all skills currently stored in the vector database.

### `coder skill info <name>`
Shows metadata and chunk statistics for a specific skill.

### `coder skill delete <name>`
Removes a skill from the vector database.

---

## 💾 Semantic Memory

Manage cross-project contextual knowledge.

### `coder memory store <title> <content>`
Stores a piece of knowledge.
- `--tags <t1,t2>`: Categorize for better retrieval.
- `--type <type>`: (e.g., `rule`, `pattern`, `fact`).

### `coder memory search <query>`
Retrieves relevant knowledge using vector similarity.

### `coder memory list`
Shows the most recent entries in memory.

### `coder memory delete <id>`
Removes a specific entry.

---

## 🛠️ System Commands

### `coder version`
Displays CLI version, commit hash, and build date.

### `coder login`
Interactive setup to configure your `coder-node` connection.

### `coder check-update`
Checks if a newer version of the CLI is available on GitHub.

### `coder self-update`
Automatically downloads and replaces the current binary with the latest released version.
