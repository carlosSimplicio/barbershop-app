-- name: ListProfessionalWorkPeriods :many
SELECT weekday, start_minute, end_minute
FROM professional_work_periods
WHERE establishment_id = $1 AND professional_id = $2
ORDER BY weekday, start_minute;
