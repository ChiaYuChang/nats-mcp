package nats

import (
	"context"
	"strings"
	"testing"

	"github.com/nats-mcp/nats-mcp/internal/testutil"
)

func TestTokenAuth_Success(t *testing.T) {
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

	client, err := NewClient()
	if err != nil {
		t.Fatalf("Expected successful connection with valid token, got error: %v", err)
	}
	defer client.Close()

	if !client.Conn.IsConnected() {
		t.Fatal("NATS client is not connected")
	}

	if client.JS == nil {
		t.Fatal("JetStream context is nil")
	}
}

func TestTokenAuth_Failure(t *testing.T) {
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
	t.Setenv("NATS_TOKEN", "wrong-token")

	client, err := NewClient()
	if err == nil {
		client.Close()
		t.Fatal("Expected connection failure with invalid token, but it succeeded")
	}

	if !strings.Contains(err.Error(), "Authorization Violation") {
		t.Logf("Note: Connection failed as expected, but error might be different: %v", err)
	}
}

func TestNoAuth_Failure(t *testing.T) {
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
	// Clear all possible auth env vars
	t.Setenv("NATS_TOKEN", "")
	t.Setenv("NATS_USER", "")
	t.Setenv("NATS_PASSWORD", "")
	t.Setenv("NATS_CREDS", "")

	client, err := NewClient()
	if err == nil {
		client.Close()
		t.Fatal("Expected connection failure with no auth, but it succeeded")
	}
}
