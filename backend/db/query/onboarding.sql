-- name: CreateRegistrationEstablishment :one
INSERT INTO establishments (name, slug, time_zone, created_at)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: CreateRegistrationOwner :one
INSERT INTO users (
    establishment_id, email, normalized_email, role, active, email_confirmed_at,
    credential_version, password_change_required, created_at, updated_at
)
VALUES ($1, $2, $3, $4, true, NULL, 1, false, $5, $5)
RETURNING id;

-- name: CreateRegistrationCredential :exec
INSERT INTO user_credentials (user_id, password_hash, created_at, updated_at)
VALUES ($1, $2, $3, $3);

-- name: CreateRegistrationEmailConfirmation :exec
INSERT INTO email_confirmations (user_id, token_hash, expires_at, created_at)
VALUES ($1, $2, $3, $4);

-- name: InvalidateActiveEmailConfirmations :exec
UPDATE email_confirmations
SET invalidated_at = $2
WHERE user_id = $1
  AND consumed_at IS NULL
  AND invalidated_at IS NULL;

-- name: LockUserForEmailConfirmation :one
SELECT id
FROM users
WHERE id = $1
FOR UPDATE;

-- name: CreateEmailConfirmation :one
INSERT INTO email_confirmations (user_id, token_hash, expires_at, created_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetEmailConfirmationForUpdate :one
SELECT ec.*, u.establishment_id, u.email, u.role, u.active, u.email_confirmed_at, u.credential_version
FROM email_confirmations AS ec
JOIN users AS u ON u.id = ec.user_id
WHERE ec.token_hash = $1
FOR UPDATE OF ec, u;

-- name: ConsumeEmailConfirmation :execrows
UPDATE email_confirmations
SET consumed_at = $2
WHERE id = $1
  AND consumed_at IS NULL
  AND invalidated_at IS NULL;

-- name: ConfirmUserEmail :execrows
UPDATE users
SET email_confirmed_at = $2,
    updated_at = $2
WHERE id = $1
  AND email_confirmed_at IS NULL;

-- name: CreateEstablishmentTrial :one
INSERT INTO establishment_trials (owner_user_id, establishment_id, starts_at, ends_at, created_at)
SELECT
    u.id,
    u.establishment_id,
    sqlc.arg(starts_at)::timestamptz,
    sqlc.arg(ends_at)::timestamptz,
    sqlc.arg(starts_at)::timestamptz
FROM users AS u
WHERE u.id = sqlc.arg(owner_user_id)
  AND u.establishment_id = sqlc.arg(establishment_id)
  AND u.role = 'owner'
RETURNING *;

-- name: GetEstablishmentWithTrial :one
SELECT e.id, e.name, e.slug, e.time_zone, t.starts_at, t.ends_at
FROM establishments AS e
LEFT JOIN establishment_trials AS t ON t.establishment_id = e.id
WHERE e.id = $1;
