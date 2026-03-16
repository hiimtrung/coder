# Coder Skill RAG Architecture

The `coder` CLI implements an advanced **Retrieval-Augmented Generation (RAG) System** for managing AI skills. Instead of blindly passing hundreds of markdown files to an LLM context—which is slow, costly, and prone to "lost-in-the-middle" issues—the new architecture chunks skills, embeds them, and searches them highly efficiently.

## 1. High-Level Architecture

```mermaid
flowchart TD
    subgraph CLI["coder CLI Client"]
        A[User specifies skill operation] --> B{Command Router}
        B -->|coder skill search| C[gRPC / HTTP Client]
        B -->|coder skill ingest| C
        B -->|coder update| C
    end

    subgraph Node["coder-node (Docker)"]
        D[gRPC / HTTP API]
        E[Skill Ingestor]
        F[Skill Service (Store)]
        
        D -->|Ingest/Search| E
        D -->|List/Delete| F
        E -->|Chunks & Embeddings| F
    end

    subgraph Storage["Databases"]
        G[(PostgreSQL + pgvector)]
        H[Ollama / OpenAI Embeddings]
    end

    subgraph Sources["Skill Knowledge Sources"]
        I[Local ~/.coder/.agents/skills/]
        J[GitHub: antigravity-awesome-skills]
        K[GitHub: ui-ux-pro-max-skill]
    end

    F -->|Persist Skills & Chunks| G
    E -->|Generate Vector Embeddings| H
    C -.->|Reads local files| I
    C -.->|Fetches APIs| J
    C -.->|Fetches APIs| K
    E -.->|Fetches API index| J
```

## 2. The Data Model

Skill data is broken down hierarchically to ensure search granularity.

1. **Skill (Parent)**: Represents a single conceptual skill (e.g., `nestjs`, `architecture`, `react-best-practices`). Contains metadata (`name`, `category`, `source`, `risk`, `tags`).
2. **SkillChunk (Child)**: The actual chunks of knowledge. A single skill will be broken down into many chunks consisting of:
   - The general **Description** (from the top of `SKILL.md`).
   - Various **Sections** (split by `##` headers in `SKILL.md`).
   - Individual **Rules** (fetched from the `rules/*.md` directory).

When a search is performed, the chunks are matched using cosine similarity, but the search returns the results grouped by their parent Skill for better context.

### Database Schema (PostgreSQL `pgvector`)
- Table `skills`: Stores the high-level metadata of the skill (JSON tags, source URL, last updated timestamps).
- Table `skill_chunks`: Stores the title, content, and the high-dimensional `vector(1024)` representation of the text using the Ollama `mxbai-embed-large` (or OpenAI `text-embedding-3-small`) model.

## 3. The Ingestion Pipeline

When you run `coder skill ingest --source local` (or trigger it automatically via `coder update`):

1. **Walking**: The CLI walks the `.agents/skills` embedded file system.
2. **Parsing**: The `ParseSkillMD` engine extracts the YAML frontmatter (for `description`, `category`, and `tags`) and splits the rest of the markdown by `##` headings.
3. **Rule Collection**: It traverses the `rules/` directory inside that skill and treats each markdown file as an independent "Rule Chunk".
4. **Transport**: The payload (Name, Metadata, and raw Sections/Rules) is sent to `coder-node` via gRPC (or HTTP fallback).
5. **Embedding Generation**: `coder-node` calls the configured embedding provider to convert text strings into mathematical floating-point vectors.
6. **Storage**: The vectors and their raw text are saved into PostgreSQL `skill_chunks` and associated with the parent `skill`.

## 4. GitHub Remote Integration

The system isn't limited to local `.agents/skills`. It has first-class support for retrieving and embedding skills directly from specialized external open-source repositories:

```bash
coder skill ingest --source github --repo sickn33/antigravity-awesome-skills
```

**How it works:**
1. The CLI queries the root `skills_index.json` from the target repository.
2. It lists out all specified skills, categories, and relative paths.
3. Over parallel HTTP requests, it fetches the raw `SKILL.md` content from GitHub.
4. It passes this remote content directly to the local `coder-node` pipeline for immediate embedding.

## 5. RAG Execution Flow

When an AI agent (or user) needs context for an operation:

1. **Query**: The agent calls `coder skill search "How do I implement event-driven architecture?"`
2. **Vectorization**: The query string is converted to a vector embedding.
3. **Similarity Search**: `pgvector` calculates the closest cosine-distance chunks across the entire `skill_chunks` table.
4. **Grouping**: The results are folded back into their parent elements. If `architecture` has 3 matching rules, they are bundled together.
5. **Output**: The system returns pure, highly-relevant markdown excerpts strictly answering the query.

This enables advanced agentic engineering by avoiding token limitations, injecting context dynamically, and ensuring coding rules are adhered to consistently.
