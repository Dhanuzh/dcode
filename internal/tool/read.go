package tool

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// ReadTool reads file contents with optional offset and limit
func ReadTool() *ToolDef {
	return &ToolDef{
		Name:        "read",
		Description: "Read the contents of a file. Supports reading specific line ranges with offset and limit parameters. Returns file content as text. Use this before editing files to understand their current state.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The absolute or relative file path to read",
				},
				"offset": map[string]interface{}{
					"type":        "integer",
					"description": "Line number to start reading from (1-based). Default: 1",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of lines to read. Default: 2000",
				},
			},
			"required": []string{"path"},
		},
		Execute: func(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
			path, _ := input["path"].(string)
			if path == "" {
				return &ToolResult{Output: "Error: path is required", IsError: true}, nil
			}

			// Resolve relative paths
			if !strings.HasPrefix(path, "/") && tc.WorkDir != "" {
				path = tc.WorkDir + "/" + path
			}

			data, err := os.ReadFile(path)
			if err != nil {
				if os.IsNotExist(err) {
					// Try to suggest similar files
					dir := path[:strings.LastIndex(path, "/")+1]
					if dir == "" {
						dir = "."
					}
					entries, _ := os.ReadDir(dir)
					suggestions := []string{}
					base := path[strings.LastIndex(path, "/")+1:]
					for _, e := range entries {
						if strings.Contains(strings.ToLower(e.Name()), strings.ToLower(base[:min(3, len(base))])) {
							suggestions = append(suggestions, e.Name())
						}
					}
					msg := fmt.Sprintf("File not found: %s", path)
					if len(suggestions) > 0 {
						msg += fmt.Sprintf("\nDid you mean: %s", strings.Join(suggestions, ", "))
					}
					return &ToolResult{Output: msg, IsError: true}, nil
				}
				return &ToolResult{Output: fmt.Sprintf("Error reading file: %v", err), IsError: true}, nil
			}

			content := string(data)
			lines := strings.Split(content, "\n")

			offset := 1
			if v, ok := input["offset"].(float64); ok && v > 0 {
				offset = int(v)
			}
			limit := 2000
			if v, ok := input["limit"].(float64); ok && v > 0 {
				limit = int(v)
			}

			// Apply offset and limit
			startIdx := offset - 1
			if startIdx < 0 {
				startIdx = 0
			}
			if startIdx >= len(lines) {
				return &ToolResult{Output: fmt.Sprintf("Offset %d exceeds file length (%d lines)", offset, len(lines))}, nil
			}
			endIdx := startIdx + limit
			if endIdx > len(lines) {
				endIdx = len(lines)
			}

			selectedLines := lines[startIdx:endIdx]
			// Add line numbers
			numbered := make([]string, len(selectedLines))
			for i, line := range selectedLines {
				numbered[i] = fmt.Sprintf("%4d | %s", startIdx+i+1, line)
			}

			result := strings.Join(numbered, "\n")

			// Truncate if too large (50KB limit)
			if len(result) > 50*1024 {
				result = result[:50*1024] + "\n... (truncated, file too large)"
			}

			header := fmt.Sprintf("File: %s (%d lines total, showing lines %d-%d)\n\n", path, len(lines), startIdx+1, endIdx)
			return &ToolResult{Output: header + result}, nil
		},
	}
}
