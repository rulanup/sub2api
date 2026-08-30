package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/alibabacloud-go/tea/dara"
	tealegacy "github.com/alibabacloud-go/tea/tea"
	"github.com/stretchr/testify/require"
)

type aliyunGuardrailsTestEncryptor struct{}

func (aliyunGuardrailsTestEncryptor) Encrypt(plaintext string) (string, error) {
	return "enc:" + plaintext, nil
}

func (aliyunGuardrailsTestEncryptor) Decrypt(ciphertext string) (string, error) {
	value, ok := strings.CutPrefix(ciphertext, "enc:")
	if !ok {
		return "", errors.New("invalid ciphertext")
	}
	return value, nil
}

type aliyunGuardrailsFailingDecryptor struct{}

func (aliyunGuardrailsFailingDecryptor) Encrypt(plaintext string) (string, error) {
	return "enc:" + plaintext, nil
}

func (aliyunGuardrailsFailingDecryptor) Decrypt(string) (string, error) {
	return "", errors.New("cannot decrypt")
}

type fakeAliyunGuardrailsFactory struct {
	mu      sync.Mutex
	configs []AliyunGuardrailsClientConfig
	client  *fakeAliyunGuardrailsClient
	err     error
}

func (f *fakeAliyunGuardrailsFactory) New(config AliyunGuardrailsClientConfig) (AliyunGuardrailsClient, error) {
	f.mu.Lock()
	f.configs = append(f.configs, config)
	f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	return f.client, nil
}

type fakeAliyunGuardrailsClient struct {
	mu        sync.Mutex
	requests  []AliyunGuardrailsRequest
	responses []*AliyunGuardrailsResponse
	errors    []error
}

func (c *fakeAliyunGuardrailsClient) MultiModalGuard(_ context.Context, request AliyunGuardrailsRequest) (*AliyunGuardrailsResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	index := len(c.requests)
	c.requests = append(c.requests, request)
	if index < len(c.errors) && c.errors[index] != nil {
		return nil, c.errors[index]
	}
	if index < len(c.responses) {
		return c.responses[index], nil
	}
	return aliyunGuardrailsResponse("pass"), nil
}

func aliyunGuardrailsResponse(suggestion string, labels ...AliyunGuardrailsLabel) *AliyunGuardrailsResponse {
	return &AliyunGuardrailsResponse{
		HTTPStatus: 200,
		Code:       200,
		Message:    "OK",
		Suggestion: suggestion,
		Details: []AliyunGuardrailsDetail{{
			Suggestion: suggestion,
			Labels:     labels,
		}},
	}
}

func newAliyunGuardrailsTestService(repo *contentModerationTestSettingRepo, factory AliyunGuardrailsClientFactory, keyConfigured bool) *ContentModerationService {
	return newContentModerationServiceWithAliyun(repo, nil, nil, nil, nil, nil, nil, nil, aliyunGuardrailsTestEncryptor{}, keyConfigured, factory)
}

func TestAliyunGuardrailsDefaultsAndLegacyConfig(t *testing.T) {
	cfg, err := parseContentModerationConfig(`{"enabled":true,"protocol":"openai_moderation","api_keys":["legacy-key"]}`)
	require.NoError(t, err)
	require.Equal(t, ContentModerationProtocolOpenAIModeration, cfg.Protocol)
	require.Equal(t, []string{"legacy-key"}, cfg.apiKeys())
	require.Equal(t, defaultAliyunGuardrailsRegion, cfg.AliyunRegionID)
	require.Equal(t, defaultAliyunGuardrailsService, cfg.AliyunService)
	require.Equal(t, defaultContentModerationBaseURL, cfg.BaseURL)
	require.False(t, cfg.hasModerationCredentials() && cfg.Protocol == ContentModerationProtocolAliyunGuardrails)

	aliyun, err := parseContentModerationConfig(`{"protocol":"aliyun_guardrails"}`)
	require.NoError(t, err)
	require.Equal(t, defaultAliyunGuardrailsEndpoint, aliyun.BaseURL)
}

func TestAliyunGuardrailsConfigEncryptRetainReplaceClearAndProtocolCredentials(t *testing.T) {
	repo := &contentModerationTestSettingRepo{values: map[string]string{}}
	svc := newAliyunGuardrailsTestService(repo, &fakeAliyunGuardrailsFactory{client: &fakeAliyunGuardrailsClient{}}, true)
	protocol := ContentModerationProtocolAliyunGuardrails
	accessKeyID := "LTAI-test-id"
	secretOne := "secret-one"
	view, err := svc.UpdateConfig(context.Background(), UpdateContentModerationConfigInput{
		Protocol:              &protocol,
		AliyunAccessKeyID:     &accessKeyID,
		AliyunAccessKeySecret: &secretOne,
	})
	require.NoError(t, err)
	require.True(t, view.AliyunAccessKeyConfigured)
	require.Equal(t, defaultAliyunGuardrailsEndpoint, view.BaseURL)
	viewJSON := mustAliyunGuardrailsJSON(t, view)
	require.NotContains(t, viewJSON, accessKeyID)
	require.NotContains(t, viewJSON, secretOne)
	require.NotContains(t, viewJSON, "aliyun_access_key_secret")
	require.NotContains(t, viewJSON, "aliyun_access_key_id_configured")
	require.Contains(t, viewJSON, `"aliyun_access_key_configured":true`)
	require.Contains(t, viewJSON, `"aliyun_access_key_id_masked":`)

	var stored ContentModerationConfig
	require.NoError(t, json.Unmarshal([]byte(repo.values[SettingKeyContentModerationConfig]), &stored))
	require.Equal(t, accessKeyID, stored.AliyunAccessKeyID)
	require.Equal(t, "enc:"+secretOne, stored.AliyunAccessKeySecret)
	require.NotContains(t, repo.values[SettingKeyContentModerationConfig], `"aliyun_access_key_secret":"`)

	view, err = svc.UpdateConfig(context.Background(), UpdateContentModerationConfigInput{})
	require.NoError(t, err)
	require.True(t, view.AliyunAccessKeyConfigured)

	blank := "  "
	view, err = svc.UpdateConfig(context.Background(), UpdateContentModerationConfigInput{AliyunAccessKeyID: &blank, AliyunAccessKeySecret: &blank})
	require.NoError(t, err)
	require.True(t, view.AliyunAccessKeyConfigured)

	secretTwo := "secret-two"
	_, err = svc.UpdateConfig(context.Background(), UpdateContentModerationConfigInput{AliyunAccessKeySecret: &secretTwo})
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal([]byte(repo.values[SettingKeyContentModerationConfig]), &stored))
	require.Equal(t, "enc:"+secretTwo, stored.AliyunAccessKeySecret)

	openAIProtocol := ContentModerationProtocolOpenAIModeration
	apiKeys := []string{"sk-openai"}
	_, err = svc.UpdateConfig(context.Background(), UpdateContentModerationConfigInput{Protocol: &openAIProtocol, APIKeys: &apiKeys, APIKeysMode: contentModerationAPIKeysModeReplace})
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal([]byte(repo.values[SettingKeyContentModerationConfig]), &stored))
	require.Equal(t, accessKeyID, stored.AliyunAccessKeyID)
	require.Equal(t, "enc:"+secretTwo, stored.AliyunAccessKeySecret)
	require.Equal(t, []string{"sk-openai"}, stored.APIKeys)

	_, err = svc.UpdateConfig(context.Background(), UpdateContentModerationConfigInput{ClearAliyunCredentials: true})
	require.NoError(t, err)
	stored = ContentModerationConfig{}
	require.NoError(t, json.Unmarshal([]byte(repo.values[SettingKeyContentModerationConfig]), &stored))
	require.Empty(t, stored.AliyunAccessKeyID)
	require.Empty(t, stored.AliyunAccessKeySecret)
	require.Equal(t, []string{"sk-openai"}, stored.APIKeys)
}

func TestAliyunGuardrailsCanClearUndecryptableCredentials(t *testing.T) {
	cfg := defaultContentModerationConfig()
	cfg.Protocol = ContentModerationProtocolAliyunGuardrails
	cfg.AliyunAccessKeyID = "saved-id"
	cfg.AliyunAccessKeySecret = "broken-ciphertext"
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)
	repo := &contentModerationTestSettingRepo{values: map[string]string{SettingKeyContentModerationConfig: string(raw)}}
	svc := newContentModerationServiceWithAliyun(repo, nil, nil, nil, nil, nil, nil, nil, aliyunGuardrailsFailingDecryptor{}, true, &fakeAliyunGuardrailsFactory{})

	view, err := svc.GetConfig(context.Background())
	require.NoError(t, err)
	require.True(t, view.AliyunAccessKeyConfigured)

	view, err = svc.UpdateConfig(context.Background(), UpdateContentModerationConfigInput{ClearAliyunCredentials: true})
	require.NoError(t, err)
	require.False(t, view.AliyunAccessKeyConfigured)
}

func TestAliyunGuardrailsProtocolAndRegionUpdateDefaultEndpoints(t *testing.T) {
	repo := &contentModerationTestSettingRepo{values: map[string]string{}}
	svc := newAliyunGuardrailsTestService(repo, &fakeAliyunGuardrailsFactory{}, true)
	aliyunProtocol := ContentModerationProtocolAliyunGuardrails
	view, err := svc.UpdateConfig(context.Background(), UpdateContentModerationConfigInput{Protocol: &aliyunProtocol})
	require.NoError(t, err)
	require.Equal(t, defaultAliyunGuardrailsEndpoint, view.BaseURL)

	beijing := "cn-beijing"
	view, err = svc.UpdateConfig(context.Background(), UpdateContentModerationConfigInput{AliyunRegionID: &beijing})
	require.NoError(t, err)
	require.Equal(t, "https://green-cip.cn-beijing.aliyuncs.com", view.BaseURL)

	openAIProtocol := ContentModerationProtocolOpenAIModeration
	view, err = svc.UpdateConfig(context.Background(), UpdateContentModerationConfigInput{Protocol: &openAIProtocol})
	require.NoError(t, err)
	require.Equal(t, defaultContentModerationBaseURL, view.BaseURL)
}

func TestAliyunGuardrailsRejectsNewSecretWithoutStableKey(t *testing.T) {
	repo := &contentModerationTestSettingRepo{values: map[string]string{}}
	svc := newAliyunGuardrailsTestService(repo, nil, false)
	secret := "new-secret"
	_, err := svc.UpdateConfig(context.Background(), UpdateContentModerationConfigInput{AliyunAccessKeySecret: &secret})
	require.ErrorIs(t, err, ErrSecretEncryptionKeyNotConfigured)
	require.Empty(t, repo.values[SettingKeyContentModerationConfig])
}

func TestAliyunGuardrailsChunksAndAggregatesSuggestions(t *testing.T) {
	client := &fakeAliyunGuardrailsClient{responses: []*AliyunGuardrailsResponse{
		aliyunGuardrailsResponse("pass"),
		aliyunGuardrailsResponse("watch", AliyunGuardrailsLabel{Label: "political", Confidence: 0.60}),
		aliyunGuardrailsResponse("mask", AliyunGuardrailsLabel{Label: "pii", Confidence: 0.90}),
		aliyunGuardrailsResponse("block", AliyunGuardrailsLabel{Label: "contraband_act", Confidence: 0.99}),
	}}
	factory := &fakeAliyunGuardrailsFactory{client: client}
	svc := newAliyunGuardrailsTestService(nil, factory, true)
	cfg := defaultContentModerationConfig()
	cfg.Protocol = ContentModerationProtocolAliyunGuardrails
	cfg.AliyunAccessKeyID = "ak-id"
	cfg.aliyunAccessKeySecretPlaintext = "ak-secret"
	cfg.ControversialAction = ContentModerationControversialActionAllow

	input := strings.Repeat("界", aliyunGuardrailsChunkRunes*3+1)
	result, err := svc.callAliyunGuardrailsOnce(context.Background(), cfg, input, nil)
	require.NoError(t, err)
	require.Len(t, client.requests, 4)
	for _, request := range client.requests[:3] {
		require.Len(t, []rune(request.Content), aliyunGuardrailsChunkRunes)
		require.Equal(t, defaultAliyunGuardrailsService, request.Service)
		require.NotEmpty(t, request.DataID)
	}
	require.Len(t, []rune(client.requests[3].Content), 1)
	require.True(t, result.Flagged)
	require.Equal(t, qwen3GuardSeverityUnsafe, result.Severity)
	require.Equal(t, 0.99, result.CategoryScores["illicit"])
	require.Equal(t, 0.9, result.CategoryScores["pii"])
	require.Contains(t, result.Categories, "political")
}

func TestAliyunGuardrailsWatchRespectsControversialAction(t *testing.T) {
	response := aliyunGuardrailsResponse("watch", AliyunGuardrailsLabel{Label: "political", Confidence: 0.50})
	allow, err := aliyunGuardrailsResponseResult(response, ContentModerationControversialActionAllow)
	require.NoError(t, err)
	block, err := aliyunGuardrailsResponseResult(response, ContentModerationControversialActionBlock)
	require.NoError(t, err)
	require.False(t, allow.Flagged)
	require.True(t, block.Flagged)
	require.Equal(t, qwen3GuardSeverityControversial, allow.Severity)
}

func TestAliyunGuardrailsRejectsUnknownSuggestionAndMapsDetailType(t *testing.T) {
	_, err := aliyunGuardrailsResponseResult(aliyunGuardrailsResponse("unknown"), ContentModerationControversialActionAllow)
	require.Error(t, err)

	response := aliyunGuardrailsResponse("block")
	response.Details = []AliyunGuardrailsDetail{{Type: "promptAttack", Suggestion: "block"}}
	result, err := aliyunGuardrailsResponseResult(response, ContentModerationControversialActionAllow)
	require.NoError(t, err)
	require.True(t, result.Flagged)
	require.Equal(t, 1.0, result.CategoryScores["jailbreak"])
}

func TestAliyunGuardrailsRetryTemporaryOnly(t *testing.T) {
	cfg := defaultContentModerationConfig()
	cfg.Protocol = ContentModerationProtocolAliyunGuardrails
	cfg.AliyunAccessKeyID = "ak-id"
	cfg.aliyunAccessKeySecretPlaintext = "ak-secret"
	cfg.RetryCount = 2

	temporaryClient := &fakeAliyunGuardrailsClient{
		errors:    []error{newAliyunGuardrailsError(500, 500, "temporary secret=must-not-leak", true)},
		responses: []*AliyunGuardrailsResponse{nil, aliyunGuardrailsResponse("pass")},
	}
	svc := newAliyunGuardrailsTestService(nil, &fakeAliyunGuardrailsFactory{client: temporaryClient}, true)
	result, err := svc.callAliyunGuardrailsWithRetry(context.Background(), cfg, "hello")
	require.NoError(t, err)
	require.False(t, result.Flagged)
	require.Len(t, temporaryClient.requests, 2)

	permanentClient := &fakeAliyunGuardrailsClient{errors: []error{newAliyunGuardrailsError(403, 403, "forbidden", false)}}
	svc = newAliyunGuardrailsTestService(nil, &fakeAliyunGuardrailsFactory{client: permanentClient}, true)
	_, err = svc.callAliyunGuardrailsWithRetry(context.Background(), cfg, "hello")
	require.Error(t, err)
	require.Len(t, permanentClient.requests, 1)
}

func TestAliyunGuardrailsFactoryConfiguresDirectAndExplicitProxy(t *testing.T) {
	factory := aliyunGuardrailsSDKFactory{}
	direct, err := factory.New(AliyunGuardrailsClientConfig{
		AccessKeyID: "id", AccessKeySecret: "secret", Region: defaultAliyunGuardrailsRegion,
		Endpoint: defaultAliyunGuardrailsEndpoint, TimeoutMS: 1234,
	})
	require.NoError(t, err)
	directSDK := direct.(*aliyunGuardrailsSDKClient)
	require.Equal(t, "green-cip.cn-shanghai.aliyuncs.com", *directSDK.runtime.NoProxy)
	require.Nil(t, directSDK.runtime.HttpsProxy)
	require.Equal(t, 1234, *directSDK.runtime.ReadTimeout)

	proxied, err := factory.New(AliyunGuardrailsClientConfig{
		AccessKeyID: "id", AccessKeySecret: "secret", Region: defaultAliyunGuardrailsRegion,
		Endpoint: defaultAliyunGuardrailsEndpoint, TimeoutMS: 1234, ProxyURL: "http://127.0.0.1:8080",
	})
	require.NoError(t, err)
	proxySDK := proxied.(*aliyunGuardrailsSDKClient)
	require.Equal(t, "http://127.0.0.1:8080", *proxySDK.runtime.HttpsProxy)
}

func TestAliyunGuardrailsClassifiesRealSDKErrorTypes(t *testing.T) {
	legacy := tealegacy.NewSDKError(map[string]any{"statusCode": 403, "code": "NoPermission", "message": "denied"})
	legacyClassified := classifyAliyunGuardrailsSDKError(legacy)
	var legacyAPIError *aliyunGuardrailsAPIError
	require.ErrorAs(t, legacyClassified, &legacyAPIError)
	require.Equal(t, 403, legacyAPIError.HTTPStatus)
	require.False(t, legacyAPIError.Temporary)

	modern := dara.NewSDKError(map[string]any{"statusCode": 588, "code": "EXCEED_QUOTA", "message": "busy"})
	modernClassified := classifyAliyunGuardrailsSDKError(modern)
	var modernAPIError *aliyunGuardrailsAPIError
	require.ErrorAs(t, modernClassified, &modernAPIError)
	require.Equal(t, 588, modernAPIError.HTTPStatus)
	require.True(t, modernAPIError.Temporary)
}

func TestAliyunGuardrailsBodyCodeClassification(t *testing.T) {
	require.False(t, isTemporaryAliyunGuardrailsBodyCode(408), "Aliyun body code 408 means permission denied")
	require.True(t, isTemporaryAliyunGuardrailsBodyCode(500))
	require.True(t, isTemporaryAliyunGuardrailsBodyCode(581))
	require.True(t, isTemporaryAliyunGuardrailsBodyCode(588))

	cfg := defaultContentModerationConfig()
	cfg.Protocol = ContentModerationProtocolAliyunGuardrails
	cfg.BaseURL = defaultAliyunGuardrailsEndpoint
	cfg.AliyunAccessKeyID = "ak-id"
	cfg.aliyunAccessKeySecretPlaintext = "ak-secret"
	cfg.RetryCount = 1

	temporaryClient := &fakeAliyunGuardrailsClient{responses: []*AliyunGuardrailsResponse{
		{HTTPStatus: 200, Code: 500, Message: "busy"},
		aliyunGuardrailsResponse("pass"),
	}}
	svc := newAliyunGuardrailsTestService(nil, &fakeAliyunGuardrailsFactory{client: temporaryClient}, true)
	_, err := svc.callAliyunGuardrailsWithRetry(context.Background(), cfg, "hello")
	require.NoError(t, err)
	require.Len(t, temporaryClient.requests, 2)

	permanentClient := &fakeAliyunGuardrailsClient{responses: []*AliyunGuardrailsResponse{{HTTPStatus: 200, Code: 400, Message: "invalid"}}}
	svc = newAliyunGuardrailsTestService(nil, &fakeAliyunGuardrailsFactory{client: permanentClient}, true)
	_, err = svc.callAliyunGuardrailsWithRetry(context.Background(), cfg, "hello")
	require.Error(t, err)
	require.Len(t, permanentClient.requests, 1)
}

func TestAliyunGuardrailsTestConnectionUsesSavedSecretAndRejectsImages(t *testing.T) {
	cfg := defaultContentModerationConfig()
	cfg.Protocol = ContentModerationProtocolAliyunGuardrails
	cfg.AliyunAccessKeyID = "saved-id"
	cfg.AliyunAccessKeySecret = "enc:saved-secret"
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)
	repo := &contentModerationTestSettingRepo{values: map[string]string{SettingKeyContentModerationConfig: string(raw)}}
	client := &fakeAliyunGuardrailsClient{responses: []*AliyunGuardrailsResponse{aliyunGuardrailsResponse("pass")}}
	factory := &fakeAliyunGuardrailsFactory{client: client}
	svc := newAliyunGuardrailsTestService(repo, factory, true)

	result, err := svc.TestAPIKeys(context.Background(), TestContentModerationAPIKeysInput{Protocol: ContentModerationProtocolAliyunGuardrails, Prompt: "hello"})
	require.NoError(t, err)
	require.True(t, result.Items[0].Configured)
	require.Len(t, result.Items, 1)
	require.Equal(t, "ok", result.Items[0].Status)
	require.Equal(t, "saved-secret", factory.configs[0].AccessKeySecret)

	_, err = svc.TestAPIKeys(context.Background(), TestContentModerationAPIKeysInput{Protocol: ContentModerationProtocolAliyunGuardrails, Images: []string{"data:image/png;base64,AA=="}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "不支持图片")
}

func TestAliyunGuardrailsImageFailOpenAlwaysRecordsError(t *testing.T) {
	repo := &contentModerationTestRepo{}
	svc := newContentModerationServiceWithAliyun(nil, nil, nil, nil, nil, nil, nil, nil, aliyunGuardrailsTestEncryptor{}, true, &fakeAliyunGuardrailsFactory{})
	svc.repo = repo
	cfg := defaultContentModerationConfig()
	cfg.Enabled = true
	cfg.Protocol = ContentModerationProtocolAliyunGuardrails
	cfg.RecordNonHits = false
	cfg.AliyunAccessKeyID = "saved-id"
	cfg.aliyunAccessKeySecretPlaintext = "saved-secret"
	content := ContentModerationInput{Text: "hello", Images: []string{"data:image/png;base64,AA=="}}

	decision := svc.checkSync(context.Background(), ContentModerationCheckInput{RequestID: "image-fail-open"}, cfg, content, content.Hash(), nil, true)
	require.True(t, decision.Allowed)
	require.False(t, decision.Blocked)
	require.Len(t, repo.logs, 1)
	require.Equal(t, ContentModerationActionError, repo.logs[0].Action)
	require.Contains(t, repo.logs[0].Error, "does not support image input")
}

func mustAliyunGuardrailsJSON(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	return string(raw)
}
