package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDescribeModel(t *testing.T) {
	chatOnly := []string{ModelProtocolChat}
	openAIText := []string{ModelProtocolResponses, ModelProtocolChat}
	cases := []struct {
		name       string
		model      string
		capability string
		protocols  []string
		known      bool
	}{
		{"gpt-image", "gpt-image-2", ModelCapabilityImage, []string{ModelProtocolImages}, true},
		{"gpt-image-flare", "gpt-image-2.6-flare", ModelCapabilityImage, []string{ModelProtocolImages}, true},
		{"gemini image", "gemini-3-pro-image", ModelCapabilityImage, []string{ModelProtocolGemini, ModelProtocolChat}, true},
		{"gemini flash image", "gemini-3.1-flash-image-preview", ModelCapabilityImage, []string{ModelProtocolGemini, ModelProtocolChat}, true},
		{"grok imagine", "grok-imagine-1.0", ModelCapabilityImage, []string{ModelProtocolImages}, true},
		{"grok image", "grok-2-image", ModelCapabilityImage, []string{ModelProtocolImages}, true},
		{"seedream", "seedream-4.0", ModelCapabilityImage, []string{ModelProtocolImages}, true},
		{"imagen", "imagen-4.0-generate", ModelCapabilityImage, []string{ModelProtocolImages}, true},
		{"grok video", "grok-imagine-video-1.5", ModelCapabilityVideo, nil, true},
		{"minimax video", "zzdh-Minimax-h3-480p", ModelCapabilityVideo, nil, true},
		{"hailuo", "MiniMax-Hailuo-02", ModelCapabilityVideo, nil, true},
		{"veo", "veo-3.1-generate", ModelCapabilityVideo, nil, true},
		{"seedance", "doubao-seedance-1-0-pro", ModelCapabilityVideo, nil, true},
		{"sora", "sora-2", ModelCapabilityVideo, nil, true},
		{"wan", "wan2.2-t2v", ModelCapabilityVideo, nil, true},
		{"gpt text", "gpt-5.6", ModelCapabilityText, openAIText, true},
		{"o series", "o4-mini", ModelCapabilityText, openAIText, true},
		{"codex", "codex-auto-review", ModelCapabilityText, openAIText, true},
		{"claude", "claude-sonnet-4-5", ModelCapabilityText, chatOnly, true},
		{"gemini text", "gemini-3-pro", ModelCapabilityText, chatOnly, true},
		{"deepseek", "deepseek-v4.1-flash", ModelCapabilityText, chatOnly, true},
		{"vendor prefixed", "deepseek/deepseek-v4-flash", ModelCapabilityText, chatOnly, true},
		{"minimax text", "MiniMax-M2", ModelCapabilityText, chatOnly, true},
		{"qwen", "qwen3-max", ModelCapabilityText, chatOnly, true},
		{"tts", "gpt-4o-mini-tts", "", nil, false},
		{"embedding", "text-embedding-3-large", "", nil, false},
		{"unknown", "my-private-alias", "", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := DescribeModel(tc.model)
			require.Equal(t, tc.known, ok)
			require.Equal(t, tc.capability, got.Capability)
			require.Equal(t, tc.protocols, got.Protocols)
		})
	}
}

func TestDescribeModelIsCaseInsensitive(t *testing.T) {
	got, ok := DescribeModel("  GPT-Image-2  ")
	require.True(t, ok)
	require.Equal(t, ModelCapabilityImage, got.Capability)
}

// TestDescribeModelNeverMarksTextAsImage 覆盖审查重点 #2：
// 文本模型不得被判成图片模型，否则客户端会挂错调用脚本。
func TestDescribeModelNeverMarksTextAsImage(t *testing.T) {
	for _, model := range []string{"gpt-5.6", "gpt-5.6-sol", "claude-sonnet-4-5", "gemini-3-pro", "deepseek-v4.1-flash"} {
		desc, ok := DescribeModel(model)
		require.True(t, ok, "model %q should be recognized", model)
		require.Equal(t, ModelCapabilityText, desc.Capability, "model %q", model)
	}
}

func TestDescribeModelLeavesUnknownAliasesUnknown(t *testing.T) {
	for _, model := range []string{"", "   ", "my-private-alias", "team/custom-v1"} {
		desc, ok := DescribeModel(model)
		require.False(t, ok, "model %q must stay unknown", model)
		require.Empty(t, desc.Capability)
		require.Nil(t, desc.Protocols)
	}
}
