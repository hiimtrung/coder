# coder

A **CLI tool** for installing AI agent skills, rules, and workflows into any project for GitHub Copilot & Antigravity.

Built in Go, cross-platform, zero dependencies. Single binary (~7MB) with embedded skills + rules + workflows.

```bash
# Install
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/install.sh)"
# or (Windows PowerShell)
irm https://raw.githubusercontent.com/hiimtrung/coder/main/install.ps1 | iex

# Use
cd /path/to/your/project
coder install be        # backend
coder install fe        # frontend
coder install fullstack # full-stack
coder update            # re-sync skills (reads last profile from manifest)
coder check-update      # check for CLI updates
coder self-update       # auto-upgrade to latest CLI version
```

---

## 🎯 What It Does

The **coder CLI** distributes a **centralized engineering knowledge system** to your projects with an **Advanced Semantic Memory**:

- **20+ Skills**: Language-specific expertise (NestJS, Java, Go, Python, Rust, C, Dart, React, Vue, etc.)
- **Architecture Guidance**: Clean Architecture, event-driven design, multi-tenancy patterns
- **Professional Standards**: Error codes, testing, documentation, UI/UX design systems
- **AI Agent Workflows**: BA analysis, development, QA, full lifecycle delivery
- **Skill RAG System**: Skills are chunked, embedded, and stored in a Vector DB for highly accurate semantic retrieval.
- **Semantic Memory Integration**: Built-in Memory management using PostgreSQL (`pgvector`) and Ollama/OpenAI.
- **One-command setup**: `coder install be` → project gets `.agents/` + `.github/copilot-instructions.md`

---

## 📦 Installation & Setup

### Requirements
- **PostgreSQL** with `pgvector` extension (for external semantic memory)
- **Ollama** (remote server or local container) running `mxbai-embed-large`

### Quick Start (macOS / Linux)
The installer is now **interactive** and will verify your connections:
```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/install.sh)"
```

### Windows (PowerShell)
```powershell
irm https://raw.githubusercontent.com/hiimtrung/coder/main/install.ps1 | iex
```

### Manual Download
Download binary from [GitHub Releases](https://github.com/hiimtrung/coder/releases) and add to `PATH`.

---

## 🚀 Usage

### Profiles

| Profile | For | Skills Included |
|---------|-----|---|
| `be` | Backend (NestJS, Java, Go, Python, Rust, C, Dart) | Core + language-specific |
| `fe` | Frontend (React, Next.js, Vue, Svelte, React Native) | Core + UI frameworks + design |
| `fullstack` | Full-stack projects | be + fe |
| `all` | Everything | All skills, rules, workflows |

Or install **individual skills** by name:
```bash
coder install golang
coder install nestjs
coder install react-best-practices
```

### Commands

```bash
# Install skills to a project
coder install <profile> [flags]
  --target, -t <dir>  Target directory (default: current directory)
  --force, -f         Overwrite existing files
  --dry-run           Preview without modifying

# Update installed skills (reads from manifest if no profile given)
coder update [profile] [flags]
  --target, -t <dir>
  --dry-run

# List profiles and skills
coder list              # all profiles + skills
coder list be           # details for 'be' profile

# Manage Skill Vector DB
coder skill ingest --source local       # Ingest local skills into Vector DB
coder skill ingest --source github --repo sickn33/antigravity-awesome-skills
coder skill search "react testing"      # Semantic search across ingested skills
coder skill list                        # View all ingested skills
coder skill info architecture           # View specific skill details

# Manage Context Semantic Memory
coder memory store "rule" "data..."
coder memory search "routing"

# Version & updates
coder version           # show version + commit + build date
coder check-update      # check for new CLI version on GitHub
coder self-update       # download and auto-upgrade CLI

# Help
coder help
```

### Examples

```bash
# Install backend stack to current directory
coder install be

# Install frontend to a specific path
coder install fe --target ./frontend-app

# Preview without making changes
coder install fullstack --dry-run --target /tmp/test

# Update all skills to latest (from last install)
cd /my/project
coder update

# Update to a different profile
coder update fe --force

# Check for CLI updates
coder check-update
```

---

## 🧠 Semantic Memory & Skill RAG System

`coder` includes a professional-grade **Cognitive Memory Framework** (RAG) that allows AI agents to "remember" cross-project patterns, decisions, and utilize highly-detailed skills dynamically.

> 📚 **Deep Dive**: For a detailed technical breakdown of how the embedding and retrieval system works (including integrations with external repositories), read the [Skill RAG Architecture Documentation](docs/skill_architecture.md).

### Configuration
During installation (or via `coder login`), you will be prompted for:
- **Protocol**: Choose between `gRPC` (default) or `HTTP` for communicating with `coder-node`.
- **Node URL**: The endpoint for your `coder-node` instance (e.g. `localhost:50051`).
- **Database**: The node uses PostgreSQL with `pgvector` for embeddings.
- **Embeddings**: Handled automatically by the node using local models (Ollama) or OpenAI.

### Managing AI Skills (RAG)
Skills are normally installed as flat markdown files, but with `coder skill`, you can ingest them into the vector database. This allows agents to semantically search massive rule sets instead of reading every file.

```bash
# 1. Ingest your local skills (automatically runs during 'coder update')
coder skill ingest --source local

# 2. Ingest remote skills directly from GitHub repositories!
coder skill ingest --source github --repo sickn33/antigravity-awesome-skills
coder skill ingest --source github --repo nextlevelbuilder/ui-ux-pro-max-skill

# 3. Perform semantic searches against the knowledge base
coder skill search "how to handle panics in golang" --limit 3

# 4. Explore ingested skills
coder skill list --category core
coder skill info nestjs
```

### Managing Project Memory
Store specific facts, rules, or decisions made during the project lifecycle:

```bash
# Store new knowledge
coder memory store "Project Pattern" "Context here..." --type "rule" --meta '{"entity_id": "core"}'

# Search with semantic similarity
coder memory search "How to handle errors" --limit 3

# Maintenance
coder memory list            # show recent entries
coder memory compact        # optimize and de-duplicate
```

---

## 📂 What Gets Installed

After `coder install be`, your project structure includes:

```
.agents/
  skills/
    architecture/
      SKILL.md
      rules/
        principles.md
    nestjs/
      SKILL.md
      rules/
        typescript.md
        errors.md
    golang/
      ...
    [19 more skills]
  rules/
    general.instructions.md  # unified coding standards
  workflows/
    new-requirement.md
    execute-plan.md
    qa-testing.md
    full-lifecycle-delivery.md
    remember.md
    capture-knowledge.md
    [8 more workflows]
  .coder.json           # manifest: version, profile, skills, installed_at

.github/
  copilot-instructions.md   # GitHub Copilot context (auto-generated from rules)
  agents/
    coder.agent.md      # custom agent definition
```

**Manifest** (`.agents/.coder.json`):
```json
{
  "version": "v0.1.0",
  "profile": "be",
  "skills": [
    "architecture",
    "general-patterns",
    ...
  ],
  "installed_at": "2026-03-02T03:45:34Z"
}
```

This allows `coder update` to re-sync skills without specifying a profile.

---

### From Source
```bash
git clone https://github.com/hiimtrung/coder.git
cd coder
go mod tidy
make build
make install-user  # or 'make install' for /usr/local/bin (requires sudo)
```

---

## 🛠️ Development & Building

### Prerequisites
- Go 1.26.0+
- **PostgreSQL** with `pgvector` extension
- **Ollama** server running `mxbai-embed-large`
- Make (optional)

### Build for current platform
```bash
make build          # creates dist/coder
```

### Build for all platforms
```bash
make build-all      # creates:
                    # - dist/coder-darwin-amd64
                    # - dist/coder-darwin-arm64
                    # - dist/coder-linux-amd64
                    # - dist/coder-linux-arm64
                    # - dist/coder-windows-amd64.exe
```

### Create a release (push to GitHub)
```bash
make tag VERSION=v0.1.0  # creates git tag and pushes
                         # GitHub Actions automatically builds & releases
```

### Clean up
```bash
make clean
```

---

## 🔄 Version Management

### Bumping Version

The version is defined in the `VERSION` file (semver format). To release a new version:

**Quick way (using Makefile):**
```bash
make tag VERSION=v0.2.0
```

This automatically:
- Updates `VERSION` file to `0.2.0`
- Commits the change
- Creates git tag `v0.2.0`
- Pushes to GitHub (triggers CI/CD)

**Manual way:**
```bash
# 1. Update VERSION file
echo "0.2.0" > VERSION

# 2. Commit
git add VERSION
git commit -m "chore: bump version to 0.2.0"
git push origin main

# 3. Create and push tag (triggers release)
git tag v0.2.0
git push origin v0.2.0
```

### Automatic Release Process

When you push a tag `v*.*.*`:

1. **Build Job** — Compiles for all 5 platforms (15-30 seconds)
2. **Release Job** — Creates GitHub Release with:
   - Binaries for darwin/linux/windows × amd64/arm64
   - `checksums.txt` (SHA256 hashes)
   - Installation instructions
3. **Available at** — https://github.com/hiimtrung/coder/releases

Users can then install the new version:
```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/install.sh)"
coder check-update  # detects new version
coder self-update   # auto-upgrade
```

---

## 🔄 Continuous Integration

GitHub Actions automatically builds and releases on tag push:

1. **Trigger**: Push a tag matching `v*.*.*` (e.g., `v0.1.0`)
2. **Build**: Compiles for 5 platforms (macOS arm64/amd64, Linux arm64/amd64, Windows amd64)
3. **Release**: Creates GitHub Release with binaries, checksums, and install instructions
4. **Details**: See [.github/workflows/release.yml](.github/workflows/release.yml)

---

## 📚 Included Skills

### Core (All Profiles)
- **architecture** — Clean Architecture, layers, DDD
- **general-patterns** — Error codes, exceptions, HTTP status, cross-language patterns
- **development** — Use cases, domain events, validation
- **database** — PostgreSQL, MongoDB, Redis, migrations, repositories
- **testing** — Unit, integration, E2E, Jest, Mockito
- **docs-analysis** — Documentation standards

### Backend
- **nestjs** — NestJS, TypeScript, type safety
- **java** — Spring Boot, Gradle, enterprise patterns
- **golang** — Concurrent Go services, interfaces, idioms
- **python** — Type hints, Pydantic, Clean Architecture
- **rust** — Ownership, Result types, Actix/Axum
- **c** — Memory safety, modular development
- **dart** — Flutter, sound null safety

### Frontend
- **frontend** — UI engineering, SSR/CSR, design systems
- **react-best-practices** — React 19, Next.js performance, bundle optimization
- **react-native-skills** — React Native, Expo, mobile performance
- **composition-patterns** — Component architecture, render props, compound components
- **web-design-guidelines** — Accessibility, Web Interface Guidelines
- **ui-ux-pro-max** — Design system generator, 96+ color palettes, 67+ UI styles

---

## 🎓 System Philosophy

This is not just a skill library—it's a **unified engineering knowledge system** built for AI agents (Copilot, Antigravity, etc.):

- **Consistency**: One source of truth for architecture, error codes, and patterns
- **Reusability**: Embed the entire system into any project with a single command
- **Scalability**: 20+ skills, 99+ UX guidelines, standardized workflows
- **Autonomy**: AI agents inherit senior-level decision-making without human intervention
- **Transparency**: All rules, skills, and workflows are version-controlled and auditable

---

## 📝 License

MIT

---

## 🤝 Contributing

Issues and pull requests welcome! For major changes, open a discussion first.

---

**Last Updated**: March 2026
**System**: coder (Unified AI Development Guidance CLI)
**Status**: Production Ready
**Built with**: Go 1.26.0
