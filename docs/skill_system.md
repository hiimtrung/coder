# 🎯 Skill RAG System

The **Skill RAG System** is the "brain" of the `coder` ecosystem. It transforms flat markdown instructions into searchable vector embeddings, allowing AI agents to retrieve exactly what they need for a specific task.

## 🖇️ Ingestion Process

When a skill is ingested, the system follow these steps:

1. **Discovery**:
    - **Local**: Reads `SKILL.md` and `rules/*.md` from the embedded filesystem.
    - **GitHub**: Fetches file contents via the GitHub API. It automatically discovers all `.md` files in the `rules/` directory of the remote repo.
2. **Parsing**:
    - The `SKILL.md` is parsed into sections (Introduction, Guidelines, Tips).
    - Each section becomes a **Chunk**.
3. **Deduplication**:
    - Every chunk is hashed (SHA256).
    - If a chunk with the same hash already exists in the database for that skill, it is **skipped**. This saves API costs and time.
4. **Vectorization**:
    - New/Changed chunks are sent to the embedding provider (Ollama or OpenAI).
    - The resulting 1024-dimension vector is stored in PostgreSQL.

## 📡 Remote GitHub Skills

You can turn any GitHub repository into a skill source for your team.

### Repository Structure
Requirements for a valid skill repo:
- `SKILL.md`: Main description and core rules.
- `rules/*.md`: (Optional) Specialized sub-rules.
- `skills_index.json`: (Required for multi-skill repos) Mapping of skill names to their subdirectory paths.

### Commands
```bash
# Ingest all skills from a community repo
coder skill ingest --source github --repo user/repo
```

## 🔍 Semantic Retrieval

AI agents usually run searches like:
`coder skill search "how to implement a singleton in golang"`

The system:
1. Embeds the user query.
2. Performs a **Cosine Similarity** search in `pgvector`.
3. Returns the top-N sections that most closely match the intent.

### Agent usage
In the [3-Gate System](architecture.md#agent-reasoning), this happens at **Gate 1**.
