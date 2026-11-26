package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type detailAccountsRepo struct {
	AccountRepository
	accounts []Account
}

func (r *detailAccountsRepo) ListSchedulableByGroupID(context.Context, int64) ([]Account, error) {
	return r.accounts, nil
}

func newDetailTestService(accounts []Account) (*GatewayService, int64) {
	return &GatewayService{accountRepo: &detailAccountsRepo{accounts: accounts}}, 7
}

func imageAccount(id int64, tiers []string) Account {
	account := Account{
		ID:          id,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"model_mapping": map[string]any{"gpt-image-2": "gpt-image-2"}},
	}
	if tiers != nil {
		values := make([]any, 0, len(tiers))
		for _, tier := range tiers {
			values = append(values, tier)
		}
		account.Extra = map[string]any{AccountExtraOpenAIImageSizeTiers: values}
	}
	return account
}

func textOnlyAccount(id int64) Account {
	return Account{
		ID:          id,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"model_mapping": map[string]any{"gpt-5.6": "gpt-5.6"}},
	}
}

func TestGetAvailableModelsDetailUnionOfTiers(t *testing.T) {
	svc, groupID := newDetailTestService([]Account{
		imageAccount(1, []string{ImageBillingSize1K}),
		imageAccount(2, []string{ImageBillingSize2K, ImageBillingSize4K}),
	})
	models, tiers, sizes := svc.GetAvailableModelsDetail(context.Background(), &groupID, PlatformOpenAI)
	require.Equal(t, []string{ImageBillingSize1K, ImageBillingSize2K, ImageBillingSize4K}, tiers)
	require.Equal(t, []string{"1024x1024", "1536x1024", "3840x2160"}, sizes)
	require.NotEmpty(t, models)
}

func TestGetAvailableModelsDetailUnboundedWhenAnyAccountUnlimited(t *testing.T) {
	svc, groupID := newDetailTestService([]Account{
		imageAccount(1, []string{ImageBillingSize1K}),
		imageAccount(2, nil),
	})
	_, tiers, sizes := svc.GetAvailableModelsDetail(context.Background(), &groupID, PlatformOpenAI)
	require.Nil(t, tiers)
	require.Nil(t, sizes)
}

// TestGetAvailableModelsDetailIgnoresTextOnlyAccounts 覆盖审查重点 #3：
// 混合分组里的纯文本账号未设置档位，不得把分组判定成「不限」。
func TestGetAvailableModelsDetailIgnoresTextOnlyAccounts(t *testing.T) {
	svc, groupID := newDetailTestService([]Account{
		textOnlyAccount(1),
		imageAccount(2, []string{ImageBillingSize1K}),
	})
	_, tiers, sizes := svc.GetAvailableModelsDetail(context.Background(), &groupID, PlatformOpenAI)
	require.Equal(t, []string{ImageBillingSize1K}, tiers)
	require.Equal(t, []string{"1024x1024"}, sizes)
}

// TestGetAvailableModelsDetailEmptyGroup 覆盖审查重点 #4：空分组不得 panic。
func TestGetAvailableModelsDetailEmptyGroup(t *testing.T) {
	svc, groupID := newDetailTestService(nil)
	models, tiers, sizes := svc.GetAvailableModelsDetail(context.Background(), &groupID, PlatformOpenAI)
	require.Nil(t, models)
	require.Nil(t, tiers)
	require.Nil(t, sizes)
}
