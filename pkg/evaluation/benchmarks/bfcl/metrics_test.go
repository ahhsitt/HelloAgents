package bfcl

import (
	"testing"

	"github.com/easyops/helloagents-go/pkg/evaluation"
)

func TestNewMetrics(t *testing.T) {
	metrics := NewMetrics()
	if metrics == nil {
		t.Error("NewMetrics should return non-nil")
	}
}

func TestMetrics_Compute_Empty(t *testing.T) {
	metrics := NewMetrics()
	summary := metrics.Compute([]*evaluation.SampleResult{})

	if summary.Accuracy != 0 {
		t.Errorf("expected Accuracy 0, got %f", summary.Accuracy)
	}
}

func TestMetrics_Compute(t *testing.T) {
	metrics := NewMetrics()

	results := []*evaluation.SampleResult{
		{
			SampleID: "test_001",
			Success:  true,
			Score:    1.0,
			Details: map[string]interface{}{
				"expected_count":  2,
				"matched_count":   2,
				"predicted_calls": []evaluation.FunctionCall{{Name: "func1"}, {Name: "func2"}},
			},
		},
		{
			SampleID: "test_002",
			Success:  false,
			Score:    0.5,
			Details: map[string]interface{}{
				"expected_count":  2,
				"matched_count":   1,
				"predicted_calls": []evaluation.FunctionCall{{Name: "func1"}},
			},
		},
	}

	summary := metrics.Compute(results)

	if summary.Accuracy != 0.5 {
		t.Errorf("expected Accuracy 0.5, got %f", summary.Accuracy)
	}

	if summary.AverageScore != 0.75 {
		t.Errorf("expected AverageScore 0.75, got %f", summary.AverageScore)
	}
}

func TestMetrics_ComputeCategoryMetrics(t *testing.T) {
	metrics := NewMetrics()

	results := []*evaluation.SampleResult{
		{SampleID: "test_001", Category: "simple", Success: true, Score: 1.0},
		{SampleID: "test_002", Category: "simple", Success: false, Score: 0.5},
		{SampleID: "test_003", Category: "multiple", Success: true, Score: 1.0},
	}

	categoryMetrics := metrics.ComputeCategoryMetrics(results)

	if len(categoryMetrics) != 2 {
		t.Errorf("expected 2 categories, got %d", len(categoryMetrics))
	}

	simpleMetrics := categoryMetrics["simple"]
	if simpleMetrics.Total != 2 {
		t.Errorf("expected simple.Total 2, got %d", simpleMetrics.Total)
	}
	if simpleMetrics.Success != 1 {
		t.Errorf("expected simple.Success 1, got %d", simpleMetrics.Success)
	}
	if simpleMetrics.Accuracy != 0.5 {
		t.Errorf("expected simple.Accuracy 0.5, got %f", simpleMetrics.Accuracy)
	}

	multipleMetrics := categoryMetrics["multiple"]
	if multipleMetrics.Total != 1 {
		t.Errorf("expected multiple.Total 1, got %d", multipleMetrics.Total)
	}
	if multipleMetrics.Accuracy != 1.0 {
		t.Errorf("expected multiple.Accuracy 1.0, got %f", multipleMetrics.Accuracy)
	}
}
