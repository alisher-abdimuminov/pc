-- name: CreateLocation :one
INSERT INTO locations (name, polygon, tolerance_meters, location_url) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: GetLocation :one
SELECT * FROM locations WHERE id = $1;

-- name: ListLocations :many
SELECT * FROM locations
WHERE sqlc.arg('search')::text = '' OR name ILIKE '%' || sqlc.arg('search') || '%'
ORDER BY name
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: CountLocations :one
SELECT count(*) FROM locations
WHERE sqlc.arg('search')::text = '' OR name ILIKE '%' || sqlc.arg('search') || '%';

-- name: UpdateLocation :one
UPDATE locations SET name = $2, polygon = $3, tolerance_meters = $4, location_url = $5, updated_at = now()
WHERE id = $1 RETURNING *;

-- name: DeleteLocation :exec
DELETE FROM locations WHERE id = $1;
