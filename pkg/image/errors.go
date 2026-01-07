package image

import "errors"

// 图像生成相关错误
var (
	// ErrInvalidPrompt 提示词无效（为空或格式错误）
	ErrInvalidPrompt = errors.New("invalid prompt: prompt cannot be empty")

	// ErrInvalidSize 图像尺寸无效
	ErrInvalidSize = errors.New("invalid image size")

	// ErrUnsupportedSize 不支持的图像尺寸
	ErrUnsupportedSize = errors.New("unsupported image size for this provider")

	// ErrContentFiltered 内容被安全系统过滤
	ErrContentFiltered = errors.New("content filtered by safety system")

	// ErrQuotaExceeded 配额或速率限制超出
	ErrQuotaExceeded = errors.New("image generation quota exceeded")

	// ErrGenerationFailed 图像生成失败
	ErrGenerationFailed = errors.New("image generation failed")

	// ErrInvalidAPIKey API 密钥无效
	ErrInvalidAPIKey = errors.New("invalid API key")

	// ErrProviderUnavailable 提供商不可用
	ErrProviderUnavailable = errors.New("image provider unavailable")

	// ErrTimeout 请求超时
	ErrTimeout = errors.New("request timeout")

	// ErrInvalidResponse 响应无效
	ErrInvalidResponse = errors.New("invalid response from provider")

	// ErrModelNotSupported 模型不支持
	ErrModelNotSupported = errors.New("model not supported")
)

// IsRetryable 判断错误是否可重试
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrQuotaExceeded) ||
		errors.Is(err, ErrTimeout) ||
		errors.Is(err, ErrProviderUnavailable)
}

// IsFatal 判断错误是否为致命错误（不可恢复）
func IsFatal(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrInvalidAPIKey) ||
		errors.Is(err, ErrInvalidPrompt) ||
		errors.Is(err, ErrModelNotSupported)
}

// WrapError 包装错误并添加上下文信息
func WrapError(err error, context string) error {
	if err == nil {
		return nil
	}
	return &wrappedError{
		context: context,
		err:     err,
	}
}

type wrappedError struct {
	context string
	err     error
}

func (e *wrappedError) Error() string {
	return e.context + ": " + e.err.Error()
}

func (e *wrappedError) Unwrap() error {
	return e.err
}
