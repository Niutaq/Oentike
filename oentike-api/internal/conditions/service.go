package conditions

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	conditionsv1 "oentike-api/internal/conditionsv1"
)

type Lookup interface {
	Cell(ctx context.Context, id string) (Cell, error)
	MaterializeForestUnit(ctx context.Context, id string) (Cell, error)
	SearchCells(ctx context.Context, query string, limit int) ([]Cell, error)
	Factors(ctx context.Context, cellID, targetDate string) (FactorSnapshot, error)
	LatestIngest(ctx context.Context, cellID string) (*time.Time, error)
}

type WeatherRefresher interface {
	Refresh(ctx context.Context, cellID string) (ingested bool, err error)
}

type Persister interface {
	SaveScore(ctx context.Context, rec ScoreRecord) error
}

type Server struct {
	conditionsv1.UnimplementedConditionsServiceServer
	lookup  Lookup
	persist Persister
	weather WeatherRefresher
	now     func() time.Time
}

func NewServer(lookup Lookup, persist Persister, weather WeatherRefresher) *Server {
	return &Server{
		lookup:  lookup,
		persist: persist,
		weather: weather,
		now:     time.Now,
	}
}

func (s *Server) resolveCell(ctx context.Context, cellID string) (Cell, error) {
	cell, err := s.lookup.Cell(ctx, cellID)
	if err == nil {
		return cell, nil
	}
	if !errors.Is(err, ErrCellNotFound) {
		return Cell{}, err
	}
	cell, err = s.lookup.MaterializeForestUnit(ctx, cellID)
	if errors.Is(err, ErrForestUnitNotFound) {
		return Cell{}, ErrCellNotFound
	}
	return cell, err
}

func summaryFrom(cell Cell) *conditionsv1.CellSummary {
	return &conditionsv1.CellSummary{
		Id:    cell.ID,
		Name:  cell.Name,
		Lon:   cell.Lon,
		Lat:   cell.Lat,
		West:  cell.West,
		South: cell.South,
		East:  cell.East,
		North: cell.North,
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

	cell, err := s.resolveCell(ctx, cellID)
	if errors.Is(err, ErrCellNotFound) {
		return nil, status.Errorf(codes.NotFound, "unknown cell %q", cellID)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load cell: %v", err)
	}

	if s.weather != nil {
		if _, err := s.weather.Refresh(ctx, cell.ID); err != nil {
			// Keep serving cached factors when refresh fails.
			_ = err
		}
	}

	snap, err := s.lookup.Factors(ctx, cell.ID, targetDate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load weather: %v", err)
	}

	resp := &conditionsv1.ConditionsResponse{
		CellId:           cell.ID,
		CellName:         cell.Name,
		SpeciesSlug:      species,
		TargetDate:       targetDate,
		Status:           "unavailable",
		Confidence:       "low",
		AlgorithmVersion: algorithmVersion,
		Factors:          factorsFrom(snap),
	}

	fetched, err := s.lookup.LatestIngest(ctx, cell.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load ingest: %v", err)
	}
	if fetched != nil {
		stamp := fetched.UTC().Format(time.RFC3339)
		resp.FetchedAt = &stamp
	}

	if species != pilotSpecies {
		return resp, nil
	}

	result, ok, err := scoreBoletus(cell.ID, targetDate, snap)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "score: %v", err)
	}
	if !ok {
		return resp, nil
	}

	score := result.Score
	resp.Status = "ready"
	resp.Score = &score
	resp.Confidence = result.Confidence
	resp.InputSha256 = &result.InputSHA256

	if s.persist != nil {
		if err := s.persist.SaveScore(ctx, ScoreRecord{
			CellID:           cell.ID,
			SpeciesSlug:      species,
			TargetDate:       targetDate,
			Status:           "ready",
			Score:            &score,
			Confidence:       result.Confidence,
			FactorsJSON:      result.FactorsJSON,
			AlgorithmVersion: algorithmVersion,
			InputSHA256:      result.InputSHA256,
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "save score: %v", err)
		}
	}

	return resp, nil
}

func (s *Server) GetSeason(ctx context.Context, req *conditionsv1.GetSeasonRequest) (*conditionsv1.SeasonResponse, error) {
	cellID := strings.TrimSpace(req.GetCellId())
	if cellID == "" {
		return nil, status.Error(codes.InvalidArgument, "cell_id is required")
	}

	species := strings.TrimSpace(req.GetSpeciesSlug())
	if species == "" {
		species = pilotSpecies
	}

	n := int(req.GetDays())
	if n <= 0 {
		n = 9
	}
	if n > 14 {
		n = 14
	}

	today, err := parseTargetDate("", s.now())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	warsaw, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load Europe/Warsaw: %v", err)
	}
	end, err := time.ParseInLocation(time.DateOnly, today, warsaw)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "parse today: %v", err)
	}

	cell, err := s.resolveCell(ctx, cellID)
	if errors.Is(err, ErrCellNotFound) {
		return nil, status.Errorf(codes.NotFound, "unknown cell %q", cellID)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load cell: %v", err)
	}

	if s.weather != nil {
		if _, err := s.weather.Refresh(ctx, cell.ID); err != nil {
			_ = err
		}
	}

	days := make([]*conditionsv1.SeasonDay, 0, n)
	for i := n - 1; i >= 0; i-- {
		date := end.AddDate(0, 0, -i).Format(time.DateOnly)
		day := &conditionsv1.SeasonDay{Date: date, Status: "unavailable"}
		if species == pilotSpecies {
			snap, err := s.lookup.Factors(ctx, cell.ID, date)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "load weather: %v", err)
			}
			result, ok, err := scoreBoletus(cell.ID, date, snap)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "score: %v", err)
			}
			if ok {
				score := result.Score
				day.Status = "ready"
				day.Score = &score
			}
		}
		days = append(days, day)
	}

	return &conditionsv1.SeasonResponse{
		CellId:           cell.ID,
		CellName:         cell.Name,
		SpeciesSlug:      species,
		AlgorithmVersion: algorithmVersion,
		Days:             days,
	}, nil
}

func (s *Server) SearchCells(ctx context.Context, req *conditionsv1.SearchCellsRequest) (*conditionsv1.SearchCellsResponse, error) {
	cells, err := s.lookup.SearchCells(ctx, req.GetQuery(), int(req.GetLimit()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "search cells: %v", err)
	}
	out := make([]*conditionsv1.CellSummary, 0, len(cells))
	for _, cell := range cells {
		out = append(out, summaryFrom(cell))
	}
	return &conditionsv1.SearchCellsResponse{Cells: out}, nil
}

func (s *Server) EnsureCell(ctx context.Context, req *conditionsv1.EnsureCellRequest) (*conditionsv1.EnsureCellResponse, error) {
	cellID := strings.TrimSpace(req.GetCellId())
	if cellID == "" {
		return nil, status.Error(codes.InvalidArgument, "cell_id is required")
	}
	cell, err := s.resolveCell(ctx, cellID)
	if errors.Is(err, ErrCellNotFound) {
		return nil, status.Errorf(codes.NotFound, "unknown cell %q", cellID)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load cell: %v", err)
	}

	ingested := false
	if s.weather != nil {
		ingested, err = s.weather.Refresh(ctx, cell.ID)
		if err != nil {
			return nil, status.Errorf(codes.Unavailable, "weather ingest: %v", err)
		}
	}
	return &conditionsv1.EnsureCellResponse{
		Cell:     summaryFrom(cell),
		Ingested: ingested,
	}, nil
}

func factorsFrom(snap FactorSnapshot) []*conditionsv1.Factor {
	return []*conditionsv1.Factor{
		{Id: "precipitation", Unit: "mm", Value: roundPtr(snap.PrecipitationMM, 1)},
		{Id: "soil_temperature", Unit: "°C", Value: roundPtr(snap.SoilTemperatureC, 1)},
		{Id: "soil_moisture", Unit: "m3/m3", Value: roundPtr(snap.SoilMoistureM3M3, 3)},
	}
}

func roundPtr(v *float64, decimals int) *float64 {
	if v == nil {
		return nil
	}
	pow := math.Pow(10, float64(decimals))
	rounded := math.Round(*v*pow) / pow
	return &rounded
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
