package tool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Plan mode tools - plan_enter and plan_exit
// Mirrors opencode's plan mode where the agent switches to read-only analysis mode

var (
	sessionPlanMode   = make(map[string]bool)
	sessionPlanModeMu sync.RWMutex
)

// IsPlanMode checks if a session is in plan mode
func IsPlanMode(sessionID string) bool {
	sessionPlanModeMu.RLock()
	defer sessionPlanModeMu.RUnlock()
	return sessionPlanMode[sessionID]
}

// SetPlanMode sets the plan mode for a session
func SetPlanMode(sessionID string, enabled bool) {
	sessionPlanModeMu.Lock()
	defer sessionPlanModeMu.Unlock()
	sessionPlanMode[sessionID] = enabled
}

func PlanEnterTool() *ToolDef {
	return &ToolDef{
		Name:        "plan_enter",
		Description: "Enter read-only plan mode for analysis and planning before making changes.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Brief description of what you're planning",
				},
			},
			"required": []string{"description"},
		},
		Execute: func(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
			description, _ := input["description"].(string)
			if description == "" {
				description = "Analysis and planning"
			}

			SetPlanMode(tc.SessionID, true)

			// Create plan file
			planDir := filepath.Join(tc.WorkDir, ".dcode", "plans")
			os.MkdirAll(planDir, 0755)

			planFile := filepath.Join(planDir, fmt.Sprintf("plan_%s.md", tc.SessionID[:8]))
			header := fmt.Sprintf("# Plan: %s\n\nSession: %s\n\n## Analysis\n\n", description, tc.SessionID)
			os.WriteFile(planFile, []byte(header), 0644)

			return &ToolResult{
				Output: fmt.Sprintf("Entered plan mode: %s\n\nYou are now in read-only analysis mode. You can:\n- Read and explore code\n- Search for patterns and definitions\n- Think through approaches\n- Write notes to .dcode/plans/\n\nUse plan_exit when you're ready to implement changes.", description),
			}, nil
		},
	}
}

func PlanExitTool() *ToolDef {
	return &ToolDef{
		Name:        "plan_exit",
		Description: "Exit plan mode and return to normal coding mode where you can make changes to files.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"summary": map[string]interface{}{
					"type":        "string",
					"description": "Summary of the plan and next steps",
				},
			},
		},
		Execute: func(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
			summary, _ := input["summary"].(string)

			SetPlanMode(tc.SessionID, false)

			// Append summary to plan file if it exists
			if summary != "" {
				planDir := filepath.Join(tc.WorkDir, ".dcode", "plans")
				planFile := filepath.Join(planDir, fmt.Sprintf("plan_%s.md", tc.SessionID[:8]))
				if _, err := os.Stat(planFile); err == nil {
					f, err := os.OpenFile(planFile, os.O_APPEND|os.O_WRONLY, 0644)
					if err == nil {
						f.WriteString(fmt.Sprintf("\n## Summary\n\n%s\n", summary))
						f.Close()
					}
				}
			}

			output := "Exited plan mode. You can now make changes to files."
			if summary != "" {
				output += fmt.Sprintf("\n\nPlan summary: %s", summary)
			}

			// List any plan files
			planDir := filepath.Join(tc.WorkDir, ".dcode", "plans")
			entries, err := os.ReadDir(planDir)
			if err == nil && len(entries) > 0 {
				var plans []string
				for _, e := range entries {
					if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
						plans = append(plans, e.Name())
					}
				}
				if len(plans) > 0 {
					output += fmt.Sprintf("\n\nPlan files: %s", strings.Join(plans, ", "))
				}
			}

			return &ToolResult{Output: output}, nil
		},
	}
}
