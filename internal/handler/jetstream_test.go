package handler

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nats-mcp/nats-mcp/internal/nats"
)

func TestJetStreamHandlers(t *testing.T) {
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
	streamName := "TEST_STREAM"

	t.Run("jetstream_create_stream", func(t *testing.T) {
		handler := jetstreamCreateStreamHandler(client)
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "jetstream_create_stream",
				Arguments: map[string]interface{}{
					"name":     streamName,
					"subjects": "test.js.>",
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

	t.Run("jetstream_list_streams", func(t *testing.T) {
		handler := jetstreamListStreamsHandler(client)
		req := mcp.CallToolRequest{}
		resp, err := handler(ctx, req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if resp.IsError {
			t.Errorf("Expected success, got error: %v", resp.Content)
		}
		text := resp.Content[0].(mcp.TextContent).Text
		if !strings.Contains(text, streamName) {
			t.Errorf("Expected stream %s in list, got: %s", streamName, text)
		}
	})

	t.Run("jetstream_purge_stream", func(t *testing.T) {
		handler := jetstreamPurgeStreamHandler(client)
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "jetstream_purge_stream",
				Arguments: map[string]interface{}{
					"stream": streamName,
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

	t.Run("jetstream_delete_stream", func(t *testing.T) {
		handler := jetstreamDeleteStreamHandler(client)
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "jetstream_delete_stream",
				Arguments: map[string]interface{}{
					"stream": streamName,
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
}
