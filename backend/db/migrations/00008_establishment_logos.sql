-- +goose Up
CREATE TABLE establishment_logos (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    establishment_id uuid NOT NULL REFERENCES establishments (id) ON DELETE RESTRICT,
    object_key text NOT NULL UNIQUE,
    content_type text NOT NULL,
    byte_size bigint NOT NULL CHECK (byte_size > 0),
    width integer NOT NULL CHECK (width > 0),
    height integer NOT NULL CHECK (height > 0),
    content_sha256 bytea NOT NULL CHECK (octet_length(content_sha256) = 32),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (establishment_id, id)
);

ALTER TABLE establishments
    ADD COLUMN logo_id uuid,
    ADD CONSTRAINT establishments_current_logo_same_tenant_fk
        FOREIGN KEY (id, logo_id)
        REFERENCES establishment_logos (establishment_id, id)
        ON DELETE RESTRICT;

-- +goose Down
-- Logos are referenced files, and there is no backup for their metadata or
-- stored objects. Refuse rollback once any metadata or association exists.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM establishment_logos)
       OR EXISTS (SELECT 1 FROM establishments WHERE logo_id IS NOT NULL) THEN
        RAISE EXCEPTION 'cannot roll back establishment logos while logo metadata exists';
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE establishments
    DROP CONSTRAINT establishments_current_logo_same_tenant_fk,
    DROP COLUMN logo_id;
DROP TABLE establishment_logos;
