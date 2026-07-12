//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCalculateCost_RateMultiplier_NegativeClampedToZero 锁定负数倍率被
// 钳制为 0（而非历史上的 1.0），避免配置异常导致静默按标准价扣费。
func TestCalculateCost_RateMultiplier_NegativeClampedToZero(t *testing.T) {
	svc := newTestBillingService()
	tokens := UsageTokens{InputTokens: 1000, OutputTokens: 500}

	tests := []struct {
		name       string
		multiplier float64
		wantRatio  float64
		wantZero   bool
	}{
		{"negative clamped to 0", -1.5, 0, true},
		{"zero short-circuits pricing", 0, 0, true},
		{"positive 2x applied", 2.0, 2.0, false},
		{"positive 0.5x applied", 0.5, 0.5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost, err := svc.CalculateCost("claude-sonnet-4", tokens, tt.multiplier)
			require.NoError(t, err)
			if tt.wantZero {
				require.Zero(t, cost.TotalCost)
			}
			require.InDelta(t, tt.wantRatio*cost.TotalCost, cost.ActualCost, 1e-9)
		})
	}
}

// TestCalculateImageCost_RateMultiplier_NegativeClampedToZero 图片按次计费路径
// 同样遵循"负数 → 0"语义。
func TestCalculateImageCost_RateMultiplier_NegativeClampedToZero(t *testing.T) {
	svc := newTestBillingService()
	price := 0.04
	cfg := &ImagePriceConfig{Price1K: &price}

	tests := []struct {
		name       string
		multiplier float64
		wantRatio  float64
	}{
		{"negative clamped to 0", -0.5, 0},
		{"zero passes through", 0, 0},
		{"positive 3x applied", 3.0, 3.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := svc.CalculateImageCost("imagen-3", "1K", 2, cfg, tt.multiplier)
			require.NotNil(t, cost)
			if tt.multiplier <= 0 {
				require.Zero(t, cost.TotalCost)
			} else {
				require.Greater(t, cost.TotalCost, 0.0)
			}
			require.InDelta(t, tt.wantRatio*cost.TotalCost, cost.ActualCost, 1e-9)
		})
	}
}
