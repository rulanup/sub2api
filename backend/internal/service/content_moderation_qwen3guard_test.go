package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQwen3GuardParseOutput(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		severity   qwen3GuardSeverity
		categories []string
		score      float64
	}{
		{
			name:     "safe",
			output:   "Safety: Safe\nCategories: None",
			severity: qwen3GuardSeveritySafe,
			score:    0,
		},
		{
			name:       "unsafe with multiple categories",
			output:     "Safety: Unsafe\nCategories: Non-violent Illegal Acts, Jailbreak, PII",
			severity:   qwen3GuardSeverityUnsafe,
			categories: []string{"illicit", "jailbreak", "pii"},
			score:      1,
		},
		{
			name:       "controversial in code fence",
			output:     "```text\nSafety: Controversial\nCategories: Politically Sensitive Topics, None\n```",
			severity:   qwen3GuardSeverityControversial,
			categories: []string{"political"},
			score:      0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parseQwen3GuardOutput(tt.output)
			require.NoError(t, err)
			require.Equal(t, tt.severity, parsed.Severity)
			require.Equal(t, tt.categories, parsed.Categories)
			require.Equal(t, tt.score, parsed.Score)
		})
	}
}

func TestQwen3GuardParseOutputRejectsUnknownOrMissingSafety(t *testing.T) {
	for _, output := range []string{
		"Safety: Maybe\nCategories: None",
		"Categories: Jailbreak",
		"",
	} {
		_, err := parseQwen3GuardOutput(output)
		require.Error(t, err, "output must not silently default to safe: %q", output)
	}
}

func TestQwen3GuardCategoryMapping(t *testing.T) {
	tests := map[string]string{
		"Violent":                       "violence",
		"Non-violent Illegal Acts":      "illicit",
		"Sexual Content or Sexual Acts": "sexual",
		"PII":                           "pii",
		"Suicide & Self-Harm":           "self-harm",
		"Unethical Acts":                "unethical",
		"Jailbreak":                     "jailbreak",
		"Copyright Violation":           "copyright",
		"Politically Sensitive Topics":  "political",
	}

	for source, expected := range tests {
		require.Equal(t, expected, normalizeQwen3GuardCategory(source))
	}
}
