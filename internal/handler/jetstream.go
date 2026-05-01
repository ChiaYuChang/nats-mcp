package handler

import (
	"context"
	"fmt"
	"strings"
	"time"

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
		mcp.WithString("storage", mcp.Description("Storage type: 'file' or 'memory' (default 'file')")),
		mcp.WithString("retention", mcp.Description("Retention policy: 'limits', 'interest', 'workqueue' (default 'limits')")),
	), jetstreamCreateStreamHandler(natsClient))

	s.AddTool(mcp.NewTool("jetstream_delete_stream",
		mcp.WithDescription("Delete a JetStream stream"),
		mcp.WithString("stream", mcp.Required(), mcp.Description("The name of the stream to delete")),
	), jetstreamDeleteStreamHandler(natsClient))

	s.AddTool(mcp.NewTool("jetstream_purge_stream",
		mcp.WithDescription("Purge all messages from a JetStream stream"),
		mcp.WithString("stream", mcp.Required(), mcp.Description("The name of the stream to purge")),
	), jetstreamPurgeStreamHandler(natsClient))

	s.AddTool(mcp.NewTool("jetstream_list_consumers",
		mcp.WithDescription("List all consumers for a JetStream stream"),
		mcp.WithString("stream", mcp.Required(), mcp.Description("The name of the stream")),
	), jetstreamListConsumersHandler(natsClient))

	s.AddTool(mcp.NewTool("jetstream_create_consumer",
		mcp.WithDescription("Create a new JetStream consumer"),
		mcp.WithString("stream", mcp.Required(), mcp.Description("The name of the stream")),
		mcp.WithString("name", mcp.Required(), mcp.Description("The name of the consumer")),
		mcp.WithString("filter_subject", mcp.Description("Filter subject (default '>')")),
		mcp.WithString("ack_policy", mcp.Description("Ack policy: 'explicit', 'all', 'none' (default 'explicit')")),
		mcp.WithString("ack_wait", mcp.Description("Ack wait duration (e.g., '30s', '1m') (default '30s')")),
		mcp.WithNumber("max_deliver", mcp.Description("Maximum delivery attempts (default 1)")),
	), jetstreamCreateConsumerHandler(natsClient))

	s.AddTool(mcp.NewTool("jetstream_delete_consumer",
		mcp.WithDescription("Delete a JetStream consumer"),
		mcp.WithString("stream", mcp.Required(), mcp.Description("The name of the stream")),
		mcp.WithString("consumer", mcp.Required(), mcp.Description("The name of the consumer")),
	), jetstreamDeleteConsumerHandler(natsClient))

	s.AddTool(mcp.NewTool("jetstream_fetch_message",
		mcp.WithDescription("Fetch a single message from a JetStream consumer. Returns an 'ack_token' for use with ack/nack tools."),
		mcp.WithString("stream", mcp.Required(), mcp.Description("The name of the stream")),
		mcp.WithString("consumer", mcp.Required(), mcp.Description("The name of the consumer")),
		mcp.WithNumber("timeout_ms", mcp.Description("Timeout in milliseconds (default 5000)")),
	), jetstreamFetchMessageHandler(natsClient))

	s.AddTool(mcp.NewTool("jetstream_ack_message",
		mcp.WithDescription("Acknowledge a message using an 'ack_token' returned by jetstream_fetch_message"),
		mcp.WithString("ack_token", mcp.Required(), mcp.Description("The reply_subject/ack_token from a fetched message")),
	), jetstreamAckMessageHandler(natsClient))

	s.AddTool(mcp.NewTool("jetstream_nack_message",
		mcp.WithDescription("Negatively acknowledge a message using an 'ack_token' returned by jetstream_fetch_message"),
		mcp.WithString("ack_token", mcp.Required(), mcp.Description("The reply_subject/ack_token from a fetched message")),
	), jetstreamNackMessageHandler(natsClient))
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

		storageStr := request.GetString("storage", "file")
		retentionStr := request.GetString("retention", "limits")

		config := jetstream.StreamConfig{
			Name:     name,
			Subjects: subjects,
		}

		switch strings.ToLower(storageStr) {
		case "memory":
			config.Storage = jetstream.MemoryStorage
		case "file":
			config.Storage = jetstream.FileStorage
		default:
			config.Storage = jetstream.FileStorage
		}

		switch strings.ToLower(retentionStr) {
		case "interest":
			config.Retention = jetstream.InterestPolicy
		case "workqueue":
			config.Retention = jetstream.WorkQueuePolicy
		case "limits":
			config.Retention = jetstream.LimitsPolicy
		default:
			config.Retention = jetstream.LimitsPolicy
		}

		_, err := natsClient.JS.CreateStream(ctx, config)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create stream %s: %v", name, err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully created stream: %s (Storage: %s, Retention: %s)",
			name, storageStr, retentionStr)), nil
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

func jetstreamListConsumersHandler(natsClient *nats.Client) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		streamName, _ := request.RequireString("stream")

		stream, err := natsClient.JS.Stream(ctx, streamName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get stream %s: %v", streamName, err)), nil
		}

		consumers := stream.ConsumerNames(ctx)
		var sb strings.Builder
		fmt.Fprintf(&sb, "Consumers for stream %s:\n", streamName)

		count := 0
		for name := range consumers.Name() {
			fmt.Fprintf(&sb, "- %s\n", name)
			count++
		}

		if count == 0 {
			return mcp.NewToolResultText(fmt.Sprintf("No consumers found for stream %s.", streamName)), nil
		}

		return mcp.NewToolResultText(sb.String()), nil
	}
}

func jetstreamCreateConsumerHandler(natsClient *nats.Client) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		streamName, _ := request.RequireString("stream")
		name, _ := request.RequireString("name")
		filterSubject := request.GetString("filter_subject", ">")
		ackPolicyStr := request.GetString("ack_policy", "explicit")
		ackWaitStr := request.GetString("ack_wait", "30s")
		maxDeliver := int(request.GetFloat("max_deliver", 1.0))

		ackWait, err := time.ParseDuration(ackWaitStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid ack_wait duration: %v", err)), nil
		}

		config := jetstream.ConsumerConfig{
			Durable:       name,
			FilterSubject: filterSubject,
			AckWait:       ackWait,
			MaxDeliver:    maxDeliver,
		}

		switch strings.ToLower(ackPolicyStr) {
		case "all":
			config.AckPolicy = jetstream.AckAllPolicy
		case "none":
			config.AckPolicy = jetstream.AckNonePolicy
		case "explicit":
			config.AckPolicy = jetstream.AckExplicitPolicy
		default:
			config.AckPolicy = jetstream.AckExplicitPolicy
		}

		stream, err := natsClient.JS.Stream(ctx, streamName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get stream %s: %v", streamName, err)), nil
		}

		_, err = stream.CreateOrUpdateConsumer(ctx, config)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create consumer %s: %v", name, err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully created consumer: %s on stream %s (AckPolicy: %s, AckWait: %s, MaxDeliver: %d)",
			name, streamName, ackPolicyStr, ackWaitStr, maxDeliver)), nil
	}
}

func jetstreamDeleteConsumerHandler(natsClient *nats.Client) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		streamName, _ := request.RequireString("stream")
		consumerName, _ := request.RequireString("consumer")

		stream, err := natsClient.JS.Stream(ctx, streamName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get stream %s: %v", streamName, err)), nil
		}

		if err := stream.DeleteConsumer(ctx, consumerName); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to delete consumer %s: %v", consumerName, err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully deleted consumer: %s from stream %s", consumerName, streamName)), nil
	}
}

func jetstreamFetchMessageHandler(natsClient *nats.Client) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		streamName, _ := request.RequireString("stream")
		consumerName, _ := request.RequireString("consumer")
		timeoutMs := int(request.GetFloat("timeout_ms", 5000.0))

		stream, err := natsClient.JS.Stream(ctx, streamName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get stream %s: %v", streamName, err)), nil
		}

		consumer, err := stream.Consumer(ctx, consumerName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get consumer %s: %v", consumerName, err)), nil
		}

		msgs, err := consumer.Fetch(1, jetstream.FetchMaxWait(time.Duration(timeoutMs)*time.Millisecond))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch message: %v", err)), nil
		}

		msg := <-msgs.Messages()
		if msg == nil {
			return mcp.NewToolResultText("No messages available."), nil
		}

		metadata, _ := msg.Metadata()

		return mcp.NewToolResultText(fmt.Sprintf("Fetched message:\nSequence: %d\nSubject: %s\nData: %s\nAck Token (Reply Subject): %s",
			metadata.Sequence.Stream, msg.Subject(), string(msg.Data()), msg.Reply())), nil
	}
}

func jetstreamAckMessageHandler(natsClient *nats.Client) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ackToken, _ := request.RequireString("ack_token")

		if err := natsClient.Conn.Publish(ackToken, []byte("+ACK")); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to publish ACK: %v", err)), nil
		}

		return mcp.NewToolResultText("Successfully acknowledged message."), nil
	}
}

func jetstreamNackMessageHandler(natsClient *nats.Client) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ackToken, _ := request.RequireString("ack_token")

		if err := natsClient.Conn.Publish(ackToken, []byte("-NAK")); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to publish NACK: %v", err)), nil
		}

		return mcp.NewToolResultText("Successfully negatively acknowledged message."), nil
	}
}

