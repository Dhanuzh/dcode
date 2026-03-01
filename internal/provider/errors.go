package provider

import (
	"fmt"
	"strings"
)

// UserFriendlyError wraps errors with helpful user-facing messages
type UserFriendlyError struct {
	Title            string // Short title for the error
	Message          string // Detailed user-friendly message
	Suggestion       string // What the user should do
	TechnicalDetails string // Technical error details (for debugging)
	Original         error  // Original error
}

func (e *UserFriendlyError) Error() string {
	var sb strings.Builder
	sb.WriteString(e.Title)
	if e.Message != "" {
		sb.WriteString(": ")
		sb.WriteString(e.Message)
	}
	if e.Suggestion != "" {
		sb.WriteString("\n\nSuggestion: ")
		sb.WriteString(e.Suggestion)
	}
	if e.TechnicalDetails != "" {
		sb.WriteString("\n\nTechnical details: ")
		sb.WriteString(e.TechnicalDetails)
	}
	return sb.String()
}

func (e *UserFriendlyError) Unwrap() error {
	return e.Original
}

// MakeUserFriendly converts various errors into user-friendly messages
func MakeUserFriendly(err error, provider string) error {
	if err == nil {
		return nil
	}

	// Check if already a ClassifiedError
	if classified, ok := err.(*ClassifiedError); ok {
		return convertClassifiedError(classified, provider)
	}

	// Check if already user-friendly
	if _, ok := err.(*UserFriendlyError); ok {
		return err
	}

	// Generic error
	return &UserFriendlyError{
		Title:            "API Error",
		Message:          fmt.Sprintf("The %s API returned an error", provider),
		TechnicalDetails: err.Error(),
		Original:         err,
	}
}

func convertClassifiedError(ce *ClassifiedError, provider string) error {
	switch ce.Type {
	case ErrorTypeContextOverflow:
		return &UserFriendlyError{
			Title:   "Context Window Exceeded",
			Message: "Your conversation has become too long for the model's context window.",
			Suggestion: `Try one of these solutions:
  1. Use 'compact' command to summarize the conversation
  2. Start a new session
  3. Use a model with a larger context window
  4. Reduce the amount of code/content in your messages`,
			TechnicalDetails: ce.Message,
			Original:         ce.Original,
		}

	case ErrorTypeAuth:
		return &UserFriendlyError{
			Title:   "Authentication Failed",
			Message: fmt.Sprintf("Unable to authenticate with %s.", provider),
			Suggestion: fmt.Sprintf(`Please check your API key:
  1. Run 'dcode login' to update your credentials
  2. Or set the environment variable for %s
  3. Verify your API key is valid at the provider's dashboard
  4. Check that your API key has the necessary permissions`, provider),
			TechnicalDetails: ce.Message,
			Original:         ce.Original,
		}

	case ErrorTypeRateLimit:
		return &UserFriendlyError{
			Title:   "Rate Limit Exceeded",
			Message: fmt.Sprintf("You've hit the rate limit for %s.", provider),
			Suggestion: `The system will automatically retry, or you can:
  1. Wait a few moments before trying again
  2. Check your plan/quota at the provider's dashboard
  3. Consider upgrading your API plan for higher limits
  4. Switch to a different provider temporarily`,
			TechnicalDetails: ce.Message,
			Original:         ce.Original,
		}

	case ErrorTypeNotFound:
		return &UserFriendlyError{
			Title:   "Model or Endpoint Not Found",
			Message: "The requested model or API endpoint could not be found.",
			Suggestion: `Try these steps:
  1. Check if the model name is spelled correctly
  2. Run 'dcode models' to see available models
  3. Verify the model is available for your account/region
  4. Try a different model for this provider`,
			TechnicalDetails: ce.Message,
			Original:         ce.Original,
		}

	case ErrorTypeTimeout:
		return &UserFriendlyError{
			Title:   "Request Timeout",
			Message: "The request took too long and timed out.",
			Suggestion: `This can happen with:
  1. Very large requests - try breaking them into smaller parts
  2. Network issues - check your internet connection
  3. Provider overload - wait a moment and try again
  4. Complex reasoning tasks - some models need more time`,
			TechnicalDetails: ce.Message,
			Original:         ce.Original,
		}

	default:
		return &UserFriendlyError{
			Title:            "API Error",
			Message:          fmt.Sprintf("An error occurred while communicating with %s.", provider),
			TechnicalDetails: ce.Message,
			Original:         ce.Original,
		}
	}
}

// SuggestProviderAlternative suggests alternative providers
func SuggestProviderAlternative(currentProvider string, reason string) string {
	alternatives := map[string][]string{
		"anthropic": {"openai", "google", "mistral"},
		"openai":    {"anthropic", "google", "azure"},
		"google":    {"anthropic", "openai", "mistral"},
		"azure":     {"openai", "anthropic"},
		"deepseek":  {"openai", "anthropic", "groq"},
		"groq":      {"deepseek", "together", "cerebras"},
		"together":  {"groq", "deepinfra", "replicate"},
	}

	if alts, ok := alternatives[currentProvider]; ok && len(alts) > 0 {
		return fmt.Sprintf("Alternative providers you could try: %s\nSwitch with: dcode --provider %s",
			strings.Join(alts, ", "), alts[0])
	}

	return ""
}

// FormatProviderError formats an error with provider context
func FormatProviderError(provider, model string, err error) error {
	if err == nil {
		return nil
	}

	// Make it user-friendly
	friendly := MakeUserFriendly(err, provider)

	// Add provider/model context
	if uf, ok := friendly.(*UserFriendlyError); ok {
		uf.TechnicalDetails = fmt.Sprintf("Provider: %s, Model: %s\n%s",
			provider, model, uf.TechnicalDetails)
	}

	return friendly
}

// ExplainProviderLimits explains common provider limitations
func ExplainProviderLimits(provider string) string {
	limits := map[string]string{
		"anthropic": `Anthropic Claude Limits:
- Context window: 200K tokens (Claude 3.x)
- Rate limits: Varies by plan (Free: 50 requests/day)
- Cost: Input ~$3/M tokens, Output ~$15/M tokens (Claude 3 Opus)`,

		"openai": `OpenAI GPT Limits:
- Context window: 128K tokens (GPT-4 Turbo)
- Rate limits: Varies by plan (Free tier limited)
- Cost: Input ~$10/M tokens, Output ~$30/M tokens (GPT-4)`,

		"google": `Google Gemini Limits:
- Context window: 2M tokens (Gemini 1.5 Pro)
- Rate limits: 60 requests/minute (free tier)
- Cost: Free tier available, then usage-based`,

		"groq": `Groq Limits:
- Context window: 32K-128K depending on model
- Rate limits: Generous free tier
- Speed: Very fast inference (up to 750 tokens/sec)`,
	}

	if limit, ok := limits[provider]; ok {
		return limit
	}

	return fmt.Sprintf("Provider: %s\nCheck the provider's documentation for specific limits.", provider)
}
