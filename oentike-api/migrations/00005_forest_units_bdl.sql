-- +goose Up
CREATE TABLE forest_units (
    id text PRIMARY KEY,
    name text NOT NULL,
    region_cd text,
    inspectorate_cd text,
    source text NOT NULL DEFAULT 'bdl-wfs',
    synced_at timestamptz NOT NULL DEFAULT now(),
    geom geometry(MultiPolygon, 2180) NOT NULL,
    CONSTRAINT forest_units_valid_geometry CHECK (ST_IsValid(geom))
);

CREATE INDEX forest_units_name_idx ON forest_units (name);
CREATE INDEX forest_units_geom_gix ON forest_units USING gist (geom);

-- Analysis cells may hold full BDL multipolygons after materialization.
ALTER TABLE cells
    ALTER COLUMN geom TYPE geometry(MultiPolygon, 2180)
    USING ST_Multi(geom);

-- +goose Down
ALTER TABLE cells
    ALTER COLUMN geom TYPE geometry(Polygon, 2180)
    USING ST_GeometryN(ST_Multi(geom), 1);

DROP TABLE IF EXISTS forest_units;
