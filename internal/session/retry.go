package session

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Retry constants matching opencode's retry.ts
const (
	RetryInitialDelay     = 2 * time.Second
	RetryBackoffFactor    = 2
	RetryMaxDelayNoHeader = 30 * time.Second
	RetryMaxDelay         = time.Duration(math.MaxInt32) * time.Millisecond
)

// RetryInfo tracks retry state for a session
type RetryInfo struct {
	Attempt int       `json:"attempt"`
	Message string    `json:"message"`
	NextAt  time.Time `json:"next_at"`
}

// ComputeRetryDelay calculates the delay before the next retry attempt
// It checks for retry-after-ms and retry-after headers first, then falls back to exponential backoff
func ComputeRetryDelay(attempt int, headers http.Header) time.Duration {
	if headers != nil {
		// Check retry-after-ms header first
		if retryAfterMs := headers.Get("Retry-After-Ms"); retryAfterMs != "" {
			if ms, err := strconv.ParseFloat(retryAfterMs, 64); err == nil {
				return time.Duration(ms) * time.Millisecond
			}
		}

		// Check retry-after header (seconds or HTTP date)
		if retryAfter := headers.Get("Retry-After"); retryAfter != "" {
			// Try parsing as seconds
			if seconds, err := strconv.ParseFloat(retryAfter, 64); err == nil {
				return time.Duration(seconds*1000) * time.Millisecond
			}
			// Try parsing as HTTP date
			if t, err := http.ParseTime(retryAfter); err == nil {
				delay := time.Until(t)
				if delay > 0 {
					return delay
				}
			}
		}

		// Headers exist but no retry-after found - use backoff without max cap
		return RetryInitialDelay * time.Duration(math.Pow(float64(RetryBackoffFactor), float64(attempt-1)))
	}

	// No headers - use backoff with cap
	delay := RetryInitialDelay * time.Duration(math.Pow(float64(RetryBackoffFactor), float64(attempt-1)))
	if delay > RetryMaxDelayNoHeader {
		delay = RetryMaxDelayNoHeader
	}
	return delay
}

// IsRetryableError determines if an error should be retried
// Returns a user-friendly message if retryable, empty string if not
func IsRetryableError(err error) string {
	if err == nil {
		return ""
	}

	msg := err.Error()

	// Context overflow errors should NOT be retried
	if isContextOverflow(msg) {
		return ""
	}

	// Check for retryable patterns
	lowerMsg := strings.ToLower(msg)

	// Rate limiting
	if strings.Contains(lowerMsg, "too_many_requests") ||
		strings.Contains(lowerMsg, "rate_limit") ||
		strings.Contains(lowerMsg, "rate limit") {
		return "Rate Limited"
	}

	// Overloaded
	if strings.Contains(msg, "Overloaded") ||
		strings.Contains(lowerMsg, "overloaded") {
		return "Provider is overloaded"
	}

	// Resource exhaustion
	if strings.Contains(lowerMsg, "exhausted") ||
		strings.Contains(lowerMsg, "unavailable") {
		return "Provider is overloaded"
	}

	// HTTP status-based retries
	if strings.Contains(msg, "429") ||
		strings.Contains(msg, "502") ||
		strings.Contains(msg, "503") ||
		strings.Contains(msg, "529") {
		return "Server error - retrying"
	}

	// Connection errors
	if strings.Contains(lowerMsg, "connection refused") ||
		strings.Contains(lowerMsg, "connection reset") ||
		strings.Contains(lowerMsg, "timeout") {
		return "Connection error"
	}

	return ""
}

// isContextOverflow checks if the error is a context overflow (not retryable)
func isContextOverflow(msg string) bool {
	overflowPatterns := []string{
		"context_length_exceeded",
		"max_tokens",
		"token limit",
		"context window",
		"too many tokens",
		"maximum context length",
		"content_too_large",
		"prompt is too long",
		"request too large",
		"input is too long",
		"exceeds the model's maximum",
	}

	lower := strings.ToLower(msg)
	for _, pattern := range overflowPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// SleepWithAbort sleeps for the given duration, but returns early if the context is cancelled
func SleepWithAbort(d time.Duration, abort <-chan struct{}) error {
	if d > RetryMaxDelay {
		d = RetryMaxDelay
	}

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-abort:
		return fmt.Errorf("aborted")
	}
}
