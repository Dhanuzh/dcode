package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
)

// Provider defines the interface for AI providers
type Provider interface {
	Name() string
	CreateMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error)
	StreamMessage(ctx context.Context, req *MessageRequest, callback func(*StreamChunk) error) error
	Models() []string
}

// ProviderError types for error classification
type ErrorType string

const (
	ErrorTypeContextOverflow ErrorType = "context_overflow"
	ErrorTypeAPIError        ErrorType = "api_error"
	ErrorTypeRateLimit       ErrorType = "rate_limit"
	ErrorTypeAuth            ErrorType = "auth_error"
	ErrorTypeNotFound        ErrorType = "not_found"
	ErrorTypeTimeout         ErrorType = "timeout"
)

// ClassifiedError wraps a provider error with classification
type ClassifiedError struct {
	Type        ErrorType
	Message     string
	StatusCode  int
	IsRetryable bool
	RetryAfter  time.Duration
	Original    error
}

func (e *ClassifiedError) Error() string {
	return e.Message
}

func (e *ClassifiedError) Unwrap() error {
	return e.Original
}

// Context overflow detection patterns from various providers
var overflowPatterns = []*regexp.Regexp{
	// Anthropic
	regexp.MustCompile(`prompt is too long`),
	regexp.MustCompile(`exceeds the model'?s maximum context`),
	regexp.MustCompile(`content exceeds model token limit`),
	// OpenAI
	regexp.MustCompile(`maximum context length`),
	regexp.MustCompile(`context_length_exceeded`),
	regexp.MustCompile(`max_tokens.*exceeds.*limit`),
	// Google
	regexp.MustCompile(`exceeds the maximum number of tokens`),
	regexp.MustCompile(`RESOURCE_EXHAUSTED.*token`),
	regexp.MustCompile(`GenerateContentRequest.*too large`),
	// Bedrock
	regexp.MustCompile(`Input is too long`),
	regexp.MustCompile(`Too many input tokens`),
	// xAI / Groq
	regexp.MustCompile(`Request too large`),
	regexp.MustCompile(`Please reduce the length`),
	// OpenRouter
	regexp.MustCompile(`context_length_exceeded`),
	// Generic
	regexp.MustCompile(`(?i)context.*(?:too long|overflow|exceeded|limit)`),
	regexp.MustCompile(`(?i)token.*(?:limit|exceeded|maximum)`),
	// Cerebras / Mistral empty body
	regexp.MustCompile(`(?:400|413)\s*\(no body\)`),
	// llama.cpp / LM Studio
	regexp.MustCompile(`context size exceeded`),
	// MiniMax / Kimi
	regexp.MustCompile(`(?i)token count.*exceeds`),
}

// IsContextOverflow checks if an error message indicates context overflow
func IsContextOverflow(msg string) bool {
	for _, pat := range overflowPatterns {
		if pat.MatchString(msg) {
			return true
		}
	}
	return false
}

// ClassifyError classifies an error from a provider
func ClassifyError(err error, statusCode int, responseBody string) *ClassifiedError {
	if err == nil {
		return nil
	}

	// If already classified, return as-is
	if ce, ok := err.(*ClassifiedError); ok {
		return ce
	}

	msg := err.Error()
	if responseBody != "" {
		msg = msg + " " + responseBody
	}

	// Check for context overflow
	if IsContextOverflow(msg) {
		return &ClassifiedError{
			Type:        ErrorTypeContextOverflow,
			Message:     "Context window exceeded. Consider compacting the conversation.",
			StatusCode:  statusCode,
			IsRetryable: false,
			Original:    err,
		}
	}

	// Rate limiting
	if statusCode == 429 || strings.Contains(strings.ToLower(msg), "rate_limit") ||
		strings.Contains(strings.ToLower(msg), "too_many_requests") ||
		strings.Contains(strings.ToLower(msg), "quota") {
		return &ClassifiedError{
			Type:        ErrorTypeRateLimit,
			Message:     "Rate limited by provider. Retrying...",
			StatusCode:  statusCode,
			IsRetryable: true,
			Original:    err,
		}
	}

	// Auth errors
	if statusCode == 401 || statusCode == 403 {
		return &ClassifiedError{
			Type:        ErrorTypeAuth,
			Message:     fmt.Sprintf("Authentication error (%d): %s", statusCode, err.Error()),
			StatusCode:  statusCode,
			IsRetryable: false,
			Original:    err,
		}
	}

	// Not found (sometimes retryable for OpenAI)
	if statusCode == 404 {
		return &ClassifiedError{
			Type:        ErrorTypeNotFound,
			Message:     fmt.Sprintf("Model or endpoint not found: %s", err.Error()),
			StatusCode:  statusCode,
			IsRetryable: true, // OpenAI sometimes returns 404 for valid models
			Original:    err,
		}
	}

	// Server errors are retryable
	if statusCode >= 500 {
		return &ClassifiedError{
			Type:        ErrorTypeAPIError,
			Message:     fmt.Sprintf("Provider server error (%d): %s", statusCode, err.Error()),
			StatusCode:  statusCode,
			IsRetryable: true,
			Original:    err,
		}
	}

	// Check for overloaded / exhausted
	lowerMsg := strings.ToLower(msg)
	if strings.Contains(lowerMsg, "overloaded") || strings.Contains(lowerMsg, "exhausted") ||
		strings.Contains(lowerMsg, "unavailable") {
		return &ClassifiedError{
			Type:        ErrorTypeAPIError,
			Message:     "Provider is overloaded. Retrying...",
			StatusCode:  statusCode,
			IsRetryable: true,
			Original:    err,
		}
	}

	// Default: non-retryable API error
	return &ClassifiedError{
		Type:        ErrorTypeAPIError,
		Message:     err.Error(),
		StatusCode:  statusCode,
		IsRetryable: false,
		Original:    err,
	}
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
	MaxAttempts   int
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		InitialDelay:  2 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		MaxAttempts:   5,
	}
}

// ComputeRetryDelay computes the retry delay for a given attempt
func ComputeRetryDelay(attempt int, cfg RetryConfig, retryAfter time.Duration) time.Duration {
	// If server provided retry-after, use it
	if retryAfter > 0 {
		return retryAfter
	}
	// Exponential backoff
	delay := time.Duration(float64(cfg.InitialDelay) * math.Pow(cfg.BackoffFactor, float64(attempt-1)))
	if delay > cfg.MaxDelay {
		delay = cfg.MaxDelay
	}
	return delay
}

// Message represents a conversation message
type Message struct {
	Role    string      `json:"role"`    // "user", "assistant", "system"
	Content interface{} `json:"content"` // string or []ContentBlock
}

// ImageSource describes how to supply an image to the model.
type ImageSource struct {
	Type      string `json:"type"`                 // "base64" or "url"
	MediaType string `json:"media_type,omitempty"` // e.g. "image/png"
	Data      string `json:"data,omitempty"`       // base64-encoded bytes (for type=base64)
	URL       string `json:"url,omitempty"`        // remote URL (for type=url)
}

// ContentBlock represents rich content (text, images, tool use, tool result)
type ContentBlock struct {
	Type string `json:"type"` // "text", "image", "tool_use", "tool_result"
	Text string `json:"text,omitempty"`

	// Image fields (Type == "image")
	Source *ImageSource `json:"source,omitempty"`

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
		// Anthropic uses OAuth only — load the stored token regardless of apiKey value.
		token, err := GetAnthropicOAuthToken()
		if err != nil {
			return nil, fmt.Errorf("failed to load Anthropic OAuth token: %w", err)
		}
		if token == nil {
			return nil, fmt.Errorf("no Anthropic OAuth token found — run 'dcode auth anthropic' to authenticate")
		}
		return NewAnthropicProviderOAuth(token.AccessToken), nil
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
	case "mistral":
		return NewMistralProvider(apiKey), nil
	case "cohere":
		return NewCohereProvider(apiKey), nil
	case "together":
		return NewTogetherProvider(apiKey), nil
	case "replicate":
		return NewReplicateProvider(apiKey), nil
	case "perplexity":
		return NewPerplexityProvider(apiKey), nil
	case "deepseek":
		return NewDeepSeekProvider(apiKey), nil
	case "azure":
		return NewAzureOpenAIProvider(apiKey, ""), nil
	case "bedrock":
		return NewBedrockProvider("us-east-1"), nil
	case "xai":
		return NewXAIProvider(apiKey), nil
	case "deepinfra":
		return NewDeepInfraProvider(apiKey), nil
	case "cerebras":
		return NewCerebrasProvider(apiKey), nil
	case "google-vertex":
		return NewGoogleVertexProvider(apiKey), nil
	case "gitlab":
		return NewGitLabProvider(apiKey), nil
	case "cloudflare-workers-ai":
		return NewCloudflareWorkersAIProvider(apiKey), nil
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
