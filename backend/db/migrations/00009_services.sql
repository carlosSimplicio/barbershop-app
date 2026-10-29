-- +goose Up
CREATE TABLE services (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    establishment_id uuid NOT NULL REFERENCES establishments (id) ON DELETE RESTRICT,
    name text NOT NULL,
    price_cents integer NOT NULL,
    duration_minutes integer NOT NULL,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (id, establishment_id)
);
CREATE INDEX services_establishment_created_idx
    ON services (establishment_id, created_at DESC, id DESC);

-- +goose Down
DROP TABLE services;
