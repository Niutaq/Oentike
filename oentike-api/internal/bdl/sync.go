package bdl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	DefaultWFSURL = "https://wfs.bdl.lasy.gov.pl/geoserver/BDL/ows"
	typeName      = "BDL:Nadleśnictwa"
	pageSize      = 100
	userAgent     = "oentike-api/0.0.1 (bdl forest-units sync)"
)

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type featureCollection struct {
	Features []feature `json:"features"`
}

type feature struct {
	Properties struct {
		InspectorateName string `json:"inspectorate_name"`
		RegionCD         string `json:"region_cd"`
		InspectorateCD   string `json:"inspectorate_cd"`
	} `json:"properties"`
	Geometry json.RawMessage `json:"geometry"`
}

func DefaultHTTPClient() *http.Client {
	return &http.Client{Timeout: 60 * time.Second}
}

func UnitID(regionCD, inspectorateCD string) string {
	return fmt.Sprintf(
		"nadl-%s-%s",
		strings.TrimSpace(regionCD),
		strings.TrimSpace(inspectorateCD),
	)
}

func DisplayName(inspectorateName string) string {
	name := strings.TrimSpace(inspectorateName)
	if name == "" {
		return "Nadleśnictwo"
	}
	return "Nadleśnictwo " + name
}

// SyncNadlesnictwa pulls every BDL inspectorate into forest_units.
func SyncNadlesnictwa(ctx context.Context, db *pgxpool.Pool, client HTTPDoer, baseURL string) (int, error) {
	if baseURL == "" {
		baseURL = DefaultWFSURL
	}
	if client == nil {
		client = DefaultHTTPClient()
	}

	now := time.Now().UTC()
	upserted := 0
	for start := 0; ; start += pageSize {
		page, err := fetchPage(ctx, client, baseURL, start, pageSize)
		if err != nil {
			return upserted, err
		}
		if len(page.Features) == 0 {
			break
		}
		for _, feat := range page.Features {
			region := strings.TrimSpace(feat.Properties.RegionCD)
			code := strings.TrimSpace(feat.Properties.InspectorateCD)
			if region == "" || code == "" || len(feat.Geometry) == 0 {
				continue
			}
			id := UnitID(region, code)
			name := DisplayName(feat.Properties.InspectorateName)
			if err := upsertUnit(ctx, db, id, name, region, code, feat.Geometry, now); err != nil {
				return upserted, fmt.Errorf("upsert %s: %w", id, err)
			}
			upserted++
		}
		if len(page.Features) < pageSize {
			break
		}
	}
	log.Printf("bdl sync: upserted %d nadleśnictw", upserted)
	return upserted, nil
}

func fetchPage(ctx context.Context, client HTTPDoer, baseURL string, startIndex, count int) (featureCollection, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return featureCollection{}, fmt.Errorf("parse wfs url: %w", err)
	}
	q := u.Query()
	q.Set("service", "WFS")
	q.Set("version", "2.0.0")
	q.Set("request", "GetFeature")
	q.Set("typeNames", typeName)
	q.Set("outputFormat", "application/json")
	q.Set("srsName", "EPSG:4326")
	q.Set("count", fmt.Sprintf("%d", count))
	q.Set("startIndex", fmt.Sprintf("%d", startIndex))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return featureCollection{}, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return featureCollection{}, fmt.Errorf("wfs getfeature: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return featureCollection{}, fmt.Errorf("read wfs body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return featureCollection{}, fmt.Errorf("wfs status %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var page featureCollection
	if err := json.Unmarshal(body, &page); err != nil {
		return featureCollection{}, fmt.Errorf("decode wfs json: %w", err)
	}
	return page, nil
}

func upsertUnit(
	ctx context.Context,
	db *pgxpool.Pool,
	id, name, regionCD, inspectorateCD string,
	geometryJSON []byte,
	syncedAt time.Time,
) error {
	_, err := db.Exec(ctx, `
		INSERT INTO forest_units (id, name, region_cd, inspectorate_cd, source, synced_at, geom)
		VALUES (
			$1, $2, $3, $4, 'bdl-wfs', $5,
			ST_Multi(
				ST_CollectionExtract(
					ST_MakeValid(
						ST_Transform(
							ST_SetSRID(ST_GeomFromGeoJSON($6::text), 4326),
							2180
						)
					),
					3
				)
			)
		)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			region_cd = EXCLUDED.region_cd,
			inspectorate_cd = EXCLUDED.inspectorate_cd,
			source = EXCLUDED.source,
			synced_at = EXCLUDED.synced_at,
			geom = EXCLUDED.geom
	`, id, name, regionCD, inspectorateCD, syncedAt, string(geometryJSON))
	return err
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
