package evaluation

import (
	"time"
)

// EvalConfig 评估配置
type EvalConfig struct {
	// MaxSamples 最大样本数（0 表示不限制）
	MaxSamples int

	// Timeout 单个样本评估超时
	Timeout time.Duration

	// ProgressCallback 进度回调函数
	ProgressCallback ProgressCallback

	// SaveIntermediateResults 是否保存中间结果
	SaveIntermediateResults bool

	// OutputDir 输出目录
	OutputDir string

	// Verbose 是否输出详细日志
	Verbose bool
}

// EvalOption 评估选项函数类型
type EvalOption func(*EvalConfig)

// DefaultEvalConfig 返回默认评估配置
func DefaultEvalConfig() *EvalConfig {
	return &EvalConfig{
		MaxSamples: 0, // 不限制
		Timeout:    5 * time.Minute,
		OutputDir:  "./evaluation_results",
		Verbose:    false,
	}
}

// ApplyOptions 应用评估选项
func (c *EvalConfig) ApplyOptions(opts ...EvalOption) {
	for _, opt := range opts {
		opt(c)
	}
}

// WithMaxSamples 设置最大样本数
//
// 参数:
//   - n: 最大样本数，0 表示不限制
func WithMaxSamples(n int) EvalOption {
	return func(c *EvalConfig) {
		c.MaxSamples = n
	}
}

// WithTimeout 设置单个样本评估超时
//
// 参数:
//   - d: 超时时间
func WithTimeout(d time.Duration) EvalOption {
	return func(c *EvalConfig) {
		c.Timeout = d
	}
}

// WithProgressCallback 设置进度回调函数
//
// 参数:
//   - callback: 进度回调函数，每完成一个样本调用一次
func WithProgressCallback(callback ProgressCallback) EvalOption {
	return func(c *EvalConfig) {
		c.ProgressCallback = callback
	}
}

// WithSaveIntermediateResults 设置是否保存中间结果
//
// 参数:
//   - save: 是否保存
func WithSaveIntermediateResults(save bool) EvalOption {
	return func(c *EvalConfig) {
		c.SaveIntermediateResults = save
	}
}

// WithOutputDir 设置输出目录
//
// 参数:
//   - dir: 输出目录路径
func WithOutputDir(dir string) EvalOption {
	return func(c *EvalConfig) {
		c.OutputDir = dir
	}
}

// WithVerbose 设置是否输出详细日志
//
// 参数:
//   - verbose: 是否详细输出
func WithVerbose(verbose bool) EvalOption {
	return func(c *EvalConfig) {
		c.Verbose = verbose
	}
}
