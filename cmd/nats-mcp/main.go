package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/nats-mcp/nats-mcp/internal/nats"
	"github.com/nats-mcp/nats-mcp/internal/server"
	"github.com/nats-mcp/nats-mcp/internal/version"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("nats-mcp", flag.ContinueOnError)
	flags.SetOutput(stderr)
	showVersion := flags.Bool("version", false, "print version and exit")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected argument: %s", flags.Arg(0))
	}
	if *showVersion {
		_, err := fmt.Fprintf(stdout, "nats-mcp %s\n", version.Version)
		return err
	}

	// 1. Initialize NATS Client
	natsClient, err := nats.NewClient()
	if err != nil {
		return fmt.Errorf("could not connect to NATS: %w", err)
	}
	defer natsClient.Close()

	// 2. Initialize MCP Server
	natsMCP := server.NewNatsServer(natsClient)

	// 3. Register Tools
	natsMCP.RegisterTools()

	// 4. Start Serving via Stdio
	log.Printf("Starting NATS MCP Server...")
	if err := natsMCP.Serve(); err != nil {
		return fmt.Errorf("serve MCP: %w", err)
	}
	return nil
}
