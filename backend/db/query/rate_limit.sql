-- The table name is retained for migration compatibility; its scoped events are shared by protected flows.

-- name: RecordRateLimitEvent :one
INSERT INTO auth_login_attempts (scope, dimension, key_hash, attempted_at)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: ListRateLimitEventsSince :many
SELECT attempted_at
FROM auth_login_attempts
WHERE scope = $1
  AND dimension = $2
  AND key_hash = $3
  AND attempted_at >= $4
ORDER BY attempted_at DESC;

-- name: DeleteRateLimitEventsBefore :exec
DELETE FROM auth_login_attempts
WHERE attempted_at < $1;

-- name: DeleteRateLimitEventsByIDs :exec
DELETE FROM auth_login_attempts
WHERE id = ANY($1::bigint[]);
