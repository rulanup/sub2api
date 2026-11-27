package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func videoFlowResp(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), ContentLength: int64(len(body))}
}

func videoFlowRun(t *testing.T, up *httpUpstreamRecorder, rangeHeader string) (*OpenAIForwardResult, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/t1/content", nil)
	if rangeHeader != "" {
		c.Request.Header.Set("Range", rangeHeader)
	}
	acct := &Account{ID: 9, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "sk-up", "base_url": "https://up.example.com", "openai_capabilities": []string{"videos"}}}
	result, err := (&OpenAIGatewayService{httpUpstream: up}).ForwardOpenAIVideo(context.Background(), c, acct, OpenAIVideoEndpointContent, OpenAIVideoTaskKey("t1"), nil, "")
	require.NoError(t, err)
	return result, w
}

func TestOpenAIVideoContentSameHostCarriesKey(t *testing.T) {
	up := &httpUpstreamRecorder{responses: []*http.Response{
		videoFlowResp(200, `{"status":"completed","video_url":"/files/v.mp4"}`),
		videoFlowResp(206, "VIDEO"),
	}}
	result, w := videoFlowRun(t, up, "bytes=0-4")
	require.Equal(t, 1, result.VideoCount)
	require.Equal(t, "VIDEO", w.Body.String())
	require.Len(t, up.requests, 2)
	require.Equal(t, "https://up.example.com/v1/videos/t1", up.requests[0].URL.String())
	require.Equal(t, "https://up.example.com/files/v.mp4", up.requests[1].URL.String())
	require.Equal(t, "Bearer sk-up", up.requests[1].Header.Get("Authorization"))
	require.Equal(t, "bytes=0-4", up.requests[1].Header.Get("Range"))
}

func TestOpenAIVideoContentCDNOmitsKey(t *testing.T) {
	gotAuth := "unset"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte("CDN"))
	}))
	defer srv.Close()
	orig := openAIVideoCDNClient
	t.Cleanup(func() { openAIVideoCDNClient = orig })
	openAIVideoCDNClient = func(string) *http.Client { return srv.Client() }
	up := &httpUpstreamRecorder{responses: []*http.Response{videoFlowResp(200, `{"status":"completed","metadata":{"url":"`+srv.URL+`/v.mp4"}}`)}}
	result, w := videoFlowRun(t, up, "")
	require.Equal(t, 1, result.VideoCount)
	require.Equal(t, "CDN", w.Body.String())
	require.Empty(t, gotAuth)
	require.Len(t, up.requests, 1)
}

func TestOpenAIVideoContentFallsBackToContentEndpoint(t *testing.T) {
	up := &httpUpstreamRecorder{responses: []*http.Response{videoFlowResp(200, `{"status":"completed"}`), videoFlowResp(200, "RAW")}}
	result, w := videoFlowRun(t, up, "")
	require.Equal(t, 1, result.VideoCount)
	require.Equal(t, "RAW", w.Body.String())
	require.Len(t, up.requests, 2)
	require.Equal(t, "https://up.example.com/v1/videos/t1/content", up.requests[1].URL.String())
}

func TestOpenAIVideoContentDownloadFailureIsNotBilled(t *testing.T) {
	up := &httpUpstreamRecorder{responses: []*http.Response{videoFlowResp(200, `{"video_url":"https://up.example.com/v.mp4"}`), videoFlowResp(403, `{"error":"expired"}`)}}
	result, w := videoFlowRun(t, up, "")
	require.Equal(t, http.StatusBadGateway, w.Code)
	require.Zero(t, result.VideoCount)
	up = &httpUpstreamRecorder{responses: []*http.Response{videoFlowResp(200, `{"video_url":"ftp://x/v.mp4"}`)}}
	result, w = videoFlowRun(t, up, "")
	require.Equal(t, http.StatusBadGateway, w.Code)
	require.Zero(t, result.VideoCount)
	require.Len(t, up.requests, 1)
}
