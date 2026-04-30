package server

import (
	"os"
	"testing"

	"github.com/nats-mcp/nats-mcp/internal/nats"
)

func TestNatsServer(t *testing.T) {
	if os.Getenv("NATS_URL") == "" {
		t.Setenv("NATS_URL", "nats://localhost:14222")
	}
	t.Setenv("NATS_TOKEN", "test-token")

	client, err := nats.NewClient()
	if err != nil {
		t.Fatalf("Failed to create NATS client: %v", err)
	}
	defer client.Close()

	srv := NewNatsServer(client)
	if srv == nil {
		t.Fatal("Expected NewNatsServer to return a server instance")
	}

	if srv.MCPServer == nil {
		t.Fatal("Expected MCPServer to be initialized")
	}

	// Test RegisterTools
	srv.RegisterTools()
	
	tools := srv.MCPServer.ListTools()
	if len(tools) == 0 {
		t.Error("Expected tools to be registered, but got 0")
	}
}
