package provider

import (
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

// ModelCapabilities describes what a model can do
type ModelCapabilities struct {
	Temperature bool          `json:"temperature"`
	Reasoning   bool          `json:"reasoning"`
	Attachment  bool          `json:"attachment"`
	ToolCall    bool          `json:"toolcall"`
	Input       ModalityFlags `json:"input"`
	Output      ModalityFlags `json:"output"`
	Interleaved bool          `json:"interleaved,omitempty"`
}

// ModalityFlags tracks input/output modality support
type ModalityFlags struct {
	Text  bool `json:"text"`
	Audio bool `json:"audio"`
	Image bool `json:"image"`
	Video bool `json:"video"`
	PDF   bool `json:"pdf"`
}

// ModelCost tracks pricing per token
type ModelCost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cache_read,omitempty"`
	CacheWrite float64 `json:"cache_write,omitempty"`
}

// ModelLimits tracks context and output limits
type ModelLimits struct {
	Context int `json:"context"`
	Input   int `json:"input,omitempty"`
	Output  int `json:"output"`
}

// ModelStatus represents the lifecycle status of a model
type ModelStatus string

const (
	ModelStatusActive     ModelStatus = "active"
	ModelStatusBeta       ModelStatus = "beta"
	ModelStatusAlpha      ModelStatus = "alpha"
	ModelStatusDeprecated ModelStatus = "deprecated"
)

// ModelInfo contains comprehensive information about a model
type ModelInfo struct {
	ID           string            `json:"id"`
	ProviderID   string            `json:"provider_id"`
	Name         string            `json:"name"`
	Family       string            `json:"family,omitempty"`
	Capabilities ModelCapabilities `json:"capabilities"`
	Cost         ModelCost         `json:"cost"`
	Limits       ModelLimits       `json:"limits"`
	Status       ModelStatus       `json:"status"`
	ReleaseDate  string            `json:"release_date,omitempty"`
	Options      map[string]string `json:"options,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
}

// ProviderInfo contains information about a provider
type ProviderInfo struct {
	ID     string      `json:"id"`
	Name   string      `json:"name"`
	Models []ModelInfo `json:"models"`
	EnvVar []string    `json:"env,omitempty"` // Environment variable names for API key
}

// ModelRegistry manages the dynamic model catalog
type ModelRegistry struct {
	mu        sync.RWMutex
	providers map[string]*ProviderInfo
	lastFetch time.Time
	cacheFile string
}

// NewModelRegistry creates a new model registry with built-in models
func NewModelRegistry() *ModelRegistry {
	home, _ := os.UserHomeDir()
	cacheFile := ""
	if home != "" {
		cacheFile = filepath.Join(home, ".config", "dcode", "models_cache.json")
	}

	mr := &ModelRegistry{
		providers: make(map[string]*ProviderInfo),
		cacheFile: cacheFile,
	}

	// Load built-in models
	mr.loadBuiltinModels()

	// Try to load from cache
	mr.loadFromCache()

	return mr
}

// GetProvider returns provider info by ID
func (mr *ModelRegistry) GetProvider(id string) *ProviderInfo {
	mr.mu.RLock()
	defer mr.mu.RUnlock()
	return mr.providers[id]
}

// GetModel returns a model by provider and model ID
func (mr *ModelRegistry) GetModel(providerID, modelID string) *ModelInfo {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	prov, ok := mr.providers[providerID]
	if !ok {
		return nil
	}

	for _, m := range prov.Models {
		if m.ID == modelID {
			return &m
		}
	}
	return nil
}

// ListProviders returns all provider IDs
func (mr *ModelRegistry) ListProviders() []string {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	ids := make([]string, 0, len(mr.providers))
	for id := range mr.providers {
		ids = append(ids, id)
	}
	return ids
}

// ListModels returns all models for a provider
func (mr *ModelRegistry) ListModels(providerID string) []ModelInfo {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	prov, ok := mr.providers[providerID]
	if !ok {
		return nil
	}
	return prov.Models
}

// GetSmallModel returns the best small/fast model for a provider
func (mr *ModelRegistry) GetSmallModel(providerID string) *ModelInfo {
	// Priority order for small models per provider
	smallModels := map[string][]string{
		"anthropic":  {"claude-haiku-4-20250414", "claude-3-5-haiku-20241022"},
		"openai":     {"gpt-4.1-mini", "gpt-4.1-nano", "gpt-4o-mini"},
		"copilot":    {"gpt-4.1-mini", "gpt-4.1-nano", "gpt-4o-mini"},
		"google":     {"gemini-2.0-flash-lite", "gemini-2.0-flash", "gemini-2.5-flash"},
		"groq":       {"llama-3.1-8b-instant", "gemma2-9b-it"},
		"deepseek":   {"deepseek-chat"},
		"mistral":    {"mistral-small-latest"},
		"xai":        {"grok-3-mini"},
		"cerebras":   {"llama-3.3-70b"},
		"openrouter": {"anthropic/claude-3-5-haiku", "openai/gpt-4o-mini"},
	}

	if candidates, ok := smallModels[providerID]; ok {
		for _, modelID := range candidates {
			if m := mr.GetModel(providerID, modelID); m != nil {
				return m
			}
		}
	}

	// Fallback: return first model
	models := mr.ListModels(providerID)
	if len(models) > 0 {
		return &models[0]
	}
	return nil
}

// GetDefaultModel returns the best default model for a provider
func (mr *ModelRegistry) GetDefaultModel(providerID string) *ModelInfo {
	defaultModels := map[string][]string{
		"anthropic":     {"claude-sonnet-4-20250514", "claude-opus-4-20250514"},
		"openai":        {"gpt-4.1", "gpt-4o"},
		"copilot":       {"claude-sonnet-4-20250514", "gpt-4.1", "gpt-4o"},
		"google":        {"gemini-2.5-flash", "gemini-2.5-pro"},
		"groq":          {"llama-3.3-70b-versatile"},
		"deepseek":      {"deepseek-chat", "deepseek-reasoner"},
		"mistral":       {"mistral-large-latest"},
		"xai":           {"grok-3", "grok-2"},
		"cerebras":      {"llama-3.3-70b"},
		"together":      {"meta-llama/Llama-3.3-70B-Instruct-Turbo"},
		"openrouter":    {"anthropic/claude-sonnet-4-20250514"},
		"perplexity":    {"sonar-pro"},
		"azure":         {"gpt-4o"},
		"bedrock":       {"anthropic.claude-3-5-sonnet-20241022-v2:0"},
		"deepinfra":     {"meta-llama/Llama-3.3-70B-Instruct-Turbo"},
		"google-vertex": {"gemini-2.5-flash"},
		"gitlab":        {"claude-sonnet-4-20250514"},
	}

	if candidates, ok := defaultModels[providerID]; ok {
		for _, modelID := range candidates {
			if m := mr.GetModel(providerID, modelID); m != nil {
				return m
			}
		}
	}

	models := mr.ListModels(providerID)
	if len(models) > 0 {
		return &models[0]
	}
	return nil
}

// mergeModelsDevData parses a models.dev API response and merges providers/models
// into the registry. Must be called with mr.mu held (write lock).
//
// The models.dev API returns a flat map: {"anthropic": {...}, "openai": {...}, ...}
// Each provider has: id, name, env ([]string), npm, api (base URL), models (map).
// Each model has flat fields: id, name, family, reasoning, tool_call, attachment,
// temperature, modalities ({input: ["text","image"], output: ["text"]}),
// cost ({input, output, cache_read, cache_write}), limit ({context, output}).
func (mr *ModelRegistry) mergeModelsDevData(modelsDevData map[string]interface{}) {
	// The API returns a flat map where top-level keys are provider IDs.
	for provID, provData := range modelsDevData {
		provMap, ok := provData.(map[string]interface{})
		if !ok {
			continue
		}

		provInfo := mr.providers[provID]
		if provInfo == nil {
			provInfo = &ProviderInfo{
				ID:   provID,
				Name: provID,
			}
		}

		if name, ok := provMap["name"].(string); ok {
			provInfo.Name = name
		}

		// Extract env vars (API key environment variable names)
		if envArr, ok := provMap["env"].([]interface{}); ok {
			envVars := make([]string, 0, len(envArr))
			for _, e := range envArr {
				if s, ok := e.(string); ok {
					envVars = append(envVars, s)
				}
			}
			if len(envVars) > 0 {
				provInfo.EnvVar = envVars
			}
		}

		// Models are a map: {"model-id": {...}, ...} (not an array)
		if models, ok := provMap["models"].(map[string]interface{}); ok {
			for _, modelData := range models {
				modelMap, ok := modelData.(map[string]interface{})
				if !ok {
					continue
				}

				model := parseModelsDevModel(provID, modelMap)
				if model.ID == "" {
					continue
				}

				// Skip deprecated models
				if model.Status == ModelStatusDeprecated {
					continue
				}

				// Update or add model
				found := false
				for i, existing := range provInfo.Models {
					if existing.ID == model.ID {
						provInfo.Models[i] = model
						found = true
						break
					}
				}
				if !found {
					provInfo.Models = append(provInfo.Models, model)
				}
			}
		}

		mr.providers[provID] = provInfo
	}
}

// Refresh fetches the latest model catalog from models.dev
func (mr *ModelRegistry) Refresh() error {
	mr.mu.Lock()
	// Don't refresh more than once per hour
	if time.Since(mr.lastFetch) < time.Hour {
		mr.mu.Unlock()
		return nil
	}
	mr.mu.Unlock()

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get("https://models.dev/api.json")
	if err != nil {
		return fmt.Errorf("failed to fetch models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("models.dev returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read models response: %w", err)
	}

	var modelsDevData map[string]interface{}
	if err := json.Unmarshal(body, &modelsDevData); err != nil {
		return fmt.Errorf("failed to parse models: %w", err)
	}

	mr.mu.Lock()
	defer mr.mu.Unlock()

	mr.mergeModelsDevData(modelsDevData)
	mr.lastFetch = time.Now()

	// Save to cache
	mr.saveToCache(body)

	return nil
}

// RefreshBackground runs Refresh in a goroutine. onDone is called (on a
// separate goroutine) after the refresh completes, whether it succeeded or not.
func (mr *ModelRegistry) RefreshBackground(onDone func()) {
	go func() {
		_ = mr.Refresh()
		if onDone != nil {
			onDone()
		}
	}()
}

// parseModelsDevModel converts a models.dev model entry to ModelInfo.
// The models.dev API returns flat fields for capabilities (reasoning, tool_call,
// attachment, temperature) and modalities as string arrays.
func parseModelsDevModel(providerID string, data map[string]interface{}) ModelInfo {
	model := ModelInfo{
		ProviderID: providerID,
		Status:     ModelStatusActive,
	}

	if id, ok := data["id"].(string); ok {
		model.ID = id
	}
	if name, ok := data["name"].(string); ok {
		model.Name = name
	}
	if family, ok := data["family"].(string); ok {
		model.Family = family
	}
	if status, ok := data["status"].(string); ok {
		model.Status = ModelStatus(status)
	}
	if date, ok := data["release_date"].(string); ok {
		model.ReleaseDate = date
	}

	// Parse capabilities – flat top-level fields in models.dev API
	if v, ok := data["temperature"].(bool); ok {
		model.Capabilities.Temperature = v
	}
	if v, ok := data["reasoning"].(bool); ok {
		model.Capabilities.Reasoning = v
	}
	if v, ok := data["attachment"].(bool); ok {
		model.Capabilities.Attachment = v
	}
	if v, ok := data["tool_call"].(bool); ok {
		model.Capabilities.ToolCall = v
	}

	// Parse modalities: {"input": ["text","image"], "output": ["text"]}
	if modalities, ok := data["modalities"].(map[string]interface{}); ok {
		if inputArr, ok := modalities["input"].([]interface{}); ok {
			model.Capabilities.Input = parseModalityFromArray(inputArr)
		}
		if outputArr, ok := modalities["output"].([]interface{}); ok {
			model.Capabilities.Output = parseModalityFromArray(outputArr)
		}
	}

	// Parse interleaved
	if v, ok := data["interleaved"]; ok {
		switch val := v.(type) {
		case bool:
			model.Capabilities.Interleaved = val
		case map[string]interface{}:
			// Has an interleaved config object (e.g. {"field": "reasoning_content"})
			model.Capabilities.Interleaved = true
		}
	}

	// Parse cost – flat fields: input, output, cache_read, cache_write
	if cost, ok := data["cost"].(map[string]interface{}); ok {
		if v, ok := cost["input"].(float64); ok {
			model.Cost.Input = v
		}
		if v, ok := cost["output"].(float64); ok {
			model.Cost.Output = v
		}
		if v, ok := cost["cache_read"].(float64); ok {
			model.Cost.CacheRead = v
		}
		if v, ok := cost["cache_write"].(float64); ok {
			model.Cost.CacheWrite = v
		}
	}

	// Parse limits
	if limits, ok := data["limit"].(map[string]interface{}); ok {
		if v, ok := limits["context"].(float64); ok {
			model.Limits.Context = int(v)
		}
		if v, ok := limits["input"].(float64); ok {
			model.Limits.Input = int(v)
		}
		if v, ok := limits["output"].(float64); ok {
			model.Limits.Output = int(v)
		}
	}

	return model
}

// parseModalityFromArray converts a string array like ["text","image"] to ModalityFlags.
func parseModalityFromArray(arr []interface{}) ModalityFlags {
	flags := ModalityFlags{}
	for _, v := range arr {
		if s, ok := v.(string); ok {
			switch s {
			case "text":
				flags.Text = true
			case "audio":
				flags.Audio = true
			case "image":
				flags.Image = true
			case "video":
				flags.Video = true
			case "pdf":
				flags.PDF = true
			}
		}
	}
	return flags
}

// loadFromCache loads cached model data and merges it into the registry.
func (mr *ModelRegistry) loadFromCache() {
	if mr.cacheFile == "" {
		return
	}

	data, err := os.ReadFile(mr.cacheFile)
	if err != nil {
		return
	}

	var cached struct {
		FetchedAt time.Time              `json:"fetched_at"`
		Data      map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(data, &cached); err != nil {
		return
	}

	// Only use cache if less than 24 hours old
	if time.Since(cached.FetchedAt) > 24*time.Hour {
		return
	}

	mr.mu.Lock()
	defer mr.mu.Unlock()

	// Merge cached provider/model data into the registry
	if cached.Data != nil {
		mr.mergeModelsDevData(cached.Data)
	}

	mr.lastFetch = cached.FetchedAt
}

// saveToCache saves model data to cache
func (mr *ModelRegistry) saveToCache(rawData []byte) {
	if mr.cacheFile == "" {
		return
	}

	dir := filepath.Dir(mr.cacheFile)
	os.MkdirAll(dir, 0755)

	cached := struct {
		FetchedAt time.Time       `json:"fetched_at"`
		Data      json.RawMessage `json:"data"`
	}{
		FetchedAt: time.Now(),
		Data:      rawData,
	}

	data, err := json.Marshal(cached)
	if err != nil {
		return
	}

	os.WriteFile(mr.cacheFile, data, 0644)
}

// loadBuiltinModels populates the registry with hardcoded model definitions
func (mr *ModelRegistry) loadBuiltinModels() {
	mr.providers = map[string]*ProviderInfo{
		"anthropic": {
			ID: "anthropic", Name: "Anthropic",
			EnvVar: []string{"ANTHROPIC_API_KEY"},
			Models: []ModelInfo{
				{ID: "claude-sonnet-4-20250514", ProviderID: "anthropic", Name: "Claude Sonnet 4", Family: "claude",
					Capabilities: ModelCapabilities{Temperature: true, Reasoning: true, Attachment: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true, PDF: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 3.0, Output: 15.0, CacheRead: 0.3, CacheWrite: 3.75},
					Limits: ModelLimits{Context: 200000, Output: 16384}, Status: ModelStatusActive},
				{ID: "claude-opus-4-20250514", ProviderID: "anthropic", Name: "Claude Opus 4", Family: "claude",
					Capabilities: ModelCapabilities{Temperature: true, Reasoning: true, Attachment: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true, PDF: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 15.0, Output: 75.0, CacheRead: 1.5, CacheWrite: 18.75},
					Limits: ModelLimits{Context: 200000, Output: 16384}, Status: ModelStatusActive},
				{ID: "claude-haiku-4-20250414", ProviderID: "anthropic", Name: "Claude Haiku 4", Family: "claude",
					Capabilities: ModelCapabilities{Temperature: true, Reasoning: false, Attachment: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true, PDF: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 0.8, Output: 4.0, CacheRead: 0.08, CacheWrite: 1.0},
					Limits: ModelLimits{Context: 200000, Output: 8192}, Status: ModelStatusActive},
				{ID: "claude-3-7-sonnet-20250219", ProviderID: "anthropic", Name: "Claude 3.7 Sonnet", Family: "claude",
					Capabilities: ModelCapabilities{Temperature: true, Reasoning: true, Attachment: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true, PDF: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 3.0, Output: 15.0, CacheRead: 0.3, CacheWrite: 3.75},
					Limits: ModelLimits{Context: 200000, Output: 16384}, Status: ModelStatusActive},
				{ID: "claude-3-5-haiku-20241022", ProviderID: "anthropic", Name: "Claude 3.5 Haiku", Family: "claude",
					Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
						Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 0.8, Output: 4.0},
					Limits: ModelLimits{Context: 200000, Output: 8192}, Status: ModelStatusActive},
				{ID: "claude-3-5-sonnet-20241022", ProviderID: "anthropic", Name: "Claude 3.5 Sonnet", Family: "claude",
					Capabilities: ModelCapabilities{Temperature: true, Attachment: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 3.0, Output: 15.0, CacheRead: 0.3, CacheWrite: 3.75},
					Limits: ModelLimits{Context: 200000, Output: 8192}, Status: ModelStatusActive},
			},
		},
		"openai": {
			ID: "openai", Name: "OpenAI",
			EnvVar: []string{"OPENAI_API_KEY"},
			Models: []ModelInfo{
				{ID: "gpt-4.1", ProviderID: "openai", Name: "GPT-4.1", Family: "gpt",
					Capabilities: ModelCapabilities{Temperature: true, Attachment: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 2.0, Output: 8.0, CacheRead: 0.5},
					Limits: ModelLimits{Context: 1047576, Output: 32768}, Status: ModelStatusActive},
				{ID: "gpt-4.1-mini", ProviderID: "openai", Name: "GPT-4.1 Mini", Family: "gpt",
					Capabilities: ModelCapabilities{Temperature: true, Attachment: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 0.4, Output: 1.6, CacheRead: 0.1},
					Limits: ModelLimits{Context: 1047576, Output: 32768}, Status: ModelStatusActive},
				{ID: "gpt-4.1-nano", ProviderID: "openai", Name: "GPT-4.1 Nano", Family: "gpt",
					Capabilities: ModelCapabilities{Temperature: true, Attachment: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 0.1, Output: 0.4, CacheRead: 0.025},
					Limits: ModelLimits{Context: 1047576, Output: 32768}, Status: ModelStatusActive},
				{ID: "gpt-4o", ProviderID: "openai", Name: "GPT-4o", Family: "gpt",
					Capabilities: ModelCapabilities{Temperature: true, Attachment: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true, Audio: true}, Output: ModalityFlags{Text: true, Audio: true}},
					Cost:   ModelCost{Input: 2.5, Output: 10.0, CacheRead: 1.25},
					Limits: ModelLimits{Context: 128000, Output: 16384}, Status: ModelStatusActive},
				{ID: "gpt-4o-mini", ProviderID: "openai", Name: "GPT-4o Mini", Family: "gpt",
					Capabilities: ModelCapabilities{Temperature: true, Attachment: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 0.15, Output: 0.6, CacheRead: 0.075},
					Limits: ModelLimits{Context: 128000, Output: 16384}, Status: ModelStatusActive},
				{ID: "o3", ProviderID: "openai", Name: "o3", Family: "o",
					Capabilities: ModelCapabilities{Temperature: false, Reasoning: true, Attachment: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 2.0, Output: 8.0},
					Limits: ModelLimits{Context: 200000, Output: 100000}, Status: ModelStatusActive},
				{ID: "o3-mini", ProviderID: "openai", Name: "o3 Mini", Family: "o",
					Capabilities: ModelCapabilities{Temperature: false, Reasoning: true, ToolCall: true,
						Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 1.1, Output: 4.4},
					Limits: ModelLimits{Context: 200000, Output: 100000}, Status: ModelStatusActive},
				{ID: "o4-mini", ProviderID: "openai", Name: "o4 Mini", Family: "o",
					Capabilities: ModelCapabilities{Temperature: false, Reasoning: true, Attachment: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 1.1, Output: 4.4},
					Limits: ModelLimits{Context: 200000, Output: 100000}, Status: ModelStatusActive},
			},
		},
		"copilot": {
			ID: "copilot", Name: "GitHub Copilot",
			EnvVar: []string{"GITHUB_TOKEN"},
			Models: []ModelInfo{
				{ID: "claude-sonnet-4-20250514", ProviderID: "copilot", Name: "Claude Sonnet 4", Family: "claude",
					Capabilities: ModelCapabilities{Temperature: true, Reasoning: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
					Cost: ModelCost{}, Limits: ModelLimits{Context: 200000, Output: 16384}, Status: ModelStatusActive},
				{ID: "claude-3.5-sonnet", ProviderID: "copilot", Name: "Claude 3.5 Sonnet", Family: "claude",
					Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
					Cost: ModelCost{}, Limits: ModelLimits{Context: 200000, Output: 8192}, Status: ModelStatusActive},
				{ID: "gpt-4.1", ProviderID: "copilot", Name: "GPT-4.1", Family: "gpt",
					Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
					Cost: ModelCost{}, Limits: ModelLimits{Context: 1047576, Output: 32768}, Status: ModelStatusActive},
				{ID: "gpt-4.1-mini", ProviderID: "copilot", Name: "GPT-4.1 Mini", Family: "gpt",
					Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
						Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
					Cost: ModelCost{}, Limits: ModelLimits{Context: 1047576, Output: 32768}, Status: ModelStatusActive},
				{ID: "gpt-4.1-nano", ProviderID: "copilot", Name: "GPT-4.1 Nano", Family: "gpt",
					Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
						Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
					Cost: ModelCost{}, Limits: ModelLimits{Context: 1047576, Output: 32768}, Status: ModelStatusActive},
				{ID: "gpt-4o", ProviderID: "copilot", Name: "GPT-4o", Family: "gpt",
					Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
					Cost: ModelCost{}, Limits: ModelLimits{Context: 128000, Output: 16384}, Status: ModelStatusActive},
				{ID: "gpt-4o-mini", ProviderID: "copilot", Name: "GPT-4o Mini", Family: "gpt",
					Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
						Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
					Cost: ModelCost{}, Limits: ModelLimits{Context: 128000, Output: 16384}, Status: ModelStatusActive},
				{ID: "o3", ProviderID: "copilot", Name: "o3", Family: "o",
					Capabilities: ModelCapabilities{Reasoning: true, ToolCall: true,
						Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
					Cost: ModelCost{}, Limits: ModelLimits{Context: 200000, Output: 100000}, Status: ModelStatusActive},
				{ID: "o3-mini", ProviderID: "copilot", Name: "o3 Mini", Family: "o",
					Capabilities: ModelCapabilities{Reasoning: true, ToolCall: true,
						Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
					Cost: ModelCost{}, Limits: ModelLimits{Context: 200000, Output: 100000}, Status: ModelStatusActive},
				{ID: "o4-mini", ProviderID: "copilot", Name: "o4 Mini", Family: "o",
					Capabilities: ModelCapabilities{Reasoning: true, ToolCall: true,
						Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
					Cost: ModelCost{}, Limits: ModelLimits{Context: 200000, Output: 100000}, Status: ModelStatusActive},
				{ID: "gemini-2.5-pro", ProviderID: "copilot", Name: "Gemini 2.5 Pro", Family: "gemini",
					Capabilities: ModelCapabilities{Temperature: true, Reasoning: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
					Cost: ModelCost{}, Limits: ModelLimits{Context: 1048576, Output: 65536}, Status: ModelStatusActive},
			},
		},
		"google": {
			ID: "google", Name: "Google",
			EnvVar: []string{"GOOGLE_API_KEY", "GEMINI_API_KEY"},
			Models: []ModelInfo{
				{ID: "gemini-2.5-pro", ProviderID: "google", Name: "Gemini 2.5 Pro", Family: "gemini",
					Capabilities: ModelCapabilities{Temperature: true, Reasoning: true, Attachment: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true, Video: true, Audio: true, PDF: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 1.25, Output: 10.0},
					Limits: ModelLimits{Context: 1048576, Output: 65536}, Status: ModelStatusActive},
				{ID: "gemini-2.5-flash", ProviderID: "google", Name: "Gemini 2.5 Flash", Family: "gemini",
					Capabilities: ModelCapabilities{Temperature: true, Reasoning: true, Attachment: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true, Video: true, Audio: true, PDF: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 0.15, Output: 0.6},
					Limits: ModelLimits{Context: 1048576, Output: 65536}, Status: ModelStatusActive},
				{ID: "gemini-2.0-flash", ProviderID: "google", Name: "Gemini 2.0 Flash", Family: "gemini",
					Capabilities: ModelCapabilities{Temperature: true, Attachment: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true, Video: true, Audio: true}, Output: ModalityFlags{Text: true, Image: true}},
					Cost:   ModelCost{Input: 0.1, Output: 0.4},
					Limits: ModelLimits{Context: 1048576, Output: 8192}, Status: ModelStatusActive},
				{ID: "gemini-2.0-flash-lite", ProviderID: "google", Name: "Gemini 2.0 Flash Lite", Family: "gemini",
					Capabilities: ModelCapabilities{Temperature: true, Attachment: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 0.075, Output: 0.3},
					Limits: ModelLimits{Context: 1048576, Output: 8192}, Status: ModelStatusActive},
				{ID: "gemini-1.5-pro", ProviderID: "google", Name: "Gemini 1.5 Pro", Family: "gemini",
					Capabilities: ModelCapabilities{Temperature: true, Attachment: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true, Video: true, Audio: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 1.25, Output: 5.0},
					Limits: ModelLimits{Context: 2097152, Output: 8192}, Status: ModelStatusActive},
			},
		},
		"xai": {
			ID: "xai", Name: "xAI",
			EnvVar: []string{"XAI_API_KEY"},
			Models: []ModelInfo{
				{ID: "grok-3", ProviderID: "xai", Name: "Grok 3", Family: "grok",
					Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 3.0, Output: 15.0},
					Limits: ModelLimits{Context: 131072, Output: 16384}, Status: ModelStatusActive},
				{ID: "grok-3-mini", ProviderID: "xai", Name: "Grok 3 Mini", Family: "grok",
					Capabilities: ModelCapabilities{Temperature: true, Reasoning: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 0.3, Output: 0.5},
					Limits: ModelLimits{Context: 131072, Output: 16384}, Status: ModelStatusActive},
				{ID: "grok-2", ProviderID: "xai", Name: "Grok 2", Family: "grok",
					Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 2.0, Output: 10.0},
					Limits: ModelLimits{Context: 131072, Output: 16384}, Status: ModelStatusActive},
			},
		},
		"deepinfra": {
			ID: "deepinfra", Name: "DeepInfra",
			EnvVar: []string{"DEEPINFRA_API_KEY"},
			Models: []ModelInfo{
				{ID: "meta-llama/Llama-3.3-70B-Instruct-Turbo", ProviderID: "deepinfra", Name: "Llama 3.3 70B", Family: "llama",
					Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
						Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 0.35, Output: 0.4},
					Limits: ModelLimits{Context: 131072, Output: 4096}, Status: ModelStatusActive},
				{ID: "meta-llama/Llama-3.1-405B-Instruct", ProviderID: "deepinfra", Name: "Llama 3.1 405B", Family: "llama",
					Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
						Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 1.8, Output: 1.8},
					Limits: ModelLimits{Context: 131072, Output: 4096}, Status: ModelStatusActive},
				{ID: "Qwen/Qwen2.5-72B-Instruct", ProviderID: "deepinfra", Name: "Qwen 2.5 72B", Family: "qwen",
					Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
						Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 0.35, Output: 0.4},
					Limits: ModelLimits{Context: 32768, Output: 4096}, Status: ModelStatusActive},
			},
		},
		"cerebras": {
			ID: "cerebras", Name: "Cerebras",
			EnvVar: []string{"CEREBRAS_API_KEY"},
			Models: []ModelInfo{
				{ID: "llama-3.3-70b", ProviderID: "cerebras", Name: "Llama 3.3 70B", Family: "llama",
					Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
						Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 0.85, Output: 1.2},
					Limits: ModelLimits{Context: 131072, Output: 8192}, Status: ModelStatusActive},
				{ID: "llama-3.1-8b", ProviderID: "cerebras", Name: "Llama 3.1 8B", Family: "llama",
					Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
						Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 0.1, Output: 0.1},
					Limits: ModelLimits{Context: 131072, Output: 8192}, Status: ModelStatusActive},
			},
		},
		"google-vertex": {
			ID: "google-vertex", Name: "Google Vertex AI",
			EnvVar: []string{"GOOGLE_CLOUD_PROJECT"},
			Models: []ModelInfo{
				{ID: "gemini-2.5-pro", ProviderID: "google-vertex", Name: "Gemini 2.5 Pro", Family: "gemini",
					Capabilities: ModelCapabilities{Temperature: true, Reasoning: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true, Video: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 1.25, Output: 10.0},
					Limits: ModelLimits{Context: 1048576, Output: 65536}, Status: ModelStatusActive},
				{ID: "gemini-2.5-flash", ProviderID: "google-vertex", Name: "Gemini 2.5 Flash", Family: "gemini",
					Capabilities: ModelCapabilities{Temperature: true, Reasoning: true, ToolCall: true,
						Input: ModalityFlags{Text: true, Image: true, Video: true}, Output: ModalityFlags{Text: true}},
					Cost:   ModelCost{Input: 0.15, Output: 0.6},
					Limits: ModelLimits{Context: 1048576, Output: 65536}, Status: ModelStatusActive},
			},
		},
		"gitlab": {
			ID: "gitlab", Name: "GitLab",
			EnvVar: []string{"GITLAB_TOKEN", "GITLAB_API_TOKEN"},
			Models: []ModelInfo{
				{ID: "claude-sonnet-4-20250514", ProviderID: "gitlab", Name: "Claude Sonnet 4", Family: "claude",
					Capabilities: ModelCapabilities{Temperature: true, Reasoning: true, ToolCall: true,
						Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
					Cost: ModelCost{}, Limits: ModelLimits{Context: 200000, Output: 16384}, Status: ModelStatusActive},
			},
		},
		"cloudflare-workers-ai": {
			ID: "cloudflare-workers-ai", Name: "Cloudflare Workers AI",
			EnvVar: []string{"CLOUDFLARE_API_TOKEN"},
			Models: []ModelInfo{
				{ID: "@cf/meta/llama-3.3-70b-instruct-fp8-fast", ProviderID: "cloudflare-workers-ai", Name: "Llama 3.3 70B", Family: "llama",
					Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
						Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
					Cost: ModelCost{}, Limits: ModelLimits{Context: 131072, Output: 4096}, Status: ModelStatusActive},
			},
		},
	}

	// Also populate: groq, openrouter, mistral, cohere, together, replicate, perplexity, deepseek, azure, bedrock
	// These inherit from the existing hardcoded lists but now have proper ModelInfo
	mr.providers["groq"] = &ProviderInfo{
		ID: "groq", Name: "Groq",
		EnvVar: []string{"GROQ_API_KEY"},
		Models: []ModelInfo{
			{ID: "llama-3.3-70b-versatile", ProviderID: "groq", Name: "Llama 3.3 70B", Family: "llama",
				Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
					Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 0.59, Output: 0.79},
				Limits: ModelLimits{Context: 128000, Output: 32768}, Status: ModelStatusActive},
			{ID: "llama-3.1-70b-versatile", ProviderID: "groq", Name: "Llama 3.1 70B", Family: "llama",
				Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
					Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 0.59, Output: 0.79},
				Limits: ModelLimits{Context: 128000, Output: 32768}, Status: ModelStatusActive},
			{ID: "llama-3.1-8b-instant", ProviderID: "groq", Name: "Llama 3.1 8B", Family: "llama",
				Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
					Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 0.05, Output: 0.08},
				Limits: ModelLimits{Context: 131072, Output: 8192}, Status: ModelStatusActive},
			{ID: "deepseek-r1-distill-llama-70b", ProviderID: "groq", Name: "DeepSeek R1 Llama 70B", Family: "deepseek",
				Capabilities: ModelCapabilities{Temperature: true, Reasoning: true,
					Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 0.75, Output: 0.99},
				Limits: ModelLimits{Context: 128000, Output: 16384}, Status: ModelStatusActive},
			{ID: "gemma2-9b-it", ProviderID: "groq", Name: "Gemma 2 9B", Family: "gemma",
				Capabilities: ModelCapabilities{Temperature: true,
					Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 0.2, Output: 0.2},
				Limits: ModelLimits{Context: 8192, Output: 8192}, Status: ModelStatusActive},
			{ID: "mixtral-8x7b-32768", ProviderID: "groq", Name: "Mixtral 8x7B", Family: "mixtral",
				Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
					Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 0.24, Output: 0.24},
				Limits: ModelLimits{Context: 32768, Output: 32768}, Status: ModelStatusActive},
		},
	}

	mr.providers["deepseek"] = &ProviderInfo{
		ID: "deepseek", Name: "DeepSeek",
		EnvVar: []string{"DEEPSEEK_API_KEY"},
		Models: []ModelInfo{
			{ID: "deepseek-chat", ProviderID: "deepseek", Name: "DeepSeek Chat (V3)", Family: "deepseek",
				Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
					Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 0.27, Output: 1.1, CacheRead: 0.07},
				Limits: ModelLimits{Context: 65536, Output: 8192}, Status: ModelStatusActive},
			{ID: "deepseek-reasoner", ProviderID: "deepseek", Name: "DeepSeek Reasoner (R1)", Family: "deepseek",
				Capabilities: ModelCapabilities{Temperature: true, Reasoning: true,
					Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 0.55, Output: 2.19, CacheRead: 0.14},
				Limits: ModelLimits{Context: 65536, Output: 8192}, Status: ModelStatusActive},
			{ID: "deepseek-coder", ProviderID: "deepseek", Name: "DeepSeek Coder", Family: "deepseek",
				Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
					Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 0.14, Output: 0.28},
				Limits: ModelLimits{Context: 65536, Output: 8192}, Status: ModelStatusActive},
		},
	}

	mr.providers["mistral"] = &ProviderInfo{
		ID: "mistral", Name: "Mistral AI",
		EnvVar: []string{"MISTRAL_API_KEY"},
		Models: []ModelInfo{
			{ID: "mistral-large-latest", ProviderID: "mistral", Name: "Mistral Large", Family: "mistral",
				Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
					Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 2.0, Output: 6.0},
				Limits: ModelLimits{Context: 128000, Output: 8192}, Status: ModelStatusActive},
			{ID: "mistral-small-latest", ProviderID: "mistral", Name: "Mistral Small", Family: "mistral",
				Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
					Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 0.1, Output: 0.3},
				Limits: ModelLimits{Context: 128000, Output: 8192}, Status: ModelStatusActive},
			{ID: "codestral-latest", ProviderID: "mistral", Name: "Codestral", Family: "mistral",
				Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
					Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 0.3, Output: 0.9},
				Limits: ModelLimits{Context: 256000, Output: 8192}, Status: ModelStatusActive},
		},
	}

	mr.providers["perplexity"] = &ProviderInfo{
		ID: "perplexity", Name: "Perplexity AI",
		EnvVar: []string{"PERPLEXITY_API_KEY"},
		Models: []ModelInfo{
			{ID: "sonar", ProviderID: "perplexity", Name: "Sonar", Family: "sonar",
				Capabilities: ModelCapabilities{Temperature: true,
					Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 1.0, Output: 1.0},
				Limits: ModelLimits{Context: 128000, Output: 8192}, Status: ModelStatusActive},
			{ID: "sonar-pro", ProviderID: "perplexity", Name: "Sonar Pro", Family: "sonar",
				Capabilities: ModelCapabilities{Temperature: true,
					Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 3.0, Output: 15.0},
				Limits: ModelLimits{Context: 200000, Output: 8192}, Status: ModelStatusActive},
			{ID: "sonar-reasoning", ProviderID: "perplexity", Name: "Sonar Reasoning", Family: "sonar",
				Capabilities: ModelCapabilities{Temperature: true, Reasoning: true,
					Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 2.0, Output: 8.0},
				Limits: ModelLimits{Context: 128000, Output: 8192}, Status: ModelStatusActive},
		},
	}

	mr.providers["cohere"] = &ProviderInfo{
		ID: "cohere", Name: "Cohere",
		EnvVar: []string{"COHERE_API_KEY", "CO_API_KEY"},
		Models: []ModelInfo{
			{ID: "command-r-plus", ProviderID: "cohere", Name: "Command R+", Family: "command",
				Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
					Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 2.5, Output: 10.0},
				Limits: ModelLimits{Context: 128000, Output: 4096}, Status: ModelStatusActive},
			{ID: "command-r", ProviderID: "cohere", Name: "Command R", Family: "command",
				Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
					Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 0.15, Output: 0.6},
				Limits: ModelLimits{Context: 128000, Output: 4096}, Status: ModelStatusActive},
		},
	}

	mr.providers["together"] = &ProviderInfo{
		ID: "together", Name: "Together AI",
		EnvVar: []string{"TOGETHER_API_KEY", "TOGETHERAI_API_KEY"},
		Models: []ModelInfo{
			{ID: "meta-llama/Llama-3.3-70B-Instruct-Turbo", ProviderID: "together", Name: "Llama 3.3 70B", Family: "llama",
				Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
					Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 0.88, Output: 0.88},
				Limits: ModelLimits{Context: 131072, Output: 4096}, Status: ModelStatusActive},
			{ID: "meta-llama/Meta-Llama-3.1-405B-Instruct-Turbo", ProviderID: "together", Name: "Llama 3.1 405B", Family: "llama",
				Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
					Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 3.5, Output: 3.5},
				Limits: ModelLimits{Context: 131072, Output: 4096}, Status: ModelStatusActive},
			{ID: "Qwen/Qwen2.5-72B-Instruct-Turbo", ProviderID: "together", Name: "Qwen 2.5 72B", Family: "qwen",
				Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
					Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 1.2, Output: 1.2},
				Limits: ModelLimits{Context: 32768, Output: 4096}, Status: ModelStatusActive},
			{ID: "deepseek-ai/DeepSeek-V3", ProviderID: "together", Name: "DeepSeek V3", Family: "deepseek",
				Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
					Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 0.9, Output: 0.9},
				Limits: ModelLimits{Context: 65536, Output: 4096}, Status: ModelStatusActive},
		},
	}

	mr.providers["openrouter"] = &ProviderInfo{
		ID: "openrouter", Name: "OpenRouter",
		EnvVar: []string{"OPENROUTER_API_KEY"},
		Models: buildOpenRouterModels(),
	}

	mr.providers["azure"] = &ProviderInfo{
		ID: "azure", Name: "Azure OpenAI",
		EnvVar: []string{"AZURE_OPENAI_API_KEY", "AZURE_API_KEY"},
		Models: []ModelInfo{
			{ID: "gpt-4o", ProviderID: "azure", Name: "GPT-4o", Family: "gpt",
				Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
					Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 2.5, Output: 10.0},
				Limits: ModelLimits{Context: 128000, Output: 16384}, Status: ModelStatusActive},
			{ID: "gpt-4o-mini", ProviderID: "azure", Name: "GPT-4o Mini", Family: "gpt",
				Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
					Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 0.15, Output: 0.6},
				Limits: ModelLimits{Context: 128000, Output: 16384}, Status: ModelStatusActive},
			{ID: "gpt-4-turbo", ProviderID: "azure", Name: "GPT-4 Turbo", Family: "gpt",
				Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
					Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 10.0, Output: 30.0},
				Limits: ModelLimits{Context: 128000, Output: 4096}, Status: ModelStatusActive},
			{ID: "gpt-4", ProviderID: "azure", Name: "GPT-4", Family: "gpt",
				Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
					Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 30.0, Output: 60.0},
				Limits: ModelLimits{Context: 8192, Output: 4096}, Status: ModelStatusActive},
		},
	}

	mr.providers["bedrock"] = &ProviderInfo{
		ID: "bedrock", Name: "Amazon Bedrock",
		EnvVar: []string{"AWS_ACCESS_KEY_ID", "AWS_PROFILE"},
		Models: []ModelInfo{
			{ID: "anthropic.claude-3-5-sonnet-20241022-v2:0", ProviderID: "bedrock", Name: "Claude 3.5 Sonnet v2", Family: "claude",
				Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
					Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 3.0, Output: 15.0},
				Limits: ModelLimits{Context: 200000, Output: 8192}, Status: ModelStatusActive},
			{ID: "anthropic.claude-3-5-haiku-20241022-v1:0", ProviderID: "bedrock", Name: "Claude 3.5 Haiku", Family: "claude",
				Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
					Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 0.8, Output: 4.0},
				Limits: ModelLimits{Context: 200000, Output: 8192}, Status: ModelStatusActive},
			{ID: "anthropic.claude-3-opus-20240229-v1:0", ProviderID: "bedrock", Name: "Claude 3 Opus", Family: "claude",
				Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
					Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 15.0, Output: 75.0},
				Limits: ModelLimits{Context: 200000, Output: 4096}, Status: ModelStatusActive},
			{ID: "meta.llama3-1-405b-instruct-v1:0", ProviderID: "bedrock", Name: "Llama 3.1 405B", Family: "llama",
				Capabilities: ModelCapabilities{Temperature: true,
					Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 5.32, Output: 16.0},
				Limits: ModelLimits{Context: 128000, Output: 4096}, Status: ModelStatusActive},
		},
	}

	mr.providers["replicate"] = &ProviderInfo{
		ID: "replicate", Name: "Replicate",
		EnvVar: []string{"REPLICATE_API_TOKEN"},
		Models: []ModelInfo{
			{ID: "meta/llama-3.1-405b-instruct", ProviderID: "replicate", Name: "Llama 3.1 405B", Family: "llama",
				Capabilities: ModelCapabilities{Temperature: true,
					Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
				Cost:   ModelCost{Input: 9.5, Output: 9.5},
				Limits: ModelLimits{Context: 131072, Output: 4096}, Status: ModelStatusActive},
		},
	}
}

// buildOpenRouterModels creates the model list for OpenRouter
func buildOpenRouterModels() []ModelInfo {
	// Key models on OpenRouter with proper capabilities
	models := []ModelInfo{
		{ID: "anthropic/claude-sonnet-4-20250514", ProviderID: "openrouter", Name: "Claude Sonnet 4", Family: "claude",
			Capabilities: ModelCapabilities{Temperature: true, Reasoning: true, ToolCall: true,
				Input: ModalityFlags{Text: true, Image: true, PDF: true}, Output: ModalityFlags{Text: true}},
			Cost: ModelCost{Input: 3.0, Output: 15.0}, Limits: ModelLimits{Context: 200000, Output: 16384}, Status: ModelStatusActive},
		{ID: "anthropic/claude-opus-4-20250514", ProviderID: "openrouter", Name: "Claude Opus 4", Family: "claude",
			Capabilities: ModelCapabilities{Temperature: true, Reasoning: true, ToolCall: true,
				Input: ModalityFlags{Text: true, Image: true, PDF: true}, Output: ModalityFlags{Text: true}},
			Cost: ModelCost{Input: 15.0, Output: 75.0}, Limits: ModelLimits{Context: 200000, Output: 16384}, Status: ModelStatusActive},
		{ID: "openai/gpt-4.1", ProviderID: "openrouter", Name: "GPT-4.1", Family: "gpt",
			Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
				Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
			Cost: ModelCost{Input: 2.0, Output: 8.0}, Limits: ModelLimits{Context: 1047576, Output: 32768}, Status: ModelStatusActive},
		{ID: "openai/gpt-4o", ProviderID: "openrouter", Name: "GPT-4o", Family: "gpt",
			Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
				Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
			Cost: ModelCost{Input: 2.5, Output: 10.0}, Limits: ModelLimits{Context: 128000, Output: 16384}, Status: ModelStatusActive},
		{ID: "google/gemini-2.5-flash", ProviderID: "openrouter", Name: "Gemini 2.5 Flash", Family: "gemini",
			Capabilities: ModelCapabilities{Temperature: true, Reasoning: true, ToolCall: true,
				Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
			Cost: ModelCost{Input: 0.15, Output: 0.6}, Limits: ModelLimits{Context: 1048576, Output: 65536}, Status: ModelStatusActive},
		{ID: "google/gemini-2.5-pro", ProviderID: "openrouter", Name: "Gemini 2.5 Pro", Family: "gemini",
			Capabilities: ModelCapabilities{Temperature: true, Reasoning: true, ToolCall: true,
				Input: ModalityFlags{Text: true, Image: true}, Output: ModalityFlags{Text: true}},
			Cost: ModelCost{Input: 1.25, Output: 10.0}, Limits: ModelLimits{Context: 1048576, Output: 65536}, Status: ModelStatusActive},
		{ID: "meta-llama/llama-3.3-70b-instruct", ProviderID: "openrouter", Name: "Llama 3.3 70B", Family: "llama",
			Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
				Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
			Cost: ModelCost{Input: 0.39, Output: 0.39}, Limits: ModelLimits{Context: 131072, Output: 4096}, Status: ModelStatusActive},
		{ID: "deepseek/deepseek-chat", ProviderID: "openrouter", Name: "DeepSeek Chat", Family: "deepseek",
			Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
				Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
			Cost: ModelCost{Input: 0.27, Output: 1.1}, Limits: ModelLimits{Context: 65536, Output: 8192}, Status: ModelStatusActive},
		{ID: "deepseek/deepseek-reasoner", ProviderID: "openrouter", Name: "DeepSeek Reasoner", Family: "deepseek",
			Capabilities: ModelCapabilities{Temperature: true, Reasoning: true,
				Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
			Cost: ModelCost{Input: 0.55, Output: 2.19}, Limits: ModelLimits{Context: 65536, Output: 8192}, Status: ModelStatusActive},
		{ID: "x-ai/grok-3", ProviderID: "openrouter", Name: "Grok 3", Family: "grok",
			Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
				Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
			Cost: ModelCost{Input: 3.0, Output: 15.0}, Limits: ModelLimits{Context: 131072, Output: 16384}, Status: ModelStatusActive},
		{ID: "qwen/qwen-2.5-72b-instruct", ProviderID: "openrouter", Name: "Qwen 2.5 72B", Family: "qwen",
			Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
				Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
			Cost: ModelCost{Input: 0.35, Output: 0.4}, Limits: ModelLimits{Context: 32768, Output: 4096}, Status: ModelStatusActive},
	}

	// Add legacy models from the old list as basic entries
	legacyModels := []string{
		"anthropic/claude-3.7-sonnet", "anthropic/claude-3.5-sonnet", "anthropic/claude-3.5-haiku",
		"anthropic/claude-3-opus", "anthropic/claude-3-sonnet", "anthropic/claude-3-haiku",
		"openai/gpt-4.1-mini", "openai/gpt-4o-mini", "openai/o3-mini", "openai/o4-mini",
		"google/gemini-2.0-flash", "google/gemini-pro-1.5",
		"meta-llama/llama-3.1-405b-instruct", "meta-llama/llama-3.1-70b-instruct",
		"mistralai/mistral-large-2411", "mistralai/codestral",
		"deepseek/deepseek-v3",
		"qwen/qwen-2.5-coder-32b-instruct",
		"cohere/command-r-plus", "cohere/command-r",
		"x-ai/grok-2", "x-ai/grok-2-vision",
	}

	for _, id := range legacyModels {
		parts := strings.SplitN(id, "/", 2)
		name := id
		if len(parts) == 2 {
			name = parts[1]
		}
		models = append(models, ModelInfo{
			ID: id, ProviderID: "openrouter", Name: name,
			Capabilities: ModelCapabilities{Temperature: true, ToolCall: true,
				Input: ModalityFlags{Text: true}, Output: ModalityFlags{Text: true}},
			Limits: ModelLimits{Context: 128000, Output: 4096}, Status: ModelStatusActive,
		})
	}

	return models
}

// FindModel does a fuzzy search for a model across all providers
func (mr *ModelRegistry) FindModel(query string) []ModelInfo {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	query = strings.ToLower(query)
	var results []ModelInfo

	for _, prov := range mr.providers {
		for _, model := range prov.Models {
			if strings.Contains(strings.ToLower(model.ID), query) ||
				strings.Contains(strings.ToLower(model.Name), query) {
				results = append(results, model)
			}
		}
	}

	return results
}

// ParseModel splits "provider/model" into provider ID and model ID
func ParseModel(spec string) (providerID, modelID string) {
	// Handle provider:model format
	if idx := strings.Index(spec, ":"); idx > 0 {
		return spec[:idx], spec[idx+1:]
	}
	// Handle provider/model format (but not openrouter models which have org/model)
	parts := strings.SplitN(spec, "/", 2)
	if len(parts) == 2 {
		// Check if the first part is a known provider
		knownProviders := []string{"anthropic", "openai", "copilot", "google", "xai", "groq",
			"deepseek", "mistral", "cohere", "together", "perplexity", "azure", "bedrock",
			"deepinfra", "cerebras", "openrouter", "replicate", "google-vertex", "gitlab",
			"cloudflare-workers-ai", "cloudflare-ai-gateway"}
		for _, p := range knownProviders {
			if parts[0] == p {
				return parts[0], parts[1]
			}
		}
	}
	return "", spec
}
