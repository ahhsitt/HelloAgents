// Package evaluation 提供评估相关的工具实现
package evaluation

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/easyops/helloagents-go/pkg/agents"
	"github.com/easyops/helloagents-go/pkg/evaluation"
	"github.com/easyops/helloagents-go/pkg/evaluation/benchmarks/bfcl"
	"github.com/easyops/helloagents-go/pkg/tools"
)

// BFCLEvaluationTool BFCL 一键评估工具
type BFCLEvaluationTool struct {
	// bfclDataDir BFCL 数据目录
	bfclDataDir string

	// outputDir 输出目录
	outputDir string

	// agent 待评估的智能体
	agent agents.Agent
}

// NewBFCLEvaluationTool 创建 BFCL 评估工具
//
// 参数:
//   - bfclDataDir: BFCL 数据目录路径
//   - outputDir: 评估结果输出目录
//   - agent: 待评估的智能体
func NewBFCLEvaluationTool(bfclDataDir, outputDir string, agent agents.Agent) *BFCLEvaluationTool {
	return &BFCLEvaluationTool{
		bfclDataDir: bfclDataDir,
		outputDir:   outputDir,
		agent:       agent,
	}
}

// Name 返回工具名称
func (t *BFCLEvaluationTool) Name() string {
	return "bfcl_evaluation"
}

// Description 返回工具描述
func (t *BFCLEvaluationTool) Description() string {
	return "BFCL (Berkeley Function Calling Leaderboard) 一键评估工具。支持评估智能体的函数调用能力。"
}

// Parameters 返回参数 Schema
func (t *BFCLEvaluationTool) Parameters() tools.ParameterSchema {
	return tools.ParameterSchema{
		Type: "object",
		Properties: map[string]tools.PropertySchema{
			"category": {
				Type:        "string",
				Description: "评估类别，如 simple_python, multiple, parallel 等",
				Enum:        bfcl.SupportedCategories,
			},
			"max_samples": {
				Type:        "integer",
				Description: "最大评估样本数（0 表示全部）",
				Default:     0,
			},
			"evaluation_mode": {
				Type:        "string",
				Description: "评估模式：ast（AST 匹配）或 execution（执行评估）",
				Enum:        []string{"ast", "execution"},
				Default:     "ast",
			},
			"export_official": {
				Type:        "boolean",
				Description: "是否导出 BFCL 官方格式",
				Default:     true,
			},
		},
		Required: []string{"category"},
	}
}

// Execute 执行评估
func (t *BFCLEvaluationTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	// 解析参数
	category, _ := args["category"].(string)
	if category == "" {
		return "", fmt.Errorf("category 参数是必需的")
	}

	maxSamples := 0
	if v, ok := args["max_samples"].(float64); ok {
		maxSamples = int(v)
	}

	evalMode := bfcl.ModeAST
	if mode, ok := args["evaluation_mode"].(string); ok && mode == "execution" {
		evalMode = bfcl.ModeExecution
	}

	exportOfficial := true
	if v, ok := args["export_official"].(bool); ok {
		exportOfficial = v
	}

	// 创建数据集
	dataset := bfcl.NewDataset(t.bfclDataDir, category)

	// 加载数据集
	if err := dataset.Load(ctx); err != nil {
		return "", fmt.Errorf("加载数据集失败: %w", err)
	}

	// 创建评估器
	evaluator := bfcl.NewEvaluator(dataset, evalMode)

	// 配置评估选项
	opts := []evaluation.EvalOption{
		evaluation.WithVerbose(true),
	}
	if maxSamples > 0 {
		opts = append(opts, evaluation.WithMaxSamples(maxSamples))
	}

	// 执行评估
	result, err := evaluator.Evaluate(ctx, t.agent, opts...)
	if err != nil {
		return "", fmt.Errorf("评估失败: %w", err)
	}

	// 生成输出文件名
	timestamp := time.Now().Format("20060102_150405")
	baseName := fmt.Sprintf("bfcl_%s_%s", category, timestamp)

	// 导出 BFCL 官方格式
	if exportOfficial {
		exporter := bfcl.NewExporter(true)
		officialPath := filepath.Join(t.outputDir, baseName+"_official.jsonl")
		if err := exporter.Export(result, officialPath); err != nil {
			return "", fmt.Errorf("导出官方格式失败: %w", err)
		}
	}

	// 导出 Markdown 报告
	exporter := bfcl.NewExporter(false)
	reportPath := filepath.Join(t.outputDir, baseName+"_report.md")
	if err := exporter.ExportMarkdownReport(result, reportPath); err != nil {
		return "", fmt.Errorf("导出报告失败: %w", err)
	}

	// 构建响应
	response := map[string]interface{}{
		"status":           "success",
		"category":         category,
		"total_samples":    result.TotalSamples,
		"success_count":    result.SuccessCount,
		"accuracy":         fmt.Sprintf("%.2f%%", result.OverallAccuracy*100),
		"duration":         result.TotalDuration.String(),
		"report_path":      reportPath,
		"evaluation_time":  result.EvaluationTime.Format("2006-01-02 15:04:05"),
	}

	if result.Metrics != nil {
		response["precision"] = fmt.Sprintf("%.2f%%", result.Metrics.Precision*100)
		response["recall"] = fmt.Sprintf("%.2f%%", result.Metrics.Recall*100)
		response["f1_score"] = fmt.Sprintf("%.2f%%", result.Metrics.F1Score*100)
	}

	jsonBytes, _ := json.MarshalIndent(response, "", "  ")
	return string(jsonBytes), nil
}
