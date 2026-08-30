package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/securitytext"
	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	green "github.com/alibabacloud-go/green-20220302/v3/client"
	"github.com/alibabacloud-go/tea/dara"
	tealegacy "github.com/alibabacloud-go/tea/tea"
)

const aliyunGuardrailsChunkRunes = 2000

var errAliyunGuardrailsImagesNotSupported = errors.New("Aliyun Guardrails moderation does not support image input")

type AliyunGuardrailsClientConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	Region          string
	Endpoint        string
	TimeoutMS       int
	ProxyURL        string
}

type AliyunGuardrailsRequest struct {
	Service string
	Content string
	DataID  string
}

type AliyunGuardrailsResponse struct {
	HTTPStatus int
	Code       int32
	Message    string
	Suggestion string
	Details    []AliyunGuardrailsDetail
}

type AliyunGuardrailsDetail struct {
	Type       string
	Suggestion string
	Level      string
	Labels     []AliyunGuardrailsLabel
}

type AliyunGuardrailsLabel struct {
	Label      string
	Confidence float64
	Level      string
}

type AliyunGuardrailsClient interface {
	MultiModalGuard(ctx context.Context, request AliyunGuardrailsRequest) (*AliyunGuardrailsResponse, error)
}

type AliyunGuardrailsClientFactory interface {
	New(config AliyunGuardrailsClientConfig) (AliyunGuardrailsClient, error)
}

type aliyunGuardrailsSDKFactory struct{}

type aliyunGuardrailsSDKClient struct {
	client  *green.Client
	runtime *dara.RuntimeOptions
}

func NewAliyunGuardrailsClientFactory() AliyunGuardrailsClientFactory {
	return aliyunGuardrailsSDKFactory{}
}

func (aliyunGuardrailsSDKFactory) New(config AliyunGuardrailsClientConfig) (AliyunGuardrailsClient, error) {
	endpoint, err := normalizeAliyunGuardrailsEndpoint(config.Endpoint)
	if err != nil {
		return nil, err
	}
	sdkConfig := &openapiutil.Config{
		AccessKeyId:     dara.String(config.AccessKeyID),
		AccessKeySecret: dara.String(config.AccessKeySecret),
		RegionId:        dara.String(config.Region),
		Endpoint:        dara.String(endpoint),
		Protocol:        dara.String("https"),
		ConnectTimeout:  dara.Int(config.TimeoutMS),
		ReadTimeout:     dara.Int(config.TimeoutMS),
	}
	runtime := (&dara.RuntimeOptions{}).
		SetAutoretry(false).
		SetConnectTimeout(config.TimeoutMS).
		SetReadTimeout(config.TimeoutMS)
	if strings.TrimSpace(config.ProxyURL) == "" {
		// Tea falls back to HTTP(S)_PROXY when proxy fields are empty. Its no_proxy
		// implementation compares exact hosts, so use the configured endpoint host.
		runtime.SetNoProxy(endpoint)
		sdkConfig.NoProxy = dara.String(endpoint)
	} else {
		runtime.SetHttpProxy(config.ProxyURL).SetHttpsProxy(config.ProxyURL)
		sdkConfig.HttpProxy = dara.String(config.ProxyURL)
		sdkConfig.HttpsProxy = dara.String(config.ProxyURL)
	}
	client, err := green.NewClient(sdkConfig)
	if err != nil {
		return nil, fmt.Errorf("create Aliyun Guardrails client: %w", err)
	}
	return &aliyunGuardrailsSDKClient{client: client, runtime: runtime}, nil
}

func normalizeAliyunGuardrailsEndpoint(endpoint string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" && parsed.Path != "/" {
		return "", errors.New("Aliyun Guardrails endpoint must be an HTTPS origin")
	}
	return parsed.Host, nil
}

func (c *aliyunGuardrailsSDKClient) MultiModalGuard(ctx context.Context, request AliyunGuardrailsRequest) (*AliyunGuardrailsResponse, error) {
	params := map[string]string{"content": request.Content}
	if strings.TrimSpace(request.DataID) != "" {
		params["dataId"] = request.DataID
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal Aliyun Guardrails service parameters: %w", err)
	}
	response, err := c.client.MultiModalGuardWithContext(ctx, &green.MultiModalGuardRequest{
		Service:           dara.String(request.Service),
		ServiceParameters: dara.String(string(raw)),
	}, c.runtime)
	if err != nil {
		return nil, classifyAliyunGuardrailsSDKError(err)
	}
	out := &AliyunGuardrailsResponse{}
	if response == nil {
		return nil, newAliyunGuardrailsError(0, 0, "empty SDK response", true)
	}
	out.HTTPStatus = int(dara.Int32Value(response.StatusCode))
	if response.Body == nil {
		return nil, newAliyunGuardrailsError(out.HTTPStatus, 0, "empty response body", out.HTTPStatus >= http.StatusInternalServerError)
	}
	out.Code = dara.Int32Value(response.Body.Code)
	out.Message = dara.StringValue(response.Body.Message)
	if response.Body.Data != nil {
		out.Suggestion = dara.StringValue(response.Body.Data.Suggestion)
		for _, detail := range response.Body.Data.Detail {
			if detail == nil {
				continue
			}
			item := AliyunGuardrailsDetail{
				Type:       dara.StringValue(detail.Type),
				Suggestion: dara.StringValue(detail.Suggestion),
				Level:      dara.StringValue(detail.Level),
			}
			for _, result := range detail.Result {
				if result == nil {
					continue
				}
				item.Labels = append(item.Labels, AliyunGuardrailsLabel{
					Label:      dara.StringValue(result.Label),
					Confidence: float64(dara.Float32Value(result.Confidence)) / 100,
					Level:      dara.StringValue(result.Level),
				})
			}
			out.Details = append(out.Details, item)
		}
	}
	return out, nil
}

type aliyunGuardrailsAPIError struct {
	HTTPStatus int
	Code       int32
	Message    string
	Temporary  bool
}

func (e *aliyunGuardrailsAPIError) Error() string {
	return fmt.Sprintf("Aliyun Guardrails request failed (http_status=%d, code=%d, temporary=%t): %s", e.HTTPStatus, e.Code, e.Temporary, redactContentModerationSecrets(e.Message))
}

func newAliyunGuardrailsError(httpStatus int, code int32, message string, temporary bool) error {
	return &aliyunGuardrailsAPIError{HTTPStatus: httpStatus, Code: code, Message: message, Temporary: temporary}
}

func classifyAliyunGuardrailsSDKError(err error) error {
	status, code, message, ok := aliyunGuardrailsSDKErrorFields(err)
	if !ok {
		return newAliyunGuardrailsError(0, 0, err.Error(), true)
	}
	if message == "" {
		message = code
	}
	return &aliyunGuardrailsAPIError{
		HTTPStatus: status,
		Message:    fmt.Sprintf("%s: %s", code, message),
		Temporary:  status == 0 || status == http.StatusRequestTimeout || status == http.StatusTooManyRequests || status >= http.StatusInternalServerError,
	}
}

func aliyunGuardrailsSDKErrorFields(err error) (status int, code, message string, ok bool) {
	var legacyErr *tealegacy.SDKError
	if errors.As(err, &legacyErr) {
		return tealegacy.IntValue(legacyErr.StatusCode), tealegacy.StringValue(legacyErr.Code), tealegacy.StringValue(legacyErr.Message), true
	}
	var daraErr *dara.SDKError
	if errors.As(err, &daraErr) {
		return dara.IntValue(daraErr.StatusCode), dara.StringValue(daraErr.Code), dara.StringValue(daraErr.Message), true
	}
	return 0, "", "", false
}

func (s *ContentModerationService) callAliyunGuardrailsWithRetry(ctx context.Context, cfg *ContentModerationConfig, input any) (*moderationAPIResult, error) {
	attempts := cfg.RetryCount + 1
	if attempts < 1 {
		attempts = 1
	}
	if attempts > maxContentModerationRetryCount+1 {
		attempts = maxContentModerationRetryCount + 1
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		status := 0
		result, err := s.callAliyunGuardrailsOnce(ctx, cfg, input, &status)
		if err == nil {
			return result, nil
		}
		lastErr = err
		var apiErr *aliyunGuardrailsAPIError
		if !errors.As(err, &apiErr) || !apiErr.Temporary || attempt == attempts-1 {
			break
		}
		wait := time.Duration(100*(attempt+1)) * time.Millisecond
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
	}
	return nil, lastErr
}

func (s *ContentModerationService) callAliyunGuardrailsOnce(ctx context.Context, cfg *ContentModerationConfig, input any, httpStatus *int) (*moderationAPIResult, error) {
	text, err := aliyunGuardrailsTextInput(input)
	if err != nil {
		return nil, err
	}
	proxyURL := ""
	if cfg.ProxyID != nil {
		proxyURL, err = s.resolveModerationProxyURL(ctx, *cfg.ProxyID)
		if err != nil {
			return nil, err
		}
	}
	factory := s.aliyunGuardrailsFactory
	if factory == nil {
		factory = NewAliyunGuardrailsClientFactory()
	}
	client, err := factory.New(AliyunGuardrailsClientConfig{
		AccessKeyID:     cfg.AliyunAccessKeyID,
		AccessKeySecret: cfg.aliyunAccessKeySecretPlaintext,
		Region:          cfg.AliyunRegionID,
		Endpoint:        cfg.BaseURL,
		TimeoutMS:       cfg.TimeoutMS,
		ProxyURL:        proxyURL,
	})
	if err != nil {
		return nil, err
	}

	var aggregate *moderationAPIResult
	for index, chunk := range chunkAliyunGuardrailsText(text, aliyunGuardrailsChunkRunes) {
		response, err := client.MultiModalGuard(ctx, AliyunGuardrailsRequest{
			Service: cfg.AliyunService,
			Content: chunk,
			DataID:  fmt.Sprintf("sub2api-%d", index+1),
		})
		if err != nil {
			var apiErr *aliyunGuardrailsAPIError
			if errors.As(err, &apiErr) && httpStatus != nil {
				*httpStatus = apiErr.HTTPStatus
			}
			return nil, redactAliyunGuardrailsCredentialError(err, cfg)
		}
		if httpStatus != nil {
			*httpStatus = response.HTTPStatus
		}
		if response.HTTPStatus < 200 || response.HTTPStatus >= 300 {
			return nil, newAliyunGuardrailsError(response.HTTPStatus, response.Code, response.Message, response.HTTPStatus == http.StatusRequestTimeout || response.HTTPStatus == http.StatusTooManyRequests || response.HTTPStatus >= http.StatusInternalServerError)
		}
		if response.Code != http.StatusOK {
			return nil, newAliyunGuardrailsError(response.HTTPStatus, response.Code, response.Message, isTemporaryAliyunGuardrailsBodyCode(response.Code))
		}
		chunkResult, err := aliyunGuardrailsResponseResult(response, cfg.ControversialAction)
		if err != nil {
			return nil, err
		}
		aggregate = mergeAliyunGuardrailsResult(aggregate, chunkResult)
	}
	if aggregate == nil {
		return nil, errors.New("Aliyun Guardrails input is empty")
	}
	return aggregate, nil
}

func redactAliyunGuardrailsCredentialError(err error, cfg *ContentModerationConfig) error {
	if err == nil || cfg == nil {
		return err
	}
	message := err.Error()
	for _, secret := range []string{cfg.AliyunAccessKeyID, cfg.aliyunAccessKeySecretPlaintext, cfg.AliyunAccessKeySecret} {
		secret = strings.TrimSpace(secret)
		if secret != "" {
			message = strings.ReplaceAll(message, secret, "[已脱敏]")
		}
	}
	var apiErr *aliyunGuardrailsAPIError
	if errors.As(err, &apiErr) {
		return &aliyunGuardrailsAPIError{HTTPStatus: apiErr.HTTPStatus, Code: apiErr.Code, Message: message, Temporary: apiErr.Temporary}
	}
	return errors.New(redactContentModerationSecrets(message))
}

func isTemporaryAliyunGuardrailsBodyCode(code int32) bool {
	// 阿里云 body code 408 表示权限拒绝，不是 HTTP 请求超时；不可重试。
	return code == 500 || code == 581 || code == 588
}

func aliyunGuardrailsTextInput(input any) (string, error) {
	switch value := input.(type) {
	case string:
		text := securitytext.Canonicalize(value).Text
		if strings.TrimSpace(text) == "" {
			return "", errors.New("Aliyun Guardrails input is empty")
		}
		return text, nil
	case []moderationAPIInputPart:
		var texts []string
		for _, part := range value {
			switch part.Type {
			case "text":
				if text := strings.TrimSpace(securitytext.Canonicalize(part.Text).Text); text != "" {
					texts = append(texts, text)
				}
			case "image_url":
				return "", errAliyunGuardrailsImagesNotSupported
			}
		}
		if len(texts) == 0 {
			return "", errors.New("Aliyun Guardrails input has no text content")
		}
		return strings.Join(texts, "\n"), nil
	case ContentModerationInput:
		if len(value.Images) > 0 {
			return "", errAliyunGuardrailsImagesNotSupported
		}
		return aliyunGuardrailsTextInput(value.Text)
	default:
		return "", fmt.Errorf("Aliyun Guardrails input type %T is not supported", input)
	}
}

func chunkAliyunGuardrailsText(text string, maxRunes int) []string {
	runes := []rune(text)
	if maxRunes <= 0 || len(runes) <= maxRunes {
		return []string{text}
	}
	chunks := make([]string, 0, (len(runes)+maxRunes-1)/maxRunes)
	for start := 0; start < len(runes); start += maxRunes {
		end := start + maxRunes
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
	}
	return chunks
}

var aliyunGuardrailsCategoryMap = map[string]string{
	"violence": "violence", "violent": "violence", "contraband_act": "illicit",
	"sexual": "sexual", "porn": "sexual", "sexual_minors": "sexual/minors",
	"self_harm": "self-harm", "suicide": "self-harm", "hate": "hate",
	"harassment": "harassment", "pii": "pii", "political": "political",
	"copyright": "copyright", "jailbreak": "jailbreak", "prompt_injection": "jailbreak",
	"unethical": "unethical", "illegal": "illicit", "illicit": "illicit",
	"promptattack": "jailbreak", "sensitivedata": "pii",
	"maliciousfile": "illicit", "maliciousurl": "illicit",
}

func normalizeAliyunGuardrailsCategory(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.NewReplacer("-", "_", " ", "_").Replace(value)
	compact := strings.ReplaceAll(value, "_", "")
	if mapped, ok := aliyunGuardrailsCategoryMap[value]; ok {
		return mapped
	}
	if mapped, ok := aliyunGuardrailsCategoryMap[compact]; ok {
		return mapped
	}
	if value == "" {
		return "aliyun/guardrails"
	}
	return "aliyun/" + value
}

func aliyunGuardrailsResponseResult(response *AliyunGuardrailsResponse, controversialAction string) (*moderationAPIResult, error) {
	if response == nil {
		return nil, errors.New("Aliyun Guardrails response is empty")
	}
	suggestion := strings.ToLower(strings.TrimSpace(response.Suggestion))
	if !isValidAliyunGuardrailsSuggestion(suggestion) {
		return nil, fmt.Errorf("Aliyun Guardrails response has invalid suggestion %q", suggestion)
	}
	scores := map[string]float64{}
	categories := []string{}
	for _, detail := range response.Details {
		detailSuggestion := strings.ToLower(strings.TrimSpace(detail.Suggestion))
		if detailSuggestion != "" && !isValidAliyunGuardrailsSuggestion(detailSuggestion) {
			return nil, fmt.Errorf("Aliyun Guardrails detail has invalid suggestion %q", detailSuggestion)
		}
		if aliyunGuardrailsSuggestionRank(detailSuggestion) > aliyunGuardrailsSuggestionRank(suggestion) {
			suggestion = detailSuggestion
		}
		if len(detail.Labels) == 0 && strings.TrimSpace(detail.Type) != "" && detailSuggestion != "pass" {
			category := normalizeAliyunGuardrailsCategory(detail.Type)
			score := aliyunGuardrailsSuggestionScore(detailSuggestion)
			if score > scores[category] {
				scores[category] = score
			}
			categories = appendUniqueString(categories, category)
		}
		for _, label := range detail.Labels {
			categorySource := label.Label
			if strings.TrimSpace(categorySource) == "" {
				categorySource = detail.Type
			}
			category := normalizeAliyunGuardrailsCategory(categorySource)
			score := label.Confidence
			if score <= 0 && suggestion != "pass" {
				score = aliyunGuardrailsSuggestionScore(suggestion)
			}
			if score > scores[category] {
				scores[category] = score
			}
			categories = appendUniqueString(categories, category)
		}
	}
	if len(scores) == 0 && suggestion != "pass" {
		scores["aliyun/guardrails"] = aliyunGuardrailsSuggestionScore(suggestion)
		categories = append(categories, "aliyun/guardrails")
	}
	severity := qwen3GuardSeveritySafe
	flagged := false
	switch suggestion {
	case "block", "mask":
		severity, flagged = qwen3GuardSeverityUnsafe, true
	case "watch":
		severity = qwen3GuardSeverityControversial
		flagged = controversialAction == ContentModerationControversialActionBlock
	}
	return &moderationAPIResult{
		Flagged:             flagged,
		CategoryScores:      scores,
		Severity:            severity,
		Categories:          categories,
		UseProviderDecision: true,
	}, nil
}

func isValidAliyunGuardrailsSuggestion(suggestion string) bool {
	switch strings.ToLower(strings.TrimSpace(suggestion)) {
	case "block", "mask", "watch", "pass":
		return true
	default:
		return false
	}
}

func mergeAliyunGuardrailsResult(current, next *moderationAPIResult) *moderationAPIResult {
	if current == nil {
		return next
	}
	if next == nil {
		return current
	}
	if aliyunGuardrailsSeverityRank(next.Severity) > aliyunGuardrailsSeverityRank(current.Severity) {
		current.Severity = next.Severity
	}
	current.Flagged = current.Flagged || next.Flagged
	current.UseProviderDecision = current.UseProviderDecision || next.UseProviderDecision
	if current.CategoryScores == nil {
		current.CategoryScores = map[string]float64{}
	}
	for category, score := range next.CategoryScores {
		if score > current.CategoryScores[category] {
			current.CategoryScores[category] = score
		}
	}
	for _, category := range next.Categories {
		current.Categories = appendUniqueString(current.Categories, category)
	}
	return current
}

func aliyunGuardrailsSuggestionRank(suggestion string) int {
	switch strings.ToLower(strings.TrimSpace(suggestion)) {
	case "block":
		return 4
	case "mask":
		return 3
	case "watch":
		return 2
	case "pass":
		return 1
	default:
		return 0
	}
}

func aliyunGuardrailsSuggestionScore(suggestion string) float64 {
	switch strings.ToLower(strings.TrimSpace(suggestion)) {
	case "block":
		return 1
	case "mask":
		return 0.9
	case "watch":
		return 0.5
	default:
		return 0
	}
}

func aliyunGuardrailsSeverityRank(severity qwen3GuardSeverity) int {
	switch severity {
	case qwen3GuardSeverityUnsafe:
		return 3
	case qwen3GuardSeverityControversial:
		return 2
	case qwen3GuardSeveritySafe:
		return 1
	default:
		return 0
	}
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
