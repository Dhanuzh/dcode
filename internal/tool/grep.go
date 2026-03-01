package tool

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// GrepTool searches file contents using regex patterns
func GrepTool() *ToolDef {
	return &ToolDef{
		Name:        "grep",
		Description: "Search file contents using regex. Returns matching lines with paths and line numbers.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "Regular expression pattern to search for",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Directory or file to search in (default: project root)",
				},
				"include": map[string]interface{}{
					"type":        "string",
					"description": "File pattern to include (e.g., '*.go', '*.ts')",
				},
			},
			"required": []string{"pattern"},
		},
		Execute: func(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
			pattern, _ := input["pattern"].(string)
			if pattern == "" {
				return &ToolResult{Output: "Error: pattern is required", IsError: true}, nil
			}

			searchPath := tc.WorkDir
			if v, ok := input["path"].(string); ok && v != "" {
				if !filepath.IsAbs(v) && tc.WorkDir != "" {
					searchPath = filepath.Join(tc.WorkDir, v)
				} else {
					searchPath = v
				}
			}
			if searchPath == "" {
				searchPath = "."
			}

			// Build ripgrep command
			args := []string{
				"--line-number",
				"--no-heading",
				"--color=never",
				"--max-count=100",
				"--max-filesize=1M",
			}

			if include, ok := input["include"].(string); ok && include != "" {
				args = append(args, "--glob", include)
			}

			args = append(args, pattern, searchPath)

			cmd := exec.CommandContext(ctx, "rg", args...)
			output, err := cmd.CombinedOutput()

			result := strings.TrimSpace(string(output))

			if err != nil {
				// ripgrep returns exit code 1 for no matches
				if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
					return &ToolResult{Output: fmt.Sprintf("No matches found for pattern: %s", pattern)}, nil
				}
				// Try fallback with grep
				return grepFallback(ctx, searchPath, pattern, input)
			}

			if result == "" {
				return &ToolResult{Output: fmt.Sprintf("No matches found for pattern: %s", pattern)}, nil
			}

			// Count matches
			lines := strings.Split(result, "\n")
			if len(lines) > 200 {
				result = strings.Join(lines[:200], "\n") + fmt.Sprintf("\n\n... (%d more matches truncated)", len(lines)-200)
			}

			return &ToolResult{Output: fmt.Sprintf("Found %d matches for '%s':\n\n%s", len(lines), pattern, result)}, nil
		},
	}
}

func grepFallback(ctx context.Context, searchPath, pattern string, input map[string]interface{}) (*ToolResult, error) {
	args := []string{"-rn", "--color=never"}

	if include, ok := input["include"].(string); ok && include != "" {
		args = append(args, "--include="+include)
	}

	args = append(args, pattern, searchPath)

	cmd := exec.CommandContext(ctx, "grep", args...)
	output, err := cmd.CombinedOutput()

	result := strings.TrimSpace(string(output))
	if err != nil || result == "" {
		return &ToolResult{Output: fmt.Sprintf("No matches found for pattern: %s", pattern)}, nil
	}

	lines := strings.Split(result, "\n")
	if len(lines) > 200 {
		result = strings.Join(lines[:200], "\n") + fmt.Sprintf("\n... (%d more matches truncated)", len(lines)-200)
	}

	return &ToolResult{Output: fmt.Sprintf("Found %d matches for '%s':\n\n%s", len(lines), pattern, result)}, nil
}
