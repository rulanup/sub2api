-- Support periodic abuse scans over narrow request-type time windows.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_sync_abuse_created_user
    ON usage_logs (created_at, user_id)
    WHERE request_type = 1 AND user_id > 0;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_cyber_usage_created_user
    ON usage_logs (created_at, user_id)
    WHERE request_type = 4 AND user_id > 0;
