package ingest

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CellRefresher materializes weather for one analysis cell on demand.
type CellRefresher struct {
	DB      *pgxpool.Pool
	Client  HTTPDoer
	BaseURL string
	Now     func() time.Time
}

func (r *CellRefresher) Refresh(ctx context.Context, cellID string) (bool, error) {
	if r == nil || r.DB == nil {
		return false, fmt.Errorf("weather refresher not configured")
	}
	now := time.Now()
	if r.Now != nil {
		now = r.Now()
	}
	last, ok, err := LastFetched(ctx, r.DB, cellID)
	if err != nil {
		return false, err
	}
	if !Due(last, ok, now, MinInterval) {
		return false, nil
	}
	jobCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	_, err = Run(jobCtx, r.DB, r.Client, r.BaseURL, cellID, now)
	if err != nil {
		return false, err
	}
	return true, nil
}
