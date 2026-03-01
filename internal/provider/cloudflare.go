package provider

import (
	"os"
)

// CloudflareWorkersAIProvider uses Cloudflare Workers AI (OpenAI-compatible)
type CloudflareWorkersAIProvider struct {
	*OpenAICompatibleProvider
}

// NewCloudflareWorkersAIProvider creates a new Cloudflare Workers AI provider
func NewCloudflareWorkersAIProvider(apiKey string) *CloudflareWorkersAIProvider {
	accountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	baseURL := "https://api.cloudflare.com/client/v4/accounts/" + accountID + "/ai/v1"
	if accountID == "" {
		baseURL = "https://api.cloudflare.com/client/v4/ai/v1"
	}

	if apiKey == "" {
		apiKey = os.Getenv("CLOUDFLARE_API_TOKEN")
	}

	p := NewOpenAICompatibleProvider("cloudflare-workers-ai", apiKey, baseURL)
	p.models = []string{
		"@cf/openai/gpt-oss-120b",
		"@cf/openai/gpt-oss-20b",
		"@cf/qwen/qwq-32b",
		"@cf/qwen/qwen3-30b-a3b-fp8",
		"@cf/qwen/qwen2.5-coder-32b-instruct",
		"@cf/meta/llama-3.3-70b-instruct-fp8-fast",
		"@cf/meta/llama-4-scout-17b-16e-instruct",
		"@cf/meta/llama-3.2-11b-vision-instruct",
		"@cf/meta/llama-3.2-3b-instruct",
		"@cf/meta/llama-3.1-8b-instruct",
		"@cf/google/gemma-3-12b-it",
		"@cf/google/gemma-3-4b-it",
		"@cf/mistralai/mistral-small-3.1-24b-instruct",
		"@cf/deepseek-ai/deepseek-r1-distill-qwen-32b",
		"@cf/deepseek-ai/deepseek-math-7b-instruct",
	}
	return &CloudflareWorkersAIProvider{OpenAICompatibleProvider: p}
}
