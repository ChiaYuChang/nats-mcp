package nats

import (
	"os"
	"strings"
	"testing"
)

func TestTokenAuth_Success(t *testing.T) {
	t.Setenv("NATS_TOKEN", "test-token")
	if os.Getenv("NATS_URL") == "" {
		t.Setenv("NATS_URL", "nats://localhost:14222")
	}

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
	t.Setenv("NATS_TOKEN", "wrong-token")
	if os.Getenv("NATS_URL") == "" {
		t.Setenv("NATS_URL", "nats://localhost:14222")
	}

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
	// Clear all possible auth env vars
	t.Setenv("NATS_TOKEN", "")
	t.Setenv("NATS_USER", "")
	t.Setenv("NATS_PASSWORD", "")
	t.Setenv("NATS_CREDS", "")

	if os.Getenv("NATS_URL") == "" {
		t.Setenv("NATS_URL", "nats://localhost:14222")
	}

	client, err := NewClient()
	if err == nil {
		client.Close()
		t.Fatal("Expected connection failure with no auth, but it succeeded")
	}
}
