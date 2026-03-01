package tool

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

// DiffData holds before/after content for rendering side-by-side diffs
type DiffData struct {
	OldContent string `json:"old_content"`
	NewContent string `json:"new_content"`
	FilePath   string `json:"file_path,omitempty"`
	Language   string `json:"language,omitempty"`
	IsFragment bool   `json:"is_fragment,omitempty"` // true for edit (partial), false for write (full file)
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Output       string           `json:"output"`
	IsError      bool             `json:"is_error"`
	Title        string           `json:"title,omitempty"`          // Optional title for tool output display
	Attachments  []FileAttachment `json:"attachments,omitempty"`    // File attachments (images, PDFs)
	DiffData     *DiffData        `json:"diff_data,omitempty"`      // Single diff (edit, write)
	DiffDataList []*DiffData      `json:"diff_data_list,omitempty"` // Multiple diffs (multiedit, patch)
}

// FileAttachment represents a base64-encoded file attachment
type FileAttachment struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id,omitempty"`
	MessageID string `json:"message_id,omitempty"`
	Type      string `json:"type"` // "file"
	MIME      string `json:"mime"` // e.g. "image/png", "application/pdf"
	URL       string `json:"url"`  // data URL: "data:<mime>;base64,<data>"
	Filename  string `json:"filename,omitempty"`
}

// ToolContext provides context for tool execution
type ToolContext struct {
	SessionID  string
	MessageID  string
	WorkDir    string
	Abort      context.Context
	OnQuestion QuestionAskFn // Optional: wired by TUI to show interactive question dialog
}

// ToolDef defines a tool that the AI can use
type ToolDef struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
	Execute     func(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error)
}

// Registry manages all available tools
type Registry struct {
	mu    sync.RWMutex
	tools map[string]*ToolDef
}

var (
	globalRegistry *Registry
	once           sync.Once
)

// GetRegistry returns the global tool registry
func GetRegistry() *Registry {
	once.Do(func() {
		globalRegistry = &Registry{
			tools: make(map[string]*ToolDef),
		}
		registerBuiltinTools(globalRegistry)
	})
	return globalRegistry
}

// Register adds a tool to the registry
func (r *Registry) Register(tool *ToolDef) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name] = tool
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (*ToolDef, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// List returns all registered tool names
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// GetAll returns all registered tools
func (r *Registry) GetAll() map[string]*ToolDef {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]*ToolDef, len(r.tools))
	for k, v := range r.tools {
		result[k] = v
	}
	return result
}

// GetFiltered returns tools filtered by allowed names (empty = all)
func (r *Registry) GetFiltered(allowed []string) map[string]*ToolDef {
	if len(allowed) == 0 {
		return r.GetAll()
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]*ToolDef)
	for _, name := range allowed {
		if t, ok := r.tools[name]; ok {
			result[name] = t
		}
	}
	return result
}

// Execute runs a tool by name with the given input
func (r *Registry) Execute(ctx context.Context, tc *ToolContext, name string, input map[string]interface{}) (*ToolResult, error) {
	tool, ok := r.Get(name)
	if !ok {
		return &ToolResult{
			Output:  fmt.Sprintf("Unknown tool: %s. Available tools: %v", name, r.List()),
			IsError: true,
		}, nil
	}
	return tool.Execute(ctx, tc, input)
}

// ToProviderTools converts registry tools to provider-compatible tool definitions
func (r *Registry) ToProviderTools(allowed []string) []ProviderTool {
	tools := r.GetFiltered(allowed)
	result := make([]ProviderTool, 0, len(tools))
	for _, t := range tools {
		result = append(result, ProviderTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.Parameters,
		})
	}
	return result
}

// ProviderTool is a simplified tool definition for LLM providers
type ProviderTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// inferLanguage returns a language identifier based on file extension
func inferLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	langs := map[string]string{
		".go":    "go",
		".js":    "javascript",
		".ts":    "typescript",
		".tsx":   "tsx",
		".jsx":   "jsx",
		".py":    "python",
		".rb":    "ruby",
		".rs":    "rust",
		".java":  "java",
		".c":     "c",
		".cpp":   "cpp",
		".h":     "c",
		".hpp":   "cpp",
		".cs":    "csharp",
		".swift": "swift",
		".kt":    "kotlin",
		".lua":   "lua",
		".sh":    "bash",
		".bash":  "bash",
		".zsh":   "zsh",
		".fish":  "fish",
		".yaml":  "yaml",
		".yml":   "yaml",
		".json":  "json",
		".toml":  "toml",
		".xml":   "xml",
		".html":  "html",
		".css":   "css",
		".scss":  "scss",
		".sql":   "sql",
		".md":    "markdown",
		".proto": "protobuf",
		".tf":    "hcl",
		".vim":   "vim",
		".el":    "elisp",
		".ex":    "elixir",
		".exs":   "elixir",
		".zig":   "zig",
		".v":     "v",
		".dart":  "dart",
		".r":     "r",
		".R":     "r",
		".php":   "php",
		".pl":    "perl",
	}
	if lang, ok := langs[ext]; ok {
		return lang
	}
	return ""
}

// registerBuiltinTools registers all built-in tools
func registerBuiltinTools(r *Registry) {
	// Core file operations
	r.Register(ReadTool())
	r.Register(WriteTool())
	r.Register(EditTool())
	r.Register(MultiEditTool())
	r.Register(PatchTool())
	r.Register(ApplyPatchTool())

	// Shell and search
	r.Register(BashTool())
	r.Register(GlobTool())
	r.Register(GrepTool())
	r.Register(LsTool())
	r.Register(CodeSearchTool())

	// Web and external
	r.Register(WebFetchTool())
	r.Register(WebSearchTool())

	// Task management
	r.Register(TodoReadTool())
	r.Register(TodoWriteTool())
	r.Register(TaskTool())

	// Interactive
	r.Register(QuestionTool())

	// Skills
	r.Register(SkillTool())

	// Batch operations
	r.Register(BatchTool())

	// Plan mode
	r.Register(PlanEnterTool())
	r.Register(PlanExitTool())

	// Development tools
	r.Register(GitTool())
	r.Register(LSPTool())
	r.Register(MCPTool())
	r.Register(DockerTool())
	r.Register(ImageTool())
}
