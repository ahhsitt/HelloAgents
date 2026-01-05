package bfcl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ahhsitt/helloagents-go/pkg/evaluation"
)

// ExportEntry BFCL 导出条目
type ExportEntry struct {
	ID           string        `json:"id"`
	Result       interface{}   `json:"result"`
	InferenceLog []interface{} `json:"inference_log,omitempty"`
}

// Exporter BFCL 结果导出器
type Exporter struct {
	// includeInferenceLog 是否包含推理日志
	includeInferenceLog bool
}

// NewExporter 创建导出器
//
// 参数:
//   - includeInferenceLog: 是否包含推理日志
func NewExporter(includeInferenceLog bool) *Exporter {
	return &Exporter{
		includeInferenceLog: includeInferenceLog,
	}
}

// Export 导出评估结果为 BFCL 官方格式
//
// 输出 JSONL 格式，每行一个 JSON 对象
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
			ID:     sr.SampleID,
			Result: e.formatResult(sr),
		}

		if e.includeInferenceLog && sr.AgentResponse != "" {
			entry.InferenceLog = e.buildInferenceLog(sr)
		}

		if err := encoder.Encode(entry); err != nil {
			return fmt.Errorf("写入条目失败: %w", err)
		}
	}

	return nil
}

// formatResult 格式化结果
func (e *Exporter) formatResult(sr *evaluation.SampleResult) interface{} {
	// 如果预测结果是函数调用列表，直接返回
	if calls, ok := sr.Predicted.([]evaluation.FunctionCall); ok {
		return e.convertCallsToOutput(calls)
	}

	// 否则返回原始响应
	if sr.AgentResponse != "" {
		return sr.AgentResponse
	}

	return sr.Predicted
}

// convertCallsToOutput 将函数调用转换为 BFCL 输出格式
func (e *Exporter) convertCallsToOutput(calls []evaluation.FunctionCall) []map[string]interface{} {
	output := make([]map[string]interface{}, len(calls))
	for i, call := range calls {
		output[i] = map[string]interface{}{
			"name":      call.Name,
			"arguments": call.Arguments,
		}
	}
	return output
}

// buildInferenceLog 构建推理日志
func (e *Exporter) buildInferenceLog(sr *evaluation.SampleResult) []interface{} {
	log := make([]interface{}, 0, 2)

	// 用户消息
	if input, ok := sr.Details["input"].(string); ok && input != "" {
		log = append(log, map[string]interface{}{
			"role":    "user",
			"content": input,
		})
	}

	// 助手响应
	if sr.AgentResponse != "" {
		log = append(log, map[string]interface{}{
			"role":    "assistant",
			"content": sr.AgentResponse,
		})
	}

	return log
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
	fmt.Fprintf(file, "# BFCL 评估报告\n\n")
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

	if result.Metrics != nil {
		if result.Metrics.Precision > 0 {
			fmt.Fprintf(file, "| 精确率 | %.2f%% |\n", result.Metrics.Precision*100)
		}
		if result.Metrics.Recall > 0 {
			fmt.Fprintf(file, "| 召回率 | %.2f%% |\n", result.Metrics.Recall*100)
		}
		if result.Metrics.F1Score > 0 {
			fmt.Fprintf(file, "| F1 分数 | %.2f%% |\n", result.Metrics.F1Score*100)
		}
	}
	fmt.Fprintf(file, "\n")

	// 分类别指标
	if len(result.CategoryMetrics) > 0 {
		fmt.Fprintf(file, "## 分类别指标\n\n")
		fmt.Fprintf(file, "| 类别 | 总数 | 成功数 | 准确率 |\n")
		fmt.Fprintf(file, "|------|------|--------|--------|\n")
		for cat, metrics := range result.CategoryMetrics {
			fmt.Fprintf(file, "| %s | %d | %d | %.2f%% |\n",
				cat, metrics.Total, metrics.Success, metrics.Accuracy*100)
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
			fmt.Fprintf(file, "### 样本: %s\n\n", sr.SampleID)
			if sr.Error != "" {
				fmt.Fprintf(file, "**错误**: %s\n\n", sr.Error)
			}
			if sr.Details != nil {
				if reason, ok := sr.Details["reason"].(string); ok {
					fmt.Fprintf(file, "**原因**: %s\n\n", reason)
				}
			}
			fmt.Fprintf(file, "---\n\n")
		}
	}

	return nil
}
