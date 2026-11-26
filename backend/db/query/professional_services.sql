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

-- name: SetProfessionalOffering :one
INSERT INTO professional_services (establishment_id, professional_id, service_id, active)
SELECT sqlc.arg(establishment_id), sqlc.arg(professional_id), sqlc.arg(service_id), sqlc.arg(associated)::boolean
WHERE sqlc.arg(associated)::boolean OR EXISTS (
    SELECT 1 FROM professional_services AS existing
    WHERE existing.establishment_id = sqlc.arg(establishment_id)
      AND existing.professional_id = sqlc.arg(professional_id)
      AND existing.service_id = sqlc.arg(service_id)
)
ON CONFLICT (establishment_id, professional_id, service_id)
DO UPDATE SET active = EXCLUDED.active,
    -- Evaluate after the conflict lock, keeping transition order even if the clock moves back.
    updated_at = CASE WHEN professional_services.active IS DISTINCT FROM EXCLUDED.active
        THEN GREATEST(clock_timestamp(), professional_services.updated_at + INTERVAL '1 microsecond')
        ELSE professional_services.updated_at END
RETURNING active;
