package evaluation

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/ahhsitt/helloagents-go/pkg/agents"
	"github.com/ahhsitt/helloagents-go/pkg/evaluation"
	"github.com/ahhsitt/helloagents-go/pkg/evaluation/benchmarks/gaia"
	"github.com/ahhsitt/helloagents-go/pkg/tools"
)

// GAIAEvaluationTool GAIA 一键评估工具
type GAIAEvaluationTool struct {
	// dataDir GAIA 数据目录
	dataDir string

	// outputDir 输出目录
	outputDir string

	// agent 待评估的智能体
	agent agents.Agent
}

// NewGAIAEvaluationTool 创建 GAIA 评估工具
//
// 参数:
//   - dataDir: GAIA 数据目录路径
//   - outputDir: 评估结果输出目录
//   - agent: 待评估的智能体
func NewGAIAEvaluationTool(dataDir, outputDir string, agent agents.Agent) *GAIAEvaluationTool {
	return &GAIAEvaluationTool{
		dataDir:   dataDir,
		outputDir: outputDir,
		agent:     agent,
	}
}

// Name 返回工具名称
func (t *GAIAEvaluationTool) Name() string {
	return "gaia_evaluation"
}

// Description 返回工具描述
func (t *GAIAEvaluationTool) Description() string {
	return "GAIA (General AI Assistants) 一键评估工具。支持评估智能体的通用 AI 助手能力，包含三个难度级别。"
}

// Parameters 返回参数 Schema
func (t *GAIAEvaluationTool) Parameters() tools.ParameterSchema {
	return tools.ParameterSchema{
		Type: "object",
		Properties: map[string]tools.PropertySchema{
			"level": {
				Type:        "integer",
				Description: "难度级别过滤（1、2、3），0 表示全部",
				Default:     0,
			},
			"split": {
				Type:        "string",
				Description: "数据集分割：validation 或 test",
				Enum:        []string{"validation", "test"},
				Default:     "validation",
			},
			"max_samples": {
				Type:        "integer",
				Description: "最大评估样本数（0 表示全部）",
				Default:     0,
			},
		},
	}
}

// Execute 执行评估
func (t *GAIAEvaluationTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	// 解析参数
	level := 0
	if v, ok := args["level"].(float64); ok {
		level = int(v)
	}

	split := "validation"
	if v, ok := args["split"].(string); ok && v != "" {
		split = v
	}

	maxSamples := 0
	if v, ok := args["max_samples"].(float64); ok {
		maxSamples = int(v)
	}

	// 创建数据集
	dataset := gaia.NewDataset(t.dataDir, level, split)

	// 加载数据集
	if err := dataset.Load(ctx); err != nil {
		return "", fmt.Errorf("加载数据集失败: %w", err)
	}

	// 创建评估器
	evaluator := gaia.NewEvaluator(dataset)

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
	baseName := fmt.Sprintf("gaia_%s_level%d_%s", split, level, timestamp)

	// 导出 GAIA 官方格式
	exporter := gaia.NewExporter()
	officialPath := filepath.Join(t.outputDir, baseName+"_submission.jsonl")
	if err := exporter.Export(result, officialPath); err != nil {
		return "", fmt.Errorf("导出官方格式失败: %w", err)
	}

	// 导出 Markdown 报告
	reportPath := filepath.Join(t.outputDir, baseName+"_report.md")
	if err := exporter.ExportMarkdownReport(result, reportPath); err != nil {
		return "", fmt.Errorf("导出报告失败: %w", err)
	}

	// 构建响应
	response := map[string]interface{}{
		"status":          "success",
		"level":           level,
		"split":           split,
		"total_samples":   result.TotalSamples,
		"success_count":   result.SuccessCount,
		"accuracy":        fmt.Sprintf("%.2f%%", result.OverallAccuracy*100),
		"duration":        result.TotalDuration.String(),
		"report_path":     reportPath,
		"submission_path": officialPath,
		"evaluation_time": result.EvaluationTime.Format("2006-01-02 15:04:05"),
	}

	// 添加级别分布
	if len(result.LevelMetrics) > 0 {
		levelResults := make(map[string]interface{})
		for lvl, lm := range result.LevelMetrics {
			levelResults[fmt.Sprintf("level_%d", lvl)] = map[string]interface{}{
				"total":            lm.Total,
				"exact_matches":    lm.ExactMatches,
				"exact_match_rate": fmt.Sprintf("%.2f%%", lm.ExactMatchRate*100),
			}
		}
		response["level_results"] = levelResults
	}

	jsonBytes, _ := json.MarshalIndent(response, "", "  ")
	return string(jsonBytes), nil
}

// GetDatasetInfo 获取数据集信息
func (t *GAIAEvaluationTool) GetDatasetInfo(ctx context.Context, level int, split string) (map[string]interface{}, error) {
	dataset := gaia.NewDataset(t.dataDir, level, split)
	if err := dataset.Load(ctx); err != nil {
		return nil, err
	}

	info := map[string]interface{}{
		"name":         dataset.Name(),
		"total":        dataset.Len(),
		"level_filter": level,
		"split":        split,
	}

	// 获取级别分布
	dist := dataset.GetLevelDistribution()
	levelDist := make(map[string]int)
	for lvl, count := range dist {
		levelDist[fmt.Sprintf("level_%d", lvl)] = count
	}
	info["level_distribution"] = levelDist

	return info, nil
}
