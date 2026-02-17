#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_DIR="/opt/gog-auth-server"

echo "Building auth-server..."
cd "$SCRIPT_DIR"
CGO_ENABLED=0 go build -ldflags="-w -s" -o auth-server .

echo "Deploying to $DEPLOY_DIR..."
sudo mkdir -p "$DEPLOY_DIR"
sudo cp auth-server ecosystem.config.js "$DEPLOY_DIR/"
sudo chmod 755 "$DEPLOY_DIR/auth-server"

echo "Starting with PM2..."
cd "$DEPLOY_DIR"
pm2 stop gog-auth-server 2>/dev/null || true
pm2 delete gog-auth-server 2>/dev/null || true
pm2 start ecosystem.config.js
pm2 save

echo "Health check..."
sleep 2
if curl -sf http://localhost:8089/health >/dev/null; then
    echo "Auth server is healthy"
else
    echo "Health check failed" >&2
    exit 1
fi
