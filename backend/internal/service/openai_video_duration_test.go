package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVideoBillingDurationSecondsFor(t *testing.T) {
	key := OpenAIVideoTaskKey("t1")
	require.True(t, IsOpenAIVideoTaskKey(key))
	require.False(t, IsOpenAIVideoTaskKey("req-1"))
	require.Equal(t, 60, VideoBillingDurationSecondsFor(key, 60))
	require.Equal(t, 3600, VideoBillingDurationSecondsFor(key, 3601))
	require.Equal(t, 15, VideoBillingDurationSecondsFor("req-1", 60), "grok keeps 15s cap")
	require.Equal(t, 8, VideoBillingDurationSecondsFor("req-1", 0))
}

func TestCalculateVideoCostExactSeconds(t *testing.T) {
	cost := (&BillingService{}).CalculateVideoCostExactSeconds("m", "720p", 1, 60, &VideoPriceConfig{Price720P: videoGateF64(0.1)}, 2)
	require.InDelta(t, 6.0, cost.TotalCost, 1e-9)
	require.InDelta(t, 12.0, cost.ActualCost, 1e-9)
}

func TestVideoModelBilledPerSecond(t *testing.T) {
	s, ctx := testVideoGateService(), context.Background()
	perReq := videoGateKey(ChannelModelPricing{Models: []string{"vid-a"}, BillingMode: BillingModePerRequest, PerRequestPrice: videoGateF64(0.5)})
	require.False(t, s.VideoModelBilledPerSecond(ctx, perReq, "vid-a", ""))
	perSec := videoGateKey(ChannelModelPricing{Models: []string{"vid-a"}, BillingMode: BillingModeVideo, PerRequestPrice: videoGateF64(0.1)})
	require.True(t, s.VideoModelBilledPerSecond(ctx, perSec, "vid-a", ""))
	flat := videoGateKey()
	flat.Group.VideoPrice480P = videoGateF64(0.05)
	require.True(t, s.VideoModelBilledPerSecond(ctx, flat, "vid-b", ""))
}

func TestOpenAIVideoCostUsesUncappedSeconds(t *testing.T) {
	s, ctx := testVideoGateService(), context.Background()
	key := videoGateKey()
	key.Group.VideoPrice480P = videoGateF64(0.1)
	require.True(t, IsOpenAIVideoTaskKey(StableGrokVideoBillingRequestID(OpenAIVideoTaskKey("t1"))))
	cost := s.calculateOpenAIVideoCost(ctx, "m", key, &OpenAIForwardResult{ResponseID: OpenAIVideoTaskKey("t1"), VideoCount: 1, VideoDurationSeconds: 60}, 1)
	require.InDelta(t, 6.0, cost.TotalCost, 1e-9)
	cost = s.calculateOpenAIVideoCost(ctx, "m", key, &OpenAIForwardResult{RequestID: StableGrokVideoBillingRequestID(OpenAIVideoTaskKey("t2")), VideoCount: 1, VideoDurationSeconds: 60}, 1)
	require.InDelta(t, 6.0, cost.TotalCost, 1e-9)
	cost = s.calculateOpenAIVideoCost(ctx, "m", key, &OpenAIForwardResult{ResponseID: "req-1", VideoCount: 1, VideoDurationSeconds: 60}, 1)
	require.InDelta(t, 1.5, cost.TotalCost, 1e-9, "grok keeps 15s cap")
}
