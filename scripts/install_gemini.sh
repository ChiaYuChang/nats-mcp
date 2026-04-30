#!/bin/bash
# Install script for NATS MCP Server to Gemini CLI
set -e

# Compile binary
echo "Compiling nats-mcp..."
mkdir -p bin
go build -o bin/nats-mcp ./cmd/nats-mcp/main.go

# Get absolute path to binary
MCP_BINARY=$(pwd)/bin/nats-mcp

echo "Adding nats-mcp to Gemini CLI configuration..."
# Use gemini mcp add to register the server in the project-level settings
gemini mcp add nats-mcp "$MCP_BINARY" --env "NATS_URL=nats://localhost:14222"

echo "Trusting the current directory for MCP execution..."
gemini trust

echo "--------------------------------------------------------"
echo "Setup complete!"
echo "To start using NATS tools in this session, please run:"
echo "  /mcp reload"
echo "Then verify with:"
echo "  /mcp list"
echo "--------------------------------------------------------"
