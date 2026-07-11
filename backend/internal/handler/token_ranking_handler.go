package handler

import (
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type TokenRankingHandler struct {
	dashboardService *service.DashboardService
	usageService     *service.UsageService
}

func NewTokenRankingHandler(dashboardService *service.DashboardService, usageService *service.UsageService) *TokenRankingHandler {
	return &TokenRankingHandler{dashboardService: dashboardService, usageService: usageService}
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
	CacheHitRate float64 `json:"cache_hit_rate"`
	Percentage   float64 `json:"percentage"`
	Rank         int     `json:"rank"`
}

type tokenRankingSummary struct {
	TotalCost      float64 `json:"total_cost"`
	TotalTokens    int64   `json:"total_tokens"`
	ActiveUsers    int     `json:"active_users"`
	AvgCostPerUser float64 `json:"avg_cost_per_user"`
	MyRank         *int    `json:"my_rank"`
	MyCost         float64 `json:"my_cost"`
}

type tokenRankingResponse struct {
	Summary tokenRankingSummary `json:"summary"`
	Items   []tokenRankingItem  `json:"items"`
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

	// Compute totals
	var totalCost float64
	var totalTokens int64
	for _, s := range stats {
		totalCost += s.ActualCost
		totalTokens += s.TotalTokens
	}

	// Build ranking items
	ranking := make([]tokenRankingItem, 0, len(stats))
	for i, s := range stats {
		cacheHitRate := 0.0
		denom := s.InputTokens + s.CacheTokens
		if denom > 0 {
			cacheHitRate = float64(s.CacheTokens) / float64(denom) * 100
		}

		percentage := 0.0
		if totalCost > 0 {
			percentage = s.ActualCost / totalCost * 100
		}

		ranking = append(ranking, tokenRankingItem{
			UserID:       s.UserID,
			Username:     maskEmail(s.Email),
			Requests:     s.Requests,
			InputTokens:  s.InputTokens,
			OutputTokens: s.OutputTokens,
			CacheTokens:  s.CacheTokens,
			TotalTokens:  s.TotalTokens,
			ActualCost:   s.ActualCost,
			CacheHitRate: cacheHitRate,
			Percentage:   percentage,
			Rank:         i + 1,
		})
	}

	// Compute summary
	avgCost := 0.0
	if len(stats) > 0 {
		avgCost = totalCost / float64(len(stats))
	}

	summary := tokenRankingSummary{
		TotalCost:      totalCost,
		TotalTokens:    totalTokens,
		ActiveUsers:    len(stats),
		AvgCostPerUser: avgCost,
		MyRank:         nil,
		MyCost:         0,
	}

	// Try to find current user's rank
	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok {
		for _, item := range ranking {
			if item.UserID == subject.UserID {
				rank := item.Rank
				summary.MyRank = &rank
				summary.MyCost = item.ActualCost
				break
			}
		}
	}

	response.Success(c, tokenRankingResponse{
		Summary: summary,
		Items:   ranking,
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
			endTime = t.AddDate(0, 0, 1)
		}
	}
	if endTime.IsZero() || endTime.After(now) {
		endTime = now
	}

	return startTime, endTime
}
