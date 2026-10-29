-- name: CreateProfessionalUser :one
INSERT INTO users (
    establishment_id, email, normalized_email, role, active, email_confirmed_at,
    credential_version, password_change_required, created_at, updated_at
)
VALUES ($1, $2, $3, 'professional', true, NULL, 1, true, $4, $4)
RETURNING id;

-- name: CreateProfessionalCredential :exec
INSERT INTO user_credentials (user_id, password_hash, created_at, updated_at)
VALUES ($1, $2, $3, $3);

-- name: CreateProfessionalProfile :exec
INSERT INTO professionals (user_id, establishment_id, name, created_at, updated_at)
VALUES ($1, $2, $3, $4, $4);

-- name: ListProfessionalsByEstablishment :many
SELECT p.user_id, p.name, u.email, u.active
FROM professionals AS p
JOIN users AS u ON u.id = p.user_id AND u.establishment_id = p.establishment_id
WHERE p.establishment_id = $1
  AND u.role = 'professional'
ORDER BY p.created_at DESC, p.user_id DESC
LIMIT $2 OFFSET $3;

-- name: GetProfessional :one
SELECT p.user_id, p.name, u.email, u.active
FROM professionals AS p
JOIN users AS u ON u.id = p.user_id AND u.establishment_id = p.establishment_id
WHERE p.user_id = $1 AND p.establishment_id = $2 AND u.role = 'professional';

-- name: UpdateProfessionalName :one
WITH professional_user AS MATERIALIZED (
    SELECT u.id, u.establishment_id, u.email, u.active
    FROM users AS u
    WHERE u.id = $1 AND u.establishment_id = $2 AND u.role = 'professional'
    FOR UPDATE OF u
)
UPDATE professionals AS p
SET name = $3, updated_at = $4
FROM professional_user AS u
WHERE p.user_id = u.id AND p.establishment_id = u.establishment_id
RETURNING p.user_id, p.name, u.email, u.active;
