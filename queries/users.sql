-- name: CreateUser :one
INSERT INTO users (username, password_hash, role, is_active, must_change_password,
    first_name, second_name, third_name, full_name, short_name, phone, group_id, faculty_id, permissions)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, COALESCE(sqlc.narg('permissions')::text[], '{}'))
RETURNING *;

-- name: SetUserPermissions :one
UPDATE users SET permissions = $2, updated_at = now() WHERE id = $1 RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1;

-- name: GetUserByRoleHemisID :one
SELECT * FROM users WHERE role = $1 AND hemis_id = $2;

-- name: GetUserByHemisLogin :one
SELECT * FROM users WHERE hemis_login = $1;

-- name: SetUserHemisLogin :exec
UPDATE users SET hemis_login = $2, updated_at = now() WHERE id = $1;

-- name: ListUsers :many
SELECT u.*, g.name AS group_name, f.name AS faculty_name
FROM users u
LEFT JOIN groups g ON g.id = u.group_id
LEFT JOIN faculties f ON f.id = u.faculty_id
WHERE (sqlc.narg('role')::text IS NULL OR u.role = sqlc.narg('role'))
  AND (sqlc.narg('faculty_id')::uuid IS NULL OR u.faculty_id = sqlc.narg('faculty_id'))
  AND (sqlc.narg('group_id')::uuid IS NULL OR u.group_id = sqlc.narg('group_id'))
  AND (sqlc.narg('is_active')::bool IS NULL OR u.is_active = sqlc.narg('is_active'))
  AND (sqlc.arg('search')::text = '' OR u.full_name ILIKE '%' || sqlc.arg('search') || '%'
       OR u.username ILIKE '%' || sqlc.arg('search') || '%')
ORDER BY u.full_name, u.username
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: CountUsers :one
SELECT count(*) FROM users
WHERE (sqlc.narg('role')::text IS NULL OR role = sqlc.narg('role'))
  AND (sqlc.narg('faculty_id')::uuid IS NULL OR faculty_id = sqlc.narg('faculty_id'))
  AND (sqlc.narg('group_id')::uuid IS NULL OR group_id = sqlc.narg('group_id'))
  AND (sqlc.narg('is_active')::bool IS NULL OR is_active = sqlc.narg('is_active'))
  AND (sqlc.arg('search')::text = '' OR full_name ILIKE '%' || sqlc.arg('search') || '%'
       OR username ILIKE '%' || sqlc.arg('search') || '%');

-- name: UpdateUserProfile :one
UPDATE users SET
    first_name = $2, second_name = $3, third_name = $4, full_name = $5, short_name = $6,
    phone = $7, group_id = $8, faculty_id = $9, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SetUserActive :one
UPDATE users SET is_active = $2, updated_at = now() WHERE id = $1 RETURNING *;

-- name: SetUserPassword :exec
UPDATE users SET password_hash = $2, must_change_password = $3, updated_at = now() WHERE id = $1;

-- name: TouchUserLogin :exec
UPDATE users SET last_login_at = now() WHERE id = $1;

-- name: SetUserFace :exec
UPDATE users SET face_embedding = $2, face_model = $3, face_enrolled_at = now(), updated_at = now() WHERE id = $1;

-- name: CountUsersByRole :one
SELECT count(*) FROM users WHERE role = $1 AND is_active;
