package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration181SynchronizesSettingsSequence(t *testing.T) {
	content, err := FS.ReadFile("181_fix_settings_sequence.sql")
	require.NoError(t, err)
	sql := string(content)
	require.Contains(t, sql, "pg_get_serial_sequence('settings', 'id')")
	require.Contains(t, sql, "SELECT MAX(id) FROM settings")
}

func TestMigration182SynchronizesOpsSystemMetricsSequence(t *testing.T) {
	content, err := FS.ReadFile("182_fix_ops_system_metrics_sequence.sql")
	require.NoError(t, err)
	sql := string(content)
	require.Contains(t, sql, "pg_get_serial_sequence('ops_system_metrics', 'id')")
	require.Contains(t, sql, "SELECT MAX(id) FROM ops_system_metrics")
}
