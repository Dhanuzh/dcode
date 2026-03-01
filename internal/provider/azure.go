package provider

// AzureOpenAIProvider uses Azure OpenAI Service
type AzureOpenAIProvider struct {
	*OpenAICompatibleProvider
}

// NewAzureOpenAIProvider creates a new Azure OpenAI provider
// endpoint should be: https://{your-resource-name}.openai.azure.com
func NewAzureOpenAIProvider(apiKey, endpoint string) *AzureOpenAIProvider {
	if endpoint == "" {
		endpoint = "https://YOUR_RESOURCE_NAME.openai.azure.com"
	}
	p := NewOpenAICompatibleProvider("azure", apiKey, endpoint)
	p.models = []string{
		// GPT-4 models
		"gpt-4",
		"gpt-4-32k",
		"gpt-4-turbo",
		"gpt-4-turbo-2024-04-09",
		"gpt-4o",
		"gpt-4o-mini",

		// GPT-3.5 models
		"gpt-35-turbo",
		"gpt-35-turbo-16k",

		// Embedding models
		"text-embedding-ada-002",
		"text-embedding-3-small",
		"text-embedding-3-large",

		// Note: Actual model names depend on your Azure deployment names
	}
	return &AzureOpenAIProvider{OpenAICompatibleProvider: p}
}
