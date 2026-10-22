package main

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/nats-mcp/nats-mcp/internal/version"
)

func TestVersionWithoutNATS(t *testing.T) {
	// Invalid configuration must not affect offline version queries.
	t.Setenv("NATS_URL", "://invalid")
	t.Setenv("NATS_CREDS", "/nonexistent/nats.creds")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"--version"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if want := "nats-mcp " + version.Version + "\n"; stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestVersionOutputFailure(t *testing.T) {
	if err := run([]string{"--version"}, failingWriter{}, io.Discard); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("error = %v, want closed pipe", err)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, io.ErrClosedPipe
}
