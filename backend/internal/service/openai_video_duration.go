package service

import (
	"context"
	"strings"
)

// /v1/videos（OpenAI 兼容上游，如 straitapi）的时长规则与 Grok 不同：上游允许 1-3600 秒，
// 不能套用 xAI 的 15 秒封顶（会少收、可被套利）；按秒计费的模型提交时必须带时长。
const VideoBillingOpenAIMaxDurationSeconds = 3600

const openAIVideoTaskKeyMarker = "openai-video:"

// IsOpenAIVideoTaskKey 判断任务键是否属于 /v1/videos（见 OpenAIVideoTaskKey）。
func IsOpenAIVideoTaskKey(id string) bool {
	return strings.HasPrefix(strings.TrimPrefix(strings.TrimSpace(id), "grok-video:"), openAIVideoTaskKeyMarker)
}

// NormalizeOpenAIVideoBillingDurationSeconds 只封顶 3600 秒。缺省时长在提交时已被拒，
// 这里的 8 秒兜底只防御异常路径（如快照丢失）。
func NormalizeOpenAIVideoBillingDurationSeconds(d int) int {
	if d <= 0 {
		return VideoBillingDefaultDurationSeconds
	}
	if d > VideoBillingOpenAIMaxDurationSeconds {
		return VideoBillingOpenAIMaxDurationSeconds
	}
	return d
}

// VideoBillingDurationSecondsFor 按任务键选时长规则：/v1/videos 任务走 3600 秒上限，其余沿用 Grok 规则。
func VideoBillingDurationSecondsFor(taskKey string, d int) int {
	if IsOpenAIVideoTaskKey(taskKey) {
		return NormalizeOpenAIVideoBillingDurationSeconds(d)
	}
	return NormalizeVideoBillingDurationSecondsOrDefault(d)
}

// CalculateVideoCostExactSeconds 与 CalculateVideoCost 相同，但不再归一化时长（调用方已按端点规则处理）。
func (s *BillingService) CalculateVideoCostExactSeconds(model, resolution string, videoCount, durationSeconds int, groupConfig *VideoPriceConfig, rateMultiplier float64) *CostBreakdown {
	if rateMultiplier <= 0 {
		return &CostBreakdown{BillingMode: string(BillingModeVideo)}
	}
	if videoCount <= 0 || durationSeconds <= 0 {
		return &CostBreakdown{}
	}
	resolution = NormalizeVideoBillingResolutionOrDefault(resolution)
	total := s.getVideoUnitPrice(model, resolution, groupConfig) * float64(durationSeconds) * float64(videoCount)
	return &CostBreakdown{TotalCost: total, ActualCost: total * rateMultiplier, BillingMode: string(BillingModeVideo)}
}

// VideoModelBilledPerSecond 与 calculateOpenAIVideoCost 的取价顺序一致：分组定价表 → 分组旧视频价 → 渠道定价。
// 未命中任何价格时按按秒处理（保守；未配价格的模型已在提交时被拒）。
func (s *OpenAIGatewayService) VideoModelBilledPerSecond(ctx context.Context, apiKey *APIKey, model, resolution string) bool {
	if s == nil || apiKey == nil || apiKey.Group == nil || model == "" {
		return true
	}
	resolved := s.resolveOpenAIChannelPricing(ctx, model, apiKey)
	if resolved != nil && resolved.Source == PricingSourceGroup && (resolved.Mode == BillingModeVideo || resolved.Mode == BillingModePerRequest) {
		return resolved.Mode == BillingModeVideo
	}
	if apiKeyHasConfiguredVideoPrice(s.apiKeyWithFreshGroupMediaPricing(ctx, apiKey), model, resolution) {
		return true
	}
	if resolved != nil && resolved.Source == PricingSourceChannel {
		switch resolved.Mode {
		case BillingModePerRequest, BillingModeImage:
			return false
		}
	}
	return true
}
