package conditions

import "testing"

func TestScoreSpeciesCoversPilotPackAndRejectsUnknown(t *testing.T) {
	precip, soilT, soilM := 28.0, 14.0, 0.26
	snap := FactorSnapshot{
		PrecipitationMM:  &precip,
		SoilTemperatureC: &soilT,
		SoilMoistureM3M3: &soilM,
	}

	for slug := range speciesModels {
		got, ok, err := scoreSpecies(slug, "cell", "2026-10-26", snap)
		if err != nil {
			t.Fatalf("%s: %v", slug, err)
		}
		if !ok || got.Score < 0 || got.Score > 100 {
			t.Fatalf("%s: expected ready score, got ok=%v score=%d", slug, ok, got.Score)
		}
	}

	_, ok, err := scoreSpecies("not-a-real-species", "cell", "2026-10-26", snap)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("unknown slug must not score")
	}
}

func TestPilotSpeciesScoresAreNotIdenticalClones(t *testing.T) {
	precip, soilT, soilM := 28.0, 14.0, 0.26
	snap := FactorSnapshot{
		PrecipitationMM:  &precip,
		SoilTemperatureC: &soilT,
		SoilMoistureM3M3: &soilM,
	}
	a, _, err := scoreSpecies("boletus-edulis", "cell", "2026-10-26", snap)
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := scoreSpecies("laetiporus-sulphureus", "cell", "2026-10-26", snap)
	if err != nil {
		t.Fatal(err)
	}
	if a.Score == b.Score && a.InputSHA256 == b.InputSHA256 {
		t.Fatal("distinct species models should not collapse to an identical payload")
	}
}
