package evaluation

import (
	"testing"
	"time"
)

func TestDefaultEvalConfig(t *testing.T) {
	config := DefaultEvalConfig()

	if config.MaxSamples != 0 {
		t.Errorf("expected MaxSamples 0, got %d", config.MaxSamples)
	}

	if config.Timeout != 5*time.Minute {
		t.Errorf("expected Timeout 5m, got %v", config.Timeout)
	}

	if config.OutputDir != "./evaluation_results" {
		t.Errorf("expected OutputDir ./evaluation_results, got %s", config.OutputDir)
	}

	if config.Verbose != false {
		t.Errorf("expected Verbose false, got %v", config.Verbose)
	}
}

func TestApplyOptions(t *testing.T) {
	config := DefaultEvalConfig()

	config.ApplyOptions(
		WithMaxSamples(100),
		WithTimeout(10*time.Minute),
		WithOutputDir("/tmp/output"),
		WithVerbose(true),
	)

	if config.MaxSamples != 100 {
		t.Errorf("expected MaxSamples 100, got %d", config.MaxSamples)
	}

	if config.Timeout != 10*time.Minute {
		t.Errorf("expected Timeout 10m, got %v", config.Timeout)
	}

	if config.OutputDir != "/tmp/output" {
		t.Errorf("expected OutputDir /tmp/output, got %s", config.OutputDir)
	}

	if config.Verbose != true {
		t.Errorf("expected Verbose true, got %v", config.Verbose)
	}
}

func TestWithProgressCallback(t *testing.T) {
	config := DefaultEvalConfig()

	callCount := 0
	callback := func(done, total int) {
		callCount++
	}

	config.ApplyOptions(WithProgressCallback(callback))

	if config.ProgressCallback == nil {
		t.Error("expected ProgressCallback to be set")
	}

	// 测试回调
	config.ProgressCallback(1, 10)
	if callCount != 1 {
		t.Errorf("expected callCount 1, got %d", callCount)
	}
}

func TestWithSaveIntermediateResults(t *testing.T) {
	config := DefaultEvalConfig()

	config.ApplyOptions(WithSaveIntermediateResults(true))

	if config.SaveIntermediateResults != true {
		t.Errorf("expected SaveIntermediateResults true, got %v", config.SaveIntermediateResults)
	}
}
