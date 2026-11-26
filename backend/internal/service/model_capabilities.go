package service

import "strings"

// ModelCapability* 是模型能力标识，取值与 infinite-canvas 等客户端的 capability 对齐。
const (
	ModelCapabilityImage = "image"
	ModelCapabilityVideo = "video"
	ModelCapabilityText  = "text"
	ModelCapabilityAudio = "audio"
)

// ModelProtocol* 是模型可用的调用协议标识。
const (
	ModelProtocolImages = "images"
	ModelProtocolGemini = "gemini"
	ModelProtocolChat   = "chat"
)

// ModelDescriptor 描述一个模型对外声明的能力与可用协议。
type ModelDescriptor struct {
	Capability string
	Protocols  []string
}

// DescribeModel 按模型 ID 判定能力与可用协议；未识别返回零值与 false。
//
// 判定只依赖名称，口径与 composite_platform.go 的 gpt-image- 前缀判定、
// image_generation_intent.go 的图片意图识别保持一致；本函数只做收敛，
// 不改变那些逻辑的运行时行为。
func DescribeModel(modelID string) (ModelDescriptor, bool) {
	name := strings.ToLower(strings.TrimSpace(modelID))
	switch {
	case name == "":
		return ModelDescriptor{}, false
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
	}
	return ModelDescriptor{}, false
}

func imageDescriptor(protocols ...string) ModelDescriptor {
	return ModelDescriptor{Capability: ModelCapabilityImage, Protocols: protocols}
}
