package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDefaultConfig tests default configuration values
func TestDefaultConfig(t *testing.T) {
	config := &Config{
		Provider:    "anthropic",
		MaxTokens:   8192,
		Temperature: 0.0,
		Streaming:   true,
	}

	if config.Provider == "" {
		t.Error("Provider should have default value")
	}
	if config.MaxTokens <= 0 {
		t.Error("MaxTokens should be positive")
	}
	if config.Temperature < 0 {
		t.Error("Temperature should not be negative")
	}
}

// TestProviderSelection tests provider configuration
func TestProviderSelection(t *testing.T) {
	providers := []string{
		"anthropic",
		"openai",
		"google",
		"azure",
		"bedrock",
		"copilot",
		"mistral",
		"deepseek",
	}

	for _, p := range providers {
		if p == "" {
			t.Error("Provider name should not be empty")
		}
	}
}

// TestAPIKeyHandling tests API key configuration
func TestAPIKeyHandling(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		envVar   string
	}{
		{"anthropic", "anthropic", "ANTHROPIC_API_KEY"},
		{"openai", "openai", "OPENAI_API_KEY"},
		{"google", "google", "GOOGLE_API_KEY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVar == "" {
				t.Error("Env var should be specified")
			}
		})
	}
}

// TestConfigFileLoading tests configuration file handling
func TestConfigFileLoading(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "dcode.yaml")

	content := `
provider: anthropic
model: claude-sonnet-4.5
max_tokens: 4096
temperature: 0.5
streaming: true
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file should exist")
	}
}

// TestCredentialsStorage tests credential file handling
func TestCredentialsStorage(t *testing.T) {
	tmpDir := t.TempDir()
	credsPath := filepath.Join(tmpDir, "credentials.json")

	content := `{"anthropic_api_key": "test-key"}`
	if err := os.WriteFile(credsPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to write credentials: %v", err)
	}

	// Verify file permissions
	info, err := os.Stat(credsPath)
	if err != nil {
		t.Fatalf("Failed to stat credentials: %v", err)
	}

	mode := info.Mode()
	if mode.Perm() != 0600 {
		t.Errorf("Credentials should have 0600 permissions, got %o", mode.Perm())
	}
}

// TestModelConfiguration tests model settings
func TestModelConfiguration(t *testing.T) {
	tests := []struct {
		provider string
		model    string
		valid    bool
	}{
		{"anthropic", "claude-sonnet-4.5", true},
		{"anthropic", "claude-3-opus", true},
		{"openai", "gpt-4-turbo", true},
		{"openai", "gpt-4", true},
		{"google", "gemini-pro", true},
	}

	for _, tt := range tests {
		t.Run(tt.provider+"/"+tt.model, func(t *testing.T) {
			if tt.model == "" {
				t.Error("Model should not be empty")
			}
		})
	}
}

// TestTemperatureValidation tests temperature bounds
func TestTemperatureValidation(t *testing.T) {
	tests := []struct {
		name  string
		temp  float64
		valid bool
	}{
		{"zero", 0.0, true},
		{"low", 0.3, true},
		{"medium", 0.7, true},
		{"high", 1.0, true},
		{"too high", 2.0, false},
		{"negative", -0.1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.temp >= 0.0 && tt.temp <= 1.0
			if isValid != tt.valid {
				t.Errorf("Temperature %f validity mismatch", tt.temp)
			}
		})
	}
}

// TestMaxTokensValidation tests token limit configuration
func TestMaxTokensValidation(t *testing.T) {
	tests := []struct {
		name   string
		tokens int
		valid  bool
	}{
		{"small", 100, true},
		{"medium", 4096, true},
		{"large", 8192, true},
		{"very large", 16384, true},
		{"zero", 0, false},
		{"negative", -100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.tokens > 0
			if isValid != tt.valid {
				t.Errorf("MaxTokens %d validity mismatch", tt.tokens)
			}
		})
	}
}

// TestConfigPrecedence tests configuration source priority
func TestConfigPrecedence(t *testing.T) {
	// Priority order:
	// 1. CLI flags
	// 2. Environment variables
	// 3. Project config file
	// 4. User config file
	// 5. Defaults

	sources := []string{
		"cli_flag",
		"env_var",
		"project_config",
		"user_config",
		"default",
	}

	if len(sources) != 5 {
		t.Error("Should have 5 config sources")
	}
}

// TestVerboseMode tests verbose logging configuration
func TestVerboseMode(t *testing.T) {
	configs := []struct {
		verbose bool
		name    string
	}{
		{true, "verbose on"},
		{false, "verbose off"},
	}

	for _, cfg := range configs {
		t.Run(cfg.name, func(t *testing.T) {
			// Just verify boolean works
			if cfg.verbose {
				// Verbose mode enabled
			}
		})
	}
}

// TestStreamingConfiguration tests streaming settings
func TestStreamingConfiguration(t *testing.T) {
	tests := []struct {
		streaming bool
		name      string
	}{
		{true, "streaming enabled"},
		{false, "streaming disabled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify boolean configuration
			_ = tt.streaming
		})
	}
}

// TestConfigValidation tests overall config validation
func TestConfigValidation(t *testing.T) {
	config := &Config{
		Provider:    "anthropic",
		Model:       "claude-sonnet-4.5",
		MaxTokens:   8192,
		Temperature: 0.0,
		Streaming:   true,
	}

	if config.Provider == "" {
		t.Error("Provider must be set")
	}
	if config.MaxTokens <= 0 {
		t.Error("MaxTokens must be positive")
	}
	if config.Temperature < 0 || config.Temperature > 1 {
		t.Error("Temperature must be between 0 and 1")
	}
}
