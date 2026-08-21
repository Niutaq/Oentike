package main

import (
	// Standard library imports
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"time"

	// Local imports
	pb "oentike-control-plane/api/v1"

	// Third-party imports
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/grpc"

	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
)

// Structs
// i18n - internationalization function
func i18n(key string) string {
	dict := map[string]string{
		"log.grpc.received":         "Received workload request from agent",
		"log.grpc.workload":         "Workload",
		"log.nats.success":          "Published FinOps event to the queue",
		"log.nats.error":            "Failed to publish event",
		"log.json.error":            "Serialization error",
		"log.nats.stream_active":    "Stream FINOPS_EVENTS (FOCUS) active.",
		"log.grpc.listening":        "ZT-FinGate Control Plane is listening on port 50051...",
		"api.status.pending_ai":     "PENDING_AI",
		"api.reasoning.nats_queued": "The request has been forwarded for LLM analysis via NATS JetStream queue.",
	}
	if val, ok := dict[key]; ok {
		return val
	}
	return key
}

// server is the gRPC server implementation for the Fingate service.
type server struct {
	pb.UnimplementedFingateServiceServer
	js jetstream.JetStream
}

// FinOpsEvent represents a FinOps event to be published on the NATS queue.
type FinOpsEvent struct {
	AgentID            string    `json:"agent_id"`
	Timestamp          time.Time `json:"timestamp"`
	Action             string    `json:"action"`
	WorkloadType       string    `json:"workload_type"`
	EstimatedCpuCycles int32     `json:"estimated_cpu_cycles"`
	EstimatedMemoryMb  int32     `json:"estimated_memory_mb"`
	ClientMtlsSetupMs  int64     `json:"client_mtls_setup_ms"`
	PayloadBytesOut    int32     `json:"payload_bytes_out"`
	ClientRegion       string    `json:"client_region"`
	ServerProcTimeUs   int64     `json:"server_proc_time_us"`
}



// StreamTelemetry handles the gRPC request for receiving continuous telemetry stream.
func (s *server) StreamTelemetry(stream pb.FingateService_StreamTelemetryServer) error {
	log.Println("[gRPC] Started receiving telemetry stream")
	provisioned := false

	for {
		start := time.Now()
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.TelemetryResponse{Status: "Stream ended"})
		}
		if err != nil {
			log.Printf("Error receiving stream: %v", err)
			return err
		}

		cpu := req.GetCpuUsagePercent()
		mem := req.GetMemoryUsedMb()
		agentID := req.GetAgentId()

		log.Printf("[Telemetry] Agent: %s, CPU: %.2f%%, Mem: %d MB", agentID, cpu, mem)

		// Publish to NATS for AI orchestrator
		var mtlsSetupMs int64
		var payloadBytes int32
		var region string

		if req.Metrics != nil {
			mtlsSetupMs = req.Metrics.MtlsSetupMs
			payloadBytes = req.Metrics.PayloadBytes
			region = req.Metrics.Region
		}

		serverProcTime := time.Since(start).Microseconds()

		event := FinOpsEvent{
			AgentID:            agentID,
			Timestamp:          time.Now(),
			Action:             "TelemetryReceived",
			WorkloadType:       "Sensor Stream",
			EstimatedCpuCycles: int32(cpu * 100), // simplistic mapping
			EstimatedMemoryMb:  mem,
			ClientMtlsSetupMs:  mtlsSetupMs,
			PayloadBytesOut:    payloadBytes,
			ClientRegion:       region,
			ServerProcTimeUs:   serverProcTime,
		}

		eventData, _ := json.Marshal(event)
		s.js.Publish(stream.Context(), "FINOPS.metrics", eventData)

		if !provisioned && cpu < 80.0 && mem < 1024 {
			log.Printf("[FinOps Budget] Metrics within FOCUS budget. Initiating segment provisioning...")
			provisioned = true

			segmentName := fmt.Sprintf("segment-%d", time.Now().Unix())
			
			cmdStr := fmt.Sprintf("sudo docker exec zt-spire-server bin/spire-server entry create -parentID spiffe://example.org/my-node -spiffeID spiffe://example.org/%s -selector unix:uid:1000", segmentName)
			cmd := exec.Command("sh", "-c", cmdStr)
			out, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("[SPIRE] Failed to generate identity: %v, output: %s", err, string(out))
			} else {
				log.Printf("[SPIRE] Identity provisioned: spiffe://example.org/%s", segmentName)

				yamlContent := fmt.Sprintf(`
# Envoy / Kubernetes NetworkPolicy YAML
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: finops-policy-%s
  namespace: secure-segments
spec:
  podSelector:
    matchLabels:
      spiffe.io/spiffe-id: "true"
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: allowed-namespace
    ports:
    - protocol: TCP
      port: 443
`, segmentName)
				log.Println(yamlContent)
			}
		}
	}
}

// main is the entry point of the application
func main() {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Fatal: Cannot connect to NATS JetStream on %s (is it running?): %v", natsURL, err)
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
	log.Printf("[NATS] %s", i18n("log.nats.stream_active"))

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "finops.db"
	}
	InitDB(dbPath)

	StartApprovalWorker(context.Background(), js)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Could not listen on port 50051: %v", err)
	}

	// --- Zero-Trust mTLS Setup (SPIFFE/SPIRE) ---
	socketPath := "unix:///tmp/spire-sockets/workload_api.sock"
	source, err := workloadapi.NewX509Source(context.Background(), workloadapi.WithClientOptions(workloadapi.WithAddr(socketPath)))
	if err != nil {
		log.Fatalf("Unable to create X509Source from SPIRE (is agent running?): %v", err)
	}
	defer source.Close()

	// The server only trusts clients that possess a certificate for the fingate identity
	clientID := spiffeid.RequireFromString("spiffe://example.org/fingate")
	creds := grpccredentials.MTLSServerCredentials(source, source, tlsconfig.AuthorizeID(clientID))

	// Register the gRPC server with strict mTLS credentials
	s := grpc.NewServer(grpc.Creds(creds))
	pb.RegisterFingateServiceServer(s, &server{js: js})

	fmt.Printf("[gRPC/mTLS] Secured %s\n", i18n("log.grpc.listening"))
	if err := s.Serve(lis); err != nil {
		log.Fatalf("gRPC server error: %v", err)
	}
}
