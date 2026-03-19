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

# Pick the right compose command once
if docker compose version > /dev/null 2>&1; then
    COMPOSE="docker compose"
else
    COMPOSE="docker-compose"
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
    curl -fsSL https://raw.githubusercontent.com/hiimtrung/coder/main/infrastructure/docker-compose.yml \
        -o "$INSTALL_DIR/docker-compose.yml"
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

# ─── Bring up services ───────────────────────────────────────────────────────
echo "Bringing up coder-node, postgres, and ollama..."
echo "(First run may take 2–3 minutes while pulling images and initialising the database)"
echo ""

# Don't exit immediately if compose fails — capture exit code so we can print
# a helpful diagnostic instead of a bare "set -e" crash.
set +e
$COMPOSE up -d 2>&1
COMPOSE_EXIT=$?
set -e

if [ $COMPOSE_EXIT -ne 0 ]; then
    echo ""
    echo "ERROR: docker compose exited with code $COMPOSE_EXIT."
    echo ""
    echo "Possible causes:"
    echo "  • postgres healthcheck timed out — the DB may still be initialising."
    echo "    Wait 30 s and try: $COMPOSE -f $INSTALL_DIR/docker-compose.yml up -d"
    echo "  • Port conflict (50051 or 8080 already in use)."
    echo "    Check: ss -tlnp | grep -E '50051|8080'"
    echo "  • Insufficient disk space / Docker daemon not running."
    echo ""
    echo "Container status:"
    $COMPOSE ps 2>/dev/null || true
    echo ""
    echo "pgvector_db logs (last 30 lines):"
    docker logs pgvector_db --tail 30 2>/dev/null || true
    echo ""
    exit $COMPOSE_EXIT
fi

# ─── Wait for postgres to actually be healthy ────────────────────────────────
echo "Waiting for postgres to be healthy..."
MAX_WAIT=120   # seconds
WAITED=0
INTERVAL=5

while true; do
    STATUS=$(docker inspect --format='{{.State.Health.Status}}' pgvector_db 2>/dev/null || echo "missing")
    if [ "$STATUS" = "healthy" ]; then
        echo "  postgres is healthy."
        break
    fi
    if [ "$STATUS" = "unhealthy" ]; then
        echo ""
        echo "ERROR: pgvector_db is unhealthy after ${WAITED}s."
        echo ""
        echo "pgvector_db logs (last 40 lines):"
        docker logs pgvector_db --tail 40 2>/dev/null || true
        echo ""
        echo "Troubleshooting tips:"
        echo "  1. Retry (sometimes a cold disk just needs more time):"
        echo "       $COMPOSE -f $INSTALL_DIR/docker-compose.yml up -d"
        echo "  2. Inspect container:"
        echo "       docker inspect pgvector_db | grep -A10 Health"
        echo "  3. Run postgres manually to see startup errors:"
        echo "       docker run --rm pgvector/pgvector:pg16 postgres --version"
        exit 1
    fi
    if [ $WAITED -ge $MAX_WAIT ]; then
        echo ""
        echo "ERROR: postgres did not become healthy within ${MAX_WAIT}s (status: $STATUS)."
        docker logs pgvector_db --tail 40 2>/dev/null || true
        exit 1
    fi
    printf "  still waiting (%ds, status: %s)...\r" "$WAITED" "$STATUS"
    sleep $INTERVAL
    WAITED=$((WAITED + INTERVAL))
done

# ─── Summary ─────────────────────────────────────────────────────────────────
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
