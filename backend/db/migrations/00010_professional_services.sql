-- +goose Up
ALTER TABLE professionals
    ADD CONSTRAINT professionals_user_establishment_unique UNIQUE (user_id, establishment_id);

CREATE TABLE professional_services (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    establishment_id uuid NOT NULL,
    professional_id uuid NOT NULL,
    service_id uuid NOT NULL,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT professional_services_pair_unique
        UNIQUE (establishment_id, professional_id, service_id),
    CONSTRAINT professional_services_professional_fk
        FOREIGN KEY (professional_id, establishment_id)
        REFERENCES professionals (user_id, establishment_id) ON DELETE RESTRICT,
    CONSTRAINT professional_services_service_fk
        FOREIGN KEY (service_id, establishment_id)
        REFERENCES services (id, establishment_id) ON DELETE RESTRICT
);

CREATE INDEX professional_services_service_idx
    ON professional_services (service_id, establishment_id);

-- +goose Down
DROP TABLE professional_services;
ALTER TABLE professionals DROP CONSTRAINT professionals_user_establishment_unique;
