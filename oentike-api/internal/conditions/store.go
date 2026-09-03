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
