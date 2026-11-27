package service

import (
	"regexp"
	"strings"
)

// ModelCapability* 是模型能力标识，取值与 infinite-canvas 等客户端的 capability 对齐。
const (
	ModelCapabilityImage = "image"
	ModelCapabilityVideo = "video"
	ModelCapabilityText  = "text"
	ModelCapabilityAudio = "audio"
)

// ModelProtocol* 是模型可用的调用协议标识。
const (
	ModelProtocolImages    = "images"
	ModelProtocolGemini    = "gemini"
	ModelProtocolChat      = "chat"
	ModelProtocolResponses = "responses"
)

// ModelDescriptor 描述一个模型对外声明的能力与可用协议。
type ModelDescriptor struct {
	Capability string
	Protocols  []string
}

var (
	veoModelPattern         = regexp.MustCompile(`(^|[/_-])veo[-_]?\d`)
	wanVideoModelPattern    = regexp.MustCompile(`(^|[/_-])wanx?\d`)
	minimaxTextModelPattern = regexp.MustCompile(`minimax[-_]?(m\d|text|abab)`)
	openAIChatModelPattern  = regexp.MustCompile(`^(gpt-|chatgpt-|codex|o\d)`)
)

var videoModelKeywords = []string{"video", "sora", "kling", "hailuo", "seedance", "vidu", "runway", "pixverse"}

// nonChatModelKeywords 命中的模型虽然名字像对话模型（如 gpt-4o-mini-tts），但不能走对话协议。
var nonChatModelKeywords = []string{"embedding", "embed", "tts", "whisper", "transcribe", "audio", "realtime", "speech", "moderation", "rerank", "dall-e"}

// chatModelPrefixes 是只声明 chat 协议的常见对话模型前缀（按去掉厂商路径后的名字匹配）。
var chatModelPrefixes = []string{
	"claude-", "gemini-", "deepseek", "qwen", "qwq", "glm-", "kimi", "moonshot", "doubao",
	"grok-", "mistral", "mixtral", "llama", "ernie", "hunyuan", "yi-", "step-", "minimax-m", "abab",
}

// DescribeModel 按模型 ID 判定能力与可用协议；未识别返回零值与 false。
//
// 判定只依赖名称，图片部分的口径与 composite_platform.go 的 gpt-image- 前缀判定、
// image_generation_intent.go 的图片意图识别保持一致；本函数只做收敛，
// 不改变那些逻辑的运行时行为。视频判定放在图片之前，否则 grok-imagine-video
// 会因为含 imagine 被误判成图片。
func DescribeModel(modelID string) (ModelDescriptor, bool) {
	name := strings.ToLower(strings.TrimSpace(modelID))
	base := name[strings.LastIndex(name, "/")+1:]
	switch {
	case name == "":
		return ModelDescriptor{}, false
	case isVideoModelName(name):
		return ModelDescriptor{Capability: ModelCapabilityVideo}, true
	case strings.HasPrefix(name, "gpt-image-"):
		return imageDescriptor(ModelProtocolImages), true
	case strings.HasPrefix(name, "gemini-") && strings.Contains(name, "-image"):
		return ModelDescriptor{
			Capability: ModelCapabilityImage,
			Protocols:  []string{ModelProtocolGemini, ModelProtocolChat},
		}, true
	case strings.HasPrefix(name, "grok-") && (strings.Contains(name, "imagine") || strings.Contains(name, "image")):
		return imageDescriptor(ModelProtocolImages), true
	case strings.Contains(name, "seedream"),
		strings.Contains(name, "flux"),
		strings.HasPrefix(name, "imagen-"):
		return imageDescriptor(ModelProtocolImages), true
	case modelNameContainsAny(name, nonChatModelKeywords):
		return ModelDescriptor{}, false
	case openAIChatModelPattern.MatchString(base):
		return ModelDescriptor{Capability: ModelCapabilityText, Protocols: []string{ModelProtocolResponses, ModelProtocolChat}}, true
	case modelNameHasPrefix(base, chatModelPrefixes):
		return ModelDescriptor{Capability: ModelCapabilityText, Protocols: []string{ModelProtocolChat}}, true
	}
	return ModelDescriptor{}, false
}

func isVideoModelName(name string) bool {
	if modelNameContainsAny(name, videoModelKeywords) || veoModelPattern.MatchString(name) || wanVideoModelPattern.MatchString(name) {
		return true
	}
	return strings.Contains(name, "minimax") && !minimaxTextModelPattern.MatchString(name)
}

func modelNameContainsAny(name string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(name, keyword) {
			return true
		}
	}
	return false
}

func modelNameHasPrefix(name string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func imageDescriptor(protocols ...string) ModelDescriptor {
	return ModelDescriptor{Capability: ModelCapabilityImage, Protocols: protocols}
}
