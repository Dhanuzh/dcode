package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// GoogleVertexProvider uses Google Vertex AI with native Gemini API
type GoogleVertexProvider struct {
	projectID string
	region    string
	apiKey    string
	client    *http.Client
}

// NewGoogleVertexProvider creates a new Google Vertex AI provider
func NewGoogleVertexProvider(apiKey string) *GoogleVertexProvider {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		projectID = os.Getenv("GCLOUD_PROJECT")
	}
	region := os.Getenv("GOOGLE_CLOUD_REGION")
	if region == "" {
		region = "us-central1"
	}

	return &GoogleVertexProvider{
		projectID: projectID,
		region:    region,
		apiKey:    apiKey,
		client:    &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *GoogleVertexProvider) Name() string {
	return "google-vertex"
}

func (p *GoogleVertexProvider) Models() []string {
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
		// Gemini 2.0
		"gemini-2.0-flash",
		"gemini-2.0-flash-lite",
		// Gemini 1.5
		"gemini-1.5-pro",
		"gemini-1.5-flash",
		// Partner models via Vertex
		"zai-org/glm-5-maas",
		"zai-org/glm-4.7-maas",
		"openai/gpt-oss-120b-maas",
		"openai/gpt-oss-20b-maas",
	}
}

func (p *GoogleVertexProvider) CreateMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error) {
	body := p.buildRequest(req)
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := p.getEndpoint(req.Model, false)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Use access token or API key
	if token := os.Getenv("GOOGLE_ACCESS_TOKEN"); token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	} else if p.apiKey != "" {
		q := httpReq.URL.Query()
		q.Set("key", p.apiKey)
		httpReq.URL.RawQuery = q.Encode()
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("vertex ai request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		classified := ClassifyError(fmt.Errorf("%s", string(respBody)), resp.StatusCode, string(respBody))
		return nil, classified
	}

	return p.parseResponse(respBody, req.Model)
}

func (p *GoogleVertexProvider) StreamMessage(ctx context.Context, req *MessageRequest, callback func(*StreamChunk) error) error {
	body := p.buildRequest(req)
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := p.getEndpoint(req.Model, true)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if token := os.Getenv("GOOGLE_ACCESS_TOKEN"); token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	} else if p.apiKey != "" {
		q := httpReq.URL.Query()
		q.Set("key", p.apiKey)
		httpReq.URL.RawQuery = q.Encode()
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("vertex ai stream request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		classified := ClassifyError(fmt.Errorf("%s", string(respBody)), resp.StatusCode, string(respBody))
		return classified
	}

	// Parse SSE stream
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}

		var chunk map[string]interface{}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		candidates, ok := chunk["candidates"].([]interface{})
		if !ok || len(candidates) == 0 {
			continue
		}

		candidate, ok := candidates[0].(map[string]interface{})
		if !ok {
			continue
		}

		content, ok := candidate["content"].(map[string]interface{})
		if !ok {
			continue
		}

		parts, ok := content["parts"].([]interface{})
		if !ok {
			continue
		}

		for _, part := range parts {
			partMap, ok := part.(map[string]interface{})
			if !ok {
				continue
			}

			if text, ok := partMap["text"].(string); ok {
				if err := callback(&StreamChunk{
					Type:  "content_block_delta",
					Delta: &Delta{Type: "text_delta", Text: text},
				}); err != nil {
					return err
				}
			}

			if fc, ok := partMap["functionCall"].(map[string]interface{}); ok {
				name, _ := fc["name"].(string)
				args, _ := fc["args"].(map[string]interface{})
				argsJSON, _ := json.Marshal(args)
				if err := callback(&StreamChunk{
					Type: "content_block_start",
					ContentBlock: &ContentBlock{
						Type:  "tool_use",
						ID:    fmt.Sprintf("call_%s", name),
						Name:  name,
						Input: args,
					},
					Delta: &Delta{Type: "input_json_delta", PartialJSON: string(argsJSON)},
				}); err != nil {
					return err
				}
			}
		}
	}

	// Send completion
	callback(&StreamChunk{
		Type: "message_stop",
		Message: &MessageResponse{
			Model:      req.Model,
			Role:       "assistant",
			StopReason: "end_turn",
		},
	})

	return scanner.Err()
}

func (p *GoogleVertexProvider) getEndpoint(model string, stream bool) string {
	action := "generateContent"
	if stream {
		action = "streamGenerateContent?alt=sse"
	}

	if p.projectID != "" {
		return fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:%s",
			p.region, p.projectID, p.region, model, action)
	}

	// Fallback to Generative Language API
	return fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:%s", model, action)
}

func (p *GoogleVertexProvider) buildRequest(req *MessageRequest) map[string]interface{} {
	contents := make([]map[string]interface{}, 0, len(req.Messages))
	for _, msg := range req.Messages {
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}
		if role == "system" {
			continue // handled separately
		}

		parts := []map[string]interface{}{}
		switch v := msg.Content.(type) {
		case string:
			parts = append(parts, map[string]interface{}{"text": v})
		case []ContentBlock:
			for _, block := range v {
				switch block.Type {
				case "text":
					parts = append(parts, map[string]interface{}{"text": block.Text})
				case "tool_use":
					parts = append(parts, map[string]interface{}{
						"functionCall": map[string]interface{}{
							"name": block.Name,
							"args": block.Input,
						},
					})
				case "tool_result":
					parts = append(parts, map[string]interface{}{
						"functionResponse": map[string]interface{}{
							"name": block.Name,
							"response": map[string]interface{}{
								"content": block.Content,
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

	body := map[string]interface{}{
		"contents": contents,
	}

	if req.System != "" {
		body["systemInstruction"] = map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": req.System},
			},
		}
	}

	generationConfig := map[string]interface{}{}
	if req.MaxTokens > 0 {
		generationConfig["maxOutputTokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		generationConfig["temperature"] = req.Temperature
	}
	if len(generationConfig) > 0 {
		body["generationConfig"] = generationConfig
	}

	if len(req.Tools) > 0 {
		funcDecls := make([]map[string]interface{}, 0, len(req.Tools))
		for _, t := range req.Tools {
			funcDecls = append(funcDecls, map[string]interface{}{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.InputSchema,
			})
		}
		body["tools"] = []map[string]interface{}{
			{"functionDeclarations": funcDecls},
		}
	}

	return body
}

func (p *GoogleVertexProvider) parseResponse(body []byte, model string) (*MessageResponse, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	resp := &MessageResponse{
		Model:      model,
		Role:       "assistant",
		StopReason: "end_turn",
	}

	candidates, ok := raw["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		return resp, nil
	}

	candidate, ok := candidates[0].(map[string]interface{})
	if !ok {
		return resp, nil
	}

	content, ok := candidate["content"].(map[string]interface{})
	if !ok {
		return resp, nil
	}

	parts, ok := content["parts"].([]interface{})
	if !ok {
		return resp, nil
	}

	for _, part := range parts {
		partMap, ok := part.(map[string]interface{})
		if !ok {
			continue
		}

		if text, ok := partMap["text"].(string); ok {
			resp.Content = append(resp.Content, ContentBlock{
				Type: "text",
				Text: text,
			})
		}

		if fc, ok := partMap["functionCall"].(map[string]interface{}); ok {
			name, _ := fc["name"].(string)
			args, _ := fc["args"].(map[string]interface{})
			resp.Content = append(resp.Content, ContentBlock{
				Type:  "tool_use",
				ID:    fmt.Sprintf("call_%s", name),
				Name:  name,
				Input: args,
			})
			resp.StopReason = "tool_use"
		}
	}

	if usage, ok := raw["usageMetadata"].(map[string]interface{}); ok {
		if v, ok := usage["promptTokenCount"].(float64); ok {
			resp.Usage.InputTokens = int(v)
		}
		if v, ok := usage["candidatesTokenCount"].(float64); ok {
			resp.Usage.OutputTokens = int(v)
		}
	}

	return resp, nil
}
