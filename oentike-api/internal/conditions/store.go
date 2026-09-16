package conditions

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCellNotFound       = errors.New("cell not found")
	ErrForestUnitNotFound = errors.New("forest unit not found")
)

type Cell struct {
	ID         string
	Name       string
	Lon        float64
	Lat        float64
	West       float64
	South      float64
	East       float64
	North      float64
	HabitatFit float64
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

const cellSelectSQL = `
	SELECT
		id,
		name,
		ST_X(ST_Transform(ST_Centroid(geom), 4326)),
		ST_Y(ST_Transform(ST_Centroid(geom), 4326)),
		ST_XMin(ST_Transform(geom, 4326)),
		ST_YMin(ST_Transform(geom, 4326)),
		ST_XMax(ST_Transform(geom, 4326)),
		ST_YMax(ST_Transform(geom, 4326)),
		habitat_fit
	FROM %s
	WHERE id = $1
`

func scanCell(row pgx.Row) (Cell, error) {
	var cell Cell
	err := row.Scan(
		&cell.ID, &cell.Name, &cell.Lon, &cell.Lat,
		&cell.West, &cell.South, &cell.East, &cell.North, &cell.HabitatFit,
	)
	return cell, err
}

func (s *Store) Cell(ctx context.Context, id string) (Cell, error) {
	cell, err := scanCell(s.db.QueryRow(ctx, fmt.Sprintf(cellSelectSQL, "cells"), id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Cell{}, ErrCellNotFound
	}
	if err != nil {
		return Cell{}, fmt.Errorf("load cell: %w", err)
	}
	return cell, nil
}

func (s *Store) MaterializeForestUnit(ctx context.Context, id string) (Cell, error) {
	tag, err := s.db.Exec(ctx, `
		INSERT INTO cells (id, name, geom, habitat_fit)
		SELECT id, name, geom, habitat_fit
		FROM forest_units
		WHERE id = $1
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			geom = EXCLUDED.geom,
			habitat_fit = EXCLUDED.habitat_fit
	`, id)
	if err != nil {
		return Cell{}, fmt.Errorf("materialize forest unit: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// ON CONFLICT still reports 1 row; check existence when insert missed
		var exists bool
		if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM forest_units WHERE id = $1)`, id).Scan(&exists); err != nil {
			return Cell{}, fmt.Errorf("check forest unit: %w", err)
		}
		if !exists {
			return Cell{}, ErrForestUnitNotFound
		}
	}
	cell, err := s.Cell(ctx, id)
	if errors.Is(err, ErrCellNotFound) {
		return Cell{}, ErrForestUnitNotFound
	}
	return cell, err
}

func (s *Store) SearchCells(ctx context.Context, query string, limit int) ([]Cell, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 20 {
		limit = 20
	}
	query = strings.TrimSpace(query)

	var count int
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM forest_units`).Scan(&count); err != nil {
		return nil, fmt.Errorf("count forest units: %w", err)
	}
	table := "forest_units"
	if count == 0 {
		table = "cells"
	}

	var (
		rows pgx.Rows
		err  error
	)
	selectList := `
		id,
		name,
		ST_X(ST_Transform(ST_Centroid(geom), 4326)),
		ST_Y(ST_Transform(ST_Centroid(geom), 4326)),
		ST_XMin(ST_Transform(geom, 4326)),
		ST_YMin(ST_Transform(geom, 4326)),
		ST_XMax(ST_Transform(geom, 4326)),
		ST_YMax(ST_Transform(geom, 4326)),
		habitat_fit
	`
	if query == "" {
		rows, err = s.db.Query(ctx, fmt.Sprintf(`
			SELECT %s FROM %s
			ORDER BY name
			LIMIT $1
		`, selectList, table), limit)
	} else {
		pattern := "%" + query + "%"
		rows, err = s.db.Query(ctx, fmt.Sprintf(`
			SELECT %s FROM %s
			WHERE id = $1 OR name ILIKE $2
			ORDER BY
				CASE WHEN id = $1 THEN 0 ELSE 1 END,
				name
			LIMIT $3
		`, selectList, table), query, pattern, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("search cells: %w", err)
	}
	defer rows.Close()

	out := make([]Cell, 0, limit)
	for rows.Next() {
		var cell Cell
		if err := rows.Scan(
			&cell.ID, &cell.Name, &cell.Lon, &cell.Lat,
			&cell.West, &cell.South, &cell.East, &cell.North, &cell.HabitatFit,
		); err != nil {
			return nil, fmt.Errorf("scan cell: %w", err)
		}
		out = append(out, cell)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cells: %w", err)
	}
	return out, nil
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

func (s *Store) FactorsV2(ctx context.Context, cellID, targetDate string) (FactorSnapshotV2, error) {
	warsaw, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		return FactorSnapshotV2{}, fmt.Errorf("load Europe/Warsaw: %w", err)
	}
	day, err := time.ParseInLocation(time.DateOnly, targetDate, warsaw)
	if err != nil {
		return FactorSnapshotV2{}, fmt.Errorf("target_date: %w", err)
	}
	dayEnd := day.AddDate(0, 0, 1)
	window14dStart := dayEnd.AddDate(0, 0, -14)
	window3dStart := dayEnd.AddDate(0, 0, -3)

	var snap FactorSnapshotV2
	err = s.db.QueryRow(ctx, `
		SELECT
			(SELECT habitat_fit FROM cells WHERE id = $1),
			(SELECT SUM(precipitation_mm) FROM weather_samples WHERE cell_id = $1 AND precipitation_mm IS NOT NULL AND valid_at >= $2 AND valid_at < $4),
			(SELECT SUM(precipitation_mm) FROM weather_samples WHERE cell_id = $1 AND precipitation_mm IS NOT NULL AND valid_at >= $3 AND valid_at < $4),
			(SELECT AVG(soil_temperature_6cm_c) FROM weather_samples WHERE cell_id = $1 AND soil_temperature_6cm_c IS NOT NULL AND valid_at >= $5 AND valid_at < $4),
			(SELECT AVG(soil_moisture_3_to_9cm_m3_m3) FROM weather_samples WHERE cell_id = $1 AND soil_moisture_3_to_9cm_m3_m3 IS NOT NULL AND valid_at >= $5 AND valid_at < $4),
			(SELECT MIN(air_temperature_2m_c) FROM weather_samples WHERE cell_id = $1 AND air_temperature_2m_c IS NOT NULL AND valid_at >= $5 AND valid_at < $4)
	`, cellID, window14dStart, window3dStart, dayEnd, day).Scan(
		&snap.HabitatFit,
		&snap.Precipitation14dMM,
		&snap.Precipitation3dMM,
		&snap.SoilTemperatureC,
		&snap.SoilMoistureM3M3,
		&snap.AirTempMin2mC,
	)
	if err != nil {
		return FactorSnapshotV2{}, fmt.Errorf("load weather factors v2: %w", err)
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
