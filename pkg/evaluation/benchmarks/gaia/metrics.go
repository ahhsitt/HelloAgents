package gaia

import (
	"github.com/ahhsitt/helloagents-go/pkg/evaluation"
)

// Metrics GAIA 指标计算器
type Metrics struct{}

// NewMetrics 创建 GAIA 指标计算器
func NewMetrics() *Metrics {
	return &Metrics{}
}

// Compute 计算 GAIA 评估指标
func (m *Metrics) Compute(results []*evaluation.SampleResult) *evaluation.MetricsSummary {
	summary := &evaluation.MetricsSummary{
		Extra: make(map[string]interface{}),
	}

	if len(results) == 0 {
		return summary
	}

	totalSamples := len(results)
	exactMatches := 0
	partialMatches := 0
	totalScore := 0.0
	errorCount := 0

	for _, r := range results {
		if r.Success {
			exactMatches++
		}
		if r.PartialSuccess {
			partialMatches++
		}
		totalScore += r.Score

		if r.Error != "" {
			errorCount++
		}
	}

	// 计算准确率
	summary.Accuracy = float64(exactMatches) / float64(totalSamples)
	summary.AverageScore = totalScore / float64(totalSamples)

	// 额外指标
	summary.Extra["total_samples"] = totalSamples
	summary.Extra["exact_matches"] = exactMatches
	summary.Extra["partial_matches"] = partialMatches
	summary.Extra["exact_match_rate"] = float64(exactMatches) / float64(totalSamples)
	summary.Extra["partial_match_rate"] = float64(partialMatches) / float64(totalSamples)
	summary.Extra["error_count"] = errorCount

	return summary
}

// ComputeLevelMetrics 计算分级别指标
func (m *Metrics) ComputeLevelMetrics(results []*evaluation.SampleResult) map[int]*evaluation.LevelMetrics {
	levelMetrics := make(map[int]*evaluation.LevelMetrics)

	for _, r := range results {
		level := r.Level
		if level == 0 {
			level = 1
		}

		if _, ok := levelMetrics[level]; !ok {
			levelMetrics[level] = &evaluation.LevelMetrics{
				Level: level,
			}
		}

		lm := levelMetrics[level]
		lm.Total++
		if r.Success {
			lm.ExactMatches++
		}
		if r.PartialSuccess {
			lm.PartialMatches++
		}
	}

	// 计算每个级别的比率
	for _, lm := range levelMetrics {
		if lm.Total > 0 {
			lm.ExactMatchRate = float64(lm.ExactMatches) / float64(lm.Total)
			lm.PartialMatchRate = float64(lm.PartialMatches) / float64(lm.Total)
		}
	}

	return levelMetrics
}

// AnalyzeDifficultyProgression 分析难度递进性能
func (m *Metrics) AnalyzeDifficultyProgression(levelMetrics map[int]*evaluation.LevelMetrics) map[string]interface{} {
	analysis := make(map[string]interface{})

	// 获取各级别准确率
	rates := make(map[int]float64)
	for level, lm := range levelMetrics {
		rates[level] = lm.ExactMatchRate
	}

	analysis["level_rates"] = rates

	// 计算级别间下降率
	drops := make(map[string]float64)
	if rate1, ok := rates[1]; ok {
		if rate2, ok := rates[2]; ok {
			drops["level1_to_level2"] = rate1 - rate2
		}
		if rate3, ok := rates[3]; ok {
			drops["level1_to_level3"] = rate1 - rate3
		}
	}
	if rate2, ok := rates[2]; ok {
		if rate3, ok := rates[3]; ok {
			drops["level2_to_level3"] = rate2 - rate3
		}
	}

	analysis["difficulty_drops"] = drops

	// 识别性能模式
	var pattern string
	if len(rates) >= 2 {
		if rates[1] > rates[2] && (len(rates) < 3 || rates[2] > rates[3]) {
			pattern = "expected_degradation" // 预期的难度递进下降
		} else if rates[1] < rates[2] {
			pattern = "anomaly_level2_better" // 异常：Level 2 表现更好
		} else {
			pattern = "inconsistent" // 不一致
		}
	}
	analysis["performance_pattern"] = pattern

	return analysis
}
