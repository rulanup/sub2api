package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func videoGateF64(v float64) *float64 { return &v }

func testVideoGateService() *OpenAIGatewayService {
	bs := &BillingService{}
	return &OpenAIGatewayService{billingService: bs, resolver: NewModelPricingResolver(nil, bs)}
}

func videoGateKey(pricing ...ChannelModelPricing) *APIKey {
	gid := int64(7)
	return &APIKey{GroupID: &gid, Group: &Group{ID: gid, Platform: PlatformGrok, RateMultiplier: 1, ModelPricing: pricing}}
}

func TestGroupVideoPricingPerRequestAndPerSecond(t *testing.T) {
	s, ctx := testVideoGateService(), context.Background()
	perReq := videoGateKey(ChannelModelPricing{Models: []string{"vid-a"}, BillingMode: BillingModePerRequest, PerRequestPrice: videoGateF64(0.5), ReferenceVideoPricePerSecond: videoGateF64(0.01)})
	cost := s.calculateOpenAIVideoCost(ctx, "vid-a", perReq, &OpenAIForwardResult{VideoCount: 1, VideoDurationSeconds: 10, ReferenceVideoSeconds: 9}, 2)
	require.InDelta(t, 0.59, cost.TotalCost, 1e-9)
	require.InDelta(t, 1.18, cost.ActualCost, 1e-9)
	require.Equal(t, string(BillingModeVideo), cost.BillingMode)
	perSec := videoGateKey(ChannelModelPricing{Models: []string{"vid-*"}, BillingMode: BillingModeVideo, PerRequestPrice: videoGateF64(0.1)})
	cost = s.calculateOpenAIVideoCost(ctx, "vid-b", perSec, &OpenAIForwardResult{VideoCount: 1, VideoDurationSeconds: 6, ReferenceVideoSeconds: 9}, 2)
	require.InDelta(t, 0.6, cost.TotalCost, 1e-9)
	require.InDelta(t, 1.2, cost.ActualCost, 1e-9)
}

func TestVideoModelPriceConfigured(t *testing.T) {
	s, ctx := testVideoGateService(), context.Background()
	require.True(t, s.VideoModelPriceConfigured(ctx, videoGateKey(ChannelModelPricing{Models: []string{"vid-a"}, BillingMode: BillingModePerRequest, PerRequestPrice: videoGateF64(0.5)}), "vid-a", ""))
	flat := videoGateKey()
	flat.Group.VideoPrice720P = videoGateF64(0.05)
	require.True(t, s.VideoModelPriceConfigured(ctx, flat, "grok-imagine-video", "720p"))
	require.False(t, s.VideoModelPriceConfigured(ctx, videoGateKey(), "vid-a", ""))
	require.False(t, s.VideoModelPriceConfigured(ctx, videoGateKey(), "grok-imagine-video", "720p"), "built-in default price does not count")
}

func TestNormalizeGroupModelPricingReferenceVideoPrice(t *testing.T) {
	_, err := normalizeGroupModelPricing(PlatformGrok, []ChannelModelPricing{{Models: []string{"vid-a"}, BillingMode: BillingModeVideo, PerRequestPrice: videoGateF64(0.1), ReferenceVideoPricePerSecond: videoGateF64(0.02)}})
	require.NoError(t, err)
	for _, bad := range []ChannelModelPricing{
		{Models: []string{"vid-a"}, BillingMode: BillingModeVideo, PerRequestPrice: videoGateF64(0.1), ReferenceVideoPricePerSecond: videoGateF64(-1)},
		{Models: []string{"m"}, BillingMode: BillingModeToken, InputPrice: videoGateF64(1), ReferenceVideoPricePerSecond: videoGateF64(0.02)},
	} {
		_, err := normalizeGroupModelPricing(PlatformGrok, []ChannelModelPricing{bad})
		require.Error(t, err)
	}
}

func videoGateGroup(id int64) *Group {
	return &Group{ID: id, Platform: PlatformOpenAI, Status: StatusActive, RateMultiplier: 1, AllowImageGeneration: true}
}

func videoGateMultiKey(groups ...*Group) *APIKey {
	ids := make([]int64, 0, len(groups))
	for _, g := range groups {
		ids = append(ids, g.ID)
	}
	gid := groups[0].ID
	return &APIKey{GroupID: &gid, Group: groups[0], GroupIDs: ids, Groups: groups, GroupScheduleStrategy: APIKeyGroupScheduleCheapest}
}

func TestVideoSubmitPricingCheckMultiGroup(t *testing.T) {
	s, ctx := testVideoGateService(), context.Background()
	models := []string{"vid-a"}
	g2, g4 := videoGateGroup(2), videoGateGroup(4)
	g4.VideoPrice480P = videoGateF64(0.01)
	key := videoGateMultiKey(g2, g4)
	priced, perSecond := s.VideoSubmitPricingCheck(ctx, key, PlatformOpenAI, models, "")
	require.True(t, priced, "price on non-primary group")
	require.True(t, perSecond)
	require.Equal(t, int64(2), *key.GroupID, "check must not mutate the key")
	require.Same(t, g2, key.Group)
	perReq := videoGateGroup(4)
	perReq.ModelPricing = []ChannelModelPricing{{Models: models, BillingMode: BillingModePerRequest, PerRequestPrice: videoGateF64(0.5)}}
	priced, perSecond = s.VideoSubmitPricingCheck(ctx, videoGateMultiKey(videoGateGroup(2), perReq), PlatformOpenAI, models, "")
	require.True(t, priced)
	require.False(t, perSecond, "per-request group needs no duration")
	priced, _ = s.VideoSubmitPricingCheck(ctx, videoGateMultiKey(videoGateGroup(2), videoGateGroup(4)), PlatformOpenAI, models, "")
	require.False(t, priced, "no candidate priced")
	noImg := videoGateGroup(4)
	noImg.VideoPrice480P = videoGateF64(0.01)
	noImg.AllowImageGeneration = false
	priced, _ = s.VideoSubmitPricingCheck(ctx, videoGateMultiKey(videoGateGroup(2), noImg), PlatformOpenAI, models, "")
	require.False(t, priced, "group without image generation is not a candidate")
	priced, _ = s.VideoSubmitPricingCheck(ctx, videoGateKey(), PlatformOpenAI, models, "")
	require.False(t, priced, "single-group key unchanged")
}
