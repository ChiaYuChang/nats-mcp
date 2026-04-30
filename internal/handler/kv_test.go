package handler

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nats-mcp/nats-mcp/internal/nats"
	"github.com/nats-mcp/nats-mcp/internal/testutil"
)

func TestKVHandlers(t *testing.T) {
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

	bucketName := "TEST_BUCKET"

	t.Run("kv_create_bucket", func(t *testing.T) {
		handler := kvCreateBucketHandler(client)
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "kv_create_bucket",
				Arguments: map[string]interface{}{
					"bucket":      bucketName,
					"ttl_seconds": 60.0,
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

	t.Run("kv_put", func(t *testing.T) {
		handler := kvPutHandler(client)
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "kv_put",
				Arguments: map[string]interface{}{
					"bucket": bucketName,
					"key":    "test_key",
					"value":  "test_value",
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

	t.Run("kv_list_keys", func(t *testing.T) {
		handler := kvListKeysHandler(client)
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "kv_list_keys",
				Arguments: map[string]interface{}{
					"bucket": bucketName,
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
		if !strings.Contains(text, "test_key") {
			t.Errorf("Expected key 'test_key' in list, got: %s", text)
		}
	})

	t.Run("kv_get", func(t *testing.T) {
		handler := kvGetHandler(client)
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "kv_get",
				Arguments: map[string]interface{}{
					"bucket": bucketName,
					"key":    "test_key",
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
		if !strings.Contains(text, "test_value") {
			t.Errorf("Expected value 'test_value' in response, got: %s", text)
		}
	})

	t.Run("kv_list_buckets", func(t *testing.T) {
		handler := kvListBucketsHandler(client)
		req := mcp.CallToolRequest{}
		resp, err := handler(ctx, req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if resp.IsError {
			t.Errorf("Expected success, got error: %v", resp.Content)
		}
		text := resp.Content[0].(mcp.TextContent).Text
		if !strings.Contains(text, bucketName) {
			t.Errorf("Expected bucket %s in list, got: %s", bucketName, text)
		}
	})

	t.Run("kv_delete_key", func(t *testing.T) {
		handler := kvDeleteKeyHandler(client)
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "kv_delete_key",
				Arguments: map[string]interface{}{
					"bucket": bucketName,
					"key":    "test_key",
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

	t.Run("kv_delete_bucket", func(t *testing.T) {
		handler := kvDeleteBucketHandler(client)
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "kv_delete_bucket",
				Arguments: map[string]interface{}{
					"bucket": bucketName,
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
