# nats-mcp

An MCP server for NATS messaging, JetStream, and Key-Value operations. It uses the MCP stdio transport and connects to NATS when it starts.

## Requirements

- Go 1.27.3 or newer for source installation.
- A reachable NATS server. JetStream must be enabled to use JetStream tools.

## Install

### 1. Install with Go

Clone the repository, then install the command from its module root:

~~~sh
git clone https://github.com/ChiaYuChang/nats-mcp.git
cd nats-mcp
go install ./cmd/nats-mcp
~~~

The executable is installed to the directory in `GOBIN`, or to `$(go env GOPATH)/bin` when `GOBIN` is unset.

### 2. Download a prebuilt binary

Download the archive for your operating system and CPU architecture from [GitHub Releases](https://github.com/ChiaYuChang/nats-mcp/releases/latest). Release builds are provided for Linux and macOS on amd64 and arm64. Each archive contains the `nats-mcp` executable; `checksums.txt` contains SHA-256 checksums.

~~~sh
tar -xzf nats-mcp-linux-amd64.tar.gz
chmod +x nats-mcp
~~~

## Configure an MCP client

Set `command` to the installed or downloaded binary's absolute path. Set `NATS_URL` to the address of your NATS server. The server defaults to `nats://localhost:14222`.

~~~json
{
  "mcpServers": {
    "nats-mcp": {
      "command": "/absolute/path/to/nats-mcp",
      "env": {
        "NATS_URL": "nats://localhost:14222"
      }
    }
  }
}
~~~

Authentication can be configured with `NATS_CREDS`, `NATS_TOKEN`, or `NATS_USER` and `NATS_PASSWORD`.

## End-to-end tests

~~~sh
go test -v -count=1 -timeout=5m ./e2e
~~~

The E2E tests require Docker. They start `nats:2-alpine` with testcontainers-go using dynamically assigned host ports, build the server binary, then check MCP initialization, tool listing, `nats_ping`, message publication and request/reply with exact payload matching, error recovery, and successful shutdown after stdin closes.

Tests are named by feature: `mcp_stdio_test.go`, `messaging_test.go`, and `error_handling_test.go`. To test an existing binary, append `-args -binary /absolute/path/to/nats-mcp` to the command. `go test ./...` also runs the E2E tests.
