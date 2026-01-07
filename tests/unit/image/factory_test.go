package image

import (
	"testing"

	"github.com/ahhsitt/helloagents-go/pkg/image"
)

func TestNewImageProvider(t *testing.T) {
	tests := []struct {
		providerType image.ProviderType
		apiKey       string
		secretKey    string
		expectError  bool
	}{
		{image.ProviderOpenAI, "test-key", "", false},
		{image.ProviderOpenAI, "", "", true}, // missing API key
		{image.ProviderStability, "test-key", "", false},
		{image.ProviderDashScope, "test-key", "", false},
		{image.ProviderERNIE, "test-key", "test-secret", false},
		{image.ProviderERNIE, "test-key", "", true}, // missing secret key
		{image.ProviderHunyuan, "test-id", "test-key", false},
		{image.ProviderHunyuan, "test-id", "", true}, // missing secret key
	}

	for _, test := range tests {
		opts := []image.Option{image.WithAPIKey(test.apiKey)}
		if test.secretKey != "" {
			opts = append(opts, image.WithSecretKey(test.secretKey))
		}

		provider, err := image.NewImageProvider(test.providerType, opts...)

		if test.expectError {
			if err == nil {
				t.Errorf("expected error for %s, got nil", test.providerType)
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error for %s: %v", test.providerType, err)
			}
			if provider == nil {
				t.Errorf("expected provider for %s, got nil", test.providerType)
			}
		}
	}
}

func TestParseProviderType(t *testing.T) {
	tests := []struct {
		input    string
		expected image.ProviderType
		hasError bool
	}{
		{"openai", image.ProviderOpenAI, false},
		{"OpenAI", image.ProviderOpenAI, false},
		{"DALL-E", image.ProviderOpenAI, false},
		{"stability", image.ProviderStability, false},
		{"stable-diffusion", image.ProviderStability, false},
		{"sd", image.ProviderStability, false},
		{"dashscope", image.ProviderDashScope, false},
		{"aliyun", image.ProviderDashScope, false},
		{"tongyi", image.ProviderDashScope, false},
		{"ernie", image.ProviderERNIE, false},
		{"baidu", image.ProviderERNIE, false},
		{"hunyuan", image.ProviderHunyuan, false},
		{"tencent", image.ProviderHunyuan, false},
		{"unknown", "", true},
		{"", "", true},
	}

	for _, test := range tests {
		result, err := image.ParseProviderType(test.input)
		if test.hasError {
			if err == nil {
				t.Errorf("expected error for %q, got nil", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error for %q: %v", test.input, err)
			}
			if result != test.expected {
				t.Errorf("ParseProviderType(%q) = %v, expected %v", test.input, result, test.expected)
			}
		}
	}
}

func TestSupportedProviders(t *testing.T) {
	providers := image.SupportedProviders()

	if len(providers) != 5 {
		t.Errorf("expected 5 providers, got %d", len(providers))
	}

	expectedProviders := map[image.ProviderType]bool{
		image.ProviderOpenAI:    true,
		image.ProviderStability: true,
		image.ProviderDashScope: true,
		image.ProviderERNIE:     true,
		image.ProviderHunyuan:   true,
	}

	for _, p := range providers {
		if !expectedProviders[p] {
			t.Errorf("unexpected provider: %s", p)
		}
	}
}

func TestNewImageProviderFromConfig(t *testing.T) {
	cfg := image.ProviderConfig{
		Type:           image.ProviderOpenAI,
		APIKey:         "test-key",
		Model:          "dall-e-3",
		TimeoutSeconds: 30,
		MaxRetries:     3,
	}

	provider, err := image.NewImageProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.Name() != "openai" {
		t.Errorf("expected provider name 'openai', got %q", provider.Name())
	}

	if provider.Model() != "dall-e-3" {
		t.Errorf("expected model 'dall-e-3', got %q", provider.Model())
	}
}
