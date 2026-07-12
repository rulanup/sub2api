package handler

import (
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
)

type UserAccountHandler struct {
	adminAccountHandler *admin.AccountHandler
}

func NewUserAccountHandler(adminAccountHandler *admin.AccountHandler) *UserAccountHandler {
	return &UserAccountHandler{adminAccountHandler: adminAccountHandler}
}

// List proxies to admin account list with user_id filter
// GET /api/v1/user/accounts
func (h *UserAccountHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	// Inject user_id filter
	c.Request.URL.RawQuery += "&user_id=" + intToStr(subject.UserID)
	h.adminAccountHandler.List(c)
}

// Create proxies to admin account create with user_id
// POST /api/v1/user/accounts
func (h *UserAccountHandler) Create(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	// Inject user_id
	c.Set("private_user_id", subject.UserID)
	h.adminAccountHandler.Create(c)
}

// GetByID proxies to admin account GetByID with ownership check
// GET /api/v1/user/accounts/:id
func (h *UserAccountHandler) GetByID(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	c.Set("private_user_id", subject.UserID)
	h.adminAccountHandler.GetByID(c)
}

// Update proxies to admin account update with ownership check
// PUT /api/v1/user/accounts/:id
func (h *UserAccountHandler) Update(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	c.Set("private_user_id", subject.UserID)
	h.adminAccountHandler.Update(c)
}

// Delete proxies to admin account delete with ownership check
// DELETE /api/v1/user/accounts/:id
func (h *UserAccountHandler) Delete(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	c.Set("private_user_id", subject.UserID)
	h.adminAccountHandler.Delete(c)
}

// Test proxies to admin account test with ownership check
// POST /api/v1/user/accounts/:id/test
func (h *UserAccountHandler) Test(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	c.Set("private_user_id", subject.UserID)
	h.adminAccountHandler.Test(c)
}

// GetStats proxies to admin account stats with ownership check
// GET /api/v1/user/accounts/:id/stats
func (h *UserAccountHandler) GetStats(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	c.Set("private_user_id", subject.UserID)
	h.adminAccountHandler.GetStats(c)
}

// GetUsage proxies to admin account usage with ownership check
// GET /api/v1/user/accounts/:id/usage
func (h *UserAccountHandler) GetUsage(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	c.Set("private_user_id", subject.UserID)
	h.adminAccountHandler.GetUsage(c)
}

// ClearError proxies to admin account clear error with ownership check
// POST /api/v1/user/accounts/:id/clear-error
func (h *UserAccountHandler) ClearError(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	c.Set("private_user_id", subject.UserID)
	h.adminAccountHandler.ClearError(c)
}

// ClearRateLimit proxies to admin account clear rate limit with ownership check
// POST /api/v1/user/accounts/:id/clear-rate-limit
func (h *UserAccountHandler) ClearRateLimit(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	c.Set("private_user_id", subject.UserID)
	h.adminAccountHandler.ClearRateLimit(c)
}

// RecoverState proxies to admin account recover state with ownership check
// POST /api/v1/user/accounts/:id/recover
func (h *UserAccountHandler) RecoverState(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	c.Set("private_user_id", subject.UserID)
	h.adminAccountHandler.RecoverState(c)
}

// Refresh proxies to admin account refresh with ownership check
// POST /api/v1/user/accounts/:id/refresh
func (h *UserAccountHandler) Refresh(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	c.Set("private_user_id", subject.UserID)
	h.adminAccountHandler.Refresh(c)
}

func intToStr(n int64) string {
	return fmt.Sprintf("%d", n)
}
