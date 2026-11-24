package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeGroupSystemPrompt_TrimsAndLimitsLength(t *testing.T) {
	out, err := NormalizeGroupSystemPrompt("  hello  ")
	require.NoError(t, err)
	require.Equal(t, "hello", out)

	_, err = NormalizeGroupSystemPrompt(strings.Repeat("x", 8001))
	require.Error(t, err)
}

func TestGroupSystemPromptFor_TriState(t *testing.T) {
	enabled := true
	disabled := false
	group := &Group{
		ID: 1, Name: "g", SystemPromptEnabled: &enabled, SystemPromptMode: "if_absent",
		SystemPrompt: " hi ",
	}
	text, mode, decided := group.SystemPromptFor()
	require.True(t, decided)
	require.Equal(t, "hi", text)
	require.Equal(t, "if_absent", mode)

	group.SystemPromptEnabled = &disabled
	text, _, decided = group.SystemPromptFor()
	require.True(t, decided)
	require.Empty(t, text)

	group.SystemPromptEnabled = nil
	_, _, decided = group.SystemPromptFor()
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
