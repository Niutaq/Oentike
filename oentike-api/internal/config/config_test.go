package config

import "testing"

func TestDiagnosticsOptIn(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OENTIKE_DEBUG_RPC", "")
	cfg := Load()
	if cfg.TraceEndpoint != "" || cfg.DebugRPC {
		t.Fatal("diagnostics must be opt-in")
	}
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317")
	t.Setenv("OENTIKE_DEBUG_RPC", "1")
	cfg = Load()
	if cfg.TraceEndpoint != "http://localhost:4317" || !cfg.DebugRPC {
		t.Fatal("explicit diagnostics ignored")
	}
}
