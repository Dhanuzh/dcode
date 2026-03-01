package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
)

// ProviderInfo describes a provider for auth purposes.
type ProviderInfo struct {
	Key     string // e.g. "anthropic"
	Name    string // e.g. "Anthropic Claude"
	URLHint string // API key page URL
	EnvVar  string // primary env var name
}

// ProviderRegistry is the ordered list of all supported providers.
// It is the single source of truth used by Login(), ProviderLogin(),
// getConfiguredProviders(), and the CLI subcommand registration.
var ProviderRegistry = []ProviderInfo{
	{"anthropic", "Anthropic Claude", "https://console.anthropic.com/", "ANTHROPIC_API_KEY"},
	{"openai", "OpenAI GPT", "https://platform.openai.com/api-keys", "OPENAI_API_KEY"},
	{"copilot", "GitHub Copilot", "https://github.com/settings/tokens", "GITHUB_TOKEN"},
	{"google", "Google Gemini", "https://aistudio.google.com/apikey", "GOOGLE_API_KEY"},
	{"groq", "Groq", "https://console.groq.com/keys", "GROQ_API_KEY"},
	{"openrouter", "OpenRouter", "https://openrouter.ai/keys", "OPENROUTER_API_KEY"},
	{"xai", "xAI (Grok)", "https://console.x.ai/", "XAI_API_KEY"},
	{"deepseek", "DeepSeek", "https://platform.deepseek.com/api_keys", "DEEPSEEK_API_KEY"},
	{"mistral", "Mistral AI", "https://console.mistral.ai/api-keys", "MISTRAL_API_KEY"},
	{"deepinfra", "DeepInfra", "https://deepinfra.com/dash/api_keys", "DEEPINFRA_API_KEY"},
	{"cerebras", "Cerebras", "https://cloud.cerebras.ai/", "CEREBRAS_API_KEY"},
	{"together", "Together AI", "https://api.together.xyz/settings/api-keys", "TOGETHER_API_KEY"},
	{"cohere", "Cohere", "https://dashboard.cohere.com/api-keys", "COHERE_API_KEY"},
	{"perplexity", "Perplexity AI", "https://www.perplexity.ai/settings/api", "PERPLEXITY_API_KEY"},
	{"azure", "Azure OpenAI", "https://portal.azure.com/", "AZURE_OPENAI_API_KEY"},
	{"gitlab", "GitLab AI", "https://gitlab.com/-/user_settings/personal_access_tokens", "GITLAB_TOKEN"},
	{"cloudflare", "Cloudflare Workers AI", "https://dash.cloudflare.com/", "CLOUDFLARE_API_TOKEN"},
	{"replicate", "Replicate", "https://replicate.com/account/api-tokens", "REPLICATE_API_TOKEN"},
}

// OpenBrowser opens the given URL in the user's default browser.
func OpenBrowser(url string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return fmt.Errorf("unsupported platform %s", runtime.GOOS)
	}
}

// Credentials stores API keys securely - supports all providers
type Credentials struct {
	OpenAIAPIKey       string            `json:"openai_api_key,omitempty"`
	GoogleAPIKey       string            `json:"google_api_key,omitempty"`
	GroqAPIKey         string            `json:"groq_api_key,omitempty"`
	OpenRouterKey      string            `json:"openrouter_api_key,omitempty"`
	XAIAPIKey          string            `json:"xai_api_key,omitempty"`
	DeepInfraAPIKey    string            `json:"deepinfra_api_key,omitempty"`
	CerebrasAPIKey     string            `json:"cerebras_api_key,omitempty"`
	DeepSeekAPIKey     string            `json:"deepseek_api_key,omitempty"`
	MistralAPIKey      string            `json:"mistral_api_key,omitempty"`
	CohereAPIKey       string            `json:"cohere_api_key,omitempty"`
	TogetherAPIKey     string            `json:"together_api_key,omitempty"`
	PerplexityAPIKey   string            `json:"perplexity_api_key,omitempty"`
	ReplicateAPIToken  string            `json:"replicate_api_token,omitempty"`
	AzureAPIKey        string            `json:"azure_api_key,omitempty"`
	GitLabToken        string            `json:"gitlab_token,omitempty"`
	CloudflareAPIToken string            `json:"cloudflare_api_token,omitempty"`
	CustomProviders    map[string]string `json:"custom_providers,omitempty"`

	// OAuth tokens with refresh support
	OAuthTokens map[string]*OAuthToken `json:"oauth_tokens,omitempty"`
}

// OAuthToken stores OAuth credentials with refresh support
type OAuthToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresAt    int64  `json:"expires_at,omitempty"`
	AccountID    string `json:"account_id,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
}

// IsExpired checks if an OAuth token is expired
func (t *OAuthToken) IsExpired() bool {
	if t.ExpiresAt == 0 {
		return false
	}
	return time.Now().Unix() > t.ExpiresAt
}

// GetCredentialsPath returns the path to the credentials file
func GetCredentialsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "dcode", "credentials.json"), nil
}

// LoadCredentials loads stored credentials
func LoadCredentials() (*Credentials, error) {
	path, err := GetCredentialsPath()
	if err != nil {
		return &Credentials{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Credentials{}, nil
		}
		return nil, err
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}

	return &creds, nil
}

// SaveCredentials saves credentials to disk
func SaveCredentials(creds *Credentials) error {
	path, err := GetCredentialsPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// readHiddenInput reads input without echoing it (for passwords/keys)
func readHiddenInput(prompt string) (string, error) {
	fmt.Print(prompt)
	bytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	fmt.Println()
	return strings.TrimSpace(string(bytes)), nil
}

// SetProviderKey sets the API key on the Credentials struct for the given provider key.
func SetProviderKey(creds *Credentials, providerKey, apiKey string) {
	switch providerKey {
	case "openai":
		creds.OpenAIAPIKey = apiKey
	case "google":
		creds.GoogleAPIKey = apiKey
	case "groq":
		creds.GroqAPIKey = apiKey
	case "openrouter":
		creds.OpenRouterKey = apiKey
	case "xai":
		creds.XAIAPIKey = apiKey
	case "deepseek":
		creds.DeepSeekAPIKey = apiKey
	case "mistral":
		creds.MistralAPIKey = apiKey
	case "deepinfra":
		creds.DeepInfraAPIKey = apiKey
	case "cerebras":
		creds.CerebrasAPIKey = apiKey
	case "together":
		creds.TogetherAPIKey = apiKey
	case "cohere":
		creds.CohereAPIKey = apiKey
	case "perplexity":
		creds.PerplexityAPIKey = apiKey
	case "azure":
		creds.AzureAPIKey = apiKey
	case "gitlab":
		creds.GitLabToken = apiKey
	case "cloudflare":
		creds.CloudflareAPIToken = apiKey
	case "replicate":
		creds.ReplicateAPIToken = apiKey
	}
}

// ProviderLogin authenticates with a single provider by key.
// For API-key providers it opens the browser and prompts for the key.
// For "copilot" it returns a sentinel error so the caller can delegate to the OAuth flow.
func ProviderLogin(providerKey string) error {
	// Look up provider info
	var info *ProviderInfo
	for i := range ProviderRegistry {
		if ProviderRegistry[i].Key == providerKey {
			info = &ProviderRegistry[i]
			break
		}
	}
	if info == nil {
		return fmt.Errorf("unknown provider: %s", providerKey)
	}

	// Copilot does not use API keys — redirect to OAuth device flow
	if providerKey == "copilot" {
		fmt.Println()
		fmt.Println("\033[36mGitHub Copilot uses OAuth device flow.\033[0m")
		fmt.Println("\033[90mRun: \033[33mdcode auth copilot\033[90m to authenticate via browser.\033[0m")
		fmt.Println()
		return nil
	}

	cyan := "\033[36m"
	green := "\033[32m"
	yellow := "\033[33m"
	gray := "\033[90m"
	reset := "\033[0m"
	bold := "\033[1m"

	fmt.Println()
	fmt.Println(cyan + bold + "╭────────────────────────────────────────────╮" + reset)
	fmt.Printf(cyan+bold+"│  🔐 Authenticate with %-21s│\n"+reset, info.Name)
	fmt.Println(cyan + bold + "╰────────────────────────────────────────────╯" + reset)
	fmt.Println()

	fmt.Println(gray + "Get your API key from: " + yellow + info.URLHint + reset)
	fmt.Println()

	// Try to open the browser (best-effort)
	if err := OpenBrowser(info.URLHint); err == nil {
		fmt.Println(gray + "Opening browser..." + reset)
		fmt.Println()
	}

	apiKey, err := readHiddenInput(yellow + "API Key" + reset + " " + gray + "(hidden, Enter to cancel): " + reset)
	if err != nil {
		return fmt.Errorf("failed to read API key: %w", err)
	}
	if apiKey == "" {
		fmt.Println(gray + "→ Cancelled" + reset)
		return nil
	}

	creds, _ := LoadCredentials()
	if creds == nil {
		creds = &Credentials{}
	}
	SetProviderKey(creds, providerKey, apiKey)

	if err := SaveCredentials(creds); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	// Set as default provider
	if err := SaveDefaultProvider(providerKey); err != nil {
		fmt.Println(yellow + "⚠ Could not save default provider: " + err.Error() + reset)
	}

	path, _ := GetCredentialsPath()
	fmt.Println()
	fmt.Println(green + "✓ " + info.Name + " API key saved" + reset)
	fmt.Println(gray + "  " + path + reset)
	fmt.Println()
	fmt.Println(yellow + "You can now run " + cyan + bold + "dcode" + reset + yellow + " to start coding!" + reset)
	fmt.Println()

	return nil
}

// Login prompts the user to enter API keys and stores them
func Login() error {
	cyan := "\033[36m"
	green := "\033[32m"
	yellow := "\033[33m"
	gray := "\033[90m"
	reset := "\033[0m"
	bold := "\033[1m"

	fmt.Println(cyan + bold + "╭────────────────────────────────────────────╮" + reset)
	fmt.Println(cyan + bold + "│       🔐 DCode Authentication Setup       │" + reset)
	fmt.Println(cyan + bold + "╰────────────────────────────────────────────╯" + reset)
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// Load existing credentials
	creds, _ := LoadCredentials()
	if creds == nil {
		creds = &Credentials{}
	}

	fmt.Println(yellow + "Select your AI provider:" + reset)
	fmt.Println(cyan + "  1" + reset + gray + " > " + reset + "Anthropic Claude " + gray + "(Sonnet 4)" + reset)
	fmt.Println(cyan + "  2" + reset + gray + " > " + reset + "OpenAI GPT " + gray + "(GPT-4.1)" + reset)
	fmt.Println(cyan + "  3" + reset + gray + " > " + reset + "GitHub Copilot " + gray + "(OAuth)" + reset)
	fmt.Println(cyan + "  4" + reset + gray + " > " + reset + "Google Gemini " + gray + "(Gemini 2.5)" + reset)
	fmt.Println(cyan + "  5" + reset + gray + " > " + reset + "Groq " + gray + "(Llama 3.3 70B)" + reset)
	fmt.Println(cyan + "  6" + reset + gray + " > " + reset + "OpenRouter " + gray + "(Multi-provider)" + reset)
	fmt.Println(cyan + "  7" + reset + gray + " > " + reset + "xAI " + gray + "(Grok)" + reset)
	fmt.Println(cyan + "  8" + reset + gray + " > " + reset + "DeepSeek " + gray + "(DeepSeek V3)" + reset)
	fmt.Println(cyan + "  9" + reset + gray + " > " + reset + "Mistral AI " + gray + "(Mistral Large)" + reset)
	fmt.Println(cyan + " 10" + reset + gray + " > " + reset + "DeepInfra " + gray + "(Llama/Qwen)" + reset)
	fmt.Println(cyan + " 11" + reset + gray + " > " + reset + "Cerebras " + gray + "(Fast inference)" + reset)
	fmt.Println(cyan + " 12" + reset + gray + " > " + reset + "Together AI " + gray + "(Open models)" + reset)
	fmt.Println(cyan + " 13" + reset + gray + " > " + reset + "Cohere " + gray + "(Command R)" + reset)
	fmt.Println(cyan + " 14" + reset + gray + " > " + reset + "Perplexity " + gray + "(Sonar)" + reset)
	fmt.Println(cyan + " 15" + reset + gray + " > " + reset + "Azure OpenAI " + gray + "(Enterprise)" + reset)
	fmt.Println(cyan + " 16" + reset + gray + " > " + reset + "AWS Bedrock " + gray + "(AWS creds)" + reset)
	fmt.Println(cyan + " 17" + reset + gray + " > " + reset + "GitLab " + gray + "(GitLab AI)" + reset)
	fmt.Println(cyan + " 18" + reset + gray + " > " + reset + "Cloudflare " + gray + "(Workers AI)" + reset)
	fmt.Println(cyan + " 19" + reset + gray + " > " + reset + "Replicate " + gray + "(Open models)" + reset)
	fmt.Println(cyan + " 20" + reset + gray + " > " + reset + "Multiple providers")
	fmt.Println()
	fmt.Print(yellow + "Enter choice [1-20]" + reset + " " + cyan + "(default: 1): " + reset)

	providerChoice, _ := reader.ReadString('\n')
	providerChoice = strings.TrimSpace(strings.ToLower(providerChoice))

	providerMap := map[string]string{
		"": "anthropic", "1": "anthropic", "2": "openai", "3": "copilot",
		"4": "google", "5": "groq", "6": "openrouter", "7": "xai",
		"8": "deepseek", "9": "mistral", "10": "deepinfra", "11": "cerebras",
		"12": "together", "13": "cohere", "14": "perplexity", "15": "azure",
		"16": "bedrock", "17": "gitlab", "18": "cloudflare-workers-ai",
		"19": "replicate", "20": "multiple",
	}

	choice, ok := providerMap[providerChoice]
	if !ok {
		choice = "multiple"
	}

	// Anthropic uses PKCE OAuth exclusively — delegate immediately
	if choice == "anthropic" {
		fmt.Println()
		fmt.Println(cyan + "Anthropic uses OAuth (PKCE). Launching browser flow..." + reset)
		fmt.Println()
		return nil // actual flow runs via 'dcode auth anthropic' / runFirstTimeAuth
	}

	// Copilot uses OAuth device flow exclusively — delegate immediately
	if choice == "copilot" {
		// CopilotLogin is defined in internal/provider — call via the import path.
		// To avoid a circular import we invoke it through the shared runFirstTimeAuth
		// path in main. Here we just print a redirect message and return a sentinel.
		fmt.Println()
		fmt.Println(cyan + "GitHub Copilot uses OAuth device flow." + reset)
		fmt.Println(gray + "Run: " + yellow + "dcode auth copilot" + gray + " to authenticate." + reset)
		fmt.Println()
		return nil
	}

	providerPrompts := []struct {
		key     string
		name    string
		urlHint string
		setter  func(string)
	}{
		{"openai", "OpenAI GPT", "https://platform.openai.com/api-keys", func(k string) { creds.OpenAIAPIKey = k }},
		{"google", "Google Gemini", "https://aistudio.google.com/apikey", func(k string) { creds.GoogleAPIKey = k }},
		{"groq", "Groq", "https://console.groq.com/keys", func(k string) { creds.GroqAPIKey = k }},
		{"openrouter", "OpenRouter", "https://openrouter.ai/keys", func(k string) { creds.OpenRouterKey = k }},
		{"xai", "xAI (Grok)", "https://console.x.ai/", func(k string) { creds.XAIAPIKey = k }},
		{"deepseek", "DeepSeek", "https://platform.deepseek.com/api_keys", func(k string) { creds.DeepSeekAPIKey = k }},
		{"mistral", "Mistral AI", "https://console.mistral.ai/api-keys", func(k string) { creds.MistralAPIKey = k }},
		{"deepinfra", "DeepInfra", "https://deepinfra.com/dash/api_keys", func(k string) { creds.DeepInfraAPIKey = k }},
		{"cerebras", "Cerebras", "https://cloud.cerebras.ai/", func(k string) { creds.CerebrasAPIKey = k }},
		{"together", "Together AI", "https://api.together.xyz/settings/api-keys", func(k string) { creds.TogetherAPIKey = k }},
		{"cohere", "Cohere", "https://dashboard.cohere.com/api-keys", func(k string) { creds.CohereAPIKey = k }},
		{"perplexity", "Perplexity AI", "https://www.perplexity.ai/settings/api", func(k string) { creds.PerplexityAPIKey = k }},
		{"azure", "Azure OpenAI", "https://portal.azure.com/", func(k string) { creds.AzureAPIKey = k }},
		{"bedrock", "AWS Bedrock", "Set AWS_ACCESS_KEY_ID & AWS_SECRET_ACCESS_KEY", func(k string) {}},
		{"gitlab", "GitLab AI", "https://gitlab.com/-/user_settings/personal_access_tokens", func(k string) { creds.GitLabToken = k }},
		{"cloudflare-workers-ai", "Cloudflare Workers AI", "https://dash.cloudflare.com/", func(k string) { creds.CloudflareAPIToken = k }},
		{"replicate", "Replicate", "https://replicate.com/account/api-tokens", func(k string) { creds.ReplicateAPIToken = k }},
	}

	for _, pp := range providerPrompts {
		if choice != pp.key && choice != "multiple" {
			continue
		}

		fmt.Println()
		fmt.Println(cyan + "──────────────────────────────────────────────" + reset)
		fmt.Println(bold + pp.name + reset)
		fmt.Println(gray + "Get your API key from: " + yellow + pp.urlHint + reset)
		fmt.Println()

		apiKey, err := readHiddenInput(yellow + "API Key" + reset + " " + gray + "(hidden, Enter to skip): " + reset)
		if err != nil {
			return fmt.Errorf("failed to read API key: %w", err)
		}

		if apiKey != "" {
			pp.setter(apiKey)
			fmt.Println(green + "✓ " + pp.name + " API key saved" + reset)
		} else {
			fmt.Println(gray + "→ Skipped" + reset)
		}
	}

	if err := SaveCredentials(creds); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	// Save selected provider to config so dcode uses it by default
	if choice != "multiple" {
		if err := SaveDefaultProvider(choice); err != nil {
			fmt.Println(yellow + "⚠ Could not save default provider to config: " + reset + err.Error())
		} else {
			fmt.Println(green + "✓ Default provider set to: " + reset + bold + choice + reset)
		}
	}

	path, _ := GetCredentialsPath()
	fmt.Println()
	fmt.Println(cyan + "──────────────────────────────────────────────" + reset)
	fmt.Println(green + "✓ Success!" + reset + " Credentials saved to:")
	fmt.Println(gray + "  " + path + reset)
	fmt.Println()
	fmt.Println(yellow + "You can now run " + cyan + bold + "dcode" + reset + yellow + " to start coding!" + reset)
	fmt.Println()

	return nil
}

// SaveDefaultProvider saves the provider choice to the config file
func SaveDefaultProvider(providerName string) error {
	configDir := GetConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "dcode.yaml")

	// Read existing config or start fresh
	existing := ""
	if data, err := os.ReadFile(configPath); err == nil {
		existing = string(data)
	}

	// Update or add provider line
	if strings.Contains(existing, "provider:") {
		lines := strings.Split(existing, "\n")
		for i, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "provider:") {
				lines[i] = "provider: " + providerName
				break
			}
		}
		existing = strings.Join(lines, "\n")
	} else {
		if existing != "" && !strings.HasSuffix(existing, "\n") {
			existing += "\n"
		}
		existing += "provider: " + providerName + "\n"
	}

	return os.WriteFile(configPath, []byte(existing), 0644)
}

// getConfiguredProviders returns a list of provider names that have stored credentials
func getConfiguredProviders(creds *Credentials) []struct {
	key  string
	name string
} {
	type entry struct {
		key  string
		name string
		val  string
	}

	entries := []entry{
		{"openai", "OpenAI GPT", creds.OpenAIAPIKey},
		{"google", "Google Gemini", creds.GoogleAPIKey},
		{"groq", "Groq", creds.GroqAPIKey},
		{"openrouter", "OpenRouter", creds.OpenRouterKey},
		{"xai", "xAI (Grok)", creds.XAIAPIKey},
		{"deepseek", "DeepSeek", creds.DeepSeekAPIKey},
		{"mistral", "Mistral AI", creds.MistralAPIKey},
		{"deepinfra", "DeepInfra", creds.DeepInfraAPIKey},
		{"cerebras", "Cerebras", creds.CerebrasAPIKey},
		{"together", "Together AI", creds.TogetherAPIKey},
		{"cohere", "Cohere", creds.CohereAPIKey},
		{"perplexity", "Perplexity AI", creds.PerplexityAPIKey},
		{"replicate", "Replicate", creds.ReplicateAPIToken},
		{"azure", "Azure OpenAI", creds.AzureAPIKey},
		{"gitlab", "GitLab AI", creds.GitLabToken},
		{"cloudflare-workers-ai", "Cloudflare Workers AI", creds.CloudflareAPIToken},
	}

	var configured []struct {
		key  string
		name string
	}
	for _, e := range entries {
		if e.val != "" {
			configured = append(configured, struct {
				key  string
				name string
			}{e.key, e.name})
		}
	}

	// Check for OAuth tokens
	for key := range creds.OAuthTokens {
		configured = append(configured, struct {
			key  string
			name string
		}{key, key + " (OAuth)"})
	}

	// Check for Copilot OAuth token file
	hasCopilotOAuth := false
	if home, err := os.UserHomeDir(); err == nil {
		path := filepath.Join(home, ".config", "dcode", "copilot_oauth.json")
		if _, err := os.Stat(path); err == nil {
			hasCopilotOAuth = true
		}
	}
	// Add copilot-oauth entry only if copilot isn't already listed
	if hasCopilotOAuth {
		found := false
		for _, c := range configured {
			if c.key == "copilot" {
				found = true
				break
			}
		}
		if !found {
			configured = append(configured, struct {
				key  string
				name string
			}{"copilot", "GitHub Copilot (OAuth)"})
		}
	}

	return configured
}

// clearProviderCredential clears the credential for a specific provider
// ClearProviderCredential removes stored credentials for a given provider key from creds.
func ClearProviderCredential(creds *Credentials, providerKey string) {
	clearProviderCredential(creds, providerKey)
}

func clearProviderCredential(creds *Credentials, providerKey string) {
	switch providerKey {
	case "openai":
		creds.OpenAIAPIKey = ""
	case "google":
		creds.GoogleAPIKey = ""
	case "groq":
		creds.GroqAPIKey = ""
	case "openrouter":
		creds.OpenRouterKey = ""
	case "xai":
		creds.XAIAPIKey = ""
	case "deepseek":
		creds.DeepSeekAPIKey = ""
	case "mistral":
		creds.MistralAPIKey = ""
	case "deepinfra":
		creds.DeepInfraAPIKey = ""
	case "cerebras":
		creds.CerebrasAPIKey = ""
	case "together":
		creds.TogetherAPIKey = ""
	case "cohere":
		creds.CohereAPIKey = ""
	case "perplexity":
		creds.PerplexityAPIKey = ""
	case "replicate":
		creds.ReplicateAPIToken = ""
	case "azure":
		creds.AzureAPIKey = ""
	case "gitlab":
		creds.GitLabToken = ""
	case "cloudflare-workers-ai":
		creds.CloudflareAPIToken = ""
	}

	// Also remove from OAuth tokens if present
	if creds.OAuthTokens != nil {
		delete(creds.OAuthTokens, providerKey)
	}
}

// RemoveCopilotOAuthToken removes the Copilot OAuth token file.
func RemoveCopilotOAuthToken() error {
	return removeCopilotOAuthToken()
}

// removeCopilotOAuthToken removes the Copilot OAuth token file
func removeCopilotOAuthToken() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	path := filepath.Join(home, ".config", "dcode", "copilot_oauth.json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Logout removes stored credentials for a selected provider or all providers
func Logout() error {
	cyan := "\033[36m"
	green := "\033[32m"
	yellow := "\033[33m"
	gray := "\033[90m"
	reset := "\033[0m"
	bold := "\033[1m"

	fmt.Println()
	fmt.Println(cyan + bold + "╭────────────────────────────────────────────╮" + reset)
	fmt.Println(cyan + bold + "│       🔓 DCode Logout                     │" + reset)
	fmt.Println(cyan + bold + "╰────────────────────────────────────────────╯" + reset)
	fmt.Println()

	creds, err := LoadCredentials()
	if err != nil {
		creds = &Credentials{}
	}

	configured := getConfiguredProviders(creds)
	if len(configured) == 0 {
		fmt.Println(yellow + "ℹ No credentials found to remove" + reset)
		return nil
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println(yellow + "Select provider to logout from:" + reset)
	for i, p := range configured {
		fmt.Printf(cyan+"  %d"+reset+gray+" > "+reset+"%s\n", i+1, p.name)
	}
	fmt.Printf(cyan+"  %d"+reset+gray+" > "+reset+bold+"All providers"+reset+"\n", len(configured)+1)
	fmt.Println()
	fmt.Printf(yellow+"Enter choice [1-%d]: "+reset, len(configured)+1)

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	choice := 0
	if _, err := fmt.Sscanf(input, "%d", &choice); err != nil || choice < 1 || choice > len(configured)+1 {
		fmt.Println(yellow + "ℹ Invalid choice, aborting" + reset)
		return nil
	}

	if choice == len(configured)+1 {
		// Remove all: delete credentials file and copilot OAuth token
		path, err := GetCredentialsPath()
		if err != nil {
			return err
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := removeCopilotOAuthToken(); err != nil {
			fmt.Println(yellow + "⚠ Could not remove Copilot OAuth token: " + err.Error() + reset)
		}
		fmt.Println()
		fmt.Println(green + "✓ All credentials removed successfully" + reset)
		fmt.Println(cyan + "Run " + yellow + "dcode login" + cyan + " to configure new credentials" + reset)
		fmt.Println()
		return nil
	}

	// Remove a specific provider
	selected := configured[choice-1]
	clearProviderCredential(creds, selected.key)

	// If copilot was selected, also remove the OAuth token file
	if selected.key == "copilot" {
		if err := removeCopilotOAuthToken(); err != nil {
			fmt.Println(yellow + "⚠ Could not remove Copilot OAuth token: " + err.Error() + reset)
		}
	}

	if err := SaveCredentials(creds); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Println()
	fmt.Println(green + "✓ " + selected.name + " credentials removed successfully" + reset)
	fmt.Println(cyan + "Run " + yellow + "dcode login" + cyan + " to configure new credentials" + reset)
	fmt.Println()
	return nil
}

// GetAPIKeyWithFallback gets API key from multiple sources in priority order
func GetAPIKeyWithFallback(providerName string, config *Config) (string, error) {
	// Provider-specific environment variables (matching opencode)
	envVars := map[string][]string{
		"anthropic":             {"ANTHROPIC_API_KEY"},
		"openai":                {"OPENAI_API_KEY"},
		"google":                {"GOOGLE_API_KEY", "GEMINI_API_KEY"},
		"groq":                  {"GROQ_API_KEY"},
		"openrouter":            {"OPENROUTER_API_KEY"},
		"xai":                   {"XAI_API_KEY"},
		"deepseek":              {"DEEPSEEK_API_KEY"},
		"mistral":               {"MISTRAL_API_KEY"},
		"deepinfra":             {"DEEPINFRA_API_KEY"},
		"cerebras":              {"CEREBRAS_API_KEY"},
		"together":              {"TOGETHER_API_KEY", "TOGETHERAI_API_KEY"},
		"cohere":                {"COHERE_API_KEY", "CO_API_KEY"},
		"perplexity":            {"PERPLEXITY_API_KEY"},
		"replicate":             {"REPLICATE_API_TOKEN"},
		"azure":                 {"AZURE_OPENAI_API_KEY", "AZURE_API_KEY"},
		"bedrock":               {"AWS_ACCESS_KEY_ID"},
		"google-vertex":         {"GOOGLE_CLOUD_PROJECT"},
		"gitlab":                {"GITLAB_TOKEN", "GITLAB_API_TOKEN"},
		"cloudflare-workers-ai": {"CLOUDFLARE_API_TOKEN"},
	}

	// 1. Environment variables
	if vars, ok := envVars[providerName]; ok {
		for _, v := range vars {
			if apiKey := os.Getenv(v); apiKey != "" {
				return apiKey, nil
			}
		}
	}

	// 2. Stored credentials
	creds, err := LoadCredentials()
	if err == nil && creds != nil {
		credMap := map[string]string{
			"openai":                creds.OpenAIAPIKey,
			"google":                creds.GoogleAPIKey,
			"groq":                  creds.GroqAPIKey,
			"openrouter":            creds.OpenRouterKey,
			"xai":                   creds.XAIAPIKey,
			"deepseek":              creds.DeepSeekAPIKey,
			"mistral":               creds.MistralAPIKey,
			"deepinfra":             creds.DeepInfraAPIKey,
			"cerebras":              creds.CerebrasAPIKey,
			"together":              creds.TogetherAPIKey,
			"cohere":                creds.CohereAPIKey,
			"perplexity":            creds.PerplexityAPIKey,
			"replicate":             creds.ReplicateAPIToken,
			"azure":                 creds.AzureAPIKey,
			"gitlab":                creds.GitLabToken,
			"cloudflare-workers-ai": creds.CloudflareAPIToken,
		}
		if key, ok := credMap[providerName]; ok && key != "" {
			return key, nil
		}
	}

	// 3a. For anthropic, check for a stored OAuth token before falling through
	if providerName == "anthropic" {
		if creds != nil && creds.OAuthTokens != nil {
			if token, ok := creds.OAuthTokens["anthropic"]; ok && token != nil && token.AccessToken != "" {
				// Return a sentinel value; CreateProvider will handle the actual token loading.
				return "oauth", nil
			}
		}
	}

	// 3. Config file
	if apiKey := config.GetAPIKey(providerName); apiKey != "" {
		return apiKey, nil
	}

	// 4. For copilot, only accept the device OAuth token file
	if providerName == "copilot" {
		home, _ := os.UserHomeDir()
		if home != "" {
			tokenPath := filepath.Join(home, ".config", "dcode", "copilot_oauth.json")
			if data, err := os.ReadFile(tokenPath); err == nil {
				var tokenInfo struct {
					AccessToken string `json:"access_token"`
				}
				if json.Unmarshal(data, &tokenInfo) == nil && tokenInfo.AccessToken != "" {
					// Return sentinel; NewCopilotProvider loads the actual token via loadCopilotOAuthToken
					return "device_oauth", nil
				}
			}
		}
	}

	return "", fmt.Errorf("no API key found for provider '%s'", providerName)
}
