package service

import "context"

// referenceVideoSurchargeBase 参考视频加收（未乘倍率）：仅分组定价表 per_request/video 条目配置了单价时生效。
func referenceVideoSurchargeBase(resolved *ResolvedPricing, seconds int) float64 {
	if resolved == nil || resolved.Source != PricingSourceGroup || resolved.channelPricing == nil || seconds <= 0 {
		return 0
	}
	if resolved.Mode != BillingModePerRequest && resolved.Mode != BillingModeVideo {
		return 0
	}
	price := resolved.channelPricing.ReferenceVideoPricePerSecond
	if price == nil || *price <= 0 {
		return 0
	}
	return *price * float64(seconds)
}

func applyReferenceVideoSurcharge(cost *CostBreakdown, resolved *ResolvedPricing, seconds int, multiplier float64) {
	extra := referenceVideoSurchargeBase(resolved, seconds)
	if cost == nil || extra <= 0 || multiplier <= 0 {
		return
	}
	cost.TotalCost += extra
	cost.ActualCost += extra * multiplier
}

// VideoModelPriceConfigured 判断分组是否为视频模型配了价格：分组/渠道定价表的按次或按秒条目，
// 或分组视频价（模型族价 / 三档分辨率价）。内置默认价不算，未配置时提交前拒绝。
func (s *OpenAIGatewayService) VideoModelPriceConfigured(ctx context.Context, apiKey *APIKey, model, resolution string) bool {
	if s == nil || apiKey == nil || apiKey.Group == nil || model == "" {
		return false
	}
	if resolved := s.resolveOpenAIChannelPricing(ctx, model, apiKey); resolved != nil && resolved.channelPricing != nil {
		switch resolved.Mode {
		case BillingModePerRequest, BillingModeVideo, BillingModeImage:
			if resolved.channelPricing.PerRequestPrice != nil || len(resolved.RequestTiers) > 0 {
				return true
			}
		}
	}
	return apiKeyHasConfiguredVideoPrice(s.apiKeyWithFreshGroupMediaPricing(ctx, apiKey), model, resolution)
}

// videoGateCandidateKeys 返回与提交调度一致的候选分组视图：多分组 key 复用 orderedAPIKeyScheduleGroups
// （同平台、CanUseGroup、按策略排序），再只留有效且允许图片生成的分组；每个视图是 apiKey 的浅拷贝。
func (s *OpenAIGatewayService) videoGateCandidateKeys(ctx context.Context, apiKey *APIKey, platform string) []*APIKey {
	if apiKey == nil {
		return nil
	}
	if len(apiKey.GroupIDs) <= 1 {
		return []*APIKey{apiKey}
	}
	var out []*APIKey
	for _, g := range s.orderedAPIKeyScheduleGroups(ctx, apiKey, NormalizeOpenAICompatiblePlatform(platform)) {
		if g == nil || !g.IsActive() || !GroupAllowsImageGeneration(g) {
			continue
		}
		cp := *apiKey
		gid := g.ID
		cp.GroupID, cp.Group = &gid, g
		out = append(out, &cp)
	}
	return out
}

// VideoSubmitPricingCheck 提交前价格检查：任一候选分组为任一模型配了价格即 priced；
// 已配价格的候选分组中任一按秒计费则 perSecond（/v1/videos 须带时长）。选中账号后 handler 会再核一次。
func (s *OpenAIGatewayService) VideoSubmitPricingCheck(ctx context.Context, apiKey *APIKey, platform string, models []string, resolution string) (priced, perSecond bool) {
	for _, k := range s.videoGateCandidateKeys(ctx, apiKey, platform) {
		for _, m := range models {
			if !s.VideoModelPriceConfigured(ctx, k, m, resolution) {
				continue
			}
			priced = true
			if s.VideoModelBilledPerSecond(ctx, k, m, resolution) {
				perSecond = true
			}
			break
		}
	}
	return priced, perSecond
}
