package handler

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// CheckinHandler handles daily check-in functionality.
type CheckinHandler struct {
	settingService *service.SettingService
	checkinService *service.CheckinService
}

// NewCheckinHandler creates a new CheckinHandler.
func NewCheckinHandler(settingService *service.SettingService, checkinService *service.CheckinService) *CheckinHandler {
	return &CheckinHandler{
		settingService: settingService,
		checkinService: checkinService,
	}
}

// GetStatus returns the current user's check-in status for today.
// GET /api/v1/user/checkin/status
func (h *CheckinHandler) GetStatus(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	enabled := h.settingService.IsCheckinEnabled(c.Request.Context())
	if !enabled {
		response.Success(c, gin.H{"enabled": false})
		return
	}

	checkedIn, amount, err := h.checkinService.GetTodayStatus(c.Request.Context(), subject.UserID)
	if err != nil {
		response.Error(c, 500, "Failed to get checkin status")
		return
	}

	minAmt, maxAmt := h.settingService.GetCheckinAmountRange(c.Request.Context())

	response.Success(c, gin.H{
		"enabled":     true,
		"checked_in":  checkedIn,
		"amount":      amount,
		"min_amount":  minAmt,
		"max_amount":  maxAmt,
	})
}

// Checkin performs a daily check-in for the current user.
// POST /api/v1/user/checkin
func (h *CheckinHandler) Checkin(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	if !h.settingService.IsCheckinEnabled(c.Request.Context()) {
		response.Error(c, 403, "Check-in is disabled")
		return
	}

	// Check if already checked in today
	checkedIn, _, err := h.checkinService.GetTodayStatus(c.Request.Context(), subject.UserID)
	if err != nil {
		response.Error(c, 500, "Failed to check status")
		return
	}
	if checkedIn {
		response.Error(c, 409, "Already checked in today")
		return
	}

	// Generate random amount within range
	minAmt, maxAmt := h.settingService.GetCheckinAmountRange(c.Request.Context())
	amount := minAmt
	if maxAmt > minAmt {
		amount = minAmt + rand.Float64()*(maxAmt-minAmt)
	}
	// Round to 6 decimal places
	amount = float64(int(amount*1000000)) / 1000000

	// Record check-in and add balance
	if err := h.checkinService.DoCheckin(c.Request.Context(), subject.UserID, amount); err != nil {
		response.Error(c, 500, "Check-in failed: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"amount":    amount,
		"message":   fmt.Sprintf("Check-in successful! Received $%.6f", amount),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
