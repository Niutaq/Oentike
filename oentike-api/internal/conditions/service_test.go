package conditions

import (
	"context"
	"testing"
	"time"

	conditionsv1 "oentike-api/internal/conditionsv1"
)

type fakeLookup struct {
	cell      Cell
	cellErr   error
	snap      FactorSnapshot
	snapErr   error
	fetchedAt *time.Time
	fetchErr  error
}

func (f fakeLookup) Cell(context.Context, string) (Cell, error) {
	return f.cell, f.cellErr
}

func (f fakeLookup) Factors(context.Context, string, string) (FactorSnapshot, error) {
	return f.snap, f.snapErr
}

func (f fakeLookup) LatestIngest(context.Context, string) (*time.Time, error) {
	return f.fetchedAt, f.fetchErr
}

func TestGetConditionsUnavailableWithoutWeather(t *testing.T) {
	server := NewServer(fakeLookup{
		cell: Cell{ID: "lasy-janowskie-01", Name: "Lasy Janowskie 01"},
	}, nil)
	server.now = func() time.Time {
		return time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	}

	got, err := server.GetConditions(context.Background(), &conditionsv1.GetConditionsRequest{
		CellId: "lasy-janowskie-01",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.GetStatus() != "unavailable" {
		t.Fatalf("status %q", got.GetStatus())
	}
	if got.Score != nil {
		t.Fatal("unavailable responses must not include a score")
	}
	if got.GetAlgorithmVersion() != algorithmVersion {
		t.Fatalf("algorithm version %q", got.GetAlgorithmVersion())
	}
	if got.GetTargetDate() != "2026-09-03" {
		t.Fatalf("target date %q", got.GetTargetDate())
	}
	if got.GetSpeciesSlug() != "boletus-edulis" {
		t.Fatalf("species %q", got.GetSpeciesSlug())
	}
	if len(got.GetFactors()) != 3 {
		t.Fatalf("expected 3 empty factors, got %d", len(got.GetFactors()))
	}
	for _, factor := range got.GetFactors() {
		if factor.Value != nil {
			t.Fatalf("factor %s should have no value", factor.GetId())
		}
	}
}

func TestGetConditionsScoresWhenFactorsComplete(t *testing.T) {
	precip, soilT, soilM := 42.5, 15.4, 0.25
	server := NewServer(fakeLookup{
		cell: Cell{ID: "lasy-janowskie-01", Name: "Lasy Janowskie 01"},
		snap: FactorSnapshot{
			PrecipitationMM:  &precip,
			SoilTemperatureC: &soilT,
			SoilMoistureM3M3: &soilM,
		},
	}, nil)
	server.now = func() time.Time {
		return time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	}

	got, err := server.GetConditions(context.Background(), &conditionsv1.GetConditionsRequest{
		CellId: "lasy-janowskie-01",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.GetStatus() != "ready" {
		t.Fatalf("status %q", got.GetStatus())
	}
	if got.Score == nil {
		t.Fatal("ready responses must include a score")
	}
	if *got.Score < 0 || *got.Score > 100 {
		t.Fatalf("score %d", *got.Score)
	}
	if got.GetConfidence() != "low" {
		t.Fatalf("confidence %q", got.GetConfidence())
	}
	if got.GetInputSha256() == "" {
		t.Fatal("missing input hash")
	}
	if got.GetFactors()[0].GetValue() != precip {
		t.Fatalf("precipitation %v", got.GetFactors()[0].GetValue())
	}
}

func TestGetConditionsIncludesFetchedAt(t *testing.T) {
	precip, soilT, soilM := 42.5, 15.4, 0.25
	fetched := time.Date(2026, 9, 4, 5, 40, 0, 0, time.UTC)
	server := NewServer(fakeLookup{
		cell: Cell{ID: "lasy-janowskie-01", Name: "Lasy Janowskie 01"},
		snap: FactorSnapshot{
			PrecipitationMM:  &precip,
			SoilTemperatureC: &soilT,
			SoilMoistureM3M3: &soilM,
		},
		fetchedAt: &fetched,
	}, nil)

	got, err := server.GetConditions(context.Background(), &conditionsv1.GetConditionsRequest{
		CellId: "lasy-janowskie-01",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.GetFetchedAt() != "2026-09-04T05:40:00Z" {
		t.Fatalf("fetched_at %q", got.GetFetchedAt())
	}
}

func TestGetConditionsUnknownSpeciesStaysUnavailable(t *testing.T) {
	precip, soilT, soilM := 42.5, 15.4, 0.25
	server := NewServer(fakeLookup{
		cell: Cell{ID: "lasy-janowskie-01", Name: "Lasy Janowskie 01"},
		snap: FactorSnapshot{
			PrecipitationMM:  &precip,
			SoilTemperatureC: &soilT,
			SoilMoistureM3M3: &soilM,
		},
	}, nil)

	got, err := server.GetConditions(context.Background(), &conditionsv1.GetConditionsRequest{
		CellId:      "lasy-janowskie-01",
		SpeciesSlug: "amanita-muscaria",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.GetStatus() != "unavailable" || got.Score != nil {
		t.Fatal("unknown species must not receive a boletus score")
	}
}

func TestFactorsFromRoundsDisplayValues(t *testing.T) {
	precip, soilT, soilM := 18.65, 17.1375, 0.092083
	got := factorsFrom(FactorSnapshot{
		PrecipitationMM:  &precip,
		SoilTemperatureC: &soilT,
		SoilMoistureM3M3: &soilM,
	})
	if got[0].GetValue() != 18.7 {
		t.Fatalf("precip %v", got[0].GetValue())
	}
	if got[1].GetValue() != 17.1 {
		t.Fatalf("soil t %v", got[1].GetValue())
	}
	if got[2].GetValue() != 0.092 {
		t.Fatalf("soil m %v", got[2].GetValue())
	}
}

func TestGetConditionsRequiresCell(t *testing.T) {
	server := NewServer(fakeLookup{}, nil)
	_, err := server.GetConditions(context.Background(), &conditionsv1.GetConditionsRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetConditionsUnknownCell(t *testing.T) {
	server := NewServer(fakeLookup{cellErr: ErrCellNotFound}, nil)
	_, err := server.GetConditions(context.Background(), &conditionsv1.GetConditionsRequest{
		CellId: "nope",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetSeasonNineDaysFromOurScores(t *testing.T) {
	precip, soilT, soilM := 25.0, 14.0, 0.25
	server := NewServer(fakeLookup{
		cell: Cell{ID: "lasy-janowskie-01", Name: "Lasy Janowskie 01"},
		snap: FactorSnapshot{
			PrecipitationMM:  &precip,
			SoilTemperatureC: &soilT,
			SoilMoistureM3M3: &soilM,
		},
	}, nil)
	server.now = func() time.Time {
		return time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	}

	got, err := server.GetSeason(context.Background(), &conditionsv1.GetSeasonRequest{
		CellId: "lasy-janowskie-01",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.GetDays()) != 9 {
		t.Fatalf("days %d", len(got.GetDays()))
	}
	if got.GetDays()[0].GetDate() != "2026-08-27" {
		t.Fatalf("first day %q", got.GetDays()[0].GetDate())
	}
	if got.GetDays()[8].GetDate() != "2026-09-04" {
		t.Fatalf("last day %q", got.GetDays()[8].GetDate())
	}
	if got.GetDays()[8].GetStatus() != "ready" || got.GetDays()[8].Score == nil {
		t.Fatal("expected scored last day")
	}
}

func TestGetSeasonUnknownSpeciesStaysUnavailable(t *testing.T) {
	precip, soilT, soilM := 25.0, 14.0, 0.25
	server := NewServer(fakeLookup{
		cell: Cell{ID: "lasy-janowskie-01", Name: "Lasy Janowskie 01"},
		snap: FactorSnapshot{
			PrecipitationMM:  &precip,
			SoilTemperatureC: &soilT,
			SoilMoistureM3M3: &soilM,
		},
	}, nil)

	got, err := server.GetSeason(context.Background(), &conditionsv1.GetSeasonRequest{
		CellId:      "lasy-janowskie-01",
		SpeciesSlug: "amanita-muscaria",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, day := range got.GetDays() {
		if day.GetStatus() != "unavailable" || day.Score != nil {
			t.Fatal("unknown species must not receive a boletus score")
		}
	}
}

func TestGetSeasonRequiresCell(t *testing.T) {
	server := NewServer(fakeLookup{}, nil)
	_, err := server.GetSeason(context.Background(), &conditionsv1.GetSeasonRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}
