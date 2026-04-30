package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/nats-mcp/nats-mcp/internal/nats"
)

// RegisterKVTools registers NATS Key-Value tools
func RegisterKVTools(s *mcpserver.MCPServer, natsClient *nats.Client) {
	// 1. kv_list_buckets
	s.AddTool(mcp.NewTool("kv_list_buckets",
		mcp.WithDescription("List all Key-Value buckets"),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		buckets := natsClient.JS.KeyValueStoreNames(ctx)
		var sb strings.Builder
		sb.WriteString("Key-Value Buckets:\n")
		
		count := 0
		for bucket := range buckets.Name() {
			sb.WriteString(fmt.Sprintf("- %s\n", bucket))
			count++
		}

		if count == 0 {
			return mcp.NewToolResultText("No Key-Value buckets found."), nil
		}

		return mcp.NewToolResultText(sb.String()), nil
	})

	// 2. kv_get
	s.AddTool(mcp.NewTool("kv_get",
		mcp.WithDescription("Get a value from a Key-Value bucket"),
		mcp.WithString("bucket", mcp.Required(), mcp.Description("The name of the KV bucket")),
		mcp.WithString("key", mcp.Required(), mcp.Description("The key to retrieve")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	})

	// 3. kv_put
	s.AddTool(mcp.NewTool("kv_put",
		mcp.WithDescription("Put/Update a value in a Key-Value bucket"),
		mcp.WithString("bucket", mcp.Required(), mcp.Description("The name of the KV bucket")),
		mcp.WithString("key", mcp.Required(), mcp.Description("The key to set")),
		mcp.WithString("value", mcp.Required(), mcp.Description("The value to set")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	})
}
