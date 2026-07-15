package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestApplyErrorPassthroughRule_NoBoundService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	status, errType, errMsg, matched := applyErrorPassthroughRule(
		c,
		PlatformAnthropic,
		http.StatusUnprocessableEntity,
		[]byte(`{"error":{"message":"invalid schema"}}`),
		http.StatusBadGateway,
		"upstream_error",
		"Upstream request failed",
	)

	assert.False(t, matched)
	assert.Equal(t, http.StatusBadGateway, status)
	assert.Equal(t, "upstream_error", errType)
	assert.Equal(t, "Upstream request failed", errMsg)
}

func TestGatewayHandleErrorResponse_NoRuleKeepsDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	svc := &GatewayService{}
	respBody := []byte(`{"error":{"message":"Invalid schema for field messages"}}`)
	resp := &http.Response{
		StatusCode: http.StatusUnprocessableEntity,
		Body:       io.NopCloser(bytes.NewReader(respBody)),
		Header:     http.Header{},
	}
	account := &Account{ID: 11, Platform: PlatformAnthropic, Type: AccountTypeAPIKey}

	_, err := svc.handleErrorResponse(context.Background(), resp, c, account)
	require.Error(t, err)
	assert.Equal(t, http.StatusBadGateway, rec.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	errField, ok := payload["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "upstream_error", errField["type"])
	assert.Equal(t, "Upstream request failed", errField["message"])
}

func TestOpenAIHandleErrorResponse_NoRuleKeepsDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	svc := &OpenAIGatewayService{}
	respBody := []byte(`{"error":{"message":"Invalid schema for field messages"}}`)
	resp := &http.Response{
		StatusCode: http.StatusUnprocessableEntity,
		Body:       io.NopCloser(bytes.NewReader(respBody)),
		Header:     http.Header{},
	}
	account := &Account{ID: 12, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

	_, err := svc.handleErrorResponse(context.Background(), resp, c, account, nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusBadGateway, rec.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	errField, ok := payload["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "upstream_error", errField["type"])
	assert.Equal(t, "Upstream request failed", errField["message"])
}

func TestOpenAIHandleErrorResponse_ContextWindow502KeepsMessageWithoutFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	svc := &OpenAIGatewayService{}
	respBody := []byte(`{"error":{"message":"Your input exceeds the context window of this model. Please adjust your input and try again.","type":"upstream_error","code":null}}`)
	resp := &http.Response{
		StatusCode: http.StatusBadGateway,
		Body:       io.NopCloser(bytes.NewReader(respBody)),
		Header:     http.Header{},
	}
	account := &Account{ID: 14, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

	_, err := svc.handleErrorResponse(context.Background(), resp, c, account, nil)
	require.Error(t, err)
	var failoverErr *UpstreamFailoverError
	require.False(t, errors.As(err, &failoverErr))
	assert.Equal(t, http.StatusBadGateway, rec.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	errField, ok := payload["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "upstream_error", errField["type"])
	assert.Equal(t, "Your input exceeds the context window of this model. Please adjust your input and try again.", errField["message"])
}

func TestGeminiWriteGeminiMappedError_NoRuleKeepsDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	svc := &GeminiMessagesCompatService{}
	respBody := []byte(`{"error":{"code":422,"message":"Invalid schema for field messages","status":"INVALID_ARGUMENT"}}`)
	account := &Account{ID: 13, Platform: PlatformGemini, Type: AccountTypeAPIKey}

	err := svc.writeGeminiMappedError(c, account, http.StatusUnprocessableEntity, "req-2", respBody)
	require.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	errField, ok := payload["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "invalid_request_error", errField["type"])
	assert.Equal(t, "Upstream request failed", errField["message"])
}

func TestGatewayHandleErrorResponse_AppliesRuleFor422(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	ruleSvc := &ErrorPassthroughService{}
	ruleSvc.setLocalCache([]*model.ErrorPassthroughRule{newNonFailoverPassthroughRule(http.StatusUnprocessableEntity, "invalid schema", http.StatusTeapot, "上游请求失败")})
	BindErrorPassthroughService(c, ruleSvc)

	svc := &GatewayService{}
	respBody := []byte(`{"error":{"message":"Invalid schema for field messages"}}`)
	resp := &http.Response{
		StatusCode: http.StatusUnprocessableEntity,
		Body:       io.NopCloser(bytes.NewReader(respBody)),
		Header:     http.Header{},
	}
	account := &Account{ID: 1, Platform: PlatformAnthropic, Type: AccountTypeAPIKey}

	_, err := svc.handleErrorResponse(context.Background(), resp, c, account)
	require.Error(t, err)
	assert.Equal(t, http.StatusTeapot, rec.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	errField, ok := payload["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "upstream_error", errField["type"])
	assert.Equal(t, "上游请求失败", errField["message"])
}

func TestOpenAIHandleErrorResponse_AppliesRuleFor422(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	ruleSvc := &ErrorPassthroughService{}
	ruleSvc.setLocalCache([]*model.ErrorPassthroughRule{newNonFailoverPassthroughRule(http.StatusUnprocessableEntity, "invalid schema", http.StatusTeapot, "OpenAI上游失败")})
	BindErrorPassthroughService(c, ruleSvc)

	svc := &OpenAIGatewayService{}
	respBody := []byte(`{"error":{"message":"Invalid schema for field messages"}}`)
	resp := &http.Response{
		StatusCode: http.StatusUnprocessableEntity,
		Body:       io.NopCloser(bytes.NewReader(respBody)),
		Header:     http.Header{},
	}
	account := &Account{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

	_, err := svc.handleErrorResponse(context.Background(), resp, c, account, nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusTeapot, rec.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	errField, ok := payload["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "upstream_error", errField["type"])
	assert.Equal(t, "OpenAI上游失败", errField["message"])
}

func TestGeminiWriteGeminiMappedError_AppliesRuleFor422(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	ruleSvc := &ErrorPassthroughService{}
	ruleSvc.setLocalCache([]*model.ErrorPassthroughRule{newNonFailoverPassthroughRule(http.StatusUnprocessableEntity, "invalid schema", http.StatusTeapot, "Gemini上游失败")})
	BindErrorPassthroughService(c, ruleSvc)

	svc := &GeminiMessagesCompatService{}
	respBody := []byte(`{"error":{"code":422,"message":"Invalid schema for field messages","status":"INVALID_ARGUMENT"}}`)
	account := &Account{ID: 3, Platform: PlatformGemini, Type: AccountTypeAPIKey}

	err := svc.writeGeminiMappedError(c, account, http.StatusUnprocessableEntity, "req-1", respBody)
	require.Error(t, err)
	assert.Equal(t, http.StatusTeapot, rec.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	errField, ok := payload["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "upstream_error", errField["type"])
	assert.Equal(t, "Gemini上游失败", errField["message"])
}

func TestApplyErrorPassthroughRule_SkipMonitoringSetsContextKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	rule := newNonFailoverPassthroughRule(http.StatusBadRequest, "prompt is too long", http.StatusBadRequest, "上下文超限")
	rule.SkipMonitoring = true

	ruleSvc := &ErrorPassthroughService{}
	ruleSvc.setLocalCache([]*model.ErrorPassthroughRule{rule})
	BindErrorPassthroughService(c, ruleSvc)

	_, _, _, matched := applyErrorPassthroughRule(
		c,
		PlatformAnthropic,
		http.StatusBadRequest,
		[]byte(`{"error":{"message":"prompt is too long"}}`),
		http.StatusBadGateway,
		"upstream_error",
		"Upstream request failed",
	)

	assert.True(t, matched)
	v, exists := c.Get(OpsSkipPassthroughKey)
	assert.True(t, exists, "OpsSkipPassthroughKey should be set when skip_monitoring=true")
	boolVal, ok := v.(bool)
	assert.True(t, ok, "value should be bool")
	assert.True(t, boolVal)
}

func TestApplyErrorPassthroughRule_NoSkipMonitoringDoesNotSetContextKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	rule := newNonFailoverPassthroughRule(http.StatusBadRequest, "prompt is too long", http.StatusBadRequest, "上下文超限")
	rule.SkipMonitoring = false

	ruleSvc := &ErrorPassthroughService{}
	ruleSvc.setLocalCache([]*model.ErrorPassthroughRule{rule})
	BindErrorPassthroughService(c, ruleSvc)

	_, _, _, matched := applyErrorPassthroughRule(
		c,
		PlatformAnthropic,
		http.StatusBadRequest,
		[]byte(`{"error":{"message":"prompt is too long"}}`),
		http.StatusBadGateway,
		"upstream_error",
		"Upstream request failed",
	)

	assert.True(t, matched)
	_, exists := c.Get(OpsSkipPassthroughKey)
	assert.False(t, exists, "OpsSkipPassthroughKey should NOT be set when skip_monitoring=false")
}

// ---- ResponseCommittedKey: service 层写完错误响应后标记，handler 层检查跳过兜底写入 ----

func TestHandleErrorResponse_SetsResponseCommitted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	svc := &GatewayService{}
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":{"message":"temperature: range: 0..1"}}`))),
		Header:     http.Header{},
	}
	account := &Account{ID: 100, Platform: PlatformAnthropic, Type: AccountTypeAPIKey}

	_, err := svc.handleErrorResponse(context.Background(), resp, c, account)
	require.Error(t, err)
	assert.True(t, IsResponseCommitted(c), "non-failover error path must mark response committed")
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
}

func TestHandleErrorResponse_PassthroughRuleSetsCommitted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	ruleSvc := &ErrorPassthroughService{}
	ruleSvc.setLocalCache([]*model.ErrorPassthroughRule{
		newNonFailoverPassthroughRule(http.StatusBadRequest, "temperature", http.StatusBadRequest, "参数错误"),
	})
	BindErrorPassthroughService(c, ruleSvc)

	svc := &GatewayService{}
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":{"message":"temperature: range: 0..1"}}`))),
		Header:     http.Header{},
	}
	account := &Account{ID: 200, Platform: PlatformAnthropic, Type: AccountTypeAPIKey}

	_, err := svc.handleErrorResponse(context.Background(), resp, c, account)
	require.Error(t, err)
	assert.True(t, IsResponseCommitted(c), "passthrough rule path must mark response committed")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	errField, ok := payload["error"].(map[string]any)
	require.True(t, ok, "payload[\"error\"] should be map[string]any")
	assert.Equal(t, "参数错误", errField["message"])
}

func TestOpenAIHandleErrorResponse_SetsResponseCommitted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	svc := &OpenAIGatewayService{}
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":{"message":"rate limit exceeded"}}`))),
		Header:     http.Header{},
	}
	account := &Account{ID: 101, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

	_, err := svc.handleErrorResponse(context.Background(), resp, c, account, nil)
	require.Error(t, err)
	assert.True(t, IsResponseCommitted(c), "OpenAI non-failover path must mark response committed")
}

func TestGeminiWriteGeminiMappedError_SetsResponseCommitted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	svc := &GeminiMessagesCompatService{}
	body := []byte(`{"error":{"message":"invalid field"}}`)
	account := &Account{ID: 102, Platform: PlatformGemini, Type: AccountTypeAPIKey}

	err := svc.writeGeminiMappedError(c, account, http.StatusBadRequest, "req-99", body)
	require.Error(t, err)
	assert.True(t, IsResponseCommitted(c), "Gemini path must mark response committed")
}

func TestErrorPassthroughCompleteReplacementFor401And429(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusTooManyRequests} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			message := "  replacement only  "
			ruleSvc := &ErrorPassthroughService{}
			ruleSvc.setLocalCache([]*model.ErrorPassthroughRule{newNonFailoverPassthroughRule(status, "upstream secret", status, message)})
			BindErrorPassthroughService(c, ruleSvc)

			original := []byte(`{"error":{"type":"authentication_error","code":"invalid_api_key","message":"upstream secret detail"}}`)
			clientStatus, clientBody, matched := applyErrorPassthroughRuleToJSON(c, PlatformOpenAI, status, original)

			require.True(t, matched)
			assert.Equal(t, status, clientStatus)
			assert.JSONEq(t, `{"error":{"type":"authentication_error","code":"invalid_api_key","message":"replacement only"}}`, string(clientBody))
			assert.Contains(t, string(original), "upstream secret detail")
		})
	}
}

func TestCyberPolicyReplacementKeepsOriginalMarkEvidence(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	custom := "Request rejected"
	status := http.StatusBadRequest
	ruleSvc := &ErrorPassthroughService{}
	ruleSvc.setLocalCache([]*model.ErrorPassthroughRule{{
		Enabled: true, Priority: 1, Keywords: []string{"cyber_policy"}, MatchMode: model.MatchModeAny,
		Platforms: []string{PlatformOpenAI}, PassthroughCode: true, PassthroughBody: false, CustomMessage: &custom,
	}})
	BindErrorPassthroughService(c, ruleSvc)
	original := []byte(`{"type":"response.failed","response":{"error":{"type":"safety_error","code":"cyber_policy","message":"original evidence"}}}`)
	MarkOpsCyberPolicy(c, CyberPolicyMark{Code: "cyber_policy", Message: "original evidence", Body: string(original), UpstreamStatus: status})

	clientBody := applyErrorPassthroughRuleToOpenAIWSEvent(c, PlatformOpenAI, original)
	mark := GetOpsCyberPolicy(c)
	require.NotNil(t, mark)
	assert.Equal(t, "original evidence", mark.Message)
	assert.Contains(t, mark.Body, "original evidence")
	assert.Equal(t, "Request rejected", gjson.GetBytes(clientBody, "response.error.message").String())
	assert.Equal(t, "cyber_policy", gjson.GetBytes(clientBody, "response.error.code").String())
}

func TestNativeGeminiErrorReplacementPreservesGoogleEnvelope(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	custom := "Gemini unavailable"
	ruleSvc := &ErrorPassthroughService{}
	ruleSvc.setLocalCache([]*model.ErrorPassthroughRule{newNonFailoverPassthroughRule(http.StatusTooManyRequests, "quota", http.StatusServiceUnavailable, custom)})
	BindErrorPassthroughService(c, ruleSvc)
	original := []byte(`{"error":{"code":429,"message":"quota exhausted","status":"RESOURCE_EXHAUSTED","details":[{"reason":"RATE_LIMIT"}]}}`)

	status, clientBody, matched := applyErrorPassthroughRuleToGoogleJSON(c, PlatformGemini, http.StatusTooManyRequests, original)

	require.True(t, matched)
	assert.Equal(t, http.StatusServiceUnavailable, status)
	assert.Equal(t, float64(http.StatusServiceUnavailable), gjson.GetBytes(clientBody, "error.code").Float())
	assert.Equal(t, "RESOURCE_EXHAUSTED", gjson.GetBytes(clientBody, "error.status").String())
	assert.Equal(t, "RATE_LIMIT", gjson.GetBytes(clientBody, "error.details.0.reason").String())
	assert.Equal(t, custom, gjson.GetBytes(clientBody, "error.message").String())
}

func TestErrorReplacementSupportsProtocolMessageShapes(t *testing.T) {
	custom := "replacement"
	ruleSvc := &ErrorPassthroughService{}
	ruleSvc.setLocalCache([]*model.ErrorPassthroughRule{newNonFailoverPassthroughRule(http.StatusBadRequest, "original", http.StatusBadRequest, custom)})
	for _, original := range []string{
		`{"message":"original"}`,
		`{"detail":"original"}`,
		`{"error":"original"}`,
	} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		BindErrorPassthroughService(c, ruleSvc)
		_, body, matched := applyErrorPassthroughRuleToJSON(c, PlatformOpenAI, http.StatusBadRequest, []byte(original))
		require.True(t, matched)
		require.NotContains(t, string(body), "original")
		require.Contains(t, string(body), custom)
	}
}

func TestErrorReplacementPreservesStructuredErrorWithoutMessage(t *testing.T) {
	custom := "replacement"
	ruleSvc := &ErrorPassthroughService{}
	ruleSvc.setLocalCache([]*model.ErrorPassthroughRule{newNonFailoverPassthroughRule(http.StatusBadRequest, "marker", http.StatusBadRequest, custom)})
	original := []byte(`{"error":{"code":"marker","type":"invalid_request_error"}}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	BindErrorPassthroughService(c, ruleSvc)

	_, body, matched := applyErrorPassthroughRuleToJSONPath(c, PlatformOpenAI, http.StatusBadRequest, original, "error.message")
	require.True(t, matched)
	require.Equal(t, "marker", gjson.GetBytes(body, "error.code").String())
	require.Equal(t, "invalid_request_error", gjson.GetBytes(body, "error.type").String())
	require.Equal(t, custom, gjson.GetBytes(body, "error.message").String())
}

func TestKnownSerializerInsertsMessageWithoutCorruptingArbitraryJSON(t *testing.T) {
	ruleSvc := &ErrorPassthroughService{}
	ruleSvc.setLocalCache([]*model.ErrorPassthroughRule{newNonFailoverPassthroughRule(http.StatusBadRequest, "marker", http.StatusBadRequest, "replacement")})
	original := []byte(`{"code":"marker","metadata":{"keep":true}}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	BindErrorPassthroughService(c, ruleSvc)
	_, generic, matched := applyErrorPassthroughRuleToJSON(c, PlatformOpenAI, http.StatusBadRequest, original)
	require.True(t, matched)
	require.JSONEq(t, string(original), string(generic))
	_, known, matched := applyErrorPassthroughRuleToJSONPath(c, PlatformOpenAI, http.StatusBadRequest, original, "error.message")
	require.True(t, matched)
	require.Equal(t, "replacement", gjson.GetBytes(known, "error.message").String())
	require.True(t, gjson.GetBytes(known, "metadata.keep").Bool())
}

func TestOpenAIWSErrorAndResponseFailedReplacementKeepsClassification(t *testing.T) {
	custom := "Please retry later"
	ruleSvc := &ErrorPassthroughService{}
	ruleSvc.setLocalCache([]*model.ErrorPassthroughRule{newNonFailoverPassthroughRule(http.StatusTooManyRequests, "rate_limit", http.StatusTooManyRequests, custom)})

	for _, payload := range [][]byte{
		[]byte(`{"type":"error","error":{"type":"rate_limit_error","code":"rate_limit_exceeded","message":"rate_limit original"}}`),
		[]byte(`{"type":"response.failed","response":{"error":{"type":"rate_limit_error","code":"rate_limit_exceeded","message":"rate_limit original"}}}`),
	} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		BindErrorPassthroughService(c, ruleSvc)
		code, errType, _ := parseOpenAIWSErrorEventFields(payload)
		originalStatus := openAIWSErrorHTTPStatusFromRaw(code, errType)
		if gjson.GetBytes(payload, "type").String() == "response.failed" {
			originalStatus = openAIStreamFailedEventSemanticStatus(payload, extractOpenAISSEErrorMessage(payload))
		}

		clientBody := applyErrorPassthroughRuleToOpenAIWSEvent(c, PlatformOpenAI, payload)

		assert.Equal(t, http.StatusTooManyRequests, originalStatus, "failover classification must use original fields")
		assert.Equal(t, "rate_limit_exceeded", firstNonEmpty(
			gjson.GetBytes(clientBody, "error.code").String(),
			gjson.GetBytes(clientBody, "response.error.code").String(),
		))
		assert.NotContains(t, string(clientBody), "rate_limit original")
		assert.Contains(t, string(clientBody), custom)
	}
}

func TestMatching429RuleDoesNotSuppressOpenAIFailover(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	ruleSvc := &ErrorPassthroughService{}
	ruleSvc.setLocalCache([]*model.ErrorPassthroughRule{newNonFailoverPassthroughRule(
		http.StatusTooManyRequests, "rate limit", http.StatusTeapot, "custom terminal message",
	)})
	BindErrorPassthroughService(c, ruleSvc)
	account := &Account{ID: 44, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	resp := &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{}}
	body := []byte(`{"error":{"message":"rate limit"}}`)

	failoverErr := (&OpenAIGatewayService{}).failoverOpenAIUpstreamHTTPError(
		context.Background(), c, account, resp, body, "rate limit", "gpt-5",
	)

	require.NotNil(t, failoverErr)
	require.Equal(t, http.StatusTooManyRequests, failoverErr.StatusCode)
	require.Equal(t, PlatformOpenAI, failoverErr.Platform)
	require.Equal(t, body, failoverErr.ResponseBody)
	require.False(t, c.Writer.Written(), "customization is terminal-only and must not write during failover")
}

func TestRuntimeCustomMessageCapIsUTF8Safe(t *testing.T) {
	message := strings.Repeat("界", model.MaxErrorPassthroughCustomMessageLength+5)
	got := SanitizeErrorPassthroughCustomMessage(message)
	assert.True(t, utf8.ValidString(got))
	assert.Equal(t, model.MaxErrorPassthroughCustomMessageLength, utf8.RuneCountInString(got))
	assert.True(t, utf8.ValidString(TruncateErrorPassthroughWSReason(strings.Repeat("界", 100))))
	assert.LessOrEqual(t, len(TruncateErrorPassthroughWSReason(strings.Repeat("界", 100))), 120)
}

func newNonFailoverPassthroughRule(statusCode int, keyword string, respCode int, customMessage string) *model.ErrorPassthroughRule {
	return &model.ErrorPassthroughRule{
		ID:              1,
		Name:            "non-failover-rule",
		Enabled:         true,
		Priority:        1,
		ErrorCodes:      []int{statusCode},
		Keywords:        []string{keyword},
		MatchMode:       model.MatchModeAll,
		PassthroughCode: false,
		ResponseCode:    &respCode,
		PassthroughBody: false,
		CustomMessage:   &customMessage,
	}
}
