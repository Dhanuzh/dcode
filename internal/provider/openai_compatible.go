package provider

import (
	"context"
	"encoding/json"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAICompatibleProvider works with any OpenAI-compatible API
type OpenAICompatibleProvider struct {
	name   string
	client *openai.Client
	apiKey string
	models []string
}

// NewOpenAICompatibleProvider creates a new OpenAI-compatible provider
func NewOpenAICompatibleProvider(name, apiKey, baseURL string) *OpenAICompatibleProvider {
	config := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		config.BaseURL = baseURL
	}
	client := openai.NewClientWithConfig(config)

	return &OpenAICompatibleProvider{
		name:   name,
		client: client,
		apiKey: apiKey,
	}
}

func (p *OpenAICompatibleProvider) Name() string { return p.name }

func (p *OpenAICompatibleProvider) Models() []string {
	return p.models
}

func (p *OpenAICompatibleProvider) CreateMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error) {
	messages := convertToOpenAIMessages(req)

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
		chatReq.Tools = convertToOpenAITools(req.Tools)
	}

	resp, err := p.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("%s API error: %w", p.name, err)
	}

	return convertFromOpenAIResponse(&resp), nil
}

func (p *OpenAICompatibleProvider) StreamMessage(ctx context.Context, req *MessageRequest, callback func(*StreamChunk) error) error {
	messages := convertToOpenAIMessages(req)

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
		chatReq.Tools = convertToOpenAITools(req.Tools)
	}

	stream, err := p.client.CreateChatCompletionStream(ctx, chatReq)
	if err != nil {
		return fmt.Errorf("%s stream error: %w", p.name, err)
	}
	defer stream.Close()

	var currentToolCall *openai.ToolCall
	var toolCallArgs string

	for {
		response, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("stream receive error: %w", err)
		}

		if len(response.Choices) == 0 {
			continue
		}

		delta := response.Choices[0].Delta

		// Handle text delta
		if delta.Content != "" {
			chunk := &StreamChunk{
				Type:  "content_block_delta",
				Index: 0,
				Delta: &Delta{
					Type: "text_delta",
					Text: delta.Content,
				},
			}
			if err := callback(chunk); err != nil {
				return err
			}
		}

		// Handle tool calls
		if len(delta.ToolCalls) > 0 {
			for _, tc := range delta.ToolCalls {
				if tc.ID != "" {
					if currentToolCall != nil {
						// Finish previous tool call
						id := currentToolCall.ID
						if id == "" {
							id = generateToolCallID()
						}
						if currentToolCall.Function.Name != "" {
							input := make(map[string]interface{})
							if toolCallArgs != "" {
								json.Unmarshal([]byte(toolCallArgs), &input)
							}

							chunk := &StreamChunk{
								Type:  "content_block_start",
								Index: 0,
								ContentBlock: &ContentBlock{
									Type:  "tool_use",
									ID:    id,
									Name:  currentToolCall.Function.Name,
									Input: input,
								},
							}
							if err := callback(chunk); err != nil {
								return err
							}
						}
					}

					currentToolCall = &tc
					toolCallArgs = ""
				}

				if tc.Function.Arguments != "" {
					toolCallArgs += tc.Function.Arguments
				}
			}
		}
	}

	// Finish last tool call if any
	if currentToolCall != nil && currentToolCall.Function.Name != "" {
		id := currentToolCall.ID
		if id == "" {
			id = generateToolCallID()
		}
		input := make(map[string]interface{})
		if toolCallArgs != "" {
			json.Unmarshal([]byte(toolCallArgs), &input)
		}

		chunk := &StreamChunk{
			Type:  "content_block_start",
			Index: 0,
			ContentBlock: &ContentBlock{
				Type:  "tool_use",
				ID:    id,
				Name:  currentToolCall.Function.Name,
				Input: input,
			},
		}
		if err := callback(chunk); err != nil {
			return err
		}
	}

	callback(&StreamChunk{Type: "message_stop"})
	return nil
}

// GroqProvider uses Groq's fast inference API (OpenAI-compatible)
type GroqProvider struct {
	*OpenAICompatibleProvider
}

// NewGroqProvider creates a new Groq provider
func NewGroqProvider(apiKey string) *GroqProvider {
	p := NewOpenAICompatibleProvider("groq", apiKey, "https://api.groq.com/openai/v1")
	p.models = []string{
		// Meta Llama
		"llama-3.3-70b-versatile",
		"llama-3.1-8b-instant",
		"llama-guard-3-8b",
		"meta-llama/llama-4-scout-17b-16e-instruct",
		"meta-llama/llama-4-maverick-17b-128e-instruct",
		"meta-llama/llama-guard-4-12b",
		"llama3-70b-8192",
		"llama3-8b-8192",
		// Qwen
		"qwen-qwq-32b",
		"qwen/qwen3-32b",
		// DeepSeek
		"deepseek-r1-distill-llama-70b",
		// Mistral
		"mistral-saba-24b",
		// Google Gemma
		"gemma2-9b-it",
		// Moonshot / Kimi
		"moonshotai/kimi-k2-instruct",
		"moonshotai/kimi-k2-instruct-0905",
		// OpenAI
		"openai/gpt-oss-120b",
		"openai/gpt-oss-20b",
	}
	return &GroqProvider{OpenAICompatibleProvider: p}
}

// OpenRouterProvider uses OpenRouter's multi-provider API
type OpenRouterProvider struct {
	*OpenAICompatibleProvider
}

// NewOpenRouterProvider creates a new OpenRouter provider
// OpenRouter provides access to 75+ models from various providers
func NewOpenRouterProvider(apiKey string) *OpenRouterProvider {
	p := NewOpenAICompatibleProvider("openrouter", apiKey, "https://openrouter.ai/api/v1")
	p.models = []string{
		// Anthropic Claude
		"anthropic/claude-opus-4.6",
		"anthropic/claude-sonnet-4.5",
		"anthropic/claude-opus-4.5",
		"anthropic/claude-opus-4.1",
		"anthropic/claude-sonnet-4",
		"anthropic/claude-opus-4",
		"anthropic/claude-haiku-4.5",
		"anthropic/claude-3.7-sonnet",
		"anthropic/claude-3.5-sonnet",
		"anthropic/claude-3.5-haiku",
		"anthropic/claude-3-opus",

		// OpenAI
		"openai/gpt-5.2-codex",
		"openai/gpt-5.2",
		"openai/gpt-5.1-codex-max",
		"openai/gpt-5.1-codex",
		"openai/gpt-5.1",
		"openai/gpt-5",
		"openai/gpt-5-mini",
		"openai/gpt-4.1",
		"openai/gpt-4o",
		"openai/gpt-4o-mini",
		"openai/o3",
		"openai/o3-pro",
		"openai/o4-mini",

		// Google Gemini
		"google/gemini-3-flash-preview",
		"google/gemini-3-pro-preview",
		"google/gemini-2.5-pro",
		"google/gemini-2.5-flash",
		"google/gemini-2.0-flash",

		// xAI Grok
		"x-ai/grok-4",
		"x-ai/grok-4-fast",
		"x-ai/grok-3",
		"x-ai/grok-3-fast",
		"x-ai/grok-3-mini",

		// Meta Llama
		"meta-llama/llama-4-maverick-17b-128e-instruct",
		"meta-llama/llama-4-scout-17b-16e-instruct",
		"meta-llama/llama-3.3-70b-instruct",
		"meta-llama/llama-3.1-405b-instruct",

		// DeepSeek
		"deepseek/deepseek-r1",
		"deepseek/deepseek-chat",

		// Mistral
		"mistralai/devstral-medium",
		"mistralai/magistral-medium",
		"mistralai/mistral-large",
		"mistralai/codestral",

		// Qwen
		"qwen/qwen3-coder-480b",
		"qwen/qwen3-235b",
		"qwen/qwq-32b",

		// Moonshot / Kimi
		"moonshotai/kimi-k2.5",
		"moonshotai/kimi-k2-instruct",

		// Cohere
		"cohere/command-r-plus",
		"cohere/command-r",

		// Perplexity
		"perplexity/sonar-pro",
		"perplexity/sonar",
	}
	return &OpenRouterProvider{OpenAICompatibleProvider: p}
}

// Shared conversion functions for OpenAI-compatible providers

func convertToOpenAIMessages(req *MessageRequest) []openai.ChatCompletionMessage {
	messages := make([]openai.ChatCompletionMessage, 0, len(req.Messages)+1)

	if req.System != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: req.System,
		})
	}

	for _, msg := range req.Messages {
		oaiMsg := openai.ChatCompletionMessage{Role: msg.Role}

		switch content := msg.Content.(type) {
		case string:
			oaiMsg.Content = content
		case []ContentBlock:
			textParts := []string{}
			var toolCalls []openai.ToolCall
			var multiContent []openai.ChatMessagePart

			for _, block := range content {
				switch block.Type {
				case "text":
					textParts = append(textParts, block.Text)
					multiContent = append(multiContent, openai.ChatMessagePart{
						Type: openai.ChatMessagePartTypeText,
						Text: block.Text,
					})
				case "image":
					// OpenAI-compatible vision: image_url content part
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
					id := block.ID
					if id == "" {
						id = generateToolCallID()
					}
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
							Name: block.Name, Arguments: string(inputJSON),
						},
					})
				case "tool_result":
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
						Role: openai.ChatMessageRoleTool, Content: resultContent, ToolCallID: block.ToolUseID,
					})
					continue
				}
			}

			// Use multi-part content when images are present; plain string otherwise
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

func convertToOpenAITools(tools []Tool) []openai.Tool {
	result := make([]openai.Tool, len(tools))
	for i, tool := range tools {
		paramsJSON, _ := json.Marshal(tool.InputSchema)
		var params map[string]interface{}
		json.Unmarshal(paramsJSON, &params)

		result[i] = openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name: tool.Name, Description: tool.Description, Parameters: params,
			},
		}
	}
	return result
}

func convertFromOpenAIResponse(resp *openai.ChatCompletionResponse) *MessageResponse {
	if len(resp.Choices) == 0 {
		return &MessageResponse{ID: resp.ID, Model: resp.Model, Role: "assistant"}
	}

	choice := resp.Choices[0]
	content := []ContentBlock{}

	if choice.Message.Content != "" {
		content = append(content, ContentBlock{Type: "text", Text: choice.Message.Content})
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
			Type: "tool_use", ID: id, Name: tc.Function.Name, Input: input,
		})
	}

	stopReason := string(choice.FinishReason)
	if stopReason == "stop" {
		stopReason = "end_turn"
	}

	return &MessageResponse{
		ID: resp.ID, Model: resp.Model, Role: "assistant",
		Content: content, StopReason: stopReason,
		Usage: Usage{InputTokens: resp.Usage.PromptTokens, OutputTokens: resp.Usage.CompletionTokens},
	}
}
