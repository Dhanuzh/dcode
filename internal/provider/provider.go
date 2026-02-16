package provider

import (
	"context"
	"encoding/json"
)

// Provider defines the interface for AI providers
type Provider interface {
	Name() string
	CreateMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error)
	StreamMessage(ctx context.Context, req *MessageRequest, callback func(*StreamChunk) error) error
	Models() []string
}

// Message represents a conversation message
type Message struct {
	Role    string      `json:"role"`    // "user", "assistant", "system"
	Content interface{} `json:"content"` // string or []ContentBlock
}

// ContentBlock represents rich content (text, images, tool use, tool result)
type ContentBlock struct {
	Type string `json:"type"` // "text", "image", "tool_use", "tool_result"
	Text string `json:"text,omitempty"`

	// Tool use fields
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`

	// Tool result fields
	ToolUseID string      `json:"tool_use_id,omitempty"`
	Content   interface{} `json:"content,omitempty"`
	IsError   bool        `json:"is_error,omitempty"`

	// Reasoning fields (for extended thinking)
	Reasoning string `json:"reasoning,omitempty"`
}

// Tool defines a tool that the AI can use
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// MessageRequest represents a request to create a message
type MessageRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
	System      string    `json:"system,omitempty"`
	Tools       []Tool    `json:"tools,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// MessageResponse represents a response from the AI
type MessageResponse struct {
	ID         string         `json:"id"`
	Model      string         `json:"model"`
	Role       string         `json:"role"`
	Content    []ContentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
	Usage      Usage          `json:"usage"`
}

// Usage tracks token usage
type Usage struct {
	InputTokens       int `json:"input_tokens"`
	OutputTokens      int `json:"output_tokens"`
	CacheReadTokens   int `json:"cache_read_tokens,omitempty"`
	CacheCreateTokens int `json:"cache_create_tokens,omitempty"`
}

// TotalTokens returns the total token count
func (u Usage) TotalTokens() int {
	return u.InputTokens + u.OutputTokens
}

// StreamChunk represents a streaming response chunk
type StreamChunk struct {
	Type         string           `json:"type"`
	Message      *MessageResponse `json:"message,omitempty"`
	Index        int              `json:"index,omitempty"`
	ContentBlock *ContentBlock    `json:"content_block,omitempty"`
	Delta        *Delta           `json:"delta,omitempty"`
}

// Delta represents incremental content updates
type Delta struct {
	Type        string                 `json:"type"`
	Text        string                 `json:"text,omitempty"`
	PartialJSON string                 `json:"partial_json,omitempty"`
	Input       map[string]interface{} `json:"input,omitempty"`
	Reasoning   string                 `json:"reasoning,omitempty"`
}

// Registry manages multiple providers
type Registry struct {
	providers map[string]Provider
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry
func (r *Registry) Register(provider Provider) {
	r.providers[provider.Name()] = provider
}

// Get retrieves a provider by name
func (r *Registry) Get(name string) (Provider, bool) {
	provider, ok := r.providers[name]
	return provider, ok
}

// List returns all registered provider names
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// CreateProvider creates a provider by name with the given API key
func CreateProvider(name, apiKey string) (Provider, error) {
	switch name {
	case "anthropic":
		return NewAnthropicProvider(apiKey), nil
	case "openai":
		return NewOpenAIProvider(apiKey), nil
	case "copilot":
		return NewCopilotProvider()
	case "google":
		return NewGoogleProvider(apiKey), nil
	case "groq":
		return NewGroqProvider(apiKey), nil
	case "openrouter":
		return NewOpenRouterProvider(apiKey), nil
	default:
		return NewOpenAICompatibleProvider(name, apiKey, ""), nil
	}
}

// mustMarshalJSON marshals v to JSON, panicking on error
// Used internally by providers for converting data
func mustMarshalJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// mustUnmarshalJSON unmarshals JSON data into v
// Used internally by providers for parsing data
func mustUnmarshalJSON(data string, v interface{}) {
	json.Unmarshal([]byte(data), v)
}
