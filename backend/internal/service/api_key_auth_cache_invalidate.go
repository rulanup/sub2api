package service

import (
	"context"
	"errors"
	"fmt"
)

// InvalidateAuthCacheByKey 清除指定 API Key 的认证缓存
func (s *APIKeyService) InvalidateAuthCacheByKey(ctx context.Context, key string) {
	if key == "" {
		return
	}
	cacheKey := s.authCacheKey(key)
	s.deleteAuthCache(ctx, cacheKey)
}

// InvalidateAuthCacheByUserID 清除用户相关的 API Key 认证缓存
func (s *APIKeyService) InvalidateAuthCacheByUserID(ctx context.Context, userID int64) {
	_ = s.InvalidateAuthCacheByUserIDWithError(ctx, userID)
}

// InvalidateAuthCacheByUserIDWithError is used by post-commit repair paths that
// must report cache failures without changing the legacy invalidator interface.
func (s *APIKeyService) InvalidateAuthCacheByUserIDWithError(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return nil
	}
	keys, err := s.apiKeyRepo.ListKeysByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("list API keys for auth cache invalidation: %w", err)
	}
	var invalidateErr error
	for _, key := range keys {
		if key == "" {
			continue
		}
		invalidateErr = errors.Join(invalidateErr, s.deleteAuthCacheWithError(ctx, s.authCacheKey(key)))
	}
	return invalidateErr
}

// InvalidateAuthCacheByGroupID 清除分组相关的 API Key 认证缓存
func (s *APIKeyService) InvalidateAuthCacheByGroupID(ctx context.Context, groupID int64) {
	if groupID <= 0 {
		return
	}
	keys, err := s.apiKeyRepo.ListKeysByGroupID(ctx, groupID)
	if err != nil {
		return
	}
	s.deleteAuthCacheByKeys(ctx, keys)
}

func (s *APIKeyService) deleteAuthCacheByKeys(ctx context.Context, keys []string) {
	if len(keys) == 0 {
		return
	}
	for _, key := range keys {
		if key == "" {
			continue
		}
		s.deleteAuthCache(ctx, s.authCacheKey(key))
	}
}
