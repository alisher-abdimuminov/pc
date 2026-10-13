-- name: CreateGroup :one
INSERT INTO groups (name, hemis_id, faculty_id, teacher_id) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: GetGroup :one
SELECT * FROM groups WHERE id = $1;

-- name: GetGroupForTeacher :one
SELECT * FROM groups WHERE id = $1 AND teacher_id = $2;

-- name: ListGroups :many
SELECT * FROM groups
WHERE (sqlc.arg('search')::text = '' OR name ILIKE '%' || sqlc.arg('search') || '%')
  AND (sqlc.narg('teacher_id')::uuid IS NULL OR teacher_id = sqlc.narg('teacher_id'))
  AND (sqlc.narg('faculty_id')::uuid IS NULL OR faculty_id = sqlc.narg('faculty_id'))
ORDER BY name
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: CountGroups :one
SELECT count(*) FROM groups
WHERE (sqlc.arg('search')::text = '' OR name ILIKE '%' || sqlc.arg('search') || '%')
  AND (sqlc.narg('teacher_id')::uuid IS NULL OR teacher_id = sqlc.narg('teacher_id'))
  AND (sqlc.narg('faculty_id')::uuid IS NULL OR faculty_id = sqlc.narg('faculty_id'));

-- name: UpdateGroup :one
UPDATE groups SET name = $2, faculty_id = $3, teacher_id = $4 WHERE id = $1 RETURNING *;

-- name: DeleteGroup :exec
DELETE FROM groups WHERE id = $1;

-- name: ListGroupStudents :many
SELECT * FROM users WHERE group_id = $1 AND role = 'student' ORDER BY full_name;

-- name: ListGroupsAdmin :many
SELECT g.*, f.name AS faculty_name, t.full_name AS teacher_name,
    (SELECT count(*) FROM users s WHERE s.group_id = g.id AND s.role = 'student')::bigint AS student_count
FROM groups g
LEFT JOIN faculties f ON f.id = g.faculty_id
LEFT JOIN users t ON t.id = g.teacher_id
WHERE (sqlc.arg('search')::text = '' OR g.name ILIKE '%' || sqlc.arg('search') || '%')
  AND (sqlc.narg('faculty_id')::uuid IS NULL OR g.faculty_id = sqlc.narg('faculty_id'))
  AND (sqlc.narg('teacher_id')::uuid IS NULL OR g.teacher_id = sqlc.narg('teacher_id'))
ORDER BY g.name
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: GetGroupAdmin :one
SELECT g.*, f.name AS faculty_name, t.full_name AS teacher_name,
    (SELECT count(*) FROM users s WHERE s.group_id = g.id AND s.role = 'student')::bigint AS student_count
FROM groups g
LEFT JOIN faculties f ON f.id = g.faculty_id
LEFT JOIN users t ON t.id = g.teacher_id
WHERE g.id = $1;

-- name: GetGroupByHemisID :one
SELECT * FROM groups WHERE hemis_id = $1;

-- name: FindUnlinkedGroupByName :one
-- A group created by hand (no HEMIS id yet) with exactly this name.
SELECT * FROM groups WHERE hemis_id IS NULL AND name = $1 ORDER BY created_at LIMIT 1;

-- name: SetGroupHemisID :exec
UPDATE groups SET hemis_id = $2 WHERE id = $1;

-- name: RenameGroup :exec
UPDATE groups SET name = $2 WHERE id = $1;

-- name: SetGroupFaculty :exec
UPDATE groups SET faculty_id = $2 WHERE id = $1;
