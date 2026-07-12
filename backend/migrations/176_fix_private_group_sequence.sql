DROP INDEX IF EXISTS groups_name_unique_active;

SELECT setval(
    pg_get_serial_sequence('groups', 'id'),
    COALESCE((SELECT MAX(id) FROM groups), 1),
    (SELECT COUNT(*) > 0 FROM groups)
);
