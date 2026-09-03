-- +goose Up
CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE cells (
    id text PRIMARY KEY,
    name text NOT NULL,
    geom geometry(Polygon, 2180) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT cells_valid_geometry CHECK (ST_IsValid(geom))
);

CREATE INDEX cells_geom_gix ON cells USING gist (geom);

CREATE TABLE ingest_runs (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    cell_id text NOT NULL REFERENCES cells(id),
    provider text NOT NULL,
    requested_model text NOT NULL,
    request_url text NOT NULL,
    response_sha256 character(64) NOT NULL,
    source_latitude double precision,
    source_longitude double precision,
    source_elevation_m double precision,
    fetched_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ingest_runs_response_hash_format
        CHECK (response_sha256 ~ '^[0-9a-f]{64}$'),
    UNIQUE (cell_id, provider, response_sha256)
);

CREATE TABLE weather_samples (
    cell_id text NOT NULL REFERENCES cells(id),
    valid_at timestamptz NOT NULL,
    data_kind text NOT NULL,
    precipitation_mm double precision,
    soil_temperature_6cm_c double precision,
    soil_moisture_3_to_9cm_m3_m3 double precision,
    ingest_run_id bigint NOT NULL REFERENCES ingest_runs(id),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (cell_id, valid_at),
    CONSTRAINT weather_samples_data_kind
        CHECK (data_kind IN ('past', 'forecast')),
    CONSTRAINT weather_samples_precipitation
        CHECK (precipitation_mm IS NULL OR precipitation_mm >= 0),
    CONSTRAINT weather_samples_soil_moisture
        CHECK (
            soil_moisture_3_to_9cm_m3_m3 IS NULL
            OR soil_moisture_3_to_9cm_m3_m3 BETWEEN 0 AND 1
        )
);

CREATE INDEX weather_samples_cell_time_idx
    ON weather_samples (cell_id, valid_at DESC);

CREATE TABLE condition_scores (
    cell_id text NOT NULL REFERENCES cells(id),
    species_slug text NOT NULL,
    target_date date NOT NULL,
    status text NOT NULL,
    score smallint,
    confidence text NOT NULL,
    factors jsonb NOT NULL,
    algorithm_version text NOT NULL,
    input_sha256 character(64),
    calculated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (cell_id, species_slug, target_date, algorithm_version),
    CONSTRAINT condition_scores_status
        CHECK (status IN ('ready', 'unavailable')),
    CONSTRAINT condition_scores_value
        CHECK (
            (status = 'ready' AND score BETWEEN 0 AND 100)
            OR (status = 'unavailable' AND score IS NULL)
        ),
    CONSTRAINT condition_scores_confidence
        CHECK (confidence IN ('low', 'medium', 'high')),
    CONSTRAINT condition_scores_input_hash_format
        CHECK (input_sha256 IS NULL OR input_sha256 ~ '^[0-9a-f]{64}$')
);

-- +goose Down
DROP TABLE IF EXISTS condition_scores;
DROP TABLE IF EXISTS weather_samples;
DROP TABLE IF EXISTS ingest_runs;
DROP TABLE IF EXISTS cells;
DROP EXTENSION IF EXISTS postgis;
