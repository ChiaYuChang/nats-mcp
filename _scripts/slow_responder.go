package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	natsgo "github.com/nats-io/nats.go"
	natsclient "github.com/nats-mcp/nats-mcp/internal/nats"
)

func main() {
	subject := flag.String("subject", "orders.slow_validate", "subject to respond on")
	delay := flag.Duration("delay", 2*time.Second, "delay before sending each reply")
	response := flag.String("response", `{"status":"ok"}`, "response payload")
	maxRequests := flag.Int("max-requests", 2, "number of requests to serve before exiting")
	flag.Parse()

	if *maxRequests <= 0 {
		log.Fatal("max-requests must be greater than 0")
	}

	client, err := natsclient.NewClient()
	if err != nil {
		log.Fatalf("connect NATS: %v", err)
	}
	defer client.Close()

	served := 0
	done := make(chan struct{})

	_, err = client.Conn.Subscribe(*subject, func(msg *natsgo.Msg) {
		if served >= *maxRequests {
			return
		}
		served++
		time.Sleep(*delay)
		if err := msg.Respond([]byte(*response)); err != nil {
			log.Printf("respond to request %d: %v", served, err)
			return
		}
		fmt.Printf("responded %d/%d on %s\n", served, *maxRequests, *subject)
		if served >= *maxRequests {
			close(done)
		}
	})
	if err != nil {
		log.Fatalf("subscribe %s: %v", *subject, err)
	}

	if err := client.Conn.Flush(); err != nil {
		log.Fatalf("flush subscription: %v", err)
	}
	if err := client.Conn.LastError(); err != nil {
		log.Fatalf("subscription completed with connection error: %v", err)
	}

	fmt.Printf("listening on %s with %v response delay\n", *subject, *delay)
	<-done
}
