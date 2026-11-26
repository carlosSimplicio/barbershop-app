-- name: GetServicesProfessional :one
SELECT p.user_id, p.name, u.active
FROM professionals AS p
JOIN users AS u ON u.id = p.user_id AND u.establishment_id = p.establishment_id
WHERE p.user_id = $1 AND p.establishment_id = $2 AND u.role = 'professional';

-- name: ListProfessionalOfferings :many
SELECT s.id, s.name, s.price_cents, s.duration_minutes, s.active,
    COALESCE(ps.active, false)::boolean AS associated
FROM services AS s
LEFT JOIN professional_services AS ps
    ON ps.establishment_id = s.establishment_id AND ps.service_id = s.id
    AND ps.professional_id = sqlc.arg(professional_id)
WHERE s.establishment_id = sqlc.arg(establishment_id)
ORDER BY s.created_at DESC, s.id DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);
