-- +goose Up
ALTER TABLE users ADD COLUMN hemis_login TEXT UNIQUE;
ALTER TABLE users ADD COLUMN hemis_synced_at TIMESTAMPTZ;

-- HEMIS student ids and employee ids come from different tables and can collide.
ALTER TABLE users DROP CONSTRAINT users_hemis_id_key;
ALTER TABLE users ADD CONSTRAINT users_role_hemis_id_key UNIQUE (role, hemis_id);

-- +goose Down
ALTER TABLE users DROP CONSTRAINT users_role_hemis_id_key;
ALTER TABLE users ADD CONSTRAINT users_hemis_id_key UNIQUE (hemis_id);
ALTER TABLE users DROP COLUMN hemis_synced_at;
ALTER TABLE users DROP COLUMN hemis_login;
