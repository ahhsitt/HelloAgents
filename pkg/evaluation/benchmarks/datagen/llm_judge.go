package datagen

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/ahhsitt/helloagents-go/pkg/core/llm"
	"github.com/ahhsitt/helloagents-go/pkg/core/message"
	"github.com/ahhsitt/helloagents-go/pkg/evaluation"
)

// JudgeConfig LLM Judge 配置
type JudgeConfig struct {
	// ReferenceSamples 参考样本（用于对比评估）
	ReferenceSamples []evaluation.Sample
}

// LLMJudge LLM 评委评估器
type LLMJudge struct {
	// llmProvider LLM 提供商
	llmProvider llm.Provider

	// config 配置
	config JudgeConfig

	// dataset 待评估数据集
	dataset *Dataset
}

// NewLLMJudge 创建 LLM Judge 评估器
//
// 参数:
//   - llmProvider: LLM 服务提供商
//   - dataset: 待评估数据集
//   - config: 评估配置
func NewLLMJudge(llmProvider llm.Provider, dataset *Dataset, config JudgeConfig) *LLMJudge {
	return &LLMJudge{
		llmProvider: llmProvider,
		dataset:     dataset,
		config:      config,
	}
}

// Name 返回评估器名称
func (j *LLMJudge) Name() string {
	return "LLMJudge"
}

// Evaluate 执行完整评估
func (j *LLMJudge) Evaluate(ctx context.Context, opts ...evaluation.EvalOption) (*evaluation.EvalResult, error) {
	config := evaluation.DefaultEvalConfig()
	config.ApplyOptions(opts...)

	// 确保数据集已加载
	if err := j.dataset.Load(ctx); err != nil {
		return nil, fmt.Errorf("加载数据集失败: %w", err)
	}

	startTime := time.Now()
	result := &evaluation.EvalResult{
		BenchmarkName:   j.Name(),
		AgentName:       j.llmProvider.Name(),
		DetailedResults: make([]*evaluation.SampleResult, 0),
		EvaluationTime:  startTime,
	}

	total := j.dataset.Len()
	if config.MaxSamples > 0 && config.MaxSamples < total {
		total = config.MaxSamples
	}
	result.TotalSamples = total

	// 遍历样本进行评估
	for i := 0; i < total; i++ {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		sample, err := j.dataset.Get(i)
		if err != nil {
			continue
		}

		// 应用超时
		evalCtx := ctx
		if config.Timeout > 0 {
			var cancel context.CancelFunc
			evalCtx, cancel = context.WithTimeout(ctx, config.Timeout)
			defer cancel()
		}

		// 获取参考样本（如果有）
		var refSample *evaluation.Sample
		if i < len(j.config.ReferenceSamples) {
			ref := j.config.ReferenceSamples[i]
			refSample = &ref
		}

		sampleResult, err := j.EvaluateSample(evalCtx, sample, refSample)
		if err != nil {
			sampleResult = &evaluation.SampleResult{
				SampleID: sample.ID,
				Category: sample.Category,
				Error:    err.Error(),
				Success:  false,
			}
		}

		result.DetailedResults = append(result.DetailedResults, sampleResult)
		if sampleResult.Success {
			result.SuccessCount++
		}

		// 进度回调
		if config.ProgressCallback != nil {
			config.ProgressCallback(i+1, total)
		}
	}

	result.TotalDuration = time.Since(startTime)
	if result.TotalSamples > 0 {
		result.OverallAccuracy = float64(result.SuccessCount) / float64(result.TotalSamples)
	}

	// 计算汇总指标
	result.Metrics = j.computeMetrics(result.DetailedResults)

	return result, nil
}

// EvaluateSample 评估单个样本
func (j *LLMJudge) EvaluateSample(ctx context.Context, sample evaluation.Sample, refSample *evaluation.Sample) (*evaluation.SampleResult, error) {
	startTime := time.Now()

	result := &evaluation.SampleResult{
		SampleID: sample.ID,
		Category: sample.Category,
		Details:  make(map[string]interface{}),
	}

	// 构建评估提示
	prompt := j.buildJudgePrompt(sample, refSample)

	// 调用 LLM
	req := llm.Request{
		Messages: []message.Message{
			message.NewSystemMessage(j.getSystemPrompt()),
			message.NewUserMessage(prompt),
		},
	}

	resp, err := j.llmProvider.Generate(ctx, req)
	if err != nil {
		result.Error = err.Error()
		result.ExecutionTime = time.Since(startTime)
		return result, nil
	}

	result.AgentResponse = resp.Content
	result.ExecutionTime = time.Since(startTime)

	// 解析评分
	score := j.parseJudgeResponse(resp.Content)
	result.Predicted = score
	result.Details["judge_score"] = score

	// 计算总分和成功判断
	totalScore := (score.Correctness + score.Clarity + score.DifficultyMatch + score.Completeness) / 4.0
	result.Score = totalScore
	result.Success = totalScore >= 3.0 // 平均分 >= 3 认为通过

	result.Details["total_score"] = totalScore
	result.Details["correctness"] = score.Correctness
	result.Details["clarity"] = score.Clarity
	result.Details["difficulty_match"] = score.DifficultyMatch
	result.Details["completeness"] = score.Completeness
	result.Details["comments"] = score.Comments

	return result, nil
}

// getSystemPrompt 获取系统提示
func (j *LLMJudge) getSystemPrompt() string {
	return `你是一个专业的题目质量评估专家。请根据以下维度对给定的题目进行评分（1-5分）：

1. 正确性 (Correctness): 题目和答案是否正确
2. 清晰度 (Clarity): 题目描述是否清晰、无歧义
3. 难度匹配 (Difficulty Match): 题目难度是否与标注一致
4. 完整性 (Completeness): 题目信息是否完整

请以 JSON 格式返回评分结果：
{
  "correctness": <1-5>,
  "clarity": <1-5>,
  "difficulty_match": <1-5>,
  "completeness": <1-5>,
  "comments": "<评价说明>"
}`
}

// buildJudgePrompt 构建评估提示
func (j *LLMJudge) buildJudgePrompt(sample evaluation.Sample, refSample *evaluation.Sample) string {
	prompt := fmt.Sprintf("## 待评估题目\n\n**问题**: %s\n", sample.Input)

	if answer, ok := sample.Expected.(string); ok && answer != "" {
		prompt += fmt.Sprintf("\n**答案**: %s\n", answer)
	}

	if sample.Category != "" {
		prompt += fmt.Sprintf("\n**类别/难度**: %s\n", sample.Category)
	}

	if refSample != nil {
		prompt += fmt.Sprintf("\n---\n\n## 参考题目（用于对比）\n\n**问题**: %s\n", refSample.Input)
		if answer, ok := refSample.Expected.(string); ok && answer != "" {
			prompt += fmt.Sprintf("\n**答案**: %s\n", answer)
		}
	}

	prompt += "\n请对待评估题目进行打分。"

	return prompt
}

// parseJudgeResponse 解析评委响应
func (j *LLMJudge) parseJudgeResponse(response string) evaluation.JudgeScore {
	score := evaluation.JudgeScore{
		Correctness:     3.0, // 默认分数
		Clarity:         3.0,
		DifficultyMatch: 3.0,
		Completeness:    3.0,
	}

	// 尝试从 Markdown 代码块中提取 JSON
	codeBlockPattern := regexp.MustCompile("```(?:json)?\\s*([\\s\\S]*?)```")
	matches := codeBlockPattern.FindStringSubmatch(response)

	var jsonContent string
	if len(matches) > 1 {
		jsonContent = matches[1]
	} else {
		// 尝试直接解析
		jsonContent = response
	}

	// 尝试解析 JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonContent), &parsed); err == nil {
		if v, ok := parsed["correctness"].(float64); ok {
			score.Correctness = v
		}
		if v, ok := parsed["clarity"].(float64); ok {
			score.Clarity = v
		}
		if v, ok := parsed["difficulty_match"].(float64); ok {
			score.DifficultyMatch = v
		}
		if v, ok := parsed["completeness"].(float64); ok {
			score.Completeness = v
		}
		if v, ok := parsed["comments"].(string); ok {
			score.Comments = v
		}
	}

	score.TotalScore = (score.Correctness + score.Clarity + score.DifficultyMatch + score.Completeness) / 4.0

	return score
}

// computeMetrics 计算汇总指标
func (j *LLMJudge) computeMetrics(results []*evaluation.SampleResult) *evaluation.MetricsSummary {
	summary := &evaluation.MetricsSummary{
		DimensionScores: make(map[string]float64),
		Extra:           make(map[string]interface{}),
	}

	if len(results) == 0 {
		return summary
	}

	var totalCorrectness, totalClarity, totalDifficultyMatch, totalCompleteness float64
	var totalScore float64
	successCount := 0
	excellentCount := 0

	for _, r := range results {
		if r.Details != nil {
			if v, ok := r.Details["correctness"].(float64); ok {
				totalCorrectness += v
			}
			if v, ok := r.Details["clarity"].(float64); ok {
				totalClarity += v
			}
			if v, ok := r.Details["difficulty_match"].(float64); ok {
				totalDifficultyMatch += v
			}
			if v, ok := r.Details["completeness"].(float64); ok {
				totalCompleteness += v
			}
		}
		totalScore += r.Score

		if r.Success {
			successCount++
		}
		if r.Score >= 4.0 {
			excellentCount++
		}
	}

	n := float64(len(results))
	summary.AverageScore = totalScore / n
	summary.PassRate = float64(successCount) / n
	summary.ExcellentRate = float64(excellentCount) / n
	summary.Accuracy = summary.PassRate

	// 各维度平均分
	summary.DimensionScores["correctness"] = totalCorrectness / n
	summary.DimensionScores["clarity"] = totalClarity / n
	summary.DimensionScores["difficulty_match"] = totalDifficultyMatch / n
	summary.DimensionScores["completeness"] = totalCompleteness / n

	summary.Extra["total_samples"] = len(results)
	summary.Extra["success_count"] = successCount
	summary.Extra["excellent_count"] = excellentCount

	return summary
}
