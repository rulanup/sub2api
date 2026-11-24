package domain

import "strings"

// 分组级系统提示词注入：与 securityaudit 的全局 prompt-audit 配置并存。
// 分组显式配置优先；未配置（Enabled 为 nil）时回退全局配置，保证老分组行为不变。
const (
	GroupSystemPromptProtocolAnthropic = "anthropic_messages"
	GroupSystemPromptProtocolOpenAI    = "openai"
	GroupSystemPromptProtocolGemini    = "gemini"
)

// GroupSystemPromptMaxLength 单个协议提示词的最大字符数（按 rune 计）。
const GroupSystemPromptMaxLength = 8000

const (
	GroupSystemPromptModePrepend  = "prepend"
	GroupSystemPromptModeIfAbsent = "if_absent"
)

// GroupSystemPrompts 是「协议 → 提示词文本」映射；空文本表示该协议不注入。
type GroupSystemPrompts map[string]string

// NormalizeGroupSystemPromptProtocol 把网关协议标识归一化为分组配置键。
func NormalizeGroupSystemPromptProtocol(protocol string) string {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "anthropic_messages", "claude_messages", "messages", "openai_anthropic_messages":
		return GroupSystemPromptProtocolAnthropic
	case "openai_responses", "responses", "responses_websocket",
		"openai_chat_completions", "openai_chat", "chat_completions":
		return GroupSystemPromptProtocolOpenAI
	case "gemini", "gemini_generate_content", "gemini_v1beta":
		return GroupSystemPromptProtocolGemini
	default:
		return ""
	}
}
