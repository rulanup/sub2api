package handler

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// 注入时可替换的变量占位符。
const (
	systemPromptPlaceholderModel     = "{{model}}"
	systemPromptPlaceholderGroupName = "{{group_name}}"
	systemPromptPlaceholderUserID    = "{{user_id}}"
)

// applySecuritySystemPrompt 按分组配置注入系统提示词。
// 分组未配置（三态为 inherit）时回退全局 prompt-audit 策略，保持老分组行为不变；
// 分组显式关闭时直接跳过，不再看全局配置。
func applySecuritySystemPrompt(body []byte, protocol string, apiKey *service.APIKey, model string, coordinator *securityaudit.Coordinator) ([]byte, error) {
	var group *service.Group
	userID := int64(0)
	if apiKey != nil {
		group = apiKey.Group
		userID = apiKey.UserID
	}
	prompt, mode, decided := group.SystemPromptFor()
	if !decided {
		if coordinator == nil {
			return body, nil
		}
		policy := coordinator.SystemPromptPolicy(groupIDOf(group))
		if !policy.Enabled {
			return body, nil
		}
		prompt = policy.Prompt
		mode = policy.Mode
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return body, nil
	}
	if strings.TrimSpace(mode) == "" {
		mode = "prepend"
	}
	prompt = truncateSystemPromptText(expandSystemPromptPlaceholders(prompt, model, groupNameOf(group), userID))
	if prompt == "" {
		return body, nil
	}

	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "anthropic_messages", "claude_messages", "messages":
		return injectAnthropicSystemPrompt(body, prompt, mode)
	case "openai_chat_completions", "openai_chat", "chat_completions":
		return injectChatSystemPrompt(body, prompt, mode)
	case "openai_responses", "responses", "responses_websocket":
		return injectResponsesSystemPrompt(body, prompt, mode)
	case "gemini", "gemini_generate_content":
		return injectGeminiSystemPrompt(body, prompt, mode)
	default:
		return body, nil
	}
}

func injectAnthropicSystemPrompt(body []byte, prompt, mode string) ([]byte, error) {
	existing := gjson.GetBytes(body, "system")
	if mode == "if_absent" && promptValuePresent(existing) {
		return body, nil
	}
	if !existing.Exists() || existing.Type == gjson.Null || !promptValuePresent(existing) {
		return sjson.SetBytes(body, "system", prompt)
	}
	if existing.Type == gjson.String {
		return sjson.SetBytes(body, "system", prompt+"\n\n"+existing.String())
	}
	if existing.IsArray() {
		return prependJSONArray(body, "system", existing, map[string]any{"type": "text", "text": prompt})
	}
	return sjson.SetBytes(body, "system", prompt)
}

func injectChatSystemPrompt(body []byte, prompt, mode string) ([]byte, error) {
	messages := gjson.GetBytes(body, "messages")
	if mode == "if_absent" && messages.IsArray() {
		for _, item := range messages.Array() {
			role := strings.ToLower(strings.TrimSpace(item.Get("role").String()))
			if (role == "system" || role == "developer") && promptValuePresent(item.Get("content")) {
				return body, nil
			}
		}
	}
	return prependJSONArray(body, "messages", messages, map[string]any{"role": "system", "content": prompt})
}

func injectResponsesSystemPrompt(body []byte, prompt, mode string) ([]byte, error) {
	path := "instructions"
	if response := gjson.GetBytes(body, "response"); response.Exists() && response.IsObject() {
		path = "response.instructions"
	}
	existing := gjson.GetBytes(body, path)
	if mode == "if_absent" && promptValuePresent(existing) {
		return body, nil
	}
	if !promptValuePresent(existing) {
		return sjson.SetBytes(body, path, prompt)
	}
	return sjson.SetBytes(body, path, prompt+"\n\n"+existing.String())
}

func injectGeminiSystemPrompt(body []byte, prompt, mode string) ([]byte, error) {
	path := "systemInstruction.parts"
	existing := gjson.GetBytes(body, "systemInstruction.parts")
	if !existing.Exists() {
		path = "system_instruction.parts"
		existing = gjson.GetBytes(body, path)
	}
	if mode == "if_absent" && promptValuePresent(existing) {
		return body, nil
	}
	return prependJSONArray(body, path, existing, map[string]any{"text": prompt})
}

func prependJSONArray(body []byte, path string, existing gjson.Result, value any) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	raw := encoded
	if existing.Exists() && existing.IsArray() && len(existing.Raw) >= 2 {
		if strings.TrimSpace(existing.Raw[1:len(existing.Raw)-1]) != "" {
			raw = append(append(append([]byte{}, encoded...), ','), existing.Raw[1:len(existing.Raw)-1]...)
		}
	}
	raw = append(append([]byte{'['}, raw...), ']')
	return sjson.SetRawBytes(body, path, raw)
}

func promptValuePresent(value gjson.Result) bool {
	if !value.Exists() || value.Type == gjson.Null {
		return false
	}
	if value.Type == gjson.String {
		return strings.TrimSpace(value.String()) != ""
	}
	if value.IsArray() {
		for _, item := range value.Array() {
			if promptValuePresent(item) {
				return true
			}
		}
		return false
	}
	if value.IsObject() {
		present := false
		value.ForEach(func(_, child gjson.Result) bool {
			if promptValuePresent(child) {
				present = true
				return false
			}
			return true
		})
		return present
	}
	return true
}

// groupIDOf 返回分组的 ID 指针；分组未水合时返回 nil（交由全局策略处理）。
func groupIDOf(group *service.Group) *int64 {
	if group == nil || group.ID == 0 {
		return nil
	}
	id := group.ID
	return &id
}

func groupNameOf(group *service.Group) string {
	if group == nil {
		return ""
	}
	return group.Name
}

// expandSystemPromptPlaceholders 替换注入文本中的变量占位符；
// 未识别的占位符保持原样，避免误吞管理员输入。
func expandSystemPromptPlaceholders(text, model, groupName string, userID int64) string {
	replacer := strings.NewReplacer(
		systemPromptPlaceholderModel, model,
		systemPromptPlaceholderGroupName, groupName,
		systemPromptPlaceholderUserID, strconv.FormatInt(userID, 10),
	)
	return replacer.Replace(text)
}

// truncateSystemPromptText 兜底截断，防止超长提示词撑爆上游请求。
func truncateSystemPromptText(text string) string {
	limit := domain.GroupSystemPromptMaxLength
	if limit <= 0 {
		return text
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return strings.TrimSpace(string(runes[:limit]))
}
