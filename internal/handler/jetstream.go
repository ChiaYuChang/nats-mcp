package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/nats-mcp/nats-mcp/internal/nats"
)

// RegisterJetStreamTools registers NATS JetStream tools
func RegisterJetStreamTools(s *mcpserver.MCPServer, natsClient *nats.Client) {
	// 1. jetstream_list_streams
	s.AddTool(mcp.NewTool("jetstream_list_streams",
		mcp.WithDescription("List all JetStream streams and their basic statistics"),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		names := natsClient.JS.StreamNames(ctx)
		var sb strings.Builder
		sb.WriteString("JetStream Streams:\n")

		count := 0
		for name := range names.Name() {
			stream, err := natsClient.JS.Stream(ctx, name)
			if err != nil {
				sb.WriteString(fmt.Sprintf("- Error getting stream %s: %v\n", name, err))
				continue
			}
			info, err := stream.Info(ctx)
			if err != nil {
				sb.WriteString(fmt.Sprintf("- Error getting info for stream %s: %v\n", name, err))
				continue
			}
			sb.WriteString(fmt.Sprintf("- %s (Messages: %d, Bytes: %d, Subjects: %v)\n",
				info.Config.Name, info.State.Msgs, info.State.Bytes, info.Config.Subjects))
			count++
		}

		if count == 0 {
			return mcp.NewToolResultText("No JetStream streams found."), nil
		}

		return mcp.NewToolResultText(sb.String()), nil
	})

	// 2. jetstream_inspect_stream
	s.AddTool(mcp.NewTool("jetstream_inspect_stream",
		mcp.WithDescription("Retrieve recent messages from a stream for debugging"),
		mcp.WithString("stream", mcp.Required(), mcp.Description("The name of the stream to inspect")),
		mcp.WithNumber("count", mcp.Description("Number of recent messages to retrieve (default 5, max 50)")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		streamName, _ := request.RequireString("stream")
		count := int(request.GetFloat("count", 5.0))
		if count > 50 {
			count = 50
		}

		stream, err := natsClient.JS.Stream(ctx, streamName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get stream %s: %v", streamName, err)), nil
		}

		info, err := stream.Info(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get stream info for %s: %v", streamName, err)), nil
		}

		if info.State.Msgs == 0 {
			return mcp.NewToolResultText(fmt.Sprintf("Stream %s is empty.", streamName)), nil
		}

		// Calculate start sequence
		startSeq := uint64(1)
		if info.State.Msgs > uint64(count) {
			startSeq = info.State.LastSeq - uint64(count) + 1
		}
		if startSeq < info.State.FirstSeq {
			startSeq = info.State.FirstSeq
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Recent %d messages from stream %s (starting at seq %d):\n", count, streamName, startSeq))

		found := 0
		for seq := startSeq; seq <= info.State.LastSeq; seq++ {
			msg, err := stream.GetMsg(ctx, seq)
			if err != nil {
				continue // Skip missing messages
			}
			sb.WriteString(fmt.Sprintf("\n--- Sequence: %d ---\nSubject: %s\nData: %s\n",
				msg.Sequence, msg.Subject, string(msg.Data)))
			found++
		}

		if found == 0 {
			return mcp.NewToolResultText(fmt.Sprintf("No messages found in the requested range for stream %s.", streamName)), nil
		}

		return mcp.NewToolResultText(sb.String()), nil
	})

	// 3. jetstream_get_message
	s.AddTool(mcp.NewTool("jetstream_get_message",
		mcp.WithDescription("Get a specific message from a stream by sequence number"),
		mcp.WithString("stream", mcp.Required(), mcp.Description("The name of the stream")),
		mcp.WithNumber("sequence", mcp.Required(), mcp.Description("The sequence number of the message")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		streamName, _ := request.RequireString("stream")
		sequence := uint64(request.GetFloat("sequence", 0))

		stream, err := natsClient.JS.Stream(ctx, streamName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get stream %s: %v", streamName, err)), nil
		}

		msg, err := stream.GetMsg(ctx, sequence)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get message %d from stream %s: %v", sequence, streamName, err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Message %d from stream %s:\nSubject: %s\nTime: %v\nData: %s",
			msg.Sequence, streamName, msg.Subject, msg.Time, string(msg.Data))), nil
	})
}
