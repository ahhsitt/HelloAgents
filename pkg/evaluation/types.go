package evaluation

import (
	"time"
)

// Sample 评估样本
//
// Sample 是所有评估基准的通用样本结构。不同基准可能只使用部分字段。
type Sample struct {
	// ID 样本唯一标识
	ID string `json:"id"`

	// Input 输入内容（问题/查询）
	Input string `json:"input"`

	// Expected 期望输出（ground truth）
	Expected interface{} `json:"expected"`

	// Category 样本类别（用于分类统计）
	Category string `json:"category,omitempty"`

	// Level 难度级别（用于 GAIA 等分级评估）
	Level int `json:"level,omitempty"`

	// Metadata 额外元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Tools 可用工具列表（用于 BFCL）
	Tools []ToolDefinition `json:"tools,omitempty"`

	// Files 附件文件列表（用于 GAIA）
	Files []string `json:"files,omitempty"`
}

// ToolDefinition 工具定义
type ToolDefinition struct {
	// Name 工具名称
	Name string `json:"name"`

	// Description 工具描述
	Description string `json:"description"`

	// Parameters 参数 Schema
	Parameters map[string]interface{} `json:"parameters"`
}

// SampleResult 单个样本的评估结果
type SampleResult struct {
	// SampleID 样本 ID
	SampleID string `json:"sample_id"`

	// Predicted 预测输出
	Predicted interface{} `json:"predicted"`

	// Expected 期望输出
	Expected interface{} `json:"expected"`

	// Success 是否成功（精确匹配）
	Success bool `json:"success"`

	// PartialSuccess 部分成功（用于 GAIA 等支持部分匹配的场景）
	PartialSuccess bool `json:"partial_success,omitempty"`

	// Score 评分（0-1 或其他范围）
	Score float64 `json:"score"`

	// Category 样本类别
	Category string `json:"category,omitempty"`

	// Level 难度级别
	Level int `json:"level,omitempty"`

	// ExecutionTime 执行时间
	ExecutionTime time.Duration `json:"execution_time"`

	// Error 错误信息（如有）
	Error string `json:"error,omitempty"`

	// Details 详细信息（用于调试）
	Details map[string]interface{} `json:"details,omitempty"`

	// AgentResponse 智能体原始响应
	AgentResponse string `json:"agent_response,omitempty"`
}

// EvalResult 完整评估结果
type EvalResult struct {
	// BenchmarkName 基准名称
	BenchmarkName string `json:"benchmark_name"`

	// AgentName 智能体名称
	AgentName string `json:"agent_name"`

	// TotalSamples 总样本数
	TotalSamples int `json:"total_samples"`

	// SuccessCount 成功数量
	SuccessCount int `json:"success_count"`

	// OverallAccuracy 总体准确率
	OverallAccuracy float64 `json:"overall_accuracy"`

	// CategoryMetrics 分类别指标
	CategoryMetrics map[string]*CategoryMetrics `json:"category_metrics,omitempty"`

	// LevelMetrics 分级别指标（用于 GAIA）
	LevelMetrics map[int]*LevelMetrics `json:"level_metrics,omitempty"`

	// DetailedResults 详细结果列表
	DetailedResults []*SampleResult `json:"detailed_results"`

	// TotalDuration 总执行时间
	TotalDuration time.Duration `json:"total_duration"`

	// EvaluationTime 评估时间戳
	EvaluationTime time.Time `json:"evaluation_time"`

	// Metrics 汇总指标
	Metrics *MetricsSummary `json:"metrics,omitempty"`
}

// CategoryMetrics 分类别指标
type CategoryMetrics struct {
	// Category 类别名称
	Category string `json:"category"`

	// Total 总数
	Total int `json:"total"`

	// Success 成功数
	Success int `json:"success"`

	// Accuracy 准确率
	Accuracy float64 `json:"accuracy"`

	// AverageScore 平均分
	AverageScore float64 `json:"average_score,omitempty"`
}

// LevelMetrics 分级别指标
type LevelMetrics struct {
	// Level 级别
	Level int `json:"level"`

	// Total 总数
	Total int `json:"total"`

	// ExactMatches 精确匹配数
	ExactMatches int `json:"exact_matches"`

	// PartialMatches 部分匹配数
	PartialMatches int `json:"partial_matches,omitempty"`

	// ExactMatchRate 精确匹配率
	ExactMatchRate float64 `json:"exact_match_rate"`

	// PartialMatchRate 部分匹配率
	PartialMatchRate float64 `json:"partial_match_rate,omitempty"`
}

// MetricsSummary 指标汇总
type MetricsSummary struct {
	// Accuracy 准确率
	Accuracy float64 `json:"accuracy"`

	// Precision 精确率
	Precision float64 `json:"precision,omitempty"`

	// Recall 召回率
	Recall float64 `json:"recall,omitempty"`

	// F1Score F1 分数
	F1Score float64 `json:"f1_score,omitempty"`

	// AverageScore 平均分
	AverageScore float64 `json:"average_score,omitempty"`

	// PassRate 通过率（用于 LLM Judge）
	PassRate float64 `json:"pass_rate,omitempty"`

	// ExcellentRate 优秀率（用于 LLM Judge）
	ExcellentRate float64 `json:"excellent_rate,omitempty"`

	// WinRate 胜率（用于 Win Rate）
	WinRate float64 `json:"win_rate,omitempty"`

	// LossRate 败率（用于 Win Rate）
	LossRate float64 `json:"loss_rate,omitempty"`

	// TieRate 平局率（用于 Win Rate）
	TieRate float64 `json:"tie_rate,omitempty"`

	// DimensionScores 各维度分数（用于 LLM Judge）
	DimensionScores map[string]float64 `json:"dimension_scores,omitempty"`

	// Extra 额外指标
	Extra map[string]interface{} `json:"extra,omitempty"`
}

// FunctionCall 函数调用结构（用于 BFCL）
type FunctionCall struct {
	// Name 函数名
	Name string `json:"name"`

	// Arguments 参数
	Arguments map[string]interface{} `json:"arguments"`
}

// JudgeScore LLM Judge 评分结果
type JudgeScore struct {
	// Correctness 正确性评分
	Correctness float64 `json:"correctness"`

	// Clarity 清晰度评分
	Clarity float64 `json:"clarity"`

	// DifficultyMatch 难度匹配评分
	DifficultyMatch float64 `json:"difficulty_match"`

	// Completeness 完整性评分
	Completeness float64 `json:"completeness"`

	// TotalScore 总分
	TotalScore float64 `json:"total_score"`

	// Comments 评语
	Comments string `json:"comments,omitempty"`
}

// ComparisonResult 对比结果（用于 Win Rate）
type ComparisonResult struct {
	// ProblemAID 问题 A 的 ID
	ProblemAID string `json:"problem_a_id"`

	// ProblemBID 问题 B 的 ID
	ProblemBID string `json:"problem_b_id"`

	// Winner 胜者（"A"、"B" 或 "Tie"）
	Winner string `json:"winner"`

	// ActualWinner 实际胜者（考虑位置随机化后）
	ActualWinner string `json:"actual_winner"`

	// Reason 理由
	Reason string `json:"reason"`

	// ExecutionTime 执行时间
	ExecutionTime time.Duration `json:"execution_time"`
}
