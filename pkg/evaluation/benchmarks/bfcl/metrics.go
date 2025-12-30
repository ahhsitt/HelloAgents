package bfcl

import (
	"github.com/easyops/helloagents-go/pkg/evaluation"
)

// Metrics BFCL 指标计算器
type Metrics struct{}

// NewMetrics 创建 BFCL 指标计算器
func NewMetrics() *Metrics {
	return &Metrics{}
}

// Compute 计算 BFCL 评估指标
func (m *Metrics) Compute(results []*evaluation.SampleResult) *evaluation.MetricsSummary {
	summary := &evaluation.MetricsSummary{
		Extra: make(map[string]interface{}),
	}

	if len(results) == 0 {
		return summary
	}

	// 基础统计
	totalSamples := len(results)
	successCount := 0
	totalScore := 0.0
	errorCount := 0

	// 函数调用级别统计
	totalExpectedCalls := 0
	totalPredictedCalls := 0
	correctCalls := 0

	for _, r := range results {
		if r.Success {
			successCount++
		}
		totalScore += r.Score

		if r.Error != "" {
			errorCount++
		}

		// 提取详细信息用于计算精确率/召回率
		if details := r.Details; details != nil {
			if ec, ok := details["expected_count"].(int); ok {
				totalExpectedCalls += ec
			}
			if mc, ok := details["matched_count"].(int); ok {
				correctCalls += mc
			}
			if pc, ok := details["predicted_calls"].([]evaluation.FunctionCall); ok {
				totalPredictedCalls += len(pc)
			}
		}
	}

	// 计算准确率
	summary.Accuracy = float64(successCount) / float64(totalSamples)
	summary.AverageScore = totalScore / float64(totalSamples)

	// 计算精确率和召回率
	if totalPredictedCalls > 0 {
		summary.Precision = float64(correctCalls) / float64(totalPredictedCalls)
	}
	if totalExpectedCalls > 0 {
		summary.Recall = float64(correctCalls) / float64(totalExpectedCalls)
	}

	// 计算 F1 分数
	if summary.Precision+summary.Recall > 0 {
		summary.F1Score = 2 * summary.Precision * summary.Recall / (summary.Precision + summary.Recall)
	}

	// 额外指标
	summary.Extra["total_samples"] = totalSamples
	summary.Extra["success_count"] = successCount
	summary.Extra["error_count"] = errorCount
	summary.Extra["total_expected_calls"] = totalExpectedCalls
	summary.Extra["total_predicted_calls"] = totalPredictedCalls
	summary.Extra["correct_calls"] = correctCalls

	return summary
}

// ComputeCategoryMetrics 计算分类别指标
func (m *Metrics) ComputeCategoryMetrics(results []*evaluation.SampleResult) map[string]*evaluation.CategoryMetrics {
	categoryMetrics := make(map[string]*evaluation.CategoryMetrics)

	for _, r := range results {
		cat := r.Category
		if cat == "" {
			cat = "default"
		}

		if _, ok := categoryMetrics[cat]; !ok {
			categoryMetrics[cat] = &evaluation.CategoryMetrics{
				Category: cat,
			}
		}

		cm := categoryMetrics[cat]
		cm.Total++
		if r.Success {
			cm.Success++
		}
		cm.AverageScore += r.Score
	}

	// 计算每个类别的平均值
	for _, cm := range categoryMetrics {
		if cm.Total > 0 {
			cm.Accuracy = float64(cm.Success) / float64(cm.Total)
			cm.AverageScore = cm.AverageScore / float64(cm.Total)
		}
	}

	return categoryMetrics
}
