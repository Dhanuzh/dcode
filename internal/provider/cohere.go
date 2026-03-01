package provider

// CohereProvider uses Cohere's API
type CohereProvider struct {
	*OpenAICompatibleProvider
}

// NewCohereProvider creates a new Cohere provider
func NewCohereProvider(apiKey string) *CohereProvider {
	p := NewOpenAICompatibleProvider("cohere", apiKey, "https://api.cohere.ai/v1")
	p.models = []string{
		// Command models
		"command-r-plus",
		"command-r",
		"command",
		"command-light",

		// Embed models
		"embed-english-v3.0",
		"embed-multilingual-v3.0",
		"embed-english-light-v3.0",
		"embed-multilingual-light-v3.0",

		// Legacy models
		"command-nightly",
		"command-light-nightly",
	}
	return &CohereProvider{OpenAICompatibleProvider: p}
}
