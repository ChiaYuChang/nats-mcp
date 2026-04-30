package handler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-mcp/nats-mcp/internal/nats"
)

// RegisterMessagingTools registers basic NATS messaging tools
func RegisterMessagingTools(s *mcpserver.MCPServer, natsClient *nats.Client) {
	s.AddTool(mcp.NewTool("nats_ping",
		mcp.WithDescription("Check connection to NATS server and measure Round-Trip Time (RTT)"),
	), natsPingHandler(natsClient))

	s.AddTool(mcp.NewTool("nats_publish",
		mcp.WithDescription("Publish a message to a NATS subject"),
		mcp.WithString("subject", mcp.Required(), mcp.Description("The subject to publish to (e.g., 'orders.new')")),
		mcp.WithString("payload", mcp.Required(), mcp.Description("The message payload (JSON or plain text)")),
	), natsPublishHandler(natsClient))

	s.AddTool(mcp.NewTool("nats_request",
		mcp.WithDescription("Send a request to a NATS subject and wait for a response (Request-Reply pattern)"),
		mcp.WithString("subject", mcp.Required(), mcp.Description("The subject to send the request to")),
		mcp.WithString("payload", mcp.Required(), mcp.Description("The request payload")),
		mcp.WithNumber("timeout_ms", mcp.Description("Timeout in milliseconds (default 2000)")),
	), natsRequestHandler(natsClient))

	s.AddTool(mcp.NewTool("nats_subscribe_sync",
		mcp.WithDescription("Subscribe to a subject and wait for a specified number of messages or until timeout"),
		mcp.WithString("subject", mcp.Required(), mcp.Description("The subject to subscribe to")),
		mcp.WithNumber("count", mcp.Description("Number of messages to wait for (default 1)")),
		mcp.WithNumber("timeout_ms", mcp.Description("Timeout in milliseconds (default 5000)")),
	), natsSubscribeSyncHandler(natsClient))
}

func natsPingHandler(natsClient *nats.Client) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()
		rtt, err := natsClient.Conn.RTT()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("NATS Ping failed: %v", err)), nil
		}
		duration := time.Since(start)

		return mcp.NewToolResultText(fmt.Sprintf("NATS Server is reachable.\nRTT: %v\nTotal round-trip (client-side): %v", rtt, duration)), nil
	}
}

func natsPublishHandler(natsClient *nats.Client) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		subject, _ := request.RequireString("subject")
		payload, _ := request.RequireString("payload")

		if err := natsClient.Conn.Publish(subject, []byte(payload)); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to publish to %s: %v", subject, err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully published message to subject: %s", subject)), nil
	}
}

func natsRequestHandler(natsClient *nats.Client) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		subject, _ := request.RequireString("subject")
		payload, _ := request.RequireString("payload")
		timeoutMs := request.GetFloat("timeout_ms", 2000.0)

		timeout := time.Duration(timeoutMs) * time.Millisecond

		msg, err := natsClient.Conn.Request(subject, []byte(payload), timeout)
		if err != nil {
			if errors.Is(err, natsgo.ErrNoResponders) {
				return mcp.NewToolResultError(fmt.Sprintf("NATS Request failed: %v. (Hint: Ensure a service is actively subscribed to '%s' and is responding.)", err, subject)), nil
			} else if errors.Is(err, natsgo.ErrTimeout) {
				return mcp.NewToolResultError(fmt.Sprintf("NATS Request timed out after %v. (Hint: The responder might be slow or offline.)", timeout)), nil
			}
			return mcp.NewToolResultError(fmt.Sprintf("NATS Request failed: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Received response from %s:\n%s", subject, string(msg.Data))), nil
	}
}

func natsSubscribeSyncHandler(natsClient *nats.Client) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		subject, _ := request.RequireString("subject")
		count := int(request.GetFloat("count", 1.0))
		timeoutMs := request.GetFloat("timeout_ms", 5000.0)

		if count <= 0 {
			count = 1
		}

		sub, err := natsClient.Conn.SubscribeSync(subject)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to subscribe to %s: %v", subject, err)), nil
		}
		defer func() { _ = sub.Unsubscribe() }()

		var sb strings.Builder
		fmt.Fprintf(&sb, "Listening on %s for %d messages (Timeout: %v)...\n", subject, count, time.Duration(timeoutMs)*time.Millisecond)

		found := 0
		start := time.Now()
		deadline := start.Add(time.Duration(timeoutMs) * time.Millisecond)

		for found < count {
			timeLeft := time.Until(deadline)
			if timeLeft <= 0 {
				break
			}

			msg, err := sub.NextMsg(timeLeft)
			if err != nil {
				if errors.Is(err, natsgo.ErrTimeout) {
					break
				}
				return mcp.NewToolResultError(fmt.Sprintf("Error while waiting for messages: %v", err)), nil
			}

			found++
			fmt.Fprintf(&sb, "\n--- Message %d ---\nSubject: %s\nData: %s\n", found, msg.Subject, string(msg.Data))
		}

		if found == 0 {
			return mcp.NewToolResultText(fmt.Sprintf("No messages received on %s within %v.", subject, time.Duration(timeoutMs)*time.Millisecond)), nil
		}

		return mcp.NewToolResultText(sb.String()), nil
	}
}
