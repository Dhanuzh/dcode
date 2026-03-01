package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/spf13/viper"
)

// ---------------------------------------------------------------------------
// Environment variable / flag constants (matching opencode)
// ---------------------------------------------------------------------------

const (
	EnvConfig               = "DCODE_CONFIG"         // path to custom config file
	EnvConfigDir            = "DCODE_CONFIG_DIR"     // path to custom .dcode dir
	EnvConfigContent        = "DCODE_CONFIG_CONTENT" // raw JSON config
	EnvDisableProjectConfig = "DCODE_DISABLE_PROJECT_CONFIG"
	EnvPermission           = "DCODE_PERMISSION"
	EnvDisableAutoCompact   = "DCODE_DISABLE_AUTOCOMPACT"
	EnvDisablePrune         = "DCODE_DISABLE_PRUNE"
	EnvTestManagedConfigDir = "DCODE_TEST_MANAGED_CONFIG_DIR"
)

// ---------------------------------------------------------------------------
// Top-level Config – matches opencode's Info schema + existing dcode fields
// ---------------------------------------------------------------------------

// Config holds all configuration for dcode.
// Fields marked "// opencode" indicate parity with the opencode project.
type Config struct {
	// --- Core settings ---
	Provider    string  `mapstructure:"provider" json:"provider"`
	Model       string  `mapstructure:"model" json:"model"`
	SmallModel  string  `mapstructure:"small_model" json:"small_model,omitempty"`
	MaxTokens   int     `mapstructure:"max_tokens" json:"max_tokens"`
	Temperature float64 `mapstructure:"temperature" json:"temperature"`

	// --- API Keys (legacy – prefer credentials/env) ---
	OpenAIAPIKey  string `mapstructure:"openai_api_key" json:"openai_api_key,omitempty"`
	GoogleAPIKey  string `mapstructure:"google_api_key" json:"google_api_key,omitempty"`
	GroqAPIKey    string `mapstructure:"groq_api_key" json:"groq_api_key,omitempty"`
	OpenRouterKey string `mapstructure:"openrouter_api_key" json:"openrouter_api_key,omitempty"`

	// --- Agent configuration ---
	DefaultAgent string                 `mapstructure:"default_agent" json:"default_agent,omitempty"`
	Agents       map[string]AgentConfig `mapstructure:"agent" json:"agent,omitempty"`

	// --- Provider overrides ---
	Providers map[string]ProviderOverride `mapstructure:"provider_config" json:"provider_config,omitempty"`

	// --- Permission configuration (opencode) ---
	Permissions PermissionConfig `mapstructure:"permission" json:"permission,omitempty"`

	// --- Behavior ---
	Streaming  bool `mapstructure:"streaming" json:"streaming"`
	Verbose    bool `mapstructure:"verbose" json:"verbose"`
	AutoTitle  bool `mapstructure:"auto_title" json:"auto_title"`
	Snapshot   bool `mapstructure:"snapshot" json:"snapshot"`
	Compaction bool `mapstructure:"compaction" json:"compaction"`

	// --- Server ---
	Server ServerConfig `mapstructure:"server" json:"server,omitempty"`

	// --- TUI ---
	Theme    string `mapstructure:"theme" json:"theme,omitempty"`
	Username string `mapstructure:"username" json:"username,omitempty"`

	// --- Session ---
	SessionDir string `mapstructure:"session_dir" json:"session_dir,omitempty"`

	// --- Instructions ---
	Instructions []string `mapstructure:"instructions" json:"instructions,omitempty"`

	// --- MCP servers ---
	MCP map[string]MCPConfig `mapstructure:"mcp" json:"mcp,omitempty"`

	// --- opencode-parity fields below ---

	// Keybinds (opencode)
	Keybinds *Keybinds `mapstructure:"keybinds" json:"keybinds,omitempty"`

	// Skills (opencode)
	Skills *SkillsConfig `mapstructure:"skills" json:"skills,omitempty"`

	// Commands from .dcode/commands/*.md (opencode)
	Commands map[string]CommandConfig `mapstructure:"command" json:"command,omitempty"`

	// TUI settings (opencode)
	TUI *TUIConfig `mapstructure:"tui" json:"tui,omitempty"`

	// Provider enable/disable (opencode)
	DisabledProviders []string `mapstructure:"disabled_providers" json:"disabled_providers,omitempty"`
	EnabledProviders  []string `mapstructure:"enabled_providers" json:"enabled_providers,omitempty"`

	// Watcher ignore patterns (opencode)
	Watcher *WatcherConfig `mapstructure:"watcher" json:"watcher,omitempty"`

	// Share (opencode) – "manual" | "auto" | "disabled"
	Share string `mapstructure:"share" json:"share,omitempty"`

	// Compaction settings (opencode – fine-grained control)
	CompactionConfig *CompactionConfig `mapstructure:"compaction_config" json:"compaction_config,omitempty"`

	// Formatter (opencode) – false disables; map configures per-formatter
	Formatter map[string]FormatterConfig `mapstructure:"formatter" json:"formatter,omitempty"`

	// LSP (opencode) – false disables; map configures per-language
	LSP map[string]LSPConfig `mapstructure:"lsp" json:"lsp,omitempty"`

	// Log level (opencode) – DEBUG | INFO | WARN | ERROR
	LogLevel string `mapstructure:"log_level" json:"log_level,omitempty"`

	// Experimental (opencode)
	Experimental *ExperimentalConfig `mapstructure:"experimental" json:"experimental,omitempty"`

	// Layout (deprecated in opencode, kept for compat)
	Layout string `mapstructure:"layout" json:"layout,omitempty"`

	// Directories that were scanned for config (populated at load time, not serialized)
	configDirectories []string `json:"-"`
}

// ---------------------------------------------------------------------------
// Sub-config types
// ---------------------------------------------------------------------------

// AgentConfig defines custom agent configuration (opencode Agent schema)
type AgentConfig struct {
	Model       string            `mapstructure:"model" json:"model,omitempty"`
	Variant     string            `mapstructure:"variant" json:"variant,omitempty"`
	Prompt      string            `mapstructure:"prompt" json:"prompt,omitempty"`
	Description string            `mapstructure:"description" json:"description,omitempty"`
	Mode        string            `mapstructure:"mode" json:"mode,omitempty"` // "primary" | "subagent" | "all"
	Steps       int               `mapstructure:"steps" json:"steps,omitempty"`
	Temperature float64           `mapstructure:"temperature" json:"temperature,omitempty"`
	TopP        float64           `mapstructure:"top_p" json:"top_p,omitempty"`
	Permission  map[string]string `mapstructure:"permission" json:"permission,omitempty"` // tool -> "ask"|"allow"|"deny"
	Tools       []string          `mapstructure:"tools" json:"tools,omitempty"`           // deprecated – use Permission
	Disable     bool              `mapstructure:"disable" json:"disable,omitempty"`
	Hidden      bool              `mapstructure:"hidden" json:"hidden,omitempty"`
	Color       string            `mapstructure:"color" json:"color,omitempty"`
	Options     map[string]any    `mapstructure:"options" json:"options,omitempty"`
}

// ProviderOverride allows customizing provider settings (opencode Provider schema)
type ProviderOverride struct {
	BaseURL   string            `mapstructure:"base_url" json:"base_url,omitempty"`
	APIKey    string            `mapstructure:"api_key" json:"api_key,omitempty"`
	Name      string            `mapstructure:"name" json:"name,omitempty"`
	Models    map[string]string `mapstructure:"models" json:"models,omitempty"`
	Whitelist []string          `mapstructure:"whitelist" json:"whitelist,omitempty"`
	Blacklist []string          `mapstructure:"blacklist" json:"blacklist,omitempty"`
	Options   *ProviderOptions  `mapstructure:"options" json:"options,omitempty"`
}

// ProviderOptions are per-provider option overrides (opencode)
type ProviderOptions struct {
	APIKey        string `mapstructure:"api_key" json:"api_key,omitempty"`
	BaseURL       string `mapstructure:"base_url" json:"base_url,omitempty"`
	EnterpriseURL string `mapstructure:"enterprise_url" json:"enterprise_url,omitempty"`
	Timeout       int    `mapstructure:"timeout" json:"timeout,omitempty"` // ms, 0=default
}

// PermissionConfig defines global permission rules (opencode Permission schema)
type PermissionConfig struct {
	Bash              string            `mapstructure:"bash" json:"bash,omitempty"`   // "ask"|"allow"|"deny"
	Edit              map[string]string `mapstructure:"edit" json:"edit,omitempty"`   // pattern -> action
	Write             map[string]string `mapstructure:"write" json:"write,omitempty"` // pattern -> action
	Read              string            `mapstructure:"read" json:"read,omitempty"`
	Glob              string            `mapstructure:"glob" json:"glob,omitempty"`
	Grep              string            `mapstructure:"grep" json:"grep,omitempty"`
	List              string            `mapstructure:"list" json:"list,omitempty"`
	Task              string            `mapstructure:"task" json:"task,omitempty"`
	WebFetch          string            `mapstructure:"webfetch" json:"webfetch,omitempty"`
	WebSearch         string            `mapstructure:"websearch" json:"websearch,omitempty"`
	CodeSearch        string            `mapstructure:"codesearch" json:"codesearch,omitempty"`
	LSP               string            `mapstructure:"lsp" json:"lsp,omitempty"`
	Question          string            `mapstructure:"question" json:"question,omitempty"`
	Skill             string            `mapstructure:"skill" json:"skill,omitempty"`
	TodoWrite         string            `mapstructure:"todowrite" json:"todowrite,omitempty"`
	TodoRead          string            `mapstructure:"todoread" json:"todoread,omitempty"`
	DoomLoop          string            `mapstructure:"doom_loop" json:"doom_loop,omitempty"`
	ExternalDirectory string            `mapstructure:"external_directory" json:"external_directory,omitempty"`
}

// ServerConfig defines the HTTP server settings
type ServerConfig struct {
	Port       int      `mapstructure:"port" json:"port,omitempty"`
	Hostname   string   `mapstructure:"hostname" json:"hostname,omitempty"`
	Enabled    bool     `mapstructure:"enabled" json:"enabled,omitempty"`
	MDNS       bool     `mapstructure:"mdns" json:"mdns,omitempty"`
	MDNSDomain string   `mapstructure:"mdns_domain" json:"mdns_domain,omitempty"`
	CORS       []string `mapstructure:"cors" json:"cors,omitempty"`
}

// MCPConfig defines an MCP server connection (opencode Mcp schema)
type MCPConfig struct {
	Type    string            `mapstructure:"type" json:"type"` // "local" | "remote"
	Command []string          `mapstructure:"command" json:"command,omitempty"`
	URL     string            `mapstructure:"url" json:"url,omitempty"`
	Env     map[string]string `mapstructure:"env" json:"env,omitempty"`
	Headers map[string]string `mapstructure:"headers" json:"headers,omitempty"`
	Enabled *bool             `mapstructure:"enabled" json:"enabled,omitempty"` // nil = enabled
	Timeout int               `mapstructure:"timeout" json:"timeout,omitempty"` // ms
}

// Keybinds stores all keybind configuration (opencode, 80+ fields with defaults)
type Keybinds struct {
	Leader                  string `mapstructure:"leader" json:"leader,omitempty"`
	AppExit                 string `mapstructure:"app_exit" json:"app_exit,omitempty"`
	EditorOpen              string `mapstructure:"editor_open" json:"editor_open,omitempty"`
	ThemeList               string `mapstructure:"theme_list" json:"theme_list,omitempty"`
	SidebarToggle           string `mapstructure:"sidebar_toggle" json:"sidebar_toggle,omitempty"`
	StatusView              string `mapstructure:"status_view" json:"status_view,omitempty"`
	SessionExport           string `mapstructure:"session_export" json:"session_export,omitempty"`
	SessionNew              string `mapstructure:"session_new" json:"session_new,omitempty"`
	SessionList             string `mapstructure:"session_list" json:"session_list,omitempty"`
	SessionTimeline         string `mapstructure:"session_timeline" json:"session_timeline,omitempty"`
	SessionFork             string `mapstructure:"session_fork" json:"session_fork,omitempty"`
	SessionRename           string `mapstructure:"session_rename" json:"session_rename,omitempty"`
	SessionDelete           string `mapstructure:"session_delete" json:"session_delete,omitempty"`
	SessionInterrupt        string `mapstructure:"session_interrupt" json:"session_interrupt,omitempty"`
	SessionCompact          string `mapstructure:"session_compact" json:"session_compact,omitempty"`
	SessionShare            string `mapstructure:"session_share" json:"session_share,omitempty"`
	ModelList               string `mapstructure:"model_list" json:"model_list,omitempty"`
	ModelProviderList       string `mapstructure:"model_provider_list" json:"model_provider_list,omitempty"`
	ModelFavoriteToggle     string `mapstructure:"model_favorite_toggle" json:"model_favorite_toggle,omitempty"`
	ModelCycleRecent        string `mapstructure:"model_cycle_recent" json:"model_cycle_recent,omitempty"`
	ModelCycleRecentReverse string `mapstructure:"model_cycle_recent_reverse" json:"model_cycle_recent_reverse,omitempty"`
	CommandList             string `mapstructure:"command_list" json:"command_list,omitempty"`
	AgentList               string `mapstructure:"agent_list" json:"agent_list,omitempty"`
	AgentCycle              string `mapstructure:"agent_cycle" json:"agent_cycle,omitempty"`
	AgentCycleReverse       string `mapstructure:"agent_cycle_reverse" json:"agent_cycle_reverse,omitempty"`
	VariantCycle            string `mapstructure:"variant_cycle" json:"variant_cycle,omitempty"`
	InputClear              string `mapstructure:"input_clear" json:"input_clear,omitempty"`
	InputPaste              string `mapstructure:"input_paste" json:"input_paste,omitempty"`
	InputSubmit             string `mapstructure:"input_submit" json:"input_submit,omitempty"`
	InputNewline            string `mapstructure:"input_newline" json:"input_newline,omitempty"`
	MessagesPageUp          string `mapstructure:"messages_page_up" json:"messages_page_up,omitempty"`
	MessagesPageDown        string `mapstructure:"messages_page_down" json:"messages_page_down,omitempty"`
	MessagesFirst           string `mapstructure:"messages_first" json:"messages_first,omitempty"`
	MessagesLast            string `mapstructure:"messages_last" json:"messages_last,omitempty"`
	MessagesCopy            string `mapstructure:"messages_copy" json:"messages_copy,omitempty"`
	MessagesUndo            string `mapstructure:"messages_undo" json:"messages_undo,omitempty"`
	MessagesRedo            string `mapstructure:"messages_redo" json:"messages_redo,omitempty"`
	MessagesToggleConceal   string `mapstructure:"messages_toggle_conceal" json:"messages_toggle_conceal,omitempty"`
	ToolDetails             string `mapstructure:"tool_details" json:"tool_details,omitempty"`
	HistoryPrevious         string `mapstructure:"history_previous" json:"history_previous,omitempty"`
	HistoryNext             string `mapstructure:"history_next" json:"history_next,omitempty"`
	TerminalSuspend         string `mapstructure:"terminal_suspend" json:"terminal_suspend,omitempty"`
}

// DefaultKeybinds returns keybinds with all opencode defaults applied.
func DefaultKeybinds() *Keybinds {
	return &Keybinds{
		Leader:                  "ctrl+x",
		AppExit:                 "ctrl+c,ctrl+d,<leader>q",
		EditorOpen:              "<leader>e",
		ThemeList:               "<leader>t",
		SidebarToggle:           "<leader>b",
		StatusView:              "<leader>s",
		SessionExport:           "<leader>x",
		SessionNew:              "<leader>n",
		SessionList:             "<leader>l",
		SessionTimeline:         "<leader>g",
		SessionFork:             "none",
		SessionRename:           "ctrl+r",
		SessionDelete:           "ctrl+d",
		SessionInterrupt:        "escape",
		SessionCompact:          "<leader>c",
		SessionShare:            "none",
		ModelList:               "<leader>m",
		ModelProviderList:       "ctrl+a",
		ModelFavoriteToggle:     "ctrl+f",
		ModelCycleRecent:        "f2",
		ModelCycleRecentReverse: "shift+f2",
		CommandList:             "ctrl+p",
		AgentList:               "<leader>a",
		AgentCycle:              "tab",
		AgentCycleReverse:       "shift+tab",
		VariantCycle:            "ctrl+t",
		InputClear:              "ctrl+c",
		InputPaste:              "ctrl+v",
		InputSubmit:             "return",
		InputNewline:            "shift+return,ctrl+return,alt+return,ctrl+j",
		MessagesPageUp:          "pageup,ctrl+alt+b",
		MessagesPageDown:        "pagedown,ctrl+alt+f",
		MessagesFirst:           "ctrl+g,home",
		MessagesLast:            "ctrl+alt+g,end",
		MessagesCopy:            "<leader>y",
		MessagesUndo:            "<leader>u",
		MessagesRedo:            "<leader>r",
		MessagesToggleConceal:   "<leader>h",
		ToolDetails:             "none",
		HistoryPrevious:         "up",
		HistoryNext:             "down",
		TerminalSuspend:         "ctrl+z",
	}
}

// SkillsConfig defines skill loading (opencode)
type SkillsConfig struct {
	Paths []string `mapstructure:"paths" json:"paths,omitempty"` // additional skill directories
	URLs  []string `mapstructure:"urls" json:"urls,omitempty"`   // URLs to fetch skills from
}

// CommandConfig defines a slash-command loaded from .dcode/commands/*.md (opencode)
type CommandConfig struct {
	Template    string `mapstructure:"template" json:"template"`
	Description string `mapstructure:"description" json:"description,omitempty"`
	Agent       string `mapstructure:"agent" json:"agent,omitempty"`
	Model       string `mapstructure:"model" json:"model,omitempty"`
	Subtask     bool   `mapstructure:"subtask" json:"subtask,omitempty"`
}

// TUIConfig holds TUI-specific settings (opencode)
type TUIConfig struct {
	ScrollSpeed        float64                   `mapstructure:"scroll_speed" json:"scroll_speed,omitempty"`
	ScrollAcceleration *ScrollAccelerationConfig `mapstructure:"scroll_acceleration" json:"scroll_acceleration,omitempty"`
	DiffStyle          string                    `mapstructure:"diff_style" json:"diff_style,omitempty"` // "auto" | "stacked"
}

// ScrollAccelerationConfig (opencode)
type ScrollAccelerationConfig struct {
	Enabled bool `mapstructure:"enabled" json:"enabled"`
}

// WatcherConfig (opencode)
type WatcherConfig struct {
	Ignore []string `mapstructure:"ignore" json:"ignore,omitempty"`
}

// CompactionConfig (opencode fine-grained control)
type CompactionConfig struct {
	Auto  *bool `mapstructure:"auto" json:"auto,omitempty"`   // enable auto-compaction (default true)
	Prune *bool `mapstructure:"prune" json:"prune,omitempty"` // enable pruning (default true)
}

// FormatterConfig (opencode)
type FormatterConfig struct {
	Disabled   bool              `mapstructure:"disabled" json:"disabled,omitempty"`
	Command    []string          `mapstructure:"command" json:"command,omitempty"`
	Env        map[string]string `mapstructure:"env" json:"env,omitempty"`
	Extensions []string          `mapstructure:"extensions" json:"extensions,omitempty"`
}

// LSPConfig (opencode)
type LSPConfig struct {
	Disabled       bool              `mapstructure:"disabled" json:"disabled,omitempty"`
	Command        []string          `mapstructure:"command" json:"command,omitempty"`
	Extensions     []string          `mapstructure:"extensions" json:"extensions,omitempty"`
	Env            map[string]string `mapstructure:"env" json:"env,omitempty"`
	Initialization map[string]any    `mapstructure:"initialization" json:"initialization,omitempty"`
}

// ExperimentalConfig (opencode)
type ExperimentalConfig struct {
	DisablePasteSummary bool     `mapstructure:"disable_paste_summary" json:"disable_paste_summary,omitempty"`
	BatchTool           bool     `mapstructure:"batch_tool" json:"batch_tool,omitempty"`
	OpenTelemetry       bool     `mapstructure:"open_telemetry" json:"open_telemetry,omitempty"`
	PrimaryTools        []string `mapstructure:"primary_tools" json:"primary_tools,omitempty"`
	ContinueLoopOnDeny  bool     `mapstructure:"continue_loop_on_deny" json:"continue_loop_on_deny,omitempty"`
	MCPTimeout          int      `mapstructure:"mcp_timeout" json:"mcp_timeout,omitempty"` // ms
}

// ModelInfo contains information about a specific model (unchanged for backward compat)
type ModelInfo struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Provider      string `json:"provider"`
	ContextWindow int    `json:"context_window"`
	MaxOutput     int    `json:"max_output"`
	SupportsTools bool   `json:"supports_tools"`
}

// ---------------------------------------------------------------------------
// Default models – expanded to cover all 20 providers
// ---------------------------------------------------------------------------

// DefaultModels maps provider names to their default model configurations
var DefaultModels = map[string]ModelInfo{
	"anthropic": {
		ID: "claude-sonnet-4-5", Name: "Claude Sonnet 4.5", Provider: "anthropic",
		ContextWindow: 200000, MaxOutput: 16384, SupportsTools: true,
	},
	"openai": {
		ID: "gpt-5.1-codex", Name: "GPT-5.1 Codex", Provider: "openai",
		ContextWindow: 1047576, MaxOutput: 32768, SupportsTools: true,
	},
	"copilot": {
		ID: "gpt-4o", Name: "GPT-4o (Copilot)", Provider: "copilot",
		ContextWindow: 128000, MaxOutput: 16384, SupportsTools: true,
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
		ID: "anthropic/claude-sonnet-4.5", Name: "Claude Sonnet 4.5 (OpenRouter)", Provider: "openrouter",
		ContextWindow: 200000, MaxOutput: 16384, SupportsTools: true,
	},
	"xai": {
		ID: "grok-4", Name: "Grok 4", Provider: "xai",
		ContextWindow: 131072, MaxOutput: 16384, SupportsTools: true,
	},
	"deepseek": {
		ID: "deepseek-chat", Name: "DeepSeek Chat", Provider: "deepseek",
		ContextWindow: 65536, MaxOutput: 8192, SupportsTools: true,
	},
	"mistral": {
		ID: "devstral-medium-latest", Name: "Devstral Medium", Provider: "mistral",
		ContextWindow: 131072, MaxOutput: 8192, SupportsTools: true,
	},
	"deepinfra": {
		ID: "deepseek-ai/DeepSeek-V3.2", Name: "DeepSeek V3.2", Provider: "deepinfra",
		ContextWindow: 131072, MaxOutput: 8192, SupportsTools: true,
	},
	"cerebras": {
		ID: "qwen-3-235b-a22b-instruct-2507", Name: "Qwen 3 235B", Provider: "cerebras",
		ContextWindow: 128000, MaxOutput: 8192, SupportsTools: true,
	},
	"together": {
		ID: "meta-llama/Llama-3.3-70B-Instruct-Turbo", Name: "Llama 3.3 70B (Together)", Provider: "together",
		ContextWindow: 131072, MaxOutput: 4096, SupportsTools: true,
	},
	"cohere": {
		ID: "command-r-plus", Name: "Command R+", Provider: "cohere",
		ContextWindow: 128000, MaxOutput: 4096, SupportsTools: true,
	},
	"perplexity": {
		ID: "sonar-pro", Name: "Sonar Pro", Provider: "perplexity",
		ContextWindow: 200000, MaxOutput: 8192, SupportsTools: false,
	},
	"azure": {
		ID: "gpt-5.1-codex", Name: "GPT-5.1 Codex (Azure)", Provider: "azure",
		ContextWindow: 1047576, MaxOutput: 32768, SupportsTools: true,
	},
	"bedrock": {
		ID: "anthropic.claude-sonnet-4-5-20250929-v1:0", Name: "Claude Sonnet 4.5 (Bedrock)", Provider: "bedrock",
		ContextWindow: 200000, MaxOutput: 16384, SupportsTools: true,
	},
	"google-vertex": {
		ID: "gemini-2.5-flash", Name: "Gemini 2.5 Flash (Vertex)", Provider: "google-vertex",
		ContextWindow: 1048576, MaxOutput: 65536, SupportsTools: true,
	},
	"gitlab": {
		ID: "duo-chat-sonnet-4-5", Name: "Claude Sonnet 4.5 (GitLab)", Provider: "gitlab",
		ContextWindow: 200000, MaxOutput: 16384, SupportsTools: true,
	},
	"cloudflare-workers-ai": {
		ID: "@cf/openai/gpt-oss-120b", Name: "GPT-OSS 120B (CF)", Provider: "cloudflare-workers-ai",
		ContextWindow: 131072, MaxOutput: 4096, SupportsTools: true,
	},
	"replicate": {
		ID: "meta/meta-llama-3-70b-instruct", Name: "Llama 3 70B (Replicate)", Provider: "replicate",
		ContextWindow: 8192, MaxOutput: 4096, SupportsTools: false,
	},
	"sambanova": {
		ID: "Meta-Llama-3.3-70B-Instruct", Name: "Llama 3.3 70B (SambaNova)", Provider: "sambanova",
		ContextWindow: 128000, MaxOutput: 8192, SupportsTools: true,
	},
	"fireworks": {
		ID: "accounts/fireworks/models/llama-v3p3-70b-instruct", Name: "Llama 3.3 70B (Fireworks)", Provider: "fireworks",
		ContextWindow: 131072, MaxOutput: 4096, SupportsTools: true,
	},
	"huggingface": {
		ID: "meta-llama/Llama-3.3-70B-Instruct", Name: "Llama 3.3 70B (HF)", Provider: "huggingface",
		ContextWindow: 128000, MaxOutput: 4096, SupportsTools: true,
	},
}

// ---------------------------------------------------------------------------
// JSONC helpers
// ---------------------------------------------------------------------------

var (
	reLineComment   = regexp.MustCompile(`(?m)^\s*//.*$`)
	reBlockComment  = regexp.MustCompile(`(?s)/\*.*?\*/`)
	reTrailingComma = regexp.MustCompile(`,\s*([\]}])`)
	reEnvVar        = regexp.MustCompile(`\{env:([^}]+)\}`)
	reFileRef       = regexp.MustCompile(`\{file:([^}]+)\}`)
)

// stripJSONC removes comments and trailing commas to produce valid JSON.
func stripJSONC(text string) string {
	text = reBlockComment.ReplaceAllString(text, "")
	text = reLineComment.ReplaceAllString(text, "")
	text = reTrailingComma.ReplaceAllString(text, "$1")
	return text
}

// substituteEnvVars replaces {env:VAR} with os.Getenv(VAR).
func substituteEnvVars(text string) string {
	return reEnvVar.ReplaceAllStringFunc(text, func(m string) string {
		parts := reEnvVar.FindStringSubmatch(m)
		if len(parts) < 2 {
			return m
		}
		return os.Getenv(parts[1])
	})
}

// substituteFileRefs replaces {file:path} with file contents.
// Paths starting with ~/ are expanded. Relative paths resolve from baseDir.
// Commented lines (starting with //) are skipped.
func substituteFileRefs(text string, baseDir string) string {
	return reFileRef.ReplaceAllStringFunc(text, func(m string) string {
		parts := reFileRef.FindStringSubmatch(m)
		if len(parts) < 2 {
			return m
		}
		filePath := parts[1]

		// Expand ~/
		if strings.HasPrefix(filePath, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				filePath = filepath.Join(home, filePath[2:])
			}
		} else if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(baseDir, filePath)
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			return "" // silently skip missing files
		}
		return strings.TrimSpace(string(data))
	})
}

// loadJSONC reads a JSONC file with env/file substitution and returns raw JSON.
func loadJSONC(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := string(data)
	text = substituteEnvVars(text)
	text = substituteFileRefs(text, filepath.Dir(path))
	text = stripJSONC(text)
	return []byte(text), nil
}

// ---------------------------------------------------------------------------
// Config loading – multi-source precedence (matching opencode)
// ---------------------------------------------------------------------------

// Load loads configuration from all sources with proper precedence:
// defaults → global config → custom config → project config → .dcode dirs → inline → managed
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("provider", "anthropic")
	v.SetDefault("model", "")
	v.SetDefault("small_model", "")
	v.SetDefault("max_tokens", 12288)
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

	_ = v.BindEnv("openai_api_key", "OPENAI_API_KEY")

	_ = v.BindEnv("google_api_key", "GOOGLE_API_KEY", "GEMINI_API_KEY")
	_ = v.BindEnv("groq_api_key", "GROQ_API_KEY")
	_ = v.BindEnv("openrouter_api_key", "OPENROUTER_API_KEY")

	_ = v.ReadInConfig()

	// Try .dcode/config.json as well (project-level JSON override)
	jsonPath := filepath.Join(".dcode", "config.json")
	if data, err := os.ReadFile(jsonPath); err == nil {
		var jsonCfg map[string]interface{}
		if json.Unmarshal(data, &jsonCfg) == nil {
			for k, val := range jsonCfg {
				v.Set(k, val)
			}
		}
	}

	// Try JSONC config files from .dcode/ and global directories
	var configDirs []string
	jsoncFiles := []string{}

	// Global config dir
	if home != "" {
		globalDir := filepath.Join(home, ".config", "dcode")
		configDirs = append(configDirs, globalDir)
		jsoncFiles = append(jsoncFiles,
			filepath.Join(globalDir, "dcode.jsonc"),
			filepath.Join(globalDir, "dcode.json"),
		)
	}

	// Project .dcode dir
	projectDcodeDir := ".dcode"
	if info, err := os.Stat(projectDcodeDir); err == nil && info.IsDir() {
		configDirs = append(configDirs, projectDcodeDir)
		jsoncFiles = append(jsoncFiles,
			filepath.Join(projectDcodeDir, "dcode.jsonc"),
			filepath.Join(projectDcodeDir, "dcode.json"),
		)
	}

	// Walk up to find .dcode directories (matching opencode's Filesystem.up)
	if cwd, err := os.Getwd(); err == nil {
		for dir := cwd; dir != "/" && dir != "."; dir = filepath.Dir(dir) {
			candidate := filepath.Join(dir, ".dcode")
			if candidate == projectDcodeDir {
				continue // already handled
			}
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				configDirs = append(configDirs, candidate)
				jsoncFiles = append(jsoncFiles,
					filepath.Join(candidate, "dcode.jsonc"),
					filepath.Join(candidate, "dcode.json"),
				)
			}
		}
	}

	// Home .dcode directory
	if home != "" {
		homeDcodeDir := filepath.Join(home, ".dcode")
		if info, err := os.Stat(homeDcodeDir); err == nil && info.IsDir() {
			found := false
			for _, d := range configDirs {
				if d == homeDcodeDir {
					found = true
					break
				}
			}
			if !found {
				configDirs = append(configDirs, homeDcodeDir)
				jsoncFiles = append(jsoncFiles,
					filepath.Join(homeDcodeDir, "dcode.jsonc"),
					filepath.Join(homeDcodeDir, "dcode.json"),
				)
			}
		}
	}

	// Custom config dir from env
	if customDir := os.Getenv(EnvConfigDir); customDir != "" {
		if info, err := os.Stat(customDir); err == nil && info.IsDir() {
			configDirs = append(configDirs, customDir)
			jsoncFiles = append(jsoncFiles,
				filepath.Join(customDir, "dcode.jsonc"),
				filepath.Join(customDir, "dcode.json"),
			)
		}
	}

	// Load JSONC overlays (each merges on top)
	for _, f := range jsoncFiles {
		if data, err := loadJSONC(f); err == nil {
			var overlay map[string]interface{}
			if json.Unmarshal(data, &overlay) == nil {
				for k, val := range overlay {
					v.Set(k, val)
				}
			}
		}
	}

	// Custom config file from env (DCODE_CONFIG)
	if customConfig := os.Getenv(EnvConfig); customConfig != "" {
		if data, err := loadJSONC(customConfig); err == nil {
			var overlay map[string]interface{}
			if json.Unmarshal(data, &overlay) == nil {
				for k, val := range overlay {
					v.Set(k, val)
				}
			}
		}
	}

	// Inline config from env (DCODE_CONFIG_CONTENT)
	if content := os.Getenv(EnvConfigContent); content != "" {
		var overlay map[string]interface{}
		if json.Unmarshal([]byte(content), &overlay) == nil {
			for k, val := range overlay {
				v.Set(k, val)
			}
		}
	}

	// Managed config directory (enterprise – highest priority)
	managedDir := getManagedConfigDir()
	for _, name := range []string{"dcode.jsonc", "dcode.json"} {
		f := filepath.Join(managedDir, name)
		if data, err := loadJSONC(f); err == nil {
			var overlay map[string]interface{}
			if json.Unmarshal(data, &overlay) == nil {
				for k, val := range overlay {
					v.Set(k, val)
				}
			}
		}
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	// Store discovered config directories
	config.configDirectories = configDirs

	// Set session directory
	if config.SessionDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			config.SessionDir = filepath.Join(home, ".config", "dcode", "sessions")
		}
	}

	// Set default username
	if config.Username == "" {
		if u, err := user.Current(); err == nil {
			config.Username = u.Username
		}
	}

	// Apply env var overrides (opencode)
	if permJSON := os.Getenv(EnvPermission); permJSON != "" {
		var perm PermissionConfig
		if json.Unmarshal([]byte(permJSON), &perm) == nil {
			mergePermissions(&config.Permissions, &perm)
		}
	}
	if isTruthy(os.Getenv(EnvDisableAutoCompact)) {
		if config.CompactionConfig == nil {
			config.CompactionConfig = &CompactionConfig{}
		}
		f := false
		config.CompactionConfig.Auto = &f
	}
	if isTruthy(os.Getenv(EnvDisablePrune)) {
		if config.CompactionConfig == nil {
			config.CompactionConfig = &CompactionConfig{}
		}
		f := false
		config.CompactionConfig.Prune = &f
	}

	// Load instructions, commands, and agents from .dcode directories
	config.loadInstructions()
	config.loadCommandsFromDirs()

	return &config, nil
}

// ---------------------------------------------------------------------------
// Managed config directory (enterprise / admin-controlled)
// ---------------------------------------------------------------------------

func getManagedConfigDir() string {
	if dir := os.Getenv(EnvTestManagedConfigDir); dir != "" {
		return dir
	}
	switch runtime.GOOS {
	case "darwin":
		return "/Library/Application Support/dcode"
	case "windows":
		if pd := os.Getenv("ProgramData"); pd != "" {
			return filepath.Join(pd, "dcode")
		}
		return `C:\ProgramData\dcode`
	default: // linux
		return "/etc/dcode"
	}
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

func isTruthy(s string) bool {
	s = strings.ToLower(s)
	return s == "true" || s == "1"
}

func mergePermissions(dst, src *PermissionConfig) {
	if src.Bash != "" {
		dst.Bash = src.Bash
	}
	if src.ExternalDirectory != "" {
		dst.ExternalDirectory = src.ExternalDirectory
	}
	if src.Read != "" {
		dst.Read = src.Read
	}
	if src.Glob != "" {
		dst.Glob = src.Glob
	}
	if src.Grep != "" {
		dst.Grep = src.Grep
	}
	if src.List != "" {
		dst.List = src.List
	}
	if src.Task != "" {
		dst.Task = src.Task
	}
	if src.WebFetch != "" {
		dst.WebFetch = src.WebFetch
	}
	if src.WebSearch != "" {
		dst.WebSearch = src.WebSearch
	}
	if src.CodeSearch != "" {
		dst.CodeSearch = src.CodeSearch
	}
	if src.LSP != "" {
		dst.LSP = src.LSP
	}
	if src.Question != "" {
		dst.Question = src.Question
	}
	if src.Skill != "" {
		dst.Skill = src.Skill
	}
	if src.TodoWrite != "" {
		dst.TodoWrite = src.TodoWrite
	}
	if src.TodoRead != "" {
		dst.TodoRead = src.TodoRead
	}
	if src.DoomLoop != "" {
		dst.DoomLoop = src.DoomLoop
	}
	if len(src.Edit) > 0 {
		if dst.Edit == nil {
			dst.Edit = make(map[string]string)
		}
		for k, v := range src.Edit {
			dst.Edit[k] = v
		}
	}
	if len(src.Write) > 0 {
		if dst.Write == nil {
			dst.Write = make(map[string]string)
		}
		for k, v := range src.Write {
			dst.Write[k] = v
		}
	}
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

// loadCommandsFromDirs scans .dcode/{commands,command}/**/*.md for slash commands
func (c *Config) loadCommandsFromDirs() {
	if c.Commands == nil {
		c.Commands = make(map[string]CommandConfig)
	}

	for _, dir := range c.configDirectories {
		for _, subdir := range []string{"commands", "command"} {
			cmdDir := filepath.Join(dir, subdir)
			_ = filepath.Walk(cmdDir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
					return nil
				}
				data, err := os.ReadFile(path)
				if err != nil {
					return nil
				}
				content := string(data)

				// Parse YAML frontmatter
				var meta map[string]string
				body := content
				if strings.HasPrefix(content, "---\n") {
					parts := strings.SplitN(content[4:], "\n---\n", 2)
					if len(parts) == 2 {
						meta = parseSimpleYAML(parts[0])
						body = strings.TrimSpace(parts[1])
					}
				}

				// Derive name from relative path
				relPath, _ := filepath.Rel(cmdDir, path)
				name := strings.TrimSuffix(relPath, ".md")
				name = strings.ReplaceAll(name, string(filepath.Separator), "/")

				cmd := CommandConfig{
					Template: body,
				}
				if meta != nil {
					if v, ok := meta["description"]; ok {
						cmd.Description = v
					}
					if v, ok := meta["agent"]; ok {
						cmd.Agent = v
					}
					if v, ok := meta["model"]; ok {
						cmd.Model = v
					}
					if v, ok := meta["subtask"]; ok {
						cmd.Subtask = isTruthy(v)
					}
				}

				c.Commands[name] = cmd
				return nil
			})
		}
	}
}

// parseSimpleYAML does minimal YAML frontmatter parsing for key: value pairs.
func parseSimpleYAML(text string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		// Strip surrounding quotes
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		result[key] = val
	}
	return result
}

// Directories returns the list of config directories that were scanned.
func (c *Config) Directories() []string {
	return c.configDirectories
}

// IsAutoCompactionEnabled returns whether auto-compaction is enabled (default: true)
func (c *Config) IsAutoCompactionEnabled() bool {
	if c.CompactionConfig != nil && c.CompactionConfig.Auto != nil {
		return *c.CompactionConfig.Auto
	}
	return c.Compaction
}

// IsPruningEnabled returns whether tool-output pruning is enabled (default: true)
func (c *Config) IsPruningEnabled() bool {
	if c.CompactionConfig != nil && c.CompactionConfig.Prune != nil {
		return *c.CompactionConfig.Prune
	}
	return true
}

// ---------------------------------------------------------------------------
// API Key / Provider helpers – expanded to all 20 providers
// ---------------------------------------------------------------------------

// GetAPIKey returns the API key for the specified provider from config fields.
func (c *Config) GetAPIKey(providerName string) string {
	// 1. Check provider overrides (highest priority)
	if po, ok := c.Providers[providerName]; ok {
		if po.APIKey != "" {
			return po.APIKey
		}
		if po.Options != nil && po.Options.APIKey != "" {
			return po.Options.APIKey
		}
	}

	// 2. Legacy top-level fields
	switch providerName {
	case "openai":
		return c.OpenAIAPIKey
	case "google":
		return c.GoogleAPIKey
	case "groq":
		return c.GroqAPIKey
	case "openrouter":
		return c.OpenRouterKey
	}

	// 3. Environment-variable-mapped fields embedded in the config (users sometimes
	//    place API keys directly under the provider name in YAML).
	//    We support these well-known env var names as top-level config aliases.
	envAliases := map[string]string{
		"xai":                   "XAI_API_KEY",
		"deepseek":              "DEEPSEEK_API_KEY",
		"mistral":               "MISTRAL_API_KEY",
		"deepinfra":             "DEEPINFRA_API_KEY",
		"cerebras":              "CEREBRAS_API_KEY",
		"together":              "TOGETHER_API_KEY",
		"cohere":                "COHERE_API_KEY",
		"perplexity":            "PERPLEXITY_API_KEY",
		"replicate":             "REPLICATE_API_TOKEN",
		"azure":                 "AZURE_OPENAI_API_KEY",
		"bedrock":               "AWS_ACCESS_KEY_ID",
		"google-vertex":         "GOOGLE_CLOUD_PROJECT",
		"gitlab":                "GITLAB_TOKEN",
		"cloudflare-workers-ai": "CLOUDFLARE_API_TOKEN",
	}
	if envVar, ok := envAliases[providerName]; ok {
		if v := os.Getenv(envVar); v != "" {
			return v
		}
	}

	return ""
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

// GetSmallModel returns the small/fast model for automated tasks (title generation, compaction).
// It is provider-aware: Copilot uses gpt-4o-mini, Anthropic uses claude-haiku-4-5, etc.
func (c *Config) GetSmallModel() string {
	if c.SmallModel != "" {
		return c.SmallModel
	}
	smallModels := map[string]string{
		"copilot":       "gpt-4o-mini",
		"openai":        "gpt-4o-mini",
		"azure":         "gpt-4o-mini",
		"anthropic":     "claude-haiku-4-5",
		"google":        "gemini-2.5-flash",
		"google-vertex": "gemini-2.5-flash",
		"groq":          "llama-3.1-8b-instant",
		"deepseek":      "deepseek-chat",
		"xai":           "grok-3-mini-fast",
		"mistral":       "mistral-small-latest",
		"openrouter":    "anthropic/claude-haiku-4-5",
	}
	if m, ok := smallModels[c.Provider]; ok {
		return m
	}
	return "gpt-4o-mini" // safe default that works on most providers
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
	return os.WriteFile(path, data, 0600)
}

// ListAvailableProviders returns providers that have API keys configured.
// Expanded to check all 20 providers.
func (c *Config) ListAvailableProviders() []string {
	creds, _ := LoadCredentials()
	providers := []string{}

	// All known providers and their env vars / credential getters
	type providerCheck struct {
		name    string
		envVars []string
		getCred func(*Credentials) string
	}

	allProviders := []providerCheck{
		{"anthropic", []string{}, func(_ *Credentials) string { return "" }},
		{"openai", []string{"OPENAI_API_KEY"}, func(cr *Credentials) string { return cr.OpenAIAPIKey }},
		{"copilot", []string{}, func(_ *Credentials) string { return "" }},
		{"google", []string{"GOOGLE_API_KEY", "GEMINI_API_KEY"}, func(cr *Credentials) string { return cr.GoogleAPIKey }},
		{"groq", []string{"GROQ_API_KEY"}, func(cr *Credentials) string { return cr.GroqAPIKey }},
		{"openrouter", []string{"OPENROUTER_API_KEY"}, func(cr *Credentials) string { return cr.OpenRouterKey }},
		{"xai", []string{"XAI_API_KEY"}, func(cr *Credentials) string { return cr.XAIAPIKey }},
		{"deepseek", []string{"DEEPSEEK_API_KEY"}, func(cr *Credentials) string { return cr.DeepSeekAPIKey }},
		{"mistral", []string{"MISTRAL_API_KEY"}, func(cr *Credentials) string { return cr.MistralAPIKey }},
		{"deepinfra", []string{"DEEPINFRA_API_KEY"}, func(cr *Credentials) string { return cr.DeepInfraAPIKey }},
		{"cerebras", []string{"CEREBRAS_API_KEY"}, func(cr *Credentials) string { return cr.CerebrasAPIKey }},
		{"together", []string{"TOGETHER_API_KEY", "TOGETHERAI_API_KEY"}, func(cr *Credentials) string { return cr.TogetherAPIKey }},
		{"cohere", []string{"COHERE_API_KEY", "CO_API_KEY"}, func(cr *Credentials) string { return cr.CohereAPIKey }},
		{"perplexity", []string{"PERPLEXITY_API_KEY"}, func(cr *Credentials) string { return cr.PerplexityAPIKey }},
		{"azure", []string{"AZURE_OPENAI_API_KEY", "AZURE_API_KEY"}, func(cr *Credentials) string { return cr.AzureAPIKey }},
		{"bedrock", []string{"AWS_ACCESS_KEY_ID"}, func(_ *Credentials) string { return "" }},
		{"google-vertex", []string{"GOOGLE_CLOUD_PROJECT"}, func(_ *Credentials) string { return "" }},
		{"gitlab", []string{"GITLAB_TOKEN", "GITLAB_API_TOKEN"}, func(cr *Credentials) string { return cr.GitLabToken }},
		{"cloudflare-workers-ai", []string{"CLOUDFLARE_API_TOKEN"}, func(cr *Credentials) string { return cr.CloudflareAPIToken }},
		{"replicate", []string{"REPLICATE_API_TOKEN"}, func(cr *Credentials) string { return cr.ReplicateAPIToken }},
	}

	for _, pc := range allProviders {
		found := false

		// Check env vars
		for _, ev := range pc.envVars {
			if os.Getenv(ev) != "" {
				found = true
				break
			}
		}

		// Check stored credentials
		if !found && creds != nil {
			if pc.getCred(creds) != "" {
				found = true
			}
		}

		// Check config file
		if !found && c.GetAPIKey(pc.name) != "" {
			found = true
		}

		// Special: anthropic uses OAuth token stored in credentials.json
		if !found && pc.name == "anthropic" {
			if creds != nil && creds.OAuthTokens != nil {
				if t, ok := creds.OAuthTokens["anthropic"]; ok && t != nil && t.AccessToken != "" {
					found = true
				}
			}
		}

		// Special: copilot uses OAuth token file only
		if !found && pc.name == "copilot" {
			if home, err := os.UserHomeDir(); err == nil {
				tokenPath := filepath.Join(home, ".config", "dcode", "copilot_oauth.json")
				if _, err := os.Stat(tokenPath); err == nil {
					found = true
				}
			}
		}

		if found {
			providers = append(providers, pc.name)
		}
	}

	return providers
}

// IsProviderEnabled checks if a provider is not disabled and (if enabledProviders is set) is in the whitelist.
func (c *Config) IsProviderEnabled(name string) bool {
	for _, d := range c.DisabledProviders {
		if d == name {
			return false
		}
	}
	if len(c.EnabledProviders) > 0 {
		for _, e := range c.EnabledProviders {
			if e == name {
				return true
			}
		}
		return false
	}
	return true
}

// GetKeybinds returns keybinds with defaults applied.
func (c *Config) GetKeybinds() *Keybinds {
	defaults := DefaultKeybinds()
	if c.Keybinds == nil {
		return defaults
	}
	// Merge: only override if set
	kb := *c.Keybinds
	if kb.Leader == "" {
		kb.Leader = defaults.Leader
	}
	if kb.AppExit == "" {
		kb.AppExit = defaults.AppExit
	}
	if kb.EditorOpen == "" {
		kb.EditorOpen = defaults.EditorOpen
	}
	if kb.ThemeList == "" {
		kb.ThemeList = defaults.ThemeList
	}
	if kb.SidebarToggle == "" {
		kb.SidebarToggle = defaults.SidebarToggle
	}
	if kb.StatusView == "" {
		kb.StatusView = defaults.StatusView
	}
	if kb.SessionExport == "" {
		kb.SessionExport = defaults.SessionExport
	}
	if kb.SessionNew == "" {
		kb.SessionNew = defaults.SessionNew
	}
	if kb.SessionList == "" {
		kb.SessionList = defaults.SessionList
	}
	if kb.SessionTimeline == "" {
		kb.SessionTimeline = defaults.SessionTimeline
	}
	if kb.SessionRename == "" {
		kb.SessionRename = defaults.SessionRename
	}
	if kb.SessionDelete == "" {
		kb.SessionDelete = defaults.SessionDelete
	}
	if kb.SessionInterrupt == "" {
		kb.SessionInterrupt = defaults.SessionInterrupt
	}
	if kb.SessionCompact == "" {
		kb.SessionCompact = defaults.SessionCompact
	}
	if kb.ModelList == "" {
		kb.ModelList = defaults.ModelList
	}
	if kb.ModelProviderList == "" {
		kb.ModelProviderList = defaults.ModelProviderList
	}
	if kb.ModelFavoriteToggle == "" {
		kb.ModelFavoriteToggle = defaults.ModelFavoriteToggle
	}
	if kb.ModelCycleRecent == "" {
		kb.ModelCycleRecent = defaults.ModelCycleRecent
	}
	if kb.ModelCycleRecentReverse == "" {
		kb.ModelCycleRecentReverse = defaults.ModelCycleRecentReverse
	}
	if kb.CommandList == "" {
		kb.CommandList = defaults.CommandList
	}
	if kb.AgentList == "" {
		kb.AgentList = defaults.AgentList
	}
	if kb.AgentCycle == "" {
		kb.AgentCycle = defaults.AgentCycle
	}
	if kb.AgentCycleReverse == "" {
		kb.AgentCycleReverse = defaults.AgentCycleReverse
	}
	if kb.VariantCycle == "" {
		kb.VariantCycle = defaults.VariantCycle
	}
	if kb.InputClear == "" {
		kb.InputClear = defaults.InputClear
	}
	if kb.InputPaste == "" {
		kb.InputPaste = defaults.InputPaste
	}
	if kb.InputSubmit == "" {
		kb.InputSubmit = defaults.InputSubmit
	}
	if kb.InputNewline == "" {
		kb.InputNewline = defaults.InputNewline
	}
	if kb.MessagesPageUp == "" {
		kb.MessagesPageUp = defaults.MessagesPageUp
	}
	if kb.MessagesPageDown == "" {
		kb.MessagesPageDown = defaults.MessagesPageDown
	}
	if kb.MessagesFirst == "" {
		kb.MessagesFirst = defaults.MessagesFirst
	}
	if kb.MessagesLast == "" {
		kb.MessagesLast = defaults.MessagesLast
	}
	if kb.MessagesCopy == "" {
		kb.MessagesCopy = defaults.MessagesCopy
	}
	if kb.MessagesUndo == "" {
		kb.MessagesUndo = defaults.MessagesUndo
	}
	if kb.MessagesRedo == "" {
		kb.MessagesRedo = defaults.MessagesRedo
	}
	if kb.MessagesToggleConceal == "" {
		kb.MessagesToggleConceal = defaults.MessagesToggleConceal
	}
	if kb.HistoryPrevious == "" {
		kb.HistoryPrevious = defaults.HistoryPrevious
	}
	if kb.HistoryNext == "" {
		kb.HistoryNext = defaults.HistoryNext
	}
	if kb.TerminalSuspend == "" {
		kb.TerminalSuspend = defaults.TerminalSuspend
	}
	return &kb
}

// String returns a human-readable representation
func (c *Config) String() string {
	return fmt.Sprintf("Config{Provider: %s, Model: %s, MaxTokens: %d}", c.Provider, c.Model, c.MaxTokens)
}
