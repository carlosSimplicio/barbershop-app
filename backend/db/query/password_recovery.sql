-- name: FindEligibleOwnerForPasswordRecovery :one
SELECT id, email
FROM users
WHERE normalized_email = $1
  AND role = 'owner'
  AND active = true
  AND email_confirmed_at IS NOT NULL;

-- name: LockEligibleOwnerForPasswordRecovery :one
SELECT id
FROM users
WHERE id = $1
  AND role = 'owner'
  AND active = true
  AND email_confirmed_at IS NOT NULL
FOR UPDATE;

-- name: InvalidateActivePasswordRecoveryTokens :exec
UPDATE password_recovery_tokens
SET invalidated_at = $2
WHERE user_id = $1
  AND consumed_at IS NULL
  AND invalidated_at IS NULL;

-- name: CreatePasswordRecoveryToken :one
INSERT INTO password_recovery_tokens (user_id, token_hash, expires_at, created_at)
VALUES ($1, $2, $3, $4)
RETURNING id;
