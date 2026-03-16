# 🤖 coder

[![Build & Release](https://github.com/hiimtrung/coder/actions/workflows/release.yml/badge.svg)](https://github.com/hiimtrung/coder/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/hiimtrung/coder)](https://goreportcard.com/report/github.com/hiimtrung/coder)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**coder** is a universal engineering intelligence system for AI agents. It distributes centralized skills, architecture standards, and semantic memory to any project.

Built in Go. Cross-platform. Single binary.

---

## ⚡ Quick Start

### 1. Install CLI
```bash
# macOS / Linux
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/install.sh)"

# Windows (PowerShell)
irm https://raw.githubusercontent.com/hiimtrung/coder/main/install.ps1 | iex
```

### 2. Set up Infrastructure
```bash
# Install coder-node (Postgres + Ollama + gRPC)
curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/install-node.sh | sh
```

### 3. Apply to a Project
```bash
cd /my/awesome-project
coder install fullstack
```

---

## 🎓 Why Coder?

Most AI agents act in a vacuum. **Coder** provides them with:

- **Institutional Knowledge**: Distribute your team's senior-level architecture rules.
- **Advanced RAG**: Agents semantically query 20+ specialized skills (NestJS, Go, Java, etc.).
- **Long-term Memory**: Cross-project semantic retrieval for decisions and fixes.
- **Workflow Enforcement**: Standardized delivery lifecycles (BA → Dev → QA).

---

## 📂 Project Structure

After running `coder install <profile>`, your project will be equipped with the "thinking engine":

```
.agents/
  workflows/            # AI Agent execution steps (slash commands)
  rules/                # Project-specific coding standards
  .coder.json           # Manifest: tracks profile and version

.github/
  agents/
    coder.agent.md      # Coder Agent persona & gate definitions
  copilot-instructions.md # Combined context for GitHub Copilot
```

> [!TIP]
> **Where are the skills?** Knowledge (NestJS, Go, etc.) is now managed entirely via the **Skill RAG System**. This keeps your repository clean while giving agents access to searchable knowledge via `coder skill search`.

---

## 📚 Documentation

Dive deeper into the system:

- **[🏗️ Architecture](docs/architecture.md)** — High-level overview and Mermaid diagrams.
- **[📥 Installation](docs/installation.md)** — Detailed setup for CLI and Node.
- **[⌨️ CLI Reference](docs/cli.md)** — Full command list and flag reference.
- **[🎯 Skill RAG System](docs/skill_system.md)** — How the vector intelligence works.
- **[💾 Memory System](docs/memory_system.md)** — Semantic memory and project context.
- **[🛠️ Development](docs/development.md)** — Building from source and CI/CD.

---

## 🚀 Key Commands

| Task | Command |
|------|---------|
| **Setup** | `coder login` |
| **New Project** | `coder install <profile>` |
| **Sync Skills** | `coder update` |
| **Search Knowledge** | `coder skill search "topic"` |
| **Save Memory** | `coder memory store "title" "data"` |
| **Update CLI** | `coder self-update` |

---

## 🤝 Contributing

We welcome issues and pull requests! Please check the [Development Guide](docs/development.md) for details.

---

**Built with ❤️ in Go for the AI-First Era.**
