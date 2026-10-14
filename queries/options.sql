-- Unpaginated id + name lists for select boxes.

-- name: ListFacultyOptions :many
SELECT id, name FROM faculties ORDER BY name;

-- name: ListGroupOptions :many
SELECT id, name FROM groups ORDER BY name;

-- name: ListLocationOptions :many
SELECT id, name FROM locations ORDER BY name;

-- name: ListScheduleOptions :many
SELECT id, name FROM schedules ORDER BY start_date DESC, created_at DESC;
