// Package datagen 实现数据生成质量评估
//
// 本包提供两种评估方式：
// - LLM Judge: 使用 LLM 作为评委进行多维度质量评估
// - Win Rate: 成对对比计算胜率
package datagen

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ahhsitt/helloagents-go/pkg/evaluation"
)

// Dataset 数据生成评估数据集
type Dataset struct {
	// dataPath 数据文件路径
	dataPath string

	// samples 加载的样本
	samples []evaluation.Sample

	// loaded 是否已加载
	loaded bool
}

// NewDataset 创建数据生成评估数据集
//
// 参数:
//   - dataPath: 数据文件路径（JSONL 格式）
func NewDataset(dataPath string) *Dataset {
	return &Dataset{
		dataPath: dataPath,
		samples:  make([]evaluation.Sample, 0),
	}
}

// Load 加载数据集
func (d *Dataset) Load(ctx context.Context) error {
	if d.loaded {
		return nil
	}

	if _, err := os.Stat(d.dataPath); os.IsNotExist(err) {
		return fmt.Errorf("数据文件不存在: %s", d.dataPath)
	}

	file, err := os.Open(d.dataPath)
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
		d.samples = append(d.samples, sample)
		idx++
	}

	d.loaded = true
	return scanner.Err()
}

// parseItem 解析单个数据项
func (d *Dataset) parseItem(item map[string]interface{}, idx int) evaluation.Sample {
	sample := evaluation.Sample{
		ID:       fmt.Sprintf("datagen_%d", idx),
		Metadata: item,
	}

	// 提取 ID
	if id, ok := item["id"].(string); ok {
		sample.ID = id
	}

	// 提取问题/内容
	if question, ok := item["question"].(string); ok {
		sample.Input = question
	} else if content, ok := item["content"].(string); ok {
		sample.Input = content
	} else if problem, ok := item["problem"].(string); ok {
		sample.Input = problem
	}

	// 提取类别
	if category, ok := item["category"].(string); ok {
		sample.Category = category
	} else if difficulty, ok := item["difficulty"].(string); ok {
		sample.Category = difficulty
	}

	// 提取答案/解决方案
	if answer, ok := item["answer"].(string); ok {
		sample.Expected = answer
	} else if solution, ok := item["solution"].(string); ok {
		sample.Expected = solution
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
	return fmt.Sprintf("DataGen_%s", filepath.Base(d.dataPath))
}

// GetSamples 获取所有样本
func (d *Dataset) GetSamples() []evaluation.Sample {
	return d.samples
}
