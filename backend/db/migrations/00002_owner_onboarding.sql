-- +goose Up
-- Existing minimal establishments cannot be completed without inventing product data.
-- Fail before changing the schema so operators can resolve them explicitly.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM establishments) THEN
        RAISE EXCEPTION 'owner onboarding migration requires an empty establishments table';
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE establishments
    ADD COLUMN name text NOT NULL,
    ADD COLUMN slug text NOT NULL,
    ADD COLUMN time_zone text NOT NULL,
    ADD CONSTRAINT establishments_slug_unique UNIQUE (slug);

CREATE TABLE email_confirmations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    token_hash bytea NOT NULL,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    invalidated_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT email_confirmations_token_hash_unique UNIQUE (token_hash),
    CONSTRAINT email_confirmations_token_hash_length CHECK (octet_length(token_hash) = 32),
    CONSTRAINT email_confirmations_expiry_after_creation CHECK (expires_at > created_at),
    CONSTRAINT email_confirmations_single_terminal_state CHECK (consumed_at IS NULL OR invalidated_at IS NULL),
    CONSTRAINT email_confirmations_consumed_after_creation CHECK (consumed_at IS NULL OR consumed_at >= created_at),
    CONSTRAINT email_confirmations_invalidated_after_creation CHECK (invalidated_at IS NULL OR invalidated_at >= created_at)
);

CREATE UNIQUE INDEX email_confirmations_one_active_per_user_idx
    ON email_confirmations (user_id)
    WHERE consumed_at IS NULL AND invalidated_at IS NULL;
CREATE INDEX email_confirmations_active_expiry_idx
    ON email_confirmations (expires_at)
    WHERE consumed_at IS NULL AND invalidated_at IS NULL;

ALTER TABLE users
    ADD CONSTRAINT users_id_establishment_unique UNIQUE (id, establishment_id);

CREATE TABLE establishment_trials (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id uuid NOT NULL,
    establishment_id uuid NOT NULL REFERENCES establishments (id) ON DELETE RESTRICT,
    starts_at timestamptz NOT NULL,
    ends_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT establishment_trials_owner_unique UNIQUE (owner_user_id),
    CONSTRAINT establishment_trials_establishment_unique UNIQUE (establishment_id),
    CONSTRAINT establishment_trials_positive_duration CHECK (ends_at > starts_at),
    CONSTRAINT establishment_trials_owner_tenant_fk
        FOREIGN KEY (owner_user_id, establishment_id)
        REFERENCES users (id, establishment_id) ON DELETE RESTRICT
);

ALTER TABLE auth_login_attempts
    ADD COLUMN scope text NOT NULL DEFAULT 'login',
    ADD CONSTRAINT auth_login_attempts_scope_not_blank CHECK (btrim(scope) <> '');

DROP INDEX auth_login_attempts_lookup_idx;
CREATE INDEX auth_login_attempts_lookup_idx
    ON auth_login_attempts (scope, dimension, key_hash, attempted_at DESC);

-- +goose Down
DROP INDEX auth_login_attempts_lookup_idx;
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM auth_login_attempts WHERE scope <> 'login') THEN
        RAISE EXCEPTION 'cannot roll back onboarding while non-login rate limit events exist';
    END IF;
END
$$;
-- +goose StatementEnd
ALTER TABLE auth_login_attempts
    DROP CONSTRAINT auth_login_attempts_scope_not_blank,
    DROP COLUMN scope;
CREATE INDEX auth_login_attempts_lookup_idx
    ON auth_login_attempts (dimension, key_hash, attempted_at DESC);

DROP TABLE establishment_trials;
ALTER TABLE users DROP CONSTRAINT users_id_establishment_unique;
DROP TABLE email_confirmations;
ALTER TABLE establishments
    DROP CONSTRAINT establishments_slug_unique,
    DROP COLUMN time_zone,
    DROP COLUMN slug,
    DROP COLUMN name;
