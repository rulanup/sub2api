package handler

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// CanvasHandler 为内嵌的无限画布提供凭据。
type CanvasHandler struct {
	apiKeyService *service.APIKeyService
}

func NewCanvasHandler(apiKeyService *service.APIKeyService) *CanvasHandler {
	return &CanvasHandler{apiKeyService: apiKeyService}
}

// Key 返回当前用户的无限画布专用 API key（不存在则创建）。
//
// 画布通过 postMessage 拿到该 key 后直接调用 /v1/images/*，
// 用量与计费归属该用户本人。
func (h *CanvasHandler) Key(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	key, err := h.apiKeyService.EnsureCanvasKey(c.Request.Context(), subject.UserID)
	if err != nil {
		response.InternalError(c, "Failed to prepare canvas API key")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user_id": subject.UserID,
		"api_key": key,
	})
}
