package gaia

import (
	"testing"

	"github.com/easyops/helloagents-go/pkg/evaluation"
)

func TestNormalizeAnswer(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"The answer", "answer"},
		{"A simple test", "simple test"},
		{"An example.", "example"},
		{"$100", "100"},
		{"50%", "50"},
		{"1,000,000", "1000000"},
		{"  extra  spaces  ", "extra spaces"},
		{"UPPERCASE", "uppercase"},
	}

	for _, tt := range tests {
		result := normalizeAnswer(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeAnswer(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestRemoveNumberCommas(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1,000", "1000"},
		{"1,000,000", "1000000"},
		{"123", "123"},
		{"1,23", "1,23"}, // 不是有效的千位分隔
	}

	for _, tt := range tests {
		result := removeNumberCommas(tt.input)
		if result != tt.expected {
			t.Errorf("removeNumberCommas(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestNewMetrics(t *testing.T) {
	metrics := NewMetrics()
	if metrics == nil {
		t.Error("NewMetrics should return non-nil")
	}
}

func TestMetrics_Compute(t *testing.T) {
	metrics := NewMetrics()

	results := []*evaluation.SampleResult{
		{SampleID: "test_001", Success: true, PartialSuccess: true, Score: 1.0, Level: 1},
		{SampleID: "test_002", Success: false, PartialSuccess: true, Score: 0.5, Level: 1},
		{SampleID: "test_003", Success: true, PartialSuccess: true, Score: 1.0, Level: 2},
	}

	summary := metrics.Compute(results)

	expectedAccuracy := 2.0 / 3.0
	if summary.Accuracy < expectedAccuracy-0.01 || summary.Accuracy > expectedAccuracy+0.01 {
		t.Errorf("expected Accuracy ~%.2f, got %f", expectedAccuracy, summary.Accuracy)
	}

	if summary.Extra["exact_matches"] != 2 {
		t.Errorf("expected exact_matches 2, got %v", summary.Extra["exact_matches"])
	}

	if summary.Extra["partial_matches"] != 3 {
		t.Errorf("expected partial_matches 3, got %v", summary.Extra["partial_matches"])
	}
}

func TestMetrics_ComputeLevelMetrics(t *testing.T) {
	metrics := NewMetrics()

	results := []*evaluation.SampleResult{
		{SampleID: "test_001", Success: true, PartialSuccess: true, Level: 1},
		{SampleID: "test_002", Success: false, PartialSuccess: true, Level: 1},
		{SampleID: "test_003", Success: true, PartialSuccess: true, Level: 2},
		{SampleID: "test_004", Success: false, PartialSuccess: false, Level: 2},
		{SampleID: "test_005", Success: false, PartialSuccess: false, Level: 3},
	}

	levelMetrics := metrics.ComputeLevelMetrics(results)

	// Level 1
	if levelMetrics[1].Total != 2 {
		t.Errorf("expected Level 1 Total 2, got %d", levelMetrics[1].Total)
	}
	if levelMetrics[1].ExactMatches != 1 {
		t.Errorf("expected Level 1 ExactMatches 1, got %d", levelMetrics[1].ExactMatches)
	}
	if levelMetrics[1].ExactMatchRate != 0.5 {
		t.Errorf("expected Level 1 ExactMatchRate 0.5, got %f", levelMetrics[1].ExactMatchRate)
	}

	// Level 2
	if levelMetrics[2].Total != 2 {
		t.Errorf("expected Level 2 Total 2, got %d", levelMetrics[2].Total)
	}

	// Level 3
	if levelMetrics[3].Total != 1 {
		t.Errorf("expected Level 3 Total 1, got %d", levelMetrics[3].Total)
	}
}

func TestMetrics_AnalyzeDifficultyProgression(t *testing.T) {
	metrics := NewMetrics()

	levelMetrics := map[int]*evaluation.LevelMetrics{
		1: {Level: 1, Total: 10, ExactMatches: 8, ExactMatchRate: 0.8},
		2: {Level: 2, Total: 10, ExactMatches: 5, ExactMatchRate: 0.5},
		3: {Level: 3, Total: 10, ExactMatches: 2, ExactMatchRate: 0.2},
	}

	analysis := metrics.AnalyzeDifficultyProgression(levelMetrics)

	drops := analysis["difficulty_drops"].(map[string]float64)
	// 使用近似比较处理浮点数精度问题
	if drops["level1_to_level2"] < 0.29 || drops["level1_to_level2"] > 0.31 {
		t.Errorf("expected level1_to_level2 drop ~0.3, got %f", drops["level1_to_level2"])
	}

	if drops["level1_to_level3"] < 0.59 || drops["level1_to_level3"] > 0.61 {
		t.Errorf("expected level1_to_level3 drop ~0.6, got %f", drops["level1_to_level3"])
	}

	pattern := analysis["performance_pattern"].(string)
	if pattern != "expected_degradation" {
		t.Errorf("expected pattern expected_degradation, got %s", pattern)
	}
}
