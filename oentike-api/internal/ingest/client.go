package ingest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const userAgent = "oentike-api/0.0.1 (mushroom-conditions ingest)"

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

func DefaultHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

func Fetch(ctx context.Context, client HTTPDoer, requestURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("open-meteo request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("open-meteo fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("read open-meteo body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("open-meteo HTTP %d", resp.StatusCode)
	}
	return body, nil
}
