# Development Guide

---

## Project structure

```
coder/
├── cmd/
│   ├── coder/              CLI entry point + command handlers
│   │   ├── main.go
│   │   ├── cmd_memory.go
│   │   ├── cmd_skill.go
│   │   ├── cmd_login.go    interactive setup wizard + auth registration
│   │   ├── cmd_activity.go fire-and-forget activity logging
│   │   └── config.go       loadConfig, getMemoryManager, getSkillClient
│   └── coder-node/
│       └── main.go         wire gRPC + HTTP + auth + use cases
│
├── internal/
│   ├── domain/             pure domain — zero framework deps
│   │   ├── auth/           Client, Activity, AuthManager, AuthRepository, context helpers
│   │   ├── memory/         Knowledge, MemoryManager, MemoryRepository
│   │   └── skill/          Skill, SkillChunk, SkillUseCase, SkillClient
│   │
│   ├── usecase/            orchestration layer
│   │   ├── auth/           bootstrap token, validate, log activity
│   │   ├── memory/         store, search (RRF), compact, revector
│   │   └── skill/          ingest (diff+embed), search (RRF), facade
│   │
│   ├── infra/              outbound adapters
│   │   ├── postgres/       AuthRepo, MemoryRepo, SkillStore (pgvector + tsvector)
│   │   ├── embedding/      Ollama + OpenAI providers
│   │   └── github/         remote skill fetcher
│   │
│   └── transport/          inbound adapters
│       ├── grpc/
│       │   ├── server/     MemoryServer, SkillServer
│       │   ├── client/     memory + skill gRPC clients (PerRPCCredentials)
│       │   ├── interceptor/ UnaryAuth, StreamAuth
│       │   └── credential/ BearerToken (PerRPCCredentials)
│       └── http/
│           ├── server/     memory, skill, auth handlers
│           ├── client/     memory + skill HTTP clients
│           └── middleware/ Auth middleware
│
├── api/
│   ├── proto/              .proto source files
│   └── grpc/               generated Go gRPC code
│
├── installer/              embedded skill/workflow assets + GitHub fetcher
├── infrastructure/         docker-compose.yml for coder-node stack
├── .github/workflows/      release.yml (build + release + Docker)
├── install.sh              CLI installer (macOS/Linux)
├── install-node.sh         coder-node installer (--secure flag)
├── uninstall.sh / uninstall-node.sh / update-node.sh
├── Dockerfile.node         multi-stage build for coder-node
├── Makefile
├── VERSION                 current version string (e.g. v0.3.5)
└── CHANGELOG.md
```

---

## Prerequisites

- **Go 1.26+**
- **Docker** (for the local coder-node stack)
- `protoc` + `protoc-gen-go` + `protoc-gen-go-grpc` (only if modifying `.proto` files)

---

## Building

```bash
# Current platform
make build

# All platforms (darwin/linux/windows × amd64/arm64)
make build-all

# Install to /usr/local/bin
make install

# Install to ~/bin (no sudo)
make install-user

# Clean dist/
make clean
```

---

## Running locally

```bash
# Start the coder-node stack
docker compose -f infrastructure/docker-compose.yml up -d

# Build and run the CLI against it
make build
./dist/coder login

# With secure mode
SECURE_MODE=true docker compose -f infrastructure/docker-compose.yml up -d
# or via .env:
echo "SECURE_MODE=true" > infrastructure/.env
docker compose -f infrastructure/docker-compose.yml up -d
```

---

## Testing

```bash
# Unit tests
go test ./...

# Smoke test on the built binary
make test

# Vet + build check (same as CI)
go vet ./...
go build ./...
```

---

## Regenerating proto files

```bash
cd api/proto
protoc --go_out=../../api/grpc/memorypb --go-grpc_out=../../api/grpc/memorypb \
       --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative \
       memory/memory.proto

protoc --go_out=../../api/grpc/skillpb --go-grpc_out=../../api/grpc/skillpb \
       --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative \
       skill/skill.proto
```

---

## Release process

### 1. Update `VERSION`

```bash
echo "v0.3.6" > VERSION
```

### 2. Update `CHANGELOG.md`

Add a `## [v0.3.6] — YYYY-MM-DD` section. The release workflow extracts this block automatically and uses it as the GitHub Release body.

### 3. Commit and tag

```bash
git add VERSION CHANGELOG.md
git commit -m "chore: bump version to v0.3.6"
git tag v0.3.6
git push origin main --tags
```

### 4. CI takes over

`.github/workflows/release.yml` runs three parallel jobs on a tag push:

| Job | What it does |
|-----|-------------|
| **build** | Cross-compiles for darwin/linux/windows × amd64/arm64; uploads artifacts |
| **release** | Downloads all artifacts; extracts changelog; creates GitHub Release with binaries + checksums |
| **docker** | Builds multi-arch `coder-node` image (`linux/amd64` + `linux/arm64`); pushes to `ghcr.io` |

---

## Adding a new command

1. Create `cmd/coder/cmd_<name>.go`
2. Register in `main.go` command dispatch
3. If it touches the node: add the use case in `internal/usecase/`, the interface in `internal/domain/`, the implementation in `internal/infra/` or transport layer
4. If it calls the HTTP client: add `addAuth(req)` before `c.client.Do(req)`
5. If it calls the gRPC client: auth is automatic via `PerRPCCredentials`

---

## Embedded assets

The CLI embeds skill libraries and workflow templates via `go:embed` (see `assets.go`). After modifying `.agents/skills/` or `.agents/workflows/`, rebuild the binary:

```bash
make build
```
