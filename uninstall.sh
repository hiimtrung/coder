#!/bin/sh
# uninstall.sh — Uninstall coder CLI
#
# Usage:
#   ./uninstall.sh [--clear-data]
#   ./uninstall.sh [--keep-data]

set -e

BINARY="coder"
INSTALL_DIR="/usr/local/bin"
DATA_DIR="$HOME/.coder"
CLEAR_DATA=false

# ── Parse arguments ───────────────────────────────────────────────────────────

while [ $# -gt 0 ]; do
  case "$1" in
    --clear-data)
      CLEAR_DATA=true; shift ;;
    --keep-data)
      CLEAR_DATA=false; shift ;;
    --help|-h)
      echo "Usage: uninstall.sh [--clear-data|--keep-data]"
      echo "  --clear-data    Remove the binary and all local data ($DATA_DIR)"
      echo "  --keep-data     Remove the binary but keep local data (default)"
      exit 0 ;;
    *)
      echo "Unknown option: $1"; exit 1 ;;
  esac
done

# ── Detect installation ───────────────────────────────────────────────────────

DEST=$(command -v $BINARY || echo "$INSTALL_DIR/$BINARY")

if [ ! -f "$DEST" ]; then
  echo "⚠ $BINARY not found at $DEST. It might already be uninstalled or installed elsewhere."
else
  echo "Uninstalling $BINARY from $DEST..."
  if rm "$DEST" 2>/dev/null; then
    echo "✓ Binary removed."
  else
    echo "Requires elevated permissions to remove $DEST..."
    sudo rm "$DEST"
    echo "✓ Binary removed."
  fi
fi

# ── Ollama Cleanup ────────────────────────────────────────────────────────────

echo "Cleaning up Ollama services..."

# Detect OS
OS="$(uname -s)"

if [ "$OS" = "Darwin" ]; then
  PLIST_PATH="$HOME/Library/LaunchAgents/com.ollama.ollama.plist"
  if [ -f "$PLIST_PATH" ]; then
    echo "Removing Ollama LaunchAgent..."
    launchctl unload "$PLIST_PATH" 2>/dev/null || true
    rm "$PLIST_PATH"
    echo "✓ LaunchAgent removed."
  fi
elif [ "$OS" = "Linux" ]; then
  SERVICE_PATH="$HOME/.config/systemd/user/ollama.service"
  if [ -f "$SERVICE_PATH" ]; then
    echo "Removing Ollama systemd service..."
    systemctl --user stop ollama 2>/dev/null || true
    systemctl --user disable ollama 2>/dev/null || true
    rm "$SERVICE_PATH"
    systemctl --user daemon-reload
    echo "✓ systemd service removed."
  fi
fi

# Kill any running ollama process
if pgrep -x "ollama" > /dev/null; then
  echo "Terminating running Ollama processes..."
  pkill -x "ollama" || true
  echo "✓ Ollama processes terminated."
fi

# ── Clear Local Data ─────────────────────────────────────────────────────────

if [ "$CLEAR_DATA" = true ]; then
  if [ -d "$DATA_DIR" ]; then
    echo "Clearing local data at $DATA_DIR..."
    rm -rf "$DATA_DIR"
    echo "✓ Local data cleared."
  else
    echo "No local data found at $DATA_DIR."
  fi
else
  echo "Keeping local data at $DATA_DIR."
fi

echo ""
echo "✓ $BINARY has been uninstalled."
