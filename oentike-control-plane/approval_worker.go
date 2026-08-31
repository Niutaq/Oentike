package main

import (
	// Standard libraries
	"context"
	"encoding/json"
	"log"

	// Third-party libraries
	"github.com/nats-io/nats.go/jetstream"
)

type AIProcessedEvent struct {
	AgentID      string `json:"agent_id"`
	Status       string `json:"status"`
	OriginalData struct {
		ChatMessage string `json:"chat_message"`
	} `json:"original_data"`
	Extracted struct {
		AIReasoning string `json:"ai_reasoning"`
	} `json:"extracted"`
}

// StartApprovalWorker starts a background worker that consumes SecOps events from JetStream
func StartApprovalWorker(ctx context.Context, js jetstream.JetStream, srv *server) {
	// Create a consumer
	consumer, err := js.CreateOrUpdateConsumer(ctx, "SECOPS_EVENTS", jetstream.ConsumerConfig{
		Durable:       "APPROVAL_WORKER",
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: "SECOPS.ai_processed",
	})
	if err != nil {
		log.Fatalf("Failed to create consumer for approval worker: %v", err)
	}

	iter, err := consumer.Messages(jetstream.PullMaxMessages(10))
	if err != nil {
		log.Fatalf("Failed to create message iterator: %v", err)
	}

	log.Println("[Approval Worker] Started consuming SECOPS.ai_processed events from stream.")

	go func() {
		for {
			select {
			case <-ctx.Done():
				iter.Stop()
				return
			default:
				msg, err := iter.Next()
				if err != nil {
					// Timeout or other error
					continue
				}

				processMessage(msg, srv)
			}
		}
	}()
}

func processMessage(msg jetstream.Msg, srv *server) {
	log.Printf("[Approval Worker] Received AI processed message: %s", string(msg.Data()))

	var event AIProcessedEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		log.Printf("[Approval Worker] Error unmarshaling event: %v", err)
		msg.Nak()
		return
	}

	decision := &BudgetDecision{
		AgentID:     event.AgentID,
		ChatMessage: event.OriginalData.ChatMessage,
		Status:      event.Status,
		DecisionAI:  event.Extracted.AIReasoning,
	}

	if err := SaveBudgetDecision(decision); err != nil {
		log.Printf("[Approval Worker] Failed to save decision: %v", err)
	} else {
		log.Printf("[Approval Worker] Decision saved to database for agent %s", event.AgentID)
	}

	if event.Status == "QUARANTINED" || event.Status == "THREAT_DETECTED" {
		log.Printf("[SecOps] AI decided to quarantine agent: %s. Applying Ban...", event.AgentID)
		if srv != nil {
			srv.BanAgent(event.AgentID)
		}
	}

	msg.Ack()
}
