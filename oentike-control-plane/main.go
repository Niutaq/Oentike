package main

import (
	// Standard library imports
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	// Local imports
	pb "oentike-control-plane/api/v1"

	// Third-party imports
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/grpc"
)

// Structs
// server is the gRPC server implementation for the Fingate service.
type server struct {
	pb.UnimplementedFingateServiceServer
	js jetstream.JetStream
}

// FinOpsEvent represents a FinOps event to be published on the NATS queue.
type FinOpsEvent struct {
	AgentID   string    `json:"agent_id"`
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
}

func (s *server) GetUsage(ctx context.Context, req *pb.UsageRequest) (*pb.UsageResponse, error) {
	agentID := req.GetAgentId()
	log.Printf("[gRPC] Got UsageRequest from agent: %s", agentID)

	event := FinOpsEvent{
		AgentID:   agentID,
		Timestamp: time.Now(),
		Action:    "UsageRequestReceived",
	}

	eventData, err := json.Marshal(event)
	if err == nil {
		_, err = s.js.Publish(ctx, "FINOPS.metrics", eventData)
		if err != nil {
			log.Printf("[NATS Error] Cannot publish event: %v", err)
		} else {
			log.Printf("[NATS] Published FinOps event for agent: %s", agentID)
		}
	} else {
		log.Printf("[JSON Error] Error serializing event: %v", err)
	}

	return &pb.UsageResponse{
		CPU:    "Registered (NATS Event Emitted)",
		Memory: "Registered (NATS Event Emitted)",
	}, nil
}

// main is the entry point of the application.
func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("Fatal: Cannot connect to NATS JetStream (is it running?): %v", err)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatalf("Fatal: Cannot initialize JetStream: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	streamConfig := jetstream.StreamConfig{
		Name:     "FINOPS_EVENTS",
		Subjects: []string{"FINOPS.>"},
		Storage:  jetstream.FileStorage,
	}

	_, err = js.CreateOrUpdateStream(ctx, streamConfig)
	if err != nil {
		log.Fatalf("Fatal: Cannot create JetStream FINOPS_EVENTS stream: %v", err)
	}
	log.Println("[NATS] Stream FINOPS_EVENTS (FOCUS) is active.")

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Could not listen on port 50051: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterFingateServiceServer(s, &server{js: js})

	fmt.Println("[gRPC] ZT-FinGate Control Plane is running on port 50051.")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("gRPC server error: %v", err)
	}
}
