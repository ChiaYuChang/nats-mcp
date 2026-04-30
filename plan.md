# NATS MCP Server Plan

## Objective
The goal of this project is to build a Model Context Protocol (MCP) server for NATS, specifically designed to integrate NATS messaging and management into the AI-assisted development workflow (e.g., Cursor, Claude Desktop). 

By providing AI models with the ability to interact with NATS, developers can use natural language to debug, monitor, and test their message-driven systems.

## Architecture
- **Language:** Go
- **MCP SDK:** `mark3labs/mcp-go`
- **Transport:** `stdio` (Standard Input/Output) for local development integration.
- **Target Audience:** Developers working with NATS-based microservices.

## Development Environment
- **NATS Server:** Docker-based (`nats:2-alpine`)
- **JetStream:** Enabled (`-js` flag)
- **Host Ports:**
  - Client: `14222` (mapped from 4222)
  - Monitoring: `18222` (mapped from 8222)

## Roadmap & Features

### Phase 1: Core Messaging (Basic Pub/Sub)
- [x] **nats_ping**
- [x] **nats_publish**
- [x] **nats_request**

### Phase 2: JetStream Management (Observability)
- [x] **jetstream_list_streams**
- [x] **jetstream_inspect_stream**
- [x] **jetstream_get_message**

### Phase 3: Key-Value (KV) Store Integration
- [x] **kv_get**
- [x] **kv_put**
- [x] **kv_list_buckets**

### Phase 4: Developer Experience (DX)
- [ ] **Connection configuration**: Support for `NATS_URL`, `NATS_USER`, `NATS_PASSWORD`, and `.creds` files.
- [ ] **Setup scripts**: Automated scripts to add the server to `claude_desktop_config.json`.
- [ ] **Detailed error reporting**: Format NATS errors into actionable feedback for the AI.

## Implementation Tasks
1. [x] Project structure initialization.
2. [x] Docker environment setup.
3. [x] Implement NATS client wrapper (`internal/nats`).
4. [x] Implement MCP Server bootstrap (`internal/server`).
5. [x] Implement Phase 1 Tools.
6. [x] Implement Phase 2 Tools.
7. [x] Implement Phase 3 Tools.
8. [ ] Testing and validation with Claude Desktop/Cursor.
