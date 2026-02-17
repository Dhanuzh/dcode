package provider

// PerplexityProvider uses Perplexity AI's API
type PerplexityProvider struct {
	*OpenAICompatibleProvider
}

// NewPerplexityProvider creates a new Perplexity AI provider
func NewPerplexityProvider(apiKey string) *PerplexityProvider {
	p := NewOpenAICompatibleProvider("perplexity", apiKey, "https://api.perplexity.ai")
	p.models = []string{
		// Sonar models (online)
		"sonar",
		"sonar-pro",

		// Sonar models (chat)
		"sonar-reasoning",

		// Legacy models
		"llama-3.1-sonar-small-128k-online",
		"llama-3.1-sonar-large-128k-online",
		"llama-3.1-sonar-huge-128k-online",
		"llama-3.1-sonar-small-128k-chat",
		"llama-3.1-sonar-large-128k-chat",

		// Open models
		"llama-3.1-8b-instruct",
		"llama-3.1-70b-instruct",
	}
	return &PerplexityProvider{OpenAICompatibleProvider: p}
}
