package service

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

// 分组级三态开关：继承全局（默认）/ 强制开启 / 强制关闭。
const (
	GroupToggleInherit = "inherit"
	GroupToggleOn      = "on"
	GroupToggleOff     = "off"
)

// ResolveGroupToggle 解析分组级三态开关；空字符串表示调用方未提供（不修改）。
func ResolveGroupToggle(value string) (state string, provided bool, err error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return "", false, nil
	case GroupToggleInherit:
		return GroupToggleInherit, true, nil
	case GroupToggleOn, "true", "1":
		return GroupToggleOn, true, nil
	case GroupToggleOff, "false", "0":
		return GroupToggleOff, true, nil
	default:
		return "", false, fmt.Errorf("无效的开关值 %q（可选 inherit/on/off）", value)
	}
}

// GroupTogglePtr 把三态字符串映射为数据库中的 *bool（inherit → nil）。
func GroupTogglePtr(state string) *bool {
	switch state {
	case GroupToggleOn:
		value := true
		return &value
	case GroupToggleOff:
		value := false
		return &value
	default:
		return nil
	}
}

// GroupToggleState 把数据库中的 *bool 还原为三态字符串。
func GroupToggleState(value *bool) string {
	switch {
	case value == nil:
		return GroupToggleInherit
	case *value:
		return GroupToggleOn
	default:
		return GroupToggleOff
	}
}

// cloneBoolPtr 复制布尔指针，避免快照与实体共享同一地址。
func cloneBoolPtr(value *bool) *bool {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

// NormalizeGroupSystemPrompt 校验并归一化分组提示词（去首尾空白 + 长度上限）。
func NormalizeGroupSystemPrompt(text string) (string, error) {
	trimmed := strings.TrimSpace(text)
	if utf8.RuneCountInString(trimmed) > domain.GroupSystemPromptMaxLength {
		return "", fmt.Errorf("系统提示词超过 %d 字符上限", domain.GroupSystemPromptMaxLength)
	}
	return trimmed, nil
}

// NormalizeGroupSystemPromptMode 归一化注入位置，默认 prepend。
func NormalizeGroupSystemPromptMode(mode string) string {
	if strings.EqualFold(strings.TrimSpace(mode), domain.GroupSystemPromptModeIfAbsent) {
		return domain.GroupSystemPromptModeIfAbsent
	}
	return domain.GroupSystemPromptModePrepend
}

// SystemPromptFor 返回分组配置的系统提示词。
// decided=false 表示分组未配置，调用方应回退全局 prompt-audit 配置；
// decided=true 且 text 为空表示分组显式关闭或未填写，均不注入。
// 提示词与协议无关：一个分组只有一个平台，注入位置由请求协议在注入时决定。
func (g *Group) SystemPromptFor() (text, mode string, decided bool) {
	if g == nil || g.SystemPromptEnabled == nil {
		return "", "", false
	}
	if !*g.SystemPromptEnabled {
		return "", "", true
	}
	return strings.TrimSpace(g.SystemPrompt), NormalizeGroupSystemPromptMode(g.SystemPromptMode), true
}

// AuditEnabledOverride 返回分组级审查开关的三态覆盖。
func (g *Group) AuditEnabledOverride() (enabled bool, decided bool) {
	if g == nil || g.SecurityAuditEnabled == nil {
		return false, false
	}
	return *g.SecurityAuditEnabled, true
}

// ApplyGroupSystemPromptInput 把管理端输入应用到分组；nil 字段表示不修改。
func ApplyGroupSystemPromptInput(group *Group, enabledState *string, prompt *string, mode *string, auditState *string) error {
	if group == nil {
		return nil
	}
	if enabledState != nil {
		state, _, err := ResolveGroupToggle(*enabledState)
		if err != nil {
			return err
		}
		group.SystemPromptEnabled = GroupTogglePtr(state)
	}
	if prompt != nil {
		normalized, err := NormalizeGroupSystemPrompt(*prompt)
		if err != nil {
			return err
		}
		group.SystemPrompt = normalized
	}
	if mode != nil {
		group.SystemPromptMode = NormalizeGroupSystemPromptMode(*mode)
	}
	if auditState != nil {
		state, _, err := ResolveGroupToggle(*auditState)
		if err != nil {
			return err
		}
		group.SecurityAuditEnabled = GroupTogglePtr(state)
	}
	return nil
}
