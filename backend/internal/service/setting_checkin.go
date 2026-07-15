package service

import (
	"context"
	"strconv"
	"strings"
)

// IsCheckinEnabled checks if daily check-in is enabled.
func (s *SettingService) IsCheckinEnabled(ctx context.Context) bool {
	val, err := s.settingRepo.GetValue(ctx, SettingKeyCheckinEnabled)
	if err != nil {
		return false
	}
	return strings.TrimSpace(val) == "true"
}

// GetCheckinAmountRange returns the min and max check-in amount.
func (s *SettingService) GetCheckinAmountRange(ctx context.Context) (float64, float64) {
	minVal, _ := s.settingRepo.GetValue(ctx, SettingKeyCheckinMinAmount)
	maxVal, _ := s.settingRepo.GetValue(ctx, SettingKeyCheckinMaxAmount)
	return parseCheckinAmountRange(minVal, maxVal)
}

func parseCheckinAmountRange(minVal, maxVal string) (float64, float64) {
	minAmt := 0.01
	maxAmt := 0.10

	if v, err := strconv.ParseFloat(strings.TrimSpace(minVal), 64); err == nil && v > 0 {
		minAmt = v
	}
	if v, err := strconv.ParseFloat(strings.TrimSpace(maxVal), 64); err == nil && v > 0 {
		maxAmt = v
	}

	return minAmt, maxAmt
}
