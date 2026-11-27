package handler

import (
	"errors"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/canvasgw"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// CanvasHandler 为内嵌的无限画布提供凭据。
type CanvasHandler struct {
	apiKeyService  *service.APIKeyService
	settingService *service.SettingService
}

func NewCanvasHandler(apiKeyService *service.APIKeyService, settingService *service.SettingService) *CanvasHandler {
	return &CanvasHandler{apiKeyService: apiKeyService, settingService: settingService}
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
	ctx := c.Request.Context()
	// 管理员在后台指定的画布分组（可多个，第一个为主分组）；未指定时沿用用户可用的第一个分组。
	groupIDs := h.settingService.GetCanvasGroupIDs(ctx)
	key, err := h.apiKeyService.EnsureCanvasKey(ctx, subject.UserID, groupIDs)
	if err != nil {
		if errors.Is(err, service.ErrGroupNotAllowed) {
			response.Forbidden(c, "当前账号无权使用画布分组，请联系管理员")
			return
		}
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

// AdminSettings 返回无限画布的管理员设置：group_ids 为画布 key 绑定的分组（第一个为主分组），
// 空表示未指定；group_id 为第一个分组，兼容旧前端。
func (h *CanvasHandler) AdminSettings(c *gin.Context) {
	response.Success(c, canvasSettingsPayload(h.settingService.GetCanvasGroupIDs(c.Request.Context())))
}

func canvasSettingsPayload(ids []int64) gin.H {
	if ids == nil {
		ids = []int64{}
	}
	first := int64(0)
	if len(ids) > 0 {
		first = ids[0]
	}
	return gin.H{"group_ids": ids, "group_id": first}
}

// UpdateAdminSettings 设置画布 key 绑定的分组：group_ids 为空表示取消指定；
// 兼容旧前端只传 group_id（0 表示取消指定）。
func (h *CanvasHandler) UpdateAdminSettings(c *gin.Context) {
	var req struct {
		GroupIDs []int64 `json:"group_ids"`
		GroupID  *int64  `json:"group_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid group_ids")
		return
	}
	ids := req.GroupIDs
	if ids == nil && req.GroupID != nil {
		if *req.GroupID < 0 {
			response.BadRequest(c, "invalid group_id")
			return
		}
		if *req.GroupID > 0 {
			ids = []int64{*req.GroupID}
		}
	}
	ctx := c.Request.Context()
	if err := h.settingService.SetCanvasGroupIDs(ctx, ids); err != nil {
		switch {
		case errors.Is(err, service.ErrCanvasGroupInvalid):
			response.BadRequest(c, "分组不存在或未启用")
		case errors.Is(err, service.ErrCanvasGroupPlatformMismatch):
			response.BadRequest(c, "画布分组必须属于同一平台")
		default:
			response.InternalError(c, "Failed to save canvas settings")
		}
		return
	}
	response.Success(c, canvasSettingsPayload(h.settingService.GetCanvasGroupIDs(ctx)))
}
