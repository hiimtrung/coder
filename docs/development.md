# 🛠️ Development Guide

This document is for developers who want to contribute to **coder** or build it from source.

## 🧱 Project Structure

- `cmd/coder/`: CLI entry point and command handlers.
- `cmd/coder-node/`: Background service for gRPC/HTTP/Vector logic.
- `internal/`: Core business logic (packages: `skill`, `memory`, `installer`, `version`, etc.).
- `api/`: gRPC proto definitions and generated code.
- `installer/`: Platform-specific installation scripts (`install.sh`, `install.ps1`).

## 🧱 Building from Source

### Prerequisites
- Go 1.26+
- Docker (for dependencies)

### Build Commands
We use a `Makefile` to simplify common tasks.

```bash
# Build binary for current platform
make build

# Build all binaries (Darwin/Linux/Windows)
make build-all

# Clean dist folder
make clean
```

## 🧪 Testing

```bash
# Run unit tests
go test ./...

# Run smoke tests on the built binary
make test
```

## 🚀 Release Process

We use GitHub Actions for automated releases.

1. **Versioning**: The version is managed in the `VERSION` file.
2. **Tagging**: When you're ready to release:
   ```bash
   make tag VERSION=v0.2.0
   ```
3. **CI/CD**: GitHub Actions picks up the tag, builds binaries for all platforms, generates checksums, and creates a new Release.

### CI Configuration
The logic is defined in `.github/workflows/release.yml`. It handles:
- Go compilation for multiple `GOOS`/`GOARCH`.
- Docker build and push to GHCR for `coder-node`.
- Checksum generation.

## 📦 Embedded Files
The CLI embeds its own skill library using `go:embed`. If you modify `.agents/skills/` or `.agents/workflows/`, you must re-build the binary to include the new changes.
