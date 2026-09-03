package conditions

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	conditionsv1 "oentike-api/internal/conditionsv1"
)

const algorithmVersion = "oentike-conditions/0.0.1"

// Lookup is the PostGIS reads GetConditions needs. Tests fake this.
type Lookup interface {
	Cell(ctx context.Context, id string) (Cell, error)
	Factors(ctx context.Context, cellID, targetDate string) (FactorSnapshot, error)
}

type Server struct {
	conditionsv1.UnimplementedConditionsServiceServer
	lookup Lookup
	now    func() time.Time
}

func NewServer(lookup Lookup) *Server {
	return &Server{
		lookup: lookup,
		now:    time.Now,
	}
}

func (s *Server) GetConditions(ctx context.Context, req *conditionsv1.GetConditionsRequest) (*conditionsv1.ConditionsResponse, error) {
	cellID := strings.TrimSpace(req.GetCellId())
	if cellID == "" {
		return nil, status.Error(codes.InvalidArgument, "cell_id is required")
	}

	species := strings.TrimSpace(req.GetSpeciesSlug())
	if species == "" {
		species = "boletus-edulis"
	}

	targetDate, err := parseTargetDate(req.GetTargetDate(), s.now())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	cell, err := s.lookup.Cell(ctx, cellID)
	if errors.Is(err, ErrCellNotFound) {
		return nil, status.Errorf(codes.NotFound, "unknown cell %q", cellID)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load cell: %v", err)
	}

	snap, err := s.lookup.Factors(ctx, cell.ID, targetDate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load weather: %v", err)
	}

	// Samples may exist; scoring is not implemented. Never invent a 0–100 score.
	return &conditionsv1.ConditionsResponse{
		CellId:           cell.ID,
		CellName:         cell.Name,
		SpeciesSlug:      species,
		TargetDate:       targetDate,
		Status:           "unavailable",
		Confidence:       "low",
		AlgorithmVersion: algorithmVersion,
		Factors:          factorsFrom(snap),
	}, nil
}

func factorsFrom(snap FactorSnapshot) []*conditionsv1.Factor {
	return []*conditionsv1.Factor{
		{Id: "precipitation", Unit: "mm", Value: snap.PrecipitationMM},
		{Id: "soil_temperature", Unit: "°C", Value: snap.SoilTemperatureC},
		{Id: "soil_moisture", Unit: "m3/m3", Value: snap.SoilMoistureM3M3},
	}
}

func parseTargetDate(raw string, now time.Time) (string, error) {
	raw = strings.TrimSpace(raw)
	warsaw, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		return "", fmt.Errorf("load Europe/Warsaw: %w", err)
	}
	if raw == "" {
		return now.In(warsaw).Format(time.DateOnly), nil
	}
	if _, err := time.ParseInLocation(time.DateOnly, raw, warsaw); err != nil {
		return "", fmt.Errorf("target_date must be YYYY-MM-DD")
	}
	return raw, nil
}
