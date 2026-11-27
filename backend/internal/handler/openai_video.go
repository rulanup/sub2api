package handler

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// OpenAIVideoCreate 提交 OpenAI 兼容视频任务（POST /v1/videos），按模型在 key 的分组里选账号。
func (h *OpenAIGatewayHandler) OpenAIVideoCreate(c *gin.Context) {
	setActualUpstreamEndpoint(c, EndpointVideos)
	h.handleGrokMedia(c, service.OpenAIVideoEndpointCreate, "")
}

// OpenAIVideoStatus 查询任务状态（GET /v1/videos/{id}），只发往提交时的账号。
func (h *OpenAIGatewayHandler) OpenAIVideoStatus(c *gin.Context) {
	setActualUpstreamEndpoint(c, EndpointVideos)
	h.handleGrokMedia(c, service.OpenAIVideoEndpointStatus, service.OpenAIVideoTaskKey(c.Param("request_id")))
}

// OpenAIVideoContent 下载视频内容（GET /v1/videos/{id}/content），只发往提交时的账号。
func (h *OpenAIGatewayHandler) OpenAIVideoContent(c *gin.Context) {
	setActualUpstreamEndpoint(c, EndpointVideos)
	h.handleGrokMedia(c, service.OpenAIVideoEndpointContent, service.OpenAIVideoTaskKey(c.Param("request_id")))
}

// resolveVideoLookupOwner 找视频任务归属的账号。多分组 key 提交时绑定写在实际选中的分组下，
// 这里按主分组、再按其余分组依次查；命中后把 key 的当前分组切到该分组，
// 后续选账号与计费都按任务所在分组进行。
func (h *OpenAIGatewayHandler) resolveVideoLookupOwner(ctx context.Context, apiKey *service.APIKey, requestID string, userID int64) (int64, error) {
	candidates := videoLookupGroupIDs(apiKey)
	if len(candidates) <= 1 {
		return h.gatewayService.ResolveGrokMediaVideoRequestAccount(ctx, apiKey.GroupID, requestID, userID, apiKey.ID)
	}
	var lastErr error
	for _, gid := range candidates {
		group := apiKeyGroupByID(apiKey, gid)
		if group == nil {
			continue
		}
		id := gid
		accountID, err := h.gatewayService.ResolveGrokMediaVideoRequestAccount(ctx, &id, requestID, userID, apiKey.ID)
		if err == nil && accountID > 0 {
			apiKey.GroupID = &id
			apiKey.Group = group
			return accountID, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = service.ErrStickySessionNotFound
	}
	return 0, lastErr
}

func videoLookupGroupIDs(apiKey *service.APIKey) []int64 {
	if apiKey == nil {
		return nil
	}
	ids := make([]int64, 0, len(apiKey.GroupIDs)+1)
	seen := make(map[int64]struct{}, len(apiKey.GroupIDs)+1)
	add := func(id int64) {
		if id <= 0 {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if apiKey.GroupID != nil {
		add(*apiKey.GroupID)
	}
	for _, id := range apiKey.GroupIDs {
		add(id)
	}
	return ids
}

func apiKeyGroupByID(apiKey *service.APIKey, id int64) *service.Group {
	if apiKey.Group != nil && apiKey.Group.ID == id {
		return apiKey.Group
	}
	for _, group := range apiKey.Groups {
		if group != nil && group.ID == id {
			return group
		}
	}
	return nil
}
