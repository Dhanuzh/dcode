package tool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LsTool lists directory contents
func LsTool() *ToolDef {
	return &ToolDef{
		Name:        "ls",
		Description: "List directory contents as a tree. Shows files with sizes. Limited to 100 entries.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Directory path to list (default: project root)",
				},
				"depth": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum depth to traverse (default: 3)",
				},
			},
		},
		Execute: func(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
			dir := tc.WorkDir
			if v, ok := input["path"].(string); ok && v != "" {
				if !filepath.IsAbs(v) && tc.WorkDir != "" {
					dir = filepath.Join(tc.WorkDir, v)
				} else {
					dir = v
				}
			}
			if dir == "" {
				dir = "."
			}

			maxDepth := 3
			if v, ok := input["depth"].(float64); ok && v > 0 {
				maxDepth = int(v)
			}

			entries := []string{}
			count := 0
			maxEntries := 100

			err := listDir(dir, "", 0, maxDepth, &entries, &count, maxEntries)
			if err != nil {
				return &ToolResult{Output: fmt.Sprintf("Error listing directory: %v", err), IsError: true}, nil
			}

			if len(entries) == 0 {
				return &ToolResult{Output: fmt.Sprintf("Directory is empty: %s", dir)}, nil
			}

			result := fmt.Sprintf("Directory: %s\n\n%s", dir, strings.Join(entries, "\n"))
			if count >= maxEntries {
				result += fmt.Sprintf("\n\n... (truncated at %d entries)", maxEntries)
			}

			return &ToolResult{Output: result}, nil
		},
	}
}

var ignoredDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"__pycache__":  true,
	".next":        true,
	".cache":       true,
	"dist":         true,
	"build":        true,
	"vendor":       true,
	".venv":        true,
	"venv":         true,
	".tox":         true,
	"target":       true,
	".idea":        true,
	".vscode":      false, // Usually want to see this
}

func listDir(dir, prefix string, depth, maxDepth int, entries *[]string, count *int, maxEntries int) error {
	if *count >= maxEntries {
		return nil
	}
	if depth >= maxDepth {
		return nil
	}

	items, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for i, item := range items {
		if *count >= maxEntries {
			break
		}

		name := item.Name()

		// Skip ignored directories
		if item.IsDir() {
			if ignored, ok := ignoredDirs[name]; ok && ignored {
				continue
			}
		}

		isLast := i == len(items)-1
		connector := "├── "
		childPrefix := prefix + "│   "
		if isLast {
			connector = "└── "
			childPrefix = prefix + "    "
		}

		info, _ := item.Info()
		if item.IsDir() {
			*entries = append(*entries, fmt.Sprintf("%s%s%s/", prefix, connector, name))
			*count++
			listDir(filepath.Join(dir, name), childPrefix, depth+1, maxDepth, entries, count, maxEntries)
		} else {
			size := ""
			if info != nil {
				size = formatSize(info.Size())
			}
			*entries = append(*entries, fmt.Sprintf("%s%s%s  %s", prefix, connector, name, size))
			*count++
		}
	}

	return nil
}

func formatSize(size int64) string {
	switch {
	case size < 1024:
		return fmt.Sprintf("%dB", size)
	case size < 1024*1024:
		return fmt.Sprintf("%.1fKB", float64(size)/1024)
	case size < 1024*1024*1024:
		return fmt.Sprintf("%.1fMB", float64(size)/(1024*1024))
	default:
		return fmt.Sprintf("%.1fGB", float64(size)/(1024*1024*1024))
	}
}
