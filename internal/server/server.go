package server

import (
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/nats-mcp/nats-mcp/internal/handler"
	"github.com/nats-mcp/nats-mcp/internal/nats"
	"github.com/nats-mcp/nats-mcp/internal/version"
)

// NatsServer wraps the MCP server and its dependencies
type NatsServer struct {
	MCPServer *mcpserver.MCPServer
	Nats      *nats.Client
}

// NewNatsServer creates a new MCP server instance for NATS
func NewNatsServer(natsClient *nats.Client) *NatsServer {
	s := mcpserver.NewMCPServer("nats-mcp", version.Version)

	return &NatsServer{
		MCPServer: s,
		Nats:      natsClient,
	}
}

// Serve starts the MCP server using stdio transport
func (n *NatsServer) Serve() error {
	return mcpserver.ServeStdio(n.MCPServer)
}

// RegisterTools will be called to add all NATS tools to the server
func (n *NatsServer) RegisterTools() {
	handler.RegisterMessagingTools(n.MCPServer, n.Nats)
	handler.RegisterJetStreamTools(n.MCPServer, n.Nats)
	handler.RegisterKVTools(n.MCPServer, n.Nats)
}
