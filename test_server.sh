#!/bin/bash
# Simple test script to verify the MCP server responds correctly

echo "Testing mcp-kroki-go server..."
echo ""

# Test 1: Send initialize request
echo "Test 1: Initialize request"
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0.0"}}}' | timeout 2 ./mcp-kroki-go 2>/dev/null | head -1

echo ""
echo "If you see a JSON response above, the server is working correctly!"
echo ""
echo "To use this server with an MCP client, configure it like:"
echo '{"mcpServers":{"kroki":{"command":"/path/to/mcp-kroki-go"}}}'
