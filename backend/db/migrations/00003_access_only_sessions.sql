-- +goose Up
ALTER TABLE auth_sessions RENAME COLUMN idle_expires_at TO expires_at;
ALTER TABLE auth_sessions RENAME CONSTRAINT auth_sessions_expiry_after_creation TO auth_sessions_expires_after_creation;
ALTER INDEX auth_sessions_active_expiry_idx RENAME TO auth_sessions_active_expires_idx;
DROP TABLE refresh_tokens;

-- +goose Down
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

ALTER TABLE auth_sessions RENAME COLUMN expires_at TO idle_expires_at;
ALTER TABLE auth_sessions RENAME CONSTRAINT auth_sessions_expires_after_creation TO auth_sessions_expiry_after_creation;
ALTER INDEX auth_sessions_active_expires_idx RENAME TO auth_sessions_active_expiry_idx;
