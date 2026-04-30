package main

import (
	"log"
	"os"

	"github.com/nats-mcp/nats-mcp/internal/nats"
	"github.com/nats-mcp/nats-mcp/internal/server"
)

func main() {
	// 1. Initialize NATS Client
	natsClient, err := nats.NewClient()
	if err != nil {
		log.Fatalf("Critical error: could not connect to NATS: %v", err)
	}
	defer natsClient.Close()

	// 2. Initialize MCP Server
	natsMCP := server.NewNatsServer(natsClient)

	// 3. Register Tools
	natsMCP.RegisterTools()

	// 4. Start Serving via Stdio
	log.Printf("Starting NATS MCP Server...")
	if err := natsMCP.Serve(); err != nil {
		log.Fatalf("Server error: %v", err)
		os.Exit(1)
	}
}
