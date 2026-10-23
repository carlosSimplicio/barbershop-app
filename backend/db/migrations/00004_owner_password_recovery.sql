-- +goose Up
CREATE TABLE password_recovery_tokens (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    token_hash bytea NOT NULL,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    invalidated_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT password_recovery_tokens_hash_unique UNIQUE (token_hash),
    CONSTRAINT password_recovery_tokens_hash_length CHECK (octet_length(token_hash) = 32),
    CONSTRAINT password_recovery_tokens_expiry_after_creation CHECK (expires_at > created_at)
);

CREATE UNIQUE INDEX password_recovery_tokens_one_active_per_user_idx
    ON password_recovery_tokens (user_id)
    WHERE consumed_at IS NULL AND invalidated_at IS NULL;

CREATE INDEX password_recovery_tokens_expiry_idx
    ON password_recovery_tokens (expires_at);

-- +goose Down
DROP TABLE password_recovery_tokens;
