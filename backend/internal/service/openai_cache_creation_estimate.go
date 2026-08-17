package service

import (
	"encoding/json"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
)

// BackfillOpenAICacheCreationFromAnthropicBody estimates cache creation only
// when the upstream does not report it. OpenAI input_tokens is an inclusive
// total, so the estimate is capped to the non-cache-read portion and is never
// added to InputTokens.
func BackfillOpenAICacheCreationFromAnthropicBody(body []byte, model string, usage *OpenAIUsage) {
	if usage == nil || usage.CacheCreationReported || usage.InputTokens <= 0 {
		return
	}
	estimated, err := estimateAnthropicCacheCreationTokens(body, model)
	if err != nil || estimated <= 0 {
		return
	}
	available := max(usage.InputTokens-usage.CacheReadInputTokens, 0)
	usage.CacheCreationInputTokens = min(estimated, available)
}

func estimateAnthropicCacheCreationTokens(body []byte, model string) (int, error) {
	var req apicompat.AnthropicRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return 0, err
	}
	if strings.TrimSpace(model) == "" {
		model = req.Model
	}
	codec, err := openAIInputTokensCodecForModel(model)
	if err != nil {
		return 0, err
	}

	parts := make([]string, 0, len(req.Tools)+len(req.Messages)+4)
	lastBreakpoint := -1
	appendPart := func(value string, cacheControl *apicompat.AnthropicCacheControl) {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			parts = append(parts, trimmed)
			if isEphemeralCacheControl(cacheControl) {
				lastBreakpoint = len(parts) - 1
			}
		}
	}

	for _, tool := range req.Tools {
		raw, marshalErr := json.Marshal(tool)
		if marshalErr != nil {
			return 0, marshalErr
		}
		appendPart(string(raw), tool.CacheControl)
	}

	var systemText string
	if err := json.Unmarshal(req.System, &systemText); err == nil {
		appendPart(systemText, nil)
	} else {
		var blocks []apicompat.AnthropicContentBlock
		if len(req.System) > 0 && json.Unmarshal(req.System, &blocks) == nil {
			for _, block := range blocks {
				appendPart(cacheEstimateBlockText(block), block.CacheControl)
			}
		}
	}

	for _, message := range req.Messages {
		parts = append(parts, "role:"+strings.TrimSpace(message.Role))
		var plain string
		if json.Unmarshal(message.Content, &plain) == nil {
			appendPart(plain, nil)
			continue
		}
		var blocks []apicompat.AnthropicContentBlock
		if json.Unmarshal(message.Content, &blocks) != nil {
			continue
		}
		for _, block := range blocks {
			appendPart(cacheEstimateBlockText(block), block.CacheControl)
		}
	}

	if lastBreakpoint < 0 {
		return 0, nil
	}
	total := 0
	for _, part := range parts[:lastBreakpoint+1] {
		count, countErr := codec.Count(part)
		if countErr != nil {
			return 0, countErr
		}
		total += count
	}
	return total, nil
}

func cacheEstimateBlockText(block apicompat.AnthropicContentBlock) string {
	if strings.TrimSpace(block.Text) != "" {
		return block.Text
	}
	raw, err := json.Marshal(block)
	if err != nil {
		return ""
	}
	return string(raw)
}

func isEphemeralCacheControl(control *apicompat.AnthropicCacheControl) bool {
	return control != nil && strings.EqualFold(strings.TrimSpace(control.Type), "ephemeral")
}
