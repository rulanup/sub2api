package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type UserAccountHandler struct {
	adminAccountHandler *admin.AccountHandler
	client              *dbent.Client
	accountTestService  *service.AccountTestService
}

func NewUserAccountHandler(adminAccountHandler *admin.AccountHandler, client *dbent.Client, accountTestService *service.AccountTestService) *UserAccountHandler {
	return &UserAccountHandler{adminAccountHandler: adminAccountHandler, client: client, accountTestService: accountTestService}
}

type quickAddPrivateAccountRequest struct {
	Platform string `json:"platform" binding:"required"`
	BaseURL  string `json:"base_url" binding:"required"`
	APIKey   string `json:"api_key" binding:"required"`
}

func privateAccountOwner(c *gin.Context) (int64, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return 0, false
	}
	return subject.UserID, true
}

func (h *UserAccountHandler) bindQuickAddRequest(c *gin.Context) (*quickAddPrivateAccountRequest, bool) {
	var req quickAddPrivateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return nil, false
	}
	req.Platform = strings.TrimSpace(req.Platform)
	req.BaseURL = strings.TrimRight(strings.TrimSpace(req.BaseURL), "/")
	req.APIKey = strings.TrimSpace(req.APIKey)
	if req.Platform != service.PlatformOpenAI && req.Platform != service.PlatformAnthropic && req.Platform != service.PlatformGemini {
		response.BadRequest(c, "Quick add supports OpenAI, Anthropic, and Gemini")
		return nil, false
	}
	parsed, err := url.ParseRequestURI(req.BaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		response.BadRequest(c, "Invalid upstream URL")
		return nil, false
	}
	return &req, true
}

func (h *UserAccountHandler) fetchQuickAddModels(c *gin.Context, req *quickAddPrivateAccountRequest) ([]string, error) {
	if h.accountTestService == nil {
		return nil, errors.New("account test service is not configured")
	}
	return h.accountTestService.FetchUpstreamSupportedModels(c.Request.Context(), &service.Account{
		Platform: req.Platform,
		Type:     service.AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url": req.BaseURL,
			"api_key":  req.APIKey,
		},
	})
}

func (h *UserAccountHandler) writeModelSyncError(c *gin.Context, err error) {
	var syncErr *service.UpstreamModelSyncError
	if errors.As(err, &syncErr) {
		if syncErr.Kind == service.UpstreamModelSyncErrorConfiguration || syncErr.Kind == service.UpstreamModelSyncErrorUnsupported {
			response.BadRequest(c, syncErr.SafeMessage())
		} else {
			response.Error(c, http.StatusBadGateway, syncErr.SafeMessage())
		}
		return
	}
	response.Error(c, http.StatusBadGateway, "Failed to fetch upstream models")
}

func (h *UserAccountHandler) PreviewModels(c *gin.Context) {
	if _, ok := privateAccountOwner(c); !ok {
		return
	}
	req, ok := h.bindQuickAddRequest(c)
	if !ok {
		return
	}
	models, err := h.fetchQuickAddModels(c, req)
	if err != nil {
		h.writeModelSyncError(c, err)
		return
	}
	response.Success(c, gin.H{"models": models})
}

func (h *UserAccountHandler) QuickAdd(c *gin.Context) {
	ownerID, ok := privateAccountOwner(c)
	if !ok {
		return
	}
	req, ok := h.bindQuickAddRequest(c)
	if !ok {
		return
	}
	models, err := h.fetchQuickAddModels(c, req)
	if err != nil {
		h.writeModelSyncError(c, err)
		return
	}

	parsedURL, _ := url.Parse(req.BaseURL)
	host := strings.TrimPrefix(parsedURL.Hostname(), "www.")
	if host == "" {
		host = req.Platform
	}
	suffix := time.Now().UnixMilli()
	name := fmt.Sprintf("%s-%s-%d", req.Platform, host, suffix)
	mapping := make(map[string]any, len(models))
	for _, model := range models {
		mapping[model] = model
	}
	credentials := map[string]any{"base_url": req.BaseURL, "api_key": req.APIKey, "model_mapping": mapping}
	if req.Platform == service.PlatformGemini {
		credentials["tier_id"] = "aistudio_free"
	}

	tx, err := h.client.Tx(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Failed to start quick add")
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	createdGroup, err := tx.Group.Create().
		SetName(name).
		SetDescription("快速添加的私人渠道").
		SetPlatform(req.Platform).
		SetStatus(domain.StatusActive).
		SetSubscriptionType(domain.SubscriptionTypeStandard).
		SetRateMultiplier(0).
		SetImageRateIndependent(false).
		SetImageRateMultiplier(0).
		SetVideoRateIndependent(false).
		SetVideoRateMultiplier(0).
		SetIsExclusive(false).
		SetIsPrivate(true).
		SetOwnerUserID(ownerID).
		Save(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	createdAccount, err := tx.Account.Create().
		SetName(name).
		SetPlatform(req.Platform).
		SetType(service.AccountTypeAPIKey).
		SetCredentials(credentials).
		SetUserID(ownerID).
		SetStatus(domain.StatusActive).
		SetSchedulable(true).
		Save(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if _, err = tx.AccountGroup.Create().SetAccountID(createdAccount.ID).SetGroupID(createdGroup.ID).Save(c.Request.Context()); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if err = tx.Commit(); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	committed = true
	response.Success(c, gin.H{
		"group":   gin.H{"id": createdGroup.ID, "name": createdGroup.Name},
		"account": gin.H{"id": createdAccount.ID, "name": createdAccount.Name},
		"models":  models,
	})
}

// List proxies to admin account list with user_id filter
// GET /api/v1/user/accounts
func (h *UserAccountHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	query := c.Request.URL.Query()
	query.Set("user_id", strconv.FormatInt(subject.UserID, 10))
	c.Request.URL.RawQuery = query.Encode()
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
	if !h.validatePrivateGroups(c, subject.UserID) {
		return
	}
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
	if !h.validatePrivateGroups(c, subject.UserID) {
		return
	}
	h.adminAccountHandler.Update(c)
}

func (h *UserAccountHandler) validatePrivateGroups(c *gin.Context, ownerID int64) bool {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.BadRequest(c, "Invalid request")
		return false
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	var payload struct {
		Platform string   `json:"platform"`
		GroupIDs *[]int64 `json:"group_ids"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.GroupIDs == nil || len(*payload.GroupIDs) == 0 {
		return true
	}
	platform := payload.Platform
	if platform == "" && c.Param("id") != "" {
		accountID, parseErr := strconv.ParseInt(c.Param("id"), 10, 64)
		if parseErr != nil {
			response.BadRequest(c, "Invalid account ID")
			return false
		}
		account, queryErr := h.client.Account.Get(c.Request.Context(), accountID)
		if queryErr != nil || account.UserID == nil || *account.UserID != ownerID {
			response.Error(c, 403, "Access denied")
			return false
		}
		platform = account.Platform
	}
	count, err := h.client.Group.Query().Where(
		group.IDIn(*payload.GroupIDs...),
		group.IsPrivateEQ(true),
		group.OwnerUserIDEQ(ownerID),
		group.PlatformEQ(platform),
	).Count(c.Request.Context())
	if err != nil || count != len(*payload.GroupIDs) {
		response.BadRequest(c, "Private accounts can only bind to your own groups on the same platform")
		return false
	}
	return true
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
