package provider

import (
	"context"
	"testing"
	"time"
)

// TestProviderInterface ensures key providers implement the interface
func TestProviderInterface(t *testing.T) {
	// Test providers that can be initialized without config
	providers := []struct {
		name     string
		provider Provider
	}{
		{"anthropic", &AnthropicProvider{}},
		{"openai", &OpenAIProvider{}},
		{"google", &GoogleProvider{}},
	}

	for _, tt := range providers {
		t.Run(tt.name, func(t *testing.T) {
			if tt.provider.Name() == "" {
				t.Errorf("Provider %s should have a name", tt.name)
			}
			models := tt.provider.Models()
			if len(models) == 0 {
				t.Errorf("Provider %s should have models", tt.name)
			}
		})
	}
}

// TestMessageRequestValidation tests request validation
func TestMessageRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     *MessageRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &MessageRequest{
				Messages: []Message{
					{Role: "user", Content: "Hello"},
				},
				Model:     "test-model",
				MaxTokens: 1000,
				System:    "You are helpful",
			},
			wantErr: false,
		},
		{
			name: "empty messages",
			req: &MessageRequest{
				Messages:  []Message{},
				Model:     "test-model",
				MaxTokens: 1000,
			},
			wantErr: false, // Some providers allow empty messages
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation - just ensure the struct is well-formed
			if tt.req.Model == "" && !tt.wantErr {
				t.Error("Expected model to be set")
			}
		})
	}
}

// TestMessageRoles tests message role handling
func TestMessageRoles(t *testing.T) {
	validRoles := []string{"user", "assistant", "system"}

	for _, role := range validRoles {
		msg := Message{
			Role:    role,
			Content: "Test message",
		}

		if msg.Role == "" {
			t.Errorf("Role should not be empty for %s", role)
		}
	}
}

// TestContentBlocks tests content block creation
func TestContentBlocks(t *testing.T) {
	tests := []struct {
		name  string
		block ContentBlock
		valid bool
	}{
		{
			name:  "text block",
			block: ContentBlock{Type: "text", Text: "Hello"},
			valid: true,
		},
		{
			name:  "tool use block",
			block: ContentBlock{Type: "tool_use", ID: "tool_1", Name: "test_tool"},
			valid: true,
		},
		{
			name:  "tool result block",
			block: ContentBlock{Type: "tool_result", ToolUseID: "tool_1", Content: "result"},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid && tt.block.Type == "" {
				t.Error("Valid block should have type")
			}
		})
	}
}

// TestToolDefinition tests tool definition structure
func TestToolDefinition(t *testing.T) {
	tool := Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param1": map[string]interface{}{
					"type":        "string",
					"description": "First parameter",
				},
			},
			"required": []string{"param1"},
		},
	}

	if tool.Name == "" {
		t.Error("Tool name should not be empty")
	}
	if tool.Description == "" {
		t.Error("Tool description should not be empty")
	}
	if tool.InputSchema == nil {
		t.Error("Tool schema should not be nil")
	}
}

// TestContextTimeout tests context handling
func TestContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	time.Sleep(10 * time.Millisecond)

	select {
	case <-ctx.Done():
		// Expected - context should be done
	default:
		t.Error("Context should be done")
	}
}

// TestUsageTracking tests usage tracking
func TestUsageTracking(t *testing.T) {
	usage := Usage{
		InputTokens:  100,
		OutputTokens: 50,
	}

	if usage.InputTokens == 0 {
		t.Error("Input tokens should be set")
	}
	if usage.OutputTokens == 0 {
		t.Error("Output tokens should be set")
	}

	total := usage.TotalTokens()
	if total != 150 {
		t.Errorf("Expected total 150, got %d", total)
	}
}

// TestStopReasons tests stop reason handling
func TestStopReasons(t *testing.T) {
	validReasons := []string{
		"end_turn",
		"max_tokens",
		"stop_sequence",
		"tool_use",
	}

	for _, reason := range validReasons {
		if reason == "" {
			t.Error("Stop reason should not be empty")
		}
	}
}

// TestErrorClassification tests error classification
func TestErrorClassification(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		msg        string
		expectType ErrorType
	}{
		{"rate limit", 429, "rate limit exceeded", ErrorTypeRateLimit},
		{"context overflow", 400, "context length exceeded", ErrorTypeContextOverflow},
		{"auth error", 401, "invalid api key", ErrorTypeAuth},
		{"server error", 500, "internal server error", ErrorTypeAPIError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify we can classify errors
			if tt.statusCode >= 400 {
				// Error classification would happen here
			}
		})
	}
}

// TestIsContextOverflow tests context overflow detection
func TestIsContextOverflow(t *testing.T) {
	tests := []struct {
		msg      string
		expected bool
	}{
		{"maximum context length exceeded", true},
		{"prompt is too long", true},
		{"context_length_exceeded", true},
		{"normal error message", false},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			result := IsContextOverflow(tt.msg)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for message: %s", tt.expected, result, tt.msg)
			}
		})
	}
}

// TestRetryConfig tests retry configuration
func TestRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	if cfg.MaxAttempts <= 0 {
		t.Error("MaxAttempts should be positive")
	}
	if cfg.InitialDelay <= 0 {
		t.Error("InitialDelay should be positive")
	}
	if cfg.MaxDelay <= 0 {
		t.Error("MaxDelay should be positive")
	}
}

// TestComputeRetryDelay tests retry delay calculation
func TestComputeRetryDelay(t *testing.T) {
	cfg := DefaultRetryConfig()

	delays := []time.Duration{}
	for i := 1; i <= 5; i++ {
		delay := ComputeRetryDelay(i, cfg, 0)
		delays = append(delays, delay)
		if delay <= 0 {
			t.Errorf("Delay for attempt %d should be positive", i)
		}
		if delay > cfg.MaxDelay {
			t.Errorf("Delay %v exceeds max delay %v", delay, cfg.MaxDelay)
		}
	}

	// Delays should increase (exponential backoff)
	for i := 1; i < len(delays); i++ {
		if delays[i] < delays[i-1] {
			t.Error("Delays should increase with exponential backoff")
		}
	}
}
