package admin

import (
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// ExportEmailConfig 一键导出邮件配置（SMTP / Resend 相关设置键）。
// GET /api/v1/admin/settings/email-config/export?include_secrets=true|false
// include_secrets 默认 false：排除 SMTP 密码与 Resend API Key 等敏感键；
// include_secrets=true 时包含敏感键，用于完整迁移到另一台部署。
func (h *SettingHandler) ExportEmailConfig(c *gin.Context) {
	includeSecrets := c.Query("include_secrets") == "true"

	settings, err := h.settingService.ExportEmailConfig(c.Request.Context(), includeSecrets)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.EmailConfigExport{
		Version:    1,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Settings:   settings,
	})
}

// ImportEmailConfigRequest 一键导入邮件配置请求体。
type ImportEmailConfigRequest struct {
	Settings map[string]string `json:"settings"`
}

// ImportEmailConfig 一键导入邮件配置（SMTP / Resend 相关设置键）。
// POST /api/v1/admin/settings/email-config/import
// 只应用邮件相关白名单键；敏感键仅当导入值为非空时才覆盖。
func (h *SettingHandler) ImportEmailConfig(c *gin.Context) {
	var req ImportEmailConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if len(req.Settings) == 0 {
		response.BadRequest(c, "settings is required")
		return
	}

	applied, err := h.settingService.ImportEmailConfig(c.Request.Context(), req.Settings)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{
		"applied": applied,
		"message": fmt.Sprintf("Imported %d email settings", applied),
	})
}
