# Changelog

All notable changes to **coder** are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

---

## [v0.5.4] — 2026-03-31

### Added
- **`make release-prepare`**: Added a branch-side release preparation command that updates `VERSION` and scaffolds a matching `CHANGELOG.md` section in one step.
- **`make release-main`**: Added a main-only release command that validates `origin/main` and delegates to tag creation after the release branch has been merged.

### Changed
- **Protected-main release flow**: Release operations are now split into two explicit phases: prepare metadata on your branch, then cut the tag from `origin/main` after merge.
- **Release instructions**: Updated the development guide to document the new two-command workflow for protected `main` branches.

### Fixed
- **Release operator confusion**: Removed the old ambiguous flow where release preparation and release cutting were mixed together, making it clearer when work happens on a branch versus on merged `main`.


## [v0.5.3] — 2026-03-31

### Added
- **gRPC auth service**: Added `AuthService` over gRPC for client registration, token rotation, identity lookup, and activity logging, so secure-mode CLI flows can stay on a single gRPC endpoint and port.
- **Release note scaffold**: Added `make release-note VERSION=vX.Y.Z` to scaffold a new changelog section in the correct format.

### Changed
- **Single-protocol login flow**: `coder login` now uses the selected protocol consistently. When `gRPC` is selected, bootstrap registration no longer jumps to HTTP or asks for a separate HTTP auth URL.
- **Token commands respect configured protocol**: `coder token show` and `coder token rotate` now use gRPC when the CLI is configured for gRPC, instead of always calling HTTP auth endpoints.
- **Release tagging flow**: Reworked `Makefile` release targets so releases are cut from a merged ref via annotated tag push only. Removed the old behavior that auto-staged files, auto-committed, and pushed the current branch.

### Fixed
- **Secure-mode login 404s**: Fixed the CLI auth flow that incorrectly derived an HTTP registration URL from a gRPC endpoint, which caused bootstrap registration to fail in deployments that did not expose the assumed HTTP port.
- **Skill ingest with empty embeddings**: Skill ingestion now falls back to FTS-only storage when an embedding provider returns an empty vector, instead of failing with `pq: vector must have at least 1 dimension`.


## [v0.5.2] — 2026-03-30

### Added
- 

### Changed
- 

### Fixed
- 


## [v0.3.4] — 2026-03-18

### Added

#### Client Authentication & Secure Mode
- **`coder-node --secure`**: New `SECURE_MODE=true` environment variable enables mandatory client authentication on the HTTP layer.
- **Bootstrap token**: On first startup in secure mode, the server generates a cryptographically random 32-byte token, prints it once to stdout, and stores only its SHA-256 hash — the plaintext is never persisted.
- **`POST /v1/auth/register-client`**: Clients supply the bootstrap token + git identity (`name`, `email`) and receive a permanent access token. Public endpoint, no prior auth required.
- **`GET /v1/auth/clients`**: Lists all registered clients (requires valid token).
- **`POST /v1/auth/log-activity`**: Records which command, repo, and branch a client used (fire-and-forget from CLI side).
- **HTTP auth middleware**: All routes except `/health` and `/v1/auth/register-client` now validate `Authorization: Bearer <token>` when in secure mode. Open mode is a transparent no-op.
- **`coder login` auth flow**: Interactive wizard now asks *"Does this server require authentication?"* — if yes, prompts for bootstrap token, auto-detects git identity, calls register endpoint, and saves the returned access token to `~/.coder/config.json`.
- **Automatic Bearer header**: Every `coder memory` and `coder skill` command now attaches `Authorization: Bearer` if an access token is configured.
- **Activity telemetry**: `coder memory store`, `coder memory search`, `coder skill search`, and `coder skill ingest` fire a background `POST /v1/auth/log-activity` after each invocation. Errors are silently discarded so commands never fail due to logging.

#### `install-node.sh` improvements
- New `--secure` flag: `./install-node.sh --secure` creates `~/.coder-node/.env` with `SECURE_MODE=true` before starting the stack.
- Post-install output now shows how to retrieve the bootstrap token and how to register developer machines.
- `--help` flag added.

#### `docker-compose.yml`
- Added `SECURE_MODE: ${SECURE_MODE:-false}` to `coder-node` service — resolves from `.env`, defaults to `false` (open mode).

### Changed
- `coder login` signature changed to accept (and ignore) trailing args for forward compatibility.
- HTTP and gRPC clients now accept an optional `accessToken` argument in their constructors (`NewClient`, `NewSkillClient`).
- `Config` struct in CLI extended with `Auth.AccessToken` field.

### Fixed
- Duplicate `/health` route panic at startup: removed the `/health` registration from `MemoryServer.RegisterHandlers` — the canonical health endpoint is now registered exclusively in `main.go` (it carries `secure_mode` status in the JSON response).
- Removed duplicate `gitCmdOutput` helper from `cmd_login.go`; callers now use the shared `gitOutput` from `cmd_activity.go`.

### Architecture
- New packages under clean architecture layers:
  - `internal/domain/auth` — `Client`, `Activity` entities + `AuthRepository` / `AuthManager` interfaces.
  - `internal/infra/postgres/auth.go` — PostgreSQL implementation; creates `coder_server_config`, `coder_clients`, `coder_client_activity` tables on startup.
  - `internal/usecase/auth/manager.go` — business logic for token lifecycle; open-mode implementation is a no-op struct.
  - `internal/transport/http/middleware/auth.go` — `Auth(mgr)` middleware factory.
  - `internal/transport/http/server/auth.go` — HTTP handlers for register, list, log-activity.
  - `cmd/coder/cmd_activity.go` — `logActivity`, `gitOutput`, `sanitiseRepoURL`.

---

## [v0.3.3] — 2026-03-17

### Added
- **Hybrid search (RRF)**: Memory and skill search now fuse pgvector cosine similarity with PostgreSQL full-text search (`tsvector`) using Reciprocal Rank Fusion (`rrf_score = 1/(60+semantic_rank) + 1/(60+keyword_rank)`). Significantly improves results for short or exact-match queries.
- `tsvector` columns added to `coder_memories` and `coder_skills`; automatically maintained via `GENERATED ALWAYS AS` or trigger.

---

## [v0.3.2] — 2026-03-16

### Changed
- **Clean Architecture refactor**: All internal code reorganized into explicit layers:
  - `internal/domain/{memory,skill}` — pure domain entities, types, interfaces. Zero framework dependencies.
  - `internal/usecase/{memory,skill}` — application services / orchestrators.
  - `internal/infra/postgres` — repository implementations (pgvector).
  - `internal/infra/embedding` — Ollama embedding provider.
  - `internal/transport/grpc/server` — gRPC service handlers.
  - `internal/transport/http/server` — HTTP REST handlers.
  - `internal/transport/http/client` — HTTP client for CLI.
  - `internal/transport/grpc/client` — gRPC client for CLI.
- Deleted legacy packages: `internal/memory/`, `internal/skill/`, `internal/grpcserver/`, `internal/httpserver/`, `internal/grpcclient/`, `internal/httpclient/`.
- All `interface{}` → `any` (Go 1.18+ alias).
- `strings.Split` + range → `strings.SplitSeq` (Go 1.26).
- `grpc.Dial` → `grpc.NewClient` (non-deprecated API).

---

## [v0.3.1] — 2026-03-15

### Added
- `coder update global` command installs agent files to `~/.claude/agents/`, `~/.config/github-copilot/`, and other global user-level directories.
- Claude CLI agent definitions (`.claude/agents/`) included in global installation targets.

---

## [v0.3.0] — 2026-03-10

### Added
- Initial public release of the Skill RAG system backed by pgvector.
- `coder skill ingest`, `coder skill search`, `coder skill list`, `coder skill info`, `coder skill delete`.
- `coder memory store`, `coder memory search`, `coder memory list`, `coder memory delete`, `coder memory compact`.
- `coder-node` Docker-based infrastructure (Postgres + pgvector + Ollama).
- gRPC and HTTP transport layers.
- 20+ embedded skills: `nestjs`, `golang`, `java`, `rust`, `python`, `architecture`, `development`, `database`, `general-patterns`, `ui-ux-pro-max`, and more.
- `coder install <profile>` scaffolds `.agents/` into any project.
- `coder self-update` for binary auto-update from GitHub Releases.
