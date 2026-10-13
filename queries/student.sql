-- name: GetTodayAttendance :one
SELECT * FROM attendances
WHERE student_id = sqlc.arg('student_id')::uuid AND date = sqlc.arg('on_date')::date
ORDER BY created_at DESC LIMIT 1;

-- name: ListStepsWithAttemptCounts :many
SELECT st.step::int AS step, st.success,
       (SELECT count(*) FROM attendance_attempts t WHERE t.step_id = st.id)::int AS attempts
FROM attendance_steps st
WHERE st.attendance_id = $1
ORDER BY st.step;

-- name: ListStudentAssignments :many
SELECT a.id, a.title, a.description, a.file, a.deadline, a.created_at,
       s.id AS submission_id, s.file AS submission_file, s.submitted_at,
       s.grade, s.feedback, s.graded_at
FROM assignments a
LEFT JOIN submissions s ON s.assignment_id = a.id AND s.student_id = sqlc.arg('student_id')::uuid
WHERE a.group_id = sqlc.arg('group_id')::uuid
ORDER BY a.deadline DESC;

-- name: GetStudentAssignment :one
SELECT a.id, a.title, a.description, a.file, a.deadline, a.created_at,
       s.id AS submission_id, s.file AS submission_file, s.submitted_at,
       s.grade, s.feedback, s.graded_at
FROM assignments a
LEFT JOIN submissions s ON s.assignment_id = a.id AND s.student_id = sqlc.arg('student_id')::uuid
WHERE a.id = sqlc.arg('assignment_id')::uuid AND a.group_id = sqlc.arg('group_id')::uuid;

-- name: GetSubmissionForUpdate :one
SELECT * FROM submissions WHERE assignment_id = $1 AND student_id = $2 FOR UPDATE;

-- name: CreateSubmission :one
INSERT INTO submissions (assignment_id, student_id, file) VALUES ($1, $2, $3) RETURNING *;

-- name: ReplaceSubmissionFile :one
UPDATE submissions SET file = $2, submitted_at = now() WHERE id = $1 RETURNING *;
