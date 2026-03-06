#!/bin/sh
# install.sh — Install coder CLI
#
# Usage:
#   /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/install.sh)"
#   /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/install.sh)" -- --version v0.1.0

set -e

REPO="hiimtrung/coder"
BINARY="coder"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION=""

# ── Detect platform ───────────────────────────────────────────────────────────

OS="$(uname -s)"
ARCH="$(uname -m)"



# ── Detect platform ───────────────────────────────────────────────────────────

OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
  Linux)  GOOS="linux" ;;
  Darwin) GOOS="darwin" ;;
  *)
    echo "Unsupported OS: $OS"
    echo "Please download manually from https://github.com/$REPO/releases"
    exit 1
    ;;
esac

case "$ARCH" in
  x86_64|amd64) GOARCH="amd64" ;;
  arm64|aarch64) GOARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    echo "Please download manually from https://github.com/$REPO/releases"
    exit 1
    ;;
esac

ASSET_NAME="${BINARY}-${GOOS}-${GOARCH}"

# ── Resolve version ───────────────────────────────────────────────────────────

if [ -z "$VERSION" ]; then
  echo "Fetching latest release..."
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"
  if [ -z "$VERSION" ]; then
    echo "Error: failed to fetch latest version from GitHub."
    exit 1
  fi
fi

echo "Installing ${BINARY} ${VERSION} (${GOOS}/${GOARCH})..."

# ── Download ──────────────────────────────────────────────────────────────────

DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET_NAME}"
TMP_FILE="$(mktemp)"

echo "Downloading: ${DOWNLOAD_URL}"
if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_FILE"; then
  echo "Error: download failed."
  echo "Check that release ${VERSION} exists: https://github.com/${REPO}/releases"
  rm -f "$TMP_FILE"
  exit 1
fi

chmod +x "$TMP_FILE"

# ── Install ───────────────────────────────────────────────────────────────────

DEST="${INSTALL_DIR}/${BINARY}"

# Ensure INSTALL_DIR exists
mkdir -p "$INSTALL_DIR" 2>/dev/null || true

# Try to move to destination; use sudo if needed
if mv "$TMP_FILE" "$DEST" 2>/dev/null; then
  :
else
  # No permission; try with sudo
  echo "Installing to $INSTALL_DIR requires elevated permissions..."
  sudo mv "$TMP_FILE" "$DEST"
fi

echo ""
echo "✓ Installed: ${DEST}"
echo ""

# ── Initialize Config ─────────────────────────────────────────────────────────

CONFIG_DIR="$HOME/.coder"
CONFIG_FILE="$CONFIG_DIR/config.json"

if [ ! -d "$CONFIG_DIR" ]; then
  mkdir -p "$CONFIG_DIR"
fi

if [ ! -f "$CONFIG_FILE" ]; then
  echo "Initializing configuration..."
  
  # Prompt for Ollama URL
  while true; do
    printf "Enter Ollama Base URL [http://127.0.0.1:11434]: "
    read -r OLLAMA_URL < /dev/tty || break
    OLLAMA_URL=${OLLAMA_URL:-http://127.0.0.1:11434}
    
    echo "Verifying Ollama connection at $OLLAMA_URL..."
    if curl -s --connect-timeout 5 "$OLLAMA_URL" >/dev/null 2>&1; then
      echo "✓ Ollama connection successful."
      break
    else
      echo "⚠ Could not connect to Ollama at $OLLAMA_URL."
      printf "Do you want to use this URL anyway? [y/N]: "
      read -r choice < /dev/tty || break
      case "$choice" in 
        y|Y ) break;;
        * ) ;;
      esac
    fi
  done
  
  # Prompt for Postgres DSN
  while true; do
    printf "Enter PostgreSQL DSN (e.g., postgres://user:pass@host:5432/dbname?sslmode=disable): "
    read -r POSTGRES_DSN < /dev/tty || break
    if [ -z "$POSTGRES_DSN" ]; then
      echo "PostgreSQL DSN cannot be empty."
      continue
    fi
    
    echo "Verifying PostgreSQL connection..."
    # Create actual config file so the coder binary can use it
    cat <<EOF > "$CONFIG_FILE"
{
  "memory": {
    "provider": "ollama",
    "database_type": "postgres",
    "base_url": "$OLLAMA_URL",
    "model": "mxbai-embed-base",
    "postgres_dsn": "$POSTGRES_DSN"
  }
}
EOF

    # Test connection using the installed binary
    if "$DEST" memory list --limit 1 >/dev/null 2> "$CONFIG_DIR/dbcheck.err"; then
      echo "✓ PostgreSQL connection successful."
      rm -f "$CONFIG_DIR/dbcheck.err"
      break
    else
      echo "⚠ Failed to connect to PostgreSQL. Error details:"
      cat "$CONFIG_DIR/dbcheck.err"
      rm -f "$CONFIG_DIR/dbcheck.err"
      printf "Do you want to re-enter the DSN? [Y/n]: "
      read -r choice < /dev/tty || break
      case "$choice" in 
        n|N ) break;;
        * ) ;;
      esac
    fi
  done
fi

# ── Verify ────────────────────────────────────────────────────────────────────

if command -v "$BINARY" >/dev/null 2>&1; then
  "$BINARY" version
else
  echo "Note: '${INSTALL_DIR}' may not be in your PATH."
  echo "Add this to your shell profile:"
  echo "  export PATH=\"\$PATH:${INSTALL_DIR}\""
fi

echo ""
echo "Get started:"
echo "  ${BINARY} install be        # backend project"
echo "  ${BINARY} install fe        # frontend project"
echo "  ${BINARY} install fullstack # full-stack project"
echo "  ${BINARY} list              # see all options"
