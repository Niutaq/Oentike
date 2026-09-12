-- +goose Up
ALTER TABLE weather_samples
    ADD COLUMN air_temperature_2m_c double precision;

-- +goose Down
ALTER TABLE weather_samples DROP COLUMN IF EXISTS air_temperature_2m_c;
