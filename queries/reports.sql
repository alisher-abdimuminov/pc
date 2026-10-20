-- name: ListReportStudents :many
SELECT u.id, u.full_name, u.group_id, g.name AS group_name
FROM users u
LEFT JOIN groups g ON g.id = u.group_id
WHERE u.role = 'student' AND u.is_active
  AND (sqlc.narg('group_id')::uuid IS NULL OR u.group_id = sqlc.narg('group_id'))
  AND (sqlc.narg('student_id')::uuid IS NULL OR u.id = sqlc.narg('student_id'))
ORDER BY g.name NULLS LAST, u.full_name;

-- name: ListSchedulesInRange :many
SELECT * FROM schedules
WHERE start_date <= sqlc.arg('to_date')::date AND end_date >= sqlc.arg('from_date')::date;

-- name: ListAttendanceSummary :many
SELECT a.student_id, a.date, a.shift::int AS shift,
       (count(st.id) FILTER (WHERE st.success))::int AS steps_done,
       (count(st.id) FILTER (WHERE st.review = 'pending'))::int AS steps_pending
FROM attendances a
JOIN users u ON u.id = a.student_id
LEFT JOIN attendance_steps st ON st.attendance_id = a.id
WHERE a.date BETWEEN sqlc.arg('from_date')::date AND sqlc.arg('to_date')::date
  AND (sqlc.narg('group_id')::uuid IS NULL OR u.group_id = sqlc.narg('group_id'))
  AND (sqlc.narg('student_id')::uuid IS NULL OR a.student_id = sqlc.narg('student_id'))
GROUP BY a.id;

-- name: ListStudentAttempts :many
SELECT a.date, a.shift::int AS shift, st.step::int AS step, st.success AS step_success, st.review AS step_review,
       t.id AS attempt_id, t.review, t.attempted_at, t.image, t.face_verified, t.location_verified,
       t.liveness_verified, t.success, t.error, t.face_distance, t.face_threshold,
       t.antispoof_score, t.gps_accuracy, t.latitude, t.longitude
FROM attendances a
JOIN attendance_steps st ON st.attendance_id = a.id
JOIN attendance_attempts t ON t.step_id = st.id
WHERE a.student_id = sqlc.arg('student_id')::uuid
  AND a.date BETWEEN sqlc.arg('from_date')::date AND sqlc.arg('to_date')::date
ORDER BY a.date DESC, st.step, t.attempted_at;

-- name: GetAttemptOwner :one
SELECT t.id, t.image, t.review, a.student_id, u.group_id, g.teacher_id
FROM attendance_attempts t
JOIN attendance_steps st ON st.id = t.step_id
JOIN attendances a ON a.id = st.attendance_id
JOIN users u ON u.id = a.student_id
LEFT JOIN groups g ON g.id = u.group_id
WHERE t.id = $1;
