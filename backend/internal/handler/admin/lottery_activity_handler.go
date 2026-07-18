package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type LotteryActivityHandler struct {
	service *service.LotteryActivityService
}

func NewLotteryActivityHandler(activityService *service.LotteryActivityService) *LotteryActivityHandler {
	return &LotteryActivityHandler{service: activityService}
}

func (h *LotteryActivityHandler) GetConfig(c *gin.Context) {
	config, err := h.service.GetConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, config)
}

func (h *LotteryActivityHandler) UpdateConfig(c *gin.Context) {
	var request service.LotteryActivityConfig
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	config, err := h.service.UpdateConfig(c.Request.Context(), &request)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, config)
}
