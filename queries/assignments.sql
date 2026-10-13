-- name: CreateAssignment :one
INSERT INTO assignments (teacher_id, group_id, title, description, file, deadline)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: GetAssignment :one
SELECT * FROM assignments WHERE id = $1;

-- name: GetAssignmentForTeacher :one
SELECT a.* FROM assignments a JOIN groups g ON g.id = a.group_id
WHERE a.id = $1 AND g.teacher_id = $2;

-- name: ListTeacherAssignments :many
SELECT a.*, g.name AS group_name,
    (SELECT count(*) FROM submissions s WHERE s.assignment_id = a.id)::bigint AS submission_count,
    (SELECT count(*) FROM users u WHERE u.group_id = a.group_id AND u.role = 'student' AND u.is_active)::bigint AS student_count
FROM assignments a JOIN groups g ON g.id = a.group_id
WHERE g.teacher_id = sqlc.arg('teacher_id')::uuid
  AND (sqlc.narg('group_id')::uuid IS NULL OR a.group_id = sqlc.narg('group_id'))
  AND (sqlc.arg('search')::text = '' OR a.title ILIKE '%' || sqlc.arg('search') || '%')
ORDER BY a.deadline DESC
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: CountTeacherAssignments :one
SELECT count(*) FROM assignments a JOIN groups g ON g.id = a.group_id
WHERE g.teacher_id = sqlc.arg('teacher_id')::uuid
  AND (sqlc.narg('group_id')::uuid IS NULL OR a.group_id = sqlc.narg('group_id'))
  AND (sqlc.arg('search')::text = '' OR a.title ILIKE '%' || sqlc.arg('search') || '%');

-- name: UpdateAssignment :one
UPDATE assignments SET title = $2, description = $3, file = $4, deadline = $5, updated_at = now()
WHERE id = $1 RETURNING *;

-- name: DeleteAssignment :exec
DELETE FROM assignments WHERE id = $1;

-- name: CountAssignmentSubmissions :one
SELECT count(*) FROM submissions WHERE assignment_id = $1;

-- name: ListAssignmentRoster :many
SELECT u.id AS student_id, u.full_name, s.id AS submission_id, s.submitted_at, s.file,
       s.grade, s.feedback, s.graded_at
FROM users u
LEFT JOIN submissions s ON s.student_id = u.id AND s.assignment_id = sqlc.arg('assignment_id')::uuid
WHERE u.role = 'student' AND u.group_id = sqlc.arg('group_id')::uuid AND (u.is_active OR s.id IS NOT NULL)
ORDER BY u.full_name;

-- name: GetSubmission :one
SELECT * FROM submissions WHERE id = $1;

-- name: GetSubmissionOwner :one
SELECT s.id, s.file, s.student_id, s.assignment_id, a.group_id, g.teacher_id
FROM submissions s
JOIN assignments a ON a.id = s.assignment_id
JOIN groups g ON g.id = a.group_id
WHERE s.id = $1;

-- name: GradeSubmission :one
UPDATE submissions SET grade = $2, feedback = $3, graded_at = now() WHERE id = $1 RETURNING *;

-- name: LockAssignment :one
SELECT * FROM assignments WHERE id = $1 FOR UPDATE;
