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

# Download the docker-compose file (in a real scenario, this would be downloaded from github)
# For now, if we are in the repo, we just copy it. If not, curl it.
if [ -f "infrastructure/docker-compose.yml" ]; then
    cp infrastructure/docker-compose.yml "$INSTALL_DIR/docker-compose.yml"
else
    # Fallback url when run remotely
    echo "Downloading docker-compose.yml..."
    curl -fsSL https://raw.githubusercontent.com/trungtran/coder/main/infrastructure/docker-compose.yml -o "$INSTALL_DIR/docker-compose.yml"
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
echo " It is now running on port 50051."
echo " In your local machine, run coder with:"
echo " export CODER_NODE_URL=http://<server-ip>:50051"
echo " Or configure it in ~/.coder/config.json with provider 'grpc'."
echo "====================================="
