package securityaudit

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/pkg/securitytext"
)

type localPolicyEvidence struct {
	strongIntent, actionMatches, specificTargets, genericTargets, ambiguous int
	protective                                                              bool
}

const (
	localPolicyStrongWeight         = 3
	localPolicyActionWeight         = 2
	localPolicySpecificTargetWeight = 2
	localPolicyGenericTargetWeight  = 1
	localPolicyAmbiguousWeight      = 1
	localPolicyBlockScore           = 5
	localPolicyMaxSegmentRunes      = 512
	localPolicyWindowRunes          = 128
)

func splitLocalPolicySegments(text string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(text, func(r rune) bool {
		switch r {
		case '\n', '\r', '.', '。', '？', '?', '！', '!', '；', ';', '：', ':', '|', '｜':
			return true
		}
		return false
	}) {
		r := []rune(strings.TrimSpace(part))
		if len(r) == 0 {
			continue
		}
		if len(r) <= localPolicyMaxSegmentRunes {
			out = append(out, string(r))
			continue
		}
		stride := localPolicyMaxSegmentRunes - localPolicyWindowRunes
		if stride <= 0 {
			stride = localPolicyMaxSegmentRunes
		}
		for start := 0; start < len(r); start += stride {
			end := start + localPolicyMaxSegmentRunes
			if end > len(r) {
				end = len(r)
			}
			out = append(out, string(r[start:end]))
			if end == len(r) {
				break
			}
		}
	}
	return out
}

func containsLocalPolicyTerm(normalized, compact, term string) bool {
	t := securitytext.Canonicalize(term)
	tn, tc := strings.ToLower(t.Text), strings.ToLower(t.Compact)
	n := strings.ToLower(normalized)
	if tn != "" && strings.Contains(n, tn) {
		for i := 0; i+len(tn) <= len(n); i++ {
			if n[i:i+len(tn)] != tn {
				continue
			}
			if isASCIITerm(tn) && !asciiBoundaries(n, i, i+len(tn)) {
				continue
			}
			return true
		}
	}
	if tc == "" || compact == "" {
		return false
	}
	c := strings.ToLower(compact)
	if !strings.Contains(c, tc) {
		return false
	}
	// Compact matching is only valid when removed separators existed for ASCII words.
	if !isASCIITerm(tc) {
		return true
	}
	for ci := 0; ; {
		j := strings.Index(c[ci:], tc)
		if j < 0 {
			return false
		}
		j += ci
		if !strings.Contains(tn, " ") && !asciiBoundaries(c, j, j+len(tc)) {
			ci = j + 1
			continue
		}
		compactRuneStart := utf8.RuneCountInString(c[:j])
		if compactOccurrenceSeparated(normalized, tc, compactRuneStart) {
			return true
		}
		ci = j + 1
	}
}

func isASCIITerm(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII || !(unicode.IsLetter(r) || unicode.IsDigit(r)) {
			return false
		}
	}
	return s != ""
}
func asciiBoundaries(s string, start, end int) bool {
	prev, next := byte(0), byte(0)
	if start > 0 {
		prev = s[start-1]
	}
	if end < len(s) {
		next = s[end]
	}
	return !isASCIIWord(prev) && !isASCIIWord(next)
}
func isASCIIWord(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9'
}
func compactOccurrenceSeparated(normalized, term string, compactRuneStart int) bool {
	ci := 0
	inMatch := false
	sawSep := false
	matchEnd := compactRuneStart + utf8.RuneCountInString(term)
	for _, r := range normalized {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if ci >= compactRuneStart && ci < matchEnd {
				inMatch = true
			}
			ci++
			if ci >= matchEnd {
				inMatch = false
			}
			continue
		}
		if inMatch {
			sawSep = true
		}
	}
	return sawSep
}

func scoreLocalPolicyRule(segment string, rule localPolicyRule) localPolicyEvidence {
	c := securitytext.Canonicalize(segment)
	e := localPolicyEvidence{protective: containsAnyLocalPolicyTerm(c.Text, c.Compact, localPolicyContextTerms)}
	strongAction := make(map[string]struct{}, len(rule.strongActions))
	for _, t := range rule.strongActions {
		strongAction[securitytext.Canonicalize(t).Text] = struct{}{}
		if containsLocalPolicyTerm(c.Text, c.Compact, t) {
			e.strongIntent++
			e.actionMatches++
		}
	}
	for _, t := range rule.actions {
		if _, ok := strongAction[securitytext.Canonicalize(t).Text]; ok {
			continue
		}
		if containsLocalPolicyTerm(c.Text, c.Compact, t) {
			e.actionMatches++
		}
	}
	strongTarget := make(map[string]struct{}, len(rule.strongTargets))
	for _, t := range rule.strongTargets {
		key := securitytext.Canonicalize(t).Text
		strongTarget[key] = struct{}{}
		if containsLocalPolicyTerm(c.Text, c.Compact, t) {
			e.specificTargets++
		}
	}
	genericTarget := make(map[string]struct{}, len(rule.genericTargets))
	for _, t := range rule.genericTargets {
		key := securitytext.Canonicalize(t).Text
		genericTarget[key] = struct{}{}
		if containsLocalPolicyTerm(c.Text, c.Compact, t) {
			e.genericTargets++
		}
	}
	for _, t := range rule.targets {
		key := securitytext.Canonicalize(t).Text
		if _, ok := strongTarget[key]; ok {
			continue
		}
		if _, ok := genericTarget[key]; ok {
			continue
		}
		if containsLocalPolicyTerm(c.Text, c.Compact, t) {
			e.specificTargets++
		}
	}
	for _, t := range rule.ambiguous {
		if containsLocalPolicyTerm(c.Text, c.Compact, t) {
			e.ambiguous++
		}
	}
	return e
}
