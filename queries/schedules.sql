-- name: CreateSchedule :one
INSERT INTO schedules (name, start_date, end_date, user_id, group_id, location_id,
    monday, tuesday, wednesday, thursday, friday, saturday)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: GetSchedule :one
SELECT * FROM schedules WHERE id = $1;

-- name: ListSchedules :many
SELECT s.*, g.name AS group_name, u.full_name AS user_name, l.name AS location_name
FROM schedules s
LEFT JOIN groups g ON g.id = s.group_id
LEFT JOIN users u ON u.id = s.user_id
LEFT JOIN locations l ON l.id = s.location_id
WHERE (sqlc.narg('user_id')::uuid IS NULL OR s.user_id = sqlc.narg('user_id'))
  AND (sqlc.narg('group_id')::uuid IS NULL OR s.group_id = sqlc.narg('group_id'))
  AND (sqlc.arg('search')::text = '' OR s.name ILIKE '%' || sqlc.arg('search') || '%')
ORDER BY s.start_date DESC, s.created_at DESC
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: CountSchedules :one
SELECT count(*) FROM schedules
WHERE (sqlc.narg('user_id')::uuid IS NULL OR user_id = sqlc.narg('user_id'))
  AND (sqlc.narg('group_id')::uuid IS NULL OR group_id = sqlc.narg('group_id'))
  AND (sqlc.arg('search')::text = '' OR name ILIKE '%' || sqlc.arg('search') || '%');

-- name: UpdateSchedule :one
UPDATE schedules SET name = $2, start_date = $3, end_date = $4, user_id = $5, group_id = $6,
    location_id = $7, monday = $8, tuesday = $9, wednesday = $10, thursday = $11,
    friday = $12, saturday = $13, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteSchedule :exec
DELETE FROM schedules WHERE id = $1;

-- name: ListOverlapCandidates :many
-- Schedules of the same target whose date range overlaps; weekday overlap is checked in the service.
SELECT * FROM schedules
WHERE id <> sqlc.arg('exclude_id')::uuid
  AND ((sqlc.narg('user_id')::uuid IS NOT NULL AND user_id = sqlc.narg('user_id'))
    OR (sqlc.narg('group_id')::uuid IS NOT NULL AND group_id = sqlc.narg('group_id')))
  AND start_date <= sqlc.arg('end_date') AND end_date >= sqlc.arg('start_date');

-- name: ListSchedulesForStudentOnDate :many
-- Personal schedules (user_id) sort before group schedules.
SELECT s.* FROM schedules s
WHERE s.start_date <= sqlc.arg('on_date') AND s.end_date >= sqlc.arg('on_date')
  AND (s.user_id = sqlc.arg('student_id')::uuid
       OR s.group_id = (SELECT group_id FROM users WHERE id = sqlc.arg('student_id')::uuid))
ORDER BY (s.user_id IS NULL), s.created_at;
