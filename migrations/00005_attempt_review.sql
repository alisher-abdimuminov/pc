-- +goose Up
-- Manual review of attempts accepted while the face-service was busy.
-- review: '' = normal, 'pending' = accepted without a face check, 'approved' / 'rejected' = decided by a teacher or admin.
ALTER TABLE attendance_attempts
    ADD COLUMN review      TEXT NOT NULL DEFAULT '' CHECK (review IN ('', 'pending', 'approved', 'rejected')),
    ADD COLUMN reviewed_by UUID NULL REFERENCES users (id) ON DELETE SET NULL,
    ADD COLUMN reviewed_at TIMESTAMPTZ NULL;
ALTER TABLE attendance_steps
    ADD COLUMN review TEXT NOT NULL DEFAULT '' CHECK (review IN ('', 'pending', 'approved', 'rejected'));
CREATE INDEX attendance_attempts_pending_idx ON attendance_attempts (attempted_at DESC) WHERE review = 'pending';

-- +goose Down
DROP INDEX attendance_attempts_pending_idx;
ALTER TABLE attendance_steps DROP COLUMN review;
ALTER TABLE attendance_attempts DROP COLUMN reviewed_at, DROP COLUMN reviewed_by, DROP COLUMN review;
