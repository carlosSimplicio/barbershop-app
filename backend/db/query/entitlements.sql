-- name: GetTrialWindowByEstablishment :one
SELECT starts_at, ends_at
FROM establishment_trials
WHERE establishment_id = $1;
