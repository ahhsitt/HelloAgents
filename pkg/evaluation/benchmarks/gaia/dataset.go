// Package gaia 实现 GAIA (General AI Assistants) 评估
//
// GAIA 用于评估 Agent 的通用 AI 助手能力，支持三个难度级别：
// - Level 1: 简单问题
// - Level 2: 中等问题
// - Level 3: 困难问题
package gaia

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/easyops/helloagents-go/pkg/evaluation"
)

// Dataset GAIA 数据集
type Dataset struct {
	// dataDir 本地数据目录
	dataDir string

	// level 难度级别过滤（0 表示不过滤）
	level int

	// split 数据集分割（validation/test）
	split string

	// samples 加载的样本
	samples []evaluation.Sample

	// loaded 是否已加载
	loaded bool
}

// NewDataset 创建 GAIA 数据集
//
// 参数:
//   - dataDir: 本地数据目录路径
//   - level: 难度级别过滤（0 表示全部）
//   - split: 数据集分割（validation 或 test）
func NewDataset(dataDir string, level int, split string) *Dataset {
	if split == "" {
		split = "validation"
	}
	return &Dataset{
		dataDir: dataDir,
		level:   level,
		split:   split,
		samples: make([]evaluation.Sample, 0),
	}
}

// Load 加载数据集
func (d *Dataset) Load(ctx context.Context) error {
	if d.loaded {
		return nil
	}

	// 检查数据目录
	if _, err := os.Stat(d.dataDir); os.IsNotExist(err) {
		return fmt.Errorf("GAIA 数据目录不存在: %s\n请从 HuggingFace 下载: huggingface-cli download gaia-benchmark/GAIA", d.dataDir)
	}

	// 尝试不同的文件格式
	possibleFiles := []string{
		filepath.Join(d.dataDir, fmt.Sprintf("%s.jsonl", d.split)),
		filepath.Join(d.dataDir, fmt.Sprintf("%s.json", d.split)),
		filepath.Join(d.dataDir, d.split, "metadata.jsonl"),
		filepath.Join(d.dataDir, d.split, "data.jsonl"),
	}

	var loadErr error
	for _, filePath := range possibleFiles {
		if _, err := os.Stat(filePath); err == nil {
			if strings.HasSuffix(filePath, ".jsonl") {
				loadErr = d.loadJSONL(ctx, filePath)
			} else {
				loadErr = d.loadJSON(ctx, filePath)
			}
			if loadErr == nil {
				break
			}
		}
	}

	if len(d.samples) == 0 {
		return fmt.Errorf("无法加载 GAIA 数据，尝试了: %v, 最后错误: %v", possibleFiles, loadErr)
	}

	d.loaded = true
	return nil
}

// loadJSONL 加载 JSONL 格式文件
func (d *Dataset) loadJSONL(ctx context.Context, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
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
			continue
		}

		sample := d.parseItem(item, idx)

		// 应用级别过滤
		if d.level > 0 && sample.Level != d.level {
			continue
		}

		d.samples = append(d.samples, sample)
		idx++
	}

	return scanner.Err()
}

// loadJSON 加载 JSON 格式文件
func (d *Dataset) loadJSON(ctx context.Context, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var items []map[string]interface{}
	if err := json.NewDecoder(file).Decode(&items); err != nil {
		return err
	}

	for idx, item := range items {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		sample := d.parseItem(item, idx)

		// 应用级别过滤
		if d.level > 0 && sample.Level != d.level {
			continue
		}

		d.samples = append(d.samples, sample)
	}

	return nil
}

// parseItem 解析单个数据项
func (d *Dataset) parseItem(item map[string]interface{}, idx int) evaluation.Sample {
	sample := evaluation.Sample{
		ID:       fmt.Sprintf("gaia_%d", idx),
		Metadata: item,
	}

	// 提取 task_id
	if id, ok := item["task_id"].(string); ok {
		sample.ID = id
	}

	// 提取问题
	if question, ok := item["question"].(string); ok {
		sample.Input = question
	} else if question, ok := item["Question"].(string); ok {
		sample.Input = question
	}

	// 提取级别
	if level, ok := item["level"].(float64); ok {
		sample.Level = int(level)
	} else if level, ok := item["Level"].(float64); ok {
		sample.Level = int(level)
	} else if level, ok := item["level"].(int); ok {
		sample.Level = level
	}

	// 设置类别
	sample.Category = fmt.Sprintf("level_%d", sample.Level)

	// 提取期望答案
	if answer, ok := item["final_answer"].(string); ok {
		sample.Expected = answer
	} else if answer, ok := item["Final answer"].(string); ok {
		sample.Expected = answer
	} else if answer, ok := item["expected_answer"].(string); ok {
		sample.Expected = answer
	}

	// 提取文件列表
	if fileName, ok := item["file_name"].(string); ok && fileName != "" {
		sample.Files = []string{fileName}
	} else if files, ok := item["file_path"].(string); ok && files != "" {
		sample.Files = []string{files}
	}

	return sample
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
	return d.samples[index], nil
}

// Iterator 返回样本迭代器
func (d *Dataset) Iterator() <-chan evaluation.Sample {
	ch := make(chan evaluation.Sample)
	go func() {
		defer close(ch)
		for _, sample := range d.samples {
			ch <- sample
		}
	}()
	return ch
}

// Name 返回数据集名称
func (d *Dataset) Name() string {
	if d.level > 0 {
		return fmt.Sprintf("GAIA_%s_Level%d", d.split, d.level)
	}
	return fmt.Sprintf("GAIA_%s", d.split)
}

// Level 返回级别过滤
func (d *Dataset) Level() int {
	return d.level
}

// GetLevelDistribution 获取级别分布
func (d *Dataset) GetLevelDistribution() map[int]int {
	dist := make(map[int]int)
	for _, s := range d.samples {
		dist[s.Level]++
	}
	return dist
}
