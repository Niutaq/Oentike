package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	Provider         = "open-meteo"
	RequestedModel   = "best_match"
	DefaultBaseURL   = "https://api.open-meteo.com"
	hourlyTimeLayout = "2006-01-02T15:04"
)

type Sample struct {
	ValidAt              time.Time
	DataKind             string
	PrecipitationMM      *float64
	SoilTemperature6cmC  *float64
	SoilMoisture3To9cmM3 *float64
}

type Forecast struct {
	Latitude   float64
	Longitude  float64
	ElevationM float64
	SHA256     string
	RequestURL string
	Samples    []Sample
}

type forecastJSON struct {
	Latitude  float64    `json:"latitude"`
	Longitude float64    `json:"longitude"`
	Elevation float64    `json:"elevation"`
	Hourly    hourlyJSON `json:"hourly"`
}

type hourlyJSON struct {
	Time               []string   `json:"time"`
	Precipitation      []*float64 `json:"precipitation"`
	SoilTemperature6cm []*float64 `json:"soil_temperature_6cm"`
	SoilMoisture3To9cm []*float64 `json:"soil_moisture_3_to_9cm"`
}

func ForecastURL(base string, latitude, longitude float64) (string, error) {
	if strings.TrimSpace(base) == "" {
		base = DefaultBaseURL
	}
	parsed, err := url.Parse(strings.TrimRight(base, "/") + "/v1/forecast")
	if err != nil {
		return "", fmt.Errorf("open-meteo base url: %w", err)
	}
	query := parsed.Query()
	query.Set("latitude", strconv.FormatFloat(latitude, 'f', 6, 64))
	query.Set("longitude", strconv.FormatFloat(longitude, 'f', 6, 64))
	query.Set("hourly", "precipitation,soil_temperature_6cm,soil_moisture_3_to_9cm")
	query.Set("timezone", "Europe/Warsaw")
	query.Set("past_days", "14")
	query.Set("forecast_days", "2")
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func ParseForecast(body []byte, requestURL string, fetchedAt time.Time, loc *time.Location) (Forecast, error) {
	sum := sha256.Sum256(body)
	var payload forecastJSON
	if err := json.Unmarshal(body, &payload); err != nil {
		return Forecast{}, fmt.Errorf("decode open-meteo json: %w", err)
	}
	hourly := payload.Hourly
	n := len(hourly.Time)
	if n == 0 {
		return Forecast{}, fmt.Errorf("open-meteo hourly.time is empty")
	}
	if err := sameLength("precipitation", n, len(hourly.Precipitation)); err != nil {
		return Forecast{}, err
	}
	if err := sameLength("soil_temperature_6cm", n, len(hourly.SoilTemperature6cm)); err != nil {
		return Forecast{}, err
	}
	if err := sameLength("soil_moisture_3_to_9cm", n, len(hourly.SoilMoisture3To9cm)); err != nil {
		return Forecast{}, err
	}

	samples := make([]Sample, 0, n)
	for i, raw := range hourly.Time {
		validAt, err := time.ParseInLocation(hourlyTimeLayout, raw, loc)
		if err != nil {
			return Forecast{}, fmt.Errorf("hourly time %q: %w", raw, err)
		}
		precip := hourly.Precipitation[i]
		soilT := hourly.SoilTemperature6cm[i]
		soilM := hourly.SoilMoisture3To9cm[i]
		if err := validateSample(precip, soilM); err != nil {
			return Forecast{}, fmt.Errorf("hour %s: %w", raw, err)
		}
		kind := "forecast"
		if !validAt.After(fetchedAt) {
			kind = "past"
		}
		samples = append(samples, Sample{
			ValidAt:              validAt,
			DataKind:             kind,
			PrecipitationMM:      precip,
			SoilTemperature6cmC:  soilT,
			SoilMoisture3To9cmM3: soilM,
		})
	}

	return Forecast{
		Latitude:   payload.Latitude,
		Longitude:  payload.Longitude,
		ElevationM: payload.Elevation,
		SHA256:     hex.EncodeToString(sum[:]),
		RequestURL: requestURL,
		Samples:    samples,
	}, nil
}

func sameLength(field string, want, got int) error {
	if want != got {
		return fmt.Errorf("open-meteo hourly.%s length %d, expected %d", field, got, want)
	}
	return nil
}

func validateSample(precip, moisture *float64) error {
	if precip != nil && *precip < 0 {
		return fmt.Errorf("precipitation_mm is negative")
	}
	if moisture != nil && (*moisture < 0 || *moisture > 1) {
		return fmt.Errorf("soil_moisture_3_to_9cm is out of 0–1")
	}
	return nil
}
