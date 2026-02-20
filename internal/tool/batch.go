package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// BatchTool executes multiple tool calls in parallel
// Mirrors opencode's batch tool for efficient parallel operations
func BatchTool() *ToolDef {
	return &ToolDef{
		Name:        "batch",
		Description: "Execute multiple independent tool calls in parallel.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"operations": map[string]interface{}{
					"type":        "array",
					"description": "List of tool operations to execute in parallel",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"tool": map[string]interface{}{
								"type":        "string",
								"description": "Name of the tool to call",
							},
							"input": map[string]interface{}{
								"type":        "object",
								"description": "Input parameters for the tool",
							},
						},
						"required": []string{"tool", "input"},
					},
				},
			},
			"required": []string{"operations"},
		},
		Execute: func(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
			opsRaw, ok := input["operations"]
			if !ok {
				return &ToolResult{Output: "Error: operations parameter required", IsError: true}, nil
			}

			opsJSON, err := json.Marshal(opsRaw)
			if err != nil {
				return &ToolResult{Output: "Error: invalid operations format", IsError: true}, nil
			}

			var operations []struct {
				Tool  string                 `json:"tool"`
				Input map[string]interface{} `json:"input"`
			}
			if err := json.Unmarshal(opsJSON, &operations); err != nil {
				return &ToolResult{Output: "Error: invalid operations format: " + err.Error(), IsError: true}, nil
			}

			if len(operations) == 0 {
				return &ToolResult{Output: "Error: at least one operation required", IsError: true}, nil
			}

			registry := GetRegistry()

			type result struct {
				index  int
				tool   string
				output string
				err    bool
			}

			results := make([]result, len(operations))
			var wg sync.WaitGroup

			for i, op := range operations {
				wg.Add(1)
				go func(idx int, toolName string, toolInput map[string]interface{}) {
					defer wg.Done()

					toolResult, err := registry.Execute(ctx, tc, toolName, toolInput)
					if err != nil {
						results[idx] = result{
							index:  idx,
							tool:   toolName,
							output: fmt.Sprintf("Error: %s", err.Error()),
							err:    true,
						}
						return
					}
					results[idx] = result{
						index:  idx,
						tool:   toolName,
						output: toolResult.Output,
						err:    toolResult.IsError,
					}
				}(i, op.Tool, op.Input)
			}

			wg.Wait()

			var sb strings.Builder
			successCount := 0
			errorCount := 0

			for _, r := range results {
				if r.err {
					errorCount++
					sb.WriteString(fmt.Sprintf("--- %s (operation %d) [ERROR] ---\n", r.tool, r.index+1))
				} else {
					successCount++
					sb.WriteString(fmt.Sprintf("--- %s (operation %d) [OK] ---\n", r.tool, r.index+1))
				}
				sb.WriteString(r.output)
				sb.WriteString("\n\n")
			}

			summary := fmt.Sprintf("Batch completed: %d/%d succeeded", successCount, len(operations))
			if errorCount > 0 {
				summary += fmt.Sprintf(", %d failed", errorCount)
			}
			sb.WriteString(summary)

			return &ToolResult{
				Output:  sb.String(),
				IsError: errorCount > 0 && successCount == 0,
			}, nil
		},
	}
}
