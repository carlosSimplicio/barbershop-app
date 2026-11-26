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

-- name: GetProfessionalSchedule :many
SELECT e.time_zone,
       COALESCE(w.weekday, 0)::smallint AS weekday,
       COALESCE(w.start_minute, 0)::smallint AS start_minute,
       COALESCE(w.end_minute, 0)::smallint AS end_minute
FROM professionals AS p
JOIN users AS u ON (u.id, u.establishment_id) = (p.user_id, p.establishment_id)
JOIN establishments AS e ON e.id = p.establishment_id
LEFT JOIN professional_work_periods AS w
    ON (w.professional_id, w.establishment_id) = (p.user_id, p.establishment_id)
WHERE p.establishment_id = $1 AND p.user_id = $2 AND u.role = 'professional'
ORDER BY w.weekday, w.start_minute;

-- name: DeleteProfessionalWorkPeriods :exec
DELETE FROM professional_work_periods
WHERE establishment_id = $1 AND professional_id = $2;

-- name: InsertProfessionalWorkPeriods :exec
INSERT INTO professional_work_periods (establishment_id, professional_id, weekday, start_minute, end_minute)
SELECT sqlc.arg(establishment_id)::uuid, sqlc.arg(professional_id)::uuid,
       unnest(sqlc.arg(weekdays)::smallint[]),
       unnest(sqlc.arg(start_minutes)::smallint[]),
       unnest(sqlc.arg(end_minutes)::smallint[]);
