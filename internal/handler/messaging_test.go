package handler

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nats-mcp/nats-mcp/internal/nats"
)

func TestMessagingHandlers(t *testing.T) {
	if os.Getenv("NATS_URL") == "" {
		t.Setenv("NATS_URL", "nats://localhost:14222")
	}
	t.Setenv("NATS_TOKEN", "test-token")

	client, err := nats.NewClient()
	if err != nil {
		t.Fatalf("Failed to create NATS client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	t.Run("nats_ping", func(t *testing.T) {
		handler := natsPingHandler(client)
		req := mcp.CallToolRequest{}
		resp, err := handler(ctx, req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if resp.IsError {
			t.Errorf("Expected success, got error: %v", resp.Content)
		}
	})

	t.Run("nats_publish", func(t *testing.T) {
		handler := natsPublishHandler(client)
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "nats_publish",
				Arguments: map[string]interface{}{
					"subject": "test.publish",
					"payload": "hello",
				},
			},
		}
		resp, err := handler(ctx, req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if resp.IsError {
			t.Errorf("Expected success, got error: %v", resp.Content)
		}
	})

	t.Run("nats_request_no_responder", func(t *testing.T) {
		handler := natsRequestHandler(client)
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "nats_request",
				Arguments: map[string]interface{}{
					"subject": "test.request.none",
					"payload": "hello",
				},
			},
		}
		resp, err := handler(ctx, req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		// Expect error since no one is listening
		if !resp.IsError {
			t.Error("Expected error for no responders, but got success")
		}
		if !strings.Contains(resp.Content[0].(mcp.TextContent).Text, "no responders") {
			t.Errorf("Expected 'no responders' error, got: %v", resp.Content[0].(mcp.TextContent).Text)
		}
	})

	t.Run("nats_subscribe_sync", func(t *testing.T) {
		handler := natsSubscribeSyncHandler(client)
		subject := "test.subscribe.sync"
		payload := "ping"

		// Send message in background
		go func() {
			time.Sleep(100 * time.Millisecond)
			_ = client.Conn.Publish(subject, []byte(payload))
		}()

		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "nats_subscribe_sync",
				Arguments: map[string]interface{}{
					"subject": subject,
					"count":   1,
				},
			},
		}
		resp, err := handler(ctx, req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if resp.IsError {
			t.Errorf("Expected success, got error: %v", resp.Content)
		}
		text := resp.Content[0].(mcp.TextContent).Text
		if !strings.Contains(text, payload) {
			t.Errorf("Expected payload %s in response, got: %s", payload, text)
		}
	})
}
