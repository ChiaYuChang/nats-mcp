package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/nats-mcp/nats-mcp/internal/nats"
)

// RegisterMessagingTools registers basic NATS messaging tools
func RegisterMessagingTools(s *mcpserver.MCPServer, natsClient *nats.Client) {
	// 1. nats_ping
	s.AddTool(mcp.NewTool("nats_ping",
		mcp.WithDescription("Check connection to NATS server and measure Round-Trip Time (RTT)"),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()
		rtt, err := natsClient.Conn.RTT()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("NATS Ping failed: %v", err)), nil
		}
		duration := time.Since(start)

		return mcp.NewToolResultText(fmt.Sprintf("NATS Server is reachable.\nRTT: %v\nTotal round-trip (client-side): %v", rtt, duration)), nil
	})

	// 2. nats_publish
	s.AddTool(mcp.NewTool("nats_publish",
		mcp.WithDescription("Publish a message to a NATS subject"),
		mcp.WithString("subject", mcp.Required(), mcp.Description("The subject to publish to (e.g., 'orders.new')")),
		mcp.WithString("payload", mcp.Required(), mcp.Description("The message payload (JSON or plain text)")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		subject, _ := request.RequireString("subject")
		payload, _ := request.RequireString("payload")

		if err := natsClient.Conn.Publish(subject, []byte(payload)); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to publish to %s: %v", subject, err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully published message to subject: %s", subject)), nil
	})

	// 3. nats_request
	s.AddTool(mcp.NewTool("nats_request",
		mcp.WithDescription("Send a request to a NATS subject and wait for a response (Request-Reply pattern)"),
		mcp.WithString("subject", mcp.Required(), mcp.Description("The subject to send the request to")),
		mcp.WithString("payload", mcp.Required(), mcp.Description("The request payload")),
		mcp.WithNumber("timeout_ms", mcp.Description("Timeout in milliseconds (default 2000)")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		subject, _ := request.RequireString("subject")
		payload, _ := request.RequireString("payload")
		timeoutMs := request.GetFloat("timeout_ms", 2000.0)
		
		timeout := time.Duration(timeoutMs) * time.Millisecond

		msg, err := natsClient.Conn.Request(subject, []byte(payload), timeout)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("NATS Request failed: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Received response from %s:\n%s", subject, string(msg.Data))), nil
	})
}
