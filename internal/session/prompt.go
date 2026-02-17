package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/dcode/internal/agent"
	"github.com/yourusername/dcode/internal/config"
	"github.com/yourusername/dcode/internal/provider"
	"github.com/yourusername/dcode/internal/tool"
)

// generateToolID generates a unique tool call ID for missing IDs in session history
func generateToolID() string {
	b := make([]byte, 12)
	rand.Read(b)
	return "call_" + hex.EncodeToString(b)
}

// DoomLoopThreshold is the number of identical consecutive tool calls before triggering
const DoomLoopThreshold = 3

// MaxRetryAttempts is the maximum number of retry attempts
const MaxRetryAttempts = 10

// PromptEngine handles the conversation loop with the LLM
type PromptEngine struct {
	store    *Store
	provider provider.Provider
	config   *config.Config
	agent    *agent.Agent
	registry *tool.Registry
	onChunk  func(chunk StreamEvent)
	snapshot *Snapshot
}

// StreamEvent represents a streaming event from the prompt engine
type StreamEvent struct {
	Type     string `json:"type"` // "text", "tool_start", "tool_end", "thinking", "error", "done", "retry", "compaction", "step_start", "step_end"
	Content  string `json:"content,omitempty"`
	ToolID   string `json:"tool_id,omitempty"`
	ToolName string `json:"tool_name,omitempty"`
	// Retry info
	Attempt int   `json:"attempt,omitempty"`
	NextAt  int64 `json:"next_at,omitempty"`
	// Step info
	Tokens *StepTokens `json:"tokens,omitempty"`
	Cost   float64     `json:"cost,omitempty"`
}

// ProcessResult represents the outcome of processing
type ProcessResult string

const (
	ProcessContinue ProcessResult = "continue"
	ProcessStop     ProcessResult = "stop"
	ProcessCompact  ProcessResult = "compact"
)

// NewPromptEngine creates a new prompt engine
func NewPromptEngine(store *Store, prov provider.Provider, cfg *config.Config, ag *agent.Agent, registry *tool.Registry) *PromptEngine {
	pe := &PromptEngine{
		store:    store,
		provider: prov,
		config:   cfg,
		agent:    ag,
		registry: registry,
	}

	// Initialize snapshot if enabled
	if cfg.Snapshot {
		dataDir := config.GetConfigDir()
		workDir := config.GetProjectDir()
		pe.snapshot = NewSnapshot(dataDir, workDir)
	}

	return pe
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
	pe.store.statusMgr.SetBusy(sessionID)
	defer func() {
		pe.store.UpdateStatus(sessionID, "idle")
		pe.store.statusMgr.SetIdle(sessionID)
	}()

	// Get system prompt
	systemPrompt := agent.GetSystemPrompt(pe.agent.Name, pe.config)

	// Enter the prompt loop
	maxSteps := pe.agent.Steps
	if maxSteps <= 0 {
		maxSteps = 50
	}

	// Track doom loop detection
	var recentToolCalls []toolCallRecord
	retryAttempt := 0
	needsCompaction := false

	for step := 0; step < maxSteps; step++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Take snapshot before each step
		var stepSnapshot string
		if pe.snapshot != nil {
			pe.emit(StreamEvent{Type: "step_start"})
			hash, err := pe.snapshot.Track()
			if err == nil {
				stepSnapshot = hash
			}
		}

		// Build messages for the LLM
		session, err := pe.store.Get(sessionID)
		if err != nil {
			return err
		}

		// Apply pruning before building LLM messages
		prunedMessages := session.Messages
		if pe.config.Compaction {
			prunedMessages = PruneToolOutputs(session.Messages, true)
		}

		llmMessages := pe.buildLLMMessages(prunedMessages)

		// Get available tools
		toolDefs := pe.registry.GetFiltered(pe.agent.Tools)

		// Filter out disabled tools based on permissions
		allToolNames := make([]string, 0, len(toolDefs))
		for _, t := range toolDefs {
			allToolNames = append(allToolNames, t.Name)
		}
		disabledTools := agent.DisabledTools(allToolNames, pe.agent.Permission)

		providerTools := make([]provider.Tool, 0, len(toolDefs))
		for _, t := range toolDefs {
			if disabledTools[t.Name] {
				continue
			}
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
			// Check if retryable
			retryMsg := IsRetryableError(err)
			if retryMsg != "" && retryAttempt < MaxRetryAttempts {
				retryAttempt++
				delay := ComputeRetryDelay(retryAttempt, nil)

				pe.store.statusMgr.SetRetry(sessionID, retryAttempt, retryMsg, delay)
				pe.emit(StreamEvent{
					Type:    "retry",
					Content: retryMsg,
					Attempt: retryAttempt,
					NextAt:  time.Now().Add(delay).UnixMilli(),
				})

				// Create abort channel from context
				abort := make(chan struct{})
				go func() {
					<-ctx.Done()
					close(abort)
				}()

				if err := SleepWithAbort(delay, abort); err != nil {
					return ctx.Err()
				}

				pe.store.statusMgr.SetBusy(sessionID)
				continue // retry
			}

			pe.emit(StreamEvent{Type: "error", Content: err.Error()})
			return fmt.Errorf("LLM error: %w", err)
		}

		// Reset retry attempt on success
		retryAttempt = 0

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

		// Check for token overflow (triggers compaction)
		if pe.config.Compaction && response.Usage.InputTokens > 0 {
			modelInfo := pe.config.GetModelInfo(pe.config.Provider)
			if IsOverflow(
				response.Usage.InputTokens,
				0, // cache read
				response.Usage.OutputTokens,
				modelInfo.ContextWindow,
				modelInfo.MaxOutput,
			) {
				needsCompaction = true
			}
		}

		// Calculate cost
		cost := calculateCost(response.Usage.InputTokens, response.Usage.OutputTokens, pe.config.Provider)

		// Add assistant message
		assistantMsg := Message{
			Role:         "assistant",
			Content:      extractText(response.Content),
			Parts:        assistantParts,
			CreatedAt:    time.Now(),
			TokensIn:     response.Usage.InputTokens,
			TokensOut:    response.Usage.OutputTokens,
			Cost:         cost,
			AgentName:    pe.agent.Name,
			FinishReason: response.StopReason,
		}
		if err := pe.store.AddMessage(sessionID, assistantMsg); err != nil {
			return err
		}

		// Track step snapshot
		if pe.snapshot != nil && stepSnapshot != "" {
			endHash, err := pe.snapshot.Track()
			if err == nil {
				patchFiles, _ := pe.snapshot.Patch(stepSnapshot)
				if len(patchFiles) > 0 {
					// Record the patch
					pe.store.AddMessage(sessionID, Message{
						Role: "system",
						Parts: []Part{{
							Type:       "patch",
							PatchHash:  stepSnapshot,
							PatchFiles: patchFiles,
							Snapshot:   endHash,
						}},
						CreatedAt: time.Now(),
					})
				}
			}

			pe.emit(StreamEvent{
				Type: "step_end",
				Tokens: &StepTokens{
					Input:  response.Usage.InputTokens,
					Output: response.Usage.OutputTokens,
				},
				Cost: cost,
			})
		}

		// If needs compaction, trigger it
		if needsCompaction {
			pe.emit(StreamEvent{Type: "compaction", Content: "Context overflow detected, compacting..."})
			// The caller should handle compaction
			return nil
		}

		// If no tool use, we're done
		if !hasToolUse || response.StopReason == "end_turn" {
			pe.emit(StreamEvent{Type: "done"})
			return nil
		}

		// Execute tool calls
		toolResults := []Part{}
		blocked := false

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

			// Doom loop detection
			if isDoomLoop(recentToolCalls, part.ToolName, part.ToolInput) {
				pe.emit(StreamEvent{
					Type:    "error",
					Content: fmt.Sprintf("Doom loop detected: %s called %d times with identical input", part.ToolName, DoomLoopThreshold),
				})

				// Check doom_loop permission
				rule := agent.EvaluatePermission("doom_loop", part.ToolName, pe.agent.Permission)
				if rule.Action == agent.PermDeny {
					blocked = true
					toolResults = append(toolResults, Part{
						Type:    "tool_result",
						ToolID:  part.ToolID,
						Content: fmt.Sprintf("Doom loop detected: %s has been called %d times with identical arguments. The tool call was blocked.", part.ToolName, DoomLoopThreshold),
						IsError: true,
					})
					continue
				}
				if rule.Action == agent.PermAsk {
					// In the Go implementation, we'll just warn and continue
					// The TUI would show a permission prompt here
					pe.emit(StreamEvent{
						Type:    "error",
						Content: "Warning: Possible doom loop detected. Continuing anyway.",
					})
				}

				// Reset doom loop tracker
				recentToolCalls = nil
			}

			// Track tool call for doom loop detection
			recentToolCalls = append(recentToolCalls, toolCallRecord{
				Name:  part.ToolName,
				Input: part.ToolInput,
			})

			// Check permissions using pattern-based rules
			permission := pe.checkPermissionRules(part.ToolName, part.ToolInput)
			if permission == agent.PermDeny {
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
				blocked = true
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
					Type:     "tool_result",
					ToolID:   part.ToolID,
					ToolName: part.ToolName,
					Content:  result.Output,
					IsError:  result.IsError,
					Title:    result.Title,
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

		// If blocked by permission denial, stop
		if blocked {
			pe.emit(StreamEvent{Type: "done"})
			return nil
		}
	}

	return fmt.Errorf("max steps (%d) reached", maxSteps)
}

// toolCallRecord tracks a tool call for doom loop detection
type toolCallRecord struct {
	Name  string
	Input map[string]interface{}
}

// isDoomLoop checks if the last N tool calls are identical
func isDoomLoop(recent []toolCallRecord, name string, input map[string]interface{}) bool {
	if len(recent) < DoomLoopThreshold-1 {
		return false
	}

	// Check if the last (threshold-1) calls match
	inputJSON, _ := json.Marshal(input)
	inputStr := string(inputJSON)

	matchCount := 0
	for i := len(recent) - 1; i >= 0 && matchCount < DoomLoopThreshold-1; i-- {
		r := recent[i]
		if r.Name != name {
			break
		}
		rInputJSON, _ := json.Marshal(r.Input)
		if string(rInputJSON) != inputStr {
			break
		}
		matchCount++
	}

	return matchCount >= DoomLoopThreshold-1
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

	// First pass: fix any missing ToolIDs in the session history
	// and build a map of valid tool call IDs for pairing validation
	toolIDMap := make(map[string]string) // maps original (possibly empty) -> generated ID
	for i, msg := range messages {
		if msg.Role != "assistant" {
			continue
		}
		for j, part := range msg.Parts {
			if part.Type == "tool_use" && part.ToolID == "" {
				newID := generateToolID()
				messages[i].Parts[j].ToolID = newID
				toolIDMap[""] = newID // track that we generated one
			}
		}
	}

	for _, msg := range messages {
		// Skip system messages (patches, etc.)
		if msg.Role == "system" {
			continue
		}

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
					id := part.ToolID
					if id == "" {
						id = generateToolID()
					}
					// Skip tool_use blocks without a tool name
					if part.ToolName == "" {
						continue
					}
					blocks = append(blocks, provider.ContentBlock{
						Type:  "tool_use",
						ID:    id,
						Name:  part.ToolName,
						Input: part.ToolInput,
					})
				case "tool_result":
					id := part.ToolID
					if id == "" {
						continue // Skip orphaned tool results with no ID
					}
					content := part.Content
					if part.IsCompacted {
						content = "[compacted]"
					}
					blocks = append(blocks, provider.ContentBlock{
						Type:      "tool_result",
						ToolUseID: id,
						Content:   content,
						IsError:   part.IsError,
					})
				case "reasoning":
					// Include reasoning in text form
					if part.Content != "" {
						blocks = append(blocks, provider.ContentBlock{
							Type: "text",
							Text: "<thinking>" + part.Content + "</thinking>",
						})
					}
				}
			}
			if len(blocks) > 0 {
				llmMessages = append(llmMessages, provider.Message{
					Role:    msg.Role,
					Content: blocks,
				})
			}
		}
	}

	return llmMessages
}

// checkPermissionRules checks permissions using the pattern-based rule system
func (pe *PromptEngine) checkPermissionRules(toolName string, toolInput map[string]interface{}) agent.PermissionAction {
	if pe.agent.Permission == nil {
		return agent.PermAllow
	}

	// Determine the permission name (edit tools all use "edit" permission)
	permission := toolName
	for _, editTool := range agent.EditTools {
		if toolName == editTool {
			permission = "edit"
			break
		}
	}

	// Extract the pattern (file path) from tool input if available
	pattern := "*"
	if filePath, ok := toolInput["filePath"].(string); ok {
		pattern = filePath
	} else if filePath, ok := toolInput["file_path"].(string); ok {
		pattern = filePath
	} else if path, ok := toolInput["path"].(string); ok {
		pattern = path
	} else if command, ok := toolInput["command"].(string); ok && toolName == "bash" {
		pattern = command
	}

	// Check for external directory access
	workDir := config.GetProjectDir()
	if pattern != "*" && !strings.HasPrefix(pattern, workDir) && pattern != "" && pattern[0] == '/' {
		extRule := agent.EvaluatePermission("external_directory", pattern, pe.agent.Permission)
		if extRule.Action == agent.PermDeny {
			return agent.PermDeny
		}
	}

	rule := agent.EvaluatePermission(permission, pattern, pe.agent.Permission)
	return rule.Action
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

// calculateCost estimates the cost of a request based on token usage
// Uses approximate pricing per million tokens
func calculateCost(inputTokens, outputTokens int, providerName string) float64 {
	// Approximate pricing per million tokens (input, output)
	type pricing struct {
		input  float64
		output float64
	}

	prices := map[string]pricing{
		"anthropic":  {3.0, 15.0},  // Claude Sonnet 4
		"openai":     {2.0, 8.0},   // GPT-4.1
		"google":     {0.15, 0.60}, // Gemini 2.5 Flash
		"copilot":    {0.0, 0.0},   // Free with GitHub
		"groq":       {0.59, 0.79}, // Llama 3.3 70B
		"openrouter": {3.0, 15.0},  // Varies
		"deepseek":   {0.14, 0.28}, // DeepSeek V3
		"mistral":    {2.0, 6.0},   // Mistral Large
		"xai":        {2.0, 10.0},  // Grok
	}

	p, ok := prices[providerName]
	if !ok {
		p = pricing{3.0, 15.0} // Default to Anthropic pricing
	}

	inputCost := float64(inputTokens) / 1_000_000.0 * p.input
	outputCost := float64(outputTokens) / 1_000_000.0 * p.output
	return inputCost + outputCost
}
