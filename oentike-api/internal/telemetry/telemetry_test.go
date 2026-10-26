package telemetry

import (
	"context"
	"testing"
)

func TestNoCollectorByDefault(t *testing.T) {
	provider, err := InitProvider(context.Background(), "", "test")
	if err != nil || provider != nil {
		t.Fatalf("default must not export: %v %v", provider, err)
	}
}

func TestInvalidEndpoint(t *testing.T) {
	if _, err := InitProvider(context.Background(), "ftp://localhost:4317", "test"); err == nil {
		t.Fatal("invalid scheme accepted")
	}
}
