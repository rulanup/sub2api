package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"

	"github.com/gin-gonic/gin"
)

// registerCanvasAdminRoutes 注册无限画布的管理员设置接口。
func registerCanvasAdminRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	canvas := admin.Group("/canvas")
	{
		canvas.GET("/settings", h.Canvas.AdminSettings)
		canvas.PUT("/settings", h.Canvas.UpdateAdminSettings)
	}
}
