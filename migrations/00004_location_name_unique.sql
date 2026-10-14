-- +goose Up
-- Rename existing duplicates (keeping the oldest as-is) so the unique index can be created.
UPDATE locations l SET name = l.name || ' (' || d.rn || ')'
FROM (
    SELECT id, row_number() OVER (PARTITION BY lower(name) ORDER BY created_at, id) AS rn FROM locations
) d
WHERE l.id = d.id AND d.rn > 1;
CREATE UNIQUE INDEX locations_name_key ON locations (lower(name));

-- +goose Down
DROP INDEX locations_name_key;
