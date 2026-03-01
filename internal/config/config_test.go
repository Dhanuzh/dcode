package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestSaveConfigPermissions verifies SaveConfig writes with 0600 permissions.
func TestSaveConfigPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dcode.json")

	cfg := &Config{Provider: "anthropic", MaxTokens: 4096}
	if err := cfg.SaveConfig(path); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("expected 0600 permissions, got %04o", perm)
	}
}

// TestSaveConfigRoundTrip writes and reads back config JSON.
func TestSaveConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dcode.json")

	cfg := &Config{
		Provider:  "openai",
		MaxTokens: 8192,
		Streaming: true,
		Theme:     "dark",
	}
	if err := cfg.SaveConfig(path); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var got Config
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.Provider != cfg.Provider {
		t.Errorf("Provider: want %q, got %q", cfg.Provider, got.Provider)
	}
	if got.MaxTokens != cfg.MaxTokens {
		t.Errorf("MaxTokens: want %d, got %d", cfg.MaxTokens, got.MaxTokens)
	}
	if got.Streaming != cfg.Streaming {
		t.Errorf("Streaming: want %v, got %v", cfg.Streaming, got.Streaming)
	}
}

// TestSaveConfigCreatesDirectory ensures SaveConfig creates missing parent dirs.
func TestSaveConfigCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "dcode.json")

	cfg := &Config{Provider: "google"}
	if err := cfg.SaveConfig(path); err != nil {
		t.Fatalf("SaveConfig with nested dirs: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected config file to exist after SaveConfig")
	}
}

// TestGetConfigDir returns a non-empty path.
func TestGetConfigDir(t *testing.T) {
	dir := GetConfigDir()
	if dir == "" {
		t.Error("GetConfigDir should return a non-empty path")
	}
}

// TestGetDefaultModel returns model for known providers and falls back gracefully.
func TestGetDefaultModel(t *testing.T) {
	cfg := &Config{}

	for _, prov := range []string{"anthropic", "openai", "google"} {
		model := cfg.GetDefaultModel(prov)
		if model == "" {
			t.Errorf("GetDefaultModel(%q) returned empty string", prov)
		}
	}

	// Unknown provider should not panic and return something
	model := cfg.GetDefaultModel("unknown-provider-xyz")
	_ = model // may be empty — just must not panic
}

// TestGetAPIKey reads keys via the legacy top-level field and provider map.
func TestGetAPIKey(t *testing.T) {
	// Use a legacy top-level field that GetAPIKey reads.
	cfg := &Config{
		OpenAIAPIKey: "sk-test-key",
	}

	key := cfg.GetAPIKey("openai")
	if key != "sk-test-key" {
		t.Errorf("GetAPIKey(openai): want %q, got %q", "sk-test-key", key)
	}

	// Unknown provider with no env var returns empty.
	if cfg.GetAPIKey("nonexistent-provider-xyz") != "" {
		t.Error("GetAPIKey for unknown provider should return empty string")
	}
}

// TestListAvailableProviders returns only providers with actual credentials —
// so in a clean test environment the slice may be empty. We just check it
// doesn't panic and that every returned name is non-empty.
func TestListAvailableProviders(t *testing.T) {
	cfg := &Config{}
	providers := cfg.ListAvailableProviders()

	for _, p := range providers {
		if p == "" {
			t.Error("ListAvailableProviders returned an empty provider name")
		}
	}

	// When a key is configured via the legacy field it should appear.
	cfgWithKey := &Config{OpenAIAPIKey: "sk-test"}
	provs := cfgWithKey.ListAvailableProviders()
	found := false
	for _, p := range provs {
		if p == "openai" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'openai' in ListAvailableProviders when OpenAIAPIKey is set")
	}
}

// TestDefaultModelsFilled ensures DefaultModels map is populated with required fields.
func TestDefaultModelsFilled(t *testing.T) {
	for prov, info := range DefaultModels {
		if info.ID == "" {
			t.Errorf("DefaultModels[%q].ID is empty", prov)
		}
		if info.ContextWindow <= 0 {
			t.Errorf("DefaultModels[%q].ContextWindow should be positive", prov)
		}
	}
}

// TestProviderRegistryNotEmpty ensures the provider registry has entries.
func TestProviderRegistryNotEmpty(t *testing.T) {
	if len(ProviderRegistry) == 0 {
		t.Error("ProviderRegistry should not be empty")
	}
	for _, p := range ProviderRegistry {
		if p.Key == "" {
			t.Error("ProviderRegistry entry has empty Key")
		}
		if p.Name == "" {
			t.Error("ProviderRegistry entry has empty Name")
		}
	}
}
