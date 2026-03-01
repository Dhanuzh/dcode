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

	"github.com/Dhanuzh/dcode/internal/config"
)

// AnthropicProvider implements the Provider interface for Anthropic Claude.
// It always uses an OAuth Bearer token — API keys are not supported.
type AnthropicProvider struct {
	oauthToken string
	baseURL    string
	client     *http.Client
}

// NewAnthropicProviderOAuth creates a new Anthropic provider using an OAuth Bearer token.
func NewAnthropicProviderOAuth(accessToken string) *AnthropicProvider {
	return &AnthropicProvider{
		oauthToken: accessToken,
		baseURL:    "https://api.anthropic.com/v1",
		client:     &http.Client{},
	}
}

// setAuthHeaders sets OAuth authentication headers on every request.
func (p *AnthropicProvider) setAuthHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+p.oauthToken)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("user-agent", "claude-cli/2.1.2 (external, cli)")
}

func (p *AnthropicProvider) Name() string { return "anthropic" }

func (p *AnthropicProvider) Models() []string {
	return []string{
		"claude-opus-4-6",
		"claude-sonnet-4-5",
		"claude-sonnet-4-5-20250929",
		"claude-opus-4-5",
		"claude-opus-4-5-20251101",
		"claude-opus-4-1",
		"claude-opus-4-1-20250805",
		"claude-opus-4-0",
		"claude-opus-4-20250514",
		"claude-sonnet-4-0",
		"claude-sonnet-4-20250514",
		"claude-haiku-4-5",
		"claude-haiku-4-5-20251001",
		"claude-3-7-sonnet-latest",
		"claude-3-7-sonnet-20250219",
		"claude-3-5-sonnet-20241022",
		"claude-3-5-sonnet-20240620",
		"claude-3-5-haiku-latest",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
	}
}

// CreateMessage sends a message to Claude and returns the response
func (p *AnthropicProvider) CreateMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error) {
	reqBody := map[string]interface{}{
		"model":      req.Model,
		"max_tokens": req.MaxTokens,
		"messages":   p.convertMessages(req.Messages),
	}

	if req.System != "" {
		reqBody["system"] = req.System
	}
	if req.Temperature != 0 {
		reqBody["temperature"] = req.Temperature
	}
	if len(req.Tools) > 0 {
		reqBody["tools"] = p.convertTools(req.Tools)
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	p.setAuthHeaders(httpReq)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

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
		return nil, p.parseAPIError(resp.StatusCode, body)
	}

	var apiResp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Role    string `json:"role"`
		Content []struct {
			Type  string                 `json:"type"`
			Text  string                 `json:"text,omitempty"`
			ID    string                 `json:"id,omitempty"`
			Name  string                 `json:"name,omitempty"`
			Input map[string]interface{} `json:"input,omitempty"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens       int `json:"input_tokens"`
			OutputTokens      int `json:"output_tokens"`
			CacheReadTokens   int `json:"cache_read_input_tokens"`
			CacheCreateTokens int `json:"cache_creation_input_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	content := make([]ContentBlock, len(apiResp.Content))
	for i, block := range apiResp.Content {
		content[i] = ContentBlock{
			Type:  block.Type,
			Text:  block.Text,
			ID:    block.ID,
			Name:  block.Name,
			Input: block.Input,
		}
	}

	return &MessageResponse{
		ID:         apiResp.ID,
		Model:      apiResp.Model,
		Role:       apiResp.Role,
		Content:    content,
		StopReason: apiResp.StopReason,
		Usage: Usage{
			InputTokens:       apiResp.Usage.InputTokens,
			OutputTokens:      apiResp.Usage.OutputTokens,
			CacheReadTokens:   apiResp.Usage.CacheReadTokens,
			CacheCreateTokens: apiResp.Usage.CacheCreateTokens,
		},
	}, nil
}

// StreamMessage streams a response from Claude using SSE
func (p *AnthropicProvider) StreamMessage(ctx context.Context, req *MessageRequest, callback func(*StreamChunk) error) error {
	reqBody := map[string]interface{}{
		"model":      req.Model,
		"max_tokens": req.MaxTokens,
		"messages":   p.convertMessages(req.Messages),
		"stream":     true,
	}

	if req.System != "" {
		reqBody["system"] = req.System
	}
	if req.Temperature != 0 {
		reqBody["temperature"] = req.Temperature
	}
	if len(req.Tools) > 0 {
		reqBody["tools"] = p.convertTools(req.Tools)
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	p.setAuthHeaders(httpReq)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return p.parseAPIError(resp.StatusCode, body)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event struct {
			Type         string           `json:"type"`
			Index        int              `json:"index"`
			Message      *json.RawMessage `json:"message,omitempty"`
			ContentBlock *json.RawMessage `json:"content_block,omitempty"`
			Delta        *json.RawMessage `json:"delta,omitempty"`
		}

		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		chunk := &StreamChunk{
			Type:  event.Type,
			Index: event.Index,
		}

		switch event.Type {
		case "message_start":
			if event.Message != nil {
				var msg MessageResponse
				json.Unmarshal(*event.Message, &msg)
				chunk.Message = &msg
			}
		case "content_block_start":
			if event.ContentBlock != nil {
				var block ContentBlock
				json.Unmarshal(*event.ContentBlock, &block)
				chunk.ContentBlock = &block
			}
		case "content_block_delta":
			if event.Delta != nil {
				var delta Delta
				json.Unmarshal(*event.Delta, &delta)
				chunk.Delta = &delta
			}
		}

		if err := callback(chunk); err != nil {
			return err
		}
	}

	// Send final stop
	callback(&StreamChunk{Type: "message_stop"})
	return scanner.Err()
}

// AnthropicLogin performs the PKCE OAuth flow for Anthropic Claude Pro/Max.
// Opens the browser, waits for the user to paste back the redirect URL code,
// exchanges it for tokens, and saves them.
func AnthropicLogin() error {
	cyan := "\033[36m"
	green := "\033[32m"
	yellow := "\033[33m"
	gray := "\033[90m"
	reset := "\033[0m"
	bold := "\033[1m"

	fmt.Println()
	fmt.Println(cyan + bold + "╭──────────────────────────────────────────────╮" + reset)
	fmt.Println(cyan + bold + "│       Anthropic Claude OAuth Login           │" + reset)
	fmt.Println(cyan + bold + "╰──────────────────────────────────────────────╯" + reset)
	fmt.Println()
	fmt.Println(gray + "  Requires a Claude Pro or Max subscription." + reset)
	fmt.Println()

	result, err := AnthropicOAuthAuthorize()
	if err != nil {
		return fmt.Errorf("failed to build OAuth URL: %w", err)
	}

	fmt.Println(gray + "  1. Open this URL in your browser:" + reset)
	fmt.Println()
	fmt.Println("     " + yellow + result.URL + reset)
	fmt.Println()

	// Best-effort browser open
	_ = config.OpenBrowser(result.URL)

	fmt.Println(gray + "  2. Authorize the app, then paste the redirect URL" + reset)
	fmt.Println(gray + "     (it looks like: https://console.anthropic.com/oauth/code/callback?code=...#...)" + reset)
	fmt.Println()
	fmt.Print(yellow + "  Paste redirect URL" + reset + " " + gray + "(Enter to cancel): " + reset)

	reader := bufio.NewReader(os.Stdin)
	redirectURL, _ := reader.ReadString('\n')
	redirectURL = strings.TrimSpace(redirectURL)
	if redirectURL == "" {
		fmt.Println(gray + "  → Cancelled" + reset)
		return nil
	}

	// Extract code#state from the URL
	codeAndState := extractAnthropicCodeFromURL(redirectURL)
	if codeAndState == "" {
		return fmt.Errorf("could not find code in URL — expected ?code=...#<state>")
	}

	fmt.Println(gray + "  Exchanging code for tokens..." + reset)
	token, err := AnthropicOAuthExchange(codeAndState, result.Verifier)
	if err != nil {
		return fmt.Errorf("token exchange failed: %w", err)
	}

	if err := SaveAnthropicOAuthToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	if err := config.SaveDefaultProvider("anthropic"); err != nil {
		fmt.Println(yellow + "  ⚠ Could not save default provider: " + err.Error() + reset)
	}

	path, _ := config.GetCredentialsPath()
	fmt.Println()
	fmt.Println("  " + green + "✓ Anthropic OAuth token saved" + reset)
	fmt.Println("  " + gray + path + reset)
	fmt.Println()
	fmt.Println("  " + yellow + "You can now run " + cyan + bold + "dcode" + reset + yellow + " to start coding!" + reset)
	fmt.Println()
	return nil
}

// extractAnthropicCodeFromURL pulls "code#state" out of a redirect URL like:
// https://console.anthropic.com/oauth/code/callback?code=ABC&state=XYZ#XYZ
// or the raw "ABC#XYZ" string if the user pasted just the code fragment.
func extractAnthropicCodeFromURL(raw string) string {
	// If it looks like a URL, parse query params
	if strings.Contains(raw, "?") || strings.Contains(raw, "://") {
		// Find ?code=...
		codeIdx := strings.Index(raw, "?code=")
		if codeIdx == -1 {
			codeIdx = strings.Index(raw, "&code=")
		}
		if codeIdx != -1 {
			// Advance past "?code=" or "&code="
			rest := raw[codeIdx+6:]
			// code ends at next & or #
			end := strings.IndexAny(rest, "&#")
			var code string
			if end == -1 {
				code = rest
			} else {
				code = rest[:end]
			}
			// Now find the state fragment (after #)
			hashIdx := strings.LastIndex(raw, "#")
			if hashIdx != -1 && hashIdx < len(raw)-1 {
				state := raw[hashIdx+1:]
				return code + "#" + state
			}
			return code
		}
	}
	// Assume user pasted "code#state" directly
	return raw
}

func (p *AnthropicProvider) convertMessages(messages []Message) []map[string]interface{} {
	result := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		msgMap := map[string]interface{}{
			"role": msg.Role,
		}

		switch content := msg.Content.(type) {
		case string:
			msgMap["content"] = content
		case []ContentBlock:
			blocks := make([]map[string]interface{}, 0, len(content))
			for _, block := range content {
				blockMap := map[string]interface{}{
					"type": block.Type,
				}
				switch block.Type {
				case "image":
					// Anthropic image format:
					// {"type":"image","source":{"type":"base64","media_type":"image/png","data":"..."}}
					if block.Source != nil {
						src := map[string]interface{}{
							"type": block.Source.Type,
						}
						if block.Source.MediaType != "" {
							src["media_type"] = block.Source.MediaType
						}
						if block.Source.Data != "" {
							src["data"] = block.Source.Data
						}
						if block.Source.URL != "" {
							src["url"] = block.Source.URL
						}
						blockMap["source"] = src
					}
				default:
					if block.Text != "" {
						blockMap["text"] = block.Text
					}
					if block.ID != "" {
						blockMap["id"] = block.ID
					}
					if block.Name != "" {
						blockMap["name"] = block.Name
					}
					if block.Input != nil {
						blockMap["input"] = block.Input
					}
					if block.ToolUseID != "" {
						blockMap["tool_use_id"] = block.ToolUseID
					}
					if block.Content != nil {
						blockMap["content"] = block.Content
					}
					if block.IsError {
						blockMap["is_error"] = block.IsError
					}
				}
				blocks = append(blocks, blockMap)
			}
			msgMap["content"] = blocks
		}

		result[i] = msgMap
	}
	return result
}

// parseAPIError parses an Anthropic API error response and returns a ClassifiedError
func (p *AnthropicProvider) parseAPIError(statusCode int, body []byte) error {
	// Try to parse Anthropic's structured error format:
	// {"type":"error","error":{"type":"...","message":"..."}}
	var apiErr struct {
		Type  string `json:"type"`
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}

	errMsg := string(body)
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error.Message != "" {
		errMsg = apiErr.Error.Message
	}

	rawErr := fmt.Errorf("%s", errMsg)

	return ClassifyError(rawErr, statusCode, errMsg)
}

func (p *AnthropicProvider) convertTools(tools []Tool) []map[string]interface{} {
	result := make([]map[string]interface{}, len(tools))
	for i, tool := range tools {
		result[i] = map[string]interface{}{
			"name":         tool.Name,
			"description":  tool.Description,
			"input_schema": tool.InputSchema,
		}
	}
	return result
}
