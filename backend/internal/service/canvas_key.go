package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// CanvasAPIKeyName 是给无限画布预留的专用 API key 名称。
// 每个用户最多一个，按名称幂等复用。
const CanvasAPIKeyName = "infinite-canvas"

// ErrCanvasKeyUnavailable 表示当前上下文无法提供画布专用 key。
var ErrCanvasKeyUnavailable = errors.New("canvas api key unavailable")

// EnsureCanvasKey 幂等确保该用户名下存在无限画布专用 API key，并返回明文。
//
// groupIDs 为管理员指定的画布分组（第一个为主分组）。指定了分组时 key 绑定整组，
// 网关会合并这些分组的模型列表，并按请求的模型在各分组里选账号（如生图 + 视频）。
// 未指定时新建的 key 绑定该用户可用的第一个分组，否则 key 无法路由到账号。
func (s *APIKeyService) EnsureCanvasKey(ctx context.Context, userID int64, groupIDs []int64) (string, error) {
	if s == nil || userID <= 0 {
		return "", ErrCanvasKeyUnavailable
	}
	existing, err := s.findCanvasKey(ctx, userID)
	if err != nil {
		return "", err
	}
	if existing != nil {
		if len(groupIDs) > 0 && !canvasKeyBoundTo(existing, groupIDs) {
			// 管理员改了画布分组：已有 key 改绑过去，Update 会校验绑定权限并刷新鉴权缓存。
			primary := groupIDs[0]
			if _, err := s.Update(ctx, existing.ID, userID, UpdateAPIKeyRequest{GroupID: &primary, GroupIDs: append([]int64(nil), groupIDs...)}); err != nil {
				return "", fmt.Errorf("rebind canvas api key: %w", err)
			}
		}
		return existing.Key, nil
	}

	req := CreateAPIKeyRequest{Name: CanvasAPIKeyName}
	if len(groupIDs) > 0 {
		// 管理员指定了画布分组：只绑这些分组，不再退回可用分组列表里的第一个。
		primary := groupIDs[0]
		req.GroupID = &primary
		req.GroupIDs = append([]int64(nil), groupIDs...)
	} else {
		req.GroupID = s.firstAvailableGroupID(ctx, userID)
	}
	created, err := s.Create(ctx, userID, req)
	if err != nil {
		return "", fmt.Errorf("create canvas api key: %w", err)
	}
	return created.Key, nil
}

func (s *APIKeyService) findCanvasKey(ctx context.Context, userID int64) (*APIKey, error) {
	keys, _, err := s.List(ctx, userID, pagination.PaginationParams{Page: 1, PageSize: 100}, APIKeyListFilters{})
	if err != nil {
		return nil, err
	}
	return matchCanvasAPIKey(keys), nil
}

// firstAvailableGroupID 取用户可绑定的第一个分组；没有则返回 nil。
func (s *APIKeyService) firstAvailableGroupID(ctx context.Context, userID int64) *int64 {
	groups, err := s.GetAvailableGroups(ctx, userID)
	if err != nil || len(groups) == 0 {
		return nil
	}
	id := groups[0].ID
	return &id
}

// matchCanvasAPIKey 按名称匹配画布专用 key，忽略大小写与首尾空白。
func matchCanvasAPIKey(keys []APIKey) *APIKey {
	for i := range keys {
		if strings.EqualFold(strings.TrimSpace(keys[i].Name), CanvasAPIKeyName) {
			return &keys[i]
		}
	}
	return nil
}

// canvasKeyBoundTo 判断 key 绑定的分组是否正好是指定的分组列表（顺序一致，第一个为主分组）。
func canvasKeyBoundTo(key *APIKey, groupIDs []int64) bool {
	got := normalizeAPIKeyGroupIDs(key.GroupID, key.GroupIDs)
	want := normalizeAPIKeyGroupIDs(nil, groupIDs)
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
