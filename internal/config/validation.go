package config

import (
	"fmt"
	"strings"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "no validation errors"
	}
	var sb strings.Builder
	sb.WriteString("configuration validation failed:\n")
	for _, err := range e {
		sb.WriteString(fmt.Sprintf("  - %s\n", err.Error()))
	}
	return sb.String()
}

// Validate validates the configuration
func (c *Config) Validate() error {
	var errors ValidationErrors

	// Validate provider
	if c.Provider == "" {
		errors = append(errors, ValidationError{
			Field:   "provider",
			Message: "provider must be specified",
		})
	} else {
		validProviders := []string{
			"anthropic", "openai", "google", "azure", "bedrock",
			"cerebras", "cloudflare", "cohere", "copilot", "deepinfra",
			"deepseek", "gitlab", "google_vertex", "mistral", "perplexity",
			"replicate", "together", "xai", "groq", "openrouter",
		}
		valid := false
		for _, p := range validProviders {
			if c.Provider == p {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, ValidationError{
				Field:   "provider",
				Message: fmt.Sprintf("unknown provider '%s', valid: %s", c.Provider, strings.Join(validProviders, ", ")),
			})
		}
	}

	// Validate max_tokens
	if c.MaxTokens <= 0 {
		errors = append(errors, ValidationError{
			Field:   "max_tokens",
			Message: "must be positive",
		})
	}
	if c.MaxTokens > 1000000 {
		errors = append(errors, ValidationError{
			Field:   "max_tokens",
			Message: "exceeds reasonable limit (1M tokens)",
		})
	}

	// Validate temperature
	if c.Temperature < 0 {
		errors = append(errors, ValidationError{
			Field:   "temperature",
			Message: "must be non-negative",
		})
	}
	if c.Temperature > 2.0 {
		errors = append(errors, ValidationError{
			Field:   "temperature",
			Message: "must be <= 2.0",
		})
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// GetConfigPrecedence returns a description of config source precedence
func GetConfigPrecedence() string {
	return `Configuration is loaded in the following order (later sources override earlier):

1. Built-in defaults
2. Global config file (~/.config/dcode/dcode.yaml)
3. Project config file (./dcode.yaml or ./.dcode/dcode.yaml)
4. Environment variables (ANTHROPIC_API_KEY, OPENAI_API_KEY, etc.)
5. Stored credentials (~/.config/dcode/credentials.json)
6. Command-line flags (--provider, --model, --config, etc.)

Example precedence resolution:
- If --provider=openai is passed, it overrides all config files
- If ANTHROPIC_API_KEY is set, it overrides keys in config files
- Project config (./dcode.yaml) overrides global config
`
}

// ValidateAPIKey checks if an API key looks valid
func ValidateAPIKey(provider, key string) error {
	if key == "" {
		return fmt.Errorf("API key is empty")
	}

	// Basic format validation
	switch provider {
	case "anthropic":
		if !strings.HasPrefix(key, "sk-ant-") {
			return fmt.Errorf("Anthropic API key should start with 'sk-ant-'")
		}
		if len(key) < 20 {
			return fmt.Errorf("Anthropic API key seems too short")
		}
	case "openai":
		if !strings.HasPrefix(key, "sk-") && !strings.HasPrefix(key, "sk-proj-") {
			return fmt.Errorf("OpenAI API key should start with 'sk-' or 'sk-proj-'")
		}
		if len(key) < 20 {
			return fmt.Errorf("OpenAI API key seems too short")
		}
	case "google":
		if len(key) < 20 {
			return fmt.Errorf("Google API key seems too short")
		}
		// Add more providers as needed
	}

	return nil
}

// GetDefaultModel returns the default model for a provider
func GetDefaultModel(provider string) string {
	defaults := map[string]string{
		"anthropic":     "claude-sonnet-4-5",
		"openai":        "gpt-4o",
		"google":        "gemini-2.0-flash-exp",
		"azure":         "gpt-4o",
		"bedrock":       "anthropic.claude-3-5-sonnet-20241022-v2:0",
		"cerebras":      "llama3.1-70b",
		"cloudflare":    "@cf/meta/llama-3.1-8b-instruct",
		"cohere":        "command-r-plus",
		"copilot":       "gpt-4o",
		"deepinfra":     "meta-llama/Meta-Llama-3.1-70B-Instruct",
		"deepseek":      "deepseek-chat",
		"gitlab":        "claude-3-5-sonnet",
		"google_vertex": "gemini-2.0-flash-exp",
		"mistral":       "mistral-large-latest",
		"perplexity":    "llama-3.1-sonar-huge-128k-online",
		"replicate":     "meta/llama-2-70b-chat",
		"together":      "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo",
		"xai":           "grok-beta",
		"groq":          "llama-3.3-70b-versatile",
		"openrouter":    "openai/gpt-4o",
	}

	if model, ok := defaults[provider]; ok {
		return model
	}
	return ""
}

// GetProviderRequirements returns what's required to use a provider
func GetProviderRequirements(provider string) string {
	requirements := map[string]string{
		"anthropic":     "Anthropic API key (get at console.anthropic.com)",
		"openai":        "OpenAI API key (get at platform.openai.com)",
		"google":        "Google API key (get at makersuite.google.com)",
		"azure":         "Azure OpenAI endpoint URL and API key",
		"bedrock":       "AWS credentials with Bedrock access",
		"cerebras":      "Cerebras API key",
		"cloudflare":    "Cloudflare account ID and API key",
		"cohere":        "Cohere API key",
		"copilot":       "GitHub Copilot token",
		"deepinfra":     "DeepInfra API key",
		"deepseek":      "DeepSeek API key",
		"gitlab":        "GitLab personal access token",
		"google_vertex": "Google Cloud project with Vertex AI enabled",
		"mistral":       "Mistral API key",
		"perplexity":    "Perplexity API key",
		"replicate":     "Replicate API key",
		"together":      "Together AI API key",
		"xai":           "xAI API key",
		"groq":          "Groq API key",
		"openrouter":    "OpenRouter API key",
	}

	if req, ok := requirements[provider]; ok {
		return req
	}
	return "API key required"
}
