package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/proto"

	"oentike-api/internal/bdl"
	"oentike-api/internal/conditions"
	conditionsv1 "oentike-api/internal/conditionsv1"
	"oentike-api/internal/config"
	"oentike-api/internal/database"
	"oentike-api/internal/httpapi"
	"oentike-api/internal/ingest"
	"oentike-api/internal/telemetry"
	"oentike-api/migrations"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: oentike-api <migrate|serve|ingest|sync-bdl> [cell_id]")
	}

	cfg := config.Load()
	switch args[0] {
	case "migrate":
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		return migrations.Up(ctx, cfg.DatabaseURL)
	case "serve":
		return serve(cfg)
	case "ingest":
		cellID := ""
		if len(args) > 1 {
			cellID = args[1]
		}
		return ingestWeather(cfg, cellID)
	case "sync-bdl":
		return syncBDL(cfg)
	default:
		return fmt.Errorf("unknown command %q; expected migrate, serve, ingest, or sync-bdl", args[0])
	}
}

func syncBDL(cfg config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	n, err := bdl.SyncNadlesnictwa(ctx, db, bdl.DefaultHTTPClient(), cfg.BdlWFSURL)
	if err != nil {
		return err
	}
	log.Printf("synced %d BDL nadleśnictw into forest_units", n)
	return nil
}

func ingestWeather(cfg config.Config, cellID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	client := ingest.DefaultHTTPClient()
	if cellID == "" || cellID == "all" {
		return ingest.RunAll(ctx, db, client, cfg.OpenMeteoURL, time.Now())
	}

	result, err := ingest.Run(ctx, db, client, cfg.OpenMeteoURL, cellID, time.Now())
	if err != nil {
		return err
	}

	log.Printf(
		"ingested %s hours=%d run_id=%d sha256=%s lat=%.6f lon=%.6f",
		result.CellID, result.Hours, result.RunID, result.SHA256, result.Latitude, result.Longitude,
	)
	return nil
}

func serve(cfg config.Config) error {
	startupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tp, err := telemetry.InitProvider(startupCtx, "127.0.0.1:4317", "oentike-api")
	if err != nil {
		log.Printf("Failed to initialize telemetry: %v", err)
	} else {
		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			if err := tp.Shutdown(shutdownCtx); err != nil {
				log.Printf("Error shutting down tracer provider: %v", err)
			}
		}()
	}

	db, err := database.Open(startupCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	store := conditions.NewStore(db)
	weather := &ingest.CellRefresher{
		DB:      db,
		Client:  ingest.DefaultHTTPClient(),
		BaseURL: cfg.OpenMeteoURL,
	}

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.New(db),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      45 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	grpcListener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen gRPC %s: %w", cfg.GRPCAddr, err)
	}
	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.UnaryInterceptor(networkInterceptor),
	)
	conditionsv1.RegisterConditionsServiceServer(grpcServer, conditions.NewServer(store, store, weather))
	reflection.Register(grpcServer)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go ingest.Loop(
		ctx,
		db,
		ingest.DefaultHTTPClient(),
		cfg.OpenMeteoURL,
		ingest.DefaultInterval,
		time.Now,
	)

	errCh := make(chan error, 2)
	go func() {
		log.Printf("Oentike HTTP health on %s", cfg.HTTPAddr)
		errCh <- httpServer.ListenAndServe()
	}()
	go func() {
		log.Printf("Oentike gRPC conditions on %s", cfg.GRPCAddr)
		errCh <- grpcServer.Serve(grpcListener)
	}()

	select {
	case err := <-errCh:
		stop()
		shutdownServers(httpServer, grpcServer)
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownServers(httpServer, grpcServer)
		return nil
	}
}

func shutdownServers(httpServer *http.Server, grpcServer *grpc.Server) {
	grpcServer.GracefulStop()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
}

func networkInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()

	reqSize := 0
	if p, ok := req.(proto.Message); ok {
		reqSize = proto.Size(p)
	}

	res, err := handler(ctx, req)
	duration := time.Since(start)

	resSize := 0
	if p, ok := res.(proto.Message); ok {
		resSize = proto.Size(p)
	}

	fmt.Printf("\n--- [NETWORK | LAYER 7] NEW gRPC PACKET ---\n")
	fmt.Printf("Method: %s\n", info.FullMethod)
	fmt.Printf("Latency: %v\n", duration)
	fmt.Printf("Payload Size (Protobuf): Req: %d bytes | Res: %d bytes\n", reqSize, resSize)

	if err != nil {
		fmt.Printf("Status: ERROR (%v)\n", err)
	} else {
		fmt.Printf("Status: SUCCESS (HTTP/2)\n")
	}
	fmt.Printf("-------------------------------------------\n\n")

	return res, err
}
