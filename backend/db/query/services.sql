-- name: CreateService :one
INSERT INTO services (establishment_id, name, price_cents, duration_minutes)
VALUES ($1, $2, $3, $4)
RETURNING id, establishment_id, name, price_cents, duration_minutes, active, created_at, updated_at;

-- name: ListServices :many
SELECT id, establishment_id, name, price_cents, duration_minutes, active, created_at, updated_at
FROM services
WHERE establishment_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2 OFFSET $3;

-- name: GetService :one
SELECT id, establishment_id, name, price_cents, duration_minutes, active, created_at, updated_at
FROM services
WHERE id = $1 AND establishment_id = $2;

-- name: UpdateService :one
UPDATE services SET
    name = CASE WHEN sqlc.arg(set_name)::boolean THEN sqlc.arg(name)::text ELSE name END,
    price_cents = CASE WHEN sqlc.arg(set_price)::boolean THEN sqlc.arg(price_cents)::integer ELSE price_cents END,
    duration_minutes = CASE WHEN sqlc.arg(set_duration)::boolean THEN sqlc.arg(duration_minutes)::integer ELSE duration_minutes END,
    updated_at = statement_timestamp()
WHERE id = sqlc.arg(id) AND establishment_id = sqlc.arg(establishment_id)
RETURNING id, establishment_id, name, price_cents, duration_minutes, active, created_at, updated_at;

-- name: SetServiceStatus :one
UPDATE services SET
    updated_at = CASE WHEN active IS DISTINCT FROM sqlc.arg(active)::boolean THEN statement_timestamp() ELSE updated_at END,
    active = sqlc.arg(active)::boolean
WHERE id = sqlc.arg(id) AND establishment_id = sqlc.arg(establishment_id)
RETURNING id, establishment_id, name, price_cents, duration_minutes, active, created_at, updated_at;
