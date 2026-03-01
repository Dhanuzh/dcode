package tool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteTool writes content to a file, creating parent directories as needed
func WriteTool() *ToolDef {
	return &ToolDef{
		Name:        "write",
		Description: "Write content to a file. Creates parent directories if needed.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The file path to write to",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "The full content to write to the file",
				},
			},
			"required": []string{"path", "content"},
		},
		Execute: func(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
			path, _ := input["path"].(string)
			content, _ := input["content"].(string)

			if path == "" {
				return &ToolResult{Output: "Error: path is required", IsError: true}, nil
			}

			// Resolve relative paths
			if !filepath.IsAbs(path) && tc.WorkDir != "" {
				path = filepath.Join(tc.WorkDir, path)
			}

			// Create parent directories
			dir := filepath.Dir(path)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return &ToolResult{Output: fmt.Sprintf("Error creating directories: %v", err), IsError: true}, nil
			}

			// Check if file exists and read old content for diff
			existed := false
			var oldContent string
			if data, err := os.ReadFile(path); err == nil {
				existed = true
				oldContent = string(data)
			}

			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return &ToolResult{Output: fmt.Sprintf("Error writing file: %v", err), IsError: true}, nil
			}

			lines := strings.Count(content, "\n") + 1
			action := "Created"
			if existed {
				action = "Updated"
			}

			result := &ToolResult{Output: fmt.Sprintf("%s %s (%d lines, %d bytes)", action, path, lines, len(content))}

			// Attach diff data for files that were updated (not newly created)
			if existed {
				result.DiffData = &DiffData{
					OldContent: oldContent,
					NewContent: content,
					FilePath:   path,
					Language:   inferLanguage(path),
					IsFragment: false,
				}
			}

			return result, nil
		},
	}
}
