package tool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ApplyPatchTool applies patches in opencode's custom "*** Begin Patch" format
// This supports the format used by some AI models for multi-file edits
func ApplyPatchTool() *ToolDef {
	return &ToolDef{
		Name:        "apply_patch",
		Description: "Apply a patch in the custom '*** Begin Patch' format for multi-file changes.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"patch": map[string]interface{}{
					"type":        "string",
					"description": "The patch content in '*** Begin Patch' format",
				},
			},
			"required": []string{"patch"},
		},
		Execute: func(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
			patchContent, _ := input["patch"].(string)
			if patchContent == "" {
				return &ToolResult{Output: "Error: patch content is required", IsError: true}, nil
			}

			sections, err := parseCustomPatch(patchContent)
			if err != nil {
				return &ToolResult{Output: fmt.Sprintf("Error parsing patch: %s", err.Error()), IsError: true}, nil
			}

			if len(sections) == 0 {
				return &ToolResult{Output: "Error: no file changes found in patch", IsError: true}, nil
			}

			var results []string
			var diffList []*DiffData
			applied := 0
			failed := 0

			for _, section := range sections {
				filePath := section.path
				if !filepath.IsAbs(filePath) {
					filePath = filepath.Join(tc.WorkDir, filePath)
				}

				if section.isNew {
					dir := filepath.Dir(filePath)
					if err := os.MkdirAll(dir, 0755); err != nil {
						results = append(results, fmt.Sprintf("Error creating directory for %s: %s", section.path, err.Error()))
						failed++
						continue
					}
					if err := os.WriteFile(filePath, []byte(section.newContent), 0644); err != nil {
						results = append(results, fmt.Sprintf("Error creating %s: %s", section.path, err.Error()))
						failed++
						continue
					}
					results = append(results, fmt.Sprintf("Created: %s", section.path))
					diffList = append(diffList, &DiffData{
						OldContent: "",
						NewContent: section.newContent,
						FilePath:   filePath,
						Language:   inferLanguage(filePath),
					})
					applied++
				} else if section.isDelete {
					if err := os.Remove(filePath); err != nil {
						results = append(results, fmt.Sprintf("Error deleting %s: %s", section.path, err.Error()))
						failed++
						continue
					}
					results = append(results, fmt.Sprintf("Deleted: %s", section.path))
					applied++
				} else {
					// Read existing file
					data, err := os.ReadFile(filePath)
					if err != nil {
						results = append(results, fmt.Sprintf("Error reading %s: %s", section.path, err.Error()))
						failed++
						continue
					}

					content := string(data)
					newContent, err := applyCustomPatchChanges(content, section.changes)
					if err != nil {
						results = append(results, fmt.Sprintf("Error applying changes to %s: %s", section.path, err.Error()))
						failed++
						continue
					}

					if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
						results = append(results, fmt.Sprintf("Error writing %s: %s", section.path, err.Error()))
						failed++
						continue
					}
					results = append(results, fmt.Sprintf("Modified: %s (%d changes)", section.path, len(section.changes)))
					diffList = append(diffList, &DiffData{
						OldContent: content,
						NewContent: newContent,
						FilePath:   filePath,
						Language:   inferLanguage(filePath),
					})
					applied++
				}
			}

			summary := fmt.Sprintf("\nApplied %d/%d file changes", applied, len(sections))
			if failed > 0 {
				summary += fmt.Sprintf(" (%d failed)", failed)
			}

			return &ToolResult{
				Output:       strings.Join(results, "\n") + summary,
				IsError:      failed > 0 && applied == 0,
				DiffDataList: diffList,
			}, nil
		},
	}
}

type customPatchSection struct {
	path       string
	isNew      bool
	isDelete   bool
	newContent string
	changes    []customPatchChange
}

type customPatchChange struct {
	before string
	after  string
}

func parseCustomPatch(content string) ([]customPatchSection, error) {
	var sections []customPatchSection

	lines := strings.Split(content, "\n")
	i := 0

	// Skip until we find "*** Begin Patch" or start parsing directly
	for i < len(lines) {
		if strings.TrimSpace(lines[i]) == "*** Begin Patch" {
			i++
			break
		}
		// If first line starts with ***, assume it's a file header
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "***") {
			break
		}
		i++
	}

	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		// End of patch
		if line == "*** End Patch" {
			break
		}

		// File header: *** path/to/file
		if strings.HasPrefix(line, "***") && !strings.HasPrefix(line, "*** Begin") && !strings.HasPrefix(line, "*** End") {
			path := strings.TrimSpace(strings.TrimPrefix(line, "***"))
			i++

			section := customPatchSection{path: path}

			// Check for new/delete markers
			if i < len(lines) {
				nextLine := strings.TrimSpace(lines[i])
				if nextLine == "*** NEW FILE ***" || nextLine == "+++ new file" {
					section.isNew = true
					i++
					// Collect new file content
					var newContent strings.Builder
					for i < len(lines) {
						if strings.HasPrefix(strings.TrimSpace(lines[i]), "***") {
							break
						}
						if lines[i] != "" && lines[i][0] == '+' {
							newContent.WriteString(lines[i][1:])
						} else {
							newContent.WriteString(lines[i])
						}
						newContent.WriteString("\n")
						i++
					}
					section.newContent = newContent.String()
					sections = append(sections, section)
					continue
				}
				if nextLine == "*** DELETE FILE ***" || nextLine == "--- deleted" {
					section.isDelete = true
					i++
					sections = append(sections, section)
					continue
				}
			}

			// Parse before/after blocks
			for i < len(lines) {
				trimmed := strings.TrimSpace(lines[i])
				if strings.HasPrefix(trimmed, "***") && !strings.HasPrefix(trimmed, "*** before") && !strings.HasPrefix(trimmed, "*** after") {
					break
				}

				if strings.HasPrefix(trimmed, "*** before") || trimmed == "---" {
					i++
					var before strings.Builder
					for i < len(lines) {
						t := strings.TrimSpace(lines[i])
						if strings.HasPrefix(t, "*** after") || t == "+++" || (strings.HasPrefix(t, "***") && !strings.HasPrefix(t, "*** before")) {
							break
						}
						if len(lines[i]) > 0 && lines[i][0] == '-' {
							before.WriteString(lines[i][1:])
						} else if len(lines[i]) > 0 && lines[i][0] == ' ' {
							before.WriteString(lines[i][1:])
						} else {
							before.WriteString(lines[i])
						}
						before.WriteString("\n")
						i++
					}

					if i < len(lines) && (strings.HasPrefix(strings.TrimSpace(lines[i]), "*** after") || strings.TrimSpace(lines[i]) == "+++") {
						i++
						var after strings.Builder
						for i < len(lines) {
							t := strings.TrimSpace(lines[i])
							if strings.HasPrefix(t, "***") || t == "---" {
								break
							}
							if len(lines[i]) > 0 && lines[i][0] == '+' {
								after.WriteString(lines[i][1:])
							} else if len(lines[i]) > 0 && lines[i][0] == ' ' {
								after.WriteString(lines[i][1:])
							} else {
								after.WriteString(lines[i])
							}
							after.WriteString("\n")
							i++
						}

						section.changes = append(section.changes, customPatchChange{
							before: before.String(),
							after:  after.String(),
						})
					}
				} else {
					i++
				}
			}

			sections = append(sections, section)
		} else {
			i++
		}
	}

	return sections, nil
}

func applyCustomPatchChanges(content string, changes []customPatchChange) (string, error) {
	result := content
	for _, change := range changes {
		before := strings.TrimRight(change.before, "\n")
		after := strings.TrimRight(change.after, "\n")

		if !strings.Contains(result, before) {
			// Try with normalized whitespace
			normalizedContent := normalizeTrailingWhitespace(result)
			normalizedBefore := normalizeTrailingWhitespace(before)
			if !strings.Contains(normalizedContent, normalizedBefore) {
				return "", fmt.Errorf("could not find before block in file:\n%s", truncateStr(before, 200))
			}
			// Find the actual position in the original content
			idx := strings.Index(normalizedContent, normalizedBefore)
			if idx >= 0 {
				// Map back to original content positions
				result = result[:idx] + after + result[idx+len(before):]
			}
		} else {
			result = strings.Replace(result, before, after, 1)
		}
	}
	return result, nil
}

func normalizeTrailingWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n")
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
