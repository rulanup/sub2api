package handler

import (
	"encoding/json"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func applySecuritySystemPrompt(body []byte, protocol string, groupID *int64, coordinator *securityaudit.Coordinator) ([]byte, error) {
	if coordinator == nil {
		return body, nil
	}
	policy := coordinator.SystemPromptPolicy(groupID)
	prompt := strings.TrimSpace(policy.Prompt)
	if !policy.Enabled || prompt == "" {
		return body, nil
	}
	mode := strings.TrimSpace(policy.Mode)
	if mode == "" {
		mode = "prepend"
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
