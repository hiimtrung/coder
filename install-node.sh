#!/bin/sh

set -e

# ─── Parse flags ─────────────────────────────────────────────────────────────
SECURE_MODE=false
for arg in "$@"; do
    case "$arg" in
        --secure) SECURE_MODE=true ;;
        --help|-h)
            echo "Usage: install-node.sh [--secure]"
            echo ""
            echo "Options:"
            echo "  --secure   Start coder-node with authentication enabled."
            echo "             A one-time bootstrap token will be printed to the logs."
            echo "             Share it with each developer so they can run: coder login"
            exit 0
            ;;
        *)
            echo "Unknown option: $arg"
            echo "Run with --help for usage."
            exit 1
            ;;
    esac
done

echo "====================================="
echo " Installing coder-node prerequisites"
echo "====================================="

# Check if Docker is installed
if ! command -v docker > /dev/null 2>&1; then
    echo "Docker is not installed. Please install Docker first."
    echo "Visit: https://docs.docker.com/get-docker/"
    exit 1
fi

if ! command -v docker compose > /dev/null 2>&1 && ! command -v docker-compose > /dev/null 2>&1; then
    echo "Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

INSTALL_DIR="$HOME/.coder-node"
echo "Setting up coder-node at $INSTALL_DIR..."

mkdir -p "$INSTALL_DIR"

# Download the docker-compose file
if [ -f "infrastructure/docker-compose.yml" ]; then
    cp infrastructure/docker-compose.yml "$INSTALL_DIR/docker-compose.yml"
else
    # Fallback url when run remotely
    echo "Downloading docker-compose.yml..."
    curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/infrastructure/docker-compose.yml -o "$INSTALL_DIR/docker-compose.yml"
fi

# Write .env file so docker compose picks up SECURE_MODE
cat > "$INSTALL_DIR/.env" <<EOF
SECURE_MODE=$SECURE_MODE
EOF

cd "$INSTALL_DIR"

if [ "$SECURE_MODE" = "true" ]; then
    echo ""
    echo "  Secure mode ENABLED."
    echo "  A bootstrap token will appear in the logs on first startup."
    echo "  Run the following to retrieve it:"
    echo ""
    echo "    docker logs coder_node 2>&1 | grep 'BOOTSTRAP TOKEN'"
    echo ""
fi

echo "Bringing up coder-node, postgres, and ollama..."
# Use docker compose if available, otherwise docker-compose
if docker compose version > /dev/null 2>&1; then
    docker compose up -d
else
    docker-compose up -d
fi

echo "====================================="
echo " coder-node installed successfully!"
echo " It is now running on:"
echo "   - gRPC: port 50051"
echo "   - HTTP: port 8080"
if [ "$SECURE_MODE" = "true" ]; then
    echo ""
    echo " Authentication is ENABLED."
    echo " Retrieve your bootstrap token:"
    echo "   docker logs coder_node 2>&1 | grep 'BOOTSTRAP TOKEN'"
    echo ""
    echo " Then on each developer machine, run:"
    echo "   coder login"
    echo " and enter the server URL + bootstrap token when prompted."
else
    echo ""
    echo " Authentication is OFF (open mode)."
    echo " To enable auth, re-run: install-node.sh --secure"
fi
echo "====================================="
