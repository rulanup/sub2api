package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
)

// SettingKeyCanvasGroupID 保存无限画布专用 key 绑定的分组：逗号分隔的分组 ID 列表，第一个为主分组。
// 兼容旧版本只存单个 ID 的写法；空表示未指定。
const SettingKeyCanvasGroupID = "canvas_group_id"

// ErrCanvasGroupInvalid 表示管理员指定的画布分组不存在或未启用。
var ErrCanvasGroupInvalid = errors.New("canvas group must be an existing active group")

// ErrCanvasGroupPlatformMismatch 表示画布分组跨了平台；一把 API key 绑定的分组必须属于同一平台。
var ErrCanvasGroupPlatformMismatch = errors.New("canvas groups must use the same platform")

// ParseCanvasGroupIDs 解析画布分组设置：按逗号分隔，忽略非法与重复项，保持原有顺序。
func ParseCanvasGroupIDs(raw string) []int64 {
	ids := make([]int64, 0, 4)
	seen := make(map[int64]struct{}, 4)
	for _, part := range strings.Split(raw, ",") {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func formatCanvasGroupIDs(ids []int64) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, strconv.FormatInt(id, 10))
	}
	return strings.Join(parts, ",")
}

// GetCanvasGroupIDs 读取画布分组；未配置或读取失败时返回空。
func (s *SettingService) GetCanvasGroupIDs(ctx context.Context) []int64 {
	if s == nil || s.settingRepo == nil {
		return nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyCanvasGroupID)
	if err != nil {
		return nil
	}
	ids := ParseCanvasGroupIDs(raw)
	if s.defaultSubGroupReader == nil || len(ids) == 0 {
		return ids
	}
	// 分组被删除或停用后跳过它，否则 key 会改绑到不存在的分组，后台也无法取消勾选。
	// 只在确认分组不存在或未启用时跳过；查询临时出错时保留，避免 key 被误改绑。
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		group, getErr := s.defaultSubGroupReader.GetByID(ctx, id)
		switch {
		case getErr == nil && group != nil && group.IsActive():
			out = append(out, id)
		case getErr == nil, errors.Is(getErr, ErrGroupNotFound):
			// 已删除或未启用：跳过。
		default:
			out = append(out, id)
		}
	}
	return out
}

// SetCanvasGroupIDs 保存画布分组；为空表示取消指定，否则要求每个分组存在、已启用且属于同一平台。
func (s *SettingService) SetCanvasGroupIDs(ctx context.Context, groupIDs []int64) error {
	if s == nil || s.settingRepo == nil {
		return errors.New("setting service unavailable")
	}
	ids := make([]int64, 0, len(groupIDs))
	seen := make(map[int64]struct{}, len(groupIDs))
	for _, id := range groupIDs {
		if id <= 0 {
			return ErrCanvasGroupInvalid
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return s.settingRepo.Set(ctx, SettingKeyCanvasGroupID, "")
	}
	if s.defaultSubGroupReader == nil {
		return errors.New("group reader unavailable")
	}
	platform := ""
	for _, id := range ids {
		group, err := s.defaultSubGroupReader.GetByID(ctx, id)
		if err != nil || group == nil || !group.IsActive() {
			return ErrCanvasGroupInvalid
		}
		if platform == "" {
			platform = group.Platform
		} else if group.Platform != platform {
			return ErrCanvasGroupPlatformMismatch
		}
	}
	return s.settingRepo.Set(ctx, SettingKeyCanvasGroupID, formatCanvasGroupIDs(ids))
}
