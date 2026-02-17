package provider

// MistralProvider uses Mistral AI's API
type MistralProvider struct {
	*OpenAICompatibleProvider
}

// NewMistralProvider creates a new Mistral AI provider
func NewMistralProvider(apiKey string) *MistralProvider {
	p := NewOpenAICompatibleProvider("mistral", apiKey, "https://api.mistral.ai/v1")
	p.models = []string{
		// Devstral (code-focused)
		"devstral-medium-latest",
		"devstral-medium-2507",
		"devstral-2512",
		"devstral-small-2505",
		"devstral-small-2507",
		"labs-devstral-small-2512",
		// Magistral (reasoning)
		"magistral-medium-latest",
		"magistral-small",
		// Mistral Large
		"mistral-large-latest",
		"mistral-large-2512",
		"mistral-large-2411",
		// Mistral Medium
		"mistral-medium-latest",
		"mistral-medium-2508",
		"mistral-medium-2505",
		// Mistral Small
		"mistral-small-latest",
		"mistral-small-2506",
		// Codestral
		"codestral-latest",
		// Ministral
		"ministral-8b-latest",
		"ministral-3b-latest",
		// Pixtral (vision)
		"pixtral-large-latest",
		"pixtral-12b",
		// Mistral Nemo
		"mistral-nemo",
		// Open models
		"open-mistral-7b",
		"open-mixtral-8x7b",
		"open-mixtral-8x22b",
		// Embeddings
		"mistral-embed",
	}
	return &MistralProvider{OpenAICompatibleProvider: p}
}
