#!/bin/sh
# install.sh — Install coder CLI
#
# Usage:
#   /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/install.sh)"
#   /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/install.sh)" -- --version v0.3.5
#   /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/install.sh)" -- --skip-login

set -e

REPO="hiimtrung/coder"
BINARY="coder"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION=""
SKIP_LOGIN=false

# ── Parse arguments ───────────────────────────────────────────────────────────

while [ $# -gt 0 ]; do
  case "$1" in
    --version)    VERSION="$2"; shift 2 ;;
    --skip-login) SKIP_LOGIN=true; shift ;;
    *)            shift ;;
  esac
done

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
  if [ -n "$GITHUB_TOKEN" ]; then
    VERSION="$(curl -fsSL -H "Authorization: Bearer ${GITHUB_TOKEN}" \
      "https://api.github.com/repos/${REPO}/releases/latest" \
      | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"
  else
    VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
      | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"
  fi
  if [ -z "$VERSION" ]; then
    echo "Error: failed to fetch latest version from GitHub."
    echo ""
    echo "This is often caused by GitHub API rate limiting (60 req/hour for unauthenticated IPs)."
    echo "To fix, set a GitHub token and retry:"
    echo "  export GITHUB_TOKEN=<your_token>"
    echo "  /bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/${REPO}/main/install.sh)\""
    echo ""
    echo "Or install a specific version directly:"
    echo "  /bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/${REPO}/main/install.sh)\" -- --version v0.3.5"
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

mkdir -p "$INSTALL_DIR" 2>/dev/null || true

if mv "$TMP_FILE" "$DEST" 2>/dev/null; then
  :
else
  echo "Installing to $INSTALL_DIR requires elevated permissions..."
  sudo mv "$TMP_FILE" "$DEST"
fi

echo ""
echo "✓ Installed: ${DEST}"
echo ""

# ── Configure connection to coder-node ───────────────────────────────────────

CONFIG_DIR="$HOME/.coder"
CONFIG_FILE="$CONFIG_DIR/config.json"

mkdir -p "$CONFIG_DIR"

if [ -f "$CONFIG_FILE" ] && [ "$SKIP_LOGIN" = "false" ]; then
  echo "Existing configuration found at $CONFIG_FILE."
  printf "Reconfigure connection? [y/N]: "
  read -r RECONFIGURE < /dev/tty || RECONFIGURE="n"
  case "$RECONFIGURE" in
    y|Y) rm -f "$CONFIG_FILE" ;;
    *)   SKIP_LOGIN=true ;;
  esac
fi

if [ "$SKIP_LOGIN" = "false" ]; then
  echo "┌─────────────────────────────────────────────────────┐"
  echo "│  Connect to coder-node                              │"
  echo "│                                                     │"
  echo "│  You will be asked for:                             │"
  echo "│   • Protocol  — gRPC (fast) or HTTP                 │"
  echo "│   • Server URL                                      │"
  echo "│   • Auth token (only if your node runs --secure)    │"
  echo "│                                                     │"
  echo "│  No coder-node yet? Skip with Ctrl-C, then run:     │"
  echo "│    curl -fsSL .../install-node.sh | sh              │"
  echo "│  Re-run setup anytime with: coder login             │"
  echo "└─────────────────────────────────────────────────────┘"
  echo ""
  "$DEST" login
fi

# ── Print version + quick-start ───────────────────────────────────────────────

echo ""
if command -v "$BINARY" >/dev/null 2>&1; then
  "$BINARY" version
else
  echo "Note: '${INSTALL_DIR}' may not be in your PATH."
  echo "Add this to your shell profile:"
  echo "  export PATH=\"\$PATH:${INSTALL_DIR}\""
fi

echo ""
echo "Get started:"
echo "  ${BINARY} install fullstack            # scaffold agent engine into a project"
echo "  ${BINARY} skill ingest --source local  # load built-in skills into vector DB"
echo "  ${BINARY} skill search \"topic\"         # semantic skill search"
echo "  ${BINARY} memory store \"title\" \"data\"  # save a knowledge snippet"
echo "  ${BINARY} list                         # see all options"
