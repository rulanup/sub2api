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
// 已存在则直接复用，不重复创建。新建时自动绑定该用户可用的第一个分组，
// 否则 key 无法路由到账号，画布调用会直接失败。
func (s *APIKeyService) EnsureCanvasKey(ctx context.Context, userID int64) (string, error) {
	if s == nil || userID <= 0 {
		return "", ErrCanvasKeyUnavailable
	}
	existing, err := s.findCanvasKey(ctx, userID)
	if err != nil {
		return "", err
	}
	if existing != nil {
		return existing.Key, nil
	}

	req := CreateAPIKeyRequest{Name: CanvasAPIKeyName}
	req.GroupID = s.firstAvailableGroupID(ctx, userID)
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
