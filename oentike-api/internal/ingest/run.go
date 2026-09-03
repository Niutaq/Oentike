package ingest

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const DefaultCellID = "lasy-janowskie-01"

type Result struct {
	CellID     string
	RunID      int64
	Hours      int
	SHA256     string
	RequestURL string
	Latitude   float64
	Longitude  float64
}

func Run(ctx context.Context, db *pgxpool.Pool, client HTTPDoer, baseURL, cellID string, now time.Time) (Result, error) {
	if cellID == "" {
		cellID = DefaultCellID
	}

	cell, err := CellPointByID(ctx, db, cellID)
	if err != nil {
		return Result{}, err
	}

	requestURL, err := ForecastURL(baseURL, cell.Latitude, cell.Longitude)
	if err != nil {
		return Result{}, err
	}

	body, err := Fetch(ctx, client, requestURL)
	if err != nil {
		return Result{}, err
	}

	warsaw, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		return Result{}, fmt.Errorf("load Europe/Warsaw: %w", err)
	}

	forecast, err := ParseForecast(body, requestURL, now, warsaw)
	if err != nil {
		return Result{}, err
	}

	runID, err := Save(ctx, db, cell.ID, now, forecast)
	if err != nil {
		return Result{}, err
	}

	return Result{
		CellID:     cell.ID,
		RunID:      runID,
		Hours:      len(forecast.Samples),
		SHA256:     forecast.SHA256,
		RequestURL: requestURL,
		Latitude:   cell.Latitude,
		Longitude:  cell.Longitude,
	}, nil
}
