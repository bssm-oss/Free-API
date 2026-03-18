package config

import (
	"os"
	"path/filepath"

	"github.com/heodongun/freeapi/internal/models"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Providers          map[string]models.ProviderConfig `yaml:"providers"`
	DefaultSystemPrompt string                          `yaml:"default_system_prompt"`
	MaxContextMessages  int                             `yaml:"max_context_messages"`
	ContextStrategy     string                          `yaml:"context_strategy"`
	DBPath              string                          `yaml:"db_path"`
}

func DefaultConfig() *Config {
	return &Config{
		Providers: map[string]models.ProviderConfig{
			"gemini": {
				Model:    "gemini-2.5-flash",
				Priority: 1,
				Enabled:  true,
				BaseURL:  "https://generativelanguage.googleapis.com/v1beta",
			},
			"groq": {
				Model:    "llama-3.3-70b-versatile",
				Priority: 2,
				Enabled:  true,
				BaseURL:  "https://api.groq.com/openai/v1",
			},
			"cerebras": {
				Model:    "llama-3.3-70b",
				Priority: 3,
				Enabled:  true,
				BaseURL:  "https://api.cerebras.ai/v1",
			},
			"mistral": {
				Model:    "mistral-small-latest",
				Priority: 4,
				Enabled:  true,
				BaseURL:  "https://api.mistral.ai/v1",
			},
			"openrouter": {
				Model:    "deepseek/deepseek-r1:free",
				Priority: 5,
				Enabled:  true,
				BaseURL:  "https://openrouter.ai/api/v1",
			},
			"cohere": {
				Model:    "command-r-plus",
				Priority: 6,
				Enabled:  true,
				BaseURL:  "https://api.cohere.ai/v2",
			},
			"github": {
				Model:    "gpt-4o",
				Priority: 7,
				Enabled:  true,
				BaseURL:  "https://models.inference.ai.azure.com",
			},
			"cloudflare": {
				Model:    "@cf/meta/llama-3.3-70b-instruct-fp8-fast",
				Priority: 8,
				Enabled:  true,
			},
		},
		DefaultSystemPrompt: "You are a helpful assistant.",
		MaxContextMessages:  50,
		ContextStrategy:     "sliding_window",
		DBPath:              "",
	}
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "freeapi")
}

func ConfigPath() string {
	return filepath.Join(configDir(), "config.yaml")
}

func DefaultDBPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "freeapi", "conversations.db")
}

// EnvVarMap maps provider names to their environment variable names.
// Multiple env vars per provider are supported (checked in order).
var EnvVarMap = map[string][]string{
	"gemini":     {"GEMINI_API_KEY", "GOOGLE_API_KEY"},
	"groq":       {"GROQ_API_KEY"},
	"cerebras":   {"CEREBRAS_API_KEY"},
	"mistral":    {"MISTRAL_API_KEY"},
	"openrouter": {"OPENROUTER_API_KEY"},
	"cohere":     {"COHERE_API_KEY", "CO_API_KEY"},
	"github":     {"GITHUB_TOKEN"},
	"cloudflare": {"CLOUDFLARE_API_TOKEN", "CF_API_TOKEN"},
}

// Load reads config from disk, falling back to defaults.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		// Config file doesn't exist - continue with defaults
	} else {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	// Fill API keys from env vars if not set in config
	for name, envVars := range EnvVarMap {
		if p, ok := cfg.Providers[name]; ok && p.APIKey == "" {
			for _, envVar := range envVars {
				if v := os.Getenv(envVar); v != "" {
					p.APIKey = v
					cfg.Providers[name] = p
					break
				}
			}
		}
	}

	if cfg.DBPath == "" {
		cfg.DBPath = DefaultDBPath()
	}

	return cfg, nil
}

// Save writes config to disk.
func Save(cfg *Config) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigPath(), data, 0600)
}
