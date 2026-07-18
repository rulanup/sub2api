SELECT setval(
    pg_get_serial_sequence('settings', 'id'),
    COALESCE((SELECT MAX(id) FROM settings), 1),
    (SELECT COUNT(*) > 0 FROM settings)
);
