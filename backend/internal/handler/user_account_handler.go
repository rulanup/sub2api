package handler

import (
	"encoding/json"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type UserAccountHandler struct {
	userAccountService *service.UserAccountService
}

func NewUserAccountHandler(userAccountService *service.UserAccountService) *UserAccountHandler {
	return &UserAccountHandler{userAccountService: userAccountService}
}

type createUserAccountRequest struct {
	Name        string          `json:"name" binding:"required"`
	Platform    string          `json:"platform" binding:"required"`
	Credentials json.RawMessage `json:"credentials" binding:"required"`
	GroupIDs    []int64         `json:"group_ids"`
	Notes       string          `json:"notes"`
}

type updateUserAccountRequest struct {
	Name        *string          `json:"name"`
	Credentials *json.RawMessage `json:"credentials"`
	GroupIDs    []int64          `json:"group_ids"`
	Notes       *string          `json:"notes"`
	Status      *string          `json:"status"`
}

// List returns all accounts for the current user
// GET /api/v1/user/accounts
func (h *UserAccountHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	platform := c.Query("platform")
	status := c.DefaultQuery("status", "")

	accounts, err := h.userAccountService.ListByUserID(c.Request.Context(), subject.UserID, platform, status)
	if err != nil {
		response.Error(c, 500, "Failed to list accounts")
		return
	}

	response.Success(c, accounts)
}

// Create adds a new account for the current user
// POST /api/v1/user/accounts
func (h *UserAccountHandler) Create(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req createUserAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	account, err := h.userAccountService.Create(c.Request.Context(), subject.UserID, &service.CreateUserAccountInput{
		Name:        req.Name,
		Platform:    req.Platform,
		Credentials: req.Credentials,
		GroupIDs:    req.GroupIDs,
		Notes:       req.Notes,
	})
	if err != nil {
		response.Error(c, 500, "Failed to create account: "+err.Error())
		return
	}

	response.Success(c, account)
}

// Update updates an account
// PUT /api/v1/user/accounts/:id
func (h *UserAccountHandler) Update(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	var req updateUserAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	account, err := h.userAccountService.Update(c.Request.Context(), subject.UserID, id, &service.UpdateUserAccountInput{
		Name:        req.Name,
		Credentials: req.Credentials,
		GroupIDs:    req.GroupIDs,
		Notes:       req.Notes,
		Status:      req.Status,
	})
	if err != nil {
		if err == service.ErrUserAccountNotFound {
			response.Error(c, 404, "Account not found")
			return
		}
		response.Error(c, 500, "Failed to update account: "+err.Error())
		return
	}

	response.Success(c, account)
}

// Delete deletes an account
// DELETE /api/v1/user/accounts/:id
func (h *UserAccountHandler) Delete(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	err = h.userAccountService.Delete(c.Request.Context(), subject.UserID, id)
	if err != nil {
		if err == service.ErrUserAccountNotFound {
			response.Error(c, 404, "Account not found")
			return
		}
		response.Error(c, 500, "Failed to delete account")
		return
	}

	response.Success(c, gin.H{"message": "Account deleted"})
}

// GetAvailableGroups returns groups that the user can bind
// GET /api/v1/user/accounts/available-groups
func (h *UserAccountHandler) GetAvailableGroups(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	groups, err := h.userAccountService.GetAvailableGroups(c.Request.Context(), subject.UserID)
	if err != nil {
		response.Error(c, 500, "Failed to get groups")
		return
	}

	response.Success(c, groups)
}
