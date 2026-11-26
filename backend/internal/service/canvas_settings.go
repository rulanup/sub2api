package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
)

// SettingKeyCanvasGroupID 是无限画布专用 key 绑定的分组 ID；空或 0 表示未指定。
const SettingKeyCanvasGroupID = "canvas_group_id"

// ErrCanvasGroupInvalid 表示管理员指定的画布分组不存在或未启用。
var ErrCanvasGroupInvalid = errors.New("canvas group must be an existing active group")

// GetCanvasGroupID 读取画布分组；未配置或配置非法时返回 0。
func (s *SettingService) GetCanvasGroupID(ctx context.Context) int64 {
	if s == nil || s.settingRepo == nil {
		return 0
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyCanvasGroupID)
	if err != nil {
		return 0
	}
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}

// SetCanvasGroupID 保存画布分组；groupID <= 0 表示取消指定，否则要求分组存在且已启用。
func (s *SettingService) SetCanvasGroupID(ctx context.Context, groupID int64) error {
	if s == nil || s.settingRepo == nil {
		return errors.New("setting service unavailable")
	}
	if groupID <= 0 {
		return s.settingRepo.Set(ctx, SettingKeyCanvasGroupID, "")
	}
	if s.defaultSubGroupReader == nil {
		return errors.New("group reader unavailable")
	}
	group, err := s.defaultSubGroupReader.GetByID(ctx, groupID)
	if err != nil || group == nil || !group.IsActive() {
		return ErrCanvasGroupInvalid
	}
	return s.settingRepo.Set(ctx, SettingKeyCanvasGroupID, strconv.FormatInt(groupID, 10))
}
