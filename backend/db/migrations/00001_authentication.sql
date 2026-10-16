-- +goose Up
CREATE TABLE establishments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    establishment_id uuid NOT NULL REFERENCES establishments (id),
    email text NOT NULL,
    normalized_email text NOT NULL,
    role text NOT NULL CHECK (role IN ('owner', 'professional')),
    active boolean NOT NULL DEFAULT true,
    email_confirmed_at timestamptz,
    credential_version bigint NOT NULL DEFAULT 1 CHECK (credential_version > 0),
    password_change_required boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_normalized_email_unique UNIQUE (normalized_email),
    CONSTRAINT users_email_not_blank CHECK (btrim(email) <> ''),
    CONSTRAINT users_normalized_email_not_blank CHECK (btrim(normalized_email) <> '')
);

CREATE INDEX users_establishment_id_idx ON users (establishment_id);

CREATE TABLE user_credentials (
    user_id uuid PRIMARY KEY REFERENCES users (id) ON DELETE RESTRICT,
    password_hash text NOT NULL CHECK (password_hash <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE auth_sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    idle_expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT auth_sessions_expiry_after_creation CHECK (idle_expires_at > created_at)
);

CREATE INDEX auth_sessions_user_id_idx ON auth_sessions (user_id);
CREATE INDEX auth_sessions_active_expiry_idx
    ON auth_sessions (idle_expires_at)
    WHERE revoked_at IS NULL;

CREATE TABLE refresh_tokens (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id uuid NOT NULL REFERENCES auth_sessions (id) ON DELETE RESTRICT,
    token_hash bytea NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    revoked_at timestamptz,
    replaced_by_id uuid REFERENCES refresh_tokens (id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT refresh_tokens_hash_not_empty CHECK (octet_length(token_hash) > 0),
    CONSTRAINT refresh_tokens_expiry_after_creation CHECK (expires_at > created_at),
    CONSTRAINT refresh_tokens_replacement_after_consumption CHECK (
        replaced_by_id IS NULL OR consumed_at IS NOT NULL
    )
);

CREATE INDEX refresh_tokens_session_id_idx ON refresh_tokens (session_id);
CREATE INDEX refresh_tokens_active_expiry_idx
    ON refresh_tokens (expires_at)
    WHERE consumed_at IS NULL AND revoked_at IS NULL;

CREATE TABLE auth_login_attempts (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    dimension text NOT NULL CHECK (dimension IN ('ip', 'email')),
    key_hash bytea NOT NULL,
    attempted_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT auth_login_attempts_key_hash_not_empty CHECK (octet_length(key_hash) > 0)
);

CREATE INDEX auth_login_attempts_lookup_idx
    ON auth_login_attempts (dimension, key_hash, attempted_at DESC);

-- +goose Down
DROP TABLE auth_login_attempts;
DROP TABLE refresh_tokens;
DROP TABLE auth_sessions;
DROP TABLE user_credentials;
DROP TABLE users;
DROP TABLE establishments;
