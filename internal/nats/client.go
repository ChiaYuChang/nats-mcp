package nats

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Client wraps the NATS connection and provides helper methods
type Client struct {
	Conn *nats.Conn
	JS   jetstream.JetStream
}

// NewClient initializes a new NATS connection based on environment variables or defaults
func NewClient() (*Client, error) {
	url := os.Getenv("NATS_URL")
	if url == "" {
		url = "nats://localhost:14222"
	}

	opts := []nats.Option{
		nats.Name("NATS MCP Server"),
		nats.Timeout(5 * time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("NATS disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("NATS reconnected to %v", nc.ConnectedUrl())
		}),
	}

	// Support for credentials file or token if provided
	creds := os.Getenv("NATS_CREDS")
	token := os.Getenv("NATS_TOKEN")
	if creds != "" {
		opts = append(opts, nats.UserCredentials(creds))
	} else if token != "" {
		opts = append(opts, nats.Token(token))
	} else {
		// Fallback to username and password
		user := os.Getenv("NATS_USER")
		pass := os.Getenv("NATS_PASSWORD")
		if user != "" || pass != "" {
			opts = append(opts, nats.UserInfo(user, pass))
		}
	}

	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS at %s: %w", url, err)
	}

	// Initialize the new JetStream interface
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to initialize JetStream: %w", err)
	}

	return &Client{
		Conn: nc,
		JS:   js,
	}, nil
}

// Close closes the NATS connection
func (c *Client) Close() {
	if c.Conn != nil {
		c.Conn.Close()
	}
}
