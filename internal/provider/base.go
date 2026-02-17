package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// BaseHTTPProvider provides common HTTP functionality for providers
type BaseHTTPProvider struct {
	client  *http.Client
	apiKey  string
	baseURL string
	headers map[string]string
}

// NewBaseHTTPProvider creates a new base HTTP provider
func NewBaseHTTPProvider(apiKey, baseURL string) *BaseHTTPProvider {
	return &BaseHTTPProvider{
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
		apiKey:  apiKey,
		baseURL: baseURL,
		headers: make(map[string]string),
	}
}

// SetHeader sets a custom header for all requests
func (b *BaseHTTPProvider) SetHeader(key, value string) {
	b.headers[key] = value
}

// DoRequest performs an HTTP request with standardized error handling
func (b *BaseHTTPProvider) DoRequest(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	url := b.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set common headers
	req.Header.Set("Content-Type", "application/json")
	if b.apiKey != "" {
		// Default authorization header (can be overridden)
		if _, ok := b.headers["Authorization"]; !ok {
			req.Header.Set("Authorization", "Bearer "+b.apiKey)
		}
	}

	// Set custom headers
	for key, value := range b.headers {
		req.Header.Set(key, value)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		return respBody, resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, resp.StatusCode, nil
}

// HandleError wraps error handling with classification
func (b *BaseHTTPProvider) HandleError(err error, statusCode int, responseBody []byte) error {
	if err == nil {
		return nil
	}

	// Classify the error
	classified := ClassifyError(err, statusCode, string(responseBody))
	return classified
}

// WithRetry wraps a function with retry logic
func (b *BaseHTTPProvider) WithRetry(ctx context.Context, fn func() error) error {
	cfg := DefaultRetryConfig()
	var lastErr error

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if classified, ok := err.(*ClassifiedError); ok {
			if !classified.IsRetryable {
				return err
			}
		} else {
			// Unknown error, don't retry
			return err
		}

		// Last attempt, return error
		if attempt == cfg.MaxAttempts {
			break
		}

		// Calculate delay
		delay := ComputeRetryDelay(attempt, cfg, 0)

		// Wait with context
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return lastErr
}

// ValidateRequest performs basic validation on a MessageRequest
func ValidateRequest(req *MessageRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}
	if req.Model == "" {
		return fmt.Errorf("model must be specified")
	}
	if req.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be positive")
	}
	if req.Temperature < 0 || req.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	return nil
}

// LogRequest logs request details (for debugging)
func LogRequest(provider, model string, tokenCount int) {
	// Basic logging - can be extended
	_ = provider
	_ = model
	_ = tokenCount
	// Future: Add structured logging here
}

// LogResponse logs response details (for debugging)
func LogResponse(provider, model string, usage Usage, stopReason string) {
	// Basic logging - can be extended
	_ = provider
	_ = model
	_ = usage
	_ = stopReason
	// Future: Add structured logging here
}
