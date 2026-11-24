package domain

// 分组级系统提示词注入：与 securityaudit 的全局 prompt-audit 配置并存。
// 一个分组只属于一个平台，因此提示词是「一份文本」，注入点由请求协议自动决定。
const (
	// GroupSystemPromptMaxLength 提示词最大字符数（按 rune 计）。
	GroupSystemPromptMaxLength = 8000
)

const (
	GroupSystemPromptModePrepend  = "prepend"
	GroupSystemPromptModeIfAbsent = "if_absent"
)
