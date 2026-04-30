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

// RegisterJetStreamTools registers NATS JetStream tools
func RegisterJetStreamTools(s *mcpserver.MCPServer, natsClient *nats.Client) {
	s.AddTool(mcp.NewTool("jetstream_list_streams",
		mcp.WithDescription("List all JetStream streams and their basic statistics"),
	), jetstreamListStreamsHandler(natsClient))

	s.AddTool(mcp.NewTool("jetstream_inspect_stream",
		mcp.WithDescription("Retrieve recent messages from a stream for debugging"),
		mcp.WithString("stream", mcp.Required(), mcp.Description("The name of the stream to inspect")),
		mcp.WithNumber("count", mcp.Description("Number of recent messages to retrieve (default 5, max 50)")),
	), jetstreamInspectStreamHandler(natsClient))

	s.AddTool(mcp.NewTool("jetstream_get_message",
		mcp.WithDescription("Get a specific message from a stream by sequence number"),
		mcp.WithString("stream", mcp.Required(), mcp.Description("The name of the stream")),
		mcp.WithNumber("sequence", mcp.Required(), mcp.Description("The sequence number of the message")),
	), jetstreamGetMessageHandler(natsClient))

	s.AddTool(mcp.NewTool("jetstream_create_stream",
		mcp.WithDescription("Create a new JetStream stream"),
		mcp.WithString("name", mcp.Required(), mcp.Description("The name of the stream")),
		mcp.WithString("subjects", mcp.Required(), mcp.Description("Comma-separated list of subjects (e.g., 'orders.*,events.>')")),
	), jetstreamCreateStreamHandler(natsClient))

	s.AddTool(mcp.NewTool("jetstream_delete_stream",
		mcp.WithDescription("Delete a JetStream stream"),
		mcp.WithString("stream", mcp.Required(), mcp.Description("The name of the stream to delete")),
	), jetstreamDeleteStreamHandler(natsClient))

	s.AddTool(mcp.NewTool("jetstream_purge_stream",
		mcp.WithDescription("Purge all messages from a JetStream stream"),
		mcp.WithString("stream", mcp.Required(), mcp.Description("The name of the stream to purge")),
	), jetstreamPurgeStreamHandler(natsClient))
}

func jetstreamListStreamsHandler(natsClient *nats.Client) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		names := natsClient.JS.StreamNames(ctx)
		var sb strings.Builder
		sb.WriteString("JetStream Streams:\n")

		count := 0
		for name := range names.Name() {
			stream, err := natsClient.JS.Stream(ctx, name)
			if err != nil {
				fmt.Fprintf(&sb, "- Error getting stream %s: %v\n", name, err)
				continue
			}
			info, err := stream.Info(ctx)
			if err != nil {
				fmt.Fprintf(&sb, "- Error getting info for stream %s: %v\n", name, err)
				continue
			}
			fmt.Fprintf(&sb, "- %s (Messages: %d, Bytes: %d, Subjects: %v)\n",
				info.Config.Name, info.State.Msgs, info.State.Bytes, info.Config.Subjects)
			count++
		}

		if count == 0 {
			return mcp.NewToolResultText("No JetStream streams found."), nil
		}

		return mcp.NewToolResultText(sb.String()), nil
	}
}

func jetstreamInspectStreamHandler(natsClient *nats.Client) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		fmt.Fprintf(&sb, "Recent %d messages from stream %s (starting at seq %d):\n", count, streamName, startSeq)

		found := 0
		for seq := startSeq; seq <= info.State.LastSeq; seq++ {
			msg, err := stream.GetMsg(ctx, seq)
			if err != nil {
				continue // Skip missing messages
			}
			fmt.Fprintf(&sb, "\n--- Sequence: %d ---\nSubject: %s\nData: %s\n",
				msg.Sequence, msg.Subject, string(msg.Data))
			found++
		}

		if found == 0 {
			return mcp.NewToolResultText(fmt.Sprintf("No messages found in the requested range for stream %s.", streamName)), nil
		}

		return mcp.NewToolResultText(sb.String()), nil
	}
}

func jetstreamGetMessageHandler(natsClient *nats.Client) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	}
}

func jetstreamCreateStreamHandler(natsClient *nats.Client) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, _ := request.RequireString("name")
		subjectsStr, _ := request.RequireString("subjects")
		subjects := strings.Split(subjectsStr, ",")
		for i := range subjects {
			subjects[i] = strings.TrimSpace(subjects[i])
		}

		_, err := natsClient.JS.CreateStream(ctx, jetstream.StreamConfig{
			Name:     name,
			Subjects: subjects,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create stream %s: %v", name, err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully created stream: %s", name)), nil
	}
}

func jetstreamDeleteStreamHandler(natsClient *nats.Client) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		streamName, _ := request.RequireString("stream")

		if err := natsClient.JS.DeleteStream(ctx, streamName); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to delete stream %s: %v", streamName, err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully deleted stream: %s", streamName)), nil
	}
}

func jetstreamPurgeStreamHandler(natsClient *nats.Client) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		streamName, _ := request.RequireString("stream")

		stream, err := natsClient.JS.Stream(ctx, streamName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get stream %s: %v", streamName, err)), nil
		}

		if err := stream.Purge(ctx); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to purge stream %s: %v", streamName, err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully purged stream: %s", streamName)), nil
	}
}
