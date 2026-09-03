package ingest

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"oentike-api/internal/conditions"
)

type CellPoint struct {
	ID        string
	Name      string
	Latitude  float64
	Longitude float64
}

func CellPointByID(ctx context.Context, db *pgxpool.Pool, id string) (CellPoint, error) {
	var cell CellPoint
	err := db.QueryRow(ctx, `
		SELECT
			id,
			name,
			ST_Y(ST_Transform(ST_Centroid(geom), 4326)),
			ST_X(ST_Transform(ST_Centroid(geom), 4326))
		FROM cells
		WHERE id = $1
	`, id).Scan(&cell.ID, &cell.Name, &cell.Latitude, &cell.Longitude)
	if errors.Is(err, pgx.ErrNoRows) {
		return CellPoint{}, conditions.ErrCellNotFound
	}
	if err != nil {
		return CellPoint{}, fmt.Errorf("load cell centroid: %w", err)
	}
	return cell, nil
}

func Save(ctx context.Context, db *pgxpool.Pool, cellID string, fetchedAt time.Time, forecast Forecast) (int64, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin ingest: %w", err)
	}
	defer tx.Rollback(ctx)

	var runID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO ingest_runs (
			cell_id, provider, requested_model, request_url, response_sha256,
			source_latitude, source_longitude, source_elevation_m, fetched_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (cell_id, provider, response_sha256) DO UPDATE
			SET fetched_at = EXCLUDED.fetched_at
		RETURNING id
	`, cellID, Provider, RequestedModel, forecast.RequestURL, forecast.SHA256,
		forecast.Latitude, forecast.Longitude, forecast.ElevationM, fetchedAt,
	).Scan(&runID)
	if err != nil {
		return 0, fmt.Errorf("insert ingest_run: %w", err)
	}

	batch := &pgx.Batch{}
	for _, sample := range forecast.Samples {
		batch.Queue(`
			INSERT INTO weather_samples (
				cell_id, valid_at, data_kind,
				precipitation_mm, soil_temperature_6cm_c, soil_moisture_3_to_9cm_m3_m3,
				ingest_run_id
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (cell_id, valid_at) DO UPDATE SET
				data_kind = EXCLUDED.data_kind,
				precipitation_mm = EXCLUDED.precipitation_mm,
				soil_temperature_6cm_c = EXCLUDED.soil_temperature_6cm_c,
				soil_moisture_3_to_9cm_m3_m3 = EXCLUDED.soil_moisture_3_to_9cm_m3_m3,
				ingest_run_id = EXCLUDED.ingest_run_id,
				updated_at = now()
		`, cellID, sample.ValidAt, sample.DataKind,
			sample.PrecipitationMM, sample.SoilTemperature6cmC, sample.SoilMoisture3To9cmM3,
			runID)
	}
	results := tx.SendBatch(ctx, batch)
	for range forecast.Samples {
		if _, err := results.Exec(); err != nil {
			_ = results.Close()
			return 0, fmt.Errorf("upsert weather_sample: %w", err)
		}
	}
	if err := results.Close(); err != nil {
		return 0, fmt.Errorf("close weather batch: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit ingest: %w", err)
	}
	return runID, nil
}
