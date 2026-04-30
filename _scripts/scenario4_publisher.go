package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	natsclient "github.com/nats-mcp/nats-mcp/internal/nats"
)

func main() {
	subject := flag.String("subject", "events.user.login", "subject to publish to")
	count := flag.Int("count", 2, "number of messages to publish")
	initialDelay := flag.Duration("initial-delay", 0, "delay before publishing the first message")
	delay := flag.Duration("delay", 500*time.Millisecond, "delay between published messages")
	payloadPrefix := flag.String("payload-prefix", "scenario4", "prefix for generated message payloads")
	flag.Parse()

	if *count <= 0 {
		log.Fatal("count must be greater than 0")
	}

	client, err := natsclient.NewClient()
	if err != nil {
		log.Fatalf("connect NATS: %v", err)
	}
	defer client.Close()

	if *initialDelay > 0 {
		time.Sleep(*initialDelay)
	}

	for i := 1; i <= *count; i++ {
		payload := fmt.Sprintf(
			`{"source":"scenario4_publisher","sequence":%d,"label":"%s-%d","published_at":"%s"}`,
			i,
			*payloadPrefix,
			i,
			time.Now().UTC().Format(time.RFC3339Nano),
		)

		if err := client.Conn.Publish(*subject, []byte(payload)); err != nil {
			log.Fatalf("publish message %d: %v", i, err)
		}

		fmt.Printf("published %d/%d to %s\n", i, *count, *subject)

		if i < *count {
			time.Sleep(*delay)
		}
	}

	if err := client.Conn.Flush(); err != nil {
		log.Fatalf("flush publish buffer: %v", err)
	}
	if err := client.Conn.LastError(); err != nil {
		log.Fatalf("publish completed with connection error: %v", err)
	}
}
