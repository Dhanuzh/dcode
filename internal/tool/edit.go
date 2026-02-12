package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EditTool performs exact string replacements in files
func EditTool() *ToolDef {
	return &ToolDef{
		Name:        "edit",
		Description: "Perform an exact string replacement in a file. The oldString must match exactly (including whitespace and indentation). Always read the file first to get the exact content. Include enough context lines for a unique match.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The file path to edit",
				},
				"old_string": map[string]interface{}{
					"type":        "string",
					"description": "The exact string to find and replace. Must match precisely including whitespace.",
				},
				"new_string": map[string]interface{}{
					"type":        "string",
					"description": "The replacement string",
				},
			},
			"required": []string{"path", "old_string", "new_string"},
		},
		Execute: func(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
			path, _ := input["path"].(string)
			oldString, _ := input["old_string"].(string)
			newString, _ := input["new_string"].(string)

			if path == "" || oldString == "" {
				return &ToolResult{Output: "Error: path and old_string are required", IsError: true}, nil
			}

			if !filepath.IsAbs(path) && tc.WorkDir != "" {
				path = filepath.Join(tc.WorkDir, path)
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return &ToolResult{Output: fmt.Sprintf("Error reading file: %v", err), IsError: true}, nil
			}

			content := string(data)

			// Count occurrences
			count := strings.Count(content, oldString)
			if count == 0 {
				// Try to find similar text for helpful error
				return &ToolResult{
					Output:  fmt.Sprintf("Error: old_string not found in %s. Make sure the string matches exactly, including whitespace and indentation. Read the file first to get the exact content.", path),
					IsError: true,
				}, nil
			}
			if count > 1 {
				return &ToolResult{
					Output:  fmt.Sprintf("Error: old_string matches %d locations in %s. Include more context lines to make the match unique.", count, path),
					IsError: true,
				}, nil
			}

			newContent := strings.Replace(content, oldString, newString, 1)

			if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
				return &ToolResult{Output: fmt.Sprintf("Error writing file: %v", err), IsError: true}, nil
			}

			// Calculate diff summary
			oldLines := strings.Count(oldString, "\n") + 1
			newLines := strings.Count(newString, "\n") + 1

			return &ToolResult{Output: fmt.Sprintf("Edited %s: replaced %d lines with %d lines", path, oldLines, newLines)}, nil
		},
	}
}

// MultiEditTool performs multiple edits on a single file sequentially
func MultiEditTool() *ToolDef {
	return &ToolDef{
		Name:        "multiedit",
		Description: "Perform multiple find-and-replace edits on a single file sequentially. More efficient than multiple edit calls. Each edit must have a unique old_string match.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The file path to edit",
				},
				"edits": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"old_string": map[string]interface{}{
								"type":        "string",
								"description": "The exact string to find",
							},
							"new_string": map[string]interface{}{
								"type":        "string",
								"description": "The replacement string",
							},
						},
						"required": []string{"old_string", "new_string"},
					},
					"description": "Array of edit operations to apply sequentially",
				},
			},
			"required": []string{"path", "edits"},
		},
		Execute: func(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
			path, _ := input["path"].(string)
			if path == "" {
				return &ToolResult{Output: "Error: path is required", IsError: true}, nil
			}

			if !filepath.IsAbs(path) && tc.WorkDir != "" {
				path = filepath.Join(tc.WorkDir, path)
			}

			editsRaw, ok := input["edits"]
			if !ok {
				return &ToolResult{Output: "Error: edits array is required", IsError: true}, nil
			}

			// Parse edits
			editsJSON, _ := json.Marshal(editsRaw)
			var edits []struct {
				OldString string `json:"old_string"`
				NewString string `json:"new_string"`
			}
			if err := json.Unmarshal(editsJSON, &edits); err != nil {
				return &ToolResult{Output: fmt.Sprintf("Error parsing edits: %v", err), IsError: true}, nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return &ToolResult{Output: fmt.Sprintf("Error reading file: %v", err), IsError: true}, nil
			}

			content := string(data)
			applied := 0
			errors := []string{}

			for i, edit := range edits {
				count := strings.Count(content, edit.OldString)
				if count == 0 {
					errors = append(errors, fmt.Sprintf("Edit %d: old_string not found", i+1))
					continue
				}
				if count > 1 {
					errors = append(errors, fmt.Sprintf("Edit %d: old_string matches %d locations", i+1, count))
					continue
				}
				content = strings.Replace(content, edit.OldString, edit.NewString, 1)
				applied++
			}

			if applied > 0 {
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					return &ToolResult{Output: fmt.Sprintf("Error writing file: %v", err), IsError: true}, nil
				}
			}

			result := fmt.Sprintf("Applied %d/%d edits to %s", applied, len(edits), path)
			if len(errors) > 0 {
				result += "\nErrors:\n" + strings.Join(errors, "\n")
			}
			return &ToolResult{Output: result, IsError: len(errors) > 0 && applied == 0}, nil
		},
	}
}
