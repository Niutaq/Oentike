package main

import (
	"github.com/nats-io/nats.go"
	"log"
)

func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer nc.Close()

	payload := []byte(`{"agent_id": "normal-agent", "status": "QUARANTINED", "extracted": {"ai_reasoning": "CBZC Auto-Isolation Protocol Triggered"}}`)
	nc.Publish("SECOPS.ai_processed", payload)
	nc.Flush()

	log.Println("Kill-Switch payload delivered to NATS")
}
