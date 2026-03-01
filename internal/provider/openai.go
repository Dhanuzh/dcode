package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAIProvider implements the Provider interface for OpenAI
type OpenAIProvider struct {
	client *openai.Client
	apiKey string
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		client: openai.NewClient(apiKey),
		apiKey: apiKey,
	}
}

func (p *OpenAIProvider) Name() string { return "openai" }

func (p *OpenAIProvider) Models() []string {
	return []string{
		// GPT-5.x series
		"gpt-5.3-codex",
		"gpt-5.3-codex-spark",
		"gpt-5.2-codex",
		"gpt-5.2",
		"gpt-5.2-pro",
		"gpt-5.2-chat-latest",
		"gpt-5.1-codex-max",
		"gpt-5.1-codex",
		"gpt-5.1-codex-mini",
		"gpt-5.1",
		"gpt-5.1-chat-latest",
		"gpt-5-codex",
		"gpt-5",
		"gpt-5-pro",
		"gpt-5-mini",
		"gpt-5-nano",
		"gpt-5-chat-latest",
		"codex-mini-latest",
		// GPT-4.x series
		"gpt-4.1",
		"gpt-4.1-mini",
		"gpt-4.1-nano",
		"gpt-4o",
		"gpt-4o-mini",
		"gpt-4o-2024-11-20",
		"gpt-4o-2024-08-06",
		"gpt-4o-2024-05-13",
		"gpt-4-turbo",
		"gpt-4",
		"gpt-3.5-turbo",
		// o-series reasoning
		"o3",
		"o3-mini",
		"o3-pro",
		"o3-deep-research",
		"o4-mini",
		"o4-mini-deep-research",
		"o1",
		"o1-pro",
		"o1-preview",
		"o1-mini",
		// Embeddings
		"text-embedding-3-large",
		"text-embedding-3-small",
		"text-embedding-ada-002",
	}
}

// CreateMessage sends a message to OpenAI and returns the response
func (p *OpenAIProvider) CreateMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error) {
	messages := p.convertMessages(req)

	chatReq := openai.ChatCompletionRequest{
		Model:    req.Model,
		Messages: messages,
	}

	if req.MaxTokens > 0 {
		chatReq.MaxTokens = req.MaxTokens
	}
	if req.Temperature != 0 {
		chatReq.Temperature = float32(req.Temperature)
	}
	if len(req.Tools) > 0 {
		chatReq.Tools = p.convertTools(req.Tools)
	}

	resp, err := p.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("openai API error: %w", err)
	}

	return p.convertResponse(&resp), nil
}

// StreamMessage streams a response from OpenAI
func (p *OpenAIProvider) StreamMessage(ctx context.Context, req *MessageRequest, callback func(*StreamChunk) error) error {
	messages := p.convertMessages(req)

	chatReq := openai.ChatCompletionRequest{
		Model:    req.Model,
		Messages: messages,
		Stream:   true,
	}

	if req.MaxTokens > 0 {
		chatReq.MaxTokens = req.MaxTokens
	}
	if req.Temperature != 0 {
		chatReq.Temperature = float32(req.Temperature)
	}
	if len(req.Tools) > 0 {
		chatReq.Tools = p.convertTools(req.Tools)
	}

	stream, err := p.client.CreateChatCompletionStream(ctx, chatReq)
	if err != nil {
		return fmt.Errorf("openai stream error: %w", err)
	}
	defer stream.Close()

	var accumulatedToolCalls map[int]*openai.ToolCall

	for {
		response, err := stream.Recv()
		if err == io.EOF {
			// Emit any accumulated tool calls
			if accumulatedToolCalls != nil {
				for _, tc := range accumulatedToolCalls {
					id := tc.ID
					if id == "" {
						id = generateToolCallID()
					}
					if tc.Function.Name == "" {
						continue
					}
					var input map[string]interface{}
					json.Unmarshal([]byte(tc.Function.Arguments), &input)
					callback(&StreamChunk{
						Type: "content_block_start",
						ContentBlock: &ContentBlock{
							Type:  "tool_use",
							ID:    id,
							Name:  tc.Function.Name,
							Input: input,
						},
					})
				}
			}
			callback(&StreamChunk{Type: "message_stop"})
			break
		}
		if err != nil {
			return fmt.Errorf("stream receive error: %w", err)
		}

		if len(response.Choices) > 0 {
			delta := response.Choices[0].Delta

			if delta.Content != "" {
				callback(&StreamChunk{
					Type:  "content_block_delta",
					Index: 0,
					Delta: &Delta{
						Type: "text_delta",
						Text: delta.Content,
					},
				})
			}

			// Accumulate tool calls
			for _, tc := range delta.ToolCalls {
				if accumulatedToolCalls == nil {
					accumulatedToolCalls = make(map[int]*openai.ToolCall)
				}
				idx := tc.Index
				if idx == nil {
					zero := 0
					idx = &zero
				}
				if existing, ok := accumulatedToolCalls[*idx]; ok {
					existing.Function.Arguments += tc.Function.Arguments
				} else {
					call := tc
					accumulatedToolCalls[*idx] = &call
				}
			}

			if response.Choices[0].FinishReason == "stop" {
				callback(&StreamChunk{Type: "message_stop"})
				break
			}
		}
	}

	return nil
}

func (p *OpenAIProvider) convertMessages(req *MessageRequest) []openai.ChatCompletionMessage {
	messages := make([]openai.ChatCompletionMessage, 0, len(req.Messages)+1)

	if req.System != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: req.System,
		})
	}

	for _, msg := range req.Messages {
		oaiMsg := openai.ChatCompletionMessage{
			Role: msg.Role,
		}

		switch content := msg.Content.(type) {
		case string:
			oaiMsg.Content = content
		case []ContentBlock:
			textParts := []string{}
			var multiContent []openai.ChatMessagePart
			var toolCalls []openai.ToolCall

			for _, block := range content {
				switch block.Type {
				case "text":
					textParts = append(textParts, block.Text)
					multiContent = append(multiContent, openai.ChatMessagePart{
						Type: openai.ChatMessagePartTypeText,
						Text: block.Text,
					})
				case "image":
					// OpenAI vision: image_url content part
					if block.Source != nil {
						var imgURL openai.ChatMessageImageURL
						if block.Source.Type == "base64" && block.Source.Data != "" {
							imgURL.URL = "data:" + block.Source.MediaType + ";base64," + block.Source.Data
						} else if block.Source.URL != "" {
							imgURL.URL = block.Source.URL
						}
						if imgURL.URL != "" {
							imgURL.Detail = openai.ImageURLDetailAuto
							multiContent = append(multiContent, openai.ChatMessagePart{
								Type:     openai.ChatMessagePartTypeImageURL,
								ImageURL: &imgURL,
							})
						}
					}
				case "tool_use":
					// Generate ID if missing
					id := block.ID
					if id == "" {
						id = generateToolCallID()
					}
					// Skip tool calls without a function name
					if block.Name == "" {
						continue
					}
					inputJSON, _ := json.Marshal(block.Input)
					if inputJSON == nil || string(inputJSON) == "null" {
						inputJSON = []byte("{}")
					}
					toolCalls = append(toolCalls, openai.ToolCall{
						ID:   id,
						Type: openai.ToolTypeFunction,
						Function: openai.FunctionCall{
							Name:      block.Name,
							Arguments: string(inputJSON),
						},
					})
				case "tool_result":
					// Skip tool results with missing tool_call_id
					if block.ToolUseID == "" {
						continue
					}
					resultContent := ""
					switch v := block.Content.(type) {
					case string:
						resultContent = v
					default:
						resultJSON, _ := json.Marshal(v)
						resultContent = string(resultJSON)
					}
					messages = append(messages, openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						Content:    resultContent,
						ToolCallID: block.ToolUseID,
					})
					continue
				}
			}

			// Use multi-part content when there are images; plain string otherwise
			hasImages := false
			for _, p := range multiContent {
				if p.Type == openai.ChatMessagePartTypeImageURL {
					hasImages = true
					break
				}
			}
			if hasImages {
				oaiMsg.MultiContent = multiContent
			} else if len(textParts) > 0 {
				oaiMsg.Content = ""
				for _, part := range textParts {
					oaiMsg.Content += part
				}
			}
			if len(toolCalls) > 0 {
				oaiMsg.ToolCalls = toolCalls
			}
		}

		messages = append(messages, oaiMsg)
	}

	return messages
}

func (p *OpenAIProvider) convertTools(tools []Tool) []openai.Tool {
	result := make([]openai.Tool, len(tools))
	for i, tool := range tools {
		paramsJSON, _ := json.Marshal(tool.InputSchema)
		var params map[string]interface{}
		json.Unmarshal(paramsJSON, &params)

		result[i] = openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  params,
			},
		}
	}
	return result
}

func (p *OpenAIProvider) convertResponse(resp *openai.ChatCompletionResponse) *MessageResponse {
	if len(resp.Choices) == 0 {
		return &MessageResponse{
			ID:    resp.ID,
			Model: resp.Model,
			Role:  "assistant",
		}
	}

	choice := resp.Choices[0]
	content := []ContentBlock{}

	if choice.Message.Content != "" {
		content = append(content, ContentBlock{
			Type: "text",
			Text: choice.Message.Content,
		})
	}

	for _, tc := range choice.Message.ToolCalls {
		id := tc.ID
		if id == "" {
			id = generateToolCallID()
		}
		if tc.Function.Name == "" {
			continue
		}
		var input map[string]interface{}
		json.Unmarshal([]byte(tc.Function.Arguments), &input)
		content = append(content, ContentBlock{
			Type:  "tool_use",
			ID:    id,
			Name:  tc.Function.Name,
			Input: input,
		})
	}

	stopReason := string(choice.FinishReason)
	if stopReason == "stop" {
		stopReason = "end_turn"
	} else if stopReason == "tool_calls" {
		stopReason = "tool_use"
	}

	return &MessageResponse{
		ID:         resp.ID,
		Model:      resp.Model,
		Role:       "assistant",
		Content:    content,
		StopReason: stopReason,
		Usage: Usage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
	}
}
