package tool

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// GitTool provides Git version control operations
func GitTool() *ToolDef {
	return &ToolDef{
		Name:        "Git",
		Description: "Execute Git commands for commits, branches, history, and repo management.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"operation": map[string]interface{}{
					"type":        "string",
					"description": "Git operation to perform",
					"enum": []string{
						"status",   // Show working tree status
						"diff",     // Show changes
						"log",      // Show commit history
						"branch",   // List or manage branches
						"add",      // Stage files
						"commit",   // Create commit
						"push",     // Push to remote
						"pull",     // Pull from remote
						"checkout", // Switch branches or restore files
						"reset",    // Reset current HEAD
						"stash",    // Stash changes
						"remote",   // Manage remotes
						"tag",      // Manage tags
						"show",     // Show commit/object
						"blame",    // Show who changed each line
						"custom",   // Custom git command
					},
				},
				"args": map[string]interface{}{
					"type":        "array",
					"description": "Additional arguments for the git command",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"message": map[string]interface{}{
					"type":        "string",
					"description": "Commit message (for commit operation)",
				},
				"files": map[string]interface{}{
					"type":        "array",
					"description": "Files to operate on",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
			"required": []string{"operation"},
		},
		Execute: executeGit,
	}
}

func executeGit(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
	operation, ok := input["operation"].(string)
	if !ok {
		return &ToolResult{
			Output:  "operation parameter is required",
			IsError: true,
		}, nil
	}

	// Build git command
	var cmdArgs []string

	switch operation {
	case "status":
		cmdArgs = []string{"status"}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		} else {
			// Default to short format if no args
			cmdArgs = append(cmdArgs, "--short", "--branch")
		}

	case "diff":
		cmdArgs = []string{"diff"}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		}
		if files, ok := input["files"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, "--")
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(files)...)
		}

	case "log":
		cmdArgs = []string{"log"}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		} else {
			// Default to concise format
			cmdArgs = append(cmdArgs, "--oneline", "--decorate", "-20")
		}

	case "branch":
		cmdArgs = []string{"branch"}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		} else {
			// Show all branches by default
			cmdArgs = append(cmdArgs, "-a")
		}

	case "add":
		cmdArgs = []string{"add"}
		if files, ok := input["files"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(files)...)
		} else {
			return &ToolResult{
				Output:  "files parameter is required for add operation",
				IsError: true,
			}, nil
		}

	case "commit":
		cmdArgs = []string{"commit"}
		message, hasMessage := input["message"].(string)
		if hasMessage && message != "" {
			cmdArgs = append(cmdArgs, "-m", message)
		} else {
			return &ToolResult{
				Output:  "message parameter is required for commit operation",
				IsError: true,
			}, nil
		}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		}

	case "push":
		cmdArgs = []string{"push"}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		}

	case "pull":
		cmdArgs = []string{"pull"}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		}

	case "checkout":
		cmdArgs = []string{"checkout"}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		} else {
			return &ToolResult{
				Output:  "args parameter is required for checkout operation (branch or file)",
				IsError: true,
			}, nil
		}

	case "reset":
		cmdArgs = []string{"reset"}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		}

	case "stash":
		cmdArgs = []string{"stash"}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		}

	case "remote":
		cmdArgs = []string{"remote"}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		} else {
			cmdArgs = append(cmdArgs, "-v")
		}

	case "tag":
		cmdArgs = []string{"tag"}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		}

	case "show":
		cmdArgs = []string{"show"}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		}

	case "blame":
		cmdArgs = []string{"blame"}
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(args)...)
		}
		if files, ok := input["files"].([]interface{}); ok && len(files) > 0 {
			cmdArgs = append(cmdArgs, interfaceSliceToStringSlice(files)...)
		} else {
			return &ToolResult{
				Output:  "files parameter is required for blame operation",
				IsError: true,
			}, nil
		}

	case "custom":
		// Allow custom git commands
		if args, ok := input["args"].([]interface{}); ok {
			cmdArgs = interfaceSliceToStringSlice(args)
		} else {
			return &ToolResult{
				Output:  "args parameter is required for custom operation",
				IsError: true,
			}, nil
		}

	default:
		return &ToolResult{
			Output:  fmt.Sprintf("unknown git operation: %s", operation),
			IsError: true,
		}, nil
	}

	// Execute git command
	cmd := exec.CommandContext(ctx, "git", cmdArgs...)
	cmd.Dir = tc.WorkDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &ToolResult{
			Output:  fmt.Sprintf("Git error: %s\nOutput: %s", err, string(output)),
			IsError: true,
		}, nil
	}

	return &ToolResult{
		Output:  string(output),
		IsError: false,
	}, nil
}

// interfaceSliceToStringSlice converts []interface{} to []string
func interfaceSliceToStringSlice(slice []interface{}) []string {
	result := make([]string, len(slice))
	for i, v := range slice {
		if str, ok := v.(string); ok {
			result[i] = str
		} else {
			result[i] = fmt.Sprintf("%v", v)
		}
	}
	return result
}

// Helper to check if git is available
func isGitAvailable() bool {
	cmd := exec.Command("git", "--version")
	err := cmd.Run()
	return err == nil
}

// Helper to get current branch
func getCurrentGitBranch(workDir string) (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// Helper to check if directory is a git repo
func isGitRepo(workDir string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = workDir
	err := cmd.Run()
	return err == nil
}
