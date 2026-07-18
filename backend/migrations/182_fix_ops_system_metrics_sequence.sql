SELECT setval(
    pg_get_serial_sequence('ops_system_metrics', 'id'),
    COALESCE((SELECT MAX(id) FROM ops_system_metrics), 1),
    (SELECT COUNT(*) > 0 FROM ops_system_metrics)
);
