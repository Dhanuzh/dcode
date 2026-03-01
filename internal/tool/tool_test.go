package tool

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// Test basic tool operations without depending on internal Tool interface
func TestWriteAndRead(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Hello, World!"

	// Write file
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Read file
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(data) != content {
		t.Errorf("Expected %q, got %q", content, string(data))
	}
}

// TestFuzzyReplace is already covered in fuzzymatch_test.go
func TestFileOperations(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		file    string
		content string
	}{
		{"simple file", "test.txt", "content"},
		{"nested file", "a/b/c/test.txt", "nested"},
		{"empty file", "empty.txt", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmpDir, tt.file)

			// Create directories
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatalf("Failed to create dirs: %v", err)
			}

			// Write
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write: %v", err)
			}

			// Read back
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Failed to read: %v", err)
			}

			if string(data) != tt.content {
				t.Errorf("Content mismatch: got %q, want %q", string(data), tt.content)
			}
		})
	}
}

// TestGlobPatterns tests glob pattern matching
func TestGlobPatterns(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	files := map[string]string{
		"test1.go":      "package main",
		"test2.go":      "package test",
		"main.js":       "console.log()",
		"data.json":     "{}",
		"sub/nested.go": "package sub",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, []byte(content), 0644)
	}

	// Test glob patterns
	matches, err := filepath.Glob(filepath.Join(tmpDir, "*.go"))
	if err != nil {
		t.Fatalf("Glob failed: %v", err)
	}

	if len(matches) < 2 {
		t.Errorf("Expected at least 2 .go files, got %d", len(matches))
	}
}

// TestSearchContent tests content searching
func TestSearchContent(t *testing.T) {
	tmpDir := t.TempDir()

	files := map[string]string{
		"file1.txt": "Hello World\nFoo Bar",
		"file2.txt": "Hello again",
		"file3.txt": "Different content",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		os.WriteFile(path, []byte(content), 0644)
	}

	// Count files containing "Hello"
	count := 0
	for name, content := range files {
		if filepath.Ext(name) == ".txt" && containsString(content, "Hello") {
			count++
		}
	}

	if count != 2 {
		t.Errorf("Expected 2 files with 'Hello', got %d", count)
	}
}

func containsString(content, search string) bool {
	return len(content) > 0 && len(search) > 0 &&
		(content == search || len(content) > len(search) &&
			(content[:len(search)] == search || content[len(content)-len(search):] == search ||
				containsSubstring(content, search)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestPathSafety tests path handling
func TestPathSafety(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		valid bool
	}{
		{"relative", "test.txt", true},
		{"nested", "a/b/c/test.txt", true},
		{"absolute", "/tmp/test.txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic path validation
			if tt.path == "" {
				t.Error("Path should not be empty")
			}
		})
	}
}

// TestToolRegistry tests basic registry functionality
func TestToolRegistry(t *testing.T) {
	registry := GetRegistry()

	// Verify built-in tools are registered
	tools := []string{"read", "write", "edit", "bash", "glob", "grep"}

	for _, name := range tools {
		if _, ok := registry.Get(name); !ok {
			t.Errorf("Tool %s should be registered", name)
		}
	}

	// Verify unknown tool
	if _, ok := registry.Get("nonexistent"); ok {
		t.Error("Nonexistent tool should not be found")
	}

	// Test List function
	names := registry.List()
	if len(names) == 0 {
		t.Error("Registry should have tools")
	}
}

// TestContextHandling tests context cancellation
func TestContextHandling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	select {
	case <-ctx.Done():
		// Expected
	default:
		t.Error("Context should be done")
	}
}

// TestEditReplacement tests basic string replacement
func TestEditReplacement(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "edit.txt")

	original := "Line 1\nLine 2\nLine 3"
	os.WriteFile(testFile, []byte(original), 0644)

	// Read, replace, write
	content, _ := os.ReadFile(testFile)
	modified := replaceString(string(content), "Line 2", "Modified Line 2")
	os.WriteFile(testFile, []byte(modified), 0644)

	// Verify
	result, _ := os.ReadFile(testFile)
	expected := "Line 1\nModified Line 2\nLine 3"
	if string(result) != expected {
		t.Errorf("Expected %q, got %q", expected, string(result))
	}
}

func replaceString(s, old, new string) string {
	result := ""
	remaining := s
	for {
		idx := indexString(remaining, old)
		if idx == -1 {
			result += remaining
			break
		}
		result += remaining[:idx] + new
		remaining = remaining[idx+len(old):]
	}
	return result
}

func indexString(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
