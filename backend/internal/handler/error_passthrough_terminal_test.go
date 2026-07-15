package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/model"
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
