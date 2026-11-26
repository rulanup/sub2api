package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDescribeModel(t *testing.T) {
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
		{"text model", "gpt-5.6", "", nil, false},
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

// TestDescribeModelLeavesNonImageModelsUnknown 覆盖审查重点 #2：
// 文本模型与私有别名不得被判成图片模型，否则客户端会挂错调用脚本。
func TestDescribeModelLeavesNonImageModelsUnknown(t *testing.T) {
	for _, model := range []string{"", "   ", "gpt-5.6", "gpt-5.6-sol", "claude-sonnet-4-5", "my-private-alias", "gemini-3-pro"} {
		desc, ok := DescribeModel(model)
		require.False(t, ok, "model %q must stay unknown", model)
		require.Empty(t, desc.Capability)
		require.Nil(t, desc.Protocols)
	}
}
