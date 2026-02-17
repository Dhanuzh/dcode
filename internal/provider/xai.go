package provider

// XAIProvider uses xAI's Grok API (OpenAI-compatible)
type XAIProvider struct {
	*OpenAICompatibleProvider
}

// NewXAIProvider creates a new xAI provider
func NewXAIProvider(apiKey string) *XAIProvider {
	p := NewOpenAICompatibleProvider("xai", apiKey, "https://api.x.ai/v1")
	p.models = []string{
		// Grok 4.x
		"grok-4",
		"grok-4-fast",
		"grok-4-fast-non-reasoning",
		"grok-4-1-fast",
		"grok-4-1-fast-non-reasoning",
		// Grok 3.x
		"grok-3",
		"grok-3-latest",
		"grok-3-fast",
		"grok-3-fast-latest",
		"grok-3-mini",
		"grok-3-mini-latest",
		"grok-3-mini-fast",
		"grok-3-mini-fast-latest",
		// Grok Code
		"grok-code-fast-1",
		// Grok 2.x
		"grok-2",
		"grok-2-latest",
		"grok-2-1212",
		"grok-2-vision",
		"grok-2-vision-latest",
		"grok-2-vision-1212",
		// Beta
		"grok-beta",
		"grok-vision-beta",
	}
	return &XAIProvider{OpenAICompatibleProvider: p}
}
