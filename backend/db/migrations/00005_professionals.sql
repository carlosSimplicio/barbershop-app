-- +goose Up
CREATE TABLE professionals (
    user_id uuid PRIMARY KEY,
    establishment_id uuid NOT NULL,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT professionals_user_establishment_fk
        FOREIGN KEY (user_id, establishment_id)
        REFERENCES users (id, establishment_id) ON DELETE RESTRICT
);

CREATE INDEX professionals_establishment_created_idx
    ON professionals (establishment_id, created_at DESC, user_id DESC);

-- +goose Down
DROP TABLE professionals;
