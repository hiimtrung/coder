#!/bin/sh

set -e

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

cd "$INSTALL_DIR"

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
echo ""
echo " In your local machine, run coder and follow the interactive setup."
echo " Or configure it manually in ~/.coder/config.json."
echo "====================================="
