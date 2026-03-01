package provider

// DeepSeekProvider uses DeepSeek's API
type DeepSeekProvider struct {
	*OpenAICompatibleProvider
}

// NewDeepSeekProvider creates a new DeepSeek provider
func NewDeepSeekProvider(apiKey string) *DeepSeekProvider {
	p := NewOpenAICompatibleProvider("deepseek", apiKey, "https://api.deepseek.com/v1")
	p.models = []string{
		"deepseek-reasoner",
		"deepseek-chat",
	}
	return &DeepSeekProvider{OpenAICompatibleProvider: p}
}
