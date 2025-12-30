package evaluation

import (
	"testing"
	"time"
)

func TestSampleResult_Fields(t *testing.T) {
	result := &SampleResult{
		SampleID:      "test_001",
		Predicted:     "predicted answer",
		Expected:      "expected answer",
		Success:       true,
		Score:         0.95,
		Category:      "simple",
		Level:         1,
		ExecutionTime: 100 * time.Millisecond,
		Details:       map[string]interface{}{"key": "value"},
	}

	if result.SampleID != "test_001" {
		t.Errorf("expected SampleID test_001, got %s", result.SampleID)
	}

	if result.Success != true {
		t.Errorf("expected Success true, got %v", result.Success)
	}

	if result.Score != 0.95 {
		t.Errorf("expected Score 0.95, got %f", result.Score)
	}
}

func TestEvalResult_Fields(t *testing.T) {
	result := &EvalResult{
		BenchmarkName:   "BFCL_simple",
		AgentName:       "test_agent",
		TotalSamples:    100,
		SuccessCount:    80,
		OverallAccuracy: 0.8,
		TotalDuration:   10 * time.Second,
		EvaluationTime:  time.Now(),
	}

	if result.BenchmarkName != "BFCL_simple" {
		t.Errorf("expected BenchmarkName BFCL_simple, got %s", result.BenchmarkName)
	}

	if result.TotalSamples != 100 {
		t.Errorf("expected TotalSamples 100, got %d", result.TotalSamples)
	}

	if result.OverallAccuracy != 0.8 {
		t.Errorf("expected OverallAccuracy 0.8, got %f", result.OverallAccuracy)
	}
}

func TestCategoryMetrics_Accuracy(t *testing.T) {
	metrics := &CategoryMetrics{
		Category: "simple",
		Total:    100,
		Success:  75,
		Accuracy: 0.75,
	}

	if metrics.Accuracy != 0.75 {
		t.Errorf("expected Accuracy 0.75, got %f", metrics.Accuracy)
	}
}

func TestLevelMetrics_Rates(t *testing.T) {
	metrics := &LevelMetrics{
		Level:            1,
		Total:            50,
		ExactMatches:     40,
		PartialMatches:   45,
		ExactMatchRate:   0.8,
		PartialMatchRate: 0.9,
	}

	if metrics.ExactMatchRate != 0.8 {
		t.Errorf("expected ExactMatchRate 0.8, got %f", metrics.ExactMatchRate)
	}

	if metrics.PartialMatchRate != 0.9 {
		t.Errorf("expected PartialMatchRate 0.9, got %f", metrics.PartialMatchRate)
	}
}

func TestMetricsSummary_Fields(t *testing.T) {
	summary := &MetricsSummary{
		Accuracy:     0.85,
		Precision:    0.80,
		Recall:       0.90,
		F1Score:      0.85,
		AverageScore: 4.2,
		PassRate:     0.75,
		WinRate:      0.60,
		DimensionScores: map[string]float64{
			"correctness": 4.5,
			"clarity":     4.0,
		},
	}

	if summary.Accuracy != 0.85 {
		t.Errorf("expected Accuracy 0.85, got %f", summary.Accuracy)
	}

	if summary.F1Score != 0.85 {
		t.Errorf("expected F1Score 0.85, got %f", summary.F1Score)
	}

	if summary.DimensionScores["correctness"] != 4.5 {
		t.Errorf("expected correctness 4.5, got %f", summary.DimensionScores["correctness"])
	}
}

func TestFunctionCall_Fields(t *testing.T) {
	call := FunctionCall{
		Name: "get_weather",
		Arguments: map[string]interface{}{
			"city": "Beijing",
			"unit": "celsius",
		},
	}

	if call.Name != "get_weather" {
		t.Errorf("expected Name get_weather, got %s", call.Name)
	}

	if call.Arguments["city"] != "Beijing" {
		t.Errorf("expected city Beijing, got %v", call.Arguments["city"])
	}
}

func TestJudgeScore_TotalScore(t *testing.T) {
	score := JudgeScore{
		Correctness:     4.0,
		Clarity:         5.0,
		DifficultyMatch: 4.0,
		Completeness:    3.0,
		TotalScore:      4.0,
		Comments:        "Good quality",
	}

	expectedTotal := (score.Correctness + score.Clarity + score.DifficultyMatch + score.Completeness) / 4.0
	if score.TotalScore != expectedTotal {
		t.Errorf("expected TotalScore %f, got %f", expectedTotal, score.TotalScore)
	}
}

func TestComparisonResult_Fields(t *testing.T) {
	result := ComparisonResult{
		ProblemAID:   "prob_001",
		ProblemBID:   "prob_002",
		Winner:       "A",
		ActualWinner: "candidate",
		Reason:       "Better clarity",
	}

	if result.Winner != "A" {
		t.Errorf("expected Winner A, got %s", result.Winner)
	}

	if result.ActualWinner != "candidate" {
		t.Errorf("expected ActualWinner candidate, got %s", result.ActualWinner)
	}
}
