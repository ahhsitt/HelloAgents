package bfcl

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/easyops/helloagents-go/pkg/agents"
	"github.com/easyops/helloagents-go/pkg/evaluation"
)

// EvaluationMode 评估模式
type EvaluationMode string

const (
	// ModeAST AST 匹配模式
	ModeAST EvaluationMode = "ast"
	// ModeExecution 执行评估模式
	ModeExecution EvaluationMode = "execution"
)

// Evaluator BFCL 评估器
type Evaluator struct {
	// dataset 数据集
	dataset *Dataset

	// mode 评估模式
	mode EvaluationMode
}

// NewEvaluator 创建 BFCL 评估器
//
// 参数:
//   - dataset: BFCL 数据集
//   - mode: 评估模式（ast 或 execution）
func NewEvaluator(dataset *Dataset, mode EvaluationMode) *Evaluator {
	if mode == "" {
		mode = ModeAST
	}
	return &Evaluator{
		dataset: dataset,
		mode:    mode,
	}
}

// Name 返回评估器名称
func (e *Evaluator) Name() string {
	return fmt.Sprintf("BFCL_%s_%s", e.dataset.Category(), e.mode)
}

// Evaluate 执行完整评估
func (e *Evaluator) Evaluate(ctx context.Context, agent agents.Agent, opts ...evaluation.EvalOption) (*evaluation.EvalResult, error) {
	config := evaluation.DefaultEvalConfig()
	config.ApplyOptions(opts...)

	// 确保数据集已加载
	if err := e.dataset.Load(ctx); err != nil {
		return nil, fmt.Errorf("加载数据集失败: %w", err)
	}

	startTime := time.Now()
	result := &evaluation.EvalResult{
		BenchmarkName:   e.Name(),
		AgentName:       agent.Name(),
		DetailedResults: make([]*evaluation.SampleResult, 0),
		CategoryMetrics: make(map[string]*evaluation.CategoryMetrics),
		EvaluationTime:  startTime,
	}

	total := e.dataset.Len()
	if config.MaxSamples > 0 && config.MaxSamples < total {
		total = config.MaxSamples
	}
	result.TotalSamples = total

	// 遍历样本进行评估
	for i := 0; i < total; i++ {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		sample, err := e.dataset.Get(i)
		if err != nil {
			continue
		}

		// 应用超时
		evalCtx := ctx
		if config.Timeout > 0 {
			var cancel context.CancelFunc
			evalCtx, cancel = context.WithTimeout(ctx, config.Timeout)
			defer cancel()
		}

		sampleResult, err := e.EvaluateSample(evalCtx, agent, sample)
		if err != nil {
			sampleResult = &evaluation.SampleResult{
				SampleID: sample.ID,
				Category: sample.Category,
				Error:    err.Error(),
				Success:  false,
			}
		}

		result.DetailedResults = append(result.DetailedResults, sampleResult)
		if sampleResult.Success {
			result.SuccessCount++
		}

		// 进度回调
		if config.ProgressCallback != nil {
			config.ProgressCallback(i+1, total)
		}
	}

	result.TotalDuration = time.Since(startTime)
	if result.TotalSamples > 0 {
		result.OverallAccuracy = float64(result.SuccessCount) / float64(result.TotalSamples)
	}

	// 计算分类别指标
	e.computeCategoryMetrics(result)

	// 计算汇总指标
	metrics := NewMetrics()
	result.Metrics = metrics.Compute(result.DetailedResults)

	return result, nil
}

// EvaluateSample 评估单个样本
func (e *Evaluator) EvaluateSample(ctx context.Context, agent agents.Agent, sample evaluation.Sample) (*evaluation.SampleResult, error) {
	startTime := time.Now()

	result := &evaluation.SampleResult{
		SampleID: sample.ID,
		Category: sample.Category,
		Expected: sample.Expected,
		Details:  make(map[string]interface{}),
	}

	// 构建输入（包含工具定义）
	input := e.buildAgentInput(sample)

	// 调用智能体
	output, err := agent.Run(ctx, input)
	if err != nil {
		result.Error = err.Error()
		result.ExecutionTime = time.Since(startTime)
		return result, nil
	}

	result.AgentResponse = output.Response
	result.ExecutionTime = time.Since(startTime)

	// 从响应中提取函数调用
	predictedCalls, err := e.extractFunctionCalls(output.Response)
	if err != nil {
		result.Error = fmt.Sprintf("提取函数调用失败: %v", err)
		result.Details["extraction_error"] = err.Error()
		return result, nil
	}
	result.Predicted = predictedCalls

	// 获取 ground truth
	groundTruth, ok := e.dataset.GetGroundTruth(sample.ID)
	if !ok {
		result.Error = "未找到 ground truth"
		return result, nil
	}

	// 评估匹配
	success, score, details := e.evaluateMatch(predictedCalls, groundTruth)
	result.Success = success
	result.Score = score
	for k, v := range details {
		result.Details[k] = v
	}

	return result, nil
}

// buildAgentInput 构建智能体输入
func (e *Evaluator) buildAgentInput(sample evaluation.Sample) agents.Input {
	// 构建工具描述
	var toolsDesc strings.Builder
	toolsDesc.WriteString("你有以下工具可以使用:\n\n")

	for _, tool := range sample.Tools {
		toolsDesc.WriteString(fmt.Sprintf("### %s\n", tool.Name))
		toolsDesc.WriteString(fmt.Sprintf("描述: %s\n", tool.Description))
		if len(tool.Parameters) > 0 {
			paramsJSON, _ := json.MarshalIndent(tool.Parameters, "", "  ")
			toolsDesc.WriteString(fmt.Sprintf("参数: %s\n", string(paramsJSON)))
		}
		toolsDesc.WriteString("\n")
	}

	toolsDesc.WriteString("\n请根据用户问题调用合适的函数。返回格式为 JSON 数组:\n")
	toolsDesc.WriteString(`[{"name": "函数名", "arguments": {"参数名": "参数值"}}]`)

	return agents.Input{
		Query: sample.Input,
		Context: map[string]interface{}{
			"tools":        sample.Tools,
			"tools_prompt": toolsDesc.String(),
		},
	}
}

// extractFunctionCalls 从响应中提取函数调用
func (e *Evaluator) extractFunctionCalls(response string) ([]evaluation.FunctionCall, error) {
	response = strings.TrimSpace(response)
	if response == "" {
		return nil, fmt.Errorf("空响应")
	}

	var calls []evaluation.FunctionCall

	// 尝试直接解析为 JSON 数组
	if err := json.Unmarshal([]byte(response), &calls); err == nil {
		return calls, nil
	}

	// 尝试从响应中提取 JSON 数组
	jsonPattern := regexp.MustCompile(`\[[\s\S]*?\{[\s\S]*?"name"[\s\S]*?\}[\s\S]*?\]`)
	matches := jsonPattern.FindAllString(response, -1)

	for _, match := range matches {
		var extracted []evaluation.FunctionCall
		if err := json.Unmarshal([]byte(match), &extracted); err == nil && len(extracted) > 0 {
			return extracted, nil
		}
	}

	// 尝试解析为单个函数调用对象
	var singleCall evaluation.FunctionCall
	if err := json.Unmarshal([]byte(response), &singleCall); err == nil && singleCall.Name != "" {
		return []evaluation.FunctionCall{singleCall}, nil
	}

	// 尝试从代码块中提取
	codeBlockPattern := regexp.MustCompile("```(?:json)?\\s*([\\s\\S]*?)```")
	codeMatches := codeBlockPattern.FindAllStringSubmatch(response, -1)
	for _, match := range codeMatches {
		if len(match) > 1 {
			content := strings.TrimSpace(match[1])
			if err := json.Unmarshal([]byte(content), &calls); err == nil {
				return calls, nil
			}
			if err := json.Unmarshal([]byte(content), &singleCall); err == nil && singleCall.Name != "" {
				return []evaluation.FunctionCall{singleCall}, nil
			}
		}
	}

	return nil, fmt.Errorf("无法从响应中提取函数调用")
}

// evaluateMatch 评估函数调用匹配
func (e *Evaluator) evaluateMatch(predicted []evaluation.FunctionCall, groundTruth interface{}) (bool, float64, map[string]interface{}) {
	details := make(map[string]interface{})

	// 解析 ground truth
	expectedCalls, err := e.parseGroundTruth(groundTruth)
	if err != nil {
		details["gt_parse_error"] = err.Error()
		return false, 0, details
	}

	details["expected_calls"] = expectedCalls
	details["predicted_calls"] = predicted

	if len(predicted) == 0 {
		details["reason"] = "未预测任何函数调用"
		return false, 0, details
	}

	if len(expectedCalls) == 0 {
		details["reason"] = "无预期函数调用"
		return false, 0, details
	}

	// 计算匹配分数
	matchedCount := 0
	totalScore := 0.0

	for _, expected := range expectedCalls {
		bestScore := 0.0
		for _, pred := range predicted {
			score := e.compareFunctionCall(pred, expected)
			if score > bestScore {
				bestScore = score
			}
		}
		if bestScore >= 1.0 {
			matchedCount++
		}
		totalScore += bestScore
	}

	avgScore := totalScore / float64(len(expectedCalls))
	success := matchedCount == len(expectedCalls)

	details["matched_count"] = matchedCount
	details["expected_count"] = len(expectedCalls)
	details["avg_score"] = avgScore

	return success, avgScore, details
}

// parseGroundTruth 解析 ground truth
func (e *Evaluator) parseGroundTruth(gt interface{}) ([]evaluation.FunctionCall, error) {
	var calls []evaluation.FunctionCall

	switch v := gt.(type) {
	case []interface{}:
		// BFCL v4 格式: [[{"func_name": {"param": [val1, val2]}}]]
		for _, item := range v {
			parsedCalls, err := e.parseGroundTruthItem(item)
			if err != nil {
				continue
			}
			calls = append(calls, parsedCalls...)
		}
	case map[string]interface{}:
		// 单个函数调用
		parsedCalls, err := e.parseGroundTruthItem(v)
		if err != nil {
			return nil, err
		}
		calls = append(calls, parsedCalls...)
	case string:
		// 字符串格式
		var parsed interface{}
		if err := json.Unmarshal([]byte(v), &parsed); err != nil {
			return nil, fmt.Errorf("解析字符串 ground truth 失败: %w", err)
		}
		return e.parseGroundTruth(parsed)
	default:
		return nil, fmt.Errorf("不支持的 ground truth 格式: %T", gt)
	}

	return calls, nil
}

// parseGroundTruthItem 解析单个 ground truth 项
func (e *Evaluator) parseGroundTruthItem(item interface{}) ([]evaluation.FunctionCall, error) {
	var calls []evaluation.FunctionCall

	switch v := item.(type) {
	case []interface{}:
		// 嵌套数组
		for _, subItem := range v {
			subCalls, err := e.parseGroundTruthItem(subItem)
			if err != nil {
				continue
			}
			calls = append(calls, subCalls...)
		}
	case map[string]interface{}:
		// BFCL v4 格式: {"func_name": {"param": [val1, val2]}}
		// 或标准格式: {"name": "...", "arguments": {...}}
		if name, ok := v["name"].(string); ok {
			call := evaluation.FunctionCall{
				Name:      name,
				Arguments: make(map[string]interface{}),
			}
			if args, ok := v["arguments"].(map[string]interface{}); ok {
				call.Arguments = args
			}
			calls = append(calls, call)
		} else {
			// BFCL v4 格式
			for funcName, params := range v {
				call := evaluation.FunctionCall{
					Name:      funcName,
					Arguments: make(map[string]interface{}),
				}
				if paramsMap, ok := params.(map[string]interface{}); ok {
					// 参数值可能是数组（多个可接受值）
					for paramName, paramVal := range paramsMap {
						if valArray, ok := paramVal.([]interface{}); ok && len(valArray) > 0 {
							// 取第一个可接受值
							call.Arguments[paramName] = valArray[0]
						} else {
							call.Arguments[paramName] = paramVal
						}
					}
				}
				calls = append(calls, call)
			}
		}
	case string:
		// Python 函数调用字符串格式
		// 如: "func_name(arg1=val1, arg2=val2)"
		call, err := e.parsePythonFunctionCall(v)
		if err == nil {
			calls = append(calls, call)
		}
	}

	return calls, nil
}

// parsePythonFunctionCall 解析 Python 函数调用字符串
func (e *Evaluator) parsePythonFunctionCall(s string) (evaluation.FunctionCall, error) {
	call := evaluation.FunctionCall{
		Arguments: make(map[string]interface{}),
	}

	// 匹配函数名和参数
	pattern := regexp.MustCompile(`^(\w+)\((.*)\)$`)
	matches := pattern.FindStringSubmatch(strings.TrimSpace(s))
	if len(matches) < 3 {
		return call, fmt.Errorf("无法解析 Python 函数调用: %s", s)
	}

	call.Name = matches[1]
	argsStr := matches[2]

	if argsStr == "" {
		return call, nil
	}

	// 简单解析参数（不处理嵌套）
	argPattern := regexp.MustCompile(`(\w+)\s*=\s*([^,]+)`)
	argMatches := argPattern.FindAllStringSubmatch(argsStr, -1)
	for _, m := range argMatches {
		if len(m) >= 3 {
			paramName := m[1]
			paramVal := strings.TrimSpace(m[2])
			// 尝试解析为 JSON 值
			var val interface{}
			if err := json.Unmarshal([]byte(paramVal), &val); err != nil {
				// 作为字符串处理
				val = strings.Trim(paramVal, `"'`)
			}
			call.Arguments[paramName] = val
		}
	}

	return call, nil
}

// compareFunctionCall 比较两个函数调用
func (e *Evaluator) compareFunctionCall(predicted, expected evaluation.FunctionCall) float64 {
	// 函数名必须匹配
	if predicted.Name != expected.Name {
		return 0
	}

	if len(expected.Arguments) == 0 {
		return 1.0
	}

	// 比较参数
	matchedParams := 0
	for paramName, expectedVal := range expected.Arguments {
		if predictedVal, ok := predicted.Arguments[paramName]; ok {
			if e.compareValues(predictedVal, expectedVal) {
				matchedParams++
			}
		}
	}

	return float64(matchedParams) / float64(len(expected.Arguments))
}

// compareValues 比较两个值是否相等
func (e *Evaluator) compareValues(a, b interface{}) bool {
	// 类型转换后比较
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)

	// 直接比较
	if aStr == bStr {
		return true
	}

	// 忽略大小写比较
	if strings.EqualFold(aStr, bStr) {
		return true
	}

	// 数值比较
	aNum, aErr := toFloat64(a)
	bNum, bErr := toFloat64(b)
	if aErr == nil && bErr == nil && aNum == bNum {
		return true
	}

	return false
}

// toFloat64 尝试转换为 float64
func toFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case string:
		var f float64
		_, err := fmt.Sscanf(val, "%f", &f)
		return f, err
	default:
		return 0, fmt.Errorf("无法转换为 float64")
	}
}

// computeCategoryMetrics 计算分类别指标
func (e *Evaluator) computeCategoryMetrics(result *evaluation.EvalResult) {
	categoryStats := make(map[string]*evaluation.CategoryMetrics)

	for _, sr := range result.DetailedResults {
		cat := sr.Category
		if cat == "" {
			cat = "default"
		}

		if _, ok := categoryStats[cat]; !ok {
			categoryStats[cat] = &evaluation.CategoryMetrics{
				Category: cat,
			}
		}

		categoryStats[cat].Total++
		if sr.Success {
			categoryStats[cat].Success++
		}
	}

	// 计算准确率
	for _, stats := range categoryStats {
		if stats.Total > 0 {
			stats.Accuracy = float64(stats.Success) / float64(stats.Total)
		}
	}

	result.CategoryMetrics = categoryStats
}
