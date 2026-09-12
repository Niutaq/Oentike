package conditions

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
)

const (
	algorithmVersionV2 = "oentike-conditions/0.2.0-boletus"
	pilotSpeciesV2     = "boletus-edulis"
)

type FactorSnapshotV2 struct {
	Precipitation14dMM *float64
	Precipitation3dMM  *float64
	SoilTemperatureC   *float64
	SoilMoistureM3M3   *float64
	AirTempMin2mC      *float64
	HabitatFit         *float64
}

func scoreBoletusV2(cellID, targetDate string, snap FactorSnapshotV2) (ScoreResult, bool, error) {
	if snap.Precipitation14dMM == nil || snap.Precipitation3dMM == nil ||
		snap.SoilTemperatureC == nil || snap.SoilMoistureM3M3 == nil ||
		snap.AirTempMin2mC == nil || snap.HabitatFit == nil {
		return ScoreResult{}, false, nil
	}

	p14 := interpolate(*snap.Precipitation14dMM, precipMM)
	p3 := interpolate(*snap.Precipitation3dMM, precipMM)
	t := interpolate(*snap.SoilTemperatureC, soilTempC)
	m := interpolate(*snap.SoilMoistureM3M3, soilMoist)

	baseWeather := 0.20*p14 + 0.25*p3 + 0.35*t + 0.20*m

	var score int32
	if *snap.AirTempMin2mC <= 2.0 {
		score = 0
	} else if *snap.HabitatFit < 0.1 {
		score = 0
	} else {
		multiplier := math.Max(0.5, *snap.HabitatFit)
		score = int32(math.Round(baseWeather * multiplier * 100))
		if score < 0 {
			score = 0
		}
		if score > 100 {
			score = 100
		}
	}

	payload := struct {
		Algorithm      string  `json:"algorithm"`
		CellID         string  `json:"cell_id"`
		SpeciesSlug    string  `json:"species_slug"`
		TargetDate     string  `json:"target_date"`
		Precip14d      float64 `json:"precip_14d"`
		Precip3d       float64 `json:"precip_3d"`
		SoilTemp       float64 `json:"soil_temp_6cm"`
		SoilMoisture   float64 `json:"soil_moisture_3_9cm"`
		AirTempMin     float64 `json:"air_temp_min_2m"`
		HabitatFit     float64 `json:"habitat_fit"`
	}{
		Algorithm:      algorithmVersionV2,
		CellID:         cellID,
		SpeciesSlug:    pilotSpeciesV2,
		TargetDate:     targetDate,
		Precip14d:      *snap.Precipitation14dMM,
		Precip3d:       *snap.Precipitation3dMM,
		SoilTemp:       *snap.SoilTemperatureC,
		SoilMoisture:   *snap.SoilMoistureM3M3,
		AirTempMin:     *snap.AirTempMin2mC,
		HabitatFit:     *snap.HabitatFit,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return ScoreResult{}, false, fmt.Errorf("hash score inputs v2: %w", err)
	}
	sum := sha256.Sum256(raw)

	factors, err := json.Marshal([]map[string]any{
		{"id": "precipitation_14d", "unit": "mm", "value": *snap.Precipitation14dMM},
		{"id": "precipitation_3d", "unit": "mm", "value": *snap.Precipitation3dMM},
		{"id": "soil_temperature", "unit": "°C", "value": *snap.SoilTemperatureC},
		{"id": "soil_moisture", "unit": "m3/m3", "value": *snap.SoilMoistureM3M3},
		{"id": "air_temp_min", "unit": "°C", "value": *snap.AirTempMin2mC},
		{"id": "habitat_fit", "unit": "", "value": *snap.HabitatFit},
	})
	if err != nil {
		return ScoreResult{}, false, fmt.Errorf("encode score factors v2: %w", err)
	}

	return ScoreResult{
		Score:       score,
		Confidence:  "low",
		InputSHA256: hex.EncodeToString(sum[:]),
		FactorsJSON: factors,
	}, true, nil
}
