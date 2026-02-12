package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/dcode/internal/agent"
	"github.com/yourusername/dcode/internal/config"
	"github.com/yourusername/dcode/internal/provider"
	"github.com/yourusername/dcode/internal/tool"
)

// PromptEngine handles the conversation loop with the LLM
type PromptEngine struct {
	store    *Store
	provider provider.Provider
	config   *config.Config
	agent    *agent.Agent
	registry *tool.Registry
	onChunk  func(chunk StreamEvent)
}

// StreamEvent represents a streaming event from the prompt engine
type StreamEvent struct {
	Type    string `json:"type"` // "text", "tool_start", "tool_end", "thinking", "error", "done"
	Content string `json:"content,omitempty"`
	ToolID  string `json:"tool_id,omitempty"`
	ToolName string `json:"tool_name,omitempty"`
}

// NewPromptEngine creates a new prompt engine
func NewPromptEngine(store *Store, prov provider.Provider, cfg *config.Config, ag *agent.Agent, registry *tool.Registry) *PromptEngine {
	return &PromptEngine{
		store:    store,
		provider: prov,
		config:   cfg,
		agent:    ag,
		registry: registry,
	}
}

// OnStream sets the streaming event callback
func (pe *PromptEngine) OnStream(callback func(StreamEvent)) {
	pe.onChunk = callback
}

func (pe *PromptEngine) emit(event StreamEvent) {
	if pe.onChunk != nil {
		pe.onChunk(event)
	}
}

// Run executes a prompt and enters the tool-use loop until completion
func (pe *PromptEngine) Run(ctx context.Context, sessionID, userMessage string) error {
	// Add user message
	userMsg := Message{
		Role:      "user",
		Content:   userMessage,
		CreatedAt: time.Now(),
	}
	if err := pe.store.AddMessage(sessionID, userMsg); err != nil {
		return fmt.Errorf("failed to add user message: %w", err)
	}

	pe.store.UpdateStatus(sessionID, "busy")
	defer pe.store.UpdateStatus(sessionID, "idle")

	// Get system prompt
	systemPrompt := agent.GetSystemPrompt(pe.agent.Name, pe.config)

	// Enter the prompt loop
	maxSteps := pe.agent.Steps
	if maxSteps <= 0 {
		maxSteps = 50
	}

	for step := 0; step < maxSteps; step++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Build messages for the LLM
		session, err := pe.store.Get(sessionID)
		if err != nil {
			return err
		}

		llmMessages := pe.buildLLMMessages(session.Messages)

		// Get available tools
		toolDefs := pe.registry.GetFiltered(pe.agent.Tools)
		providerTools := make([]provider.Tool, 0, len(toolDefs))
		for _, t := range toolDefs {
			providerTools = append(providerTools, provider.Tool{
				Name:        t.Name,
				Description: t.Description,
				InputSchema: t.Parameters,
			})
		}

		// Create LLM request
		model := pe.agent.Model
		if model == "" {
			model = pe.config.GetDefaultModel(pe.config.Provider)
		}

		req := &provider.MessageRequest{
			Model:       model,
			Messages:    llmMessages,
			MaxTokens:   pe.config.MaxTokens,
			Temperature: pe.agent.Temperature,
			System:      systemPrompt,
			Tools:       providerTools,
		}

		// Stream or create message
		var response *provider.MessageResponse

		if pe.config.Streaming && pe.onChunk != nil {
			response, err = pe.streamMessage(ctx, req)
		} else {
			response, err = pe.provider.CreateMessage(ctx, req)
		}

		if err != nil {
			pe.emit(StreamEvent{Type: "error", Content: err.Error()})
			return fmt.Errorf("LLM error: %w", err)
		}

		// Process response
		assistantParts := []Part{}
		hasToolUse := false

		for _, block := range response.Content {
			switch block.Type {
			case "text":
				if block.Text != "" {
					assistantParts = append(assistantParts, Part{
						Type:    "text",
						Content: block.Text,
					})
				}
			case "tool_use":
				hasToolUse = true
				assistantParts = append(assistantParts, Part{
					Type:      "tool_use",
					ToolID:    block.ID,
					ToolName:  block.Name,
					ToolInput: block.Input,
					Status:    "pending",
				})
			}
		}

		// Add assistant message
		assistantMsg := Message{
			Role:      "assistant",
			Content:   extractText(response.Content),
			Parts:     assistantParts,
			CreatedAt: time.Now(),
			TokensIn:  response.Usage.InputTokens,
			TokensOut: response.Usage.OutputTokens,
		}
		if err := pe.store.AddMessage(sessionID, assistantMsg); err != nil {
			return err
		}

		// If no tool use, we're done
		if !hasToolUse || response.StopReason == "end_turn" {
			pe.emit(StreamEvent{Type: "done"})
			return nil
		}

		// Execute tool calls
		toolResults := []Part{}
		for _, part := range assistantParts {
			if part.Type != "tool_use" {
				continue
			}

			pe.emit(StreamEvent{
				Type:     "tool_start",
				ToolID:   part.ToolID,
				ToolName: part.ToolName,
				Content:  fmt.Sprintf("Running %s...", part.ToolName),
			})

			// Check permissions
			permission := pe.checkPermission(part.ToolName)
			if permission == "deny" {
				toolResults = append(toolResults, Part{
					Type:    "tool_result",
					ToolID:  part.ToolID,
					Content: "Permission denied: this tool is not allowed for the current agent",
					IsError: true,
				})
				pe.emit(StreamEvent{
					Type:     "tool_end",
					ToolID:   part.ToolID,
					ToolName: part.ToolName,
					Content:  "Permission denied",
				})
				continue
			}

			// Execute the tool
			tc := &tool.ToolContext{
				SessionID: sessionID,
				WorkDir:   config.GetProjectDir(),
				Abort:     ctx,
			}

			result, err := pe.registry.Execute(ctx, tc, part.ToolName, part.ToolInput)
			if err != nil {
				toolResults = append(toolResults, Part{
					Type:    "tool_result",
					ToolID:  part.ToolID,
					Content: fmt.Sprintf("Tool error: %v", err),
					IsError: true,
				})
			} else {
				toolResults = append(toolResults, Part{
					Type:    "tool_result",
					ToolID:  part.ToolID,
					Content: result.Output,
					IsError: result.IsError,
				})
			}

			// Update summary
			session, _ := pe.store.Get(sessionID)
			if session != nil && session.Summary != nil {
				session.Summary.ToolCalls++
			}

			pe.emit(StreamEvent{
				Type:     "tool_end",
				ToolID:   part.ToolID,
				ToolName: part.ToolName,
			})
		}

		// Add tool results as user message
		toolMsg := Message{
			Role:      "user",
			Parts:     toolResults,
			CreatedAt: time.Now(),
		}
		if err := pe.store.AddMessage(sessionID, toolMsg); err != nil {
			return err
		}
	}

	return fmt.Errorf("max steps (%d) reached", maxSteps)
}

// streamMessage handles streaming response
func (pe *PromptEngine) streamMessage(ctx context.Context, req *provider.MessageRequest) (*provider.MessageResponse, error) {
	var response provider.MessageResponse
	response.Content = []provider.ContentBlock{}

	currentBlockIdx := -1
	var currentText strings.Builder
	var currentToolInput strings.Builder

	err := pe.provider.StreamMessage(ctx, req, func(chunk *provider.StreamChunk) error {
		switch chunk.Type {
		case "message_start":
			if chunk.Message != nil {
				response.ID = chunk.Message.ID
				response.Model = chunk.Message.Model
				response.Role = chunk.Message.Role
				response.Usage = chunk.Message.Usage
			}

		case "content_block_start":
			currentBlockIdx++
			if chunk.ContentBlock != nil {
				response.Content = append(response.Content, *chunk.ContentBlock)
				currentText.Reset()
				currentToolInput.Reset()
			}

		case "content_block_delta":
			if chunk.Delta != nil {
				switch chunk.Delta.Type {
				case "text_delta":
					currentText.WriteString(chunk.Delta.Text)
					pe.emit(StreamEvent{
						Type:    "text",
						Content: chunk.Delta.Text,
					})
				case "input_json_delta":
					currentToolInput.WriteString(chunk.Delta.PartialJSON)
				}
				if chunk.Delta.Reasoning != "" {
					pe.emit(StreamEvent{
						Type:    "thinking",
						Content: chunk.Delta.Reasoning,
					})
				}
			}

		case "content_block_stop":
			if currentBlockIdx >= 0 && currentBlockIdx < len(response.Content) {
				block := &response.Content[currentBlockIdx]
				if block.Type == "text" {
					block.Text = currentText.String()
				} else if block.Type == "tool_use" {
					inputStr := currentToolInput.String()
					if inputStr != "" {
						var input map[string]interface{}
						json.Unmarshal([]byte(inputStr), &input)
						block.Input = input
					}
				}
			}

		case "message_delta":
			if chunk.Delta != nil {
				// Handle stop reason from delta
			}

		case "message_stop":
			response.StopReason = "end_turn"
			// Check if we have tool use
			for _, block := range response.Content {
				if block.Type == "tool_use" {
					response.StopReason = "tool_use"
					break
				}
			}
		}
		return nil
	})

	return &response, err
}

// buildLLMMessages converts session messages to LLM provider format
func (pe *PromptEngine) buildLLMMessages(messages []Message) []provider.Message {
	llmMessages := make([]provider.Message, 0, len(messages))

	for _, msg := range messages {
		if len(msg.Parts) == 0 {
			// Simple text message
			llmMessages = append(llmMessages, provider.Message{
				Role:    msg.Role,
				Content: msg.Content,
			})
		} else {
			// Message with parts
			blocks := make([]provider.ContentBlock, 0, len(msg.Parts))
			for _, part := range msg.Parts {
				switch part.Type {
				case "text":
					blocks = append(blocks, provider.ContentBlock{
						Type: "text",
						Text: part.Content,
					})
				case "tool_use":
					blocks = append(blocks, provider.ContentBlock{
						Type:  "tool_use",
						ID:    part.ToolID,
						Name:  part.ToolName,
						Input: part.ToolInput,
					})
				case "tool_result":
					content := part.Content
					blocks = append(blocks, provider.ContentBlock{
						Type:      "tool_result",
						ToolUseID: part.ToolID,
						Content:   content,
						IsError:   part.IsError,
					})
				}
			}
			llmMessages = append(llmMessages, provider.Message{
				Role:    msg.Role,
				Content: blocks,
			})
		}
	}

	return llmMessages
}

// checkPermission checks if a tool is allowed for the current agent
func (pe *PromptEngine) checkPermission(toolName string) string {
	if pe.agent.Permission == nil {
		return "allow"
	}
	if perm, ok := pe.agent.Permission[toolName]; ok {
		return perm
	}
	// Default allow for unlisted tools
	return "allow"
}

func extractText(blocks []provider.ContentBlock) string {
	texts := []string{}
	for _, block := range blocks {
		if block.Type == "text" && block.Text != "" {
			texts = append(texts, block.Text)
		}
	}
	return strings.Join(texts, "\n")
}
