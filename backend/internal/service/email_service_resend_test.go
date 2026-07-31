package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type resendTestSettingRepo struct {
	mu     sync.RWMutex
	values map[string]string
}

func newResendTestSettingRepo(values map[string]string) *resendTestSettingRepo {
	store := make(map[string]string, len(values))
	for k, v := range values {
		store[k] = v
	}
	return &resendTestSettingRepo{values: store}
}

func (r *resendTestSettingRepo) Get(_ context.Context, key string) (*Setting, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.values[key]
	if !ok {
		return nil, ErrSettingNotFound
	}
	return &Setting{Key: key, Value: value}, nil
}

func (r *resendTestSettingRepo) GetValue(ctx context.Context, key string) (string, error) {
	setting, err := r.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

func (r *resendTestSettingRepo) Set(_ context.Context, key, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values[key] = value
	return nil
}

func (r *resendTestSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := r.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (r *resendTestSettingRepo) SetMultiple(_ context.Context, settings map[string]string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, value := range settings {
		r.values[key] = value
	}
	return nil
}

func (r *resendTestSettingRepo) GetAll(_ context.Context) (map[string]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]string, len(r.values))
	for key, value := range r.values {
		out[key] = value
	}
	return out, nil
}

func (r *resendTestSettingRepo) Delete(_ context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.values, key)
	return nil
}

func TestEmailService_SendEmailViaResend_Success(t *testing.T) {
	var mu sync.Mutex
	var captured struct {
		auth    string
		from    string
		to      string
		subject string
		html    string
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/emails", r.URL.Path)
		mu.Lock()
		captured.auth = r.Header.Get("Authorization")
		mu.Unlock()

		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		mu.Lock()
		from, fromOK := payload["from"].(string)
		toList, toOK := payload["to"].([]any)
		subject, subjectOK := payload["subject"].(string)
		html, htmlOK := payload["html"].(string)
		to := ""
		if toOK && len(toList) > 0 {
			to, toOK = toList[0].(string)
		}
		captured.from = from
		captured.to = to
		captured.subject = subject
		captured.html = html
		require.True(t, fromOK && toOK && subjectOK && htmlOK)
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"email-id-1"}`))
	}))
	defer server.Close()

	svc := NewEmailService(newResendTestSettingRepo(nil), nil)
	config := &ResendConfig{
		APIKey:  "re_test_key",
		From:    "Sub2API <noreply@example.com>",
		BaseURL: server.URL,
	}

	err := svc.SendEmailViaResend(context.Background(), config, "user@example.com", "Subject Line", "<p>body</p>")
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, "Bearer re_test_key", captured.auth)
	require.Equal(t, "Sub2API <noreply@example.com>", captured.from)
	require.Equal(t, "user@example.com", captured.to)
	require.Equal(t, "Subject Line", captured.subject)
	require.Equal(t, "<p>body</p>", captured.html)
}

func TestEmailService_SendEmailViaResend_FromNameFormatting(t *testing.T) {
	var mu sync.Mutex
	var from string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		mu.Lock()
		from = payload["from"].(string)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"email-id-2"}`))
	}))
	defer server.Close()

	svc := NewEmailService(newResendTestSettingRepo(nil), nil)
	err := svc.SendEmailViaResend(context.Background(), &ResendConfig{
		APIKey:   "re_test_key",
		From:     "noreply@example.com",
		FromName: "Sub2API",
		BaseURL:  server.URL,
	}, "user@example.com", "S", "<p>body</p>")
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, "Sub2API <noreply@example.com>", from)
}

func TestEmailService_SendEmailViaResend_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"name":"validation_error","message":"API key does not exist.","statusCode":401}`))
	}))
	defer server.Close()

	svc := NewEmailService(newResendTestSettingRepo(nil), nil)
	err := svc.SendEmailViaResend(context.Background(), &ResendConfig{
		APIKey:  "re_bad_key",
		From:    "noreply@example.com",
		BaseURL: server.URL,
	}, "user@example.com", "S", "<p>body</p>")
	require.Error(t, err)
	require.Contains(t, err.Error(), "status 401")
	require.Contains(t, err.Error(), "API key does not exist")
}

func TestEmailService_SendEmail_SMTPByDefault(t *testing.T) {
	repo := newResendTestSettingRepo(map[string]string{})
	svc := NewEmailService(repo, nil)

	// 未配置任何邮件通道时，默认走 SMTP 路径并返回未配置错误，
	// 而不是报 Resend 配置缺失。
	err := svc.SendEmail(context.Background(), "user@example.com", "S", "<p>body</p>")
	require.ErrorIs(t, err, ErrEmailNotConfigured)
}

func TestEmailService_SendEmail_ResendWhenConfigured(t *testing.T) {
	var mu sync.Mutex
	var hits []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits = append(hits, r.Method+" "+r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"email-id-3"}`))
	}))
	defer server.Close()

	repo := newResendTestSettingRepo(map[string]string{
		SettingKeyMailProvider:    MailProviderResend,
		SettingKeyResendAPIKey:    "re_test_key",
		SettingKeyResendFromEmail: "noreply@example.com",
		SettingKeyResendBaseURL:   server.URL,
	})
	svc := NewEmailService(repo, nil)

	err := svc.SendEmail(context.Background(), "user@example.com", "Subject", "<p>body</p>")
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []string{"POST /emails"}, hits)
}

func TestEmailService_SendEmail_ResendWithoutAPIKey(t *testing.T) {
	repo := newResendTestSettingRepo(map[string]string{
		SettingKeyMailProvider:    MailProviderResend,
		SettingKeyResendFromEmail: "noreply@example.com",
	})
	svc := NewEmailService(repo, nil)

	err := svc.SendEmail(context.Background(), "user@example.com", "S", "<p>body</p>")
	require.ErrorIs(t, err, ErrEmailNotConfigured)
}

func TestEmailService_TestResendConnectionWithConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/domains", r.URL.Path)
		require.Equal(t, "Bearer re_test_key", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[],"total":0}`))
	}))
	defer server.Close()

	orig := resendAPIBaseURL
	resendAPIBaseURL = server.URL
	defer func() { resendAPIBaseURL = orig }()

	svc := NewEmailService(newResendTestSettingRepo(nil), nil)
	require.NoError(t, svc.TestResendConnectionWithConfig(context.Background(), "re_test_key"))
}

func TestEmailService_TestResendConnectionWithConfig_InvalidKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"name":"unauthorized_error","message":"API key does not exist.","statusCode":401}`))
	}))
	defer server.Close()

	orig := resendAPIBaseURL
	resendAPIBaseURL = server.URL
	defer func() { resendAPIBaseURL = orig }()

	svc := NewEmailService(newResendTestSettingRepo(nil), nil)
	err := svc.TestResendConnectionWithConfig(context.Background(), "re_bad_key")
	require.Error(t, err)
	require.Contains(t, err.Error(), "authentication failed")
}

func TestEmailService_TestResendConnectionWithConfig_EmptyKey(t *testing.T) {
	svc := NewEmailService(newResendTestSettingRepo(nil), nil)
	require.ErrorIs(t, svc.TestResendConnectionWithConfig(context.Background(), ""), ErrEmailNotConfigured)
}

func TestEmailService_GetMailProvider(t *testing.T) {
	svc := NewEmailService(newResendTestSettingRepo(map[string]string{
		SettingKeyMailProvider: MailProviderResend,
	}), nil)
	require.Equal(t, MailProviderResend, svc.GetMailProvider(context.Background()))

	svc = NewEmailService(newResendTestSettingRepo(map[string]string{}), nil)
	require.Equal(t, MailProviderSMTP, svc.GetMailProvider(context.Background()))

	svc = NewEmailService(newResendTestSettingRepo(map[string]string{
		SettingKeyMailProvider: "unknown-provider",
	}), nil)
	require.Equal(t, MailProviderSMTP, svc.GetMailProvider(context.Background()))
}

func TestResendErrorMessage(t *testing.T) {
	msg := resendErrorMessage([]byte(`{"name":"validation_error","message":"from email is required","statusCode":422}`))
	require.Equal(t, "from email is required", msg)

	msg = resendErrorMessage([]byte(`raw non-json body`))
	require.Equal(t, "raw non-json body", msg)

	msg = resendErrorMessage([]byte(``))
	require.Equal(t, "unknown error", msg)
}

func TestEmailService_GetResendConfig_RequiresFrom(t *testing.T) {
	repo := newResendTestSettingRepo(map[string]string{
		SettingKeyResendAPIKey: "re_test_key",
	})
	svc := NewEmailService(repo, nil)
	_, err := svc.GetResendConfig(context.Background())
	require.ErrorIs(t, err, ErrEmailNotConfigured)
}
