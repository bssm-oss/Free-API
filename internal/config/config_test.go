package config

import (
	"os"
	"path/filepath"
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

func TestLoadAppliesCloudflareAccountIDFromEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLOUDFLARE_API_TOKEN", "token-from-env")
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "account-from-env")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	cf := cfg.Providers["cloudflare"]
	if cf.APIKey != "token-from-env" {
		t.Fatalf("expected cloudflare token from env, got %q", cf.APIKey)
	}
	if cf.AccountID != "account-from-env" {
		t.Fatalf("expected cloudflare account from env, got %q", cf.AccountID)
	}
}

func TestLoadRawDoesNotApplyEnvOverrides(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GROQ_API_KEY", "env-only-key")

	cfg, err := LoadRaw()
	if err != nil {
		t.Fatalf("load raw: %v", err)
	}

	if got := cfg.Providers["groq"].APIKey; got != "" {
		t.Fatalf("expected raw load to ignore env key, got %q", got)
	}
}

func TestLoadPreservesDefaultProvidersWhenConfigFileIsPartial(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := os.MkdirAll(filepath.Dir(ConfigPath()), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}

	data := []byte("providers:\n  groq:\n    model: custom-model\n")
	if err := os.WriteFile(ConfigPath(), data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadRaw()
	if err != nil {
		t.Fatalf("load raw: %v", err)
	}

	if _, ok := cfg.Providers["gemini"]; !ok {
		t.Fatal("expected missing providers to be restored from defaults")
	}
	if got := cfg.Providers["groq"].Model; got != "custom-model" {
		t.Fatalf("expected custom groq model, got %q", got)
	}
}
