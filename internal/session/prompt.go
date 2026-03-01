package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Dhanuzh/dcode/internal/agent"
	"github.com/Dhanuzh/dcode/internal/config"
	"github.com/Dhanuzh/dcode/internal/provider"
	"github.com/Dhanuzh/dcode/internal/tool"
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

// PermissionAskFn is a callback invoked when a tool requires interactive permission.
// It should block until the user responds. Returns true if allowed, false if denied.
type PermissionAskFn func(toolName, toolInput string) bool

// PromptEngine handles the conversation loop with the LLM
type PromptEngine struct {
	store      *Store
	provider   provider.Provider
	config     *config.Config
	agent      *agent.Agent
	registry   *tool.Registry
	onChunk    func(chunk StreamEvent)
	snapshot   *Snapshot
	onPermAsk  PermissionAskFn    // Called when PermAsk action fires
	onQuestion tool.QuestionAskFn // Called when the question tool needs user input
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
	// Diff data for realtime display in chat (populated on tool_end for edit/write tools)
	DiffData     *tool.DiffData   `json:"diff_data,omitempty"`
	DiffDataList []*tool.DiffData `json:"diff_data_list,omitempty"`
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

// OnPermissionAsk sets the callback invoked when a tool permission must be asked interactively.
func (pe *PromptEngine) OnPermissionAsk(fn PermissionAskFn) {
	pe.onPermAsk = fn
}

// OnQuestionAsk sets the callback invoked when the question tool needs user input.
func (pe *PromptEngine) OnQuestionAsk(fn tool.QuestionAskFn) {
	pe.onQuestion = fn
}

// SetModel overrides the model used for the next request.
// Allows live model switching within the same provider session.
func (pe *PromptEngine) SetModel(model string) {
	pe.config.Model = model
}

// SetAgent replaces the agent used for the next request.
func (pe *PromptEngine) SetAgent(ag *agent.Agent) {
	pe.agent = ag
}

func (pe *PromptEngine) emit(event StreamEvent) {
	if pe.onChunk != nil {
		pe.onChunk(event)
	}
}

// Run executes a prompt and enters the tool-use loop until completion
func (pe *PromptEngine) Run(ctx context.Context, sessionID, userMessage string) error {
	return pe.RunWithAttachments(ctx, sessionID, userMessage, nil)
}

// RunWithAttachments executes a prompt with optional inline image attachments.
func (pe *PromptEngine) RunWithAttachments(ctx context.Context, sessionID, userMessage string, images []ImageAttachment) error {
	// Build user message.
	// When images are present we store everything as Parts so buildLLMMessages
	// can emit properly-typed content blocks for each provider:
	//   Part{Type:"text"}  → text block (always first)
	//   Part{Type:"image"} → image block (base64 source)
	// The top-level Content field is kept for session display / fallback only.
	userMsg := Message{
		Role:      "user",
		Content:   userMessage, // kept for display; may be overridden below
		CreatedAt: time.Now(),
	}

	if len(images) > 0 {
		// Text part first, then image parts — providers expect this order
		if userMessage != "" {
			userMsg.Parts = append(userMsg.Parts, Part{Type: "text", Content: userMessage})
		}
		for _, img := range images {
			imgCopy := img
			userMsg.Parts = append(userMsg.Parts, Part{
				Type:  "image",
				Image: &imgCopy,
			})
		}
		// Content is now in Parts; clear the top-level field so buildLLMMessages
		// doesn't send it as a duplicate plain-string message.
		userMsg.Content = userMessage // keep for UI display only
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
		if pe.config.IsPruningEnabled() {
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
			// Classify the error for better handling
			classified := provider.ClassifyError(err, 0, "")

			// Check if retryable (using both classification and pattern matching)
			retryMsg := ""
			if classified != nil && classified.IsRetryable {
				retryMsg = classified.Message
			}
			if retryMsg == "" {
				retryMsg = IsRetryableError(err)
			}

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

			// Make the error user-friendly
			friendlyErr := provider.MakeUserFriendly(err, pe.provider.Name())
			pe.emit(StreamEvent{Type: "error", Content: friendlyErr.Error()})
			return friendlyErr
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
		if pe.config.IsAutoCompactionEnabled() && response.Usage.InputTokens > 0 {
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

		// If needs compaction, run the compaction agent inline
		if needsCompaction {
			needsCompaction = false
			pe.emit(StreamEvent{Type: "compaction", Content: "Context overflow detected, compacting..."})
			if err := pe.runCompaction(ctx, sessionID); err != nil {
				// Non-fatal: just warn and continue; the next request will still
				// include the pruned context from PruneToolOutputs.
				pe.emit(StreamEvent{Type: "error", Content: fmt.Sprintf("compaction failed: %v", err)})
			}
		}

		// If no tool use, we're done
		if !hasToolUse || response.StopReason == "end_turn" {
			// Auto-generate session title on first assistant response
			if step == 0 && pe.config.AutoTitle {
				go pe.generateTitle(sessionID, userMessage, extractText(response.Content))
			}
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
				Content:  toolStartDetail(part.ToolName, part.ToolInput),
			})

			// Track diff data for realtime display
			var realtimeDiffData *tool.DiffData
			var realtimeDiffDataList []*tool.DiffData

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
					desc := fmt.Sprintf("Doom loop: %s called %d+ times with identical args", part.ToolName, DoomLoopThreshold)
					allowed := pe.askPermission(part.ToolName, desc)
					if !allowed {
						blocked = true
						toolResults = append(toolResults, Part{
							Type:    "tool_result",
							ToolID:  part.ToolID,
							Content: "Permission denied: user rejected doom-loop continuation.",
							IsError: true,
						})
						continue
					}
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
			if permission == agent.PermAsk {
				// Ask user interactively; block until response
				inputDesc := fmt.Sprintf("%v", part.ToolInput)
				allowed := pe.askPermission(part.ToolName, inputDesc)
				if !allowed {
					toolResults = append(toolResults, Part{
						Type:    "tool_result",
						ToolID:  part.ToolID,
						Content: "Permission denied by user",
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
			}

			// Execute the tool
			tc := &tool.ToolContext{
				SessionID:  sessionID,
				WorkDir:    config.GetProjectDir(),
				Abort:      ctx,
				OnQuestion: pe.onQuestion,
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
				resultPart := Part{
					Type:     "tool_result",
					ToolID:   part.ToolID,
					ToolName: part.ToolName,
					Content:  result.Output,
					IsError:  result.IsError,
					Title:    result.Title,
				}

				// Attach diff data to metadata for TUI rendering (persisted in session)
				if result.DiffData != nil {
					// Cap at 10KB to prevent session bloat
					if len(result.DiffData.OldContent)+len(result.DiffData.NewContent) <= 10240 {
						resultPart.Metadata = map[string]interface{}{
							"diff_data": result.DiffData,
						}
						realtimeDiffData = result.DiffData
					}
				} else if len(result.DiffDataList) > 0 {
					// Filter to items under size cap
					var capped []*tool.DiffData
					totalSize := 0
					for _, d := range result.DiffDataList {
						size := len(d.OldContent) + len(d.NewContent)
						if totalSize+size <= 10240 {
							capped = append(capped, d)
							totalSize += size
						}
					}
					if len(capped) > 0 {
						resultPart.Metadata = map[string]interface{}{
							"diff_data_list": capped,
						}
						realtimeDiffDataList = capped
					}
				}

				toolResults = append(toolResults, resultPart)
			}

			// Update summary
			session, _ := pe.store.Get(sessionID)
			if session != nil && session.Summary != nil {
				session.Summary.ToolCalls++
			}

			// Emit tool_end with diff data for realtime display in chat
			pe.emit(StreamEvent{
				Type:         "tool_end",
				ToolID:       part.ToolID,
				ToolName:     part.ToolName,
				DiffData:     realtimeDiffData,
				DiffDataList: realtimeDiffDataList,
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

// MaxToolOutputChars is the maximum characters of a tool result sent to the LLM.
// Longer outputs are truncated to head + tail to save tokens.
const MaxToolOutputChars = 12000

// truncateToolOutput keeps the first and last portions of large tool outputs.
func truncateToolOutput(s string) string {
	if len(s) <= MaxToolOutputChars {
		return s
	}
	half := MaxToolOutputChars / 2
	return s[:half] + "\n\n... (truncated) ...\n\n" + s[len(s)-half:]
}

// buildLLMMessages converts session messages to LLM provider format
func (pe *PromptEngine) buildLLMMessages(messages []Message) []provider.Message {
	llmMessages := make([]provider.Message, 0, len(messages))

	// First pass: fix any missing ToolIDs in the session history
	for i, msg := range messages {
		if msg.Role != "assistant" {
			continue
		}
		for j, part := range msg.Parts {
			if part.Type == "tool_use" && part.ToolID == "" {
				newID := generateToolID()
				messages[i].Parts[j].ToolID = newID
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
					} else {
						content = truncateToolOutput(content)
					}
					blocks = append(blocks, provider.ContentBlock{
						Type:      "tool_result",
						ToolUseID: id,
						Content:   content,
						IsError:   part.IsError,
					})
				case "image":
					// Inline image attachment — pass directly to the model
					if part.Image != nil && part.Image.Data != "" {
						blocks = append(blocks, provider.ContentBlock{
							Type: "image",
							Source: &provider.ImageSource{
								Type:      "base64",
								MediaType: part.Image.MediaType,
								Data:      part.Image.Data,
							},
						})
					}
				case "reasoning":
					// Keep reasoning but cap at 500 chars to save tokens
					if part.Content != "" {
						reasoning := part.Content
						if len(reasoning) > 500 {
							reasoning = reasoning[:500] + "..."
						}
						blocks = append(blocks, provider.ContentBlock{
							Type: "text",
							Text: "<thinking>" + reasoning + "</thinking>",
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

// askPermission calls onPermAsk if set, otherwise defaults to allowing.
// Blocks the goroutine until the user responds.
func (pe *PromptEngine) askPermission(toolName, description string) bool {
	if pe.onPermAsk == nil {
		return true // default: allow when no handler is registered
	}
	return pe.onPermAsk(toolName, description)
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

// CompactSession runs the compaction agent for the given session (public API for TUI).
func (pe *PromptEngine) CompactSession(ctx context.Context, sessionID string) error {
	return pe.runCompaction(ctx, sessionID)
}

// generateTitle calls the title agent to generate a short session title and
// saves it to the session. Runs in a goroutine so it doesn't block the main flow.
func (pe *PromptEngine) generateTitle(sessionID, userMsg, assistantReply string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Only title the session if it doesn't already have a meaningful title
	session, err := pe.store.Get(sessionID)
	if err != nil {
		return
	}
	if session.Title != "" && session.Title != "New Session" && !strings.HasPrefix(session.Title, "Session ") {
		return
	}

	// Build a short prompt for the title agent
	prompt := fmt.Sprintf("User: %s\n\nAssistant: %s", userMsg, assistantReply)
	if len(prompt) > 1000 {
		prompt = prompt[:1000] + "..."
	}
	titlePrompt := "Summarise the following conversation in 4-6 words as a session title. Reply with ONLY the title, no punctuation:\n\n" + prompt

	req := &provider.MessageRequest{
		Model:       pe.config.GetSmallModel(),
		Messages:    []provider.Message{{Role: "user", Content: titlePrompt}},
		MaxTokens:   20,
		Temperature: 0.5,
		System:      "You generate ultra-concise session titles.",
	}

	resp, err := pe.provider.CreateMessage(ctx, req)
	if err != nil {
		return
	}
	title := strings.TrimSpace(extractText(resp.Content))
	// Sanitise: strip surrounding quotes, truncate
	title = strings.Trim(title, `"'`)
	if len(title) > 60 {
		title = title[:57] + "..."
	}
	if title != "" {
		_ = pe.store.UpdateTitle(sessionID, title)
	}
}

// runCompaction runs the compaction agent to summarise the conversation so far,
// then replaces the session's message history with a single summary message.
// This mirrors opencode's compaction flow.
func (pe *PromptEngine) runCompaction(ctx context.Context, sessionID string) error {
	session, err := pe.store.Get(sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	// Build messages to send to the compaction agent (include all conversation)
	compactionMessages := BuildCompactionMessages(session.Messages, nil)

	// Use a small/fast model for compaction
	compactionModel := pe.config.GetSmallModel()
	compactionProvider := pe.config.Provider

	// Use the same provider but with the compaction agent system prompt
	compactionAgentPrompt := "You are a helpful AI assistant tasked with summarizing conversations. Create a concise but complete summary of the conversation above that could be used to continue the conversation from scratch."

	req := &provider.MessageRequest{
		Model:       compactionModel,
		Messages:    pe.buildLLMMessages(compactionMessages),
		MaxTokens:   4096,
		Temperature: 0,
		System:      compactionAgentPrompt,
	}

	_ = compactionProvider
	response, err := pe.provider.CreateMessage(ctx, req)
	if err != nil {
		return fmt.Errorf("compaction LLM call: %w", err)
	}

	summary := extractText(response.Content)
	if summary == "" {
		return fmt.Errorf("compaction returned empty summary")
	}

	// Replace session messages with a summary marker + the summary
	summaryMsg := Message{
		Role:      "user",
		Content:   "[Context compacted. Previous conversation summary:]\n\n" + summary,
		IsSummary: true,
		CreatedAt: session.Messages[0].CreatedAt, // keep original timestamp
	}

	// Keep any patch/system messages from after the last user message
	// (they contain snapshot info). For simplicity, just replace all messages.
	if err := pe.store.ReplaceMessages(sessionID, []Message{summaryMsg}); err != nil {
		return fmt.Errorf("replace messages: %w", err)
	}

	pe.emit(StreamEvent{Type: "compaction", Content: "Context compacted successfully."})
	return nil
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

// toolStartDetail returns a short human-readable description of what a tool is doing,
// shown in the TUI next to the spinner during execution.
func toolStartDetail(toolName string, input map[string]interface{}) string {
	str := func(key string) string {
		if v, ok := input[key]; ok {
			s := fmt.Sprintf("%v", v)
			if len(s) > 60 {
				s = s[:57] + "..."
			}
			return s
		}
		return ""
	}
	switch toolName {
	case "read":
		if p := str("file_path"); p != "" {
			return p
		}
	case "write":
		if p := str("file_path"); p != "" {
			return p
		}
	case "edit", "patch":
		if p := str("file_path"); p != "" {
			return p
		}
	case "ls":
		if p := str("path"); p != "" {
			return p
		}
		if p := str("directory"); p != "" {
			return p
		}
	case "glob":
		if p := str("pattern"); p != "" {
			return p
		}
	case "grep":
		if p := str("pattern"); p != "" {
			q := str("path")
			if q != "" {
				return p + " in " + q
			}
			return p
		}
	case "bash":
		if c := str("command"); c != "" {
			return c
		}
	case "webfetch":
		if u := str("url"); u != "" {
			return u
		}
	case "task":
		if d := str("description"); d != "" {
			return d
		}
	}
	return fmt.Sprintf("Running %s...", toolName)
}
