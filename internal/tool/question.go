package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// Question tool - allows the AI to ask the user questions during execution
// Mirrors opencode's question tool

var (
	pendingQuestions   = make(map[string]*PendingQuestion)
	pendingQuestionsMu sync.RWMutex
)

// PendingQuestion represents a question waiting for user response
type PendingQuestion struct {
	ID       string   `json:"id"`
	Question string   `json:"question"`
	Header   string   `json:"header"`
	Options  []Option `json:"options"`
	Multiple bool     `json:"multiple"`
	Custom   bool     `json:"custom"`
	Answer   []string `json:"answer,omitempty"`
	Answered chan struct{}
}

// Option represents a question choice
type Option struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// GetPendingQuestion returns a pending question by session ID
func GetPendingQuestion(sessionID string) *PendingQuestion {
	pendingQuestionsMu.RLock()
	defer pendingQuestionsMu.RUnlock()
	return pendingQuestions[sessionID]
}

// AnswerQuestion answers a pending question
func AnswerQuestion(sessionID string, answer []string) {
	pendingQuestionsMu.Lock()
	q, ok := pendingQuestions[sessionID]
	if ok {
		q.Answer = answer
		close(q.Answered)
		delete(pendingQuestions, sessionID)
	}
	pendingQuestionsMu.Unlock()
}

func QuestionTool() *ToolDef {
	return &ToolDef{
		Name:        "question",
		Description: "Ask the user a question during execution. This allows you to gather preferences, clarify ambiguous instructions, get decisions on implementation choices, or offer choices about what direction to take. Answers are returned as arrays of labels.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"questions": map[string]interface{}{
					"type":        "array",
					"description": "Questions to ask",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"question": map[string]interface{}{
								"type":        "string",
								"description": "Complete question text",
							},
							"header": map[string]interface{}{
								"type":        "string",
								"description": "Very short label (max 30 chars)",
							},
							"options": map[string]interface{}{
								"type":        "array",
								"description": "Available choices",
								"items": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"label": map[string]interface{}{
											"type":        "string",
											"description": "Display text (1-5 words)",
										},
										"description": map[string]interface{}{
											"type":        "string",
											"description": "Explanation of choice",
										},
									},
									"required": []string{"label", "description"},
								},
							},
							"multiple": map[string]interface{}{
								"type":        "boolean",
								"description": "Allow selecting multiple choices",
							},
						},
						"required": []string{"question", "header", "options"},
					},
				},
			},
			"required": []string{"questions"},
		},
		Execute: func(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
			questionsRaw, ok := input["questions"]
			if !ok {
				return &ToolResult{Output: "Error: questions parameter required", IsError: true}, nil
			}

			questionsJSON, err := json.Marshal(questionsRaw)
			if err != nil {
				return &ToolResult{Output: "Error: invalid questions format", IsError: true}, nil
			}

			var questions []struct {
				Question string   `json:"question"`
				Header   string   `json:"header"`
				Options  []Option `json:"options"`
				Multiple bool     `json:"multiple"`
			}
			if err := json.Unmarshal(questionsJSON, &questions); err != nil {
				return &ToolResult{Output: "Error: invalid questions format: " + err.Error(), IsError: true}, nil
			}

			var results []string
			for _, q := range questions {
				pq := &PendingQuestion{
					ID:       fmt.Sprintf("%s-%s", tc.SessionID, q.Header),
					Question: q.Question,
					Header:   q.Header,
					Options:  q.Options,
					Multiple: q.Multiple,
					Custom:   true,
					Answered: make(chan struct{}),
				}

				pendingQuestionsMu.Lock()
				pendingQuestions[tc.SessionID] = pq
				pendingQuestionsMu.Unlock()

				// Wait for answer or context cancellation
				select {
				case <-pq.Answered:
					results = append(results, fmt.Sprintf("%s: %s", q.Header, strings.Join(pq.Answer, ", ")))
				case <-ctx.Done():
					pendingQuestionsMu.Lock()
					delete(pendingQuestions, tc.SessionID)
					pendingQuestionsMu.Unlock()
					return &ToolResult{Output: "Question cancelled by user", IsError: true}, nil
				}
			}

			return &ToolResult{
				Output: strings.Join(results, "\n"),
			}, nil
		},
	}
}
