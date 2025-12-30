package evaluation

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/easyops/helloagents-go/pkg/core/llm"
	"github.com/easyops/helloagents-go/pkg/evaluation"
	"github.com/easyops/helloagents-go/pkg/evaluation/benchmarks/datagen"
	"github.com/easyops/helloagents-go/pkg/tools"
)

// WinRateTool Win Rate 评估工具
type WinRateTool struct {
	// llmProvider LLM 提供商
	llmProvider llm.Provider

	// outputDir 输出目录
	outputDir string
}

// NewWinRateTool 创建 Win Rate 工具
//
// 参数:
//   - llmProvider: LLM 服务提供商
//   - outputDir: 评估结果输出目录
func NewWinRateTool(llmProvider llm.Provider, outputDir string) *WinRateTool {
	return &WinRateTool{
		llmProvider: llmProvider,
		outputDir:   outputDir,
	}
}

// Name 返回工具名称
func (t *WinRateTool) Name() string {
	return "win_rate"
}

// Description 返回工具描述
func (t *WinRateTool) Description() string {
	return "Win Rate 评估工具。通过成对对比计算胜率，评估生成数据相对于参考数据的质量。"
}

// Parameters 返回参数 Schema
func (t *WinRateTool) Parameters() tools.ParameterSchema {
	return tools.ParameterSchema{
		Type: "object",
		Properties: map[string]tools.PropertySchema{
			"candidate_path": {
				Type:        "string",
				Description: "候选数据文件路径（JSONL 格式）",
			},
			"reference_path": {
				Type:        "string",
				Description: "参考数据文件路径（JSONL 格式）",
			},
			"max_samples": {
				Type:        "integer",
				Description: "最大对比样本数（0 表示全部）",
				Default:     0,
			},
			"random_seed": {
				Type:        "integer",
				Description: "随机种子（用于位置随机化）",
				Default:     0,
			},
		},
		Required: []string{"candidate_path", "reference_path"},
	}
}

// Execute 执行评估
func (t *WinRateTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	// 解析参数
	candidatePath, ok := args["candidate_path"].(string)
	if !ok || candidatePath == "" {
		return "", fmt.Errorf("candidate_path 参数是必需的")
	}

	referencePath, ok := args["reference_path"].(string)
	if !ok || referencePath == "" {
		return "", fmt.Errorf("reference_path 参数是必需的")
	}

	maxSamples := 0
	if v, ok := args["max_samples"].(float64); ok {
		maxSamples = int(v)
	}

	randomSeed := int64(0)
	if v, ok := args["random_seed"].(float64); ok {
		randomSeed = int64(v)
	}

	// 创建数据集
	candidateDataset := datagen.NewDataset(candidatePath)
	if err := candidateDataset.Load(ctx); err != nil {
		return "", fmt.Errorf("加载候选数据集失败: %w", err)
	}

	referenceDataset := datagen.NewDataset(referencePath)
	if err := referenceDataset.Load(ctx); err != nil {
		return "", fmt.Errorf("加载参考数据集失败: %w", err)
	}

	// 创建评估器
	config := datagen.WinRateConfig{
		RandomSeed: randomSeed,
	}
	evaluator := datagen.NewWinRateEvaluator(t.llmProvider, candidateDataset, referenceDataset, config)

	// 配置评估选项
	opts := []evaluation.EvalOption{
		evaluation.WithVerbose(true),
	}
	if maxSamples > 0 {
		opts = append(opts, evaluation.WithMaxSamples(maxSamples))
	}

	// 执行评估
	result, err := evaluator.Evaluate(ctx, opts...)
	if err != nil {
		return "", fmt.Errorf("评估失败: %w", err)
	}

	// 生成输出文件名
	timestamp := time.Now().Format("20060102_150405")
	baseName := fmt.Sprintf("win_rate_%s", timestamp)

	// 导出报告
	exporter := datagen.NewExporter()
	reportPath := filepath.Join(t.outputDir, baseName+"_report.md")
	if err := exporter.ExportWinRateReport(result, reportPath); err != nil {
		return "", fmt.Errorf("导出报告失败: %w", err)
	}

	// 导出 JSON 结果
	jsonPath := filepath.Join(t.outputDir, baseName+"_result.json")
	if err := exporter.ExportJSON(result, jsonPath); err != nil {
		return "", fmt.Errorf("导出 JSON 失败: %w", err)
	}

	// 构建响应
	response := map[string]interface{}{
		"status":          "success",
		"total_comparisons": result.TotalSamples,
		"duration":        result.TotalDuration.String(),
		"report_path":     reportPath,
		"result_path":     jsonPath,
		"evaluation_time": result.EvaluationTime.Format("2006-01-02 15:04:05"),
	}

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

		response["wins"] = wins
		response["losses"] = losses
		response["ties"] = ties
		response["win_rate"] = fmt.Sprintf("%.2f%%", result.Metrics.WinRate*100)
		response["loss_rate"] = fmt.Sprintf("%.2f%%", result.Metrics.LossRate*100)
		response["tie_rate"] = fmt.Sprintf("%.2f%%", result.Metrics.TieRate*100)

		// 结论
		if result.Metrics.WinRate > 0.6 {
			response["conclusion"] = "候选数据集显著优于参考数据集"
		} else if result.Metrics.WinRate > 0.4 {
			response["conclusion"] = "候选数据集与参考数据集质量相当"
		} else {
			response["conclusion"] = "候选数据集不及参考数据集"
		}
	}

	jsonBytes, _ := json.MarshalIndent(response, "", "  ")
	return string(jsonBytes), nil
}
