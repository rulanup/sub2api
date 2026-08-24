package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserRiskProfilesMigration(t *testing.T) {
	content, err := FS.ReadFile("231_user_risk_profiles.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS user_risk_profiles")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS user_risk_score_events")
	require.Contains(t, sql, "REFERENCES users(id) ON DELETE CASCADE")
	require.Contains(t, sql, "score INTEGER NOT NULL DEFAULT 0")
	require.Contains(t, sql, "CHECK (score >= 0 AND score <= 100)")
	require.Contains(t, sql, "last_reason_code TEXT NOT NULL DEFAULT ''")
	require.Contains(t, sql, "version BIGINT NOT NULL DEFAULT 1")
	require.Contains(t, sql, "UNIQUE (user_id, dedupe_key)")
	require.Contains(t, sql, "idx_user_risk_profiles_level_score_updated")
}
