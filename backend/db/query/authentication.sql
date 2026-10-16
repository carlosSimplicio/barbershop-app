-- name: GetAuthUserByNormalizedEmail :one
SELECT
    u.id,
    u.establishment_id,
    u.email,
    u.normalized_email,
    u.role,
    u.active,
    u.email_confirmed_at,
    u.credential_version,
    u.password_change_required,
    c.password_hash
FROM users AS u
JOIN user_credentials AS c ON c.user_id = u.id
WHERE u.normalized_email = $1;

-- name: GetAuthUserByID :one
SELECT
    u.id,
    u.establishment_id,
    u.email,
    u.normalized_email,
    u.role,
    u.active,
    u.email_confirmed_at,
    u.credential_version,
    u.password_change_required,
    c.password_hash
FROM users AS u
JOIN user_credentials AS c ON c.user_id = u.id
WHERE u.id = $1;

-- name: CreateAuthSession :one
INSERT INTO auth_sessions (user_id, idle_expires_at)
VALUES ($1, $2)
RETURNING *;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (session_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRefreshTokenForUpdate :one
SELECT
    rt.id AS token_id,
    rt.session_id,
    rt.expires_at AS token_expires_at,
    rt.consumed_at,
    rt.revoked_at AS token_revoked_at,
    s.user_id,
    s.idle_expires_at,
    s.revoked_at AS session_revoked_at
FROM refresh_tokens AS rt
JOIN auth_sessions AS s ON s.id = rt.session_id
WHERE rt.token_hash = $1
FOR UPDATE OF rt, s;

-- name: ConsumeRefreshToken :exec
UPDATE refresh_tokens
SET consumed_at = $2,
    replaced_by_id = $3
WHERE id = $1
  AND consumed_at IS NULL
  AND revoked_at IS NULL;

-- name: ExtendAuthSession :exec
UPDATE auth_sessions
SET idle_expires_at = $2,
    updated_at = $3
WHERE id = $1
  AND revoked_at IS NULL;

-- name: RevokeAuthSession :exec
UPDATE auth_sessions
SET revoked_at = COALESCE(revoked_at, $2),
    updated_at = $2
WHERE id = $1;

-- name: RevokeRefreshTokensBySession :exec
UPDATE refresh_tokens
SET revoked_at = COALESCE(revoked_at, $2)
WHERE session_id = $1;

-- name: RevokeAllUserSessions :exec
UPDATE auth_sessions
SET revoked_at = COALESCE(revoked_at, $2),
    updated_at = $2
WHERE user_id = $1;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens AS rt
SET revoked_at = COALESCE(rt.revoked_at, $2)
FROM auth_sessions AS s
WHERE rt.session_id = s.id
  AND s.user_id = $1;

-- name: UpdateUserPassword :one
WITH updated_credential AS (
    UPDATE user_credentials
    SET password_hash = $2,
        updated_at = $3
    WHERE user_id = $1
)
UPDATE users
SET password_change_required = false,
    credential_version = credential_version + 1,
    updated_at = $3
WHERE id = $1
RETURNING credential_version;

-- name: RecordLoginAttempt :exec
INSERT INTO auth_login_attempts (dimension, key_hash, attempted_at)
VALUES ($1, $2, $3);

-- name: CountLoginAttemptsSince :one
SELECT count(*)
FROM auth_login_attempts
WHERE dimension = $1
  AND key_hash = $2
  AND attempted_at >= $3;

-- name: DeleteLoginAttemptsBefore :exec
DELETE FROM auth_login_attempts
WHERE attempted_at < $1;
