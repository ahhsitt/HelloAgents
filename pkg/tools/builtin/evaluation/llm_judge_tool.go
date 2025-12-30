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

// LLMJudgeTool LLM Judge 评估工具
type LLMJudgeTool struct {
	// llmProvider LLM 提供商
	llmProvider llm.Provider

	// outputDir 输出目录
	outputDir string
}

// NewLLMJudgeTool 创建 LLM Judge 工具
//
// 参数:
//   - llmProvider: LLM 服务提供商
//   - outputDir: 评估结果输出目录
func NewLLMJudgeTool(llmProvider llm.Provider, outputDir string) *LLMJudgeTool {
	return &LLMJudgeTool{
		llmProvider: llmProvider,
		outputDir:   outputDir,
	}
}

// Name 返回工具名称
func (t *LLMJudgeTool) Name() string {
	return "llm_judge"
}

// Description 返回工具描述
func (t *LLMJudgeTool) Description() string {
	return "LLM Judge 评估工具。使用 LLM 作为评委对生成的数据进行多维度质量评估，包括正确性、清晰度、难度匹配和完整性。"
}

// Parameters 返回参数 Schema
func (t *LLMJudgeTool) Parameters() tools.ParameterSchema {
	return tools.ParameterSchema{
		Type: "object",
		Properties: map[string]tools.PropertySchema{
			"data_path": {
				Type:        "string",
				Description: "待评估数据文件路径（JSONL 格式）",
			},
			"reference_path": {
				Type:        "string",
				Description: "参考数据文件路径（可选，用于对比评估）",
			},
			"max_samples": {
				Type:        "integer",
				Description: "最大评估样本数（0 表示全部）",
				Default:     0,
			},
		},
		Required: []string{"data_path"},
	}
}

// Execute 执行评估
func (t *LLMJudgeTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	// 解析参数
	dataPath, ok := args["data_path"].(string)
	if !ok || dataPath == "" {
		return "", fmt.Errorf("data_path 参数是必需的")
	}

	referencePath, _ := args["reference_path"].(string)

	maxSamples := 0
	if v, ok := args["max_samples"].(float64); ok {
		maxSamples = int(v)
	}

	// 创建数据集
	dataset := datagen.NewDataset(dataPath)
	if err := dataset.Load(ctx); err != nil {
		return "", fmt.Errorf("加载数据集失败: %w", err)
	}

	// 加载参考数据（如果有）
	config := datagen.JudgeConfig{}
	if referencePath != "" {
		refDataset := datagen.NewDataset(referencePath)
		if err := refDataset.Load(ctx); err != nil {
			return "", fmt.Errorf("加载参考数据集失败: %w", err)
		}
		config.ReferenceSamples = refDataset.GetSamples()
	}

	// 创建评估器
	judge := datagen.NewLLMJudge(t.llmProvider, dataset, config)

	// 配置评估选项
	opts := []evaluation.EvalOption{
		evaluation.WithVerbose(true),
	}
	if maxSamples > 0 {
		opts = append(opts, evaluation.WithMaxSamples(maxSamples))
	}

	// 执行评估
	result, err := judge.Evaluate(ctx, opts...)
	if err != nil {
		return "", fmt.Errorf("评估失败: %w", err)
	}

	// 生成输出文件名
	timestamp := time.Now().Format("20060102_150405")
	baseName := fmt.Sprintf("llm_judge_%s", timestamp)

	// 导出报告
	exporter := datagen.NewExporter()
	reportPath := filepath.Join(t.outputDir, baseName+"_report.md")
	if err := exporter.ExportJudgeReport(result, reportPath); err != nil {
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
		"total_samples":   result.TotalSamples,
		"pass_count":      result.SuccessCount,
		"duration":        result.TotalDuration.String(),
		"report_path":     reportPath,
		"result_path":     jsonPath,
		"evaluation_time": result.EvaluationTime.Format("2006-01-02 15:04:05"),
	}

	if result.Metrics != nil {
		response["average_score"] = fmt.Sprintf("%.2f", result.Metrics.AverageScore)
		response["pass_rate"] = fmt.Sprintf("%.2f%%", result.Metrics.PassRate*100)
		response["excellent_rate"] = fmt.Sprintf("%.2f%%", result.Metrics.ExcellentRate*100)

		if len(result.Metrics.DimensionScores) > 0 {
			dimensions := make(map[string]string)
			for dim, score := range result.Metrics.DimensionScores {
				dimensions[dim] = fmt.Sprintf("%.2f", score)
			}
			response["dimension_scores"] = dimensions
		}
	}

	jsonBytes, _ := json.MarshalIndent(response, "", "  ")
	return string(jsonBytes), nil
}
