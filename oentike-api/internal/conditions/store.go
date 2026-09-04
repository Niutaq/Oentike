package conditions

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrCellNotFound = errors.New("cell not found")

type Cell struct {
	ID   string
	Name string
}

type FactorSnapshot struct {
	PrecipitationMM  *float64
	SoilTemperatureC *float64
	SoilMoistureM3M3 *float64
}

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) Cell(ctx context.Context, id string) (Cell, error) {
	var cell Cell
	err := s.db.QueryRow(ctx, `SELECT id, name FROM cells WHERE id = $1`, id).Scan(&cell.ID, &cell.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return Cell{}, ErrCellNotFound
	}
	if err != nil {
		return Cell{}, fmt.Errorf("load cell: %w", err)
	}
	return cell, nil
}

func (s *Store) Factors(ctx context.Context, cellID, targetDate string) (FactorSnapshot, error) {
	warsaw, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		return FactorSnapshot{}, fmt.Errorf("load Europe/Warsaw: %w", err)
	}
	day, err := time.ParseInLocation(time.DateOnly, targetDate, warsaw)
	if err != nil {
		return FactorSnapshot{}, fmt.Errorf("target_date: %w", err)
	}
	dayEnd := day.AddDate(0, 0, 1)
	windowStart := dayEnd.AddDate(0, 0, -14)

	var snap FactorSnapshot
	err = s.db.QueryRow(ctx, `
		SELECT
			SUM(precipitation_mm) FILTER (
				WHERE precipitation_mm IS NOT NULL
					AND valid_at >= $2 AND valid_at < $3
			),
			AVG(soil_temperature_6cm_c) FILTER (
				WHERE soil_temperature_6cm_c IS NOT NULL
					AND valid_at >= $4 AND valid_at < $3
			),
			AVG(soil_moisture_3_to_9cm_m3_m3) FILTER (
				WHERE soil_moisture_3_to_9cm_m3_m3 IS NOT NULL
					AND valid_at >= $4 AND valid_at < $3
			)
		FROM weather_samples
		WHERE cell_id = $1
	`, cellID, windowStart, dayEnd, day).Scan(
		&snap.PrecipitationMM,
		&snap.SoilTemperatureC,
		&snap.SoilMoistureM3M3,
	)
	if err != nil {
		return FactorSnapshot{}, fmt.Errorf("load weather factors: %w", err)
	}
	return snap, nil
}

func (s *Store) LatestIngest(ctx context.Context, cellID string) (*time.Time, error) {
	var fetched time.Time
	err := s.db.QueryRow(ctx, `
		SELECT fetched_at
		FROM ingest_runs
		WHERE cell_id = $1
		ORDER BY fetched_at DESC
		LIMIT 1
	`, cellID).Scan(&fetched)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load latest ingest: %w", err)
	}
	return &fetched, nil
}

type ScoreRecord struct {
	CellID           string
	SpeciesSlug      string
	TargetDate       string
	Status           string
	Score            *int32
	Confidence       string
	FactorsJSON      []byte
	AlgorithmVersion string
	InputSHA256      string
}

func (s *Store) SaveScore(ctx context.Context, rec ScoreRecord) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO condition_scores (
			cell_id, species_slug, target_date, status, score, confidence,
			factors, algorithm_version, input_sha256
		) VALUES ($1, $2, $3::date, $4, $5, $6, $7::jsonb, $8, $9)
		ON CONFLICT (cell_id, species_slug, target_date, algorithm_version)
		DO UPDATE SET
			status = EXCLUDED.status,
			score = EXCLUDED.score,
			confidence = EXCLUDED.confidence,
			factors = EXCLUDED.factors,
			input_sha256 = EXCLUDED.input_sha256,
			calculated_at = now()
	`, rec.CellID, rec.SpeciesSlug, rec.TargetDate, rec.Status, rec.Score,
		rec.Confidence, rec.FactorsJSON, rec.AlgorithmVersion, rec.InputSHA256)
	if err != nil {
		return fmt.Errorf("save condition score: %w", err)
	}
	return nil
}
