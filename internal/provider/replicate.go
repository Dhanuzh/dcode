package provider

// ReplicateProvider uses Replicate's API
type ReplicateProvider struct {
	*OpenAICompatibleProvider
}

// NewReplicateProvider creates a new Replicate provider
func NewReplicateProvider(apiKey string) *ReplicateProvider {
	p := NewOpenAICompatibleProvider("replicate", apiKey, "https://api.replicate.com/v1")
	p.models = []string{
		// Meta Llama models
		"meta/llama-3.1-405b-instruct",
		"meta/llama-3.1-70b-instruct",
		"meta/llama-3.1-8b-instruct",
		"meta/llama-2-70b-chat",
		"meta/llama-2-13b-chat",
		"meta/llama-2-7b-chat",

		// Mistral models
		"mistralai/mixtral-8x7b-instruct-v0.1",
		"mistralai/mistral-7b-instruct-v0.2",

		// Other popular models
		"stability-ai/stable-diffusion",
		"stability-ai/sdxl",
		"black-forest-labs/flux-schnell",
		"yorickvp/llava-13b",
	}
	return &ReplicateProvider{OpenAICompatibleProvider: p}
}
