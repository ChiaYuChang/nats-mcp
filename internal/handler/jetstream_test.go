package handler

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-mcp/nats-mcp/internal/nats"
	"github.com/nats-mcp/nats-mcp/internal/testutil"
)

func TestJetStreamHandlers(t *testing.T) {
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

	streamName := "TEST_STREAM"

	t.Run("jetstream_create_stream", func(t *testing.T) {
		handler := jetstreamCreateStreamHandler(client)
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "jetstream_create_stream",
				Arguments: map[string]interface{}{
					"name":      streamName,
					"subjects":  "test.js.>",
					"storage":   "memory",
					"retention": "limits",
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

	t.Run("jetstream_list_consumers", func(t *testing.T) {
		// Create a consumer first
		stream, err := client.JS.Stream(ctx, streamName)
		if err != nil {
			t.Fatalf("Failed to get stream: %v", err)
		}
		_, err = stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
			Durable: "TEST_CONSUMER",
		})
		if err != nil {
			t.Fatalf("Failed to create consumer: %v", err)
		}

		handler := jetstreamListConsumersHandler(client)
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "jetstream_list_consumers",
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
		text := resp.Content[0].(mcp.TextContent).Text
		if !strings.Contains(text, "TEST_CONSUMER") {
			t.Errorf("Expected consumer 'TEST_CONSUMER' in list, got: %s", text)
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
