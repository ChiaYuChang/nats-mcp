package handler

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nats-io/nats.go/jetstream"
	internalnats "github.com/nats-mcp/nats-mcp/internal/nats"
	"github.com/nats-mcp/nats-mcp/internal/testutil"
)

func TestConsumerHandlers(t *testing.T) {
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

	client, err := internalnats.NewClient()
	if err != nil {
		t.Fatalf("Failed to create NATS client: %v", err)
	}
	defer client.Close()

	streamName := "CONSUMER_TEST"
	consumerName := "TEST_PULL"

	// Setup Stream
	_, err = client.JS.CreateStream(ctx, jetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{"test.>"},
	})
	if err != nil {
		t.Fatalf("Failed to create stream: %v", err)
	}

	t.Run("jetstream_create_consumer", func(t *testing.T) {
		handler := jetstreamCreateConsumerHandler(client)
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "jetstream_create_consumer",
				Arguments: map[string]interface{}{
					"stream":     streamName,
					"name":       consumerName,
					"ack_wait":   "2s",
					"max_deliver": 2,
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

	t.Run("jetstream_fetch_message_and_ack", func(t *testing.T) {
		// Publish a message
		_, err := client.JS.Publish(ctx, "test.1", []byte("hello"))
		if err != nil {
			t.Fatalf("Failed to publish: %v", err)
		}

		fetchHandler := jetstreamFetchMessageHandler(client)
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "jetstream_fetch_message",
				Arguments: map[string]interface{}{
					"stream":   streamName,
					"consumer": consumerName,
				},
			},
		}
		resp, err := fetchHandler(ctx, req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if resp.IsError {
			t.Errorf("Expected success, got error: %v", resp.Content)
		}

		text := resp.Content[0].(mcp.TextContent).Text
		if !strings.Contains(text, "Ack Token") {
			t.Errorf("Expected ack token in response, got: %s", text)
		}

		// Extract Ack Token (Reply Subject)
		lines := strings.Split(text, "\n")
		var ackToken string
		for _, line := range lines {
			if strings.HasPrefix(line, "Ack Token (Reply Subject): ") {
				ackToken = strings.TrimPrefix(line, "Ack Token (Reply Subject): ")
				break
			}
		}

		if ackToken == "" {
			t.Fatalf("Could not find ack token in response")
		}

		// Ack the message
		ackHandler := jetstreamAckMessageHandler(client)
		ackReq := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "jetstream_ack_message",
				Arguments: map[string]interface{}{
					"ack_token": ackToken,
				},
			},
		}
		ackResp, err := ackHandler(ctx, ackReq)
		if err != nil {
			t.Errorf("Ack error: %v", err)
		}
		if ackResp.IsError {
			t.Errorf("Ack failed: %v", ackResp.Content)
		}

		// Verify no more messages
		resp2, _ := fetchHandler(ctx, req)
		if !strings.Contains(resp2.Content[0].(mcp.TextContent).Text, "No messages available") {
			t.Errorf("Expected no messages after ACK, got: %s", resp2.Content[0].(mcp.TextContent).Text)
		}
	})

	t.Run("jetstream_fetch_message_and_nack", func(t *testing.T) {
		// Publish another message
		_, err := client.JS.Publish(ctx, "test.2", []byte("world"))
		if err != nil {
			t.Fatalf("Failed to publish: %v", err)
		}

		fetchHandler := jetstreamFetchMessageHandler(client)
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "jetstream_fetch_message",
				Arguments: map[string]interface{}{
					"stream":   streamName,
					"consumer": consumerName,
				},
			},
		}
		resp, err := fetchHandler(ctx, req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		text := resp.Content[0].(mcp.TextContent).Text
		lines := strings.Split(text, "\n")
		var ackToken string
		for _, line := range lines {
			if strings.HasPrefix(line, "Ack Token (Reply Subject): ") {
				ackToken = strings.TrimPrefix(line, "Ack Token (Reply Subject): ")
				break
			}
		}

		// Nack the message
		nackHandler := jetstreamNackMessageHandler(client)
		nackReq := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "jetstream_nack_message",
				Arguments: map[string]interface{}{
					"ack_token": ackToken,
				},
			},
		}
		_, _ = nackHandler(ctx, nackReq)

		// Message should be available again immediately (depending on redelivery)
		time.Sleep(100 * time.Millisecond)
		resp2, _ := fetchHandler(ctx, req)
		if strings.Contains(resp2.Content[0].(mcp.TextContent).Text, "No messages available") {
			t.Errorf("Expected message to be available after NACK")
		}
	})

	t.Run("jetstream_delete_consumer", func(t *testing.T) {
		handler := jetstreamDeleteConsumerHandler(client)
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "jetstream_delete_consumer",
				Arguments: map[string]interface{}{
					"stream":   streamName,
					"consumer": consumerName,
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
