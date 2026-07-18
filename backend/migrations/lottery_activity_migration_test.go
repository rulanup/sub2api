package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration180CreatesTransactionalLotteryAuditSchema(t *testing.T) {
	content, err := FS.ReadFile("180_lottery_activity_draws.sql")
	require.NoError(t, err)
	sql := string(content)
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS lottery_activity_draws")
	require.Contains(t, sql, "UNIQUE (activity_id, user_id, idempotency_hash)")
	require.NotContains(t, sql, "idempotency_key ")
	require.Contains(t, sql, "config_snapshot              JSONB NOT NULL")
	require.Contains(t, sql, "idx_lottery_activity_draws_activity_total")
	require.Contains(t, sql, "idx_lottery_activity_draws_user_day")
	require.Contains(t, sql, "subscription_expires_before")
}
