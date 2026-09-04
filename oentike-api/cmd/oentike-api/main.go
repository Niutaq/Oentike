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

	"oentike-api/internal/conditions"
	conditionsv1 "oentike-api/internal/conditionsv1"
	"oentike-api/internal/config"
	"oentike-api/internal/database"
	"oentike-api/internal/httpapi"
	"oentike-api/internal/ingest"
	"oentike-api/migrations"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: oentike-api <migrate|serve|ingest> [cell_id]")
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
		cellID := ingest.DefaultCellID
		if len(args) > 1 {
			cellID = args[1]
		}
		return ingestWeather(cfg, cellID)
	default:
		return fmt.Errorf("unknown command %q; expected migrate, serve, or ingest", args[0])
	}
}

func ingestWeather(cfg config.Config, cellID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	result, err := ingest.Run(ctx, db, ingest.DefaultHTTPClient(), cfg.OpenMeteoURL, cellID, time.Now())
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

	db, err := database.Open(startupCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	store := conditions.NewStore(db)

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.New(db),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	grpcListener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen gRPC %s: %w", cfg.GRPCAddr, err)
	}
	grpcServer := grpc.NewServer()
	conditionsv1.RegisterConditionsServiceServer(grpcServer, conditions.NewServer(store, store))
	reflection.Register(grpcServer)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go ingest.Loop(
		ctx,
		db,
		ingest.DefaultHTTPClient(),
		cfg.OpenMeteoURL,
		ingest.DefaultCellID,
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
