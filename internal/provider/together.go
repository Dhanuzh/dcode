package provider

// TogetherProvider uses Together AI's API
type TogetherProvider struct {
	*OpenAICompatibleProvider
}

// NewTogetherProvider creates a new Together AI provider
func NewTogetherProvider(apiKey string) *TogetherProvider {
	p := NewOpenAICompatibleProvider("together", apiKey, "https://api.together.xyz/v1")
	p.models = []string{
		// Meta Llama models
		"meta-llama/Llama-3.3-70B-Instruct-Turbo",
		"meta-llama/Llama-3.2-90B-Vision-Instruct-Turbo",
		"meta-llama/Meta-Llama-3.1-405B-Instruct-Turbo",
		"meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo",
		"meta-llama/Meta-Llama-3.1-8B-Instruct-Turbo",
		"meta-llama/Llama-3.2-11B-Vision-Instruct-Turbo",
		"meta-llama/Llama-3.2-3B-Instruct-Turbo",

		// Google models
		"google/gemma-2-27b-it",
		"google/gemma-2-9b-it",

		// Mistral models
		"mistralai/Mixtral-8x7B-Instruct-v0.1",
		"mistralai/Mistral-7B-Instruct-v0.3",

		// Qwen models
		"Qwen/Qwen2.5-72B-Instruct-Turbo",
		"Qwen/Qwen2.5-7B-Instruct-Turbo",

		// DeepSeek models
		"deepseek-ai/deepseek-llm-67b-chat",

		// Others
		"NousResearch/Nous-Hermes-2-Mixtral-8x7B-DPO",
		"upstage/SOLAR-10.7B-Instruct-v1.0",
		"zero-one-ai/Yi-34B-Chat",
	}
	return &TogetherProvider{OpenAICompatibleProvider: p}
}
