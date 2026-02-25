#!/usr/bin/env bash
# Test graceful shutdown

set -e

echo "=== Graceful Shutdown Test ==="

# Start server
echo "Starting goclaw server..."
./bin/goclaw.exe -config config/config.example.yaml > /tmp/goclaw-test.log 2>&1 &
PID=$!
echo "Server started with PID: $PID"

# Wait for server to be ready
echo "Waiting for server to be ready..."
sleep 3

# Test health check
echo "Testing health check..."
curl -s http://localhost:8080/health
echo ""

# Submit a workflow
echo "Submitting test workflow..."
WORKFLOW_ID=$(curl -s -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{"name":"shutdown-test","tasks":[{"id":"task-1","name":"Test","type":"function"}]}' \
  | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Workflow ID: $WORKFLOW_ID"

# Send SIGTERM for graceful shutdown
echo "Sending SIGTERM to server..."
kill -TERM $PID

# Wait for shutdown
echo "Waiting for graceful shutdown..."
wait $PID 2>/dev/null || true

echo "Server stopped"

# Check logs for graceful shutdown messages
echo ""
echo "=== Shutdown Logs ==="
grep -i "shutdown\|stopping\|stopped" /tmp/goclaw-test.log || echo "No shutdown logs found"

echo ""
echo "=== Test Complete ==="
