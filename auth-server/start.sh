#!/bin/bash
# gog-cli OAuth Callback Server
# Runs on http://10.114.0.111:8089
#
# Required: Set these env vars or edit this script
# - GOG_CLIENT_ID
# - GOG_CLIENT_SECRET

PORT=8089
REDIRECT_URL="http://10.114.0.111:${PORT}/callback"

# Check for credentials
if [ -z "$GOG_CLIENT_ID" ] || [ -z "$GOG_CLIENT_SECRET" ]; then
    echo "⚠️  Missing OAuth credentials!"
    echo ""
    echo "Set these environment variables:"
    echo "  export GOG_CLIENT_ID='your-client-id'"
    echo "  export GOG_CLIENT_SECRET='your-client-secret'"
    echo ""
    echo "Get credentials from: https://console.cloud.google.com/apis/credentials"
    echo "Create OAuth 2.0 Client ID → Web Application"
    echo "Add redirect URI: ${REDIRECT_URL}"
    exit 1
fi

echo "🚀 Starting gog-cli OAuth Callback Server"
echo "   Port: ${PORT}"
echo "   Redirect URL: ${REDIRECT_URL}"
echo "   Health check: http://10.114.0.111:${PORT}/health"
echo ""

cd "$(dirname "$0")"
./auth-server \
    --port="${PORT}" \
    --redirect-url="${REDIRECT_URL}"
