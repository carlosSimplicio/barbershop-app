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

-- name: GetAuthUserByIDForUpdate :one
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
WHERE u.id = $1
FOR UPDATE OF u, c;

-- name: CreateAuthSession :one
INSERT INTO auth_sessions (id, user_id, expires_at)
VALUES ($1, $2, $3)
RETURNING id, user_id, expires_at, revoked_at, created_at, updated_at;

-- name: GetActiveSessionUser :one
SELECT
    u.id,
    u.establishment_id,
    u.email,
    u.normalized_email,
    u.role,
    u.active,
    u.email_confirmed_at,
    u.credential_version,
    u.password_change_required
FROM auth_sessions AS s
JOIN users AS u ON u.id = s.user_id
WHERE s.id = $1
  AND s.revoked_at IS NULL
  AND s.expires_at > $2;

-- name: RevokeAuthSession :exec
UPDATE auth_sessions
SET revoked_at = COALESCE(revoked_at, $2),
    updated_at = $2
WHERE id = $1
  AND user_id = $3
  AND revoked_at IS NULL;

-- name: RevokeAllUserSessions :exec
UPDATE auth_sessions
SET revoked_at = COALESCE(revoked_at, $2),
    updated_at = $2
WHERE user_id = $1;

-- name: UpdateUserPassword :one
WITH updated_user AS (
    UPDATE users
    SET password_change_required = false,
        credential_version = credential_version + 1,
        updated_at = $3
    WHERE id = $1
      AND credential_version = $4
      AND password_change_required = true
    RETURNING credential_version
), updated_credential AS (
    UPDATE user_credentials AS c
    SET password_hash = $2,
        updated_at = $3
    FROM updated_user
    WHERE c.user_id = $1
)
SELECT credential_version FROM updated_user;

-- name: LockAuthSessionUser :one
SELECT id FROM users WHERE id = $1 FOR UPDATE;
