package main

import (
	"context"
	"github.com/nats-io/nats.go"
	"log"
	"math/rand"
	"time"

	pb "oentike-control-plane/api/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
)

func main() {
	log.Println("[Sensor] Booting Edge Sensor...")
	ctx := context.Background()

	socketPath := "unix:///tmp/spire-sockets/workload_api.sock"
	source, err := workloadapi.NewX509Source(ctx, workloadapi.WithClientOptions(workloadapi.WithAddr(socketPath)))
	if err != nil {
		log.Fatalf("Unable to connect to SPIRE workload API: %v", err)
	}
	defer source.Close()

	// We trust the envoy/control plane identity
	serverID := spiffeid.RequireFromString("spiffe://example.org/fingate")
	creds := grpccredentials.MTLSClientCredentials(source, source, tlsconfig.AuthorizeID(serverID))

	// Bypass Envoy to directly hit Go Control Plane for precise mTLS execution test
	conn, err := grpc.DialContext(ctx, "localhost:50051", grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("Failed to connect to Envoy: %v", err)
	}
	defer conn.Close()

	client := pb.NewFingateServiceClient(conn)

	// Connect to NATS to receive attack commands from the UI
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	log.Println("[Sensor] Subscribed to SECOPS.attack_cmd for remote execution.")
	_, err = nc.Subscribe("SECOPS.attack_cmd", func(msg *nats.Msg) {
		cmd := string(msg.Data)
		log.Printf("[Sensor] Received C2 Attack Command: %s", cmd)

		if cmd == "WASM_SQLI" {
			log.Println("--- INITIATING WAF SIGNATURE ATTACK ---")
			sendTelemetry(client, "malicious-agent-waf", "' OR 1=1--", 45.0, 512)
		} else if cmd == "AI_DDOS" {
			log.Println("--- INITIATING AI VOLUMETRIC ANOMALY ---")
			sendTelemetry(client, "malicious-agent-ai", "grpc-go-client/1.0", 99.9, 8192)
		}
	})
	if err != nil {
		log.Fatalf("Failed to subscribe to attack commands: %v", err)
	}

	ticker := time.NewTicker(2 * time.Second) // Send healthy telemetry every 2 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sendTelemetry(client, "normal-agent", "grpc-go-client/1.0", 45.0+rand.Float32()*10, int32(512+rand.Intn(100)))
		}
	}
}

func sendTelemetry(client pb.FingateServiceClient, agentID, userAgent string, cpu float32, mem int32) {
	log.Printf("[Sensor] Starting Persistent Stream for %s (UA: %s)", agentID, userAgent)

	ctx := metadata.AppendToOutgoingContext(context.Background(), "x-threat-payload", userAgent)

	stream, err := client.StreamTelemetry(ctx)
	if err != nil {
		log.Printf("[Sensor] Error opening stream: %v", err)
		return
	}

	for {
		req := &pb.TelemetryRequest{
			AgentId:         agentID,
			CpuUsagePercent: cpu,
			MemoryUsedMb:    mem,
		}

		err = stream.Send(req)
		if err != nil {
			log.Fatalf("\n[x] FATAL: STREAM TERMINATED.\nReason: %v\n", err)
			return
		}
		log.Printf("[Sensor] ---> Pumping telemetry: CPU=%.1f, Mem=%d", cpu, mem)

		time.Sleep(2 * time.Second)
	}
}
