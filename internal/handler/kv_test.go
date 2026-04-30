package handler

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nats-mcp/nats-mcp/internal/nats"
)

func TestKVHandlers(t *testing.T) {
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
	bucketName := "TEST_BUCKET"

	t.Run("kv_create_bucket", func(t *testing.T) {
		handler := kvCreateBucketHandler(client)
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "kv_create_bucket",
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

	t.Run("kv_put_and_get", func(t *testing.T) {
		putHandler := kvPutHandler(client)
		getHandler := kvGetHandler(client)
		key := "foo"
		val := "bar"

		// Put
		putReq := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "kv_put",
				Arguments: map[string]interface{}{
					"bucket": bucketName,
					"key":    key,
					"value":  val,
				},
			},
		}
		_, err := putHandler(ctx, putReq)
		if err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		// Get
		getReq := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "kv_get",
				Arguments: map[string]interface{}{
					"bucket": bucketName,
					"key":    key,
				},
			},
		}
		getResp, err := getHandler(ctx, getReq)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		text := getResp.Content[0].(mcp.TextContent).Text
		if !strings.Contains(text, val) {
			t.Errorf("Expected value %s in response, got: %s", val, text)
		}
	})

	t.Run("kv_list_buckets", func(t *testing.T) {
		handler := kvListBucketsHandler(client)
		req := mcp.CallToolRequest{}
		resp, err := handler(ctx, req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
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
					"key":    "foo",
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
