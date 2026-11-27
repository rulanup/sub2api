package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

// 完成任务的视频地址字段（straitapi / new-api 文档顺序）；data 兼容数组与对象两种包装。
var openAIVideoResultURLPaths = []string{
	"video_url", "url", "content_url", "metadata.video_url", "metadata.url",
	"data.0.video_url", "data.0.url", "videos.0.url", "data.video_url", "data.url",
}

const openAIVideoStatusBodyLimit = 4 << 20

func openAIVideoResultURL(body []byte) string {
	for _, p := range openAIVideoResultURLPaths {
		if v := gjson.GetBytes(body, p); v.Type == gjson.String {
			if u := strings.TrimSpace(v.String()); u != "" {
				return u
			}
		}
	}
	return ""
}

var sharedOpenAIVideoDownloadClient = sync.OnceValue(func() *http.Client {
	return newOpenAIVideoDownloadClient(http.ProxyFromEnvironment)
})

// newOpenAIVideoDownloadClient 复用图片下载的 SSRF 防护（socket 层拒绝非公网 IP，重定向每跳校验）；
// 视频体积大，不设整体超时，由请求 ctx 控制；重定向最多 3 跳且不带 Authorization。
func newOpenAIVideoDownloadClient(proxy func(*http.Request) (*url.URL, error)) *http.Client {
	client := newImageDownloadHTTPClient(proxy)
	client.Timeout = 0
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return fmt.Errorf("too many redirects")
		}
		req.Header.Del("Authorization")
		return nil
	}
	return client
}

func openAIVideoContentError(c *gin.Context, message string) {
	if c == nil || IsResponseCommitted(c) {
		return
	}
	c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"type": "upstream_error", "message": message}})
}

// forwardOpenAIVideoResultURL 处理没有 /content 接口的上游：先查任务状态，从结果字段取视频地址再转发。
// 取不到地址时 handled=false，调用方回退到 {base}/v1/videos/{id}/content 透传。
// 地址与 base_url 同主机才带上游 key；第三方 CDN 不带 key，经 SSRF 安全客户端下载。
// 下载失败返回 502 且不计费、不记账号故障（结果地址过期很常见）。
func (s *OpenAIGatewayService) forwardOpenAIVideoResultURL(ctx context.Context, c *gin.Context, account *Account, base, token, taskKey string) (handled, billable bool) {
	statusURL, err := buildOpenAIVideoURL(base, OpenAIVideoEndpointStatus, openAIVideoUpstreamID(taskKey))
	baseURL, baseErr := url.Parse(base)
	if err != nil || baseErr != nil {
		return false, false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
	if err != nil {
		return false, false
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	account.ApplyHeaderOverrides(req.Header)
	proxy := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxy = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.Do(req, proxy, account.ID, account.Concurrency)
	if err != nil {
		return false, false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, openAIVideoStatusBodyLimit))
	_ = resp.Body.Close()
	raw := ""
	if err == nil && resp.StatusCode == http.StatusOK {
		raw = openAIVideoResultURL(body)
	}
	if raw == "" {
		return false, false
	}
	target, err := baseURL.Parse(raw)
	if err != nil || (target.Scheme != "https" && target.Scheme != "http") || target.User != nil || target.Host == "" {
		openAIVideoContentError(c, "upstream returned an unsupported video URL")
		return true, false
	}
	dlReq, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		openAIVideoContentError(c, "upstream returned an unsupported video URL")
		return true, false
	}
	if r := strings.TrimSpace(c.GetHeader("Range")); r != "" {
		dlReq.Header.Set("Range", r)
	}
	var dl *http.Response
	if strings.EqualFold(target.Host, baseURL.Host) {
		dlReq.Header.Set("Authorization", "Bearer "+token)
		account.ApplyHeaderOverrides(dlReq.Header)
		dl, err = s.httpUpstream.Do(dlReq, proxy, account.ID, account.Concurrency)
	} else {
		client := openAIVideoCDNClient(proxy)
		if proxy != "" {
			defer client.CloseIdleConnections()
		}
		dl, err = client.Do(dlReq)
	}
	if err != nil {
		openAIVideoContentError(c, "video content download failed")
		return true, false
	}
	defer func() { _ = dl.Body.Close() }()
	if dl.StatusCode < 200 || dl.StatusCode >= 300 {
		openAIVideoContentError(c, fmt.Sprintf("video content download failed: upstream status %d", dl.StatusCode))
		return true, false
	}
	if err := writeGrokMediaContentResponse(c, dl); err != nil {
		return true, false
	}
	return true, true
}

// openAIVideoCDNClient 返回第三方 CDN 视频下载客户端（SSRF 安全；配了账号代理时经代理）。
// 声明为变量仅供测试替换：httptest 监听回环地址，会被 SSRF 防护拦截。
var openAIVideoCDNClient = func(proxy string) *http.Client {
	if proxy != "" {
		if p, err := url.Parse(proxy); err == nil {
			return newOpenAIVideoDownloadClient(http.ProxyURL(p))
		}
	}
	return sharedOpenAIVideoDownloadClient()
}
