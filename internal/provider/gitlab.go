package provider

import (
	"os"
)

// GitLabProvider uses GitLab's AI API (OpenAI-compatible)
type GitLabProvider struct {
	*OpenAICompatibleProvider
}

// NewGitLabProvider creates a new GitLab provider
func NewGitLabProvider(apiKey string) *GitLabProvider {
	baseURL := os.Getenv("GITLAB_API_URL")
	if baseURL == "" {
		baseURL = "https://gitlab.com/api/v4/ai/v1"
	}

	if apiKey == "" {
		apiKey = os.Getenv("GITLAB_TOKEN")
		if apiKey == "" {
			apiKey = os.Getenv("GITLAB_API_TOKEN")
		}
	}

	p := NewOpenAICompatibleProvider("gitlab", apiKey, baseURL)
	p.models = []string{
		"duo-chat-opus-4-6",
		"duo-chat-sonnet-4-5",
		"duo-chat-opus-4-5",
		"duo-chat-haiku-4-5",
		"duo-chat-gpt-5-2-codex",
		"duo-chat-gpt-5-2",
		"duo-chat-gpt-5-codex",
		"duo-chat-gpt-5-mini",
		"duo-chat-gpt-5-1",
	}
	return &GitLabProvider{OpenAICompatibleProvider: p}
}
