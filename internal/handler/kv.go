package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-mcp/nats-mcp/internal/nats"
)

// RegisterKVTools registers NATS Key-Value tools
func RegisterKVTools(s *mcpserver.MCPServer, natsClient *nats.Client) {
	s.AddTool(mcp.NewTool("kv_list_buckets",
		mcp.WithDescription("List all Key-Value buckets"),
	), kvListBucketsHandler(natsClient))

	s.AddTool(mcp.NewTool("kv_get",
		mcp.WithDescription("Get a value from a Key-Value bucket"),
		mcp.WithString("bucket", mcp.Required(), mcp.Description("The name of the KV bucket")),
		mcp.WithString("key", mcp.Required(), mcp.Description("The key to retrieve")),
	), kvGetHandler(natsClient))

	s.AddTool(mcp.NewTool("kv_put",
		mcp.WithDescription("Put/Update a value in a Key-Value bucket"),
		mcp.WithString("bucket", mcp.Required(), mcp.Description("The name of the KV bucket")),
		mcp.WithString("key", mcp.Required(), mcp.Description("The key to set")),
		mcp.WithString("value", mcp.Required(), mcp.Description("The value to set")),
	), kvPutHandler(natsClient))

	s.AddTool(mcp.NewTool("kv_create_bucket",
		mcp.WithDescription("Create a new Key-Value bucket"),
		mcp.WithString("bucket", mcp.Required(), mcp.Description("The name of the bucket to create")),
	), kvCreateBucketHandler(natsClient))

	s.AddTool(mcp.NewTool("kv_delete_key",
		mcp.WithDescription("Delete a key from a Key-Value bucket"),
		mcp.WithString("bucket", mcp.Required(), mcp.Description("The name of the KV bucket")),
		mcp.WithString("key", mcp.Required(), mcp.Description("The key to delete")),
	), kvDeleteKeyHandler(natsClient))

	s.AddTool(mcp.NewTool("kv_delete_bucket",
		mcp.WithDescription("Delete an entire Key-Value bucket"),
		mcp.WithString("bucket", mcp.Required(), mcp.Description("The name of the bucket to delete")),
	), kvDeleteBucketHandler(natsClient))
}

func kvListBucketsHandler(natsClient *nats.Client) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		buckets := natsClient.JS.KeyValueStoreNames(ctx)
		var sb strings.Builder
		sb.WriteString("Key-Value Buckets:\n")

		count := 0
		for bucket := range buckets.Name() {
			fmt.Fprintf(&sb, "- %s\n", bucket)
			count++
		}

		if count == 0 {
			return mcp.NewToolResultText("No Key-Value buckets found."), nil
		}

		return mcp.NewToolResultText(sb.String()), nil
	}
}

func kvGetHandler(natsClient *nats.Client) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		bucketName, _ := request.RequireString("bucket")
		key, _ := request.RequireString("key")

		kv, err := natsClient.JS.KeyValue(ctx, bucketName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to bind to bucket %s: %v", bucketName, err)), nil
		}

		entry, err := kv.Get(ctx, key)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get key %s from bucket %s: %v", key, bucketName, err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Bucket: %s\nKey: %s\nValue: %s\nRevision: %d",
			bucketName, key, string(entry.Value()), entry.Revision())), nil
	}
}

func kvPutHandler(natsClient *nats.Client) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		bucketName, _ := request.RequireString("bucket")
		key, _ := request.RequireString("key")
		value, _ := request.RequireString("value")

		kv, err := natsClient.JS.KeyValue(ctx, bucketName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to bind to bucket %s: %v", bucketName, err)), nil
		}

		rev, err := kv.Put(ctx, key, []byte(value))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to put key %s in bucket %s: %v", key, bucketName, err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully set %s in bucket %s (Revision: %d)", key, bucketName, rev)), nil
	}
}

func kvCreateBucketHandler(natsClient *nats.Client) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		bucketName, _ := request.RequireString("bucket")

		_, err := natsClient.JS.CreateKeyValue(ctx, jetstream.KeyValueConfig{
			Bucket: bucketName,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create bucket %s: %v", bucketName, err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully created Key-Value bucket: %s", bucketName)), nil
	}
}

func kvDeleteKeyHandler(natsClient *nats.Client) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		bucketName, _ := request.RequireString("bucket")
		key, _ := request.RequireString("key")

		kv, err := natsClient.JS.KeyValue(ctx, bucketName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to bind to bucket %s: %v", bucketName, err)), nil
		}

		if err := kv.Delete(ctx, key); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to delete key %s from bucket %s: %v", key, bucketName, err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully deleted key %s from bucket %s", key, bucketName)), nil
	}
}

func kvDeleteBucketHandler(natsClient *nats.Client) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		bucketName, _ := request.RequireString("bucket")

		if err := natsClient.JS.DeleteKeyValue(ctx, bucketName); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to delete bucket %s: %v", bucketName, err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully deleted Key-Value bucket: %s", bucketName)), nil
	}
}
