package tool

import (
	"context"
	"fmt"
)

// TaskTool spawns a subtask/subagent for parallel work
func TaskTool() *ToolDef {
	return &ToolDef{
		Name:        "task",
		Description: "Spawn a subtask as a separate agent session for parallel work.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"prompt": map[string]interface{}{
					"type":        "string",
					"description": "Detailed instructions for the subtask agent",
				},
				"agent": map[string]interface{}{
					"type":        "string",
					"description": "Agent type to use: 'explorer' (fast read-only), 'researcher' (general purpose). Default: explorer",
					"enum":        []string{"explorer", "researcher"},
				},
			},
			"required": []string{"prompt"},
		},
		Execute: func(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
			prompt, _ := input["prompt"].(string)
			if prompt == "" {
				return &ToolResult{Output: "Error: prompt is required", IsError: true}, nil
			}

			agentType := "explorer"
			if v, ok := input["agent"].(string); ok && v != "" {
				agentType = v
			}

			// Note: Actual subtask execution requires the session/prompt engine
			// This is a placeholder that will be wired up when the session system is complete
			return &ToolResult{
				Output: fmt.Sprintf("[Task spawned] Agent: %s\nPrompt: %s\n\nNote: Subtask execution will run when the session engine processes this.", agentType, prompt),
			}, nil
		},
	}
}
