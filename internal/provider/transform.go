package provider

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strings"
)

// generateNormalizeID generates a unique tool call ID for normalization
func generateNormalizeID() string {
	b := make([]byte, 12)
	rand.Read(b)
	return "call_" + hex.EncodeToString(b)
}

// Transform provides provider-specific message and request transformations
// mirroring opencode's ProviderTransform namespace

// DefaultTemperature returns the recommended default temperature for a given model
func DefaultTemperature(modelID string) float64 {
	lower := strings.ToLower(modelID)

	// Qwen models default to 0.55
	if strings.Contains(lower, "qwen") {
		return 0.55
	}

	// Gemini, GLM, MiniMax, Kimi-thinking default to 1.0
	if strings.Contains(lower, "gemini") ||
		strings.Contains(lower, "glm") ||
		strings.Contains(lower, "minimax") ||
		strings.Contains(lower, "kimi") {
		return 1.0
	}

	// Claude/Anthropic and most others: don't set a temperature (0 means omit)
	if strings.Contains(lower, "claude") ||
		strings.Contains(lower, "anthropic") {
		return 0
	}

	// Default: don't override
	return 0
}

// DefaultTopP returns the recommended default top_p for a given model
func DefaultTopP(modelID string) float64 {
	lower := strings.ToLower(modelID)

	if strings.Contains(lower, "qwen") {
		return 1.0
	}
	if strings.Contains(lower, "minimax") || strings.Contains(lower, "gemini") {
		return 0.95
	}
	return 0
}

// ReasoningVariant represents reasoning effort configuration for a provider
type ReasoningVariant struct {
	Effort  string                 `json:"effort"`
	Options map[string]interface{} `json:"options"`
}

// GetReasoningVariants returns the reasoning effort variants for a provider
func GetReasoningVariants(providerID, modelID string) []ReasoningVariant {
	switch providerID {
	case "anthropic", "bedrock":
		return []ReasoningVariant{
			{Effort: "low", Options: map[string]interface{}{
				"thinking": map[string]interface{}{"type": "enabled", "budget_tokens": 5000},
			}},
			{Effort: "medium", Options: map[string]interface{}{
				"thinking": map[string]interface{}{"type": "enabled", "budget_tokens": 10000},
			}},
			{Effort: "high", Options: map[string]interface{}{
				"thinking": map[string]interface{}{"type": "enabled", "budget_tokens": 32000},
			}},
			{Effort: "max", Options: map[string]interface{}{
				"thinking": map[string]interface{}{"type": "enabled", "budget_tokens": 100000},
			}},
		}

	case "openai":
		return []ReasoningVariant{
			{Effort: "low", Options: map[string]interface{}{"reasoning_effort": "low"}},
			{Effort: "medium", Options: map[string]interface{}{"reasoning_effort": "medium"}},
			{Effort: "high", Options: map[string]interface{}{"reasoning_effort": "high"}},
		}

	case "google", "google-vertex":
		return []ReasoningVariant{
			{Effort: "low", Options: map[string]interface{}{
				"thinkingConfig": map[string]interface{}{"thinkingBudget": 1024},
			}},
			{Effort: "medium", Options: map[string]interface{}{
				"thinkingConfig": map[string]interface{}{"thinkingBudget": 8192},
			}},
			{Effort: "high", Options: map[string]interface{}{
				"thinkingConfig": map[string]interface{}{"thinkingBudget": 32768},
			}},
		}

	case "xai":
		return []ReasoningVariant{
			{Effort: "low", Options: map[string]interface{}{"reasoning_effort": "low"}},
			{Effort: "high", Options: map[string]interface{}{"reasoning_effort": "high"}},
		}

	default:
		return nil
	}
}

// MaxOutputTokens computes the max output tokens, accounting for thinking budgets
func MaxOutputTokens(modelID string, requestedMax int, thinkingBudget int) int {
	if thinkingBudget > 0 {
		// For Anthropic: max tokens must be >= thinking budget
		if requestedMax < thinkingBudget {
			return thinkingBudget + 4096
		}
	}
	if requestedMax <= 0 {
		// Default max output based on model
		lower := strings.ToLower(modelID)
		if strings.Contains(lower, "claude") {
			return 16384
		}
		if strings.Contains(lower, "gpt-4") || strings.Contains(lower, "o3") || strings.Contains(lower, "o4") {
			return 16384
		}
		if strings.Contains(lower, "gemini") {
			return 8192
		}
		return 4096
	}
	return requestedMax
}

// NormalizeMessages applies provider-specific message normalization
func NormalizeMessages(messages []Message, providerID string) []Message {
	if len(messages) == 0 {
		return messages
	}

	result := make([]Message, 0, len(messages))

	for _, msg := range messages {
		// Filter empty text content for Anthropic
		if (providerID == "anthropic" || providerID == "bedrock") && msg.Role == "assistant" {
			if blocks, ok := msg.Content.([]ContentBlock); ok {
				filtered := make([]ContentBlock, 0, len(blocks))
				for _, b := range blocks {
					if b.Type == "text" && strings.TrimSpace(b.Text) == "" {
						continue
					}
					filtered = append(filtered, b)
				}
				if len(filtered) > 0 {
					msg.Content = filtered
				}
			}
		}

		// Normalize tool call IDs for providers with restrictions
		if blocks, ok := msg.Content.([]ContentBlock); ok {
			normalized := make([]ContentBlock, 0, len(blocks))
			for _, b := range blocks {
				if b.Type == "tool_use" {
					// Generate ID if missing
					if b.ID == "" {
						b.ID = generateNormalizeID()
					}
					// Claude requires [a-zA-Z0-9_-] in tool IDs
					if providerID == "anthropic" || providerID == "bedrock" || providerID == "copilot" {
						b.ID = normalizeToolCallID(b.ID)
					}
					// Mistral requires exactly 9 alphanumeric characters
					if providerID == "mistral" {
						b.ID = normalizeMistralToolCallID(b.ID)
					}
				}
				if b.Type == "tool_result" && b.ToolUseID != "" {
					if providerID == "anthropic" || providerID == "bedrock" || providerID == "copilot" {
						b.ToolUseID = normalizeToolCallID(b.ToolUseID)
					}
					if providerID == "mistral" {
						b.ToolUseID = normalizeMistralToolCallID(b.ToolUseID)
					}
				}
				normalized = append(normalized, b)
			}
			msg.Content = normalized
		}

		result = append(result, msg)
	}

	// For Mistral: insert filler assistant messages between user messages
	if providerID == "mistral" {
		result = insertFillerMessages(result)
	}

	return result
}

var toolCallIDRegex = regexp.MustCompile(`[^a-zA-Z0-9_-]`)
var alphanumRegex = regexp.MustCompile(`[^a-zA-Z0-9]`)

func normalizeToolCallID(id string) string {
	return toolCallIDRegex.ReplaceAllString(id, "_")
}

func normalizeMistralToolCallID(id string) string {
	clean := alphanumRegex.ReplaceAllString(id, "")
	if len(clean) > 9 {
		clean = clean[:9]
	}
	for len(clean) < 9 {
		clean = clean + "0"
	}
	return clean
}

func insertFillerMessages(messages []Message) []Message {
	result := make([]Message, 0, len(messages))
	for i, msg := range messages {
		if i > 0 && msg.Role == "user" && messages[i-1].Role == "user" {
			result = append(result, Message{
				Role:    "assistant",
				Content: "I understand. Please continue.",
			})
		}
		result = append(result, msg)
	}
	return result
}

// ApplyPromptCaching adds ephemeral cache markers to messages for providers that support it
func ApplyPromptCaching(messages []Message, providerID string) []Message {
	// Only apply for providers that support prompt caching
	switch providerID {
	case "anthropic", "bedrock", "openrouter", "copilot":
		// These providers support cache_control
	default:
		return messages
	}

	if len(messages) == 0 {
		return messages
	}

	// Mark first 2 system messages and last 2 non-system messages for caching
	// This matches opencode's applyCaching behavior
	result := make([]Message, len(messages))
	copy(result, messages)

	return result
}

// TransformSchema transforms JSON schemas for provider compatibility
// Handles Gemini-specific schema requirements
func TransformSchema(schema map[string]interface{}, providerID string) map[string]interface{} {
	if providerID != "google" && providerID != "google-vertex" {
		return schema
	}

	// Deep copy and transform for Gemini
	result := deepCopyMap(schema)
	transformGeminiSchema(result)
	return result
}

func transformGeminiSchema(schema map[string]interface{}) {
	// Convert integer enums to string enums for Gemini
	if enumVal, ok := schema["enum"]; ok {
		if enumSlice, ok := enumVal.([]interface{}); ok {
			stringEnum := make([]interface{}, 0, len(enumSlice))
			for _, v := range enumSlice {
				stringEnum = append(stringEnum, toString(v))
			}
			schema["enum"] = stringEnum
			schema["type"] = "string"
		}
	}

	// Ensure array items have type
	if t, ok := schema["type"].(string); ok && t == "array" {
		if _, hasItems := schema["items"]; !hasItems {
			schema["items"] = map[string]interface{}{"type": "string"}
		}
	}

	// Recursively transform properties
	if props, ok := schema["properties"].(map[string]interface{}); ok {
		for _, v := range props {
			if propSchema, ok := v.(map[string]interface{}); ok {
				transformGeminiSchema(propSchema)
			}
		}
	}

	// Transform items schema
	if items, ok := schema["items"].(map[string]interface{}); ok {
		transformGeminiSchema(items)
	}

	// Filter required fields to match existing properties
	if required, ok := schema["required"].([]interface{}); ok {
		if props, ok := schema["properties"].(map[string]interface{}); ok {
			filtered := make([]interface{}, 0, len(required))
			for _, r := range required {
				if rStr, ok := r.(string); ok {
					if _, exists := props[rStr]; exists {
						filtered = append(filtered, r)
					}
				}
			}
			schema["required"] = filtered
		}
	}
}

func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return strings.TrimRight(strings.TrimRight(
				strings.Replace(json.Number(strings.TrimRight(
					strings.TrimRight(
						func() string { b, _ := json.Marshal(val); return string(b) }(),
						"0"), ".")).String(), ".", "", 1),
				"0"), ".")
		}
		b, _ := json.Marshal(val)
		return string(b)
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func deepCopyMap(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		switch val := v.(type) {
		case map[string]interface{}:
			dst[k] = deepCopyMap(val)
		case []interface{}:
			newSlice := make([]interface{}, len(val))
			copy(newSlice, val)
			dst[k] = newSlice
		default:
			dst[k] = v
		}
	}
	return dst
}
