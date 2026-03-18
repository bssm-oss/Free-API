package config

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if len(cfg.Providers) == 0 {
		t.Fatal("expected providers in default config")
	}

	gemini, ok := cfg.Providers["gemini"]
	if !ok {
		t.Fatal("expected gemini provider")
	}
	if gemini.Priority != 1 {
		t.Errorf("gemini priority: expected 1, got %d", gemini.Priority)
	}
	if !gemini.Enabled {
		t.Error("gemini should be enabled by default")
	}

	groq, ok := cfg.Providers["groq"]
	if !ok {
		t.Fatal("expected groq provider")
	}
	if groq.Priority != 2 {
		t.Errorf("groq priority: expected 2, got %d", groq.Priority)
	}
}

func TestDefaultConfigHasAllProviders(t *testing.T) {
	cfg := DefaultConfig()
	expected := []string{"gemini", "groq", "cerebras", "mistral", "openrouter", "cohere", "github", "cloudflare"}

	for _, name := range expected {
		if _, ok := cfg.Providers[name]; !ok {
			t.Errorf("missing provider: %s", name)
		}
	}
}

func TestEnvVarMap(t *testing.T) {
	// Verify all providers have env var mappings
	cfg := DefaultConfig()
	for name := range cfg.Providers {
		if _, ok := EnvVarMap[name]; !ok {
			t.Errorf("no env var mapping for provider: %s", name)
		}
	}
}

func TestGeminiHasMultipleEnvVars(t *testing.T) {
	vars := EnvVarMap["gemini"]
	if len(vars) < 2 {
		t.Error("gemini should support GEMINI_API_KEY and GOOGLE_API_KEY")
	}
}
