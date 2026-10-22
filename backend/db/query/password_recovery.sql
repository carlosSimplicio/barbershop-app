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

-- name: GetPasswordRecoveryCandidate :one
SELECT user_id
FROM password_recovery_tokens
WHERE token_hash = $1;

-- name: GetPasswordRecoveryForUpdate :one
SELECT id, user_id, expires_at, consumed_at, invalidated_at
FROM password_recovery_tokens
WHERE token_hash = $1
FOR UPDATE;

-- name: ConsumePasswordRecoveryToken :execrows
UPDATE password_recovery_tokens
SET consumed_at = $3
WHERE id = $1
  AND user_id = $2
  AND expires_at > clock_timestamp()
  AND consumed_at IS NULL
  AND invalidated_at IS NULL;

-- name: AdvanceRecoveredOwnerCredentialVersion :execrows
UPDATE users
SET credential_version = credential_version + 1,
    updated_at = $2
WHERE id = $1
  AND role = 'owner'
  AND active = true
  AND email_confirmed_at IS NOT NULL;

-- name: UpdateRecoveredOwnerPasswordHash :execrows
UPDATE user_credentials
SET password_hash = $2,
    updated_at = $3
WHERE user_id = $1;
