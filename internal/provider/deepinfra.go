package provider

// DeepInfraProvider uses DeepInfra API (OpenAI-compatible)
type DeepInfraProvider struct {
	*OpenAICompatibleProvider
}

// NewDeepInfraProvider creates a new DeepInfra provider
func NewDeepInfraProvider(apiKey string) *DeepInfraProvider {
	p := NewOpenAICompatibleProvider("deepinfra", apiKey, "https://api.deepinfra.com/v1/openai")
	p.models = []string{
		// Anthropic
		"anthropic/claude-4-opus",
		"anthropic/claude-3-7-sonnet-latest",
		// OpenAI
		"openai/gpt-oss-120b",
		"openai/gpt-oss-20b",
		// Qwen
		"Qwen/Qwen3-Coder-480B-A35B-Instruct-Turbo",
		"Qwen/Qwen3-Coder-480B-A35B-Instruct",
		// Moonshot / Kimi
		"moonshotai/Kimi-K2-Instruct",
		"moonshotai/Kimi-K2.5",
		"moonshotai/Kimi-K2-Thinking",
		// DeepSeek
		"deepseek-ai/DeepSeek-R1-0528",
		"deepseek-ai/DeepSeek-V3.2",
		// MiniMax
		"MiniMaxAI/MiniMax-M2",
		"MiniMaxAI/MiniMax-M2.1",
		// GLM
		"zai-org/GLM-4.7-Flash",
		"zai-org/GLM-4.7",
		"zai-org/GLM-4.5",
	}
	return &DeepInfraProvider{OpenAICompatibleProvider: p}
}
