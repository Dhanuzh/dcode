package tool

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// BashTool executes shell commands
func BashTool() *ToolDef {
	return &ToolDef{
		Name:        "bash",
		Description: "Execute a shell command in the project directory. Default timeout: 120s.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "The shell command to execute",
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "Timeout in seconds (default: 120)",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Brief description of what the command does",
				},
			},
			"required": []string{"command"},
		},
		Execute: func(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
			command, _ := input["command"].(string)
			if command == "" {
				return &ToolResult{Output: "Error: command is required", IsError: true}, nil
			}

			timeoutSecs := 120
			if v, ok := input["timeout"].(float64); ok && v > 0 {
				timeoutSecs = int(v)
			}

			workDir := tc.WorkDir
			if workDir == "" {
				workDir = "."
			}

			// Create context with timeout
			timeout := time.Duration(timeoutSecs) * time.Second
			cmdCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			cmd := exec.CommandContext(cmdCtx, "bash", "-c", command)
			cmd.Dir, _ = filepath.Abs(workDir)

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()

			output := stdout.String()
			errOutput := stderr.String()

			if errOutput != "" {
				output += "\n" + errOutput
			}

			// Truncate output if too large (30KB cap to save tokens)
			if len(output) > 30*1024 {
				output = output[:15*1024] + "\n\n... (output truncated) ...\n\n" + output[len(output)-15*1024:]
			}

			if err != nil {
				if cmdCtx.Err() == context.DeadlineExceeded {
					return &ToolResult{
						Output:  fmt.Sprintf("Command timed out after %d seconds.\nPartial output:\n%s", timeoutSecs, output),
						IsError: true,
					}, nil
				}
				exitCode := -1
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				}
				return &ToolResult{
					Output:  fmt.Sprintf("Command failed (exit code %d):\n%s", exitCode, output),
					IsError: true,
				}, nil
			}

			if strings.TrimSpace(output) == "" {
				output = "(no output)"
			}

			return &ToolResult{Output: output}, nil
		},
	}
}
