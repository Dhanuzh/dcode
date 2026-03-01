//go:build integration

package provider

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// Integration tests require API keys to be set
// Run with: go test -tags=integration ./internal/provider/...

// TestAnthropicIntegration tests real Anthropic API calls
func TestAnthropicIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set, skipping integration test")
	}

	provider := NewAnthropicProvider(apiKey)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := &MessageRequest{
		Model:     "claude-3-haiku-20240307",
		MaxTokens: 100,
		Messages: []Message{
			{Role: "user", Content: "Say 'Hello, World!' and nothing else."},
		},
	}

	resp, err := provider.CreateMessage(ctx, req)
	if err != nil {
		t.Fatalf("CreateMessage failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}

	if len(resp.Content) == 0 {
		t.Fatal("Response has no content")
	}

	// Check that response contains expected text
	hasHello := false
	for _, block := range resp.Content {
		if block.Type == "text" && strings.Contains(strings.ToLower(block.Text), "hello") {
			hasHello = true
			break
		}
	}

	if !hasHello {
		t.Errorf("Response doesn't contain 'hello': %+v", resp.Content)
	}
}

// TestOpenAIIntegration tests real OpenAI API calls
func TestOpenAIIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	provider := NewOpenAIProvider(apiKey)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := &MessageRequest{
		Model:     "gpt-3.5-turbo",
		MaxTokens: 100,
		Messages: []Message{
			{Role: "user", Content: "Say 'Hello, World!' and nothing else."},
		},
	}

	resp, err := provider.CreateMessage(ctx, req)
	if err != nil {
		t.Fatalf("CreateMessage failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}

	if len(resp.Content) == 0 {
		t.Fatal("Response has no content")
	}

	t.Logf("Response: %+v", resp.Content)
}

// TestGoogleIntegration tests real Google API calls
func TestGoogleIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		t.Skip("GOOGLE_API_KEY not set, skipping integration test")
	}

	provider := NewGoogleProvider(apiKey)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := &MessageRequest{
		Model:     "gemini-2.0-flash-exp",
		MaxTokens: 100,
		Messages: []Message{
			{Role: "user", Content: "Say 'Hello, World!' and nothing else."},
		},
	}

	resp, err := provider.CreateMessage(ctx, req)
	if err != nil {
		t.Fatalf("CreateMessage failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}

	t.Logf("Response: %+v", resp.Content)
}

// TestContextOverflowDetection tests context overflow error detection
func TestContextOverflowDetection(t *testing.T) {
	testCases := []struct {
		name     string
		message  string
		expected bool
	}{
		{
			name:     "anthropic overflow",
			message:  "prompt is too long: 150000 tokens",
			expected: true,
		},
		{
			name:     "openai overflow",
			message:  "maximum context length exceeded",
			expected: true,
		},
		{
			name:     "google overflow",
			message:  "RESOURCE_EXHAUSTED: token limit exceeded",
			expected: true,
		},
		{
			name:     "normal error",
			message:  "invalid api key",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsContextOverflow(tc.message)
			if result != tc.expected {
				t.Errorf("IsContextOverflow(%q) = %v, want %v", tc.message, result, tc.expected)
			}
		})
	}
}

// TestRetryLogic tests the retry mechanism
func TestRetryLogic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping retry test in short mode")
	}

	attempts := 0
	maxAttempts := 3

	base := NewBaseHTTPProvider("fake-key", "https://fake.api")
	ctx := context.Background()

	err := base.WithRetry(ctx, func() error {
		attempts++
		if attempts < maxAttempts {
			// Return retryable error
			return &ClassifiedError{
				Type:        ErrorTypeRateLimit,
				Message:     "rate limited",
				IsRetryable: true,
			}
		}
		return nil // Success on 3rd attempt
	})

	if err != nil {
		t.Errorf("Expected success after retries, got: %v", err)
	}

	if attempts != maxAttempts {
		t.Errorf("Expected %d attempts, got %d", maxAttempts, attempts)
	}
}

// TestUserFriendlyErrors tests error message formatting
func TestUserFriendlyErrors(t *testing.T) {
	testCases := []struct {
		name     string
		input    error
		provider string
	}{
		{
			name: "context overflow",
			input: &ClassifiedError{
				Type:    ErrorTypeContextOverflow,
				Message: "context too long",
			},
			provider: "anthropic",
		},
		{
			name: "auth error",
			input: &ClassifiedError{
				Type:    ErrorTypeAuth,
				Message: "invalid api key",
			},
			provider: "openai",
		},
		{
			name: "rate limit",
			input: &ClassifiedError{
				Type:    ErrorTypeRateLimit,
				Message: "rate limit exceeded",
			},
			provider: "google",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			friendly := MakeUserFriendly(tc.input, tc.provider)
			if friendly == nil {
				t.Fatal("Expected non-nil error")
			}

			uf, ok := friendly.(*UserFriendlyError)
			if !ok {
				t.Fatalf("Expected UserFriendlyError, got %T", friendly)
			}

			if uf.Title == "" {
				t.Error("Expected non-empty title")
			}
			if uf.Message == "" {
				t.Error("Expected non-empty message")
			}

			t.Logf("Error message:\n%s", uf.Error())
		})
	}
}

// BenchmarkProviderCreation benchmarks provider initialization
func BenchmarkProviderCreation(b *testing.B) {
	b.Run("Anthropic", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewAnthropicProvider("fake-key")
		}
	})

	b.Run("OpenAI", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewOpenAIProvider("fake-key")
		}
	})

	b.Run("Google", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewGoogleProvider("fake-key")
		}
	})
}

// TestProviderSwitching tests switching between providers
func TestProviderSwitching(t *testing.T) {
	providers := map[string]Provider{
		"anthropic": NewAnthropicProvider("fake-key"),
		"openai":    NewOpenAIProvider("fake-key"),
		"google":    NewGoogleProvider("fake-key"),
	}

	for name, p := range providers {
		if p.Name() != name {
			t.Errorf("Provider %s returned wrong name: %s", name, p.Name())
		}

		models := p.Models()
		if len(models) == 0 {
			t.Errorf("Provider %s has no models", name)
		}
	}
}
