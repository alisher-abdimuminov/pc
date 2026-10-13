-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE faculties (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    hemis_id   BIGINT UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username             TEXT UNIQUE,
    password_hash        TEXT,
    role                 TEXT NOT NULL CHECK (role IN ('admin', 'teacher', 'student')),
    is_active            BOOLEAN NOT NULL DEFAULT true,
    must_change_password BOOLEAN NOT NULL DEFAULT false,
    last_login_at        TIMESTAMPTZ,

    hemis_id       BIGINT UNIQUE,
    first_name     TEXT NOT NULL DEFAULT '',
    second_name    TEXT NOT NULL DEFAULT '',
    third_name     TEXT NOT NULL DEFAULT '',
    full_name      TEXT NOT NULL DEFAULT '',
    short_name     TEXT NOT NULL DEFAULT '',
    image          TEXT NOT NULL DEFAULT '',
    hemis_img_url  TEXT NOT NULL DEFAULT '',
    birth_date     TEXT NOT NULL DEFAULT '',
    phone          TEXT NOT NULL DEFAULT '',
    passport_pin   TEXT NOT NULL DEFAULT '',
    passport_number TEXT NOT NULL DEFAULT '',
    gender         TEXT NOT NULL DEFAULT '',
    payment_form   TEXT NOT NULL DEFAULT '',
    level          TEXT NOT NULL DEFAULT '',
    semester       TEXT NOT NULL DEFAULT '',
    gpa            TEXT NOT NULL DEFAULT '',
    address        TEXT NOT NULL DEFAULT '',
    country        TEXT NOT NULL DEFAULT '',
    province       TEXT NOT NULL DEFAULT '',
    district       TEXT NOT NULL DEFAULT '',

    group_id   UUID,
    faculty_id UUID REFERENCES faculties (id) ON DELETE SET NULL,

    face_embedding   REAL[],
    face_model       TEXT NOT NULL DEFAULT '',
    face_enrolled_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE groups (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    hemis_id   BIGINT UNIQUE,
    faculty_id UUID REFERENCES faculties (id) ON DELETE SET NULL,
    teacher_id UUID REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE users
    ADD CONSTRAINT users_group_id_fkey FOREIGN KEY (group_id) REFERENCES groups (id) ON DELETE SET NULL;

CREATE INDEX users_role_idx ON users (role);
CREATE INDEX users_group_id_idx ON users (group_id);
CREATE INDEX users_faculty_id_idx ON users (faculty_id);
CREATE INDEX groups_teacher_id_idx ON groups (teacher_id);
CREATE INDEX groups_faculty_id_idx ON groups (faculty_id);

CREATE TABLE sessions (
    id         TEXT PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    ip         INET,
    user_agent TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX sessions_user_id_idx ON sessions (user_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

CREATE TABLE locations (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name             TEXT NOT NULL,
    polygon          JSONB NOT NULL CHECK (jsonb_typeof(polygon) = 'array' AND jsonb_array_length(polygon) >= 3),
    tolerance_meters INTEGER NOT NULL DEFAULT 30 CHECK (tolerance_meters >= 0),
    location_url     TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE schedules (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL DEFAULT '',
    start_date  DATE NOT NULL,
    end_date    DATE NOT NULL,
    user_id     UUID REFERENCES users (id) ON DELETE SET NULL,
    group_id    UUID REFERENCES groups (id) ON DELETE SET NULL,
    location_id UUID REFERENCES locations (id) ON DELETE SET NULL,
    monday      BOOLEAN NOT NULL DEFAULT false,
    tuesday     BOOLEAN NOT NULL DEFAULT false,
    wednesday   BOOLEAN NOT NULL DEFAULT false,
    thursday    BOOLEAN NOT NULL DEFAULT false,
    friday      BOOLEAN NOT NULL DEFAULT false,
    saturday    BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT schedules_target_chk CHECK ((user_id IS NULL) <> (group_id IS NULL)),
    CONSTRAINT schedules_dates_chk CHECK (end_date >= start_date)
);
CREATE INDEX schedules_user_id_idx ON schedules (user_id);
CREATE INDEX schedules_group_id_idx ON schedules (group_id);
CREATE INDEX schedules_dates_idx ON schedules (start_date, end_date);

CREATE TABLE attendances (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id  UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    schedule_id UUID NOT NULL REFERENCES schedules (id) ON DELETE RESTRICT,
    shift       SMALLINT NOT NULL CHECK (shift IN (1, 2)),
    date        DATE NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (student_id, schedule_id, date)
);
CREATE INDEX attendances_date_idx ON attendances (date);

CREATE TABLE attendance_steps (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attendance_id UUID NOT NULL REFERENCES attendances (id) ON DELETE CASCADE,
    step          SMALLINT NOT NULL CHECK (step IN (1, 2, 3)),
    success       BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (attendance_id, step)
);

CREATE TABLE attendance_attempts (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    step_id           UUID NOT NULL REFERENCES attendance_steps (id) ON DELETE CASCADE,
    image             TEXT NOT NULL DEFAULT '',
    ip_address        INET,
    user_agent        TEXT NOT NULL DEFAULT '',
    latitude          DOUBLE PRECISION NOT NULL DEFAULT 0,
    longitude         DOUBLE PRECISION NOT NULL DEFAULT 0,
    gps_accuracy      DOUBLE PRECISION NOT NULL DEFAULT 0,
    location_id       UUID REFERENCES locations (id) ON DELETE SET NULL,
    face_verified     BOOLEAN NOT NULL DEFAULT false,
    location_verified BOOLEAN NOT NULL DEFAULT false,
    liveness_verified BOOLEAN NOT NULL DEFAULT false,
    success           BOOLEAN NOT NULL DEFAULT false,
    error             TEXT NOT NULL DEFAULT '',
    face_distance     DOUBLE PRECISION,
    face_threshold    DOUBLE PRECISION,
    antispoof_score   DOUBLE PRECISION,
    attempted_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX attendance_attempts_step_id_idx ON attendance_attempts (step_id);
CREATE INDEX attendance_attempts_attempted_at_idx ON attendance_attempts (attempted_at);

CREATE TABLE assignments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    teacher_id  UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    group_id    UUID NOT NULL REFERENCES groups (id) ON DELETE RESTRICT,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    file        TEXT NOT NULL DEFAULT '',
    deadline    TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX assignments_group_id_idx ON assignments (group_id);

CREATE TABLE submissions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id UUID NOT NULL REFERENCES assignments (id) ON DELETE CASCADE,
    student_id    UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    file          TEXT NOT NULL DEFAULT '',
    submitted_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    grade         NUMERIC(2, 1) CHECK (grade BETWEEN 1 AND 5),
    feedback      TEXT NOT NULL DEFAULT '',
    graded_at     TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (assignment_id, student_id)
);

CREATE TABLE audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id    UUID REFERENCES users (id) ON DELETE SET NULL,
    actor_role  TEXT NOT NULL DEFAULT '',
    action      TEXT NOT NULL CHECK (action IN (
        'create', 'update', 'delete', 'login', 'login_failed', 'logout',
        'view_sensitive', 'export', 'grade', 'attendance_attempt')),
    entity_type TEXT NOT NULL DEFAULT '',
    entity_id   UUID,
    changes     JSONB NOT NULL DEFAULT '{}'::jsonb,
    ip          INET,
    user_agent  TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX audit_logs_entity_idx ON audit_logs (entity_type, entity_id);
CREATE INDEX audit_logs_actor_idx ON audit_logs (actor_id, created_at DESC);
CREATE INDEX audit_logs_created_idx ON audit_logs (created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS submissions;
DROP TABLE IF EXISTS assignments;
DROP TABLE IF EXISTS attendance_attempts;
DROP TABLE IF EXISTS attendance_steps;
DROP TABLE IF EXISTS attendances;
DROP TABLE IF EXISTS schedules;
DROP TABLE IF EXISTS locations;
DROP TABLE IF EXISTS sessions;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_group_id_fkey;
DROP TABLE IF EXISTS groups;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS faculties;
