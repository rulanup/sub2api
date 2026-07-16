package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type terminalErrorRuleRepo struct {
	rules []*model.ErrorPassthroughRule
}

func (r terminalErrorRuleRepo) List(context.Context) ([]*model.ErrorPassthroughRule, error) {
	return r.rules, nil
}
func (r terminalErrorRuleRepo) GetByID(context.Context, int64) (*model.ErrorPassthroughRule, error) {
	return nil, nil
}
func (r terminalErrorRuleRepo) Create(_ context.Context, rule *model.ErrorPassthroughRule) (*model.ErrorPassthroughRule, error) {
	return rule, nil
}
func (r terminalErrorRuleRepo) Update(_ context.Context, rule *model.ErrorPassthroughRule) (*model.ErrorPassthroughRule, error) {
	return rule, nil
}
func (r terminalErrorRuleRepo) Delete(context.Context, int64) error { return nil }

func terminalErrorRuleService(platform string, skipMonitoring bool) *service.ErrorPassthroughService {
	status := http.StatusTeapot
	message := "custom exhausted message"
	return service.NewErrorPassthroughService(terminalErrorRuleRepo{rules: []*model.ErrorPassthroughRule{{
		ID: 1, Name: "terminal", Enabled: true, Priority: 1,
		ErrorCodes: []int{http.StatusTooManyRequests}, Keywords: []string{"original upstream"},
		MatchMode: model.MatchModeAll, Platforms: []string{platform}, PassthroughCode: false,
		ResponseCode: &status, PassthroughBody: false, CustomMessage: &message, SkipMonitoring: skipMonitoring,
	}}}, nil)
}

type terminalWhitelistSettingRepo struct{}

func (terminalWhitelistSettingRepo) Get(context.Context, string) (*service.Setting, error) {
	return &service.Setting{Value: `[42]`}, nil
}
func (terminalWhitelistSettingRepo) GetValue(context.Context, string) (string, error) {
	return `[42]`, nil
}
func (terminalWhitelistSettingRepo) Set(context.Context, string, string) error { return nil }
func (terminalWhitelistSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	return nil, nil
}
func (terminalWhitelistSettingRepo) SetMultiple(context.Context, map[string]string) error { return nil }
func (terminalWhitelistSettingRepo) GetAll(context.Context) (map[string]string, error) {
	return nil, nil
}
func (terminalWhitelistSettingRepo) Delete(context.Context, string) error { return nil }

func terminalWhitelistedErrorRuleService(platform string) *service.ErrorPassthroughService {
	svc := terminalErrorRuleService(platform, true)
	svc.SetSettingRepository(terminalWhitelistSettingRepo{})
	return svc
}

func terminalStatusOnlyRuleService(platform string) *service.ErrorPassthroughService {
	status := http.StatusTeapot
	message := "status-only customization"
	return service.NewErrorPassthroughService(terminalErrorRuleRepo{rules: []*model.ErrorPassthroughRule{{
		ID: 2, Name: "status-only", Enabled: true, Priority: 1,
		ErrorCodes: []int{http.StatusUnauthorized}, MatchMode: model.MatchModeAny,
		Platforms: []string{platform}, PassthroughCode: false, ResponseCode: &status,
		PassthroughBody: false, CustomMessage: &message,
	}}}, nil)
}

func TestCCFailoverExhaustedUsesActualPlatformAndPreservesOpsEvidence(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	h := &GatewayHandler{errorPassthroughService: terminalErrorRuleService(service.PlatformAntigravity, false)}
	failoverErr := &service.UpstreamFailoverError{
		StatusCode:   http.StatusTooManyRequests,
		Platform:     service.PlatformAntigravity,
		ResponseBody: []byte(`{"error":{"message":"original upstream detail"}}`),
	}

	h.handleCCFailoverExhausted(c, failoverErr, false)

	require.Equal(t, http.StatusTeapot, rec.Code)
	require.Equal(t, "custom exhausted message", gjson.GetBytes(rec.Body.Bytes(), "error.message").String())
	require.Equal(t, http.StatusTooManyRequests, c.GetInt(service.OpsUpstreamStatusCodeKey))
	require.Equal(t, "original upstream detail", c.GetString(service.OpsUpstreamErrorMessageKey))
}

func TestWSFailoverExhaustedHonorsSkipMonitoringAndHTTPClass(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	h := &OpenAIGatewayHandler{errorPassthroughService: terminalErrorRuleService(service.PlatformOpenAI, true)}
	failoverErr := &service.UpstreamFailoverError{
		StatusCode:   http.StatusTooManyRequests,
		Platform:     service.PlatformOpenAI,
		ResponseBody: []byte(`{"error":{"message":"original upstream detail"}}`),
	}

	h.closeOpenAIWSFailoverExhausted(c, nil, service.PlatformAntigravity, failoverErr)

	skip, exists := c.Get(service.OpsSkipPassthroughKey)
	require.True(t, exists)
	require.Equal(t, true, skip)
	require.Equal(t, coderws.StatusPolicyViolation, openAIWSCloseStatusForHTTP(http.StatusTeapot, coderws.StatusTryAgainLater))
	require.Equal(t, coderws.StatusTryAgainLater, openAIWSCloseStatusForHTTP(http.StatusServiceUnavailable, coderws.StatusInternalError))
}

func TestCCFailoverExhaustedWhitelistOverridesCustomization(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxkey.UserID, int64(42)))
	c.Request = req
	h := &GatewayHandler{errorPassthroughService: terminalWhitelistedErrorRuleService(service.PlatformAntigravity)}
	failoverErr := &service.UpstreamFailoverError{
		StatusCode:   http.StatusTooManyRequests,
		Platform:     service.PlatformAntigravity,
		ResponseBody: []byte(`{"error":{"message":"original upstream detail"}}`),
	}

	h.handleCCFailoverExhausted(c, failoverErr, false)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.Equal(t, "original upstream detail", gjson.GetBytes(rec.Body.Bytes(), "error.message").String())
	_, marked := c.Get(service.OpsSkipPassthroughKey)
	require.False(t, marked)
}

func whitelistedTerminalContext() (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	c.Request = req.WithContext(context.WithValue(req.Context(), ctxkey.UserID, int64(42)))
	return c, rec
}

func TestEmptyBodyTerminalFailoverWhitelistPreservesStatus(t *testing.T) {
	const upstreamStatus = http.StatusUnauthorized
	failoverErr := func(platform string) *service.UpstreamFailoverError {
		return &service.UpstreamFailoverError{StatusCode: upstreamStatus, Platform: platform}
	}

	t.Run("gateway", func(t *testing.T) {
		c, rec := whitelistedTerminalContext()
		h := &GatewayHandler{errorPassthroughService: terminalWhitelistedErrorRuleService(service.PlatformAnthropic)}
		h.handleFailoverExhausted(c, failoverErr(service.PlatformAnthropic), service.PlatformAnthropic, false)
		require.Equal(t, upstreamStatus, rec.Code)
		require.NotEmpty(t, gjson.GetBytes(rec.Body.Bytes(), "error.message").String())
	})

	t.Run("chat completions", func(t *testing.T) {
		c, rec := whitelistedTerminalContext()
		h := &GatewayHandler{errorPassthroughService: terminalWhitelistedErrorRuleService(service.PlatformAnthropic)}
		h.handleCCFailoverExhausted(c, failoverErr(service.PlatformAnthropic), false)
		require.Equal(t, upstreamStatus, rec.Code)
		require.Equal(t, "All available accounts exhausted", gjson.GetBytes(rec.Body.Bytes(), "error.message").String())
	})

	t.Run("responses", func(t *testing.T) {
		c, rec := whitelistedTerminalContext()
		h := &GatewayHandler{errorPassthroughService: terminalWhitelistedErrorRuleService(service.PlatformAnthropic)}
		h.handleResponsesFailoverExhausted(c, failoverErr(service.PlatformAnthropic), false)
		require.Equal(t, upstreamStatus, rec.Code)
		require.Equal(t, "All available accounts exhausted", gjson.GetBytes(rec.Body.Bytes(), "error.message").String())
	})

	t.Run("Gemini", func(t *testing.T) {
		c, rec := whitelistedTerminalContext()
		h := &GatewayHandler{errorPassthroughService: terminalWhitelistedErrorRuleService(service.PlatformGemini)}
		h.handleGeminiFailoverExhausted(c, failoverErr(service.PlatformGemini))
		require.Equal(t, upstreamStatus, rec.Code)
		require.NotEmpty(t, gjson.GetBytes(rec.Body.Bytes(), "error.message").String())
	})

	t.Run("OpenAI", func(t *testing.T) {
		c, rec := whitelistedTerminalContext()
		h := &OpenAIGatewayHandler{errorPassthroughService: terminalWhitelistedErrorRuleService(service.PlatformOpenAI)}
		h.handleFailoverExhausted(c, failoverErr(service.PlatformOpenAI), false)
		require.Equal(t, upstreamStatus, rec.Code)
		require.NotEmpty(t, gjson.GetBytes(rec.Body.Bytes(), "error.message").String())
	})

	t.Run("OpenAI Anthropic envelope", func(t *testing.T) {
		c, rec := whitelistedTerminalContext()
		h := &OpenAIGatewayHandler{errorPassthroughService: terminalWhitelistedErrorRuleService(service.PlatformOpenAI)}
		h.handleAnthropicFailoverExhausted(c, failoverErr(service.PlatformOpenAI), false)
		require.Equal(t, upstreamStatus, rec.Code)
		require.NotEmpty(t, gjson.GetBytes(rec.Body.Bytes(), "error.message").String())
	})

	t.Run("gateway simple", func(t *testing.T) {
		c, rec := whitelistedTerminalContext()
		h := &GatewayHandler{errorPassthroughService: terminalWhitelistedErrorRuleService(service.PlatformAnthropic)}
		h.handleFailoverExhaustedSimple(c, upstreamStatus, false)
		require.Equal(t, upstreamStatus, rec.Code)
	})

	t.Run("OpenAI simple", func(t *testing.T) {
		c, rec := whitelistedTerminalContext()
		h := &OpenAIGatewayHandler{errorPassthroughService: terminalWhitelistedErrorRuleService(service.PlatformOpenAI)}
		h.handleFailoverExhaustedSimple(c, upstreamStatus, false)
		require.Equal(t, upstreamStatus, rec.Code)
	})
}

func TestWhitelistedSilentRefusalRemainsSanitized(t *testing.T) {
	c, rec := whitelistedTerminalContext()
	h := &GatewayHandler{errorPassthroughService: terminalWhitelistedErrorRuleService(service.PlatformOpenAI)}
	hidden := "hidden detector evidence"
	body := []byte(`{"error":{"code":"openai_silent_refusal","message":"` + hidden + `"}}`)

	h.handleFailoverExhausted(c, &service.UpstreamFailoverError{
		StatusCode: http.StatusBadGateway, Platform: service.PlatformOpenAI, ResponseBody: body,
	}, service.PlatformOpenAI, false)

	require.Equal(t, http.StatusBadGateway, rec.Code)
	require.Equal(t, service.OpenAISilentRefusalClientMessage(), gjson.GetBytes(rec.Body.Bytes(), "error.message").String())
	require.NotContains(t, rec.Body.String(), hidden)
	_, marked := c.Get(service.OpsSkipPassthroughKey)
	require.False(t, marked, "whitelist bypasses editable customization, not security sanitization")
}

func TestEmptyBodySimpleFailoverStillAppliesStatusOnlyRules(t *testing.T) {
	t.Run("gateway", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		h := &GatewayHandler{errorPassthroughService: terminalStatusOnlyRuleService(service.PlatformAnthropic)}
		h.handleFailoverExhaustedSimple(c, http.StatusUnauthorized, false)
		require.Equal(t, http.StatusTeapot, rec.Code)
		require.Equal(t, "status-only customization", gjson.GetBytes(rec.Body.Bytes(), "error.message").String())
	})

	t.Run("OpenAI", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		h := &OpenAIGatewayHandler{errorPassthroughService: terminalStatusOnlyRuleService(service.PlatformOpenAI)}
		h.handleFailoverExhaustedSimple(c, http.StatusUnauthorized, false)
		require.Equal(t, http.StatusTeapot, rec.Code)
		require.Equal(t, "status-only customization", gjson.GetBytes(rec.Body.Bytes(), "error.message").String())
	})
}

func TestWhitelistedStreamingFailoverPreservesSSEEnvelopeAndEvidence(t *testing.T) {
	c, rec := whitelistedTerminalContext()
	h := &GatewayHandler{errorPassthroughService: terminalWhitelistedErrorRuleService(service.PlatformAnthropic)}
	body := []byte(`{"error":{"message":"original stream failure"}}`)

	h.handleFailoverExhausted(c, &service.UpstreamFailoverError{
		StatusCode: http.StatusTooManyRequests, Platform: service.PlatformAnthropic, ResponseBody: body,
	}, service.PlatformAnthropic, true)

	require.Contains(t, rec.Body.String(), `data: {"type":"error"`)
	require.Contains(t, rec.Body.String(), "original stream failure")
	streamErr, ok := service.GetOpsStreamError(c)
	require.True(t, ok)
	require.Equal(t, http.StatusTooManyRequests, streamErr.IntendedStatus)
	require.Equal(t, "original stream failure", streamErr.Message)
	_, marked := c.Get(service.OpsSkipPassthroughKey)
	require.False(t, marked)
}

func TestWhitelistedWSFailoverClosePreservesOpsEvidenceWithoutSkip(t *testing.T) {
	c, _ := whitelistedTerminalContext()
	h := &OpenAIGatewayHandler{errorPassthroughService: terminalWhitelistedErrorRuleService(service.PlatformOpenAI)}
	h.closeOpenAIWSFailoverExhausted(c, nil, service.PlatformOpenAI, &service.UpstreamFailoverError{
		StatusCode:   http.StatusTooManyRequests,
		Platform:     service.PlatformOpenAI,
		ResponseBody: []byte(`{"error":{"message":"original websocket failure"}}`),
	})

	require.Equal(t, http.StatusTooManyRequests, c.GetInt(service.OpsUpstreamStatusCodeKey))
	require.Equal(t, "original websocket failure", c.GetString(service.OpsUpstreamErrorMessageKey))
	_, marked := c.Get(service.OpsSkipPassthroughKey)
	require.False(t, marked)
}
