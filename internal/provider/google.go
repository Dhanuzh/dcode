package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// GoogleProvider implements the Provider interface for Google Gemini
type GoogleProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewGoogleProvider creates a new Google Gemini provider
func NewGoogleProvider(apiKey string) *GoogleProvider {
	return &GoogleProvider{
		apiKey:  apiKey,
		baseURL: "https://generativelanguage.googleapis.com/v1beta",
		client:  &http.Client{},
	}
}

func (p *GoogleProvider) Name() string { return "google" }

func (p *GoogleProvider) Models() []string {
	return []string{
		// Gemini 3.x
		"gemini-3-flash-preview",
		"gemini-3-pro-preview",
		// Gemini 2.5
		"gemini-2.5-pro",
		"gemini-2.5-pro-preview-06-05",
		"gemini-2.5-pro-preview-05-06",
		"gemini-2.5-pro-preview-tts",
		"gemini-2.5-flash",
		"gemini-2.5-flash-lite",
		"gemini-2.5-flash-preview-09-2025",
		"gemini-2.5-flash-preview-05-20",
		"gemini-2.5-flash-preview-04-17",
		"gemini-2.5-flash-lite-preview-09-2025",
		"gemini-2.5-flash-lite-preview-06-17",
		"gemini-2.5-flash-image",
		"gemini-2.5-flash-image-preview",
		"gemini-2.5-flash-preview-tts",
		// Gemini 2.0
		"gemini-2.0-flash",
		"gemini-2.0-flash-lite",
		// Gemini 1.5
		"gemini-1.5-pro",
		"gemini-1.5-flash",
		"gemini-1.5-flash-8b",
		// Aliases
		"gemini-flash-latest",
		"gemini-flash-lite-latest",
		// Live / Embedding
		"gemini-live-2.5-flash",
		"gemini-live-2.5-flash-preview-native-audio",
		"gemini-embedding-001",
	}
}

func (p *GoogleProvider) CreateMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error) {
	model := req.Model
	if model == "" {
		model = "gemini-2.5-flash"
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, model, p.apiKey)

	reqBody := p.buildRequest(req)

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	return p.parseResponse(body, model)
}

func (p *GoogleProvider) StreamMessage(ctx context.Context, req *MessageRequest, callback func(*StreamChunk) error) error {
	// Use non-streaming for simplicity, then convert to chunks
	resp, err := p.CreateMessage(ctx, req)
	if err != nil {
		return err
	}

	callback(&StreamChunk{Type: "message_start", Message: resp})

	for i, block := range resp.Content {
		callback(&StreamChunk{Type: "content_block_start", Index: i, ContentBlock: &block})

		if block.Type == "text" {
			callback(&StreamChunk{
				Type: "content_block_delta", Index: i,
				Delta: &Delta{Type: "text_delta", Text: block.Text},
			})
		}

		callback(&StreamChunk{Type: "content_block_stop", Index: i})
	}

	callback(&StreamChunk{Type: "message_stop"})
	return nil
}

func (p *GoogleProvider) buildRequest(req *MessageRequest) map[string]interface{} {
	contents := []map[string]interface{}{}

	for _, msg := range req.Messages {
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}

		parts := []map[string]interface{}{}

		switch content := msg.Content.(type) {
		case string:
			parts = append(parts, map[string]interface{}{"text": content})
		case []ContentBlock:
			for _, block := range content {
				switch block.Type {
				case "text":
					parts = append(parts, map[string]interface{}{"text": block.Text})
				case "image":
					// Gemini inline image: {"inlineData": {"mimeType": "image/png", "data": "<base64>"}}
					if block.Source != nil && block.Source.Type == "base64" && block.Source.Data != "" {
						parts = append(parts, map[string]interface{}{
							"inlineData": map[string]interface{}{
								"mimeType": block.Source.MediaType,
								"data":     block.Source.Data,
							},
						})
					} else if block.Source != nil && block.Source.URL != "" {
						// URL reference for Gemini
						parts = append(parts, map[string]interface{}{
							"fileData": map[string]interface{}{
								"mimeType": block.Source.MediaType,
								"fileUri":  block.Source.URL,
							},
						})
					}
				case "tool_use":
					inputJSON, _ := json.Marshal(block.Input)
					parts = append(parts, map[string]interface{}{
						"functionCall": map[string]interface{}{
							"name": block.Name,
							"args": json.RawMessage(inputJSON),
						},
					})
				case "tool_result":
					resultStr := ""
					switch v := block.Content.(type) {
					case string:
						resultStr = v
					default:
						j, _ := json.Marshal(v)
						resultStr = string(j)
					}
					parts = append(parts, map[string]interface{}{
						"functionResponse": map[string]interface{}{
							"name": block.Name,
							"response": map[string]interface{}{
								"content": resultStr,
							},
						},
					})
				}
			}
		}

		if len(parts) > 0 {
			contents = append(contents, map[string]interface{}{
				"role":  role,
				"parts": parts,
			})
		}
	}

	result := map[string]interface{}{
		"contents": contents,
	}

	if req.System != "" {
		result["systemInstruction"] = map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": req.System},
			},
		}
	}

	genConfig := map[string]interface{}{}
	if req.MaxTokens > 0 {
		genConfig["maxOutputTokens"] = req.MaxTokens
	}
	if req.Temperature != 0 {
		genConfig["temperature"] = req.Temperature
	}
	if len(genConfig) > 0 {
		result["generationConfig"] = genConfig
	}

	if len(req.Tools) > 0 {
		funcDecls := make([]map[string]interface{}, len(req.Tools))
		for i, tool := range req.Tools {
			funcDecls[i] = map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  tool.InputSchema,
			}
		}
		result["tools"] = []map[string]interface{}{
			{"functionDeclarations": funcDecls},
		}
	}

	return result
}

func (p *GoogleProvider) parseResponse(body []byte, model string) (*MessageResponse, error) {
	var apiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text         string `json:"text,omitempty"`
					FunctionCall *struct {
						Name string                 `json:"name"`
						Args map[string]interface{} `json:"args"`
					} `json:"functionCall,omitempty"`
				} `json:"parts"`
				Role string `json:"role"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	candidate := apiResp.Candidates[0]
	content := []ContentBlock{}
	toolCallIdx := 0

	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			content = append(content, ContentBlock{Type: "text", Text: part.Text})
		}
		if part.FunctionCall != nil {
			toolCallIdx++
			content = append(content, ContentBlock{
				Type:  "tool_use",
				ID:    fmt.Sprintf("call_%d", toolCallIdx),
				Name:  part.FunctionCall.Name,
				Input: part.FunctionCall.Args,
			})
		}
	}

	stopReason := strings.ToLower(candidate.FinishReason)
	if stopReason == "stop" {
		stopReason = "end_turn"
	}

	return &MessageResponse{
		ID: "gemini-resp", Model: model, Role: "assistant",
		Content: content, StopReason: stopReason,
		Usage: Usage{
			InputTokens:  apiResp.UsageMetadata.PromptTokenCount,
			OutputTokens: apiResp.UsageMetadata.CandidatesTokenCount,
		},
	}, nil
}
