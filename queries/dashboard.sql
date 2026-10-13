-- name: DayAttendanceStats :one
-- Students expected on a date (an active schedule covers the date and weekday) and how many
-- of them completed 3/3 (full) or 1-2 (partial) steps. absent = expected - full - partial.
WITH sched AS (
    SELECT s.* FROM schedules s
    WHERE s.start_date <= sqlc.arg('on_date')::date AND s.end_date >= sqlc.arg('on_date')::date
      AND CASE extract(isodow FROM sqlc.arg('on_date')::date)::int
            WHEN 1 THEN s.monday WHEN 2 THEN s.tuesday WHEN 3 THEN s.wednesday
            WHEN 4 THEN s.thursday WHEN 5 THEN s.friday WHEN 6 THEN s.saturday
            ELSE false END
),
expected AS (
    SELECT DISTINCT u.id FROM users u
    JOIN sched s ON s.user_id = u.id OR s.group_id = u.group_id
    WHERE u.role = 'student' AND u.is_active
),
done AS (
    SELECT a.student_id, count(*) FILTER (WHERE st.success) AS ok
    FROM attendances a
    LEFT JOIN attendance_steps st ON st.attendance_id = a.id
    WHERE a.date = sqlc.arg('on_date')::date
    GROUP BY a.student_id
)
SELECT count(*)::bigint AS expected,
       (count(*) FILTER (WHERE d.ok = 3))::bigint AS full_count,
       (count(*) FILTER (WHERE d.ok BETWEEN 1 AND 2))::bigint AS partial_count
FROM expected e
LEFT JOIN done d ON d.student_id = e.id;
