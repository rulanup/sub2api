package handler

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type systemPromptTestEngine struct {
	policy securityaudit.SystemPromptPolicy
}

func (e systemPromptTestEngine) EffectiveMode() securityaudit.Mode                    { return securityaudit.ModeOff }
func (e systemPromptTestEngine) Enqueue(context.Context, securityaudit.Request) error { return nil }
func (e systemPromptTestEngine) Evaluate(context.Context, securityaudit.Request) (*securityaudit.PromptDecision, error) {
	return nil, nil
}
func (e systemPromptTestEngine) SystemPromptPolicy(*int64) securityaudit.SystemPromptPolicy {
	return e.policy
}

func testSystemPromptCoordinator(mode string) *securityaudit.Coordinator {
	return securityaudit.NewCoordinator(nil, systemPromptTestEngine{policy: securityaudit.SystemPromptPolicy{
		Enabled: true, Prompt: "ADMIN SAFETY POLICY", Mode: mode,
	}})
}

func testSystemPromptGroupID() *int64 {
	groupID := int64(1)
	return &groupID
}

func TestApplySecuritySystemPrompt_AnthropicPrepend(t *testing.T) {
	body, err := applySecuritySystemPrompt(
		[]byte(`{"model":"claude","system":"client system","messages":[{"role":"user","content":"hello"}]}`),
		"anthropic_messages", testSystemPromptGroupID(), testSystemPromptCoordinator("prepend"),
	)
	require.NoError(t, err)
	require.Equal(t, "ADMIN SAFETY POLICY\n\nclient system", gjson.GetBytes(body, "system").String())
}

func TestApplySecuritySystemPrompt_ChatIfAbsentPreservesClientSystem(t *testing.T) {
	body, err := applySecuritySystemPrompt(
		[]byte(`{"model":"gpt","messages":[{"role":"system","content":"client system"},{"role":"user","content":"hello"}]}`),
		"openai_chat_completions", testSystemPromptGroupID(), testSystemPromptCoordinator("if_absent"),
	)
	require.NoError(t, err)
	require.Equal(t, "client system", gjson.GetBytes(body, "messages.0.content").String())
	require.Equal(t, 2, len(gjson.GetBytes(body, "messages").Array()))
}

func TestApplySecuritySystemPrompt_ResponsesPrepend(t *testing.T) {
	body, err := applySecuritySystemPrompt(
		[]byte(`{"model":"gpt","instructions":"client instructions","input":"hello"}`),
		"openai_responses", testSystemPromptGroupID(), testSystemPromptCoordinator("prepend"),
	)
	require.NoError(t, err)
	require.Equal(t, "ADMIN SAFETY POLICY\n\nclient instructions", gjson.GetBytes(body, "instructions").String())
}

func TestApplySecuritySystemPrompt_GeminiPrepend(t *testing.T) {
	body, err := applySecuritySystemPrompt(
		[]byte(`{"systemInstruction":{"parts":[{"text":"client system"}]},"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`),
		"gemini", testSystemPromptGroupID(), testSystemPromptCoordinator("prepend"),
	)
	require.NoError(t, err)
	require.Equal(t, "ADMIN SAFETY POLICY", gjson.GetBytes(body, "systemInstruction.parts.0.text").String())
	require.Equal(t, "client system", gjson.GetBytes(body, "systemInstruction.parts.1.text").String())
}

func TestApplySecuritySystemPrompt_PreservesLargeJSONInteger(t *testing.T) {
	body, err := applySecuritySystemPrompt(
		[]byte(`{"model":"gpt","metadata":{"external_id":9007199254740993},"messages":[{"role":"user","content":"hello"}]}`),
		"openai_chat_completions", testSystemPromptGroupID(), testSystemPromptCoordinator("prepend"),
	)
	require.NoError(t, err)
	require.Contains(t, string(body), `"external_id":9007199254740993`)
}

func TestApplySecuritySystemPrompt_ResponsesWebSocketNestedPayload(t *testing.T) {
	body, err := applySecuritySystemPrompt(
		[]byte(`{"type":"response.create","response":{"model":"gpt","instructions":"client instructions","input":"hello"}}`),
		"responses_websocket", testSystemPromptGroupID(), testSystemPromptCoordinator("prepend"),
	)
	require.NoError(t, err)
	require.Equal(t, "ADMIN SAFETY POLICY\n\nclient instructions", gjson.GetBytes(body, "response.instructions").String())
}

func TestApplySecuritySystemPrompt_IfAbsentTreatsEmptyPromptAsAbsent(t *testing.T) {
	body, err := applySecuritySystemPrompt(
		[]byte(`{"model":"gpt","messages":[{"role":"system","content":""},{"role":"user","content":"hello"}]}`),
		"openai_chat_completions", testSystemPromptGroupID(), testSystemPromptCoordinator("if_absent"),
	)
	require.NoError(t, err)
	require.Equal(t, "ADMIN SAFETY POLICY", gjson.GetBytes(body, "messages.0.content").String())
}
