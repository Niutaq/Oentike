package ingest

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	DefaultInterval = time.Hour
	MinInterval     = 50 * time.Minute
)

func LastFetched(ctx context.Context, db *pgxpool.Pool, cellID string) (time.Time, bool, error) {
	var fetched time.Time
	err := db.QueryRow(ctx, `
		SELECT fetched_at
		FROM ingest_runs
		WHERE cell_id = $1
		ORDER BY fetched_at DESC
		LIMIT 1
	`, cellID).Scan(&fetched)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, fmt.Errorf("last ingest: %w", err)
	}
	return fetched, true, nil
}

func Due(last time.Time, ok bool, now time.Time, minAge time.Duration) bool {
	if !ok {
		return true
	}
	if last.After(now) {
		return false
	}
	return now.Sub(last) >= minAge
}

func Loop(
	ctx context.Context,
	db *pgxpool.Pool,
	client HTTPDoer,
	baseURL, cellID string,
	interval time.Duration,
	now func() time.Time,
) {
	if interval <= 0 {
		interval = DefaultInterval
	}
	if cellID == "" {
		cellID = DefaultCellID
	}
	if now == nil {
		now = time.Now
	}

	run := func() {
		at := now()
		last, ok, err := LastFetched(ctx, db, cellID)
		if err != nil {
			log.Printf("ingest last fetch: %v", err)
			return
		}
		if !Due(last, ok, at, MinInterval) {
			return
		}

		jobCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		defer cancel()
		result, err := Run(jobCtx, db, client, baseURL, cellID, at)
		if err != nil {
			log.Printf("ingest: %v", err)
			return
		}
		log.Printf(
			"ingested %s hours=%d run_id=%d sha256=%s lat=%.6f lon=%.6f",
			result.CellID, result.Hours, result.RunID, result.SHA256, result.Latitude, result.Longitude,
		)
	}

	run()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}
