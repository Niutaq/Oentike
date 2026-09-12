-- +goose Up
ALTER TABLE forest_units
    ADD COLUMN habitat_fit NUMERIC(3,2) NOT NULL DEFAULT 0.30
    CONSTRAINT forest_units_habitat_fit_range CHECK (habitat_fit BETWEEN 0 AND 1);

ALTER TABLE cells
    ADD COLUMN habitat_fit NUMERIC(3,2) NOT NULL DEFAULT 0.30
    CONSTRAINT cells_habitat_fit_range CHECK (habitat_fit BETWEEN 0 AND 1);

-- +goose Down
ALTER TABLE cells DROP COLUMN IF EXISTS habitat_fit;
ALTER TABLE forest_units DROP COLUMN IF EXISTS habitat_fit;
