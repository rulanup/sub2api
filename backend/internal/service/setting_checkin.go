package service

import (
	"context"
	"math"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	CheckinMinAmountDefault = 0.01
	CheckinMaxAmountDefault = 0.10
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
	minAmt := CheckinMinAmountDefault
	maxAmt := CheckinMaxAmountDefault

	if v, err := strconv.ParseFloat(strings.TrimSpace(minVal), 64); err == nil && v >= 0 && !math.IsNaN(v) && !math.IsInf(v, 0) {
		minAmt = v
	}
	if v, err := strconv.ParseFloat(strings.TrimSpace(maxVal), 64); err == nil && v >= 0 && !math.IsNaN(v) && !math.IsInf(v, 0) {
		maxAmt = v
	}
	if minAmt > maxAmt {
		minAmt, maxAmt = maxAmt, minAmt
	}

	return minAmt, maxAmt
}

func ValidateCheckinAmountRange(minAmount, maxAmount float64) error {
	if math.IsNaN(minAmount) || math.IsInf(minAmount, 0) ||
		math.IsNaN(maxAmount) || math.IsInf(maxAmount, 0) ||
		minAmount < 0 || maxAmount < 0 || minAmount > maxAmount {
		return infraerrors.BadRequest("INVALID_CHECKIN_AMOUNT_RANGE", "check-in amounts must be finite non-negative numbers with min less than or equal to max")
	}
	return nil
}
