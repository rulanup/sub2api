package model

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validErrorPassthroughRule() *ErrorPassthroughRule {
	message := " replacement "
	return &ErrorPassthroughRule{
		Name: " rule ", Enabled: true, ErrorCodes: []int{429}, Keywords: []string{" quota "},
		MatchMode: MatchModeAny, PassthroughCode: true, PassthroughBody: false, CustomMessage: &message,
	}
}

func TestErrorPassthroughRuleValidateNormalizesWriteInput(t *testing.T) {
	rule := validErrorPassthroughRule()
	rule.ErrorCodes = []int{429, 429}
	rule.Keywords = []string{" quota ", "quota"}
	rule.Platforms = []string{" openai ", "openai", "gemini"}
	require.NoError(t, rule.Validate())
	assert.Equal(t, "rule", rule.Name)
	assert.Equal(t, []string{"quota"}, rule.Keywords)
	assert.Equal(t, []int{429}, rule.ErrorCodes)
	assert.Equal(t, []string{"openai", "gemini"}, rule.Platforms)
	assert.Equal(t, "replacement", *rule.CustomMessage)
}

func TestErrorPassthroughRuleValidateBounds(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ErrorPassthroughRule)
	}{
		{"low error code", func(r *ErrorPassthroughRule) { r.ErrorCodes = []int{99} }},
		{"high error code", func(r *ErrorPassthroughRule) { r.ErrorCodes = []int{600} }},
		{"invalid response code", func(r *ErrorPassthroughRule) { code := 99; r.PassthroughCode = false; r.ResponseCode = &code }},
		{"empty keyword", func(r *ErrorPassthroughRule) { r.Keywords = []string{" "} }},
		{"unsupported platform", func(r *ErrorPassthroughRule) { r.Platforms = []string{"openai", "unknown"} }},
		{"long keyword", func(r *ErrorPassthroughRule) {
			r.Keywords = []string{strings.Repeat("k", MaxErrorPassthroughKeywordLength+1)}
		}},
		{"too many keywords", func(r *ErrorPassthroughRule) {
			r.Keywords = make([]string, MaxErrorPassthroughKeywords+1)
			for i := range r.Keywords {
				r.Keywords[i] = "k"
			}
		}},
		{"empty custom message", func(r *ErrorPassthroughRule) { value := "  "; r.CustomMessage = &value }},
		{"long custom message", func(r *ErrorPassthroughRule) {
			value := strings.Repeat("界", MaxErrorPassthroughCustomMessageLength+1)
			r.CustomMessage = &value
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := validErrorPassthroughRule()
			tt.mutate(rule)
			err := rule.Validate()
			require.Error(t, err)
			assert.True(t, utf8.ValidString(err.Error()))
		})
	}
}
