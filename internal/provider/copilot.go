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
	"path/filepath"
	"strings"
	"time"
)

// CopilotProvider implements the Provider interface for GitHub Copilot
type CopilotProvider struct {
	authToken string
	baseURL   string
	client    *http.Client
}

// copilotOAuthInfo stores the OAuth token from device flow
type copilotOAuthInfo struct {
	AccessToken string `json:"access_token"`
	CreatedAt   int64  `json:"created_at"`
}

const copilotClientID = "Ov23li8tweQw6odWQebz"

// NewCopilotProvider creates a new Copilot provider
func NewCopilotProvider() (*CopilotProvider, error) {
	token, err := getGitHubToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub token: %w", err)
	}

	p := &CopilotProvider{
		authToken: token,
		baseURL:   "https://api.individual.githubcopilot.com",
		client:    &http.Client{Timeout: 120 * time.Second},
	}

	return p, nil
}

func (p *CopilotProvider) Name() string { return "copilot" }

func (p *CopilotProvider) Models() []string {
	return []string{
		"claude-sonnet-4-20250514",
		"claude-3.5-sonnet",
		"gpt-4.1",
		"gpt-4.1-mini",
		"gpt-4.1-nano",
		"gpt-4o",
		"gpt-4o-mini",
		"gpt-4-turbo",
		"o3",
		"o3-mini",
		"o4-mini",
	}
}

// getGitHubToken gets a GitHub OAuth token for Copilot.
// Only OAuth tokens from the device flow work with Copilot's API.
// Personal access tokens (PATs) from GITHUB_TOKEN or gh auth will NOT work.
func getGitHubToken() (string, error) {
	// Only use OAuth token obtained via 'dcode copilot-login'
	if token, err := loadCopilotOAuthToken(); err == nil && token != "" {
		return token, nil
	}

	return "", fmt.Errorf("no Copilot OAuth token found. Run 'dcode copilot-login' to authenticate with GitHub Copilot")
}

// loadCopilotOAuthToken loads a previously stored Copilot OAuth token
func loadCopilotOAuthToken() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(home, ".config", "dcode", "copilot_oauth.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var info copilotOAuthInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return "", err
	}
	return info.AccessToken, nil
}

// saveCopilotOAuthToken stores the Copilot OAuth token
func saveCopilotOAuthToken(token string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".config", "dcode")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	info := copilotOAuthInfo{AccessToken: token, CreatedAt: time.Now().Unix()}
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "copilot_oauth.json"), data, 0600)
}



// CopilotLogin performs the GitHub OAuth device flow for Copilot authentication
func CopilotLogin() error {
	client := &http.Client{Timeout: 30 * time.Second}

	// Step 1: Request device code
	deviceBody, _ := json.Marshal(map[string]string{
		"client_id": copilotClientID,
		"scope":     "read:user",
	})

	deviceReq, _ := http.NewRequest("POST", "https://github.com/login/device/code",
		bytes.NewReader(deviceBody))
	deviceReq.Header.Set("Content-Type", "application/json")
	deviceReq.Header.Set("Accept", "application/json")

	resp, err := client.Do(deviceReq)
	if err != nil {
		return fmt.Errorf("failed to request device code: %w", err)
	}
	defer resp.Body.Close()

	var deviceResp struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		Interval        int    `json:"interval"`
		ExpiresIn       int    `json:"expires_in"`
	}

	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &deviceResp); err != nil {
		return fmt.Errorf("failed to parse device code response: %w", err)
	}

	fmt.Printf("\n\033[1;36m╭──────────────────────────────────────────────╮\033[0m\n")
	fmt.Printf("\033[1;36m│      GitHub Copilot Authentication           │\033[0m\n")
	fmt.Printf("\033[1;36m╰──────────────────────────────────────────────╯\033[0m\n\n")
	fmt.Printf("  1. Open: \033[4m%s\033[0m\n", deviceResp.VerificationURI)
	fmt.Printf("  2. Enter code: \033[1;33m%s\033[0m\n\n", deviceResp.UserCode)
	fmt.Printf("  Waiting for authorization...\n")

	// Step 2: Poll for access token
	interval := time.Duration(deviceResp.Interval+1) * time.Second
	deadline := time.Now().Add(time.Duration(deviceResp.ExpiresIn) * time.Second)

	for time.Now().Before(deadline) {
		time.Sleep(interval)

		tokenBody, _ := json.Marshal(map[string]string{
			"client_id":   copilotClientID,
			"device_code": deviceResp.DeviceCode,
			"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
		})

		tokenReq, _ := http.NewRequest("POST", "https://github.com/login/oauth/access_token",
			bytes.NewReader(tokenBody))
		tokenReq.Header.Set("Content-Type", "application/json")
		tokenReq.Header.Set("Accept", "application/json")

		tokenResp, err := client.Do(tokenReq)
		if err != nil {
			continue
		}

		tokenRespBody, _ := io.ReadAll(tokenResp.Body)
		tokenResp.Body.Close()

		var result struct {
			AccessToken string `json:"access_token"`
			TokenType   string `json:"token_type"`
			Error       string `json:"error"`
		}

		if err := json.Unmarshal(tokenRespBody, &result); err != nil {
			continue
		}

		switch result.Error {
		case "authorization_pending":
			continue
		case "slow_down":
			interval += 5 * time.Second
			continue
		case "expired_token":
			return fmt.Errorf("device code expired, please try again")
		case "access_denied":
			return fmt.Errorf("authorization denied by user")
		case "":
			if result.AccessToken != "" {
				// Save the token
				if err := saveCopilotOAuthToken(result.AccessToken); err != nil {
					return fmt.Errorf("failed to save token: %w", err)
				}
				fmt.Printf("\n  \033[1;32m✓ GitHub Copilot authenticated successfully!\033[0m\n\n")
				return nil
			}
		default:
			return fmt.Errorf("authentication error: %s", result.Error)
		}
	}

	return fmt.Errorf("authentication timed out")
}

func (p *CopilotProvider) CreateMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error) {
	messages := p.convertMessages(req.Messages)

	if req.System != "" {
		messages = append([]map[string]interface{}{
			{"role": "system", "content": req.System},
		}, messages...)
	}

	reqBody := map[string]interface{}{
		"messages":    messages,
		"model":       req.Model,
		"temperature": req.Temperature,
		"top_p":       1,
		"n":           1,
		"stream":      false,
	}

	if req.MaxTokens > 0 {
		reqBody["max_tokens"] = req.MaxTokens
	}
	if len(req.Tools) > 0 {
		reqBody["tools"] = p.convertTools(req.Tools)
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.authToken)
	httpReq.Header.Set("User-Agent", "DCode/2.0")
	httpReq.Header.Set("Openai-Intent", "conversation-edits")

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
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("Copilot authentication failed (401). Run 'dcode copilot-login' to re-authenticate")
		}
		return nil, fmt.Errorf("Copilot API error (%d): %s", resp.StatusCode, string(body))
	}

	return p.parseOpenAIResponse(body)
}

func (p *CopilotProvider) StreamMessage(ctx context.Context, req *MessageRequest, callback func(*StreamChunk) error) error {
	messages := p.convertMessages(req.Messages)

	if req.System != "" {
		messages = append([]map[string]interface{}{
			{"role": "system", "content": req.System},
		}, messages...)
	}

	reqBody := map[string]interface{}{
		"messages":    messages,
		"model":       req.Model,
		"temperature": req.Temperature,
		"top_p":       1,
		"n":           1,
		"stream":      true,
	}

	if req.MaxTokens > 0 {
		reqBody["max_tokens"] = req.MaxTokens
	}
	if len(req.Tools) > 0 {
		reqBody["tools"] = p.convertTools(req.Tools)
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.authToken)
	httpReq.Header.Set("User-Agent", "DCode/2.0")
	httpReq.Header.Set("Openai-Intent", "conversation-edits")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("Copilot authentication failed (401). Run 'dcode copilot-login' to re-authenticate")
		}
		return fmt.Errorf("Copilot API error (%d): %s", resp.StatusCode, string(body))
	}

	// Parse SSE stream
	return p.parseSSEStream(resp.Body, callback)
}

// parseSSEStream parses the Server-Sent Events stream from Copilot
func (p *CopilotProvider) parseSSEStream(body io.Reader, callback func(*StreamChunk) error) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for large responses

	var accumulated MessageResponse
	accumulated.Role = "assistant"
	accumulated.Content = []ContentBlock{}

	textContent := ""
	var toolCalls []struct {
		ID       string
		Name     string
		Args     string
	}

	sentStart := false

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk struct {
			ID      string `json:"id"`
			Model   string `json:"model"`
			Choices []struct {
				Index int `json:"index"`
				Delta struct {
					Role      string `json:"role"`
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls,omitempty"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage,omitempty"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if !sentStart {
			accumulated.ID = chunk.ID
			accumulated.Model = chunk.Model
			callback(&StreamChunk{Type: "message_start", Message: &accumulated})
			sentStart = true
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		choice := chunk.Choices[0]
		delta := choice.Delta

		// Handle text content
		if delta.Content != "" {
			if textContent == "" {
				callback(&StreamChunk{
					Type:         "content_block_start",
					Index:        0,
					ContentBlock: &ContentBlock{Type: "text"},
				})
			}
			textContent += delta.Content
			callback(&StreamChunk{
				Type:  "content_block_delta",
				Index: 0,
				Delta: &Delta{Type: "text_delta", Text: delta.Content},
			})
		}

		// Handle tool calls
		for _, tc := range delta.ToolCalls {
			for len(toolCalls) <= tc.Index {
				toolCalls = append(toolCalls, struct {
					ID   string
					Name string
					Args string
				}{})
			}
			if tc.ID != "" {
				toolCalls[tc.Index].ID = tc.ID
			}
			if tc.Function.Name != "" {
				toolCalls[tc.Index].Name = tc.Function.Name
			}
			toolCalls[tc.Index].Args += tc.Function.Arguments
		}

		// Handle finish
		if choice.FinishReason != nil {
			if textContent != "" {
				callback(&StreamChunk{Type: "content_block_stop", Index: 0})
			}

			// Emit tool call blocks
			for i, tc := range toolCalls {
				idx := i + 1
				if textContent == "" {
					idx = i
				}
				var input map[string]interface{}
				json.Unmarshal([]byte(tc.Args), &input)
				block := ContentBlock{
					Type: "tool_use", ID: tc.ID, Name: tc.Name, Input: input,
				}
				callback(&StreamChunk{
					Type: "content_block_start", Index: idx,
					ContentBlock: &block,
				})
				callback(&StreamChunk{Type: "content_block_stop", Index: idx})
			}

			if chunk.Usage != nil {
				accumulated.Usage = Usage{
					InputTokens:  chunk.Usage.PromptTokens,
					OutputTokens: chunk.Usage.CompletionTokens,
				}
			}
		}
	}

	callback(&StreamChunk{Type: "message_stop"})

	// Build final content
	accumulated.Content = []ContentBlock{}
	if textContent != "" {
		accumulated.Content = append(accumulated.Content, ContentBlock{Type: "text", Text: textContent})
	}
	for _, tc := range toolCalls {
		var input map[string]interface{}
		json.Unmarshal([]byte(tc.Args), &input)
		accumulated.Content = append(accumulated.Content, ContentBlock{
			Type: "tool_use", ID: tc.ID, Name: tc.Name, Input: input,
		})
	}

	if len(toolCalls) > 0 {
		accumulated.StopReason = "tool_use"
	} else {
		accumulated.StopReason = "end_turn"
	}

	return scanner.Err()
}

func (p *CopilotProvider) parseOpenAIResponse(body []byte) (*MessageResponse, error) {
	var apiResp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Index   int `json:"index"`
			Message struct {
				Role      string `json:"role"`
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls,omitempty"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := apiResp.Choices[0]
	content := []ContentBlock{}

	if choice.Message.Content != "" {
		content = append(content, ContentBlock{Type: "text", Text: choice.Message.Content})
	}

	for _, toolCall := range choice.Message.ToolCalls {
		var input map[string]interface{}
		json.Unmarshal([]byte(toolCall.Function.Arguments), &input)
		content = append(content, ContentBlock{
			Type: "tool_use", ID: toolCall.ID, Name: toolCall.Function.Name, Input: input,
		})
	}

	stopReason := choice.FinishReason
	if stopReason == "stop" {
		stopReason = "end_turn"
	}

	return &MessageResponse{
		ID: apiResp.ID, Model: apiResp.Model, Role: "assistant",
		Content: content, StopReason: stopReason,
		Usage: Usage{InputTokens: apiResp.Usage.PromptTokens, OutputTokens: apiResp.Usage.CompletionTokens},
	}, nil
}

func (p *CopilotProvider) convertMessages(messages []Message) []map[string]interface{} {
	result := make([]map[string]interface{}, 0)

	for _, msg := range messages {
		msgMap := map[string]interface{}{"role": msg.Role}

		switch content := msg.Content.(type) {
		case string:
			msgMap["content"] = content
		case []ContentBlock:
			textParts := []string{}
			var toolCalls []map[string]interface{}

			for _, block := range content {
				switch block.Type {
				case "text":
					textParts = append(textParts, block.Text)
				case "tool_use":
					inputJSON, _ := json.Marshal(block.Input)
					toolCalls = append(toolCalls, map[string]interface{}{
						"id": block.ID, "type": "function",
						"function": map[string]interface{}{
							"name": block.Name, "arguments": string(inputJSON),
						},
					})
				case "tool_result":
					resultContent := ""
					switch v := block.Content.(type) {
					case string:
						resultContent = v
					default:
						resultJSON, _ := json.Marshal(v)
						resultContent = string(resultJSON)
					}
					result = append(result, map[string]interface{}{
						"role": "tool", "content": resultContent, "tool_call_id": block.ToolUseID,
					})
					continue
				}
			}

			if len(textParts) > 0 {
				msgMap["content"] = strings.Join(textParts, "\n")
			}
			if len(toolCalls) > 0 {
				msgMap["tool_calls"] = toolCalls
			}
		}

		result = append(result, msgMap)
	}

	return result
}

func (p *CopilotProvider) convertTools(tools []Tool) []map[string]interface{} {
	result := make([]map[string]interface{}, len(tools))
	for i, tool := range tools {
		result[i] = map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  tool.InputSchema,
			},
		}
	}
	return result
}
