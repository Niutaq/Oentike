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

func ingestOne(
	ctx context.Context,
	db *pgxpool.Pool,
	client HTTPDoer,
	baseURL, cellID string,
	now time.Time,
) {
	last, ok, err := LastFetched(ctx, db, cellID)
	if err != nil {
		log.Printf("ingest last fetch %s: %v", cellID, err)
		return
	}
	if !Due(last, ok, now, MinInterval) {
		return
	}

	jobCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	result, err := Run(jobCtx, db, client, baseURL, cellID, now)
	if err != nil {
		log.Printf("ingest %s: %v", cellID, err)
		return
	}
	log.Printf(
		"ingested %s hours=%d run_id=%d sha256=%s lat=%.6f lon=%.6f",
		result.CellID, result.Hours, result.RunID, result.SHA256, result.Latitude, result.Longitude,
	)
}

// Loop refreshes every seeded cell on an hourly cadence (per-cell min age).
func Loop(
	ctx context.Context,
	db *pgxpool.Pool,
	client HTTPDoer,
	baseURL string,
	interval time.Duration,
	now func() time.Time,
) {
	if interval <= 0 {
		interval = DefaultInterval
	}
	if now == nil {
		now = time.Now
	}

	runAll := func() {
		at := now()
		ids, err := ListCellIDs(ctx, db)
		if err != nil {
			log.Printf("ingest list cells: %v", err)
			return
		}
		if len(ids) == 0 {
			log.Printf("ingest: no cells in database")
			return
		}
		for _, cellID := range ids {
			ingestOne(ctx, db, client, baseURL, cellID, at)
		}
	}

	runAll()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runAll()
		}
	}
}

// RunAll fetches Open-Meteo for every cell (ignores MinInterval).
func RunAll(ctx context.Context, db *pgxpool.Pool, client HTTPDoer, baseURL string, now time.Time) error {
	ids, err := ListCellIDs(ctx, db)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return fmt.Errorf("no cells to ingest")
	}
	var first error
	for _, cellID := range ids {
		result, err := Run(ctx, db, client, baseURL, cellID, now)
		if err != nil {
			log.Printf("ingest %s: %v", cellID, err)
			if first == nil {
				first = err
			}
			continue
		}
		log.Printf(
			"ingested %s hours=%d run_id=%d sha256=%s lat=%.6f lon=%.6f",
			result.CellID, result.Hours, result.RunID, result.SHA256, result.Latitude, result.Longitude,
		)
	}
	return first
}
