package provider

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// BedrockProvider uses AWS Bedrock with native Anthropic Messages API and proper AWS SigV4 auth
type BedrockProvider struct {
	region    string
	accessKey string
	secretKey string
	token     string // session token for temporary credentials
	profile   string
	client    *http.Client
}

// NewBedrockProvider creates a new AWS Bedrock provider with auto-detected credentials
func NewBedrockProvider(region string) *BedrockProvider {
	if region == "" {
		region = os.Getenv("AWS_DEFAULT_REGION")
		if region == "" {
			region = os.Getenv("AWS_REGION")
			if region == "" {
				region = "us-east-1"
			}
		}
	}

	p := &BedrockProvider{
		region: region,
		client: &http.Client{Timeout: 120 * time.Second},
	}

	// Auto-detect AWS credentials from environment
	p.accessKey = os.Getenv("AWS_ACCESS_KEY_ID")
	p.secretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	p.token = os.Getenv("AWS_SESSION_TOKEN")
	p.profile = os.Getenv("AWS_PROFILE")

	return p
}

func (p *BedrockProvider) Name() string {
	return "bedrock"
}

func (p *BedrockProvider) Models() []string {
	return []string{
		// Claude 4.x
		"anthropic.claude-opus-4-6-v1",
		"anthropic.claude-sonnet-4-5-20250929-v1:0",
		"anthropic.claude-opus-4-5-20251101-v1:0",
		"anthropic.claude-opus-4-1-20250805-v1:0",
		"anthropic.claude-sonnet-4-20250514-v1:0",
		"anthropic.claude-opus-4-20250514-v1:0",
		"anthropic.claude-haiku-4-5-20251001-v1:0",
		// Claude 3.x
		"anthropic.claude-3-7-sonnet-20250219-v1:0",
		"anthropic.claude-3-5-sonnet-20241022-v2:0",
		"anthropic.claude-3-5-haiku-20241022-v1:0",
		"anthropic.claude-3-opus-20240229-v1:0",
		// US cross-region
		"us.anthropic.claude-opus-4-6-v1",
		"us.anthropic.claude-sonnet-4-5-20250929-v1:0",
		"us.anthropic.claude-opus-4-5-20251101-v1:0",
		"us.anthropic.claude-opus-4-1-20250805-v1:0",
		"us.anthropic.claude-sonnet-4-20250514-v1:0",
		"us.anthropic.claude-opus-4-20250514-v1:0",
		// EU cross-region
		"eu.anthropic.claude-sonnet-4-20250514-v1:0",
		"eu.anthropic.claude-3-5-sonnet-20241022-v2:0",
	}
}

func (p *BedrockProvider) CreateMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error) {
	modelID := req.Model

	// Build Anthropic Messages API body
	body := p.buildRequestBody(req)
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com/model/%s/invoke", p.region, modelID)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// Sign request with AWS SigV4
	if err := p.signRequest(httpReq, jsonBody); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("bedrock request failed: %w", err)
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

	return p.parseAnthropicResponse(respBody, req.Model)
}

func (p *BedrockProvider) StreamMessage(ctx context.Context, req *MessageRequest, callback func(*StreamChunk) error) error {
	modelID := req.Model

	body := p.buildRequestBody(req)
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com/model/%s/invoke-with-response-stream", p.region, modelID)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/vnd.amazon.eventstream")

	if err := p.signRequest(httpReq, jsonBody); err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("bedrock stream request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		classified := ClassifyError(fmt.Errorf("%s", string(respBody)), resp.StatusCode, string(respBody))
		return classified
	}

	// Parse SSE-like streaming response
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") && !strings.HasPrefix(line, "{") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}

		var event map[string]interface{}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		// Handle Bedrock event stream wrapping
		if bytes, ok := event["bytes"]; ok {
			if bytesStr, ok := bytes.(string); ok {
				data = bytesStr
				if err := json.Unmarshal([]byte(data), &event); err != nil {
					continue
				}
			}
		}

		eventType, _ := event["type"].(string)

		switch eventType {
		case "message_start":
			if err := callback(&StreamChunk{
				Type: "message_start",
				Message: &MessageResponse{
					Role:  "assistant",
					Model: req.Model,
				},
			}); err != nil {
				return err
			}

		case "content_block_start":
			block := &ContentBlock{}
			if cb, ok := event["content_block"].(map[string]interface{}); ok {
				if t, ok := cb["type"].(string); ok {
					block.Type = t
				}
				if t, ok := cb["text"].(string); ok {
					block.Text = t
				}
				if id, ok := cb["id"].(string); ok {
					block.ID = id
				}
				if name, ok := cb["name"].(string); ok {
					block.Name = name
				}
			}
			idx := 0
			if i, ok := event["index"].(float64); ok {
				idx = int(i)
			}
			if err := callback(&StreamChunk{
				Type:         "content_block_start",
				Index:        idx,
				ContentBlock: block,
			}); err != nil {
				return err
			}

		case "content_block_delta":
			delta := &Delta{}
			if d, ok := event["delta"].(map[string]interface{}); ok {
				if t, ok := d["type"].(string); ok {
					delta.Type = t
				}
				if t, ok := d["text"].(string); ok {
					delta.Text = t
				}
				if pj, ok := d["partial_json"].(string); ok {
					delta.PartialJSON = pj
				}
				if r, ok := d["thinking"].(string); ok {
					delta.Reasoning = r
				}
			}
			if err := callback(&StreamChunk{
				Type:  "content_block_delta",
				Delta: delta,
			}); err != nil {
				return err
			}

		case "message_stop", "message_delta":
			usage := Usage{}
			if u, ok := event["usage"].(map[string]interface{}); ok {
				if v, ok := u["input_tokens"].(float64); ok {
					usage.InputTokens = int(v)
				}
				if v, ok := u["output_tokens"].(float64); ok {
					usage.OutputTokens = int(v)
				}
			}
			if err := callback(&StreamChunk{
				Type: eventType,
				Message: &MessageResponse{
					Usage: usage,
				},
			}); err != nil {
				return err
			}
		}
	}

	return scanner.Err()
}

func (p *BedrockProvider) buildRequestBody(req *MessageRequest) map[string]interface{} {
	messages := make([]map[string]interface{}, 0, len(req.Messages))
	for _, msg := range req.Messages {
		m := map[string]interface{}{
			"role": msg.Role,
		}
		switch v := msg.Content.(type) {
		case string:
			m["content"] = v
		case []ContentBlock:
			parts := make([]map[string]interface{}, 0, len(v))
			for _, block := range v {
				part := map[string]interface{}{"type": block.Type}
				switch block.Type {
				case "text":
					part["text"] = block.Text
				case "tool_use":
					part["id"] = block.ID
					part["name"] = block.Name
					part["input"] = block.Input
				case "tool_result":
					part["tool_use_id"] = block.ToolUseID
					part["content"] = block.Content
					if block.IsError {
						part["is_error"] = true
					}
				}
				parts = append(parts, part)
			}
			m["content"] = parts
		default:
			m["content"] = fmt.Sprintf("%v", v)
		}
		messages = append(messages, m)
	}

	body := map[string]interface{}{
		"anthropic_version": "bedrock-2023-05-31",
		"messages":          messages,
		"max_tokens":        req.MaxTokens,
	}
	if req.System != "" {
		body["system"] = req.System
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if len(req.Tools) > 0 {
		tools := make([]map[string]interface{}, 0, len(req.Tools))
		for _, t := range req.Tools {
			tools = append(tools, map[string]interface{}{
				"name":         t.Name,
				"description":  t.Description,
				"input_schema": t.InputSchema,
			})
		}
		body["tools"] = tools
	}
	return body
}

func (p *BedrockProvider) parseAnthropicResponse(body []byte, model string) (*MessageResponse, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	resp := &MessageResponse{
		Model: model,
		Role:  "assistant",
	}

	if id, ok := raw["id"].(string); ok {
		resp.ID = id
	}
	if sr, ok := raw["stop_reason"].(string); ok {
		resp.StopReason = sr
	}

	if content, ok := raw["content"].([]interface{}); ok {
		for _, c := range content {
			if block, ok := c.(map[string]interface{}); ok {
				cb := ContentBlock{}
				if t, ok := block["type"].(string); ok {
					cb.Type = t
				}
				switch cb.Type {
				case "text":
					if text, ok := block["text"].(string); ok {
						cb.Text = text
					}
				case "tool_use":
					if id, ok := block["id"].(string); ok {
						cb.ID = id
					}
					if name, ok := block["name"].(string); ok {
						cb.Name = name
					}
					if input, ok := block["input"].(map[string]interface{}); ok {
						cb.Input = input
					}
				}
				resp.Content = append(resp.Content, cb)
			}
		}
	}

	if usage, ok := raw["usage"].(map[string]interface{}); ok {
		if v, ok := usage["input_tokens"].(float64); ok {
			resp.Usage.InputTokens = int(v)
		}
		if v, ok := usage["output_tokens"].(float64); ok {
			resp.Usage.OutputTokens = int(v)
		}
	}

	return resp, nil
}

// signRequest signs an HTTP request using AWS Signature V4
func (p *BedrockProvider) signRequest(req *http.Request, payload []byte) error {
	if p.accessKey == "" || p.secretKey == "" {
		return fmt.Errorf("AWS credentials not configured. Set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables")
	}

	now := time.Now().UTC()
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")

	req.Header.Set("X-Amz-Date", amzDate)
	if p.token != "" {
		req.Header.Set("X-Amz-Security-Token", p.token)
	}

	// Canonical request
	payloadHash := sha256Hex(payload)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)

	canonicalHeaders, signedHeaders := p.buildCanonicalHeaders(req)

	canonicalRequest := strings.Join([]string{
		req.Method,
		req.URL.Path,
		req.URL.RawQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	// String to sign
	service := "bedrock"
	credentialScope := dateStamp + "/" + p.region + "/" + service + "/aws4_request"
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	// Signing key
	signingKey := p.deriveSigningKey(dateStamp, service)

	// Signature
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	// Authorization header
	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		p.accessKey, credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)

	return nil
}

func (p *BedrockProvider) buildCanonicalHeaders(req *http.Request) (string, string) {
	headers := make(map[string]string)
	headers["host"] = req.URL.Host
	for key := range req.Header {
		lk := strings.ToLower(key)
		if lk == "content-type" || lk == "host" || strings.HasPrefix(lk, "x-amz-") {
			headers[lk] = strings.TrimSpace(req.Header.Get(key))
		}
	}

	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var canonical strings.Builder
	for _, k := range keys {
		canonical.WriteString(k + ":" + headers[k] + "\n")
	}

	return canonical.String(), strings.Join(keys, ";")
}

func (p *BedrockProvider) deriveSigningKey(dateStamp, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+p.secretKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(p.region))
	kService := hmacSHA256(kRegion, []byte(service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}
