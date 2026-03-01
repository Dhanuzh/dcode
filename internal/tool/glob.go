package tool

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// GlobTool finds files matching glob patterns using ripgrep or native glob
func GlobTool() *ToolDef {
	return &ToolDef{
		Name:        "glob",
		Description: "Find files matching a glob pattern. Returns up to 100 matches sorted by modification time.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "Glob pattern to match (e.g., '**/*.go', 'src/**/*.ts', '*.md')",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Directory to search in (default: project root)",
				},
			},
			"required": []string{"pattern"},
		},
		Execute: func(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
			pattern, _ := input["pattern"].(string)
			if pattern == "" {
				return &ToolResult{Output: "Error: pattern is required", IsError: true}, nil
			}

			searchDir := tc.WorkDir
			if v, ok := input["path"].(string); ok && v != "" {
				if !filepath.IsAbs(v) && tc.WorkDir != "" {
					searchDir = filepath.Join(tc.WorkDir, v)
				} else {
					searchDir = v
				}
			}
			if searchDir == "" {
				searchDir = "."
			}

			// Try ripgrep first for speed
			matches, err := globWithRipgrep(ctx, searchDir, pattern)
			if err != nil {
				// Fallback to native glob
				matches, err = globNative(searchDir, pattern)
				if err != nil {
					return &ToolResult{Output: fmt.Sprintf("Error: %v", err), IsError: true}, nil
				}
			}

			if len(matches) == 0 {
				return &ToolResult{Output: fmt.Sprintf("No files matching pattern: %s", pattern)}, nil
			}

			// Sort by modification time (newest first)
			type fileInfo struct {
				path    string
				modTime time.Time
			}
			files := make([]fileInfo, 0, len(matches))
			for _, m := range matches {
				info, err := os.Stat(m)
				if err != nil {
					continue
				}
				rel, _ := filepath.Rel(searchDir, m)
				if rel == "" {
					rel = m
				}
				files = append(files, fileInfo{path: rel, modTime: info.ModTime()})
			}
			sort.Slice(files, func(i, j int) bool {
				return files[i].modTime.After(files[j].modTime)
			})

			// Limit to 100
			if len(files) > 100 {
				files = files[:100]
			}

			lines := make([]string, len(files))
			for i, f := range files {
				lines[i] = f.path
			}

			return &ToolResult{Output: fmt.Sprintf("Found %d files matching '%s':\n\n%s", len(files), pattern, strings.Join(lines, "\n"))}, nil
		},
	}
}

func globWithRipgrep(ctx context.Context, dir, pattern string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "rg", "--files", "--glob", pattern, dir)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	matches := make([]string, 0, len(lines))
	for _, line := range lines {
		if line != "" {
			matches = append(matches, line)
		}
	}
	return matches, nil
}

func globNative(dir, pattern string) ([]string, error) {
	var matches []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		// Skip common ignored directories
		if info.IsDir() {
			base := filepath.Base(path)
			ignored := []string{".git", "node_modules", "__pycache__", ".next", "dist", "build", ".cache", "vendor"}
			for _, ig := range ignored {
				if base == ig {
					return filepath.SkipDir
				}
			}
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		matched, _ := filepath.Match(pattern, filepath.Base(path))
		if !matched {
			// Try matching against relative path for ** patterns
			matched = matchGlob(pattern, rel)
		}
		if matched {
			matches = append(matches, path)
		}
		return nil
	})
	return matches, err
}

func matchGlob(pattern, path string) bool {
	// Simple ** glob support
	if strings.Contains(pattern, "**") {
		parts := strings.Split(pattern, "**")
		if len(parts) == 2 {
			suffix := strings.TrimPrefix(parts[1], "/")
			if suffix == "" {
				return true
			}
			matched, _ := filepath.Match(suffix, filepath.Base(path))
			return matched
		}
	}
	matched, _ := filepath.Match(pattern, path)
	return matched
}
