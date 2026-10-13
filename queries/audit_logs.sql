-- name: CreateAuditLog :one
INSERT INTO audit_logs (actor_id, actor_role, action, entity_type, entity_id, changes, ip, user_agent)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetAuditLog :one
SELECT a.*, u.username AS actor_username, u.full_name AS actor_name
FROM audit_logs a
LEFT JOIN users u ON u.id = a.actor_id
WHERE a.id = $1;

-- name: ListAuditLogs :many
SELECT a.*, u.username AS actor_username, u.full_name AS actor_name
FROM audit_logs a
LEFT JOIN users u ON u.id = a.actor_id
WHERE (sqlc.narg('actor_id')::uuid IS NULL OR a.actor_id = sqlc.narg('actor_id'))
  AND (sqlc.arg('action')::text = '' OR a.action = sqlc.arg('action'))
  AND (sqlc.arg('entity_type')::text = '' OR a.entity_type = sqlc.arg('entity_type'))
  AND (sqlc.narg('from_ts')::timestamptz IS NULL OR a.created_at >= sqlc.narg('from_ts'))
  AND (sqlc.narg('to_ts')::timestamptz IS NULL OR a.created_at < sqlc.narg('to_ts'))
ORDER BY a.created_at DESC
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: CountAuditLogs :one
SELECT count(*) FROM audit_logs
WHERE (sqlc.narg('actor_id')::uuid IS NULL OR actor_id = sqlc.narg('actor_id'))
  AND (sqlc.arg('action')::text = '' OR action = sqlc.arg('action'))
  AND (sqlc.arg('entity_type')::text = '' OR entity_type = sqlc.arg('entity_type'))
  AND (sqlc.narg('from_ts')::timestamptz IS NULL OR created_at >= sqlc.narg('from_ts'))
  AND (sqlc.narg('to_ts')::timestamptz IS NULL OR created_at < sqlc.narg('to_ts'));
