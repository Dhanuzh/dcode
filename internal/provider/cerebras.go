package provider

// CerebrasProvider uses Cerebras API (OpenAI-compatible)
type CerebrasProvider struct {
	*OpenAICompatibleProvider
}

// NewCerebrasProvider creates a new Cerebras provider
func NewCerebrasProvider(apiKey string) *CerebrasProvider {
	p := NewOpenAICompatibleProvider("cerebras", apiKey, "https://api.cerebras.ai/v1")
	p.models = []string{
		"qwen-3-235b-a22b-instruct-2507",
		"gpt-oss-120b",
		"llama3.1-8b",
		"zai-glm-4.7",
	}
	return &CerebrasProvider{OpenAICompatibleProvider: p}
}
