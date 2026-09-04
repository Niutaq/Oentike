package conditions

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
)

const (
	algorithmVersion = "oentike-conditions/0.1.0-boletus"
	pilotSpecies     = "boletus-edulis"
)

// First published heuristic for Boletus edulis from Open-Meteo cell factors.
// Trapezoids are coarse habitat envelopes, not a field survey. Confidence stays low.
var (
	precipMM  = [][2]float64{{0, 0}, {10, 0.3}, {25, 1}, {50, 0.75}, {90, 0.2}, {120, 0.05}}
	soilTempC = [][2]float64{{5, 0.1}, {10, 0.7}, {14, 1}, {18, 0.7}, {22, 0.2}, {26, 0.05}}
	soilMoist = [][2]float64{{0.05, 0.15}, {0.15, 0.7}, {0.25, 1}, {0.35, 0.7}, {0.5, 0.2}}
)

type ScoreResult struct {
	Score       int32
	Confidence  string
	InputSHA256 string
	FactorsJSON []byte
}

func scoreBoletus(cellID, targetDate string, snap FactorSnapshot) (ScoreResult, bool, error) {
	if snap.PrecipitationMM == nil || snap.SoilTemperatureC == nil || snap.SoilMoistureM3M3 == nil {
		return ScoreResult{}, false, nil
	}

	p := interpolate(*snap.PrecipitationMM, precipMM)
	t := interpolate(*snap.SoilTemperatureC, soilTempC)
	m := interpolate(*snap.SoilMoistureM3M3, soilMoist)
	weighted := 0.40*p + 0.35*t + 0.25*m
	score := int32(math.Round(100 * weighted))
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	payload := struct {
		Algorithm     string  `json:"algorithm"`
		CellID        string  `json:"cell_id"`
		SpeciesSlug   string  `json:"species_slug"`
		TargetDate    string  `json:"target_date"`
		Precipitation float64 `json:"precipitation_mm"`
		SoilTemp      float64 `json:"soil_temperature_c"`
		SoilMoisture  float64 `json:"soil_moisture_m3_m3"`
	}{
		Algorithm:     algorithmVersion,
		CellID:        cellID,
		SpeciesSlug:   pilotSpecies,
		TargetDate:    targetDate,
		Precipitation: *snap.PrecipitationMM,
		SoilTemp:      *snap.SoilTemperatureC,
		SoilMoisture:  *snap.SoilMoistureM3M3,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return ScoreResult{}, false, fmt.Errorf("hash score inputs: %w", err)
	}
	sum := sha256.Sum256(raw)

	factors, err := json.Marshal([]map[string]any{
		{"id": "precipitation", "unit": "mm", "value": *snap.PrecipitationMM},
		{"id": "soil_temperature", "unit": "°C", "value": *snap.SoilTemperatureC},
		{"id": "soil_moisture", "unit": "m3/m3", "value": *snap.SoilMoistureM3M3},
	})
	if err != nil {
		return ScoreResult{}, false, fmt.Errorf("encode score factors: %w", err)
	}

	return ScoreResult{
		Score:       score,
		Confidence:  "low",
		InputSHA256: hex.EncodeToString(sum[:]),
		FactorsJSON: factors,
	}, true, nil
}

func interpolate(x float64, pts [][2]float64) float64 {
	if x <= pts[0][0] {
		return pts[0][1]
	}
	last := len(pts) - 1
	if x >= pts[last][0] {
		return pts[last][1]
	}
	for i := 0; i < last; i++ {
		x0, y0 := pts[i][0], pts[i][1]
		x1, y1 := pts[i+1][0], pts[i+1][1]
		if x <= x1 {
			t := (x - x0) / (x1 - x0)
			return y0 + t*(y1-y0)
		}
	}
	return pts[last][1]
}
