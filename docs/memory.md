# Memory Feature Usage Guide (Vector DB)

The **Memory** feature allows `coder` to intelligently remember information using a Vector Database. Unlike traditional keyword search, a Vector DB enables **semantic search**, allowing the tool to understand your intent even if you don't use the exact keywords.

## 🛠 Configuration

To use this feature, you need to configure an embedding provider in your configuration file (default at `~/.coder/config.json`).

### Supported Providers:
1. **OpenAI**: Uses the `text-embedding-3-small` model (recommended).
2. **Local (Ollama)**: A **100% offline** solution, no API Key required and completely free.

---

## 🏠 Fully Local Setup with Ollama (Recommended)

For a local-only setup without an API Key, use **Ollama**. This is the best solution for pulling embedding models to your local machine.

### Step 1: Install and Pull Model
Install Ollama from [ollama.com](https://ollama.com), then run the command to pull a specialized embedding model:
```bash
ollama pull nomic-embed-text
```

### Step 2: Configure `coder`
Set `OPENAI_BASE_URL` to point to your local Ollama instance:

```bash
export OPENAI_BASE_URL=http://localhost:11434/v1
export OPENAI_EMBEDDING_MODEL=nomic-embed-text
export OPENAI_API_KEY=ollama # Any value works
```

### Step 3: Configuration File (Optional)
Alternatively, configure it in `~/.coder/config.json`:
```json
{
  "memory": {
    "provider": "openai",
    "base_url": "http://localhost:11434/v1",
    "model": "nomic-embed-text",
    "api_key": "ollama"
  }
}
```

> [!NOTE]
> `coder` is designed to be extremely lightweight (~7MB) and thus does not embed the models (which can be hundreds of MBs) directly. Using Ollama allows you to flexibly switch models without bloating the CLI tool.

---

## 🚀 Basic Commands

### 1. Store Knowledge
Save a piece of information to memory.

```bash
coder memory store "Variable Naming Rules" "Use camelCase for local variables, PascalCase for Structs and Interfaces in Go." --tags "golang,coding-style"
```

### 2. Semantic Search
Search for information based on meaning.

```bash
coder memory search "how to name things in go"
```
*The result will return "Variable Naming Rules" even though the keywords do not match exactly.*

### 3. List entries
List saved items.

```bash
coder memory list --limit 10
```

### 4. Delete entry
Remove an item from memory.

```bash
coder memory delete <uuid>
```

---

## 🏗 Technical Architecture

- **Database**: SQLite (pure Go) ensures zero CGO/external C library dependencies.
- **Vector Search**: Uses a **Cosine Similarity** algorithm optimized for CLI environments.
- **Embedding**: Converts text into 1536-dimensional vectors (OpenAI) for comparison.
- **Storage**: Data is stored at `~/.coder/memory.db`.

---

## 💎 Advanced Features

### 🧩 Automatic Chunking
`coder` automatically splits large documents into multiple **overlapping chunks** (default: 1000 chars with 200 chars overlap). This ensures:
- Better semantic precision for long documents.
- Compatibility with model token limits.
- Precise retrieval of relevant sections instead of entire files.

### 🧹 Memory Compaction & Re-vectoring
Maintain your memory health with the `compact` command:

```bash
# Re-generate all embeddings (useful if you switch models)
coder memory compact --revector

# Remove near-duplicate entries (automatic threshold)
coder memory compact
```
