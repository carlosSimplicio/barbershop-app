-- +goose Up
ALTER TABLE establishments
    ADD COLUMN operational_data_started_at timestamptz;

-- +goose Down
-- Removing a populated marker would let an establishment change its time zone
-- after operational history has already been created.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM establishments
        WHERE operational_data_started_at IS NOT NULL
    ) THEN
        RAISE EXCEPTION 'cannot roll back establishment operational data markers while markers exist';
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE establishments
    DROP COLUMN operational_data_started_at;
