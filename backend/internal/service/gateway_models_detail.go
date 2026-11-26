package service

import "context"

// imageSizeTierDetail 是尺寸档位并集的缓存值。
type imageSizeTierDetail struct {
	tiers []string
	sizes []string
}

// modelsDetailTiersCacheKey 与模型列表缓存分开，避免改变既有缓存的值类型。
func modelsDetailTiersCacheKey(groupID *int64, platform string) string {
	return modelsListCacheKey(groupID, platform) + ":tiers"
}

// GetAvailableModelsDetail 返回分组可用模型与生图尺寸档位并集。
// tiers/sizes 为空表示不做收敛（账号不限，或无法判定）。
func (s *GatewayService) GetAvailableModelsDetail(ctx context.Context, groupID *int64, platform string) (models []string, tiers []string, sizes []string) {
	models = s.GetAvailableModels(ctx, groupID, platform)
	tiers, sizes = s.imageSizeTiersForGroup(ctx, groupID, platform)
	return models, tiers, sizes
}

// imageSizeTiersForGroup 计算分组内「实际能服务图片模型」账号的尺寸档位并集。
//
// 混合分组里纯文本账号既未设置档位、也不服务图片模型，必须排除；
// 否则它们会把分组误判成「不限」，客户端放开大尺寸后请求打到受限账号上报错。
func (s *GatewayService) imageSizeTiersForGroup(ctx context.Context, groupID *int64, platform string) ([]string, []string) {
	cacheKey := modelsDetailTiersCacheKey(groupID, platform)
	if s.modelsListCache != nil {
		if cached, found := s.modelsListCache.Get(cacheKey); found {
			if detail, ok := cached.(imageSizeTierDetail); ok {
				return cloneStringSlice(detail.tiers), cloneStringSlice(detail.sizes)
			}
		}
	}

	accounts, err := s.loadSchedulableAccounts(ctx, groupID, platform)
	if err != nil || len(accounts) == 0 {
		return nil, nil
	}

	union := make(map[string]struct{}, len(openAIImageSizeTierOrder))
	candidates := 0
	for i := range accounts {
		account := &accounts[i]
		tiers := account.OpenAIImageSizeTiers()
		if tiers == nil && !accountServesImageModels(account) {
			continue
		}
		if tiers == nil {
			// 能服务图片模型但未限制档位，视为不限。
			return nil, nil
		}
		candidates++
		for _, tier := range tiers {
			union[tier] = struct{}{}
		}
	}
	if candidates == 0 {
		return nil, nil
	}

	ordered := make([]string, 0, len(union))
	for _, tier := range openAIImageSizeTierOrder {
		if _, ok := union[tier]; ok {
			ordered = append(ordered, tier)
		}
	}
	if len(ordered) == 0 {
		return nil, nil
	}
	sizes := make([]string, 0, len(ordered))
	for _, tier := range ordered {
		if size, ok := openAIImageSizeTestSizes[tier]; ok {
			sizes = append(sizes, size)
		}
	}

	if s.modelsListCache != nil {
		s.modelsListCache.Set(cacheKey, imageSizeTierDetail{
			tiers: cloneStringSlice(ordered),
			sizes: cloneStringSlice(sizes),
		}, s.modelsListCacheTTL)
	}
	return ordered, sizes
}

// accountServesImageModels 判断账号的模型映射里是否含图片模型。
func accountServesImageModels(account *Account) bool {
	for _, model := range batchImageModelsFromAccountMapping(account) {
		if descriptor, ok := DescribeModel(model); ok && descriptor.Capability == ModelCapabilityImage {
			return true
		}
	}
	return false
}
