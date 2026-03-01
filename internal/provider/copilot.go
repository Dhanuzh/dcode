package provider

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// generateToolCallID generates a unique tool call ID like "call_xxxxxxxxxxxx"
func generateToolCallID() string {
	b := make([]byte, 12)
	rand.Read(b)
	return "call_" + hex.EncodeToString(b)
}

// CopilotProvider implements the Provider interface for GitHub Copilot.
// It uses the GitHub OAuth token directly against api.individual.githubcopilot.com.
type CopilotProvider struct {
	authToken    string // GitHub OAuth token from device flow
	baseURL      string
	client       *http.Client
	cachedModels []string // models fetched from /models endpoint
	modelsMu     sync.Mutex
}

// copilotOAuthInfo stores the OAuth token from device flow
type copilotOAuthInfo struct {
	AccessToken string `json:"access_token"`
	CreatedAt   int64  `json:"created_at"`
}

const copilotClientID = "Ov23li8tweQw6odWQebz"

// NewCopilotProvider creates a new Copilot provider using the stored OAuth token.
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

// copilotModelEntry is the shape of each object in the /models response
type copilotModelEntry struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	ModelPickerEnabled bool   `json:"model_picker_enabled"`
	Policy             *struct {
		State string `json:"state"`
	} `json:"policy,omitempty"`
	SupportedEndpoints []string `json:"supported_endpoints,omitempty"`
}

// FetchModels calls the Copilot /models endpoint and returns the enabled model IDs.
// It only returns models that are enabled (policy.state == "enabled" or no policy)
// and support the /chat/completions endpoint.
func (p *CopilotProvider) FetchModels() ([]string, error) {
	req, err := http.NewRequest("GET", p.baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	p.setHeaders(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("models API error (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []copilotModelEntry `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse models response: %w", err)
	}

	// Deduplicate by ID — the API returns duplicate entries for some models
	seen := make(map[string]bool)
	var ids []string
	for _, m := range result.Data {
		if seen[m.ID] {
			continue
		}
		// Skip models with a policy that is not "enabled"
		if m.Policy != nil && m.Policy.State != "enabled" {
			continue
		}
		// Only include models that support /chat/completions
		supportsChat := len(m.SupportedEndpoints) == 0 // legacy entries have no endpoints field
		for _, ep := range m.SupportedEndpoints {
			if ep == "/chat/completions" {
				supportsChat = true
				break
			}
		}
		if !supportsChat {
			continue
		}
		seen[m.ID] = true
		ids = append(ids, m.ID)
	}
	return ids, nil
}

// Models returns the list of model IDs available for this account.
// It fetches from the API on the first call and caches the result.
// Falls back to a safe default list if the API call fails.
func (p *CopilotProvider) Models() []string {
	p.modelsMu.Lock()
	defer p.modelsMu.Unlock()

	if len(p.cachedModels) > 0 {
		return p.cachedModels
	}

	fetched, err := p.FetchModels()
	if err == nil && len(fetched) > 0 {
		p.cachedModels = fetched
		return p.cachedModels
	}

	// Fallback: models known to work on the free plan
	p.cachedModels = []string{
		"gpt-4o",
		"gpt-4.1",
		"gpt-5-mini",
		"claude-haiku-4.5",
		"grok-code-fast-1",
	}
	return p.cachedModels
}

// setHeaders adds the required headers to every Copilot API request.
func (p *CopilotProvider) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+p.authToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "DCode/2.0")
	req.Header.Set("Openai-Intent", "conversation-edits")
	req.Header.Set("Editor-Version", "vscode/1.95.3")
	req.Header.Set("Editor-Plugin-Version", "copilot-chat/0.26.7")
	req.Header.Set("Copilot-Integration-Id", "vscode-chat")
}

// CopilotStatusInfo holds authentication and subscription status for GitHub Copilot
type CopilotStatusInfo struct {
	Username     string
	Email        string
	Plan         string
	IsActive     bool
	SessionLimit string
	Error        string
}

// GetCopilotStatus checks GitHub authentication and Copilot subscription status.
func (p *CopilotProvider) GetCopilotStatus() *CopilotStatusInfo {
	info := &CopilotStatusInfo{}

	// Get GitHub user info
	userReq, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		info.Error = "failed to build request: " + err.Error()
		return info
	}
	userReq.Header.Set("Authorization", "Bearer "+p.authToken)
	userReq.Header.Set("Accept", "application/vnd.github+json")
	userReq.Header.Set("User-Agent", "DCode/2.0")

	userResp, err := p.client.Do(userReq)
	if err != nil {
		info.Error = "GitHub API unreachable: " + err.Error()
		return info
	}
	defer userResp.Body.Close()

	userBody, _ := io.ReadAll(userResp.Body)
	if userResp.StatusCode == http.StatusUnauthorized {
		info.Error = "token invalid or expired — re-login with Copilot OAuth"
		return info
	}

	var ghUser struct {
		Login string `json:"login"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if jsonErr := json.Unmarshal(userBody, &ghUser); jsonErr == nil {
		info.Username = ghUser.Login
		if ghUser.Name != "" {
			info.Username = ghUser.Name + " (@" + ghUser.Login + ")"
		}
		info.Email = ghUser.Email
	}

	// Get Copilot user info (free plan quotas etc.)
	copilotReq, err := http.NewRequest("GET", "https://api.github.com/copilot_internal/user", nil)
	if err == nil {
		copilotReq.Header.Set("Authorization", "token "+p.authToken)
		copilotReq.Header.Set("Accept", "application/json")
		copilotReq.Header.Set("User-Agent", "DCode/2.0")

		copilotResp, copilotErr := p.client.Do(copilotReq)
		if copilotErr == nil {
			defer copilotResp.Body.Close()
			copilotBody, _ := io.ReadAll(copilotResp.Body)

			if copilotResp.StatusCode == http.StatusOK {
				var cu struct {
					CopilotPlan   string `json:"copilot_plan"`
					AccessTypeSku string `json:"access_type_sku"`
					ChatEnabled   bool   `json:"chat_enabled"`
					MonthlyQuotas *struct {
						Chat        int `json:"chat"`
						Completions int `json:"completions"`
					} `json:"monthly_quotas,omitempty"`
				}
				if json.Unmarshal(copilotBody, &cu) == nil {
					info.IsActive = cu.ChatEnabled
					info.Plan = cu.CopilotPlan
					if cu.AccessTypeSku != "" {
						info.Plan = cu.AccessTypeSku
					}
					if cu.MonthlyQuotas != nil {
						info.SessionLimit = fmt.Sprintf("Chat: %d/mo, Completions: %d/mo",
							cu.MonthlyQuotas.Chat, cu.MonthlyQuotas.Completions)
					}
				}
			}
		}
	}

	if info.Username != "" && info.Plan == "" {
		info.IsActive = true
		info.Plan = "Active"
	}

	return info
}

func (p *CopilotProvider) CreateMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error) {
	normalizedMsgs := NormalizeMessages(req.Messages, "copilot")
	messages := p.convertMessages(normalizedMsgs)

	if req.System != "" {
		messages = append([]map[string]interface{}{
			{"role": "system", "content": req.System},
		}, messages...)
	}

	model := req.Model
	if model == "" {
		model = "gpt-4o"
	}

	reqBody := map[string]interface{}{
		"messages":    messages,
		"model":       model,
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
	p.setHeaders(httpReq)

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
			return nil, fmt.Errorf("Copilot authentication failed (401) — run Copilot OAuth Login from Ctrl+P")
		}
		// Try to surface the API error message
		var apiErr struct {
			Error struct {
				Message string `json:"message"`
				Code    string `json:"code"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Error.Message != "" {
			return nil, fmt.Errorf("Copilot API error: %s (model: %s)", apiErr.Error.Message, model)
		}
		return nil, fmt.Errorf("Copilot API error (%d): %s", resp.StatusCode, string(body))
	}

	return p.parseOpenAIResponse(body)
}

func (p *CopilotProvider) StreamMessage(ctx context.Context, req *MessageRequest, callback func(*StreamChunk) error) error {
	normalizedMsgs := NormalizeMessages(req.Messages, "copilot")
	messages := p.convertMessages(normalizedMsgs)

	if req.System != "" {
		messages = append([]map[string]interface{}{
			{"role": "system", "content": req.System},
		}, messages...)
	}

	model := req.Model
	if model == "" {
		model = "gpt-4o"
	}

	reqBody := map[string]interface{}{
		"messages":    messages,
		"model":       model,
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
	p.setHeaders(httpReq)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("Copilot authentication failed (401) — run Copilot OAuth Login from Ctrl+P")
		}
		var apiErr struct {
			Error struct {
				Message string `json:"message"`
				Code    string `json:"code"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Error.Message != "" {
			return fmt.Errorf("Copilot API error: %s (model: %s)", apiErr.Error.Message, model)
		}
		return fmt.Errorf("Copilot API error (%d): %s", resp.StatusCode, string(body))
	}

	return p.parseSSEStream(resp.Body, callback)
}

// parseSSEStream parses the Server-Sent Events stream from Copilot
func (p *CopilotProvider) parseSSEStream(body io.Reader, callback func(*StreamChunk) error) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var accumulated MessageResponse
	accumulated.Role = "assistant"
	accumulated.Content = []ContentBlock{}

	textContent := ""
	var toolCalls []struct {
		ID   string
		Name string
		Args string
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

		if choice.FinishReason != nil {
			if textContent != "" {
				callback(&StreamChunk{Type: "content_block_stop", Index: 0})
			}
			for i, tc := range toolCalls {
				if tc.ID == "" {
					tc.ID = generateToolCallID()
					toolCalls[i].ID = tc.ID
				}
				if tc.Name == "" {
					continue
				}
				idx := i + 1
				if textContent == "" {
					idx = i
				}
				var input map[string]interface{}
				json.Unmarshal([]byte(tc.Args), &input)
				block := ContentBlock{Type: "tool_use", ID: tc.ID, Name: tc.Name, Input: input}
				callback(&StreamChunk{Type: "content_block_start", Index: idx, ContentBlock: &block})
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

	accumulated.Content = []ContentBlock{}
	if textContent != "" {
		accumulated.Content = append(accumulated.Content, ContentBlock{Type: "text", Text: textContent})
	}
	validToolCalls := 0
	for _, tc := range toolCalls {
		if tc.ID == "" {
			tc.ID = generateToolCallID()
		}
		if tc.Name == "" {
			continue
		}
		var input map[string]interface{}
		json.Unmarshal([]byte(tc.Args), &input)
		accumulated.Content = append(accumulated.Content, ContentBlock{
			Type: "tool_use", ID: tc.ID, Name: tc.Name, Input: input,
		})
		validToolCalls++
	}

	if validToolCalls > 0 {
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
		id := toolCall.ID
		if id == "" {
			id = generateToolCallID()
		}
		if toolCall.Function.Name == "" {
			continue
		}
		var input map[string]interface{}
		json.Unmarshal([]byte(toolCall.Function.Arguments), &input)
		content = append(content, ContentBlock{
			Type: "tool_use", ID: id, Name: toolCall.Function.Name, Input: input,
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

	validToolCallIDs := make(map[string]bool)
	for _, msg := range messages {
		if msg.Role != "assistant" {
			continue
		}
		if blocks, ok := msg.Content.([]ContentBlock); ok {
			for _, block := range blocks {
				if block.Type == "tool_use" && block.ID != "" && block.Name != "" {
					validToolCallIDs[block.ID] = true
				}
			}
		}
	}

	for _, msg := range messages {
		msgMap := map[string]interface{}{"role": msg.Role}
		hasContent := false

		switch content := msg.Content.(type) {
		case string:
			msgMap["content"] = content
			hasContent = true
		case []ContentBlock:
			textParts := []string{}
			var multiContent []map[string]interface{} // for vision messages
			var toolCalls []map[string]interface{}

			for _, block := range content {
				switch block.Type {
				case "text":
					textParts = append(textParts, block.Text)
					multiContent = append(multiContent, map[string]interface{}{
						"type": "text",
						"text": block.Text,
					})
				case "image":
					// OpenAI-compatible image_url content part
					if block.Source != nil {
						var url string
						if block.Source.Type == "base64" && block.Source.Data != "" {
							url = "data:" + block.Source.MediaType + ";base64," + block.Source.Data
						} else if block.Source.URL != "" {
							url = block.Source.URL
						}
						if url != "" {
							multiContent = append(multiContent, map[string]interface{}{
								"type":      "image_url",
								"image_url": map[string]interface{}{"url": url, "detail": "auto"},
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
					toolCalls = append(toolCalls, map[string]interface{}{
						"id":   id,
						"type": "function",
						"function": map[string]interface{}{
							"name":      block.Name,
							"arguments": string(inputJSON),
						},
					})
				case "tool_result":
					if block.ToolUseID == "" {
						continue
					}
					if !validToolCallIDs[block.ToolUseID] {
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
					result = append(result, map[string]interface{}{
						"role": "tool", "content": resultContent, "tool_call_id": block.ToolUseID,
					})
					continue
				}
			}

			// Use array content when images present, plain string otherwise
			hasImages := false
			for _, mc := range multiContent {
				if mc["type"] == "image_url" {
					hasImages = true
					break
				}
			}
			if hasImages {
				msgMap["content"] = multiContent
				hasContent = true
			} else if len(textParts) > 0 {
				msgMap["content"] = strings.Join(textParts, "\n")
				hasContent = true
			}
			if len(toolCalls) > 0 {
				msgMap["tool_calls"] = toolCalls
				hasContent = true
				if _, ok := msgMap["content"]; !ok {
					msgMap["content"] = nil
				}
			}
		}

		if hasContent {
			result = append(result, msgMap)
		}
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

// ── OAuth / Token storage ─────────────────────────────────────────────────────

// getGitHubToken loads the stored Copilot OAuth token. Only tokens from the
// device flow (dcode copilot-login) work; PATs do NOT work.
func getGitHubToken() (string, error) {
	if token, err := loadCopilotOAuthToken(); err == nil && token != "" {
		return token, nil
	}
	return "", fmt.Errorf("no Copilot OAuth token found — use Ctrl+P → 'GitHub Copilot OAuth Login' to authenticate")
}

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

// ── Device Flow OAuth ─────────────────────────────────────────────────────────

// CopilotDeviceCodeResponse holds the result of starting a GitHub device flow
type CopilotDeviceCodeResponse struct {
	DeviceCode      string
	UserCode        string
	VerificationURI string
	Interval        int
	ExpiresIn       int
}

// StartCopilotDeviceFlow requests a GitHub device code for TUI-based auth.
func StartCopilotDeviceFlow() (interface{}, error) {
	client := &http.Client{Timeout: 30 * time.Second}

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
		return nil, fmt.Errorf("failed to request device code: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var deviceResp struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		Interval        int    `json:"interval"`
		ExpiresIn       int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &deviceResp); err != nil {
		return nil, fmt.Errorf("failed to parse device code response: %w", err)
	}

	return &CopilotDeviceCodeResponse{
		DeviceCode:      deviceResp.DeviceCode,
		UserCode:        deviceResp.UserCode,
		VerificationURI: deviceResp.VerificationURI,
		Interval:        deviceResp.Interval,
		ExpiresIn:       deviceResp.ExpiresIn,
	}, nil
}

// PollCopilotDeviceFlow polls GitHub for the access token after user authorization.
func PollCopilotDeviceFlow(deviceCode string, intervalSecs int, expiresIn int) error {
	client := &http.Client{Timeout: 30 * time.Second}
	interval := time.Duration(intervalSecs+1) * time.Second
	deadline := time.Now().Add(time.Duration(expiresIn) * time.Second)

	for time.Now().Before(deadline) {
		time.Sleep(interval)

		tokenBody, _ := json.Marshal(map[string]string{
			"client_id":   copilotClientID,
			"device_code": deviceCode,
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
				return saveCopilotOAuthToken(result.AccessToken)
			}
		default:
			return fmt.Errorf("authentication error: %s", result.Error)
		}
	}
	return fmt.Errorf("authentication timed out")
}

// CopilotLogin performs the GitHub OAuth device flow (CLI use only).
func CopilotLogin() error {
	client := &http.Client{Timeout: 30 * time.Second}

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
