package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestForecastURL(t *testing.T) {
	got, err := ForecastURL("https://api.open-meteo.com", 50.60125, 22.189584)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"latitude=50.601250",
		"longitude=22.189584",
		"hourly=precipitation%2Csoil_temperature_6cm%2Csoil_moisture_3_to_9cm",
		"timezone=Europe%2FWarsaw",
		"past_days=14",
		"forecast_days=2",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("url %s missing %s", got, want)
		}
	}
}

func TestParseForecastHashesBodyAndSplitsPastForecast(t *testing.T) {
	warsaw, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{
		"latitude": 50.60125,
		"longitude": 22.189584,
		"elevation": 172,
		"hourly": {
			"time": ["2026-09-03T10:00", "2026-09-03T14:00"],
			"precipitation": [0.2, 1.5],
			"soil_temperature_6cm": [15.4, 16.1],
			"soil_moisture_3_to_9cm": [0.096, 0.101]
		}
	}`)
	fetchedAt := time.Date(2026, 9, 3, 12, 0, 0, 0, warsaw)
	got, err := ParseForecast(body, "https://example.test/forecast", fetchedAt, warsaw)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	if got.SHA256 != hex.EncodeToString(sum[:]) {
		t.Fatalf("sha256 %s", got.SHA256)
	}
	if got.ElevationM != 172 {
		t.Fatalf("elevation %v", got.ElevationM)
	}
	if len(got.Samples) != 2 {
		t.Fatalf("samples %d", len(got.Samples))
	}
	if got.Samples[0].DataKind != "past" {
		t.Fatalf("first kind %s", got.Samples[0].DataKind)
	}
	if got.Samples[1].DataKind != "forecast" {
		t.Fatalf("second kind %s", got.Samples[1].DataKind)
	}
	if got.Samples[0].PrecipitationMM == nil || *got.Samples[0].PrecipitationMM != 0.2 {
		t.Fatal("precipitation")
	}
}

func TestParseForecastRejectsBadMoisture(t *testing.T) {
	warsaw := time.UTC
	body := []byte(`{
		"latitude": 1, "longitude": 2, "elevation": 3,
		"hourly": {
			"time": ["2026-09-03T10:00"],
			"precipitation": [0],
			"soil_temperature_6cm": [10],
			"soil_moisture_3_to_9cm": [1.2]
		}
	}`)
	_, err := ParseForecast(body, "u", time.Now(), warsaw)
	if err == nil {
		t.Fatal("expected moisture range error")
	}
}

func TestFetchRejectsNonOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	t.Cleanup(server.Close)

	_, err := Fetch(t.Context(), server.Client(), server.URL)
	if err == nil {
		t.Fatal("expected HTTP error")
	}
}
