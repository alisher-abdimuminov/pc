-- name: UpsertFacultyByHemis :one
INSERT INTO faculties (name, hemis_id) VALUES ($1, $2)
ON CONFLICT (hemis_id) DO UPDATE SET name = EXCLUDED.name
RETURNING *;

-- name: UpsertGroupByHemis :one
INSERT INTO groups (name, hemis_id, faculty_id) VALUES ($1, $2, $3)
ON CONFLICT (hemis_id) DO UPDATE SET name = EXCLUDED.name, faculty_id = EXCLUDED.faculty_id
RETURNING *;

-- name: CreateHemisStudent :one
INSERT INTO users (username, role, is_active, hemis_id, first_name, second_name, third_name, full_name, short_name,
    birth_date, phone, passport_pin, passport_number, gender, payment_form, level, semester, gpa, address,
    country, province, district, group_id, faculty_id)
VALUES ($1, 'student', true, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
RETURNING *;

-- name: UpdateUserFromHemis :one
UPDATE users SET hemis_id = $2, first_name = $3, second_name = $4, third_name = $5, full_name = $6, short_name = $7,
    birth_date = $8, phone = $9, passport_pin = $10, passport_number = $11, gender = $12, payment_form = $13,
    level = $14, semester = $15, gpa = $16, address = $17, country = $18, province = $19, district = $20,
    group_id = $21, faculty_id = $22, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SetUserHemisImageURL :exec
UPDATE users SET hemis_img_url = $2 WHERE id = $1;

-- name: CreateHemisTeacher :one
INSERT INTO users (role, is_active, hemis_login, hemis_id, first_name, second_name, third_name, full_name, short_name,
    birth_date, phone, passport_pin, passport_number, hemis_synced_at)
VALUES ('teacher', true, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, now())
RETURNING *;

-- Teacher sync: a value HEMIS did not send never wipes what we already have.
-- name: UpdateTeacherFromHemis :one
UPDATE users SET
    hemis_id        = COALESCE(sqlc.narg('hemis_id')::bigint, hemis_id),
    first_name      = COALESCE(NULLIF(sqlc.arg('first_name')::text, ''), first_name),
    second_name     = COALESCE(NULLIF(sqlc.arg('second_name')::text, ''), second_name),
    third_name      = COALESCE(NULLIF(sqlc.arg('third_name')::text, ''), third_name),
    full_name       = COALESCE(NULLIF(sqlc.arg('full_name')::text, ''), full_name),
    short_name      = COALESCE(NULLIF(sqlc.arg('short_name')::text, ''), short_name),
    birth_date      = COALESCE(NULLIF(sqlc.arg('birth_date')::text, ''), birth_date),
    phone           = COALESCE(NULLIF(sqlc.arg('phone')::text, ''), phone),
    passport_pin    = COALESCE(NULLIF(sqlc.arg('passport_pin')::text, ''), passport_pin),
    passport_number = COALESCE(NULLIF(sqlc.arg('passport_number')::text, ''), passport_number),
    hemis_synced_at = now(),
    updated_at      = now()
WHERE id = sqlc.arg('id')
RETURNING *;
