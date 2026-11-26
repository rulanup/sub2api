package handler

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/canvasgw"
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

// Session 为画布网关签发一次性登录票据（60 秒内有效、只能用一次）。
//
// 前端把 iframe 指向返回的 url，网关验票后种下画布会话 cookie，
// 之后所有请求由网关注入 X-Canvas-User，画布据此切换到该用户的数据目录。
func (h *CanvasHandler) Session(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	gw := canvasgw.Default()
	if gw == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "canvas gateway disabled"})
		return
	}
	ticket := gw.IssueTicket(subject.UserID)
	c.JSON(http.StatusOK, gin.H{
		"url":    gw.LoginURL(c.Request, ticket),
		"origin": gw.PublicOrigin(c.Request),
	})
}
