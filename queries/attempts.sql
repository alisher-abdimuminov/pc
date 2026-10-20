-- name: EnsureAttendance :exec
INSERT INTO attendances (student_id, schedule_id, shift, date)
VALUES ($1, $2, $3, $4)
ON CONFLICT (student_id, schedule_id, date) DO NOTHING;

-- name: LockAttendance :one
SELECT * FROM attendances
WHERE student_id = $1 AND schedule_id = $2 AND date = $3
FOR UPDATE;

-- name: EnsureStep :exec
INSERT INTO attendance_steps (attendance_id, step) VALUES ($1, $2)
ON CONFLICT (attendance_id, step) DO NOTHING;

-- name: GetStep :one
SELECT * FROM attendance_steps WHERE attendance_id = $1 AND step = $2;

-- name: CountStepAttempts :one
SELECT count(*) FROM attendance_attempts WHERE step_id = $1;

-- name: CreateAttempt :one
INSERT INTO attendance_attempts (step_id, image, ip_address, user_agent, latitude, longitude, gps_accuracy,
    location_id, face_verified, location_verified, liveness_verified, success, error,
    face_distance, face_threshold, antispoof_score, review)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
RETURNING *;

-- name: MarkStepSuccess :exec
UPDATE attendance_steps SET success = true WHERE id = $1;

-- name: MarkStepPending :exec
-- The step counts as done for the student, but a human has to confirm it.
UPDATE attendance_steps SET success = true, review = 'pending' WHERE id = $1;

-- name: SetUserImage :exec
UPDATE users SET image = $2, updated_at = now() WHERE id = $1;

-- name: ListExpiredSelfies :many
SELECT id, image FROM attendance_attempts
WHERE image <> '' AND attempted_at < $1
ORDER BY attempted_at
LIMIT $2;

-- name: ClearAttemptImage :exec
UPDATE attendance_attempts SET image = '' WHERE id = $1;

-- name: LockAttendanceNoLock :one
SELECT * FROM attendances WHERE student_id = $1 AND schedule_id = $2 AND date = $3;

-- name: CountFaceMismatches :one
-- Attempts of one attendance where the face was real (liveness passed) but did not match the stored face.
SELECT count(*) FROM attendance_attempts t
JOIN attendance_steps st ON st.id = t.step_id
WHERE st.attendance_id = $1 AND t.liveness_verified AND NOT t.face_verified;

-- name: ListAttemptsAdmin :many
SELECT t.id, t.attempted_at, t.image, t.review, t.latitude, t.longitude, t.gps_accuracy, t.face_verified, t.location_verified,
       t.liveness_verified, t.success, t.error, t.face_distance, t.face_threshold, t.antispoof_score, t.ip_address,
       a.date, a.shift::int AS shift, st.step::int AS step,
       u.id AS student_id, u.full_name AS student_name, u.username AS student_username, u.is_active AS student_active, u.group_id, g.name AS group_name
FROM attendance_attempts t
JOIN attendance_steps st ON st.id = t.step_id
JOIN attendances a ON a.id = st.attendance_id
JOIN users u ON u.id = a.student_id
LEFT JOIN groups g ON g.id = u.group_id
WHERE (sqlc.arg('search')::text = '' OR u.full_name ILIKE '%' || sqlc.arg('search') || '%' OR u.username ILIKE '%' || sqlc.arg('search') || '%')
  AND (sqlc.narg('group_id')::uuid IS NULL OR u.group_id = sqlc.narg('group_id'))
  AND (sqlc.narg('student_id')::uuid IS NULL OR u.id = sqlc.narg('student_id'))
  AND (sqlc.narg('success')::bool IS NULL OR t.success = sqlc.narg('success'))
  AND (sqlc.narg('from_ts')::timestamptz IS NULL OR t.attempted_at >= sqlc.narg('from_ts'))
  AND (sqlc.narg('to_ts')::timestamptz IS NULL OR t.attempted_at < sqlc.narg('to_ts'))
  AND (sqlc.arg('kind')::text = ''
       OR (sqlc.arg('kind') = 'location' AND NOT t.location_verified)
       OR (sqlc.arg('kind') = 'face' AND t.liveness_verified AND NOT t.face_verified)
       OR (sqlc.arg('kind') = 'liveness' AND NOT t.liveness_verified AND t.antispoof_score IS NOT NULL)
       OR (sqlc.arg('kind') = 'no_face' AND NOT t.liveness_verified AND t.antispoof_score IS NULL AND t.review = '')
       OR (sqlc.arg('kind') = 'pending' AND t.review = 'pending'))
ORDER BY t.attempted_at DESC
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: CountAttemptsAdmin :one
SELECT count(*) FROM attendance_attempts t
JOIN attendance_steps st ON st.id = t.step_id
JOIN attendances a ON a.id = st.attendance_id
JOIN users u ON u.id = a.student_id
WHERE (sqlc.arg('search')::text = '' OR u.full_name ILIKE '%' || sqlc.arg('search') || '%' OR u.username ILIKE '%' || sqlc.arg('search') || '%')
  AND (sqlc.narg('group_id')::uuid IS NULL OR u.group_id = sqlc.narg('group_id'))
  AND (sqlc.narg('student_id')::uuid IS NULL OR u.id = sqlc.narg('student_id'))
  AND (sqlc.narg('success')::bool IS NULL OR t.success = sqlc.narg('success'))
  AND (sqlc.narg('from_ts')::timestamptz IS NULL OR t.attempted_at >= sqlc.narg('from_ts'))
  AND (sqlc.narg('to_ts')::timestamptz IS NULL OR t.attempted_at < sqlc.narg('to_ts'))
  AND (sqlc.arg('kind')::text = ''
       OR (sqlc.arg('kind') = 'location' AND NOT t.location_verified)
       OR (sqlc.arg('kind') = 'face' AND t.liveness_verified AND NOT t.face_verified)
       OR (sqlc.arg('kind') = 'liveness' AND NOT t.liveness_verified AND t.antispoof_score IS NOT NULL)
       OR (sqlc.arg('kind') = 'no_face' AND NOT t.liveness_verified AND t.antispoof_score IS NULL AND t.review = '')
       OR (sqlc.arg('kind') = 'pending' AND t.review = 'pending'));

-- name: CreateSeedAttempt :one
-- Like CreateAttempt but with an explicit attempted_at (demo data only).
INSERT INTO attendance_attempts (step_id, image, ip_address, user_agent, latitude, longitude, gps_accuracy,
    location_id, face_verified, location_verified, liveness_verified, success, error,
    face_distance, face_threshold, antispoof_score, attempted_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
RETURNING id;

-- name: ListDemoAttemptImages :many
SELECT image FROM attendance_attempts WHERE user_agent = 'seed-demo' AND image <> '';

-- name: DeleteDemoAttendances :execrows
DELETE FROM attendances WHERE id IN (
    SELECT DISTINCT st.attendance_id FROM attendance_attempts t JOIN attendance_steps st ON st.id = t.step_id
    WHERE t.user_agent = 'seed-demo');

-- name: CountDemoAttempts :one
SELECT count(*) FROM attendance_attempts WHERE user_agent = 'seed-demo';

-- name: GetAttemptForDelete :one
SELECT t.id, t.step_id, t.image, t.success, t.attempted_at, a.student_id, a.date, st.step::int AS step
FROM attendance_attempts t
JOIN attendance_steps st ON st.id = t.step_id
JOIN attendances a ON a.id = st.attendance_id
WHERE t.id = $1
FOR UPDATE OF t;

-- name: DeleteAttempt :exec
DELETE FROM attendance_attempts WHERE id = $1;

-- name: RefreshStepSuccess :exec
-- Re-derives a step's success flag from its remaining attempts (used after an attempt is deleted).
UPDATE attendance_steps SET
    success = EXISTS (SELECT 1 FROM attendance_attempts x WHERE x.step_id = attendance_steps.id AND x.success),
    review = COALESCE((SELECT t.review FROM attendance_attempts t WHERE t.step_id = attendance_steps.id AND (t.success OR t.review = 'rejected') ORDER BY t.attempted_at DESC LIMIT 1), '')
WHERE attendance_steps.id = $1;

-- name: GetAttemptForReview :one
SELECT t.id, t.step_id, t.review, t.success, t.location_verified, a.student_id, a.date, st.step::int AS step, g.teacher_id
FROM attendance_attempts t
JOIN attendance_steps st ON st.id = t.step_id
JOIN attendances a ON a.id = st.attendance_id
JOIN users u ON u.id = a.student_id
LEFT JOIN groups g ON g.id = u.group_id
WHERE t.id = $1
FOR UPDATE OF t, st;

-- name: ReviewAttempt :exec
UPDATE attendance_attempts SET review = $2, success = $3, reviewed_by = $4, reviewed_at = now() WHERE id = $1;

-- name: SetStepReview :exec
UPDATE attendance_steps SET review = $2, success = $3 WHERE id = $1;
