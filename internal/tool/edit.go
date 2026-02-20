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
		Description: "Find-and-replace in a file with fuzzy matching. Read the file first. Include enough context for a unique match.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The file path to edit",
				},
				"old_string": map[string]interface{}{
					"type":        "string",
					"description": "The string to find and replace. Fuzzy matching will handle minor whitespace and indentation differences.",
				},
				"new_string": map[string]interface{}{
					"type":        "string",
					"description": "The replacement string",
				},
				"replace_all": map[string]interface{}{
					"type":        "boolean",
					"description": "Replace all occurrences instead of just the first unique match. Default: false",
				},
			},
			"required": []string{"path", "old_string", "new_string"},
		},
		Execute: func(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
			path, _ := input["path"].(string)
			oldString, _ := input["old_string"].(string)
			newString, _ := input["new_string"].(string)
			replaceAll, _ := input["replace_all"].(bool)

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

			// Use fuzzy replacement with 9 fallback strategies
			newContent, err := FuzzyReplace(content, oldString, newString, replaceAll)
			if err != nil {
				return &ToolResult{
					Output:  fmt.Sprintf("Error in %s: %v", path, err),
					IsError: true,
				}, nil
			}

			if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
				return &ToolResult{Output: fmt.Sprintf("Error writing file: %v", err), IsError: true}, nil
			}

			// Calculate diff summary
			oldLines := strings.Count(oldString, "\n") + 1
			newLines := strings.Count(newString, "\n") + 1

			return &ToolResult{
				Output: fmt.Sprintf("Edited %s: replaced %d lines with %d lines", path, oldLines, newLines),
				DiffData: &DiffData{
					OldContent: oldString,
					NewContent: newString,
					FilePath:   path,
					Language:   inferLanguage(path),
					IsFragment: true,
				},
			}, nil
		},
	}
}

// MultiEditTool performs multiple edits on a single file sequentially
func MultiEditTool() *ToolDef {
	return &ToolDef{
		Name:        "multiedit",
		Description: "Multiple find-and-replace edits on one file. More efficient than multiple edit calls.",
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
			var diffList []*DiffData

			for i, edit := range edits {
				newContent, err := FuzzyReplace(content, edit.OldString, edit.NewString, false)
				if err != nil {
					errors = append(errors, fmt.Sprintf("Edit %d: %v", i+1, err))
					continue
				}
				diffList = append(diffList, &DiffData{
					OldContent: edit.OldString,
					NewContent: edit.NewString,
					FilePath:   path,
					Language:   inferLanguage(path),
					IsFragment: true,
				})
				content = newContent
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
			return &ToolResult{
				Output:       result,
				IsError:      len(errors) > 0 && applied == 0,
				DiffDataList: diffList,
			}, nil
		},
	}
}
