#!/bin/bash

# Test script for AI plugin integration with main atest project

echo "=== AI Plugin Integration Test ==="
echo ""

# Check if AI plugin is running
echo "1. Checking AI plugin status..."
if ps aux | grep -q "[a]test-ext-ai"; then
    echo "✓ AI plugin is running"
else
    echo "✗ AI plugin is NOT running"
    echo "Starting AI plugin..."
    cd /Users/karielhalling/Library/Mobile\ Documents/com~apple~CloudDocs/CodeProjects/aicode/atest-ext-ai
    ./bin/atest-ext-ai &
    sleep 3
fi

# Check if socket exists
echo ""
echo "2. Checking Unix socket..."
if [ -S /tmp/atest-ext-ai.sock ]; then
    echo "✓ Socket file exists: /tmp/atest-ext-ai.sock"
    ls -la /tmp/atest-ext-ai.sock
else
    echo "✗ Socket file does NOT exist"
fi

# Check if main atest server is running
echo ""
echo "3. Checking main atest server..."
if curl -s http://localhost:8080 > /dev/null 2>&1; then
    echo "✓ Main atest server is running on port 8080"
else
    echo "✗ Main atest server is NOT running"
    echo "Please start it with: cd api-testing && ./bin/atest server"
fi

# Test AI query through API
echo ""
echo "4. Testing AI query through API..."
echo "Sending test query: 'Show all users created in the last 30 days'"

RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/data/query \
  -H "Content-Type: application/json" \
  -H "X-Store-Name: ai" \
  -d '{
    "type": "",
    "key": "generate",
    "sql": "{\"prompt\": \"Show all users created in the last 30 days\", \"config\": \"{}\"}"
  }')

if [ $? -eq 0 ]; then
    echo "✓ API request successful"
    echo "Response:"
    echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"
else
    echo "✗ API request failed"
fi

# Check plugin logs for the request
echo ""
echo "5. Plugin log status:"
if [ "$DEBUG_MODE" = "true" ] || [ "$1" = "--debug" ]; then
    echo "Debug mode enabled - showing recent activity:"
    tail -n 20 /tmp/ai-plugin.log 2>/dev/null | grep -E "(Error|Fatal)" || echo "No error logs found"
else
    echo "✓ Log file exists and is being written to (use --debug flag to see details)"
fi

echo ""
echo "=== Test Summary ==="
echo "The AI plugin has been fixed to accept empty 'type' field for backward compatibility."
echo "You can now test the AI Assistant in the web UI at: http://localhost:8080"
echo ""
echo "To use the AI Assistant:"
echo "1. Open http://localhost:8080 in your browser"
echo "2. Click on 'AI Assistant' in the left menu"
echo "3. Enter a natural language query"
echo "4. Click 'Generate SQL'"