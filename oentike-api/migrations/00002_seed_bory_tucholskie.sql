-- +goose Up
WITH center AS (
    SELECT ST_Transform(
        ST_SetSRID(ST_MakePoint(17.55, 53.81), 4326),
        2180
    ) AS point
)
INSERT INTO cells (id, name, geom)
SELECT
    'bory-tucholskie-01',
    'Bory Tucholskie 01',
    ST_MakeEnvelope(
        ST_X(point) - 5000,
        ST_Y(point) - 5000,
        ST_X(point) + 5000,
        ST_Y(point) + 5000,
        2180
    )
FROM center;

-- +goose Down
DELETE FROM cells WHERE id = 'bory-tucholskie-01';
