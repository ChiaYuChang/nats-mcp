#!/bin/bash
# Install script for NATS MCP Server to Claude Desktop
set -e

# Compile binary
echo "Compiling nats-mcp..."
mkdir -p bin
go build -o bin/nats-mcp ./cmd/nats-mcp/main.go

# Get absolute path to binary
MCP_BINARY=$(pwd)/bin/nats-mcp

CLAUDE_CONFIG_DIR=""
if [[ "$OSTYPE" == "darwin"* ]]; then
    CLAUDE_CONFIG_DIR="$HOME/Library/Application Support/Claude"
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    CLAUDE_CONFIG_DIR="$HOME/.config/Claude"
fi

if [ -z "$CLAUDE_CONFIG_DIR" ]; then
    echo "Unsupported OS for automatic Claude Desktop setup."
    echo "Please manually add the following configuration to your MCP client:"
    echo "{"
    echo "  \"mcpServers\": {"
    echo "    \"nats-mcp\": {"
    echo "      \"command\": \"$MCP_BINARY\","
    echo "      \"env\": { \"NATS_URL\": \"nats://localhost:14222\" }"
    echo "    }"
    echo "  }"
    echo "}"
    exit 1
fi

CONFIG_FILE="$CLAUDE_CONFIG_DIR/claude_desktop_config.json"
mkdir -p "$CLAUDE_CONFIG_DIR"

if [ ! -f "$CONFIG_FILE" ]; then
    echo "{ \"mcpServers\": {} }" > "$CONFIG_FILE"
fi

python3 -c "
import json
import sys

config_file = sys.argv[1]
mcp_binary = sys.argv[2]

try:
    with open(config_file, 'r') as f:
        data = json.load(f)
except Exception:
    data = {'mcpServers': {}}

if 'mcpServers' not in data:
    data['mcpServers'] = {}

data['mcpServers']['nats-mcp'] = {
    'command': mcp_binary,
    'args': [],
    'env': {
        'NATS_URL': 'nats://localhost:14222'
    }
}

with open(config_file, 'w') as f:
    json.dump(data, f, indent=2)

print(f'Successfully added nats-mcp to {config_file}')
" "$CONFIG_FILE" "$MCP_BINARY"

echo "Please restart Claude Desktop for the changes to take effect."
