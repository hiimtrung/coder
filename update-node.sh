#!/bin/sh
# update-node.sh — Update coder-node to the latest version
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/update-node.sh | sh

set -e

INSTALL_DIR="$HOME/.coder-node"
REPO="hiimtrung/coder"
COMPOSE_URL="https://raw.githubusercontent.com/${REPO}/main/infrastructure/docker-compose.yml"

echo "====================================="
echo " Updating coder-node..."
echo "====================================="

# ── Prereq check ─────────────────────────────────────────────────────────────

if ! command -v docker > /dev/null 2>&1; then
    echo "Error: Docker is not installed."
    exit 1
fi

if [ ! -d "$INSTALL_DIR" ]; then
    echo "Error: coder-node is not installed at $INSTALL_DIR."
    echo "Run install-node.sh first:"
    echo "  curl -fsSL https://raw.githubusercontent.com/${REPO}/main/install-node.sh | sh"
    exit 1
fi

# ── Helper: pick docker compose command ──────────────────────────────────────

compose() {
    if docker compose version > /dev/null 2>&1; then
        docker compose "$@"
    else
        docker-compose "$@"
    fi
}

cd "$INSTALL_DIR"

# ── Step 1: Update docker-compose.yml ────────────────────────────────────────

echo "Downloading latest docker-compose.yml..."
curl -fsSL "$COMPOSE_URL" -o docker-compose.yml
echo "  ✓ docker-compose.yml updated"

# ── Step 2: Pull latest images ───────────────────────────────────────────────

echo "Pulling latest images..."
compose pull coder-node
echo "  ✓ coder-node image updated"

# ── Step 3: Restart coder-node with new image ────────────────────────────────
# postgres and ollama are NOT restarted to avoid downtime / data loss.

echo "Restarting coder-node..."
compose up -d --no-deps coder-node
echo "  ✓ coder-node restarted"

# ── Done ─────────────────────────────────────────────────────────────────────

echo ""
echo "====================================="
echo " coder-node updated successfully!"
echo " Running on:"
echo "   - gRPC : port 50051"
echo "   - HTTP : port 8080"
echo "====================================="
