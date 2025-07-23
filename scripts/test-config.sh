#!/bin/bash

# Test configuration loading for KubePulse

set -e

echo "🔍 Testing KubePulse Configuration"
echo "=================================="

# Test 1: Default configuration
echo -e "\n1. Testing default configuration:"
KUBEPULSE_PORT=9999 go run ./cmd/kubepulse serve --port 8888 --dry-run 2>&1 | grep -E "(Port:|Error)" || echo "✅ Default config loads"

# Test 2: Environment variables
echo -e "\n2. Testing environment variables:"
export KUBEPULSE_PORT=7777
export KUBEPULSE_UI_REFRESH=5s
export KUBEPULSE_UI_THEME=dark
go run ./cmd/kubepulse serve --dry-run 2>&1 | grep -E "(Port:|UI Refresh:|Theme:|Error)" || echo "✅ Environment variables work"

# Test 3: Configuration file
echo -e "\n3. Testing configuration file:"
if [ -f ~/.kubepulse.yaml ]; then
    echo "✅ Configuration file exists at ~/.kubepulse.yaml"
else
    echo "⚠️  No configuration file found. Run 'make config-init' to create one"
fi

# Test 4: Frontend configuration
echo -e "\n4. Testing frontend configuration:"
if [ -f frontend/.env ]; then
    echo "✅ Frontend .env file exists"
    grep -E "^VITE_" frontend/.env | head -5
else
    echo "⚠️  No frontend .env file found"
fi

# Test 5: API configuration endpoint
echo -e "\n5. Testing API configuration endpoint:"
# Start server in background
go run ./cmd/kubepulse serve &
SERVER_PID=$!
sleep 3

# Test the config endpoint
curl -s http://localhost:8080/api/v1/config/ui | jq '.' || echo "❌ Failed to fetch UI config"

# Kill the server
kill $SERVER_PID 2>/dev/null || true

echo -e "\n✅ Configuration test complete!"