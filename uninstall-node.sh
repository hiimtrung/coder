#!/bin/sh
# uninstall-node.sh — Uninstall coder-node service (Docker based)

set -e

INSTALL_DIR="$HOME/.coder-node"
CLEAR_DATA=false

# ── Parse arguments ───────────────────────────────────────────────────────────

while [ $# -gt 0 ]; do
  case "$1" in
    --clear-data)
      CLEAR_DATA=true; shift ;;
    --keep-data)
      CLEAR_DATA=false; shift ;;
    --help|-h)
      echo "Usage: uninstall-node.sh [--clear-data|--keep-data]"
      echo "  --clear-data    Remove the service and all volumes (database/models data)"
      echo "  --keep-data     Remove the service but keep data volumes (default)"
      exit 0 ;;
    *)
      echo "Unknown option: $1"; exit 1 ;;
  esac
done

if [ ! -d "$INSTALL_DIR" ]; then
    echo "⚠ coder-node installation directory not found at $INSTALL_DIR."
    echo "It might already be uninstalled."
    exit 0
fi

echo "====================================="
echo " Uninstalling coder-node..."
echo "====================================="

cd "$INSTALL_DIR"

# Stop and remove containers
if docker compose version > /dev/null 2>&1; then
    if [ "$CLEAR_DATA" = true ]; then
        echo "Removing service and wiping all data volumes..."
        docker compose down -v
    else
        echo "Stopping service and removing containers (keeping data volumes)..."
        docker compose down
    fi
else
    if [ "$CLEAR_DATA" = true ]; then
        echo "Removing service and wiping all data volumes..."
        docker-compose down -v
    else
        echo "Stopping service and removing containers (keeping data volumes)..."
        docker-compose down
    fi
fi

# Remove installation directory
echo "Removing installation files from $INSTALL_DIR..."
rm -rf "$INSTALL_DIR"

echo "====================================="
echo " coder-node uninstalled successfully!"
if [ "$CLEAR_DATA" = false ]; then
    echo " Note: Docker volumes (pg_data, ollama_data) were preserved."
fi
echo "====================================="
