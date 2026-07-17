package service

import (
	"math"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestParseCheckinAmountRange(t *testing.T) {
	tests := []struct {
		name    string
		minRaw  string
		maxRaw  string
		wantMin float64
		wantMax float64
	}{
		{name: "defaults", wantMin: CheckinMinAmountDefault, wantMax: CheckinMaxAmountDefault},
		{name: "stored values including zero", minRaw: "0", maxRaw: "1.5", wantMin: 0, wantMax: 1.5},
		{name: "invalid values use defaults", minRaw: "NaN", maxRaw: "+Inf", wantMin: CheckinMinAmountDefault, wantMax: CheckinMaxAmountDefault},
		{name: "negative values use defaults", minRaw: "-1", maxRaw: "-2", wantMin: CheckinMinAmountDefault, wantMax: CheckinMaxAmountDefault},
		{name: "inverted stored range is canonicalized", minRaw: "2", maxRaw: "1", wantMin: 1, wantMax: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			minAmount, maxAmount := parseCheckinAmountRange(tt.minRaw, tt.maxRaw)
			require.Equal(t, tt.wantMin, minAmount)
			require.Equal(t, tt.wantMax, maxAmount)
		})
	}
}

func TestSettingServiceParseSettings_CheckinDefaultsAndStoredValues(t *testing.T) {
	svc := NewSettingService(nil, &config.Config{})

	defaults := svc.parseSettings(map[string]string{})
	require.False(t, defaults.CheckinEnabled)
	require.Equal(t, CheckinMinAmountDefault, defaults.CheckinMinAmount)
	require.Equal(t, CheckinMaxAmountDefault, defaults.CheckinMaxAmount)

	stored := svc.parseSettings(map[string]string{
		SettingKeyCheckinEnabled:   "true",
		SettingKeyCheckinMinAmount: "0",
		SettingKeyCheckinMaxAmount: "2.75",
	})
	require.True(t, stored.CheckinEnabled)
	require.Equal(t, 0.0, stored.CheckinMinAmount)
	require.Equal(t, 2.75, stored.CheckinMaxAmount)
	require.False(t, math.IsNaN(stored.CheckinMinAmount))
}
