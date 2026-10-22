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

### 3. Install with Nix

~~~sh
nix profile add github:ChiaYuChang/nats-mcp
~~~

Run without installing:

~~~sh
nix run github:ChiaYuChang/nats-mcp -- --version
~~~

From a local checkout:

~~~sh
nix build path:.
./result/bin/nats-mcp --version
~~~

The Nix package downloads the fixed `v0.1.0` release binary for Linux or macOS on amd64 or arm64 and verifies its SHA-256 hash. Go is not required. `default.nix` defines the release version and platform hashes; update these together when packaging a new release. `flake.lock` pins nixpkgs 26.05, which supports all four platforms.

## Configure an MCP client

Check the installed version without connecting to NATS:

~~~sh
nats-mcp --version
~~~

Release binaries report their tag, such as `nats-mcp v0.1.0`. Local builds default to `nats-mcp dev`. The MCP initialization response uses the same version.

Pushing a stable `vX.Y.Z` tag triggers release builds. Tags with leading zeros or suffixes are rejected. Each Linux/macOS and amd64/arm64 archive is extracted and its binary executed on a matching runner to verify that `--version` equals `nats-mcp <tag>`. All four checks must pass before publishing the release.

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
