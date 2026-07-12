package handler

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type PrivateAccountHandler struct {
	privateAccountService *service.PrivateAccountService
}

func NewPrivateAccountHandler(privateAccountService *service.PrivateAccountService) *PrivateAccountHandler {
	return &PrivateAccountHandler{privateAccountService: privateAccountService}
}

type createPrivateAccountRequest struct {
	Name        string          `json:"name" binding:"required"`
	Platform    string          `json:"platform" binding:"required"`
	Credentials json.RawMessage `json:"credentials" binding:"required"`
	GroupIDs    []int64         `json:"group_ids"`
	Notes       string          `json:"notes"`
}

type updatePrivateAccountRequest struct {
	Name        *string          `json:"name"`
	Credentials *json.RawMessage `json:"credentials"`
	GroupIDs    []int64          `json:"group_ids"`
	Notes       *string          `json:"notes"`
	Status      *string          `json:"status"`
}

// List returns all private accounts for the current user
// GET /api/v1/private-accounts
func (h *PrivateAccountHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	accounts, err := h.privateAccountService.ListByUserID(c.Request.Context(), subject.UserID)
	if err != nil {
		response.Error(c, 500, "Failed to list private accounts")
		return
	}

	response.Success(c, accounts)
}

// Create adds a new private account for the current user
// POST /api/v1/private-accounts
func (h *PrivateAccountHandler) Create(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req createPrivateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	account, err := h.privateAccountService.Create(c.Request.Context(), subject.UserID, &service.CreatePrivateAccountInput{
		Name:        req.Name,
		Platform:    req.Platform,
		Credentials: req.Credentials,
		GroupIDs:    req.GroupIDs,
		Notes:       req.Notes,
	})
	if err != nil {
		response.Error(c, 500, "Failed to create private account: "+err.Error())
		return
	}

	response.Success(c, account)
}

// Update updates a private account
// PUT /api/v1/private-accounts/:id
func (h *PrivateAccountHandler) Update(c *gin.Context) {
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

	var req updatePrivateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	account, err := h.privateAccountService.Update(c.Request.Context(), subject.UserID, id, &service.UpdatePrivateAccountInput{
		Name:        req.Name,
		Credentials: req.Credentials,
		GroupIDs:    req.GroupIDs,
		Notes:       req.Notes,
		Status:      req.Status,
	})
	if err != nil {
		if err == service.ErrPrivateAccountNotFound {
			response.Error(c, 404, "Account not found")
			return
		}
		response.Error(c, 500, "Failed to update private account: "+err.Error())
		return
	}

	response.Success(c, account)
}

// Delete deletes a private account
// DELETE /api/v1/private-accounts/:id
func (h *PrivateAccountHandler) Delete(c *gin.Context) {
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

	err = h.privateAccountService.Delete(c.Request.Context(), subject.UserID, id)
	if err != nil {
		if err == service.ErrPrivateAccountNotFound {
			response.Error(c, 404, "Account not found")
			return
		}
		response.Error(c, 500, "Failed to delete private account")
		return
	}

	response.Success(c, gin.H{"message": "Account deleted"})
}

// GetModels returns available models for the user's private accounts
// GET /api/v1/private-accounts/models
func (h *PrivateAccountHandler) GetModels(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	models, err := h.privateAccountService.GetAvailableModels(c.Request.Context(), subject.UserID)
	if err != nil {
		response.Error(c, 500, "Failed to get models")
		return
	}

	response.Success(c, models)
}

// GetGroups returns groups that the user can bind to their private accounts
// GET /api/v1/private-accounts/available-groups
func (h *PrivateAccountHandler) GetAvailableGroups(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	groups, err := h.privateAccountService.GetAvailableGroups(c.Request.Context(), subject.UserID)
	if err != nil {
		response.Error(c, 500, "Failed to get groups")
		return
	}

	response.Success(c, groups)
}

type privateAccountResponse struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Platform    string     `json:"platform"`
	Status      string     `json:"status"`
	GroupIDs    []int64    `json:"group_ids"`
	Notes       string     `json:"notes"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
}
