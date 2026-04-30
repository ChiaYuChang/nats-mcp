package handler

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-mcp/nats-mcp/internal/nats"
	"github.com/nats-mcp/nats-mcp/internal/testutil"
)

func TestMessagingHandlers(t *testing.T) {
	ctx := context.Background()
	token := "test-token"
	container, err := testutil.SetupNATS(ctx, token)
	if err != nil {
		t.Fatalf("Failed to setup NATS: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	t.Setenv("NATS_URL", container.URL)
	t.Setenv("NATS_TOKEN", token)

	client, err := nats.NewClient()
	if err != nil {
		t.Fatalf("Failed to create NATS client: %v", err)
	}
	defer client.Close()

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

	t.Run("nats_publish_and_subscribe", func(t *testing.T) {
		subject := "test.messaging"
		payload := "hello world"

		// Start a subscription in a separate goroutine or use the sync tool
		// For simplicity, we use the nats_subscribe_sync handler
		subHandler := natsSubscribeSyncHandler(client)
		pubHandler := natsPublishHandler(client)

		// Publish message
		pubReq := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "nats_publish",
				Arguments: map[string]interface{}{
					"subject": subject,
					"payload": payload,
				},
			},
		}

		// We need to subscribe first then publish.
		// Since subHandler is synchronous and blocking, we'd need to run it in a goroutine.
		// Alternatively, we can just test them independently or use a small delay.

		go func() {
			_, _ = pubHandler(ctx, pubReq)
		}()

		subReq := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "nats_subscribe_sync",
				Arguments: map[string]interface{}{
					"subject":    subject,
					"count":      1.0,
					"timeout_ms": 2000.0,
				},
			},
		}

		resp, err := subHandler(ctx, subReq)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if resp.IsError {
			t.Errorf("Expected success, got error: %v", resp.Content)
		}
		text := resp.Content[0].(mcp.TextContent).Text
		if !strings.Contains(text, payload) {
			t.Errorf("Expected payload '%s' in received message, got: %s", payload, text)
		}
	})

	t.Run("nats_request", func(t *testing.T) {
		subject := "test.request"
		requestPayload := "ping"
		responsePayload := "pong"

		// Setup a responder
		_, err := client.Conn.Subscribe(subject, func(msg *natsgo.Msg) {
			_ = msg.Respond([]byte(responsePayload))
		})
		if err != nil {
			t.Fatalf("Failed to setup responder: %v", err)
		}

		handler := natsRequestHandler(client)
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "nats_request",
				Arguments: map[string]interface{}{
					"subject":    subject,
					"payload":    requestPayload,
					"timeout_ms": 1000.0,
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
		if !strings.Contains(text, responsePayload) {
			t.Errorf("Expected response '%s', got: %s", responsePayload, text)
		}
	})
}
