package tool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PatchTool applies unified diff patches to files
func PatchTool() *ToolDef {
	return &ToolDef{
		Name:        "patch",
		Description: "Apply a unified diff patch to files. Standard unified diff format.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"patch": map[string]interface{}{
					"type":        "string",
					"description": "The unified diff patch to apply. Should follow standard patch format with --- and +++ headers and @@ hunks.",
				},
			},
			"required": []string{"patch"},
		},
		Execute: func(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
			patch, _ := input["patch"].(string)
			if patch == "" {
				return &ToolResult{Output: "Error: patch is required", IsError: true}, nil
			}

			results := []string{}
			errors := []string{}
			var diffList []*DiffData

			// Parse patch into file sections
			sections := parsePatchSections(patch)

			for _, section := range sections {
				targetPath := section.path
				if !filepath.IsAbs(targetPath) && tc.WorkDir != "" {
					targetPath = filepath.Join(tc.WorkDir, targetPath)
				}

				if section.isDelete {
					if err := os.Remove(targetPath); err != nil {
						errors = append(errors, fmt.Sprintf("Failed to delete %s: %v", section.path, err))
					} else {
						results = append(results, fmt.Sprintf("Deleted %s", section.path))
					}
					continue
				}

				if section.isNew {
					dir := filepath.Dir(targetPath)
					os.MkdirAll(dir, 0755)
					if err := os.WriteFile(targetPath, []byte(section.newContent), 0644); err != nil {
						errors = append(errors, fmt.Sprintf("Failed to create %s: %v", section.path, err))
					} else {
						results = append(results, fmt.Sprintf("Created %s", section.path))
						diffList = append(diffList, &DiffData{
							OldContent: "",
							NewContent: section.newContent,
							FilePath:   targetPath,
							Language:   inferLanguage(targetPath),
						})
					}
					continue
				}

				// Apply hunks to existing file
				data, err := os.ReadFile(targetPath)
				if err != nil {
					errors = append(errors, fmt.Sprintf("Failed to read %s: %v", section.path, err))
					continue
				}

				oldContent := string(data)
				newContent, err := applyHunks(oldContent, section.hunks)
				if err != nil {
					errors = append(errors, fmt.Sprintf("Failed to apply patch to %s: %v", section.path, err))
					continue
				}

				if err := os.WriteFile(targetPath, []byte(newContent), 0644); err != nil {
					errors = append(errors, fmt.Sprintf("Failed to write %s: %v", section.path, err))
				} else {
					results = append(results, fmt.Sprintf("Patched %s", section.path))
					diffList = append(diffList, &DiffData{
						OldContent: oldContent,
						NewContent: newContent,
						FilePath:   targetPath,
						Language:   inferLanguage(targetPath),
					})
				}
			}

			output := strings.Join(results, "\n")
			if len(errors) > 0 {
				output += "\nErrors:\n" + strings.Join(errors, "\n")
			}
			if output == "" {
				return &ToolResult{Output: "No changes applied", IsError: true}, nil
			}
			return &ToolResult{
				Output:       output,
				IsError:      len(errors) > 0 && len(results) == 0,
				DiffDataList: diffList,
			}, nil
		},
	}
}

type patchSection struct {
	path       string
	isNew      bool
	isDelete   bool
	newContent string
	hunks      []patchHunk
}

type patchHunk struct {
	oldStart int
	oldCount int
	newStart int
	newCount int
	lines    []string
}

func parsePatchSections(patch string) []patchSection {
	lines := strings.Split(patch, "\n")
	sections := []patchSection{}
	var current *patchSection
	var currentHunk *patchHunk
	newContent := strings.Builder{}

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if strings.HasPrefix(line, "--- ") {
			if current != nil {
				if currentHunk != nil {
					current.hunks = append(current.hunks, *currentHunk)
				}
				if current.isNew {
					current.newContent = newContent.String()
				}
				sections = append(sections, *current)
			}
			current = &patchSection{}
			currentHunk = nil
			newContent.Reset()

			oldPath := strings.TrimPrefix(line, "--- ")
			oldPath = strings.TrimPrefix(oldPath, "a/")
			if oldPath == "/dev/null" {
				current.isNew = true
			}
			continue
		}

		if strings.HasPrefix(line, "+++ ") {
			newPath := strings.TrimPrefix(line, "+++ ")
			newPath = strings.TrimPrefix(newPath, "b/")
			if newPath == "/dev/null" {
				if current != nil {
					current.isDelete = true
				}
			} else if current != nil {
				current.path = newPath
			}
			continue
		}

		if strings.HasPrefix(line, "@@ ") {
			if currentHunk != nil && current != nil {
				current.hunks = append(current.hunks, *currentHunk)
			}
			currentHunk = &patchHunk{}
			// Parse @@ -start,count +start,count @@
			fmt.Sscanf(line, "@@ -%d,%d +%d,%d @@",
				&currentHunk.oldStart, &currentHunk.oldCount,
				&currentHunk.newStart, &currentHunk.newCount)
			if currentHunk.oldStart == 0 {
				fmt.Sscanf(line, "@@ -%d +%d,%d @@",
					&currentHunk.oldStart,
					&currentHunk.newStart, &currentHunk.newCount)
				currentHunk.oldCount = 1
			}
			continue
		}

		if current != nil && current.isNew {
			if strings.HasPrefix(line, "+") {
				newContent.WriteString(strings.TrimPrefix(line, "+") + "\n")
			}
			continue
		}

		if currentHunk != nil {
			currentHunk.lines = append(currentHunk.lines, line)
		}
	}

	if current != nil {
		if currentHunk != nil {
			current.hunks = append(current.hunks, *currentHunk)
		}
		if current.isNew {
			current.newContent = newContent.String()
		}
		sections = append(sections, *current)
	}

	return sections
}

func applyHunks(content string, hunks []patchHunk) (string, error) {
	lines := strings.Split(content, "\n")
	offset := 0

	for _, hunk := range hunks {
		startLine := hunk.oldStart - 1 + offset
		if startLine < 0 {
			startLine = 0
		}

		newLines := []string{}
		removedCount := 0
		addedCount := 0

		for _, line := range hunk.lines {
			if strings.HasPrefix(line, "-") {
				removedCount++
			} else if strings.HasPrefix(line, "+") {
				newLines = append(newLines, strings.TrimPrefix(line, "+"))
				addedCount++
			} else if strings.HasPrefix(line, " ") {
				newLines = append(newLines, strings.TrimPrefix(line, " "))
			} else {
				newLines = append(newLines, line)
			}
		}

		endLine := startLine + hunk.oldCount
		if endLine > len(lines) {
			endLine = len(lines)
		}

		result := make([]string, 0, len(lines)+addedCount-removedCount)
		result = append(result, lines[:startLine]...)
		result = append(result, newLines...)
		if endLine < len(lines) {
			result = append(result, lines[endLine:]...)
		}

		lines = result
		offset += addedCount - removedCount
	}

	return strings.Join(lines, "\n"), nil
}
