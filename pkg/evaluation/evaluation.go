// Package evaluation 提供智能体评估框架
//
// 本包实现了多种评估基准测试，用于评估 Agent 的各项能力：
// - BFCL (Berkeley Function Calling Leaderboard): 工具/函数调用能力评估
// - GAIA (General AI Assistants): 通用 AI 助手能力评估
// - LLM Judge: 使用 LLM 作为评委进行质量评估
// - Win Rate: 成对对比计算胜率
package evaluation

import (
	"context"

	"github.com/ahhsitt/helloagents-go/pkg/agents"
)

// Dataset 数据集接口
//
// 所有评估数据集必须实现此接口。数据集负责加载和管理评估样本。
type Dataset interface {
	// Load 加载数据集
	//
	// 参数:
	//   - ctx: 上下文，用于取消控制
	//
	// 返回:
	//   - error: 加载错误
	Load(ctx context.Context) error

	// Len 返回数据集大小
	Len() int

	// Get 根据索引获取样本
	//
	// 参数:
	//   - index: 样本索引
	//
	// 返回:
	//   - Sample: 样本数据
	//   - error: 获取错误（如索引越界）
	Get(index int) (Sample, error)

	// Iterator 返回样本迭代器
	//
	// 返回一个 channel，用于遍历所有样本。
	// channel 会在所有样本发送完成后关闭。
	Iterator() <-chan Sample

	// Name 返回数据集名称
	Name() string
}

// Evaluator 评估器接口
//
// 所有评估器必须实现此接口。评估器负责执行评估逻辑。
type Evaluator interface {
	// Evaluate 执行完整评估
	//
	// 参数:
	//   - ctx: 上下文，用于取消和超时控制
	//   - agent: 要评估的智能体
	//   - opts: 评估选项
	//
	// 返回:
	//   - *EvalResult: 评估结果
	//   - error: 评估错误
	Evaluate(ctx context.Context, agent agents.Agent, opts ...EvalOption) (*EvalResult, error)

	// EvaluateSample 评估单个样本
	//
	// 参数:
	//   - ctx: 上下文
	//   - agent: 要评估的智能体
	//   - sample: 评估样本
	//
	// 返回:
	//   - *SampleResult: 样本评估结果
	//   - error: 评估错误
	EvaluateSample(ctx context.Context, agent agents.Agent, sample Sample) (*SampleResult, error)

	// Name 返回评估器名称
	Name() string
}

// Metrics 指标计算接口
//
// 指标计算器负责从样本结果中计算汇总指标。
type Metrics interface {
	// Compute 计算指标
	//
	// 参数:
	//   - results: 样本评估结果列表
	//
	// 返回:
	//   - *MetricsSummary: 指标汇总
	Compute(results []*SampleResult) *MetricsSummary
}

// ProgressCallback 进度回调函数类型
//
// 参数:
//   - done: 已完成数量
//   - total: 总数量
type ProgressCallback func(done, total int)
