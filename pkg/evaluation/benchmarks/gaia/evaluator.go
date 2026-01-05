package gaia

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/ahhsitt/helloagents-go/pkg/agents"
	"github.com/ahhsitt/helloagents-go/pkg/evaluation"
)

// Evaluator GAIA 评估器
type Evaluator struct {
	// dataset 数据集
	dataset *Dataset
}

// NewEvaluator 创建 GAIA 评估器
func NewEvaluator(dataset *Dataset) *Evaluator {
	return &Evaluator{
		dataset: dataset,
	}
}

// Name 返回评估器名称
func (e *Evaluator) Name() string {
	return e.dataset.Name()
}

// Evaluate 执行完整评估
func (e *Evaluator) Evaluate(ctx context.Context, agent agents.Agent, opts ...evaluation.EvalOption) (*evaluation.EvalResult, error) {
	config := evaluation.DefaultEvalConfig()
	config.ApplyOptions(opts...)

	// 确保数据集已加载
	if err := e.dataset.Load(ctx); err != nil {
		return nil, fmt.Errorf("加载数据集失败: %w", err)
	}

	startTime := time.Now()
	result := &evaluation.EvalResult{
		BenchmarkName:   e.Name(),
		AgentName:       agent.Name(),
		DetailedResults: make([]*evaluation.SampleResult, 0),
		LevelMetrics:    make(map[int]*evaluation.LevelMetrics),
		EvaluationTime:  startTime,
	}

	total := e.dataset.Len()
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

		sample, err := e.dataset.Get(i)
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

		sampleResult, err := e.EvaluateSample(evalCtx, agent, sample)
		if err != nil {
			sampleResult = &evaluation.SampleResult{
				SampleID: sample.ID,
				Level:    sample.Level,
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

	// 计算级别指标
	e.computeLevelMetrics(result)

	// 计算汇总指标
	metrics := NewMetrics()
	result.Metrics = metrics.Compute(result.DetailedResults)

	return result, nil
}

// EvaluateSample 评估单个样本
func (e *Evaluator) EvaluateSample(ctx context.Context, agent agents.Agent, sample evaluation.Sample) (*evaluation.SampleResult, error) {
	startTime := time.Now()

	result := &evaluation.SampleResult{
		SampleID: sample.ID,
		Level:    sample.Level,
		Category: sample.Category,
		Expected: sample.Expected,
		Details:  make(map[string]interface{}),
	}

	// 构建输入
	input := agents.Input{
		Query: sample.Input,
		Context: map[string]interface{}{
			"files": sample.Files,
		},
	}

	// 调用智能体
	output, err := agent.Run(ctx, input)
	if err != nil {
		result.Error = err.Error()
		result.ExecutionTime = time.Since(startTime)
		return result, nil
	}

	result.AgentResponse = output.Response
	result.ExecutionTime = time.Since(startTime)

	// 从响应中提取答案
	predictedAnswer := e.extractAnswer(output.Response)
	result.Predicted = predictedAnswer
	result.Details["extracted_answer"] = predictedAnswer

	// 获取期望答案
	expectedAnswer, ok := sample.Expected.(string)
	if !ok {
		result.Error = "期望答案格式错误"
		return result, nil
	}

	// 评估匹配
	exactMatch, partialMatch := e.evaluateMatch(predictedAnswer, expectedAnswer)
	result.Success = exactMatch
	result.PartialSuccess = partialMatch

	if exactMatch {
		result.Score = 1.0
	} else if partialMatch {
		result.Score = 0.5
	}

	result.Details["exact_match"] = exactMatch
	result.Details["partial_match"] = partialMatch

	return result, nil
}

// extractAnswer 从响应中提取答案
func (e *Evaluator) extractAnswer(response string) string {
	response = strings.TrimSpace(response)
	if response == "" {
		return ""
	}

	// 查找 "FINAL ANSWER: [答案]" 模式
	patterns := []string{
		`(?i)FINAL\s+ANSWER:\s*(.+?)(?:\n|$)`,
		`(?i)答案[：:]\s*(.+?)(?:\n|$)`,
		`(?i)Answer[：:]\s*(.+?)(?:\n|$)`,
		`(?i)The\s+answer\s+is[：:]\s*(.+?)(?:\n|$)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(response)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}

	// 回退：获取最后一个非空行
	lines := strings.Split(response, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			return line
		}
	}

	return response
}

// evaluateMatch 评估答案匹配
func (e *Evaluator) evaluateMatch(predicted, expected string) (exactMatch, partialMatch bool) {
	// 标准化答案
	normalizedPred := normalizeAnswer(predicted)
	normalizedExp := normalizeAnswer(expected)

	// 精确匹配
	if normalizedPred == normalizedExp {
		return true, true
	}

	// 部分匹配检查
	// 1. 包含检查
	if strings.Contains(normalizedPred, normalizedExp) || strings.Contains(normalizedExp, normalizedPred) {
		return false, true
	}

	// 2. 词汇覆盖检查（70% 阈值）
	expectedWords := strings.Fields(normalizedExp)
	if len(expectedWords) > 0 {
		matchedCount := 0
		for _, word := range expectedWords {
			if strings.Contains(normalizedPred, word) {
				matchedCount++
			}
		}
		coverage := float64(matchedCount) / float64(len(expectedWords))
		if coverage >= 0.7 {
			return false, true
		}
	}

	return false, false
}

// normalizeAnswer 标准化答案
func normalizeAnswer(answer string) string {
	// 转为小写
	answer = strings.ToLower(strings.TrimSpace(answer))

	// 移除前导冠词
	articles := []string{"the ", "a ", "an "}
	for _, article := range articles {
		if strings.HasPrefix(answer, article) {
			answer = strings.TrimPrefix(answer, article)
			break
		}
	}

	// 移除尾随标点
	answer = strings.TrimRightFunc(answer, func(r rune) bool {
		return unicode.IsPunct(r)
	})

	// 移除货币符号和百分号
	answer = strings.ReplaceAll(answer, "$", "")
	answer = strings.ReplaceAll(answer, "%", "")
	answer = strings.ReplaceAll(answer, "¥", "")
	answer = strings.ReplaceAll(answer, "€", "")
	answer = strings.ReplaceAll(answer, "£", "")

	// 移除数字中的逗号分隔符
	answer = removeNumberCommas(answer)

	// 规范化空白
	answer = strings.Join(strings.Fields(answer), " ")

	return answer
}

// removeNumberCommas 移除数字中的逗号
func removeNumberCommas(s string) string {
	// 匹配形如 1,000 或 1,000,000 的数字
	re := regexp.MustCompile(`(\d),(\d{3})`)
	for re.MatchString(s) {
		s = re.ReplaceAllString(s, "$1$2")
	}
	return s
}

// computeLevelMetrics 计算级别指标
func (e *Evaluator) computeLevelMetrics(result *evaluation.EvalResult) {
	levelStats := make(map[int]*evaluation.LevelMetrics)

	for _, sr := range result.DetailedResults {
		level := sr.Level
		if level == 0 {
			level = 1 // 默认级别
		}

		if _, ok := levelStats[level]; !ok {
			levelStats[level] = &evaluation.LevelMetrics{
				Level: level,
			}
		}

		levelStats[level].Total++
		if sr.Success {
			levelStats[level].ExactMatches++
		}
		if sr.PartialSuccess {
			levelStats[level].PartialMatches++
		}
	}

	// 计算比率
	for _, stats := range levelStats {
		if stats.Total > 0 {
			stats.ExactMatchRate = float64(stats.ExactMatches) / float64(stats.Total)
			stats.PartialMatchRate = float64(stats.PartialMatches) / float64(stats.Total)
		}
	}

	result.LevelMetrics = levelStats
}
