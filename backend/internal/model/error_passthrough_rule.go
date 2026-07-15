// Package model 定义服务层使用的数据模型。
package model

import (
	"strings"
	"time"
	"unicode/utf8"
)

// ErrorPassthroughRule 全局错误透传规则
// 用于控制上游错误如何返回给客户端
type ErrorPassthroughRule struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`             // 规则名称
	Enabled         bool      `json:"enabled"`          // 是否启用
	Priority        int       `json:"priority"`         // 优先级（数字越小优先级越高）
	ErrorCodes      []int     `json:"error_codes"`      // 匹配的错误码列表（OR关系）
	Keywords        []string  `json:"keywords"`         // 匹配的关键词列表（OR关系）
	MatchMode       string    `json:"match_mode"`       // "any"(任一条件) 或 "all"(所有条件)
	Platforms       []string  `json:"platforms"`        // 适用平台列表
	PassthroughCode bool      `json:"passthrough_code"` // 是否透传原始状态码
	ResponseCode    *int      `json:"response_code"`    // 自定义状态码（passthrough_code=false 时使用）
	PassthroughBody bool      `json:"passthrough_body"` // 是否透传原始错误信息
	CustomMessage   *string   `json:"custom_message"`   // 自定义错误信息（passthrough_body=false 时使用）
	SkipMonitoring  bool      `json:"skip_monitoring"`  // 是否跳过运维监控记录
	Description     *string   `json:"description"`      // 规则描述
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// MatchModeAny 表示任一条件匹配即可
const MatchModeAny = "any"

// MatchModeAll 表示所有条件都必须匹配
const MatchModeAll = "all"

const (
	MaxErrorPassthroughCustomMessageLength = 2000
	MaxErrorPassthroughKeywords            = 50
	MaxErrorPassthroughKeywordLength       = 256
)

// 支持的平台常量
const (
	PlatformAnthropic   = "anthropic"
	PlatformOpenAI      = "openai"
	PlatformGemini      = "gemini"
	PlatformAntigravity = "antigravity"
	PlatformGrok        = "grok"
)

// AllPlatforms 返回所有支持的平台列表
func AllPlatforms() []string {
	return []string{PlatformAnthropic, PlatformOpenAI, PlatformGemini, PlatformAntigravity, PlatformGrok}
}

// Validate 验证规则配置的有效性
func (r *ErrorPassthroughRule) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return &ValidationError{Field: "name", Message: "name is required"}
	}
	if r.MatchMode != MatchModeAny && r.MatchMode != MatchModeAll {
		return &ValidationError{Field: "match_mode", Message: "match_mode must be 'any' or 'all'"}
	}
	// 至少需要配置一个匹配条件（错误码或关键词）
	if len(r.ErrorCodes) == 0 && len(r.Keywords) == 0 {
		return &ValidationError{Field: "conditions", Message: "at least one error_code or keyword is required"}
	}
	seenCodes := make(map[int]struct{}, len(r.ErrorCodes))
	codes := r.ErrorCodes[:0]
	for _, code := range r.ErrorCodes {
		if code < 100 || code > 599 {
			return &ValidationError{Field: "error_codes", Message: "error codes must be between 100 and 599"}
		}
		if _, exists := seenCodes[code]; !exists {
			seenCodes[code] = struct{}{}
			codes = append(codes, code)
		}
	}
	r.ErrorCodes = codes
	if len(r.Keywords) > MaxErrorPassthroughKeywords {
		return &ValidationError{Field: "keywords", Message: "too many keywords"}
	}
	seenKeywords := make(map[string]struct{}, len(r.Keywords))
	keywords := r.Keywords[:0]
	for _, keyword := range r.Keywords {
		keyword = strings.TrimSpace(keyword)
		if keyword == "" {
			return &ValidationError{Field: "keywords", Message: "keywords must not be empty"}
		}
		if utf8.RuneCountInString(keyword) > MaxErrorPassthroughKeywordLength {
			return &ValidationError{Field: "keywords", Message: "keyword is too long"}
		}
		if _, exists := seenKeywords[keyword]; !exists {
			seenKeywords[keyword] = struct{}{}
			keywords = append(keywords, keyword)
		}
	}
	r.Keywords = keywords
	validPlatforms := make(map[string]struct{}, len(AllPlatforms()))
	for _, platform := range AllPlatforms() {
		validPlatforms[platform] = struct{}{}
	}
	seenPlatforms := make(map[string]struct{}, len(r.Platforms))
	platforms := r.Platforms[:0]
	for _, platform := range r.Platforms {
		platform = strings.TrimSpace(platform)
		if _, valid := validPlatforms[platform]; !valid {
			return &ValidationError{Field: "platforms", Message: "unsupported platform: " + platform}
		}
		if _, exists := seenPlatforms[platform]; !exists {
			seenPlatforms[platform] = struct{}{}
			platforms = append(platforms, platform)
		}
	}
	r.Platforms = platforms
	if !r.PassthroughCode && r.ResponseCode == nil {
		return &ValidationError{Field: "response_code", Message: "response_code is required when passthrough_code is false"}
	}
	if r.ResponseCode != nil && (*r.ResponseCode < 100 || *r.ResponseCode > 599) {
		return &ValidationError{Field: "response_code", Message: "response_code must be between 100 and 599"}
	}
	if r.CustomMessage != nil {
		message := strings.TrimSpace(*r.CustomMessage)
		if utf8.RuneCountInString(message) > MaxErrorPassthroughCustomMessageLength {
			return &ValidationError{Field: "custom_message", Message: "custom_message is too long"}
		}
		r.CustomMessage = &message
	}
	if !r.PassthroughBody && (r.CustomMessage == nil || *r.CustomMessage == "") {
		return &ValidationError{Field: "custom_message", Message: "custom_message is required when passthrough_body is false"}
	}
	return nil
}

// ValidationError 表示验证错误
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
