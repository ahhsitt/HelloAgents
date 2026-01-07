package image

import (
	"net/http"
	"time"
)

// Option 图像生成配置选项函数
type Option func(*Options)

// Options 图像生成配置选项
type Options struct {
	// APIKey API 密钥
	APIKey string
	// SecretKey 密钥（部分厂商如百度需要）
	SecretKey string
	// BaseURL 自定义 API 端点
	BaseURL string
	// Model 模型名称
	Model string
	// Timeout 请求超时
	Timeout time.Duration
	// MaxRetries 最大重试次数
	MaxRetries int
	// RetryDelay 重试间隔基数
	RetryDelay time.Duration
	// HTTPClient 自定义 HTTP 客户端
	HTTPClient *http.Client
	// DefaultSize 默认图像尺寸
	DefaultSize ImageSize
	// DefaultQuality 默认质量
	DefaultQuality ImageQuality
	// DefaultStyle 默认风格
	DefaultStyle ImageStyle
	// DefaultFormat 默认响应格式
	DefaultFormat ResponseFormat
}

// DefaultOptions 返回默认选项
func DefaultOptions() *Options {
	return &Options{
		Timeout:        60 * time.Second,
		MaxRetries:     3,
		RetryDelay:     time.Second,
		DefaultSize:    ImageSize{Width: 1024, Height: 1024},
		DefaultQuality: QualityStandard,
		DefaultFormat:  FormatURL,
	}
}

// WithAPIKey 设置 API 密钥
func WithAPIKey(key string) Option {
	return func(o *Options) {
		o.APIKey = key
	}
}

// WithSecretKey 设置密钥（部分厂商需要）
func WithSecretKey(key string) Option {
	return func(o *Options) {
		o.SecretKey = key
	}
}

// WithBaseURL 设置自定义端点
func WithBaseURL(url string) Option {
	return func(o *Options) {
		o.BaseURL = url
	}
}

// WithModel 设置模型
func WithModel(model string) Option {
	return func(o *Options) {
		o.Model = model
	}
}

// WithTimeout 设置超时时间
func WithTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.Timeout = d
	}
}

// WithMaxRetries 设置最大重试次数
func WithMaxRetries(n int) Option {
	return func(o *Options) {
		o.MaxRetries = n
	}
}

// WithRetryDelay 设置重试间隔
func WithRetryDelay(d time.Duration) Option {
	return func(o *Options) {
		o.RetryDelay = d
	}
}

// WithHTTPClient 设置自定义 HTTP 客户端
func WithHTTPClient(client *http.Client) Option {
	return func(o *Options) {
		o.HTTPClient = client
	}
}

// WithDefaultSize 设置默认图像尺寸
func WithDefaultSize(size ImageSize) Option {
	return func(o *Options) {
		o.DefaultSize = size
	}
}

// WithDefaultQuality 设置默认质量
func WithDefaultQuality(quality ImageQuality) Option {
	return func(o *Options) {
		o.DefaultQuality = quality
	}
}

// WithDefaultStyle 设置默认风格
func WithDefaultStyle(style ImageStyle) Option {
	return func(o *Options) {
		o.DefaultStyle = style
	}
}

// WithDefaultFormat 设置默认响应格式
func WithDefaultFormat(format ResponseFormat) Option {
	return func(o *Options) {
		o.DefaultFormat = format
	}
}

// ApplyOptions 应用选项到 Options
func ApplyOptions(opts *Options, options ...Option) {
	for _, opt := range options {
		opt(opts)
	}
}
