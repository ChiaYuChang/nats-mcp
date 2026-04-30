package testutil

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/nats"
)

// NATSContainer represents the NATS container for testing
type NATSContainer struct {
	*nats.NATSContainer
	URL string
}

// SetupNATS starts a NATS container for testing.
// If token is not empty, it enables token authentication.
func SetupNATS(ctx context.Context, token string) (*NATSContainer, error) {
	opts := []testcontainers.ContainerCustomizer{
		testcontainers.WithImage("nats:2-alpine"),
	}

	cmd := []string{"-js"}
	if token != "" {
		cmd = append(cmd, "--auth", token)
	}
	opts = append(opts, testcontainers.WithCmd(cmd...))

	natsContainer, err := nats.Run(ctx, "nats:2-alpine", opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to start NATS container: %w", err)
	}

	url, err := natsContainer.ConnectionString(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	return &NATSContainer{
		NATSContainer: natsContainer,
		URL:           url,
	}, nil
}

// Terminate stops the NATS container
func (n *NATSContainer) Terminate(ctx context.Context) error {
	// Give some time for cleanup
	time.Sleep(100 * time.Millisecond)
	return n.NATSContainer.Terminate(ctx)
}
