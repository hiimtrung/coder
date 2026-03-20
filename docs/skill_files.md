# Skill Asset Management (SkillFiles)

The `coder` system supports skills that require more than just text-based instructions. The **SkillFiles** architecture allows skills to carry binary assets, scripts, and data files that are stored in the vector database and extracted to the local machine at runtime.

## Supported File Types

The system automatically scans for and ingests files with the following extensions:
- **Scripts**: `.py`, `.js`, `.cjs`, `.sh`, `.sql`
- **Data**: `.csv`, `.json`, `.txt`
- **Documentation**: `.md`

## Directory Structure

To be automatically recognized during ingestion, files should be placed in one of these subdirectories within a skill:
- `scripts/`: Executable scripts (Python, Node, Bash).
- `data/`: Reference datasets (CSV, JSON).
- `references/`: Supplemental documentation or code examples.
- `templates/`: Boilerplate or code generation templates.

## Workflow: Ingestion & Extraction

### 1. Ingestion
When running `coder skill ingest --include-files`, the CLI reads these files, calculates their SHA256 hashes, and sends them to the `coder-node` to be stored as BLOBs in the PostgreSQL `skill_files` table.

### 2. Manual Pull
Users can manually extract assets for a specific skill:
```bash
coder skill cache pull ui-ux-pro-max
```
This extracts the files to `~/.coder/cache/ui-ux-pro-max/`.

### 3. Automated Runtime Extraction (RAG)
When an AI agent performs a `coder skill search`, the CLI identifies that the matching skill has associated files. 
- It checks the local `~/.coder/cache/` for that skill.
- If the files are missing or the `content_hash` has changed, the CLI automatically pulls the latest versions from the server.
- The CLI then rewrites paths in the AI instructions (e.g., `run scripts/search.py`) to point to the absolute path on the local machine (e.g., `/Users/user/.coder/cache/my-skill/scripts/search.py`).

## Management Commands

| Command | Description |
|---------|-------------|
| `coder skill cache list` | View all cached skills and their file counts |
| `coder skill cache pull --all` | Pre-fetch all assets for all ingested skills |
| `coder skill cache clear <name>` | Securely remove cached assets from the local machine |

## Benefits
- **Portability**: Your skills carry their own tools and data, regardless of which machine you are working on.
- **Up-to-Date Tools**: When you update a script in the repository, all connected clients get the new version automatically via the hash-check.
- **Reduced Context**: Agents don't need to read the *content* of large data files; they just need the *path* to execute them or read them locally.
