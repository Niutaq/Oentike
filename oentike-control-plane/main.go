package main

import (
	// Standard library imports
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	// Local imports
	pb "oentike-control-plane/api/v1"

	// Third-party imports
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// Structs
// i18n - internationalization function
func i18n(key string) string {
	dict := map[string]string{
		"log.grpc.received":         "Received workload request from agent",
		"log.grpc.workload":         "Workload",
		"log.nats.success":          "Published SecOps event to the queue",
		"log.nats.error":            "Failed to publish event",
		"log.json.error":            "Serialization error",
		"log.nats.stream_active":    "Stream SECOPS_EVENTS active.",
		"log.grpc.listening":        "Oentike Control Plane is listening on port 50051...",
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
	js            jetstream.JetStream
	mu            sync.RWMutex
	activeStreams map[string][]context.CancelFunc
	bannedAgents  map[string]bool
}

// BanAgent forcefully adds an agent to the blacklist and cancels all its active streams.
func (s *server) BanAgent(agentID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.bannedAgents[agentID] = true
	log.Printf("[Context Watcher] Agent %s banned. Terminating active streams: %d", agentID, len(s.activeStreams[agentID]))

	for _, cancel := range s.activeStreams[agentID] {
		cancel()
	}
	delete(s.activeStreams, agentID)
}

// SecurityAuditEvent represents a threat intelligence event to be published on the NATS queue.
type SecurityAuditEvent struct {
	AgentID           string    `json:"agent_id"`
	Timestamp         time.Time `json:"timestamp"`
	Action            string    `json:"action"`
	WorkloadType      string    `json:"workload_type"`
	RiskScore         int32     `json:"risk_score"`
	AnomalyDetected   bool      `json:"anomaly_detected"`
	ClientMtlsSetupMs int64     `json:"client_mtls_setup_ms"`
	PayloadBytesOut   int32     `json:"payload_bytes_out"`
	ClientRegion      string    `json:"client_region"`
	ServerProcTimeUs  int64     `json:"server_proc_time_us"`
}

// StreamTelemetry handles the gRPC request for receiving continuous telemetry stream.
func (s *server) StreamTelemetry(stream pb.FingateService_StreamTelemetryServer) error {
	log.Println("[gRPC] Started receiving telemetry stream")

	// We read the first message to identify the agent
	req, err := stream.Recv()
	if err != nil {
		return err
	}
	agentID := req.GetAgentId()

	// 1. Edge-like Validation (ExtAuthz equivalent inside the app)
	s.mu.RLock()
	isBanned := s.bannedAgents[agentID]
	s.mu.RUnlock()
	if isBanned {
		log.Printf("[Edge] Rejected connection from quarantined agent: %s", agentID)
		return status.Errorf(codes.PermissionDenied, "Forbidden: Agent %s is quarantined", agentID)
	}

	// 2. Register Active Stream for Context Cancellation
	ctx, cancel := context.WithCancel(stream.Context())
	s.mu.Lock()
	s.activeStreams[agentID] = append(s.activeStreams[agentID], cancel)
	s.mu.Unlock()

	defer func() {
		cancel() // cleanup
	}()

	errCh := make(chan error, 1)

	// 3. Worker Goroutine for continuous streaming
	go func() {
		provisioned := false
		// Process the first message that we already received
		s.processTelemetryFrame(stream.Context(), req, startProc(req), &provisioned)

		for {
			req, err := stream.Recv()
			if err == io.EOF {
				stream.SendAndClose(&pb.TelemetryResponse{Status: "Stream ended"})
				errCh <- nil
				return
			}
			if err != nil {
				errCh <- err
				return
			}
			s.processTelemetryFrame(stream.Context(), req, startProc(req), &provisioned)
		}
	}()

	// 4. The Watcher (Select Block)
	select {
	case <-ctx.Done():
		// Stream was killed asynchronously by BanAgent()
		log.Printf("[Context Watcher] Forcibly aborted stream for agent: %s", agentID)
		return status.Errorf(codes.PermissionDenied, "Connection forcefully terminated by SecOps Auto-Remediation")
	case err := <-errCh:
		// Normal exit or network error
		return err
	}
}

func startProc(req *pb.TelemetryRequest) time.Time {
	return time.Now()
}

func (s *server) processTelemetryFrame(ctx context.Context, req *pb.TelemetryRequest, start time.Time, provisioned *bool) {
	cpu := req.GetCpuUsagePercent()
	mem := req.GetMemoryUsedMb()
	agentID := req.GetAgentId()

	log.Printf("[Telemetry] Agent: %s, CPU: %.2f%%, Mem: %d MB", agentID, cpu, mem)

	var mtlsSetupMs int64
	var payloadBytes int32
	var region string

	if req.Metrics != nil {
		mtlsSetupMs = req.Metrics.MtlsSetupMs
		payloadBytes = req.Metrics.PayloadBytes
		region = req.Metrics.Region
	}

	serverProcTime := time.Since(start).Microseconds()
	ctx, span := otel.Tracer("oentike-control-plane").Start(ctx, "ProcessTelemetryFrame")
	defer span.End()

	span.SetAttributes(
		attribute.String("agent.id", agentID),
		attribute.Float64("agent.cpu", float64(cpu)),
		attribute.Int64("agent.mem", int64(mem)),
	)

	riskScore := int32(0)
	anomaly := false
	if cpu > 90.0 || mem > 2048 {
		riskScore = 85
		anomaly = true
		span.AddEvent("Anomaly detected based on resource usage")
	}
	span.SetAttributes(attribute.Int("risk.score", int(riskScore)))

	event := SecurityAuditEvent{
		AgentID:           agentID,
		Timestamp:         time.Now(),
		Action:            "TelemetryReceived",
		WorkloadType:      "Sensor Stream",
		RiskScore:         riskScore,
		AnomalyDetected:   anomaly,
		ClientMtlsSetupMs: mtlsSetupMs,
		PayloadBytesOut:   payloadBytes,
		ClientRegion:      region,
		ServerProcTimeUs:  serverProcTime,
	}

	eventData, _ := json.Marshal(event)
	s.js.Publish(ctx, "SECOPS.metrics", eventData)
}

// main is the entry point of the application
func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tp, err := InitTracer(ctx)
	if err != nil {
		log.Printf("Warning: Failed to initialize OpenTelemetry: %v", err)
	} else {
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				log.Printf("Error shutting down tracer provider: %v", err)
			}
		}()
		log.Println("[OTel] OpenTelemetry Tracer initialized successfully")
	}

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

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	streamConfig := jetstream.StreamConfig{
		Name:     "SECOPS_EVENTS",
		Subjects: []string{"SECOPS.>"},
		Storage:  jetstream.FileStorage,
	}

	_, err = js.CreateOrUpdateStream(ctx, streamConfig)
	if err != nil {
		log.Fatalf("Fatal: Cannot create JetStream SECOPS_EVENTS stream: %v", err)
	}
	log.Printf("[NATS] %s", i18n("log.nats.stream_active"))

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "secops.db"
	}
	InitDB(dbPath)

	s := &server{
		js:            js,
		activeStreams: make(map[string][]context.CancelFunc),
		bannedAgents:  make(map[string]bool),
	}

	StartApprovalWorker(context.Background(), js, s)

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

	// Register the gRPC server with strict mTLS credentials and OpenTelemetry instrumentation
	statsHandler := otelgrpc.NewServerHandler()
	grpcServer := grpc.NewServer(
		grpc.Creds(creds),
		grpc.StatsHandler(statsHandler),
	)
	pb.RegisterFingateServiceServer(grpcServer, s)

	// Register Envoy ExtAuthz Service
	authv3.RegisterAuthorizationServer(grpcServer, &ExtAuthzServer{controlPlane: s})

	fmt.Printf("[gRPC/mTLS] Secured %s\n", i18n("log.grpc.listening"))

	// --- REST Telemetry Endpoint (WASM-to-NATS Bridge) ---
	go func() {
		http.HandleFunc("/telemetry", func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}

			// Publish directly to NATS JetStream
			s.js.Publish(context.Background(), "SECOPS.ai_processed", body)

			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "OK")
		})
		log.Println("[HTTP] Listening for WASM Callouts on :8080...")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC server error: %v", err)
	}
}
