package tool

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CodeSearchTool provides semantic code search capabilities
// Mirrors opencode's codesearch tool
func CodeSearchTool() *ToolDef {
	return &ToolDef{
		Name:        "codesearch",
		Description: "Search code for definitions, references, and symbols. Falls back to ripgrep.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query - can be a symbol name, function signature, or pattern",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"description": "Type of search: definition, reference, symbol, or pattern",
					"enum":        []string{"definition", "reference", "symbol", "pattern"},
				},
				"language": map[string]interface{}{
					"type":        "string",
					"description": "Filter by programming language (e.g., go, typescript, python)",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Directory or file to search in",
				},
			},
			"required": []string{"query"},
		},
		Execute: func(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
			query, _ := input["query"].(string)
			if query == "" {
				return &ToolResult{Output: "Error: query is required", IsError: true}, nil
			}

			searchType, _ := input["type"].(string)
			if searchType == "" {
				searchType = "pattern"
			}

			language, _ := input["language"].(string)
			searchPath, _ := input["path"].(string)
			if searchPath == "" {
				searchPath = tc.WorkDir
			} else if !strings.HasPrefix(searchPath, "/") {
				searchPath = tc.WorkDir + "/" + searchPath
			}

			var args []string

			switch searchType {
			case "definition":
				// Search for function/class/type definitions
				patterns := getDefinitionPatterns(query, language)
				for _, pattern := range patterns {
					result, err := runRipgrep(ctx, pattern, searchPath, language)
					if err == nil && result != "" {
						return &ToolResult{
							Output: fmt.Sprintf("Definitions of '%s':\n\n%s", query, result),
						}, nil
					}
				}
				// Fallback to basic pattern
				args = []string{"--line-number", "--no-heading", "--color=never", "-e", query, searchPath}

			case "reference":
				// Search for references/usages
				args = []string{"--line-number", "--no-heading", "--color=never", "--word-regexp", "-e", query, searchPath}

			case "symbol":
				// Search for symbol definitions (broader than definition)
				args = []string{"--line-number", "--no-heading", "--color=never", "-e", fmt.Sprintf(`\b%s\b`, query), searchPath}

			default:
				// Pattern search
				args = []string{"--line-number", "--no-heading", "--color=never", "-e", query, searchPath}
			}

			if language != "" {
				args = append([]string{"--type", mapLanguageToRgType(language)}, args...)
			}

			cmd := exec.CommandContext(ctx, "rg", args...)
			cmd.Dir = tc.WorkDir
			output, err := cmd.CombinedOutput()
			if err != nil {
				if len(output) == 0 {
					return &ToolResult{Output: fmt.Sprintf("No results found for '%s' (type: %s)", query, searchType)}, nil
				}
			}

			result := string(output)
			lines := strings.Split(result, "\n")
			if len(lines) > 200 {
				result = strings.Join(lines[:200], "\n") + fmt.Sprintf("\n\n... truncated (%d total matches)", len(lines))
			}

			return &ToolResult{
				Output: fmt.Sprintf("Code search results for '%s' (type: %s):\n\n%s", query, searchType, result),
			}, nil
		},
	}
}

func getDefinitionPatterns(query, language string) []string {
	switch language {
	case "go":
		return []string{
			fmt.Sprintf(`func\s+%s\b`, query),
			fmt.Sprintf(`func\s+\([^)]+\)\s+%s\b`, query),
			fmt.Sprintf(`type\s+%s\b`, query),
			fmt.Sprintf(`var\s+%s\b`, query),
			fmt.Sprintf(`const\s+%s\b`, query),
		}
	case "typescript", "javascript", "ts", "js":
		return []string{
			fmt.Sprintf(`(?:export\s+)?(?:async\s+)?function\s+%s\b`, query),
			fmt.Sprintf(`(?:export\s+)?(?:const|let|var)\s+%s\b`, query),
			fmt.Sprintf(`(?:export\s+)?(?:class|interface|type|enum)\s+%s\b`, query),
		}
	case "python", "py":
		return []string{
			fmt.Sprintf(`(?:def|class)\s+%s\b`, query),
			fmt.Sprintf(`%s\s*=`, query),
		}
	case "rust", "rs":
		return []string{
			fmt.Sprintf(`(?:pub\s+)?(?:fn|struct|enum|trait|type|const|static)\s+%s\b`, query),
			fmt.Sprintf(`(?:pub\s+)?impl\s+%s\b`, query),
		}
	default:
		return []string{
			fmt.Sprintf(`(?:func|function|def|class|type|struct|interface|enum|trait)\s+%s\b`, query),
		}
	}
}

func runRipgrep(ctx context.Context, pattern, path, language string) (string, error) {
	args := []string{"--line-number", "--no-heading", "--color=never", "-e", pattern, path}
	if language != "" {
		args = append([]string{"--type", mapLanguageToRgType(language)}, args...)
	}

	cmd := exec.CommandContext(ctx, "rg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil && len(output) == 0 {
		return "", err
	}
	return string(output), nil
}

func mapLanguageToRgType(language string) string {
	typeMap := map[string]string{
		"go":         "go",
		"typescript": "ts",
		"ts":         "ts",
		"javascript": "js",
		"js":         "js",
		"python":     "py",
		"py":         "py",
		"rust":       "rust",
		"rs":         "rust",
		"java":       "java",
		"c":          "c",
		"cpp":        "cpp",
		"ruby":       "ruby",
		"rb":         "ruby",
		"php":        "php",
		"swift":      "swift",
		"kotlin":     "kotlin",
		"scala":      "scala",
	}
	if t, ok := typeMap[language]; ok {
		return t
	}
	return language
}
