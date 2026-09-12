-- +goose Up
-- Pilot cells across Poland (10 x 10 km). Names are searchable via SearchCells.

WITH centers AS (
    SELECT * FROM (VALUES
        ('lasy-janowskie-01', 'Lasy Janowskie 01', 22.189584, 50.60125),
        ('bory-tucholskie-01', 'Bory Tucholskie 01', 17.55, 53.81),
        ('puszcza-notecka-01', 'Puszcza Notecka 01', 16.15, 52.72),
        ('puszcza-niepolomicka-01', 'Puszcza Niepołomicka 01', 20.35, 50.03)
    ) AS t(id, name, lon, lat)
),
projected AS (
    SELECT
        id,
        name,
        ST_Transform(ST_SetSRID(ST_MakePoint(lon, lat), 4326), 2180) AS point
    FROM centers
)
INSERT INTO cells (id, name, geom)
SELECT
    id,
    name,
    ST_MakeEnvelope(
        ST_X(point) - 5000,
        ST_Y(point) - 5000,
        ST_X(point) + 5000,
        ST_Y(point) + 5000,
        2180
    )
FROM projected
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    geom = EXCLUDED.geom;

-- +goose Down
DELETE FROM cells WHERE id IN (
    'bory-tucholskie-01',
    'puszcza-notecka-01',
    'puszcza-niepolomicka-01'
);
-- Keep lasy-janowskie-01 (seeded earlier).
