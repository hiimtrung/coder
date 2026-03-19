# Installation & Setup

> **TL;DR**: Install the CLI → start coder-node → run `coder login`

---

## Table of contents

1. [CLI (client)](#1-cli-client)
2. [coder-node (infrastructure)](#2-coder-node-infrastructure)
3. [Secure mode](#3-secure-mode)
4. [Connect the CLI (`coder login`)](#4-connect-the-cli-coder-login)
5. [Verify](#5-verify)
6. [Updates](#6-updates)
7. [Uninstall](#7-uninstall)

---

## 1. CLI (client)

Single binary, ~7 MB, no runtime dependencies.

### macOS / Linux

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/install.sh)"
```

The installer downloads the correct binary for your platform, installs it to `/usr/local/bin`, and launches the interactive `coder login` setup wizard.

**Options**

| Flag | Description |
|------|-------------|
| `--version v0.x.y` | Install a specific version instead of latest |
| `--skip-login` | Skip the setup wizard (useful in CI/automated environments) |

```bash
# Install specific version
/bin/bash -c "$(curl -fsSL .../install.sh)" -- --version v0.3.5

# CI — install only, no wizard
/bin/bash -c "$(curl -fsSL .../install.sh)" -- --skip-login
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/hiimtrung/coder/main/install.ps1 | iex
```

### Manual

1. Go to [GitHub Releases](https://github.com/hiimtrung/coder/releases).
2. Download the binary for your platform:

| Platform | File |
|----------|------|
| macOS (Apple Silicon) | `coder-darwin-arm64` |
| macOS (Intel) | `coder-darwin-amd64` |
| Linux x86-64 | `coder-linux-amd64` |
| Linux ARM64 | `coder-linux-arm64` |
| Windows x86-64 | `coder-windows-amd64.exe` |

3. Rename to `coder` (or `coder.exe`) and place in your `PATH`.

---

## 2. coder-node (infrastructure)

`coder-node` is the backend: vector embeddings, semantic search, and auth. It runs as a Docker stack.

**Requirements**: Docker ≥ 24 · Docker Compose · ~4 GB RAM (for the Ollama embedding model)

### Start (open mode)

```bash
curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/install-node.sh | sh
```

Creates `~/.coder-node/` and starts three containers:

| Container | Port | Role |
|-----------|------|------|
| `postgres` | — | Vector store (pgvector + full-text search) |
| `ollama` | — | Local embeddings — auto-pulls `mxbai-embed-large` (~700 MB) |
| `coder-node` | 50051 (gRPC) · 8080 (HTTP) | API layer + auth |

> First start takes 3–5 minutes while Ollama downloads the model.

---

## 3. Secure mode

By default coder-node runs in **open mode** — anyone with network access can call it.

**Secure mode** requires every developer to register their machine before using the node.

### Start with authentication enabled

```bash
curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/install-node.sh | sh -s -- --secure
```

This writes `SECURE_MODE=true` to `~/.coder-node/.env` before starting the stack.

### Retrieve the bootstrap token

On first startup the server generates a cryptographically random token, prints it once, and stores only its SHA-256 hash — the plaintext is never persisted.

```bash
docker logs coder_node 2>&1 | grep 'BOOTSTRAP TOKEN'
# BOOTSTRAP TOKEN (shown once): a3f9c2e1d4b87f6a2c…
#    Share this with clients so they can run: coder login
```

Share this token with developers via a secure channel (Slack DM, 1Password, etc.).

### Toggle secure mode after installation

```bash
# Enable
echo "SECURE_MODE=true"  > ~/.coder-node/.env
docker compose -f ~/.coder-node/docker-compose.yml up -d

# Disable
echo "SECURE_MODE=false" > ~/.coder-node/.env
docker compose -f ~/.coder-node/docker-compose.yml up -d
```

### Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTGRES_DSN` | _(compose internal)_ | PostgreSQL connection string (`sslmode=disable` required) |
| `OLLAMA_BASE_URL` | `http://ollama:11434` | Ollama endpoint |
| `OLLAMA_EMBEDDING_MODEL` | `mxbai-embed-large` | Embedding model name |
| `GRPC_PORT` | `50051` | gRPC service port |
| `HTTP_PORT` | `8080` | HTTP service port |
| `SECURE_MODE` | `false` | `true` to require Bearer token auth on all calls |

---

## 4. Connect the CLI (`coder login`)

Run on each developer machine after the node is up:

```bash
coder login
```

**Open mode server** — answer `N` to the auth question:

```
Choose protocol:
  1) gRPC  — recommended
  2) HTTP  — use this when the server runs --secure
Selection [1]: 1

Enter coder-node grpc URL [localhost:50051]: 192.168.1.10:50051

Requires authentication? (y/N): N

Configuration saved.
✓ Connection successful.
```

**Secure mode server** — answer `y`, enter the bootstrap token:

```
Choose protocol:
  1) gRPC  — recommended
  2) HTTP  — use this when the server runs --secure
Selection [1]: 2

Enter coder-node http URL [localhost:8080]: 192.168.1.10:8080

Requires authentication? (y/N): y
Enter bootstrap token: a3f9c2e1d4b87f…

Registering client as dev@company.com...
✓ Registered — access token saved to ~/.coder/config.json

✓ Connection successful.
```

> **Protocol note**: when registering with a secure-mode server, use **HTTP** (port 8080) because the registration endpoint is HTTP-only. For everyday `skill search` / `memory store` commands you can switch back to gRPC — the access token is injected automatically into gRPC metadata.

The token is stored in `~/.coder/config.json` and attached to every future call with no extra steps required.

---

## 5. Verify

```bash
# Should return [] or a list — not a connection error
coder memory list --limit 1

# Health check (shows whether secure mode is active)
curl http://your-server:8080/health
# {"status":"ok","secure_mode":true}
```

---

## 6. Updates

### Update the CLI

```bash
coder self-update
```

### Update coder-node

```bash
curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/update-node.sh | sh
# or manually:
cd ~/.coder-node && docker compose pull && docker compose up -d
```

---

## 7. Uninstall

### CLI

```bash
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/uninstall.sh | sh -s -- --clear-data

# Windows (PowerShell)
irm https://raw.githubusercontent.com/hiimtrung/coder/main/uninstall.ps1 | iex -Arguments "-ClearData"
```

### coder-node

```bash
# Stop containers, keep database volumes
curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/uninstall-node.sh | sh

# Stop containers AND delete all data (irreversible)
curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/uninstall-node.sh | sh -s -- --clear-data
```
