package gaia

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ahhsitt/helloagents-go/pkg/evaluation"
)

// ExportEntry GAIA 导出条目（官方提交格式）
type ExportEntry struct {
	TaskID      string `json:"task_id"`
	ModelAnswer string `json:"model_answer"`
}

// Exporter GAIA 结果导出器
type Exporter struct{}

// NewExporter 创建导出器
func NewExporter() *Exporter {
	return &Exporter{}
}

// Export 导出评估结果为 GAIA 官方提交格式
func (e *Exporter) Export(result *evaluation.EvalResult, outputPath string) error {
	// 确保目录存在
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)

	for _, sr := range result.DetailedResults {
		entry := ExportEntry{
			TaskID: sr.SampleID,
		}

		// 获取预测答案
		if predicted, ok := sr.Predicted.(string); ok {
			entry.ModelAnswer = predicted
		} else if sr.AgentResponse != "" {
			entry.ModelAnswer = sr.AgentResponse
		}

		if err := encoder.Encode(entry); err != nil {
			return fmt.Errorf("写入条目失败: %w", err)
		}
	}

	return nil
}

// ExportMarkdownReport 导出 Markdown 报告
func (e *Exporter) ExportMarkdownReport(result *evaluation.EvalResult, outputPath string) error {
	// 确保目录存在
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	// 写入报告头
	fmt.Fprintf(file, "# GAIA 评估报告\n\n")
	fmt.Fprintf(file, "## 概览\n\n")
	fmt.Fprintf(file, "- **基准**: %s\n", result.BenchmarkName)
	fmt.Fprintf(file, "- **智能体**: %s\n", result.AgentName)
	fmt.Fprintf(file, "- **评估时间**: %s\n", result.EvaluationTime.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "- **总耗时**: %s\n\n", result.TotalDuration)

	// 总体指标
	fmt.Fprintf(file, "## 总体指标\n\n")
	fmt.Fprintf(file, "| 指标 | 值 |\n")
	fmt.Fprintf(file, "|------|----|\n")
	fmt.Fprintf(file, "| 总样本数 | %d |\n", result.TotalSamples)
	fmt.Fprintf(file, "| 成功数 | %d |\n", result.SuccessCount)
	fmt.Fprintf(file, "| 准确率 | %.2f%% |\n", result.OverallAccuracy*100)

	if result.Metrics != nil && result.Metrics.Extra != nil {
		if partialRate, ok := result.Metrics.Extra["partial_match_rate"].(float64); ok {
			fmt.Fprintf(file, "| 部分匹配率 | %.2f%% |\n", partialRate*100)
		}
	}
	fmt.Fprintf(file, "\n")

	// 分级别指标
	if len(result.LevelMetrics) > 0 {
		fmt.Fprintf(file, "## 分级别指标\n\n")
		fmt.Fprintf(file, "| 级别 | 总数 | 精确匹配 | 精确匹配率 | 部分匹配率 |\n")
		fmt.Fprintf(file, "|------|------|----------|------------|------------|\n")
		for level := 1; level <= 3; level++ {
			if lm, ok := result.LevelMetrics[level]; ok {
				fmt.Fprintf(file, "| Level %d | %d | %d | %.2f%% | %.2f%% |\n",
					level, lm.Total, lm.ExactMatches,
					lm.ExactMatchRate*100, lm.PartialMatchRate*100)
			}
		}
		fmt.Fprintf(file, "\n")

		// 难度分析
		metrics := NewMetrics()
		analysis := metrics.AnalyzeDifficultyProgression(result.LevelMetrics)

		fmt.Fprintf(file, "### 难度递进分析\n\n")
		if drops, ok := analysis["difficulty_drops"].(map[string]float64); ok {
			for transition, drop := range drops {
				fmt.Fprintf(file, "- **%s**: %.2f%% 下降\n", transition, drop*100)
			}
		}
		if pattern, ok := analysis["performance_pattern"].(string); ok {
			fmt.Fprintf(file, "- **性能模式**: %s\n", pattern)
		}
		fmt.Fprintf(file, "\n")
	}

	// 错误样本
	var errorSamples []*evaluation.SampleResult
	for _, sr := range result.DetailedResults {
		if !sr.Success {
			errorSamples = append(errorSamples, sr)
		}
	}

	if len(errorSamples) > 0 {
		fmt.Fprintf(file, "## 失败样本（前 10 个）\n\n")
		maxShow := 10
		if len(errorSamples) < maxShow {
			maxShow = len(errorSamples)
		}
		for i := 0; i < maxShow; i++ {
			sr := errorSamples[i]
			fmt.Fprintf(file, "### 样本: %s (Level %d)\n\n", sr.SampleID, sr.Level)
			if expected, ok := sr.Expected.(string); ok {
				fmt.Fprintf(file, "**期望答案**: %s\n\n", expected)
			}
			if predicted, ok := sr.Predicted.(string); ok {
				fmt.Fprintf(file, "**预测答案**: %s\n\n", predicted)
			}
			if sr.Error != "" {
				fmt.Fprintf(file, "**错误**: %s\n\n", sr.Error)
			}
			fmt.Fprintf(file, "---\n\n")
		}
	}

	// 提交指南
	fmt.Fprintf(file, "## 提交指南\n\n")
	fmt.Fprintf(file, "1. 导出的 JSONL 文件可用于 GAIA 官方评估\n")
	fmt.Fprintf(file, "2. 提交地址: https://huggingface.co/spaces/gaia-benchmark/leaderboard\n")
	fmt.Fprintf(file, "3. 确保每个条目包含 `task_id` 和 `model_answer` 字段\n")

	return nil
}
