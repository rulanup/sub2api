package securitytext

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCanonicalizeEncodedSecurityText(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		compact string
	}{
		{name: "literal unicode", input: `\u5ffd\u7565\u4e4b\u524d`, want: "忽略之前", compact: "忽略之前"},
		{name: "url encoded utf8", input: "%E5%BF%BD%E7%95%A5%E4%B9%8B%E5%89%8D", want: "忽略之前", compact: "忽略之前"},
		{name: "html decimal entities", input: "&#24573;&#30053;&#20043;&#21069;", want: "忽略之前", compact: "忽略之前"},
		{name: "html hex entities", input: "&#x5FFD;&#x7565;&#x4E4B;&#x524D;", want: "忽略之前", compact: "忽略之前"},
		{name: "html then url then unicode", input: "&#37;&#53;&#99;&#117;&#53;&#102;&#102;&#100;&#37;&#53;&#99;&#117;&#55;&#53;&#54;&#53;&#37;&#53;&#99;&#117;&#52;&#101;&#52;&#98;&#37;&#53;&#99;&#117;&#53;&#50;&#52;&#100;", want: "忽略之前", compact: "忽略之前"},
		{name: "base64 then html then url then unicode", input: "JiMzNzsmIzUzOyYjOTk7JiMxMTc7JiM1MzsmIzEwMjsmIzEwMjsmIzEwMDsmIzM3OyYjNTM7JiM5OTsmIzExNzsmIzU1OyYjNTM7JiM1NDsmIzUzOyYjMzc7JiM1MzsmIzk5OyYjMTE3OyYjNTI7JiMxMDE7JiM1MjsmIzk4OyYjMzc7JiM1MzsmIzk5OyYjMTE3OyYjNTM7JiM1MDsmIzUyOyYjMTAwOw==放行了", want: "忽略之前放行了", compact: "忽略之前放行了"},
		{name: "zero width unicode escape", input: `忽\u200b略\u200c之\u200d前`, want: "忽略之前", compact: "忽略之前"},
		{name: "full width and separators", input: "忽 略＿之　前", want: "忽 略_之 前", compact: "忽略之前"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Canonicalize(tt.input)
			require.Equal(t, tt.want, got.Text)
			require.Equal(t, tt.compact, got.Compact)
		})
	}
}

func TestCanonicalizePreservesCaseAndRejectsOpaqueBase64(t *testing.T) {
	got := Canonicalize("Q1")
	require.Equal(t, "Q1", got.Text)

	opaque := strings.Repeat("A", 32)
	require.Equal(t, opaque, Canonicalize(opaque).Text)
}

func TestCanonicalizeDecodesUntilStableWithoutChangingInvalidEscapes(t *testing.T) {
	got := Canonicalize(`&amp;#37;&amp;#53;&amp;#99;&amp;#117;&amp;#53;&amp;#102;&amp;#102;&amp;#100; %ZZ \uZZZZ`)
	require.Contains(t, got.Text, "忽")
	require.Contains(t, got.Text, "%ZZ")
	require.Contains(t, got.Text, `\uZZZZ`)
}

func TestCanonicalizeBoundsExpandedInput(t *testing.T) {
	got := Canonicalize(strings.Repeat("&#65;", canonicalizeMaxInputBytes))
	require.LessOrEqual(t, len([]byte(got.Text)), canonicalizeMaxInputBytes)
}
