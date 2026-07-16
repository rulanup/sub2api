package service

import (
	"context"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const errorPassthroughServiceContextKey = "error_passthrough_service"

// MatchRuleForRequest safely uses the authenticated request context when available.
func (s *ErrorPassthroughService) MatchRuleForRequest(c *gin.Context, platform string, statusCode int, body []byte) *model.ErrorPassthroughRule {
	ctx := context.Background()
	if c != nil && c.Request != nil {
		ctx = c.Request.Context()
	}
	return s.MatchRuleWithContext(ctx, platform, statusCode, body)
}

// BindErrorPassthroughService 将错误透传服务绑定到请求上下文，供 service 层在非 failover 场景下复用规则。
func BindErrorPassthroughService(c *gin.Context, svc *ErrorPassthroughService) {
	if c == nil || svc == nil {
		return
	}
	c.Set(errorPassthroughServiceContextKey, svc)
}

func getBoundErrorPassthroughService(c *gin.Context) *ErrorPassthroughService {
	if c == nil {
		return nil
	}
	v, ok := c.Get(errorPassthroughServiceContextKey)
	if !ok {
		return nil
	}
	svc, ok := v.(*ErrorPassthroughService)
	if !ok {
		return nil
	}
	return svc
}

// applyErrorPassthroughRule 按规则改写错误响应；未命中时返回默认响应参数。
func applyErrorPassthroughRule(
	c *gin.Context,
	platform string,
	upstreamStatus int,
	responseBody []byte,
	defaultStatus int,
	defaultErrType string,
	defaultErrMsg string,
) (status int, errType string, errMsg string, matched bool) {
	status = defaultStatus
	errType = defaultErrType
	errMsg = defaultErrMsg

	rule := resolveErrorPassthroughRule(c, platform, upstreamStatus, responseBody)
	if rule == nil {
		return status, errType, errMsg, false
	}

	status = upstreamStatus
	if !rule.PassthroughCode && rule.ResponseCode != nil {
		status = *rule.ResponseCode
	}

	errMsg = ExtractUpstreamErrorMessage(responseBody)
	if !rule.PassthroughBody && rule.CustomMessage != nil {
		errMsg = SanitizeErrorPassthroughCustomMessage(*rule.CustomMessage)
	}

	// 命中 skip_monitoring 时在 context 中标记，供 ops_error_logger 跳过记录。
	if rule.SkipMonitoring {
		c.Set(OpsSkipPassthroughKey, true)
	}

	// 与现有 failover 场景保持一致：命中规则时统一返回 upstream_error。
	errType = "upstream_error"
	return status, errType, errMsg, true
}

func resolveErrorPassthroughRule(c *gin.Context, platform string, upstreamStatus int, responseBody []byte) *model.ErrorPassthroughRule {
	svc := getBoundErrorPassthroughService(c)
	if svc == nil {
		return nil
	}
	rule := svc.MatchRuleForRequest(c, platform, upstreamStatus, responseBody)
	if rule != nil && rule.SkipMonitoring {
		c.Set(OpsSkipPassthroughKey, true)
	}
	return rule
}

func SanitizeErrorPassthroughCustomMessage(message string) string {
	message = strings.TrimSpace(message)
	if utf8.RuneCountInString(message) <= model.MaxErrorPassthroughCustomMessageLength {
		return message
	}
	runes := []rune(message)
	return string(runes[:model.MaxErrorPassthroughCustomMessageLength])
}

// applyErrorPassthroughRuleToJSON replaces only the protocol error message.
// Matching always uses originalBody so customized text never affects classification or evidence.
func applyErrorPassthroughRuleToJSON(c *gin.Context, platform string, upstreamStatus int, originalBody []byte) (int, []byte, bool) {
	return applyErrorPassthroughRuleToJSONPath(c, platform, upstreamStatus, originalBody, "")
}

// fallbackPath is only supplied by a serializer that owns the response envelope.
func applyErrorPassthroughRuleToJSONPath(c *gin.Context, platform string, upstreamStatus int, originalBody []byte, fallbackPath string) (int, []byte, bool) {
	rule := resolveErrorPassthroughRule(c, platform, upstreamStatus, originalBody)
	if rule == nil {
		return upstreamStatus, originalBody, false
	}
	status := upstreamStatus
	if !rule.PassthroughCode && rule.ResponseCode != nil {
		status = *rule.ResponseCode
	}
	if rule.PassthroughBody || rule.CustomMessage == nil {
		return status, originalBody, true
	}
	message := SanitizeErrorPassthroughCustomMessage(*rule.CustomMessage)
	if !gjson.ValidBytes(originalBody) {
		if fallbackPath == "" {
			return status, originalBody, true
		}
		body, err := sjson.SetBytes([]byte(`{}`), fallbackPath, message)
		if err == nil {
			return status, body, true
		}
		return status, originalBody, true
	}
	for _, path := range []string{"response.error.message", "error.message", "message", "detail"} {
		if !gjson.GetBytes(originalBody, path).Exists() {
			continue
		}
		updated, err := sjson.SetBytes(originalBody, path, message)
		if err == nil {
			return status, updated, true
		}
	}
	if errorValue := gjson.GetBytes(originalBody, "error"); errorValue.Exists() && errorValue.Type == gjson.String {
		if updated, err := sjson.SetBytes(originalBody, "error", message); err == nil {
			return status, updated, true
		}
	}
	if fallbackPath != "" {
		if updated, err := sjson.SetBytes(originalBody, fallbackPath, message); err == nil {
			return status, updated, true
		}
	}
	return status, originalBody, true
}

func applyErrorPassthroughRuleToGoogleJSON(c *gin.Context, platform string, upstreamStatus int, originalBody []byte) (int, []byte, bool) {
	status, body, matched := applyErrorPassthroughRuleToJSONPath(c, platform, upstreamStatus, originalBody, "error.message")
	if !matched || status == upstreamStatus || !gjson.ValidBytes(body) {
		return status, body, matched
	}
	updated, err := sjson.SetBytes(body, "error.code", status)
	if err == nil {
		body = updated
	}
	return status, body, matched
}

func applyErrorPassthroughRuleToOpenAIWSEvent(c *gin.Context, platform string, payload []byte) []byte {
	eventType := strings.TrimSpace(gjson.GetBytes(payload, "type").String())
	var status int
	switch eventType {
	case "response.failed":
		status = openAIStreamFailedEventSemanticStatus(payload, extractOpenAISSEErrorMessage(payload))
	case "error":
		code, errType, message := parseOpenAIWSErrorEventFields(payload)
		status = openAIWSErrorHTTPStatusFromRaw(code, errType)
		if status == http.StatusBadGateway {
			status = openAIStreamFailedEventSemanticStatus(payload, message)
		}
	default:
		return payload
	}
	_, updated, matched := applyErrorPassthroughRuleToJSON(c, platform, status, payload)
	if !matched {
		return payload
	}
	return updated
}

func TruncateErrorPassthroughWSReason(value string) string {
	const maxBytes = 120
	value = strings.TrimSpace(value)
	if len(value) <= maxBytes {
		return value
	}
	value = value[:maxBytes]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}
