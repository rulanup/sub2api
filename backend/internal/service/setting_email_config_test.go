package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSettingService_ExportEmailConfig_ExcludesSecretsByDefault(t *testing.T) {
	repo := newResendTestSettingRepo(map[string]string{
		SettingKeyMailProvider:      MailProviderResend,
		SettingKeySMTPHost:          "smtp.example.com",
		SettingKeySMTPPassword:      "smtp-secret",
		SettingKeyResendAPIKey:      "re_secret",
		SettingKeyResendFromEmail:   "noreply@example.com",
		SettingKeyFrontendURL:       "https://example.com",
		SettingKeyDefaultBalance:    "1.5",
	})
	svc := NewSettingService(repo, nil)

	out, err := svc.ExportEmailConfig(context.Background(), false)
	require.NoError(t, err)
	require.NotContains(t, out, SettingKeySMTPPassword, "SMTP 密码默认不得导出")
	require.NotContains(t, out, SettingKeyResendAPIKey, "Resend API Key 默认不得导出")
	require.Contains(t, out, SettingKeyMailProvider)
	require.Contains(t, out, SettingKeySMTPHost)
	require.Contains(t, out, SettingKeyResendFromEmail)
	require.NotContains(t, out, SettingKeyFrontendURL, "非邮件键不得导出")
	require.NotContains(t, out, SettingKeyDefaultBalance, "非邮件键不得导出")
}

func TestSettingService_ExportEmailConfig_IncludeSecrets(t *testing.T) {
	repo := newResendTestSettingRepo(map[string]string{
		SettingKeySMTPPassword: "smtp-secret",
		SettingKeyResendAPIKey: "re_secret",
		SettingKeySMTPFrom:     "noreply@example.com",
	})
	svc := NewSettingService(repo, nil)

	out, err := svc.ExportEmailConfig(context.Background(), true)
	require.NoError(t, err)
	require.Equal(t, "smtp-secret", out[SettingKeySMTPPassword])
	require.Equal(t, "re_secret", out[SettingKeyResendAPIKey])
}

func TestSettingService_ExportEmailConfig_OmitsEmptyValues(t *testing.T) {
	repo := newResendTestSettingRepo(map[string]string{
		SettingKeyMailProvider: "",
		SettingKeySMTPHost:     "smtp.example.com",
	})
	svc := NewSettingService(repo, nil)

	out, err := svc.ExportEmailConfig(context.Background(), true)
	require.NoError(t, err)
	require.NotContains(t, out, SettingKeyMailProvider, "空值键不得导出")
	require.Contains(t, out, SettingKeySMTPHost)
}

func TestSettingService_ImportEmailConfig_FiltersWhitelist(t *testing.T) {
	repo := newResendTestSettingRepo(map[string]string{
		SettingKeySMTPHost: "old.example.com",
	})
	svc := NewSettingService(repo, nil)

	applied, err := svc.ImportEmailConfig(context.Background(), map[string]string{
		SettingKeySMTPHost:       "new.example.com",
		SettingKeySMTPPassword:   "smtp-secret",
		SettingKeyFrontendURL:    "https://evil.example.com",
		SettingKeyDefaultBalance: "999",
	})
	require.NoError(t, err)
	require.Equal(t, 2, applied, "只应应用白名单内的 SMTP 键")

	values, err := repo.GetMultiple(context.Background(), []string{
		SettingKeySMTPHost, SettingKeySMTPPassword, SettingKeyFrontendURL, SettingKeyDefaultBalance,
	})
	require.NoError(t, err)
	require.Equal(t, "new.example.com", values[SettingKeySMTPHost])
	require.Equal(t, "smtp-secret", values[SettingKeySMTPPassword])
	require.NotContains(t, values, SettingKeyFrontendURL)
	require.NotContains(t, values, SettingKeyDefaultBalance)
}

func TestSettingService_ImportEmailConfig_EmptySecretDoesNotOverwrite(t *testing.T) {
	repo := newResendTestSettingRepo(map[string]string{
		SettingKeyResendAPIKey:    "re_existing",
		SettingKeyResendFromEmail: "noreply@example.com",
	})
	svc := NewSettingService(repo, nil)

	applied, err := svc.ImportEmailConfig(context.Background(), map[string]string{
		SettingKeyResendAPIKey:    "",
		SettingKeyResendFromEmail: "new@example.com",
	})
	require.NoError(t, err)
	require.Equal(t, 1, applied, "空的敏感键应被跳过")

	values, err := repo.GetMultiple(context.Background(), []string{
		SettingKeyResendAPIKey, SettingKeyResendFromEmail,
	})
	require.NoError(t, err)
	require.Equal(t, "re_existing", values[SettingKeyResendAPIKey], "现有 API Key 不得被空值清空")
	require.Equal(t, "new@example.com", values[SettingKeyResendFromEmail])
}

func TestSettingService_ImportEmailConfig_InvokesOnUpdate(t *testing.T) {
	repo := newResendTestSettingRepo(map[string]string{
		SettingKeySMTPHost: "old.example.com",
	})
	svc := NewSettingService(repo, nil)
	called := false
	svc.SetOnUpdateCallback(func() { called = true })

	_, err := svc.ImportEmailConfig(context.Background(), map[string]string{
		SettingKeySMTPHost: "new.example.com",
	})
	require.NoError(t, err)
	require.True(t, called, "导入成功后应触发缓存失效回调")
}

func TestSettingService_ImportEmailConfig_EmptyInput(t *testing.T) {
	svc := NewSettingService(newResendTestSettingRepo(map[string]string{}), nil)
	applied, err := svc.ImportEmailConfig(context.Background(), map[string]string{})
	require.NoError(t, err)
	require.Zero(t, applied)

	applied, err = svc.ImportEmailConfig(context.Background(), map[string]string{
		SettingKeyFrontendURL: "https://example.com",
	})
	require.NoError(t, err)
	require.Zero(t, applied, "白名单外键应被忽略")
}
