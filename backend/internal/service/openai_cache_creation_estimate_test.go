package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackfillOpenAICacheCreationFromAnthropicBody_UpstreamWins(t *testing.T) {
	body := []byte(`{"model":"gpt-5.4","system":[{"type":"text","text":"cached instructions","cache_control":{"type":"ephemeral"}}],"messages":[{"role":"user","content":"hello"}]}`)
	usage := OpenAIUsage{InputTokens: 100, CacheCreationInputTokens: 17, CacheCreationReported: true}

	BackfillOpenAICacheCreationFromAnthropicBody(body, "gpt-5.4", &usage)

	require.Equal(t, 17, usage.CacheCreationInputTokens)
}

func TestBackfillOpenAICacheCreationFromAnthropicBody_ExplicitUpstreamZeroWins(t *testing.T) {
	body := []byte(`{"model":"gpt-5.4","system":[{"type":"text","text":"cached instructions","cache_control":{"type":"ephemeral"}}],"messages":[{"role":"user","content":"hello"}]}`)
	usage := OpenAIUsage{InputTokens: 100, CacheCreationReported: true}

	BackfillOpenAICacheCreationFromAnthropicBody(body, "gpt-5.4", &usage)

	require.Zero(t, usage.CacheCreationInputTokens)
}

func TestBackfillOpenAICacheCreationFromAnthropicBody_EstimatesMissingUpstreamUsage(t *testing.T) {
	body := []byte(`{"model":"gpt-5.4","tools":[{"name":"lookup","description":"cached tool schema","input_schema":{"type":"object"},"cache_control":{"type":"ephemeral"}}],"system":[{"type":"text","text":"cached instructions","cache_control":{"type":"ephemeral"}}],"messages":[{"role":"user","content":[{"type":"text","text":"cached user prefix","cache_control":{"type":"ephemeral"}},{"type":"text","text":"uncached tail"}]}]}`)
	usage := OpenAIUsage{InputTokens: 100}

	BackfillOpenAICacheCreationFromAnthropicBody(body, "gpt-5.4", &usage)

	require.Positive(t, usage.CacheCreationInputTokens)
	require.LessOrEqual(t, usage.CacheCreationInputTokens, usage.InputTokens)
	require.Equal(t, 100, usage.InputTokens, "cache creation must partition, not increase, total input")
}

func TestBackfillOpenAICacheCreationFromAnthropicBody_NoMarkerLeavesZero(t *testing.T) {
	body := []byte(`{"model":"gpt-5.4","system":"plain instructions","messages":[{"role":"user","content":"hello"}]}`)
	usage := OpenAIUsage{InputTokens: 100}

	BackfillOpenAICacheCreationFromAnthropicBody(body, "gpt-5.4", &usage)

	require.Zero(t, usage.CacheCreationInputTokens)
}

func TestBackfillOpenAICacheCreationFromAnthropicBody_CapsToNonReadInput(t *testing.T) {
	body := []byte(`{"model":"gpt-5.4","system":[{"type":"text","text":"a very long cached instruction prefix repeated repeated repeated repeated repeated repeated","cache_control":{"type":"ephemeral"}}],"messages":[{"role":"user","content":"hello"}]}`)
	usage := OpenAIUsage{InputTokens: 10, CacheReadInputTokens: 8}

	BackfillOpenAICacheCreationFromAnthropicBody(body, "gpt-5.4", &usage)

	require.Equal(t, 2, usage.CacheCreationInputTokens)
}
