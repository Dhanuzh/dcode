package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// TodoItem represents a single todo item
type TodoItem struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"` // "not-started", "in-progress", "completed"
}

// sessionTodos stores todos per session
var (
	sessionTodos   = make(map[string][]TodoItem)
	sessionTodosMu sync.RWMutex
)

// TodoReadTool reads the current session's todo list
func TodoReadTool() *ToolDef {
	return &ToolDef{
		Name:        "todo_read",
		Description: "Read the current session's todo list with statuses.",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Execute: func(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
			sessionTodosMu.RLock()
			defer sessionTodosMu.RUnlock()

			todos, ok := sessionTodos[tc.SessionID]
			if !ok || len(todos) == 0 {
				return &ToolResult{Output: "No todos for this session."}, nil
			}

			lines := []string{"Todo List:"}
			for _, todo := range todos {
				icon := "○"
				switch todo.Status {
				case "in-progress":
					icon = "◐"
				case "completed":
					icon = "●"
				}
				lines = append(lines, fmt.Sprintf("  %s [%s] %s", icon, todo.ID, todo.Title))
			}

			stats := countTodoStats(todos)
			lines = append(lines, fmt.Sprintf("\n%d total | %d completed | %d in-progress | %d not-started",
				len(todos), stats["completed"], stats["in-progress"], stats["not-started"]))

			return &ToolResult{Output: strings.Join(lines, "\n")}, nil
		},
	}
}

// TodoWriteTool updates the session's todo list
func TodoWriteTool() *ToolDef {
	return &ToolDef{
		Name:        "todo_write",
		Description: "Update the session's todo list with new items and statuses.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"todos": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"id": map[string]interface{}{
								"type":        "string",
								"description": "Unique identifier for the todo",
							},
							"title": map[string]interface{}{
								"type":        "string",
								"description": "Description of the task",
							},
							"status": map[string]interface{}{
								"type":        "string",
								"description": "Status: not-started, in-progress, or completed",
								"enum":        []string{"not-started", "in-progress", "completed"},
							},
						},
						"required": []string{"id", "title", "status"},
					},
					"description": "Complete list of all todo items",
				},
			},
			"required": []string{"todos"},
		},
		Execute: func(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
			todosRaw, ok := input["todos"]
			if !ok {
				return &ToolResult{Output: "Error: todos array is required", IsError: true}, nil
			}

			todosJSON, _ := json.Marshal(todosRaw)
			var todos []TodoItem
			if err := json.Unmarshal(todosJSON, &todos); err != nil {
				return &ToolResult{Output: fmt.Sprintf("Error parsing todos: %v", err), IsError: true}, nil
			}

			sessionTodosMu.Lock()
			sessionTodos[tc.SessionID] = todos
			sessionTodosMu.Unlock()

			stats := countTodoStats(todos)
			return &ToolResult{Output: fmt.Sprintf("Updated todo list: %d items (%d completed, %d in-progress, %d not-started)",
				len(todos), stats["completed"], stats["in-progress"], stats["not-started"])}, nil
		},
	}
}

func countTodoStats(todos []TodoItem) map[string]int {
	stats := map[string]int{
		"completed":   0,
		"in-progress": 0,
		"not-started": 0,
	}
	for _, t := range todos {
		stats[t.Status]++
	}
	return stats
}

// GetSessionTodos returns the todos for a given session (for external use)
func GetSessionTodos(sessionID string) []TodoItem {
	sessionTodosMu.RLock()
	defer sessionTodosMu.RUnlock()
	return sessionTodos[sessionID]
}
