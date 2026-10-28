-- +goose Up
ALTER TABLE establishments
    ADD COLUMN description text,
    ADD COLUMN address text,
    ADD COLUMN contact_phone text,
    ADD COLUMN minimum_booking_notice_minutes integer,
    ADD COLUMN booking_horizon_days integer,
    ADD COLUMN customer_change_deadline_minutes integer;

CREATE TABLE establishment_social_links (
    establishment_id uuid NOT NULL REFERENCES establishments (id) ON DELETE RESTRICT,
    position integer NOT NULL,
    label text NOT NULL,
    url text NOT NULL,
    PRIMARY KEY (establishment_id, position)
);

-- +goose Down
-- Refuse to discard settings that may have been saved after this migration.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM establishments
        WHERE description IS NOT NULL
           OR address IS NOT NULL
           OR contact_phone IS NOT NULL
           OR minimum_booking_notice_minutes IS NOT NULL
           OR booking_horizon_days IS NOT NULL
           OR customer_change_deadline_minutes IS NOT NULL
    ) OR EXISTS (SELECT 1 FROM establishment_social_links) THEN
        RAISE EXCEPTION 'cannot roll back establishment settings while saved settings exist';
    END IF;
END
$$;
-- +goose StatementEnd

DROP TABLE establishment_social_links;
ALTER TABLE establishments
    DROP COLUMN description,
    DROP COLUMN address,
    DROP COLUMN contact_phone,
    DROP COLUMN minimum_booking_notice_minutes,
    DROP COLUMN booking_horizon_days,
    DROP COLUMN customer_change_deadline_minutes;
