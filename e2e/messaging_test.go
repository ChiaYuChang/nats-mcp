package e2e

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

func checkMessaging(ctx context.Context, stdin io.Writer, scanner *bufio.Scanner, url, token string) error {
	conn, err := nats.Connect(url, nats.Token(token), nats.Timeout(5*time.Second))
	if err != nil {
		return fmt.Errorf("connect message observer: %w", err)
	}
	defer conn.Close()

	subject := nats.NewInbox()
	payload := "e2e-payload:" + nats.NewInbox() + ":中文"
	sub, err := conn.SubscribeSync(subject)
	if err != nil {
		return err
	}
	defer func() { _ = sub.Unsubscribe() }()
	_, err = conn.Subscribe(subject+".request", func(msg *nats.Msg) {
		_ = msg.Respond(msg.Data)
	})
	if err != nil {
		return err
	}
	// Confirm both subscriptions exist before the MCP process publishes.
	if err := conn.FlushTimeout(5 * time.Second); err != nil {
		return err
	}
	response, err := callTool(stdin, scanner, 10, "nats_publish", map[string]any{
		"subject": subject, "payload": payload,
	})
	if err != nil {
		return err
	}
	if response.Result.IsError {
		return fmt.Errorf("nats_publish failed: %+v", response.Result.Content)
	}
	readCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	msg, err := sub.NextMsgWithContext(readCtx)
	if err != nil {
		return fmt.Errorf("receive MCP publication: %w", err)
	}
	if msg.Subject != subject || string(msg.Data) != payload {
		return fmt.Errorf("publication mismatch: subject=%q payload=%q", msg.Subject, msg.Data)
	}
	response, err = callTool(stdin, scanner, 11, "nats_request", map[string]any{
		"subject": subject + ".request", "payload": payload, "timeout_ms": 5000,
	})
	if err != nil {
		return err
	}
	if response.Result.IsError {
		return fmt.Errorf("nats_request failed: %+v", response.Result.Content)
	}
	for _, content := range response.Result.Content {
		if content.Type == "text" && content.Text == "Received response from "+subject+".request:\n"+payload {
			return nil
		}
	}
	return errors.New("nats_request did not return the exact echoed payload")
}

func confirmsPing(response rpcResponse) bool {
	if response.Result.IsError {
		return false
	}
	for _, content := range response.Result.Content {
		if content.Type == "text" && strings.Contains(content.Text, "NATS Server is reachable") {
			return true
		}
	}
	return false
}
