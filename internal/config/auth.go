package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// Credentials stores API keys securely
type Credentials struct {
	AnthropicAPIKey string `json:"anthropic_api_key,omitempty"`
	OpenAIAPIKey    string `json:"openai_api_key,omitempty"`
	GitHubToken     string `json:"github_token,omitempty"`
	GoogleAPIKey    string `json:"google_api_key,omitempty"`
	GroqAPIKey      string `json:"groq_api_key,omitempty"`
	OpenRouterKey   string `json:"openrouter_api_key,omitempty"`
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
	fmt.Println(cyan + "  1" + reset + gray + " ❯ " + reset + "Anthropic Claude " + gray + "(Sonnet 4)" + reset)
	fmt.Println(cyan + "  2" + reset + gray + " ❯ " + reset + "OpenAI GPT " + gray + "(GPT-4.1)" + reset)
	fmt.Println(cyan + "  3" + reset + gray + " ❯ " + reset + "GitHub Copilot " + gray + "(GPT-4)" + reset)
	fmt.Println(cyan + "  4" + reset + gray + " ❯ " + reset + "Google Gemini " + gray + "(Gemini 2.5)" + reset)
	fmt.Println(cyan + "  5" + reset + gray + " ❯ " + reset + "Groq " + gray + "(Llama 3.3 70B)" + reset)
	fmt.Println(cyan + "  6" + reset + gray + " ❯ " + reset + "OpenRouter " + gray + "(Multi-provider)" + reset)
	fmt.Println(cyan + "  7" + reset + gray + " ❯ " + reset + "Multiple providers")
	fmt.Println()
	fmt.Print(yellow + "Enter choice [1-7]" + reset + " " + cyan + "(default: 1): " + reset)

	providerChoice, _ := reader.ReadString('\n')
	providerChoice = strings.TrimSpace(strings.ToLower(providerChoice))

	providerMap := map[string]string{
		"": "anthropic", "1": "anthropic", "2": "openai", "3": "copilot",
		"4": "google", "5": "groq", "6": "openrouter", "7": "multiple",
	}

	choice, ok := providerMap[providerChoice]
	if !ok {
		choice = "multiple"
	}

	providerPrompts := []struct {
		key     string
		name    string
		urlHint string
		setter  func(string)
	}{
		{"anthropic", "Anthropic Claude", "https://console.anthropic.com/", func(k string) { creds.AnthropicAPIKey = k }},
		{"openai", "OpenAI GPT", "https://platform.openai.com/api-keys", func(k string) { creds.OpenAIAPIKey = k }},
		{"copilot", "GitHub Copilot", "https://github.com/settings/tokens", func(k string) { creds.GitHubToken = k }},
		{"google", "Google Gemini", "https://aistudio.google.com/apikey", func(k string) { creds.GoogleAPIKey = k }},
		{"groq", "Groq", "https://console.groq.com/keys", func(k string) { creds.GroqAPIKey = k }},
		{"openrouter", "OpenRouter", "https://openrouter.ai/keys", func(k string) { creds.OpenRouterKey = k }},
	}

	for _, pp := range providerPrompts {
		if choice != pp.key && choice != "multiple" {
			continue
		}

		// Special handling for Copilot - auto-detect from gh CLI
		if pp.key == "copilot" {
			fmt.Println()
			fmt.Println(cyan + "──────────────────────────────────────────────" + reset)
			fmt.Println(bold + pp.name + reset)

			// Try auto-detect from gh CLI
			ghToken := ""
			if cmd, err := exec.Command("gh", "auth", "token").Output(); err == nil {
				ghToken = strings.TrimSpace(string(cmd))
			}
			if ghToken != "" {
				creds.GitHubToken = ghToken
				fmt.Println(green + "✓ Auto-detected GitHub token from 'gh auth'" + reset)
			} else {
				fmt.Println(gray + "Could not auto-detect token from 'gh auth'. Please provide manually." + reset)
				fmt.Println(gray + "Get your token from: " + yellow + pp.urlHint + reset)
				fmt.Println()
				apiKey, err := readHiddenInput(yellow + "GitHub Token" + reset + " " + gray + "(hidden, Enter to skip): " + reset)
				if err != nil {
					return fmt.Errorf("failed to read token: %w", err)
				}
				if apiKey != "" {
					creds.GitHubToken = apiKey
					fmt.Println(green + "✓ GitHub token saved" + reset)
				} else {
					fmt.Println(gray + "→ Skipped" + reset)
				}
			}
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
		if err := saveDefaultProvider(choice); err != nil {
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

// saveDefaultProvider saves the provider choice to the config file
func saveDefaultProvider(providerName string) error {
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

// Logout removes stored credentials
func Logout() error {
	cyan := "\033[36m"
	green := "\033[32m"
	yellow := "\033[33m"
	reset := "\033[0m"

	path, err := GetCredentialsPath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			fmt.Println(yellow + "ℹ No credentials found to remove" + reset)
			return nil
		}
		return err
	}

	fmt.Println()
	fmt.Println(green + "✓ Credentials removed successfully" + reset)
	fmt.Println(cyan + "Run " + yellow + "dcode login" + cyan + " to configure new credentials" + reset)
	fmt.Println()
	return nil
}

// GetAPIKeyWithFallback gets API key from multiple sources in priority order
func GetAPIKeyWithFallback(providerName string, config *Config) (string, error) {
	envVars := map[string][]string{
		"anthropic":  {"ANTHROPIC_API_KEY"},
		"openai":     {"OPENAI_API_KEY"},
		"copilot":    {"GITHUB_TOKEN"},
		"google":     {"GOOGLE_API_KEY", "GEMINI_API_KEY"},
		"groq":       {"GROQ_API_KEY"},
		"openrouter": {"OPENROUTER_API_KEY"},
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
			"anthropic":  creds.AnthropicAPIKey,
			"openai":     creds.OpenAIAPIKey,
			"copilot":    creds.GitHubToken,
			"google":     creds.GoogleAPIKey,
			"groq":       creds.GroqAPIKey,
			"openrouter": creds.OpenRouterKey,
		}
		if key, ok := credMap[providerName]; ok && key != "" {
			return key, nil
		}
	}

	// 3. Config file
	if apiKey := config.GetAPIKey(providerName); apiKey != "" {
		return apiKey, nil
	}

	// 4. For copilot, try auto-detect from gh CLI
	if providerName == "copilot" {
		if cmd, err := exec.Command("gh", "auth", "token").Output(); err == nil {
			token := strings.TrimSpace(string(cmd))
			if token != "" {
				return token, nil
			}
		}
		// Try reading from gh hosts.yml
		home, _ := os.UserHomeDir()
		if home != "" {
			data, err := os.ReadFile(filepath.Join(home, ".config", "gh", "hosts.yml"))
			if err == nil {
				for _, line := range strings.Split(string(data), "\n") {
					if strings.Contains(line, "oauth_token:") {
						parts := strings.SplitN(line, ":", 2)
						if len(parts) == 2 {
							token := strings.TrimSpace(parts[1])
							if token != "" {
								return token, nil
							}
						}
					}
				}
			}
		}
	}

	return "", fmt.Errorf("no API key found for provider '%s'", providerName)
}
