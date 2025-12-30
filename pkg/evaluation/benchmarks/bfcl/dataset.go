// Package bfcl 实现 Berkeley Function Calling Leaderboard 评估
//
// BFCL 用于评估 Agent 的工具/函数调用能力，支持多种评估类别：
// - simple: 简单函数调用
// - multiple: 多函数调用
// - parallel: 并行函数调用
// - irrelevance: 无关检测
package bfcl

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/easyops/helloagents-go/pkg/evaluation"
)

// 支持的 BFCL 类别
var SupportedCategories = []string{
	"simple_python",
	"simple_java",
	"simple_javascript",
	"multiple",
	"parallel",
	"parallel_multiple",
	"irrelevance",
	"live_simple",
	"live_multiple",
	"live_parallel",
	"multi_turn_base",
	"multi_turn_miss_func",
	"multi_turn_miss_param",
	"multi_turn_long_context",
}

// Dataset BFCL 数据集
type Dataset struct {
	// dataDir BFCL 数据目录
	dataDir string

	// category 评估类别
	category string

	// samples 加载的样本
	samples []evaluation.Sample

	// groundTruth ground truth 数据
	groundTruth map[string]interface{}

	// loaded 是否已加载
	loaded bool
}

// NewDataset 创建 BFCL 数据集
//
// 参数:
//   - dataDir: BFCL 数据目录路径（如 ./temp_gorilla/berkeley-function-call-leaderboard/bfcl_eval/data）
//   - category: 评估类别
func NewDataset(dataDir, category string) *Dataset {
	return &Dataset{
		dataDir:     dataDir,
		category:    category,
		samples:     make([]evaluation.Sample, 0),
		groundTruth: make(map[string]interface{}),
	}
}

// Load 加载数据集
func (d *Dataset) Load(ctx context.Context) error {
	if d.loaded {
		return nil
	}

	// 检查数据目录
	if _, err := os.Stat(d.dataDir); os.IsNotExist(err) {
		return fmt.Errorf("BFCL 数据目录不存在: %s\n请先克隆 BFCL 仓库：git clone --depth 1 https://github.com/ShishirPatil/gorilla.git temp_gorilla", d.dataDir)
	}

	// 加载评估数据
	dataFile := filepath.Join(d.dataDir, fmt.Sprintf("BFCL_v4_%s.json", d.category))
	if err := d.loadDataFile(ctx, dataFile); err != nil {
		return fmt.Errorf("加载数据文件失败: %w", err)
	}

	// 加载 ground truth
	gtFile := filepath.Join(d.dataDir, "possible_answer", fmt.Sprintf("BFCL_v4_%s.json", d.category))
	if err := d.loadGroundTruth(ctx, gtFile); err != nil {
		return fmt.Errorf("加载 ground truth 失败: %w", err)
	}

	d.loaded = true
	return nil
}

// loadDataFile 加载数据文件
func (d *Dataset) loadDataFile(ctx context.Context, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// 增加缓冲区大小以处理长行
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	idx := 0
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		var item map[string]interface{}
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return fmt.Errorf("解析第 %d 行失败: %w", idx+1, err)
		}

		sample := d.parseItem(item, idx)
		d.samples = append(d.samples, sample)
		idx++
	}

	return scanner.Err()
}

// parseItem 解析单个数据项
func (d *Dataset) parseItem(item map[string]interface{}, idx int) evaluation.Sample {
	sample := evaluation.Sample{
		ID:       fmt.Sprintf("%s_%d", d.category, idx),
		Category: d.category,
		Metadata: item,
	}

	// 提取 ID
	if id, ok := item["id"].(string); ok {
		sample.ID = id
	}

	// 提取问题
	if question, ok := item["question"].([]interface{}); ok && len(question) > 0 {
		// BFCL 格式：[[{"role": "user", "content": "..."}]]
		if turn, ok := question[0].([]interface{}); ok && len(turn) > 0 {
			if msg, ok := turn[0].(map[string]interface{}); ok {
				if content, ok := msg["content"].(string); ok {
					sample.Input = content
				}
			}
		}
	}

	// 提取工具定义
	if functions, ok := item["function"].([]interface{}); ok {
		for _, fn := range functions {
			if fnMap, ok := fn.(map[string]interface{}); ok {
				tool := evaluation.ToolDefinition{
					Name:        getString(fnMap, "name"),
					Description: getString(fnMap, "description"),
				}
				if params, ok := fnMap["parameters"].(map[string]interface{}); ok {
					tool.Parameters = params
				}
				sample.Tools = append(sample.Tools, tool)
			}
		}
	}

	return sample
}

// loadGroundTruth 加载 ground truth
func (d *Dataset) loadGroundTruth(ctx context.Context, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		// ground truth 文件可能不存在
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	idx := 0
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		var item map[string]interface{}
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return fmt.Errorf("解析 ground truth 第 %d 行失败: %w", idx+1, err)
		}

		// 提取 ID 和 ground truth
		id := fmt.Sprintf("%s_%d", d.category, idx)
		if idVal, ok := item["id"].(string); ok {
			id = idVal
		}

		if gt, ok := item["ground_truth"]; ok {
			d.groundTruth[id] = gt
		}
		idx++
	}

	return scanner.Err()
}

// Len 返回数据集大小
func (d *Dataset) Len() int {
	return len(d.samples)
}

// Get 根据索引获取样本
func (d *Dataset) Get(index int) (evaluation.Sample, error) {
	if index < 0 || index >= len(d.samples) {
		return evaluation.Sample{}, fmt.Errorf("索引越界: %d", index)
	}

	sample := d.samples[index]
	// 附加 ground truth
	if gt, ok := d.groundTruth[sample.ID]; ok {
		sample.Expected = gt
	}

	return sample, nil
}

// Iterator 返回样本迭代器
func (d *Dataset) Iterator() <-chan evaluation.Sample {
	ch := make(chan evaluation.Sample)
	go func() {
		defer close(ch)
		for i := range d.samples {
			sample, _ := d.Get(i)
			ch <- sample
		}
	}()
	return ch
}

// Name 返回数据集名称
func (d *Dataset) Name() string {
	return fmt.Sprintf("BFCL_%s", d.category)
}

// GetGroundTruth 获取指定样本的 ground truth
func (d *Dataset) GetGroundTruth(sampleID string) (interface{}, bool) {
	gt, ok := d.groundTruth[sampleID]
	return gt, ok
}

// Category 返回类别
func (d *Dataset) Category() string {
	return d.category
}

// getString 安全获取字符串值
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
