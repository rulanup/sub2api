package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// OpenAI 兼容的 /v1/videos 异步视频任务（new-api 等聚合上游）。提交、查询、下载原样透传到账号上游，
// 只按账号模型映射改写 model。任务 ID 加命名空间，归属绑定与计费键和 Grok / Seedance 隔离。
const (
	OpenAIVideoEndpointCreate      GrokMediaEndpoint        = "openai_video_create"
	OpenAIVideoEndpointStatus      GrokMediaEndpoint        = "openai_video_status"
	OpenAIVideoEndpointContent     GrokMediaEndpoint        = "openai_video_content"
	OpenAIEndpointCapabilityVideos OpenAIEndpointCapability = "videos"

	openAIVideoTaskPrefix = "openai-video:"
)

// IsOpenAIVideo 判断是否为 OpenAI 兼容视频任务的端点。
func (e GrokMediaEndpoint) IsOpenAIVideo() bool {
	return e == OpenAIVideoEndpointCreate || e == OpenAIVideoEndpointStatus || e == OpenAIVideoEndpointContent
}

// OpenAIVideoTaskKey 把上游任务 ID 转成网关内部的任务键；空 ID 返回空串。
func OpenAIVideoTaskKey(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	return openAIVideoTaskPrefix + id
}

func openAIVideoUpstreamID(taskKey string) string {
	return strings.TrimPrefix(strings.TrimSpace(taskKey), openAIVideoTaskPrefix)
}

// ParseOpenAIVideoRequest 解析 /v1/videos 提交请求（JSON 或 multipart）。
// DurationSeconds 取请求里的 seconds（或 duration），缺省为 0，由计费决定如何处理。
func ParseOpenAIVideoRequest(contentType string, body []byte) (GrokMediaRequestInfo, error) {
	info := GrokMediaRequestInfo{N: 1}
	fields := map[string]string{}
	textRefs := map[string][]string{}
	if gjson.ValidBytes(body) {
		root := gjson.ParseBytes(body)
		if !root.IsObject() {
			return info, fmt.Errorf("request body must be a JSON object")
		}
		for _, name := range []string{"model", "prompt", "seconds", "duration", "size", "resolution", "resolution_name"} {
			if value := root.Get(name); value.Exists() {
				fields[name] = strings.TrimSpace(value.String())
			}
		}
	} else {
		mediaType, params, err := mime.ParseMediaType(strings.TrimSpace(contentType))
		if err != nil || !strings.EqualFold(mediaType, "multipart/form-data") || strings.TrimSpace(params["boundary"]) == "" {
			return info, fmt.Errorf("request body must be JSON or multipart/form-data")
		}
		reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return info, fmt.Errorf("invalid multipart body: %w", err)
			}
			name := strings.TrimSpace(part.FormName())
			data, readErr := io.ReadAll(part)
			_ = part.Close()
			if readErr != nil {
				return info, fmt.Errorf("invalid multipart body: %w", readErr)
			}
			if name == "" {
				continue
			}
			if fileName := strings.TrimSpace(part.FileName()); fileName != "" {
				info.Uploads = append(info.Uploads, OpenAIImagesUpload{
					FieldName:   name,
					FileName:    fileName,
					ContentType: strings.TrimSpace(part.Header.Get("Content-Type")),
					Data:        data,
				})
				continue
			}
			textRefs[name] = append(textRefs[name], strings.TrimSpace(string(data)))
			if _, ok := fields[name]; !ok {
				fields[name] = strings.TrimSpace(string(data))
			}
		}
	}
	info.Model = fields["model"]
	if info.Model == "" {
		return info, fmt.Errorf("model is required")
	}
	info.Prompt = fields["prompt"]
	info.Size = fields["size"]
	info.Resolution = firstNonEmptyTrimmed(fields["resolution"], fields["resolution_name"])
	info.DurationSeconds = parseOpenAIVideoSeconds(firstNonEmptyTrimmed(fields["seconds"], fields["duration"]))
	// 参考视频：上传文件读 mvhd 时长计费；Uploads 保持原样（转发仍用原始 body）。
	refs := collectJSONReferenceVideos(body)
	for _, name := range []string{"input_reference", "video", "video_url", "input_video", "reference_video", "reference_videos"} {
		for _, value := range textRefs[name] {
			if value != "" && (name != "input_reference" || looksLikeVideoReference(value, "", "")) {
				refs = append(refs, ReferenceVideoInput{URL: value})
			}
		}
	}
	for _, upload := range info.Uploads {
		if isReferenceVideoFieldName(upload.FieldName) || isReferenceVideoUpload(upload) {
			refs = append(refs, referenceVideoFromUpload(upload))
		}
	}
	info.ReferenceVideoSeconds = ReferenceVideoSecondsTotal(refs)
	return info, nil
}

func firstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func parseOpenAIVideoSeconds(raw string) int {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || value <= 0 || math.IsInf(value, 0) || math.IsNaN(value) {
		return 0
	}
	return int(math.Ceil(value))
}

// rewriteMultipartFormField 重写 multipart 里指定的文本字段，其余部分（包括文件）按原始字节保留。
func rewriteMultipartFormField(body []byte, contentType, field, value string) ([]byte, error) {
	mediaType, params, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	boundary := strings.TrimSpace(params["boundary"])
	if err != nil || !strings.EqualFold(mediaType, "multipart/form-data") || boundary == "" {
		return nil, fmt.Errorf("request body must be multipart/form-data")
	}
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	var out bytes.Buffer
	writer := multipart.NewWriter(&out)
	if err := writer.SetBoundary(boundary); err != nil {
		return nil, err
	}
	for {
		part, err := reader.NextRawPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("invalid multipart body: %w", err)
		}
		header := make(textproto.MIMEHeader, len(part.Header))
		for key, values := range part.Header {
			header[key] = append([]string(nil), values...)
		}
		dst, err := writer.CreatePart(header)
		if err != nil {
			_ = part.Close()
			return nil, err
		}
		if part.FormName() == field && part.FileName() == "" {
			_, err = io.WriteString(dst, value)
		} else {
			_, err = io.Copy(dst, part)
		}
		_ = part.Close()
		if err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func rewriteOpenAIVideoModel(body []byte, contentType, model string) ([]byte, error) {
	if gjson.ValidBytes(body) {
		return sjson.SetBytes(body, "model", model)
	}
	return rewriteMultipartFormField(body, contentType, "model", model)
}

// buildOpenAIVideoURL 拼上游地址：base 可以是站点根地址或已带 /v1 的地址。
func buildOpenAIVideoURL(base string, endpoint GrokMediaEndpoint, taskID string) (string, error) {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if !strings.HasSuffix(base, "/v1") {
		base += "/v1"
	}
	target := base + "/videos"
	if endpoint == OpenAIVideoEndpointCreate {
		return target, nil
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" || validateUpstreamPathSegment("video task ID", taskID) != nil {
		return "", fmt.Errorf("invalid video task ID")
	}
	target += "/" + url.PathEscape(taskID)
	if endpoint == OpenAIVideoEndpointContent {
		target += "/content"
	}
	return target, nil
}

func openAIVideoResponseID(body []byte) string {
	for _, path := range []string{"id", "task_id", "data.id", "data.task_id"} {
		if value := gjson.GetBytes(body, path); value.Type == gjson.String {
			if id := strings.TrimSpace(value.String()); id != "" {
				return id
			}
		}
	}
	return ""
}

// openAIVideoStatusDone 判断查询结果是否已完成（只有完成的任务才计费）。
func openAIVideoStatusDone(body []byte) bool {
	for _, path := range []string{"status", "data.status", "state"} {
		switch strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, path).String())) {
		case "completed", "succeeded", "success", "done", "finished":
			return true
		}
	}
	return false
}

// ForwardOpenAIVideo 把 /v1/videos 的提交、查询、下载转发到账号上游。
// 上游 4xx 原样返回且不算账号故障；5xx / 429 / 重定向按转发失败处理。
func (s *OpenAIGatewayService) ForwardOpenAIVideo(ctx context.Context, c *gin.Context, account *Account, endpoint GrokMediaEndpoint, taskKey string, body []byte, contentType string) (*OpenAIForwardResult, error) {
	if !endpoint.IsOpenAIVideo() || !account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityVideos) {
		return nil, fmt.Errorf("openai videos require an OpenAI API key account with a custom base URL")
	}
	base, err := s.validateUpstreamBaseURL(account.GetCredential("base_url"))
	if err != nil {
		return nil, err
	}
	target, err := buildOpenAIVideoURL(base, endpoint, openAIVideoUpstreamID(taskKey))
	if err != nil {
		return nil, err
	}
	token := strings.TrimSpace(account.GetCredential("api_key"))
	if token == "" {
		return nil, fmt.Errorf("openai video account missing api_key")
	}

	result := &OpenAIForwardResult{}
	method := http.MethodGet
	var reqBody io.Reader
	if endpoint == OpenAIVideoEndpointCreate {
		info, parseErr := ParseOpenAIVideoRequest(contentType, body)
		if parseErr != nil {
			return nil, &GrokMediaInputValidationError{Reason: parseErr.Error()}
		}
		upstreamModel := info.Model
		if mapped := strings.TrimSpace(account.GetMappedModel(info.Model)); mapped != "" {
			upstreamModel = mapped
		}
		if upstreamModel != info.Model {
			if body, err = rewriteOpenAIVideoModel(body, contentType, upstreamModel); err != nil {
				return nil, &GrokMediaInputValidationError{Reason: err.Error()}
			}
		}
		if strings.TrimSpace(contentType) == "" {
			contentType = "application/json"
		}
		method = http.MethodPost
		reqBody = bytes.NewReader(body)
		result.Model = info.Model
		result.BillingModel = info.Model
		result.UpstreamModel = upstreamModel
		result.VideoDurationSeconds = info.DurationSeconds
		result.VideoResolution = info.Resolution
	}

	if endpoint == OpenAIVideoEndpointContent {
		// 无 /content 接口的上游（straitapi 等）：先查状态取视频地址下载；取不到再回退透传 /content。
		if handled, billable := s.forwardOpenAIVideoResultURL(ctx, c, account, base, token, taskKey); handled {
			result.ResponseID = taskKey
			if billable {
				result.VideoCount = 1
			}
			return result, nil
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, target, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if endpoint == OpenAIVideoEndpointCreate {
		req.Header.Set("Content-Type", contentType)
	}
	if endpoint != OpenAIVideoEndpointContent {
		req.Header.Set("Accept", "application/json")
	}
	account.ApplyHeaderOverrides(req.Header)
	proxy := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxy = account.Proxy.URL()
	}
	started := time.Now()
	resp, err := s.httpUpstream.Do(req, proxy, account.ID, account.Concurrency)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(started).Milliseconds())
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	result.Duration = time.Since(started)
	result.ResponseHeaders = resp.Header.Clone()

	if endpoint == OpenAIVideoEndpointContent && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if err := writeGrokMediaContentResponse(c, resp); err != nil {
			return nil, err
		}
		// 能下载到内容说明任务已完成；与查询共用同一个计费键，只会计费一次。
		result.ResponseID = taskKey
		result.VideoCount = 1
		return result, nil
	}

	responseBody, err := ReadUpstreamResponseBody(resp.Body, s.cfg, c, openAITooLargeError)
	if err != nil {
		return nil, err
	}
	writeGrokMediaResponse(c, resp, responseBody, s.responseHeaderFilter)
	if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests || (resp.StatusCode >= 300 && resp.StatusCode < 400) {
		return nil, fmt.Errorf("openai video upstream status %d", resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return result, nil
	}
	switch endpoint {
	case OpenAIVideoEndpointCreate:
		id := openAIVideoResponseID(responseBody)
		if id == "" {
			return nil, fmt.Errorf("openai video create response missing task ID")
		}
		result.ResponseID = OpenAIVideoTaskKey(id)
	case OpenAIVideoEndpointStatus:
		result.ResponseID = taskKey
		if model := strings.TrimSpace(gjson.GetBytes(responseBody, "model").String()); model != "" {
			result.UpstreamModel = model
		}
		if openAIVideoStatusDone(responseBody) {
			result.VideoCount = 1
		}
	}
	return result, nil
}
