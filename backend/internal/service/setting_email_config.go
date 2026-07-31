package service

import (
	"context"
	"fmt"
	"strings"
)

// 邮件配置一键导出/导入：仅涉及邮件相关设置键，不触碰其他系统设置。

// emailConfigSettingKeys 邮件相关设置键白名单（导出与导入均只处理这些键）。
var emailConfigSettingKeys = []string{
	SettingKeyMailProvider,
	SettingKeySMTPHost,
	SettingKeySMTPPort,
	SettingKeySMTPUsername,
	SettingKeySMTPPassword,
	SettingKeySMTPFrom,
	SettingKeySMTPFromName,
	SettingKeySMTPUseTLS,
	SettingKeyResendAPIKey,
	SettingKeyResendFromEmail,
	SettingKeyResendFromName,
	SettingKeyResendBaseURL,
}

// emailConfigSensitiveKeys 敏感设置键：
//   - 导出时默认排除（除非 includeSecrets=true）
//   - 导入时仅当值为非空时才覆盖，避免导入文件丢失密钥导致清空现有配置
var emailConfigSensitiveKeys = map[string]struct{}{
	SettingKeySMTPPassword: {},
	SettingKeyResendAPIKey: {},
}

// ExportEmailConfig 导出邮件相关设置键值对。
// includeSecrets=false（默认）时排除 SMTP 密码与 Resend API Key 等敏感键；
// includeSecrets=true 时包含全部键（用于完整迁移）。
func (s *SettingService) ExportEmailConfig(ctx context.Context, includeSecrets bool) (map[string]string, error) {
	values, err := s.settingRepo.GetMultiple(ctx, emailConfigSettingKeys)
	if err != nil {
		return nil, fmt.Errorf("get email settings: %w", err)
	}

	out := make(map[string]string)
	for _, key := range emailConfigSettingKeys {
		value, ok := values[key]
		if !ok || strings.TrimSpace(value) == "" {
			continue
		}
		if !includeSecrets {
			if _, sensitive := emailConfigSensitiveKeys[key]; sensitive {
				continue
			}
		}
		out[key] = value
	}
	return out, nil
}

// ImportEmailConfig 导入邮件相关设置键值对。
// 只接受 emailConfigSettingKeys 白名单内的键，其余忽略；
// 敏感键仅当导入值为非空时才覆盖，避免误清空现有密钥。
// 返回实际写入的设置项数量。
func (s *SettingService) ImportEmailConfig(ctx context.Context, values map[string]string) (int, error) {
	if len(values) == 0 {
		return 0, nil
	}

	updates := make(map[string]string)
	for _, key := range emailConfigSettingKeys {
		value, ok := values[key]
		if !ok {
			continue
		}
		if _, sensitive := emailConfigSensitiveKeys[key]; sensitive && strings.TrimSpace(value) == "" {
			continue
		}
		updates[key] = value
	}

	if len(updates) == 0 {
		return 0, nil
	}
	if err := s.settingRepo.SetMultiple(ctx, updates); err != nil {
		return 0, fmt.Errorf("import email settings: %w", err)
	}
	if s.onUpdate != nil {
		s.onUpdate()
	}
	return len(updates), nil
}
