-- name: ListProfessionalWorkPeriods :many
SELECT weekday, start_minute, end_minute
FROM professional_work_periods
WHERE establishment_id = $1 AND professional_id = $2
ORDER BY weekday, start_minute;

-- name: GetProfessionalScheduleTimeZone :one
SELECT e.time_zone
FROM professionals AS p
JOIN users AS u ON (u.id, u.establishment_id) = (p.user_id, p.establishment_id)
JOIN establishments AS e ON e.id = p.establishment_id
WHERE p.establishment_id = $1 AND p.user_id = $2 AND u.role = 'professional';
