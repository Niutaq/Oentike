package conditions

import "testing"

func TestScoreBoletusNeedsAllThreeFactors(t *testing.T) {
	precip := 20.2
	_, ok, err := scoreBoletus("lasy-janowskie-01", "2026-09-03", FactorSnapshot{
		PrecipitationMM: &precip,
	})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("incomplete snapshot must not produce a score")
	}
}

func TestScoreBoletusPilotWeather(t *testing.T) {
	precip, soilT, soilM := 20.2, 16.033333333333335, 0.09016666666666667
	got, ok, err := scoreBoletus("lasy-janowskie-01", "2026-09-03", FactorSnapshot{
		PrecipitationMM:  &precip,
		SoilTemperatureC: &soilT,
		SoilMoistureM3M3: &soilM,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected a score")
	}
	if got.Score < 60 || got.Score > 80 {
		t.Fatalf("score %d outside expected band for this weather", got.Score)
	}
	if got.Confidence != "low" {
		t.Fatalf("confidence %q", got.Confidence)
	}
	if len(got.InputSHA256) != 64 {
		t.Fatalf("input hash %q", got.InputSHA256)
	}
}

func TestInterpolateClamp(t *testing.T) {
	if interpolate(-1, precipMM) != 0 {
		t.Fatal("below range")
	}
	if interpolate(200, precipMM) != 0.05 {
		t.Fatal("above range")
	}
}
