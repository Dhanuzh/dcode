package tool

import (
	"context"
	"fmt"
	"sync"
)

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Output  string `json:"output"`
	IsError bool   `json:"is_error"`
}

// ToolContext provides context for tool execution
type ToolContext struct {
	SessionID string
	MessageID string
	WorkDir   string
	Abort     context.Context
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

// registerBuiltinTools registers all built-in tools
func registerBuiltinTools(r *Registry) {
	r.Register(ReadTool())
	r.Register(WriteTool())
	r.Register(EditTool())
	r.Register(MultiEditTool())
	r.Register(PatchTool())
	r.Register(BashTool())
	r.Register(GlobTool())
	r.Register(GrepTool())
	r.Register(LsTool())
	r.Register(WebFetchTool())
	r.Register(TodoReadTool())
	r.Register(TodoWriteTool())
	r.Register(TaskTool())
	r.Register(GitTool())
	r.Register(WebSearchTool())
	r.Register(LSPTool())
	r.Register(MCPTool())
	r.Register(DockerTool())
	r.Register(ImageTool())
}
