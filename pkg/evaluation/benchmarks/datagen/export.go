package datagen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ahhsitt/helloagents-go/pkg/evaluation"
)

// Exporter 数据生成评估结果导出器
type Exporter struct{}

// NewExporter 创建导出器
func NewExporter() *Exporter {
	return &Exporter{}
}

// ExportJudgeReport 导出 LLM Judge 报告
func (e *Exporter) ExportJudgeReport(result *evaluation.EvalResult, outputPath string) error {
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
	fmt.Fprintf(file, "# LLM Judge 评估报告\n\n")
	fmt.Fprintf(file, "## 概览\n\n")
	fmt.Fprintf(file, "- **评估器**: %s\n", result.BenchmarkName)
	fmt.Fprintf(file, "- **LLM**: %s\n", result.AgentName)
	fmt.Fprintf(file, "- **评估时间**: %s\n", result.EvaluationTime.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "- **总耗时**: %s\n\n", result.TotalDuration)

	// 总体指标
	fmt.Fprintf(file, "## 总体指标\n\n")
	fmt.Fprintf(file, "| 指标 | 值 |\n")
	fmt.Fprintf(file, "|------|----|\n")
	fmt.Fprintf(file, "| 总样本数 | %d |\n", result.TotalSamples)
	fmt.Fprintf(file, "| 通过数 | %d |\n", result.SuccessCount)

	if result.Metrics != nil {
		fmt.Fprintf(file, "| 平均分 | %.2f |\n", result.Metrics.AverageScore)
		fmt.Fprintf(file, "| 通过率 | %.2f%% |\n", result.Metrics.PassRate*100)
		fmt.Fprintf(file, "| 优秀率 | %.2f%% |\n", result.Metrics.ExcellentRate*100)
	}
	fmt.Fprintf(file, "\n")

	// 各维度评分
	if result.Metrics != nil && len(result.Metrics.DimensionScores) > 0 {
		fmt.Fprintf(file, "## 各维度评分\n\n")
		fmt.Fprintf(file, "| 维度 | 平均分 |\n")
		fmt.Fprintf(file, "|------|--------|\n")
		dimensionNames := map[string]string{
			"correctness":      "正确性",
			"clarity":          "清晰度",
			"difficulty_match": "难度匹配",
			"completeness":     "完整性",
		}
		for dim, score := range result.Metrics.DimensionScores {
			name := dimensionNames[dim]
			if name == "" {
				name = dim
			}
			fmt.Fprintf(file, "| %s | %.2f |\n", name, score)
		}
		fmt.Fprintf(file, "\n")
	}

	// 低分样本
	var lowScoreSamples []*evaluation.SampleResult
	for _, sr := range result.DetailedResults {
		if sr.Score < 3.0 {
			lowScoreSamples = append(lowScoreSamples, sr)
		}
	}

	if len(lowScoreSamples) > 0 {
		fmt.Fprintf(file, "## 低分样本（得分 < 3.0）\n\n")
		maxShow := 10
		if len(lowScoreSamples) < maxShow {
			maxShow = len(lowScoreSamples)
		}
		for i := 0; i < maxShow; i++ {
			sr := lowScoreSamples[i]
			fmt.Fprintf(file, "### 样本: %s (得分: %.2f)\n\n", sr.SampleID, sr.Score)
			if sr.Details != nil {
				if comments, ok := sr.Details["comments"].(string); ok && comments != "" {
					fmt.Fprintf(file, "**评语**: %s\n\n", comments)
				}
			}
			fmt.Fprintf(file, "---\n\n")
		}
	}

	return nil
}

// ExportWinRateReport 导出 Win Rate 报告
func (e *Exporter) ExportWinRateReport(result *evaluation.EvalResult, outputPath string) error {
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
	fmt.Fprintf(file, "# Win Rate 评估报告\n\n")
	fmt.Fprintf(file, "## 概览\n\n")
	fmt.Fprintf(file, "- **评估器**: %s\n", result.BenchmarkName)
	fmt.Fprintf(file, "- **LLM**: %s\n", result.AgentName)
	fmt.Fprintf(file, "- **评估时间**: %s\n", result.EvaluationTime.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "- **总耗时**: %s\n\n", result.TotalDuration)

	// 胜率统计
	fmt.Fprintf(file, "## 胜率统计\n\n")
	if result.Metrics != nil {
		wins := 0
		losses := 0
		ties := 0
		if v, ok := result.Metrics.Extra["wins"].(int); ok {
			wins = v
		}
		if v, ok := result.Metrics.Extra["losses"].(int); ok {
			losses = v
		}
		if v, ok := result.Metrics.Extra["ties"].(int); ok {
			ties = v
		}

		fmt.Fprintf(file, "| 结果 | 数量 | 比例 |\n")
		fmt.Fprintf(file, "|------|------|------|\n")
		fmt.Fprintf(file, "| 胜 | %d | %.2f%% |\n", wins, result.Metrics.WinRate*100)
		fmt.Fprintf(file, "| 负 | %d | %.2f%% |\n", losses, result.Metrics.LossRate*100)
		fmt.Fprintf(file, "| 平 | %d | %.2f%% |\n", ties, result.Metrics.TieRate*100)
		fmt.Fprintf(file, "\n")
	}

	// 结论
	fmt.Fprintf(file, "## 结论\n\n")
	if result.Metrics != nil {
		if result.Metrics.WinRate > 0.6 {
			fmt.Fprintf(file, "候选数据集**显著优于**参考数据集（胜率 > 60%%）\n")
		} else if result.Metrics.WinRate > 0.4 {
			fmt.Fprintf(file, "候选数据集与参考数据集**质量相当**（胜率在 40-60%% 之间）\n")
		} else {
			fmt.Fprintf(file, "候选数据集**不及**参考数据集（胜率 < 40%%）\n")
		}
	}
	fmt.Fprintf(file, "\n")

	// 详细对比（前 10 个）
	fmt.Fprintf(file, "## 详细对比（前 10 个）\n\n")
	maxShow := 10
	if len(result.DetailedResults) < maxShow {
		maxShow = len(result.DetailedResults)
	}
	for i := 0; i < maxShow; i++ {
		sr := result.DetailedResults[i]
		fmt.Fprintf(file, "### 对比 #%d\n\n", i+1)
		if sr.Details != nil {
			if winner, ok := sr.Details["actual_winner"].(string); ok {
				fmt.Fprintf(file, "**胜者**: %s\n", winner)
			}
			if reason, ok := sr.Details["reason"].(string); ok && reason != "" {
				fmt.Fprintf(file, "**理由**: %s\n", reason)
			}
		}
		fmt.Fprintf(file, "\n---\n\n")
	}

	return nil
}

// ExportJSON 导出 JSON 格式结果
func (e *Exporter) ExportJSON(result *evaluation.EvalResult, outputPath string) error {
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
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
