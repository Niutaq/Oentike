package conditions

import (
	"testing"
)

func ptr(v float64) *float64 { return &v }

func TestScoreBoletusV2(t *testing.T) {
	cases := []struct {
		name          string
		snap          FactorSnapshotV2
		expectedScore int32
	}{
		{
			name: "perfect weather, good habitat",
			snap: FactorSnapshotV2{
				Precipitation14dMM: ptr(25.0),
				Precipitation3dMM:  ptr(25.0),
				SoilTemperatureC:   ptr(14.0),
				SoilMoistureM3M3:   ptr(0.25),
				AirTempMin2mC:      ptr(10.0),
				HabitatFit:         ptr(0.8),
			},
			expectedScore: 80, // 100 * max(0.5, 0.8) = 80
		},
		{
			name: "perfect weather, poor habitat (clamp to 0.5)",
			snap: FactorSnapshotV2{
				Precipitation14dMM: ptr(25.0),
				Precipitation3dMM:  ptr(25.0),
				SoilTemperatureC:   ptr(14.0),
				SoilMoistureM3M3:   ptr(0.25),
				AirTempMin2mC:      ptr(10.0),
				HabitatFit:         ptr(0.15),
			},
			expectedScore: 50, // 100 * max(0.5, 0.15) = 50
		},
		{
			name: "perfect weather, zero habitat (hard gate)",
			snap: FactorSnapshotV2{
				Precipitation14dMM: ptr(25.0),
				Precipitation3dMM:  ptr(25.0),
				SoilTemperatureC:   ptr(14.0),
				SoilMoistureM3M3:   ptr(0.25),
				AirTempMin2mC:      ptr(10.0),
				HabitatFit:         ptr(0.05),
			},
			expectedScore: 0,
		},
		{
			name: "frost (hard gate)",
			snap: FactorSnapshotV2{
				Precipitation14dMM: ptr(25.0),
				Precipitation3dMM:  ptr(25.0),
				SoilTemperatureC:   ptr(14.0),
				SoilMoistureM3M3:   ptr(0.25),
				AirTempMin2mC:      ptr(1.5),
				HabitatFit:         ptr(0.8),
			},
			expectedScore: 0,
		},
		{
			name: "missing factor",
			snap: FactorSnapshotV2{
				Precipitation14dMM: nil,
				Precipitation3dMM:  ptr(25.0),
				SoilTemperatureC:   ptr(14.0),
				SoilMoistureM3M3:   ptr(0.25),
				AirTempMin2mC:      ptr(10.0),
				HabitatFit:         ptr(0.8),
			},
			expectedScore: -1, // Indicates not ready
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, ready, err := scoreBoletusV2("test-cell", "2026-09-08", tc.snap)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.expectedScore == -1 {
				if ready {
					t.Errorf("expected not ready, got ready")
				}
				return
			}
			if !ready {
				t.Fatalf("expected ready, got not ready")
			}
			if res.Score != tc.expectedScore {
				t.Errorf("expected score %d, got %d", tc.expectedScore, res.Score)
			}
			if len(res.InputSHA256) == 0 {
				t.Errorf("expected SHA256, got empty")
			}
			t.Logf("Test %s passed. Input Hash: %s", tc.name, res.InputSHA256)
		})
	}
}
