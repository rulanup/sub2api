package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type qwen3GuardSeverity string

const (
	qwen3GuardSeveritySafe                  qwen3GuardSeverity = "safe"
	qwen3GuardSeverityUnsafe                qwen3GuardSeverity = "unsafe"
	qwen3GuardSeverityControversial         qwen3GuardSeverity = "controversial"
	contentModerationProtocolQwen3GuardChat                    = ContentModerationProtocolQwen3GuardChat
)

type qwen3GuardOutput struct {
	Severity   qwen3GuardSeverity
	Categories []string
	Score      float64
}

var (
	qwen3GuardSafetyPattern     = regexp.MustCompile(`(?im)\bSafety\s*:\s*(Safe|Unsafe|Controversial)\b`)
	qwen3GuardCategoriesPattern = regexp.MustCompile(`(?im)^\s*Categories\s*:\s*(.*)$`)
)

var qwen3GuardCategoryMap = map[string]string{
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

func parseQwen3GuardOutput(output string) (*qwen3GuardOutput, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return nil, fmt.Errorf("qwen3guard response is empty")
	}

	match := qwen3GuardSafetyPattern.FindStringSubmatch(output)
	if len(match) != 2 {
		return nil, fmt.Errorf("qwen3guard response missing a valid Safety label")
	}

	severity := qwen3GuardSeverity(strings.ToLower(match[1]))
	parsed := &qwen3GuardOutput{Severity: severity}
	switch severity {
	case qwen3GuardSeveritySafe:
		parsed.Score = 0
	case qwen3GuardSeverityUnsafe:
		parsed.Score = 1
	case qwen3GuardSeverityControversial:
		parsed.Score = 0.5
	default:
		return nil, fmt.Errorf("qwen3guard response has unsupported Safety label %q", match[1])
	}

	categoryMatch := qwen3GuardCategoriesPattern.FindStringSubmatch(output)
	if len(categoryMatch) == 2 {
		for _, rawCategory := range strings.Split(categoryMatch[1], ",") {
			rawCategory = strings.TrimSpace(strings.Trim(rawCategory, "`"))
			if rawCategory == "" || strings.EqualFold(rawCategory, "None") {
				continue
			}
			parsed.Categories = append(parsed.Categories, normalizeQwen3GuardCategory(rawCategory))
		}
	}

	return parsed, nil
}

func normalizeQwen3GuardCategory(category string) string {
	category = strings.TrimSpace(category)
	if mapped, ok := qwen3GuardCategoryMap[category]; ok {
		return mapped
	}
	return strings.ToLower(strings.NewReplacer(" ", "-", "_", "-").Replace(category))
}

type qwen3GuardChatRequest struct {
	Model       string                  `json:"model"`
	Messages    []qwen3GuardChatMessage `json:"messages"`
	Stream      bool                    `json:"stream"`
	Temperature float64                 `json:"temperature"`
	MaxTokens   int                     `json:"max_tokens"`
}

type qwen3GuardChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type qwen3GuardChatResponse struct {
	Choices []struct {
		Message qwen3GuardChatMessage `json:"message"`
	} `json:"choices"`
}

func (s *ContentModerationService) callQwen3GuardChatOnce(ctx context.Context, cfg *ContentModerationConfig, apiKey string, input any, httpStatus *int) (*moderationAPIResult, error) {
	text, err := qwen3GuardTextInput(input)
	if err != nil {
		return nil, err
	}
	endpoint, err := qwen3GuardEndpoint(cfg.BaseURL)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(qwen3GuardChatRequest{
		Model: cfg.Model,
		Messages: []qwen3GuardChatMessage{{
			Role:    "user",
			Content: text,
		}},
		Stream:      false,
		Temperature: 0,
		MaxTokens:   128,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal qwen3guard request: %w", err)
	}

	timeout := time.Duration(cfg.TimeoutMS) * time.Millisecond
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	client, err := s.moderationHTTPClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if httpStatus != nil {
		*httpStatus = resp.StatusCode
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("qwen3guard api status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var response qwen3GuardChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode qwen3guard response: %w", err)
	}
	if len(response.Choices) == 0 || strings.TrimSpace(response.Choices[0].Message.Content) == "" {
		return nil, errors.New("qwen3guard response has no choices[0].message.content")
	}
	parsed, err := parseQwen3GuardOutput(response.Choices[0].Message.Content)
	if err != nil {
		return nil, err
	}
	scores := make(map[string]float64, len(parsed.Categories))
	for _, category := range parsed.Categories {
		scores[category] = parsed.Score
	}
	if len(scores) == 0 && parsed.Score > 0 {
		scores["qwen3guard"] = parsed.Score
	}
	flagged := parsed.Severity == qwen3GuardSeverityUnsafe || (parsed.Severity == qwen3GuardSeverityControversial && cfg.ControversialAction == ContentModerationControversialActionBlock)
	return &moderationAPIResult{
		Flagged:        flagged,
		CategoryScores: scores,
		Severity:       parsed.Severity,
		Categories:     append([]string(nil), parsed.Categories...),
	}, nil
}

func qwen3GuardEndpoint(baseURL string) (string, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(baseURL, "/v1/chat/completions") {
		return baseURL, nil
	}
	if strings.HasSuffix(baseURL, "/v1") {
		return url.JoinPath(baseURL, "/chat/completions")
	}
	return url.JoinPath(baseURL, "/v1/chat/completions")
}

func qwen3GuardTextInput(input any) (string, error) {
	switch value := input.(type) {
	case string:
		if strings.TrimSpace(value) == "" {
			return "", errors.New("qwen3guard input is empty")
		}
		return value, nil
	case []moderationAPIInputPart:
		var parts []string
		for _, part := range value {
			switch part.Type {
			case "text":
				if strings.TrimSpace(part.Text) != "" {
					parts = append(parts, part.Text)
				}
			case "image_url":
				return "", errors.New("qwen3guard chat moderation does not support image input")
			}
		}
		if len(parts) == 0 {
			return "", errors.New("qwen3guard input has no text content")
		}
		return strings.Join(parts, "\n"), nil
	case ContentModerationInput:
		if len(value.Images) > 0 {
			return "", errors.New("qwen3guard chat moderation does not support image input")
		}
		return qwen3GuardTextInput(value.Text)
	default:
		return "", fmt.Errorf("qwen3guard input type %T is not supported", input)
	}
}
