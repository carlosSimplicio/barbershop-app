-- +goose Up
CREATE TABLE professional_work_periods (
    establishment_id uuid NOT NULL,
    professional_id uuid NOT NULL,
    weekday smallint NOT NULL CHECK (weekday BETWEEN 1 AND 7),
    start_minute smallint NOT NULL,
    end_minute smallint NOT NULL,
    CHECK (0 <= start_minute AND start_minute < end_minute AND end_minute <= 1440),
    PRIMARY KEY (establishment_id, professional_id, weekday, start_minute),
    CONSTRAINT professional_work_periods_professional_fk
        FOREIGN KEY (professional_id, establishment_id)
        REFERENCES professionals (user_id, establishment_id) ON DELETE RESTRICT
);

-- +goose Down
-- Refuse to discard configured schedules when rolling back the schema.
LOCK TABLE professional_work_periods IN ACCESS EXCLUSIVE MODE;
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM professional_work_periods) THEN
        RAISE EXCEPTION 'cannot roll back professional work periods while schedules exist';
    END IF;
END;
$$;
-- +goose StatementEnd
DROP TABLE professional_work_periods;
