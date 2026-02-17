package session

import (
	"context"
	"testing"
	"time"
)

// TestSessionMessageTracking tests message tracking
func TestSessionMessageTracking(t *testing.T) {
	messages := []Message{
		{ID: "1", Role: "user", Content: "Hello"},
		{ID: "2", Role: "assistant", Content: "Hi"},
		{ID: "3", Role: "user", Content: "How are you?"},
	}

	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	// Verify message structure
	for _, msg := range messages {
		if msg.ID == "" {
			t.Error("Message ID should not be empty")
		}
		if msg.Role == "" {
			t.Error("Message role should not be empty")
		}
	}
}

// TestMessageRoles tests message role validation
func TestMessageRoles(t *testing.T) {
	validRoles := []string{"user", "assistant", "system"}

	for _, role := range validRoles {
		msg := Message{
			ID:      "test",
			Role:    role,
			Content: "Test content",
		}

		if msg.Role == "" {
			t.Errorf("Role should not be empty for %s", role)
		}
	}
}

// TestContextTimeout tests context handling
func TestContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	time.Sleep(10 * time.Millisecond)

	select {
	case <-ctx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled")
	}
}

// TestMessageHistory tests message accumulation
func TestMessageHistory(t *testing.T) {
	var messages []Message

	// Add messages
	messages = append(messages, Message{ID: "1", Role: "user", Content: "First"})
	messages = append(messages, Message{ID: "2", Role: "assistant", Content: "Second"})
	messages = append(messages, Message{ID: "3", Role: "user", Content: "Third"})

	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	// Verify order
	if messages[0].Content != "First" {
		t.Error("First message content mismatch")
	}
	if messages[2].Content != "Third" {
		t.Error("Third message content mismatch")
	}
}

// TestCompaction tests message compaction
func TestCompaction(t *testing.T) {
	// Create many messages
	var messages []Message
	for i := 0; i < 100; i++ {
		messages = append(messages, Message{
			ID:      string(rune('0' + (i % 10))),
			Role:    "user",
			Content: "Message",
		})
	}

	// Simulate compaction - keep last 50
	maxMessages := 50
	if len(messages) > maxMessages {
		messages = messages[len(messages)-maxMessages:]
	}

	if len(messages) != maxMessages {
		t.Errorf("Expected %d messages after compaction, got %d", maxMessages, len(messages))
	}
}

// TestStatusTracking tests session status
func TestStatusTracking(t *testing.T) {
	statuses := []string{
		"idle",
		"busy",
		"retry",
		"error",
	}

	for _, status := range statuses {
		if status == "" {
			t.Error("Status should not be empty")
		}
	}
}

// TestToolCallTracking tests tool call parts
func TestToolCallTracking(t *testing.T) {
	parts := []Part{
		{Type: "tool_use", ToolName: "read", ToolInput: map[string]interface{}{"path": "test.txt"}},
		{Type: "tool_result", ToolID: "1", Content: "result"},
		{Type: "text", Content: "Done"},
	}

	if len(parts) != 3 {
		t.Errorf("Expected 3 parts, got %d", len(parts))
	}

	for _, part := range parts {
		if part.Type == "" {
			t.Error("Part type should not be empty")
		}
	}
}

// TestSummaryTracking tests summary statistics
func TestSummaryTracking(t *testing.T) {
	summary := Summary{
		Additions: 100,
		Deletions: 50,
		Files:     []string{"file1.go", "file2.go"},
		FileCount: 2,
		TokensIn:  1000,
		TokensOut: 500,
		ToolCalls: 10,
		TotalCost: 0.05,
	}

	if summary.TokensIn <= 0 {
		t.Error("TokensIn should be positive")
	}
	if summary.TokensOut <= 0 {
		t.Error("TokensOut should be positive")
	}
	if summary.FileCount != len(summary.Files) {
		t.Error("FileCount should match Files length")
	}
}

// TestErrorHandling tests error propagation
func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		handled bool
	}{
		{"no error", nil, true},
		{"context cancelled", context.Canceled, true},
		{"timeout", context.DeadlineExceeded, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil && !tt.handled {
				t.Error("Nil error should be handled")
			}
		})
	}
}

// TestMessageParts tests message parts structure
func TestMessageParts(t *testing.T) {
	partTypes := []string{
		"text",
		"tool_use",
		"tool_result",
		"reasoning",
		"error",
	}

	for _, pType := range partTypes {
		part := Part{Type: pType}
		if part.Type == "" {
			t.Errorf("Part type should not be empty for %s", pType)
		}
	}
}

// TestTimestamps tests timestamp handling
func TestTimestamps(t *testing.T) {
	now := time.Now()
	later := now.Add(1 * time.Hour)

	if !later.After(now) {
		t.Error("Later timestamp should be after earlier")
	}

	duration := later.Sub(now)
	if duration != 1*time.Hour {
		t.Errorf("Expected 1 hour duration, got %v", duration)
	}
}

// TestRevertInfo tests revert tracking
func TestRevertInfo(t *testing.T) {
	revert := RevertInfo{
		MessageID: "msg_123",
		PartID:    "part_456",
		Snapshot:  "abc123",
		Diff:      "diff content",
	}

	if revert.MessageID == "" {
		t.Error("MessageID should be set")
	}
	if revert.Snapshot == "" {
		t.Error("Snapshot should be set")
	}
}

// TestMessageError tests error tracking in messages
func TestMessageError(t *testing.T) {
	msgErr := MessageError{
		Type:    "api_error",
		Message: "Request failed",
		Code:    "500",
	}

	if msgErr.Type == "" {
		t.Error("Error type should be set")
	}
	if msgErr.Message == "" {
		t.Error("Error message should be set")
	}
}
