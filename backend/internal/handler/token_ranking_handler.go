package handler

import (
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type TokenRankingHandler struct {
	dashboardService *service.DashboardService
}

func NewTokenRankingHandler(dashboardService *service.DashboardService) *TokenRankingHandler {
	return &TokenRankingHandler{dashboardService: dashboardService}
}

type tokenRankingItem struct {
	UserID       int64   `json:"user_id"`
	Username     string  `json:"username"`
	Requests     int64   `json:"requests"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	CacheTokens  int64   `json:"cache_tokens"`
	TotalTokens  int64   `json:"total_tokens"`
	ActualCost   float64 `json:"actual_cost"`
}

// GetTokenRanking handles getting user token ranking for all users.
// GET /api/v1/usage/token-ranking
func (h *TokenRankingHandler) GetTokenRanking(c *gin.Context) {
	startTime, endTime := parseUserTokenRankingTimeRange(c)

	dim := usagestats.UserBreakdownDimension{}
	dim.Model = c.Query("model")
	rawModelSource := strings.TrimSpace(c.DefaultQuery("model_source", usagestats.ModelSourceRequested))
	if !usagestats.IsValidModelSource(rawModelSource) {
		rawModelSource = usagestats.ModelSourceRequested
	}
	dim.ModelType = rawModelSource

	dim.SortBy = strings.TrimSpace(c.Query("sort_by"))

	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	stats, err := h.dashboardService.GetUserBreakdownStats(
		c.Request.Context(), startTime, endTime, dim, limit,
	)
	if err != nil {
		response.Error(c, 500, "Failed to get token ranking")
		return
	}

	// Mask emails for privacy, keep only necessary fields
	ranking := make([]tokenRankingItem, 0, len(stats))
	for _, s := range stats {
		ranking = append(ranking, tokenRankingItem{
			UserID:       s.UserID,
			Username:     maskEmail(s.Email),
			Requests:     s.Requests,
			InputTokens:  s.InputTokens,
			OutputTokens: s.OutputTokens,
			CacheTokens:  s.CacheTokens,
			TotalTokens:  s.TotalTokens,
			ActualCost:   s.ActualCost,
		})
	}

	response.Success(c, gin.H{
		"users":      ranking,
		"start_date": startTime.Format("2006-01-02"),
		"end_date":   endTime.Format("2006-01-02"),
	})
}

func maskEmail(email string) string {
	if email == "" {
		return "Anonymous"
	}
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return "***"
	}
	name := parts[0]
	if len(name) <= 2 {
		return name + "***"
	}
	return name[:2] + "***"
}

func parseUserTokenRankingTimeRange(c *gin.Context) (time.Time, time.Time) {
	now := time.Now()
	defaultStart := now.AddDate(0, 0, -7)

	startStr := c.Query("start_date")
	endStr := c.Query("end_date")

	var startTime time.Time
	if startStr != "" {
		if t, err := time.Parse("2006-01-02", startStr); err == nil {
			startTime = t
		}
	}
	if startTime.IsZero() {
		startTime = defaultStart
	}

	var endTime time.Time
	if endStr != "" {
		if t, err := time.Parse("2006-01-02", endStr); err == nil {
			endTime = t.AddDate(0, 0, 1) // include the full end day
		}
	}
	if endTime.IsZero() || endTime.After(now) {
		endTime = now
	}

	return startTime, endTime
}
