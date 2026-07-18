package handler

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type LotteryActivityHandler struct {
	service *service.LotteryActivityService
}

func NewLotteryActivityHandler(activityService *service.LotteryActivityService) *LotteryActivityHandler {
	return &LotteryActivityHandler{service: activityService}
}

func (h *LotteryActivityHandler) GetStatus(c *gin.Context) {
	userID, ok := lotteryUserID(c)
	if !ok {
		return
	}
	status, err := h.service.Status(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, status)
}

func (h *LotteryActivityHandler) Draw(c *gin.Context) {
	userID, ok := lotteryUserID(c)
	if !ok {
		return
	}
	key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if key == "" {
		response.ErrorFrom(c, service.ErrLotteryInvalidKey)
		return
	}
	result, err := h.service.Draw(c.Request.Context(), userID, key)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *LotteryActivityHandler) History(c *gin.Context) {
	userID, ok := lotteryUserID(c)
	if !ok {
		return
	}
	limit := 20
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			response.BadRequest(c, "limit must be between 1 and 100")
			return
		}
		limit = parsed
	}
	draws, err := h.service.History(c.Request.Context(), userID, limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": draws})
}

func lotteryUserID(c *gin.Context) (int64, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return 0, false
	}
	return subject.UserID, true
}
