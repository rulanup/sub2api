package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeGroupSystemPrompts_ValidatesProtocolAndLength(t *testing.T) {
	out, err := NormalizeGroupSystemPrompts(map[string]string{
		"anthropic_messages": " a ",
		"openai_responses":   "b",
		"gemini":             "c",
	})
	require.NoError(t, err)
	require.Equal(t, "a", out["anthropic_messages"])
	require.Equal(t, "b", out["openai"])
	require.Equal(t, "c", out["gemini"])

	_, err = NormalizeGroupSystemPrompts(map[string]string{"unknown": "x"})
	require.Error(t, err)

	_, err = NormalizeGroupSystemPrompts(map[string]string{"gemini": strings.Repeat("x", 8001)})
	require.Error(t, err)
}

func TestGroupSystemPromptFor_TriState(t *testing.T) {
	enabled := true
	disabled := false
	group := &Group{
		ID: 1, Name: "g", SystemPromptEnabled: &enabled, SystemPromptMode: "if_absent",
		SystemPrompts: GroupSystemPrompts{"openai": " hi "},
	}
	text, mode, decided := group.SystemPromptFor("openai_chat_completions")
	require.True(t, decided)
	require.Equal(t, "hi", text)
	require.Equal(t, "if_absent", mode)

	group.SystemPromptEnabled = &disabled
	text, _, decided = group.SystemPromptFor("openai")
	require.True(t, decided)
	require.Empty(t, text)

	group.SystemPromptEnabled = nil
	_, _, decided = group.SystemPromptFor("openai")
	require.False(t, decided)
}

func TestGroupAuditEnabledOverride(t *testing.T) {
	var nilGroup *Group
	_, decided := nilGroup.AuditEnabledOverride()
	require.False(t, decided)

	off := false
	group := &Group{SecurityAuditEnabled: &off}
	enabled, decided := group.AuditEnabledOverride()
	require.True(t, decided)
	require.False(t, enabled)

	on := true
	group.SecurityAuditEnabled = &on
	enabled, decided = group.AuditEnabledOverride()
	require.True(t, decided)
	require.True(t, enabled)
}
