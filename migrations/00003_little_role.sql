-- +goose Up
-- A "little" admin: a staff account that may use only the admin sections it was granted.
ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'little', 'teacher', 'student'));
ALTER TABLE users ADD COLUMN permissions TEXT[] NOT NULL DEFAULT '{}';

-- +goose Down
DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE role = 'little');
UPDATE users SET role = 'teacher' WHERE role = 'little';
ALTER TABLE users DROP COLUMN permissions;
ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'teacher', 'student'));
