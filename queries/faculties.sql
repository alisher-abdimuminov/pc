-- name: CreateFaculty :one
INSERT INTO faculties (name, hemis_id) VALUES ($1, $2) RETURNING *;

-- name: GetFaculty :one
SELECT * FROM faculties WHERE id = $1;

-- name: ListFaculties :many
SELECT * FROM faculties
WHERE sqlc.arg('search')::text = '' OR name ILIKE '%' || sqlc.arg('search') || '%'
ORDER BY name
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: CountFaculties :one
SELECT count(*) FROM faculties
WHERE sqlc.arg('search')::text = '' OR name ILIKE '%' || sqlc.arg('search') || '%';

-- name: UpdateFaculty :one
UPDATE faculties SET name = $2 WHERE id = $1 RETURNING *;

-- name: DeleteFaculty :exec
DELETE FROM faculties WHERE id = $1;

-- name: GetFacultyByHemisID :one
SELECT * FROM faculties WHERE hemis_id = $1;

-- name: FindUnlinkedFacultyByName :one
-- A faculty created by hand (no HEMIS id yet) with exactly this name.
SELECT * FROM faculties WHERE hemis_id IS NULL AND name = $1 ORDER BY created_at LIMIT 1;

-- name: SetFacultyHemisID :exec
UPDATE faculties SET hemis_id = $2 WHERE id = $1;

-- name: RenameFaculty :exec
UPDATE faculties SET name = $2 WHERE id = $1;
