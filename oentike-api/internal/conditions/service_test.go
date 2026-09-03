package conditions

import (
	"context"
	"testing"
	"time"

	conditionsv1 "oentike-api/internal/conditionsv1"
)

type fakeLookup struct {
	cell    Cell
	cellErr error
	snap    FactorSnapshot
	snapErr error
}

func (f fakeLookup) Cell(context.Context, string) (Cell, error) {
	return f.cell, f.cellErr
}

func (f fakeLookup) Factors(context.Context, string, string) (FactorSnapshot, error) {
	return f.snap, f.snapErr
}

func TestGetConditionsUnavailableWithoutWeather(t *testing.T) {
	server := NewServer(fakeLookup{
		cell: Cell{ID: "lasy-janowskie-01", Name: "Lasy Janowskie 01"},
	})
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

func TestGetConditionsFillsFactorsWithoutInventingScore(t *testing.T) {
	precip, soilT, soilM := 42.5, 15.4, 0.096
	server := NewServer(fakeLookup{
		cell: Cell{ID: "lasy-janowskie-01", Name: "Lasy Janowskie 01"},
		snap: FactorSnapshot{
			PrecipitationMM:  &precip,
			SoilTemperatureC: &soilT,
			SoilMoistureM3M3: &soilM,
		},
	})
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
		t.Fatal("must not invent a score")
	}
	if got.GetFactors()[0].GetValue() != precip {
		t.Fatalf("precipitation %v", got.GetFactors()[0].GetValue())
	}
	if got.GetFactors()[1].GetValue() != soilT {
		t.Fatalf("soil temperature %v", got.GetFactors()[1].GetValue())
	}
	if got.GetFactors()[2].GetValue() != soilM {
		t.Fatalf("soil moisture %v", got.GetFactors()[2].GetValue())
	}
}

func TestGetConditionsRequiresCell(t *testing.T) {
	server := NewServer(fakeLookup{})
	_, err := server.GetConditions(context.Background(), &conditionsv1.GetConditionsRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetConditionsUnknownCell(t *testing.T) {
	server := NewServer(fakeLookup{cellErr: ErrCellNotFound})
	_, err := server.GetConditions(context.Background(), &conditionsv1.GetConditionsRequest{
		CellId: "nope",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
