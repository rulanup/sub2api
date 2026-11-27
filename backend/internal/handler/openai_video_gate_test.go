//go:build unit

package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

var openAIVideoGateMessages = []string{"is required for per-second", "must be between", "no video price configured"}

func openAIVideoGateRun(t *testing.T, group *service.Group, body string) *httptest.ResponseRecorder {
	t.Helper()
	h, _, _, _ := newGrokMediaSlotHandler(t, true, false)
	gid := group.ID
	req := httptest.NewRequest(http.MethodPost, "/v1/videos", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{ID: 20, UserID: 10, GroupID: &gid, Group: group, User: &service.User{ID: 10}})
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 10, Concurrency: 5})
	h.OpenAIVideoCreate(c)
	return w
}

func TestOpenAIVideoCreateDurationGate(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	perSec := func() *service.Group {
		return &service.Group{ID: 31, Platform: service.PlatformOpenAI, Status: service.StatusActive, AllowImageGeneration: true, VideoPrice480P: f(0.1)}
	}
	w := openAIVideoGateRun(t, perSec(), `{"model":"vid-a","prompt":"p"}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "is required for per-second")
	w = openAIVideoGateRun(t, perSec(), `{"model":"vid-a","prompt":"p","seconds":"3601"}`)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "must be between 1 and 3600")
	for _, body := range []string{`{"model":"vid-a","prompt":"p","seconds":"60"}`, `{"model":"vid-a","prompt":"p","duration":60}`} {
		w = openAIVideoGateRun(t, perSec(), body)
		for _, msg := range openAIVideoGateMessages {
			require.NotContains(t, w.Body.String(), msg, body)
		}
	}
	// 按次计费模型免带时长：handler 测试的 service 没有注入定价解析器（生产由 wire 注入），
	// 该组合由 service 包 TestVideoModelPriceConfigured / TestVideoModelBilledPerSecond 覆盖。
}

func TestIsVideoCompletionBillingEndpoint(t *testing.T) {
	for _, e := range []service.GrokMediaEndpoint{service.GrokMediaEndpointVideoStatus, service.GrokMediaEndpointVideoContent, service.OpenAIVideoEndpointStatus, service.OpenAIVideoEndpointContent} {
		require.True(t, isVideoCompletionBillingEndpoint(e), string(e))
	}
	for _, e := range []service.GrokMediaEndpoint{service.OpenAIVideoEndpointCreate, service.GrokMediaEndpointVideosGenerations, service.SeedanceEndpointStatus} {
		require.False(t, isVideoCompletionBillingEndpoint(e), string(e))
	}
}
