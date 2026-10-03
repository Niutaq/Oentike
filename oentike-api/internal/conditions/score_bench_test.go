package conditions

import "testing"

var benchmarkScore ScoreResult

func BenchmarkScoreBoletus(b *testing.B) {
	p, t, m := 25.0, 14.0, 0.25
	snap := FactorSnapshot{PrecipitationMM: &p, SoilTemperatureC: &t, SoilMoistureM3M3: &m}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		result, ready, err := scoreBoletus("fixture-cell", "2026-09-24", snap)
		if err != nil || !ready {
			b.Fatalf("score: ready=%v err=%v", ready, err)
		}
		benchmarkScore = result
	}
}

func BenchmarkScoreBoletusV2(b *testing.B) {
	p, t, m, a, h := 25.0, 14.0, 0.25, 10.0, 0.8
	snap := FactorSnapshotV2{Precipitation14dMM: &p, Precipitation3dMM: &p, SoilTemperatureC: &t, SoilMoistureM3M3: &m, AirTempMin2mC: &a, HabitatFit: &h}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		result, ready, err := scoreBoletusV2("fixture-cell", "2026-09-24", snap)
		if err != nil || !ready {
			b.Fatalf("score: ready=%v err=%v", ready, err)
		}
		benchmarkScore = result
	}
}
