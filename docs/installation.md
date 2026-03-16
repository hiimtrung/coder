# 📥 Installation & Setup

This guide covers how to set up the **coder** CLI and its backend infrastructure (`coder-node`).

## 🖥️ Coder CLI (Client)

The CLI is a single binary (~7MB) and requires no dependencies.

### Automatic Installation

#### macOS / Linux
The installer is interactive and will help you verify your `coder-node` connection.
```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/install.sh)"
```

#### Windows (PowerShell)
```powershell
irm https://raw.githubusercontent.com/hiimtrung/coder/main/install.ps1 | iex
```

### Manual Installation
1. Go to [GitHub Releases](https://github.com/hiimtrung/coder/releases).
2. Download the binary for your platform (e.g., `coder-darwin-arm64` for Apple Silicon).
3. Rename to `coder` (or `coder.exe`) and add to your `PATH`.

---

## 🐳 Coder Node (Infrastructure)

The `coder-node` handles vector embeddings and database management. It is best run via Docker.

### Requirements
- **Docker** and **Docker Compose**.
- **PostgreSQL** (included in Compose, or use external).
- **Ollama** (for local embeddings) OR **OpenAI API Key**.

### Quick Self-Hosted Setup

We provide a one-command installer for the infrastructure:
```bash
curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/install-node.sh | sh
```

This creates a `~/.coder-node/` directory and starts:
1. **`postgres`**: With `pgvector` enabled for embedding storage.
2. **`ollama`**: Pre-configured to pull the `mxbai-embed-large` model.
3. **`coder-node`**: The gRPC/HTTP service layer.

### Manual Configuration

You can configure the service via environment variables in `docker-compose.yml`:

| Variable | Description | Default |
|----------|-------------|---------|
| `POSTGRES_DSN` | Connection string for Postgres | Requirements: `sslmode=disable` |
| `OLLAMA_BASE_URL` | URL for Ollama server | `http://ollama:11434` |
| `OLLAMA_EMBEDDING_MODEL` | Model used for vectors | `mxbai-embed-large` |
| `GRPC_PORT` | Port for gRPC service | `50051` |
| `HTTP_PORT` | Port for HTTP service | `8080` |

---

## 🔐 Client Configuration

Once the node is running, link your CLI to it:

```bash
coder login
```

Choose your protocol (**gRPC** is recommended for speed) and enter the URL (e.g., `localhost:50051`). This information is stored in `~/.coder/config.json`.

---

## 🧪 Verifying the Setup

Check that everything is working by querying the memory:

```bash
# Returns an empty list [] if working, or a connection error if not.
coder memory list --limit 1
```

---

## 🔄 Updates

### Update the CLI
```bash
coder self-update
```

### Update the Node
```bash
cd ~/.coder-node
docker compose pull
docker compose up -d
```
