-- +goose Up
DELETE FROM cells WHERE id = 'bory-tucholskie-01';

WITH center AS (
    SELECT ST_Transform(
        ST_SetSRID(ST_MakePoint(22.189584, 50.60125), 4326),
        2180
    ) AS point
)
INSERT INTO cells (id, name, geom)
SELECT
    'lasy-janowskie-01',
    'Lasy Janowskie 01',
    ST_MakeEnvelope(
        ST_X(point) - 5000,
        ST_Y(point) - 5000,
        ST_X(point) + 5000,
        ST_Y(point) + 5000,
        2180
    )
FROM center
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    geom = EXCLUDED.geom;

-- +goose Down
DELETE FROM cells WHERE id = 'lasy-janowskie-01';
