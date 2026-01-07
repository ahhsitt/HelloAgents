package image

import (
	"fmt"
	"strings"
	"time"
)

// ProviderType 提供商类型
type ProviderType string

const (
	// ProviderOpenAI OpenAI DALL-E / GPT Image
	ProviderOpenAI ProviderType = "openai"
	// ProviderStability Stability AI
	ProviderStability ProviderType = "stability"
	// ProviderDashScope 阿里云 DashScope
	ProviderDashScope ProviderType = "dashscope"
	// ProviderERNIE 百度 ERNIE
	ProviderERNIE ProviderType = "ernie"
	// ProviderHunyuan 腾讯混元
	ProviderHunyuan ProviderType = "hunyuan"
)

// NewImageProvider 根据提供商类型创建图像生成客户端
func NewImageProvider(providerType ProviderType, opts ...Option) (ImageProvider, error) {
	switch providerType {
	case ProviderOpenAI:
		return NewOpenAI(opts...)
	case ProviderStability:
		return NewStability(opts...)
	case ProviderDashScope:
		return NewDashScope(opts...)
	case ProviderERNIE:
		return NewERNIE(opts...)
	case ProviderHunyuan:
		return NewHunyuan(opts...)
	default:
		return nil, fmt.Errorf("unknown provider type: %s", providerType)
	}
}

// ProviderConfig 提供商配置
type ProviderConfig struct {
	// Type 提供商类型
	Type ProviderType `json:"type" yaml:"type"`
	// APIKey API 密钥
	APIKey string `json:"api_key" yaml:"api_key"`
	// SecretKey 密钥（部分厂商需要）
	SecretKey string `json:"secret_key,omitempty" yaml:"secret_key,omitempty"`
	// BaseURL 自定义 API 端点
	BaseURL string `json:"base_url,omitempty" yaml:"base_url,omitempty"`
	// Model 模型名称
	Model string `json:"model,omitempty" yaml:"model,omitempty"`
	// TimeoutSeconds 超时秒数
	TimeoutSeconds int `json:"timeout_seconds,omitempty" yaml:"timeout_seconds,omitempty"`
	// MaxRetries 最大重试次数
	MaxRetries int `json:"max_retries,omitempty" yaml:"max_retries,omitempty"`
}

// NewImageProviderFromConfig 从配置创建图像生成客户端
func NewImageProviderFromConfig(cfg ProviderConfig) (ImageProvider, error) {
	opts := []Option{
		WithAPIKey(cfg.APIKey),
	}

	if cfg.SecretKey != "" {
		opts = append(opts, WithSecretKey(cfg.SecretKey))
	}
	if cfg.BaseURL != "" {
		opts = append(opts, WithBaseURL(cfg.BaseURL))
	}
	if cfg.Model != "" {
		opts = append(opts, WithModel(cfg.Model))
	}
	if cfg.TimeoutSeconds > 0 {
		opts = append(opts, WithTimeout(
			time.Duration(cfg.TimeoutSeconds)*time.Second,
		))
	}
	if cfg.MaxRetries > 0 {
		opts = append(opts, WithMaxRetries(cfg.MaxRetries))
	}

	return NewImageProvider(cfg.Type, opts...)
}

// ParseProviderType 从字符串解析提供商类型
func ParseProviderType(s string) (ProviderType, error) {
	switch strings.ToLower(s) {
	case "openai", "dall-e", "dalle", "gpt-image":
		return ProviderOpenAI, nil
	case "stability", "stable-diffusion", "sd":
		return ProviderStability, nil
	case "dashscope", "aliyun", "wanx", "tongyi":
		return ProviderDashScope, nil
	case "ernie", "baidu", "wenxin", "yige":
		return ProviderERNIE, nil
	case "hunyuan", "tencent":
		return ProviderHunyuan, nil
	default:
		return "", fmt.Errorf("unknown provider: %s", s)
	}
}

// SupportedProviders 返回支持的提供商列表
func SupportedProviders() []ProviderType {
	return []ProviderType{
		ProviderOpenAI,
		ProviderStability,
		ProviderDashScope,
		ProviderERNIE,
		ProviderHunyuan,
	}
}
