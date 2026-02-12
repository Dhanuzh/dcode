package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for dcode
type Config struct {
	// Core settings
	Provider    string  `mapstructure:"provider" json:"provider"`
	Model       string  `mapstructure:"model" json:"model"`
	SmallModel  string  `mapstructure:"small_model" json:"small_model,omitempty"`
	MaxTokens   int     `mapstructure:"max_tokens" json:"max_tokens"`
	Temperature float64 `mapstructure:"temperature" json:"temperature"`

	// API Keys
	AnthropicAPIKey string `mapstructure:"anthropic_api_key" json:"anthropic_api_key,omitempty"`
	OpenAIAPIKey    string `mapstructure:"openai_api_key" json:"openai_api_key,omitempty"`
	GitHubToken     string `mapstructure:"github_token" json:"github_token,omitempty"`
	GoogleAPIKey    string `mapstructure:"google_api_key" json:"google_api_key,omitempty"`
	GroqAPIKey      string `mapstructure:"groq_api_key" json:"groq_api_key,omitempty"`
	OpenRouterKey   string `mapstructure:"openrouter_api_key" json:"openrouter_api_key,omitempty"`

	// Agent configuration
	DefaultAgent string                 `mapstructure:"default_agent" json:"default_agent,omitempty"`
	Agents       map[string]AgentConfig `mapstructure:"agent" json:"agent,omitempty"`

	// Provider overrides
	Providers map[string]ProviderOverride `mapstructure:"provider_config" json:"provider_config,omitempty"`

	// Permission configuration
	Permissions PermissionConfig `mapstructure:"permission" json:"permission,omitempty"`

	// Behavior
	Streaming  bool `mapstructure:"streaming" json:"streaming"`
	Verbose    bool `mapstructure:"verbose" json:"verbose"`
	AutoTitle  bool `mapstructure:"auto_title" json:"auto_title"`
	Snapshot   bool `mapstructure:"snapshot" json:"snapshot"`
	Compaction bool `mapstructure:"compaction" json:"compaction"`

	// Server
	Server ServerConfig `mapstructure:"server" json:"server,omitempty"`

	// TUI
	Theme    string `mapstructure:"theme" json:"theme,omitempty"`
	Username string `mapstructure:"username" json:"username,omitempty"`

	// Session
	SessionDir string `mapstructure:"session_dir" json:"session_dir,omitempty"`

	// Instructions
	Instructions []string `mapstructure:"instructions" json:"instructions,omitempty"`

	// MCP servers
	MCP map[string]MCPConfig `mapstructure:"mcp" json:"mcp,omitempty"`
}

// AgentConfig defines custom agent configuration
type AgentConfig struct {
	Model       string            `mapstructure:"model" json:"model,omitempty"`
	Prompt      string            `mapstructure:"prompt" json:"prompt,omitempty"`
	Description string            `mapstructure:"description" json:"description,omitempty"`
	Mode        string            `mapstructure:"mode" json:"mode,omitempty"` // "primary" or "subagent"
	Steps       int               `mapstructure:"steps" json:"steps,omitempty"`
	Temperature float64           `mapstructure:"temperature" json:"temperature,omitempty"`
	Permission  map[string]string `mapstructure:"permission" json:"permission,omitempty"`
	Tools       []string          `mapstructure:"tools" json:"tools,omitempty"`
}

// ProviderOverride allows customizing provider settings
type ProviderOverride struct {
	BaseURL   string            `mapstructure:"base_url" json:"base_url,omitempty"`
	APIKey    string            `mapstructure:"api_key" json:"api_key,omitempty"`
	Models    map[string]string `mapstructure:"models" json:"models,omitempty"`
	Whitelist []string          `mapstructure:"whitelist" json:"whitelist,omitempty"`
	Blacklist []string          `mapstructure:"blacklist" json:"blacklist,omitempty"`
}

// PermissionConfig defines global permission rules
type PermissionConfig struct {
	Bash              string            `mapstructure:"bash" json:"bash,omitempty"`
	Edit              map[string]string `mapstructure:"edit" json:"edit,omitempty"`
	Write             map[string]string `mapstructure:"write" json:"write,omitempty"`
	ExternalDirectory string            `mapstructure:"external_directory" json:"external_directory,omitempty"`
}

// ServerConfig defines the HTTP server settings
type ServerConfig struct {
	Port     int    `mapstructure:"port" json:"port,omitempty"`
	Hostname string `mapstructure:"hostname" json:"hostname,omitempty"`
	Enabled  bool   `mapstructure:"enabled" json:"enabled,omitempty"`
}

// MCPConfig defines an MCP server connection
type MCPConfig struct {
	Type    string            `mapstructure:"type" json:"type"`
	Command []string          `mapstructure:"command" json:"command,omitempty"`
	URL     string            `mapstructure:"url" json:"url,omitempty"`
	Env     map[string]string `mapstructure:"env" json:"env,omitempty"`
}

// ModelInfo contains information about a specific model
type ModelInfo struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Provider      string `json:"provider"`
	ContextWindow int    `json:"context_window"`
	MaxOutput     int    `json:"max_output"`
	SupportsTools bool   `json:"supports_tools"`
}

// DefaultModels maps provider names to their default model configurations
var DefaultModels = map[string]ModelInfo{
	"anthropic": {
		ID: "claude-sonnet-4-20250514", Name: "Claude Sonnet 4", Provider: "anthropic",
		ContextWindow: 200000, MaxOutput: 16384, SupportsTools: true,
	},
	"openai": {
		ID: "gpt-4.1", Name: "GPT-4.1", Provider: "openai",
		ContextWindow: 1047576, MaxOutput: 32768, SupportsTools: true,
	},
	"copilot": {
		ID: "gpt-4", Name: "GPT-4 (Copilot)", Provider: "copilot",
		ContextWindow: 128000, MaxOutput: 8192, SupportsTools: true,
	},
	"google": {
		ID: "gemini-2.5-flash", Name: "Gemini 2.5 Flash", Provider: "google",
		ContextWindow: 1048576, MaxOutput: 65536, SupportsTools: true,
	},
	"groq": {
		ID: "llama-3.3-70b-versatile", Name: "Llama 3.3 70B", Provider: "groq",
		ContextWindow: 128000, MaxOutput: 32768, SupportsTools: true,
	},
	"openrouter": {
		ID: "anthropic/claude-sonnet-4-20250514", Name: "Claude Sonnet 4 (OpenRouter)", Provider: "openrouter",
		ContextWindow: 200000, MaxOutput: 16384, SupportsTools: true,
	},
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("provider", "anthropic")
	v.SetDefault("model", "")
	v.SetDefault("small_model", "")
	v.SetDefault("max_tokens", 16384)
	v.SetDefault("temperature", 0.0)
	v.SetDefault("streaming", true)
	v.SetDefault("verbose", false)
	v.SetDefault("auto_title", true)
	v.SetDefault("snapshot", true)
	v.SetDefault("compaction", true)
	v.SetDefault("default_agent", "coder")
	v.SetDefault("theme", "dark")
	v.SetDefault("server.port", 4096)
	v.SetDefault("server.hostname", "localhost")
	v.SetDefault("server.enabled", false)

	// Config file locations (precedence: project > home)
	home, err := os.UserHomeDir()
	if err == nil {
		v.AddConfigPath(filepath.Join(home, ".config", "dcode"))
	}
	v.AddConfigPath(".")
	v.AddConfigPath(".dcode")

	v.SetConfigName("dcode")
	v.SetConfigType("yaml")

	// Environment variables
	v.SetEnvPrefix("DCODE")
	v.AutomaticEnv()

	// Map environment variables to config keys
	v.BindEnv("anthropic_api_key", "ANTHROPIC_API_KEY")
	v.BindEnv("openai_api_key", "OPENAI_API_KEY")
	v.BindEnv("github_token", "GITHUB_TOKEN")
	v.BindEnv("google_api_key", "GOOGLE_API_KEY", "GEMINI_API_KEY")
	v.BindEnv("groq_api_key", "GROQ_API_KEY")
	v.BindEnv("openrouter_api_key", "OPENROUTER_API_KEY")

	_ = v.ReadInConfig()

	// Try .dcode/config.json as well
	jsonPath := filepath.Join(".dcode", "config.json")
	if data, err := os.ReadFile(jsonPath); err == nil {
		var jsonCfg map[string]interface{}
		if json.Unmarshal(data, &jsonCfg) == nil {
			for k, val := range jsonCfg {
				v.Set(k, val)
			}
		}
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	// Set session directory
	if config.SessionDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			config.SessionDir = filepath.Join(home, ".config", "dcode", "sessions")
		}
	}

	config.loadInstructions()

	return &config, nil
}

// loadInstructions reads instruction files referenced in config
func (c *Config) loadInstructions() {
	expanded := make([]string, 0, len(c.Instructions))
	for _, instr := range c.Instructions {
		if strings.HasPrefix(instr, "./") || strings.HasPrefix(instr, "/") {
			if data, err := os.ReadFile(instr); err == nil {
				expanded = append(expanded, string(data))
			}
		} else {
			expanded = append(expanded, instr)
		}
	}
	c.Instructions = expanded
}

// GetAPIKey returns the API key for the specified provider
func (c *Config) GetAPIKey(provider string) string {
	switch provider {
	case "anthropic":
		return c.AnthropicAPIKey
	case "openai":
		return c.OpenAIAPIKey
	case "copilot":
		return c.GitHubToken
	case "google":
		return c.GoogleAPIKey
	case "groq":
		return c.GroqAPIKey
	case "openrouter":
		return c.OpenRouterKey
	default:
		return ""
	}
}

// GetDefaultModel returns the appropriate default model for a provider
func (c *Config) GetDefaultModel(provider string) string {
	if c.Model != "" {
		return c.Model
	}
	if info, ok := DefaultModels[provider]; ok {
		return info.ID
	}
	return "claude-sonnet-4-20250514"
}

// GetSmallModel returns the small/fast model for automated tasks
func (c *Config) GetSmallModel() string {
	if c.SmallModel != "" {
		return c.SmallModel
	}
	return "claude-haiku-3-5"
}

// GetModelInfo returns model information for the given provider
func (c *Config) GetModelInfo(provider string) ModelInfo {
	if info, ok := DefaultModels[provider]; ok {
		return info
	}
	return ModelInfo{
		ID:            c.GetDefaultModel(provider),
		Name:          provider,
		Provider:      provider,
		ContextWindow: 128000,
		MaxOutput:     8192,
		SupportsTools: true,
	}
}

// GetProjectDir returns the project directory (cwd)
func GetProjectDir() string {
	dir, _ := os.Getwd()
	return dir
}

// GetConfigDir returns the dcode config directory
func GetConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".dcode"
	}
	return filepath.Join(home, ".config", "dcode")
}

// SaveConfig writes the config to a JSON file
func (c *Config) SaveConfig(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// ListAvailableProviders returns providers that have API keys configured
func (c *Config) ListAvailableProviders() []string {
	creds, _ := LoadCredentials()
	providers := []string{}

	checks := map[string]func() bool{
		"anthropic":  func() bool { return c.AnthropicAPIKey != "" || os.Getenv("ANTHROPIC_API_KEY") != "" || (creds != nil && creds.AnthropicAPIKey != "") },
		"openai":     func() bool { return c.OpenAIAPIKey != "" || os.Getenv("OPENAI_API_KEY") != "" || (creds != nil && creds.OpenAIAPIKey != "") },
		"copilot":    func() bool {
			if c.GitHubToken != "" || os.Getenv("GITHUB_TOKEN") != "" || (creds != nil && creds.GitHubToken != "") {
				return true
			}
			// Auto-detect from gh CLI
			if cmd, err := exec.Command("gh", "auth", "token").Output(); err == nil && strings.TrimSpace(string(cmd)) != "" {
				return true
			}
			return false
		},
		"google":     func() bool { return c.GoogleAPIKey != "" || os.Getenv("GOOGLE_API_KEY") != "" || os.Getenv("GEMINI_API_KEY") != "" || (creds != nil && creds.GoogleAPIKey != "") },
		"groq":       func() bool { return c.GroqAPIKey != "" || os.Getenv("GROQ_API_KEY") != "" || (creds != nil && creds.GroqAPIKey != "") },
		"openrouter": func() bool { return c.OpenRouterKey != "" || os.Getenv("OPENROUTER_API_KEY") != "" || (creds != nil && creds.OpenRouterKey != "") },
	}

	for name, check := range checks {
		if check() {
			providers = append(providers, name)
		}
	}

	return providers
}

// String returns a human-readable representation
func (c *Config) String() string {
	return fmt.Sprintf("Config{Provider: %s, Model: %s, MaxTokens: %d}", c.Provider, c.Model, c.MaxTokens)
}
