package handler

import (
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/account"
	"github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

type UserGroupHandler struct {
	client *dbent.Client
}

type userGroupRequest struct {
	Name        string `json:"name" binding:"required,max=100"`
	Description string `json:"description"`
	Platform    string `json:"platform" binding:"required,max=50"`
}

type userGroupResponse struct {
	*dbent.Group
	AccountCount            int `json:"account_count"`
	ActiveAccountCount      int `json:"active_account_count"`
	RateLimitedAccountCount int `json:"rate_limited_account_count"`
}

func NewUserGroupHandler(client *dbent.Client) *UserGroupHandler {
	return &UserGroupHandler{client: client}
}

func privateGroupOwner(c *gin.Context) (int64, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return 0, false
	}
	return subject.UserID, true
}

func (h *UserGroupHandler) List(c *gin.Context) {
	ownerID, ok := privateGroupOwner(c)
	if !ok {
		return
	}
	groups, err := h.client.Group.Query().
		Where(group.IsPrivateEQ(true), group.OwnerUserIDEQ(ownerID)).
		WithAccounts(func(q *dbent.AccountQuery) {
			q.Where(account.UserIDEQ(ownerID))
		}).
		Order(dbent.Asc(group.FieldName)).
		All(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	now := time.Now()
	items := make([]userGroupResponse, 0, len(groups))
	for _, item := range groups {
		counts := userGroupResponse{Group: item, AccountCount: len(item.Edges.Accounts)}
		for _, privateAccount := range item.Edges.Accounts {
			limited := (privateAccount.RateLimitResetAt != nil && privateAccount.RateLimitResetAt.After(now)) ||
				(privateAccount.OverloadUntil != nil && privateAccount.OverloadUntil.After(now)) ||
				(privateAccount.TempUnschedulableUntil != nil && privateAccount.TempUnschedulableUntil.After(now))
			if limited {
				counts.RateLimitedAccountCount++
			}
			if privateAccount.Status == domain.StatusActive && privateAccount.Schedulable && !limited &&
				(privateAccount.ExpiresAt == nil || privateAccount.ExpiresAt.After(now) || !privateAccount.AutoPauseOnExpired) {
				counts.ActiveAccountCount++
			}
		}
		items = append(items, counts)
	}
	response.Success(c, items)
}

func (h *UserGroupHandler) Create(c *gin.Context) {
	ownerID, ok := privateGroupOwner(c)
	if !ok {
		return
	}
	var req userGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Platform = strings.TrimSpace(req.Platform)
	if req.Name == "" || req.Platform == "" {
		response.BadRequest(c, "Name and platform are required")
		return
	}
	created, err := h.client.Group.Create().
		SetName(req.Name).
		SetDescription(req.Description).
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
	response.Success(c, created)
}

func (h *UserGroupHandler) Update(c *gin.Context) {
	ownerID, id, ok := privateGroupParams(c)
	if !ok {
		return
	}
	var req userGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Platform = strings.TrimSpace(req.Platform)
	updated, err := h.client.Group.Update().
		Where(group.IDEQ(id), group.IsPrivateEQ(true), group.OwnerUserIDEQ(ownerID)).
		SetName(req.Name).
		SetDescription(req.Description).
		SetPlatform(req.Platform).
		Save(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if updated == 0 {
		response.Error(c, 404, "Group not found")
		return
	}
	item, err := h.client.Group.Get(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *UserGroupHandler) Delete(c *gin.Context) {
	ownerID, id, ok := privateGroupParams(c)
	if !ok {
		return
	}
	deleted, err := h.client.Group.Delete().
		Where(group.IDEQ(id), group.IsPrivateEQ(true), group.OwnerUserIDEQ(ownerID)).
		Exec(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if deleted == 0 {
		response.Error(c, 404, "Group not found")
		return
	}
	response.Success(c, gin.H{"message": "Group deleted"})
}

func privateGroupParams(c *gin.Context) (int64, int64, bool) {
	ownerID, ok := privateGroupOwner(c)
	if !ok {
		return 0, 0, false
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid group ID")
		return 0, 0, false
	}
	return ownerID, id, true
}
