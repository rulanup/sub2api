// Package securitytext contains bounded, scan-only text canonicalization used
// by the local security policy and external moderation adapters.
package securitytext

import (
	"encoding/base64"
	"html"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

const (
	canonicalizeMaxInputBytes = 64 * 1024
	canonicalizeMaxRounds     = 4
)

// Result contains the readable normalized text and a compact form used for
// matching terms separated by punctuation or whitespace. It is deliberately a
// scan copy; callers must not use it to rewrite the gateway request body.
type Result struct {
	Text    string
	Compact string
}

// Canonicalize decodes a bounded number of mixed encoding layers and applies
// Unicode normalization. Invalid escapes are preserved verbatim.
func Canonicalize(input string) Result {
	current := truncateUTF8(input, canonicalizeMaxInputBytes)
	for round := 0; round < canonicalizeMaxRounds; round++ {
		next := html.UnescapeString(current)
		next = decodePercentEscapes(next)
		next = decodeLiteralUnicodeEscapes(next)
		next = decodeBase64Segments(next)
		next = truncateUTF8(next, canonicalizeMaxInputBytes)
		if next == current {
			break
		}
		current = next
	}

	normalized := normalizeUnicode(current)
	return Result{Text: normalized, Compact: compact(normalized)}
}

func normalizeUnicode(input string) string {
	input = norm.NFKC.String(input)
	var builder strings.Builder
	builder.Grow(len(input))
	for _, r := range input {
		switch {
		case unicode.In(r, unicode.Cf):
			// Includes zero-width spaces, BOM and other format controls.
		case unicode.IsSpace(r) || unicode.IsControl(r):
			builder.WriteByte(' ')
		default:
			builder.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(builder.String()), " ")
}

func compact(input string) string {
	var builder strings.Builder
	for _, r := range input {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(unicode.ToLower(r))
		}
	}
	return builder.String()
}

const base64MinSegmentLength = 24

func decodeBase64Segments(input string) string {
	var builder strings.Builder
	builder.Grow(len(input))
	changed := false
	for index := 0; index < len(input); {
		if !isBase64SegmentByte(input[index]) {
			builder.WriteByte(input[index])
			index++
			continue
		}
		start := index
		for index < len(input) && isBase64SegmentByte(input[index]) {
			index++
		}
		segment := input[start:index]
		decoded, ok := decodeBase64Candidate(segment)
		if ok {
			builder.WriteString(decoded)
			changed = true
		} else {
			builder.WriteString(segment)
		}
	}
	if !changed {
		return input
	}
	return builder.String()
}

func isBase64SegmentByte(value byte) bool {
	return value >= 'A' && value <= 'Z' ||
		value >= 'a' && value <= 'z' ||
		value >= '0' && value <= '9' ||
		value == '+' || value == '/' || value == '-' || value == '_' || value == '='
}

func decodeBase64Candidate(candidate string) (string, bool) {
	if len(candidate) < base64MinSegmentLength {
		return "", false
	}
	decoders := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	for _, decoder := range decoders {
		decoded, err := decoder.DecodeString(candidate)
		if err != nil || !isReadableBase64Text(decoded) {
			continue
		}
		return string(decoded), true
	}
	return "", false
}

func isReadableBase64Text(input []byte) bool {
	if len(input) == 0 || !utf8.Valid(input) {
		return false
	}
	runeCount := 0
	printableCount := 0
	hasWord := false
	for _, r := range string(input) {
		runeCount++
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			hasWord = true
		}
		if unicode.IsControl(r) && !unicode.IsSpace(r) {
			return false
		}
		if !unicode.IsPrint(r) && !unicode.IsSpace(r) {
			return false
		}
		printableCount++
	}
	return runeCount >= 3 && hasWord && printableCount*100 >= runeCount*85
}

func decodePercentEscapes(input string) string {
	if !strings.Contains(input, "%") {
		return input
	}
	var builder strings.Builder
	builder.Grow(len(input))
	changed := false
	for index := 0; index < len(input); {
		if input[index] == '%' && index+2 < len(input) {
			if value, ok := parseHexByte(input[index+1], input[index+2]); ok {
				builder.WriteByte(value)
				index += 3
				changed = true
				continue
			}
		}
		builder.WriteByte(input[index])
		index++
	}
	if !changed {
		return input
	}
	return builder.String()
}

func decodeLiteralUnicodeEscapes(input string) string {
	if !strings.Contains(input, `\u`) && !strings.Contains(input, `\U`) {
		return input
	}
	var builder strings.Builder
	builder.Grow(len(input))
	changed := false
	for index := 0; index < len(input); {
		if input[index] == '\\' && index+1 < len(input) {
			digits := 0
			switch input[index+1] {
			case 'u':
				digits = 4
			case 'U':
				digits = 8
			}
			if digits > 0 && index+2+digits <= len(input) {
				if value, ok := parseHexRune(input[index+2 : index+2+digits]); ok {
					builder.WriteRune(value)
					index += 2 + digits
					changed = true
					continue
				}
			}
		}
		builder.WriteByte(input[index])
		index++
	}
	if !changed {
		return input
	}
	return builder.String()
}

func parseHexByte(left, right byte) (byte, bool) {
	leftValue, leftOK := hexValue(left)
	rightValue, rightOK := hexValue(right)
	if !leftOK || !rightOK {
		return 0, false
	}
	return leftValue<<4 | rightValue, true
}

func parseHexRune(value string) (rune, bool) {
	parsed, err := strconv.ParseUint(value, 16, 32)
	if err != nil || parsed > utf8.MaxRune {
		return 0, false
	}
	r := rune(parsed)
	if !utf8.ValidRune(r) {
		return 0, false
	}
	return r, true
}

func hexValue(value byte) (byte, bool) {
	switch {
	case value >= '0' && value <= '9':
		return value - '0', true
	case value >= 'a' && value <= 'f':
		return value - 'a' + 10, true
	case value >= 'A' && value <= 'F':
		return value - 'A' + 10, true
	default:
		return 0, false
	}
}

func truncateUTF8(input string, limit int) string {
	if limit <= 0 || len(input) <= limit {
		return input
	}
	truncated := input[:limit]
	for len(truncated) > 0 && !utf8.ValidString(truncated) {
		truncated = truncated[:len(truncated)-1]
	}
	return truncated
}
