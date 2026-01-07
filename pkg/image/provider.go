// Package image 提供文生图（Text-to-Image）服务的统一接口
package image

import (
	"context"
)

// ImageProvider 定义图像生成提供商接口
//
// 统一不同图像生成服务的调用方式，支持 OpenAI DALL-E、Stability AI、通义万象等。
type ImageProvider interface {
	// Generate 生成图像
	//
	// 参数:
	//   - ctx: 上下文
	//   - req: 请求参数
	//
	// 返回:
	//   - ImageResponse: 生成结果
	//   - error: 调用错误
	Generate(ctx context.Context, req ImageRequest) (ImageResponse, error)

	// Name 返回提供商名称
	Name() string

	// Model 返回当前模型名称
	Model() string

	// SupportedSizes 返回支持的图像尺寸列表
	SupportedSizes() []ImageSize

	// Close 关闭客户端连接
	Close() error
}

// ImageSize 图像尺寸
type ImageSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// String 返回尺寸字符串表示
func (s ImageSize) String() string {
	return formatSize(s.Width, s.Height)
}

// Pixels 返回总像素数
func (s ImageSize) Pixels() int {
	return s.Width * s.Height
}

// AspectRatio 返回宽高比
func (s ImageSize) AspectRatio() float64 {
	if s.Height == 0 {
		return 0
	}
	return float64(s.Width) / float64(s.Height)
}

// ImageQuality 图像质量等级
type ImageQuality string

const (
	// QualityStandard 标准质量
	QualityStandard ImageQuality = "standard"
	// QualityHD 高清质量
	QualityHD ImageQuality = "hd"
	// QualityUltra 超高清质量
	QualityUltra ImageQuality = "ultra"
)

// ImageStyle 图像风格
type ImageStyle string

const (
	// StyleVivid 鲜艳风格（生成更鲜艳、更具戏剧性的图像）
	StyleVivid ImageStyle = "vivid"
	// StyleNatural 自然风格（生成更自然、更写实的图像）
	StyleNatural ImageStyle = "natural"
	// StyleAnime 动漫风格
	StyleAnime ImageStyle = "anime"
	// StylePhotographic 摄影风格
	StylePhotographic ImageStyle = "photographic"
	// StyleDigitalArt 数字艺术风格
	StyleDigitalArt ImageStyle = "digital-art"
	// StyleInkWash 水墨风格
	StyleInkWash ImageStyle = "ink-wash"
)

// ResponseFormat 响应格式
type ResponseFormat string

const (
	// FormatURL 返回图像 URL
	FormatURL ResponseFormat = "url"
	// FormatBase64 返回 Base64 编码的图像数据
	FormatBase64 ResponseFormat = "base64"
)

// ImageRequest 图像生成请求
type ImageRequest struct {
	// Prompt 生成提示词（必填）
	Prompt string `json:"prompt"`

	// NegativePrompt 负面提示词（可选，部分厂商支持）
	NegativePrompt string `json:"negative_prompt,omitempty"`

	// Size 图像尺寸（与 AspectRatio 二选一）
	Size ImageSize `json:"size,omitempty"`

	// AspectRatio 宽高比，如 "16:9"、"1:1"（与 Size 二选一）
	AspectRatio string `json:"aspect_ratio,omitempty"`

	// N 生成数量（默认 1）
	N int `json:"n,omitempty"`

	// Quality 质量等级
	Quality ImageQuality `json:"quality,omitempty"`

	// Style 风格预设
	Style ImageStyle `json:"style,omitempty"`

	// Seed 随机种子（可选，用于可复现生成）
	Seed *int64 `json:"seed,omitempty"`

	// ResponseFormat 响应格式
	ResponseFormat ResponseFormat `json:"response_format,omitempty"`

	// Extra 厂商特定参数
	Extra map[string]interface{} `json:"extra,omitempty"`
}

// ImageResponse 图像生成响应
type ImageResponse struct {
	// Images 生成的图像列表
	Images []GeneratedImage `json:"images"`

	// Created 创建时间（Unix 时间戳）
	Created int64 `json:"created"`

	// Model 使用的模型
	Model string `json:"model,omitempty"`
}

// GeneratedImage 生成的单张图像
type GeneratedImage struct {
	// URL 图像 URL（临时链接，通常有效期有限）
	URL string `json:"url,omitempty"`

	// Base64 Base64 编码的图像数据
	Base64 string `json:"base64,omitempty"`

	// RevisedPrompt 模型修改后的提示词（OpenAI 特有）
	RevisedPrompt string `json:"revised_prompt,omitempty"`

	// Seed 实际使用的随机种子
	Seed *int64 `json:"seed,omitempty"`

	// ContentType 图像内容类型，如 "image/png"
	ContentType string `json:"content_type,omitempty"`
}

// formatSize 格式化尺寸为字符串
func formatSize(width, height int) string {
	return string(rune('0'+width/1000)) + string(rune('0'+(width%1000)/100)) +
		string(rune('0'+(width%100)/10)) + string(rune('0'+width%10)) + "x" +
		string(rune('0'+height/1000)) + string(rune('0'+(height%1000)/100)) +
		string(rune('0'+(height%100)/10)) + string(rune('0'+height%10))
}

// ParseSize 从字符串解析尺寸，如 "1024x1024"
func ParseSize(s string) (ImageSize, error) {
	var width, height int
	_, err := parseSize(s, &width, &height)
	if err != nil {
		return ImageSize{}, err
	}
	return ImageSize{Width: width, Height: height}, nil
}

// parseSize 解析尺寸字符串
func parseSize(s string, width, height *int) (bool, error) {
	// 简单解析 WIDTHxHEIGHT 格式
	var w, h int
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			w = w*10 + int(s[i]-'0')
		} else if s[i] == 'x' || s[i] == 'X' {
			n = i + 1
			break
		} else {
			return false, ErrInvalidSize
		}
	}
	if n == 0 {
		return false, ErrInvalidSize
	}
	for i := n; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			h = h*10 + int(s[i]-'0')
		} else {
			return false, ErrInvalidSize
		}
	}
	if w == 0 || h == 0 {
		return false, ErrInvalidSize
	}
	*width = w
	*height = h
	return true, nil
}
