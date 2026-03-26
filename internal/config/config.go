package config

import (
	"os"
	"path/filepath"

	"github.com/bssm-oss/Free-API/internal/models"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Providers           map[string]models.ProviderConfig `yaml:"providers"`
	DefaultSystemPrompt string                           `yaml:"default_system_prompt"`
	MaxContextMessages  int                              `yaml:"max_context_messages"`
	ContextStrategy     string                           `yaml:"context_strategy"`
	DBPath              string                           `yaml:"db_path"`
	LogPath             string                           `yaml:"log_path"`
	LogLevel            string                           `yaml:"log_level"`
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
		LogPath:             "",
		LogLevel:            "info",
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

func DefaultLogPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "freeapi", "logs", "freeapi.log")
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

var ProviderFieldEnvVarMap = map[string]map[string][]string{
	"cloudflare": {
		"account_id": {"CLOUDFLARE_ACCOUNT_ID", "CF_ACCOUNT_ID"},
	},
}

func load(applyEnv bool) (*Config, error) {
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

	applyProviderDefaults(cfg)

	if applyEnv {
		applyEnvOverrides(cfg)
	}

	if cfg.DBPath == "" {
		cfg.DBPath = DefaultDBPath()
	}
	if cfg.LogPath == "" {
		cfg.LogPath = DefaultLogPath()
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	return cfg, nil
}

// Load reads config from disk and applies environment variable overrides.
func Load() (*Config, error) {
	return load(true)
}

// LoadRaw reads config from disk without applying environment variable overrides.
// Use this when mutating the config file so env-sourced secrets are not persisted.
func LoadRaw() (*Config, error) {
	return load(false)
}

func applyProviderDefaults(cfg *Config) {
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]models.ProviderConfig)
	}

	defaults := DefaultConfig()
	for name, def := range defaults.Providers {
		cur, ok := cfg.Providers[name]
		if !ok {
			cfg.Providers[name] = def
			continue
		}
		if cur.Model == "" {
			cur.Model = def.Model
		}
		if cur.Priority == 0 {
			cur.Priority = def.Priority
		}
		if cur.BaseURL == "" {
			cur.BaseURL = def.BaseURL
		}
		cfg.Providers[name] = cur
	}
}

func applyEnvOverrides(cfg *Config) {
	// Fill API keys from env vars if not set in config
	for name, envVars := range EnvVarMap {
		p, ok := cfg.Providers[name]
		if !ok || p.APIKey != "" {
			continue
		}
		for _, envVar := range envVars {
			if v := os.Getenv(envVar); v != "" {
				p.APIKey = v
				cfg.Providers[name] = p
				break
			}
		}
	}

	for name, fieldEnvMap := range ProviderFieldEnvVarMap {
		p, ok := cfg.Providers[name]
		if !ok {
			continue
		}
		for field, envVars := range fieldEnvMap {
			switch field {
			case "account_id":
				if p.AccountID != "" {
					continue
				}
				for _, envVar := range envVars {
					if v := os.Getenv(envVar); v != "" {
						p.AccountID = v
						cfg.Providers[name] = p
						break
					}
				}
			}
		}
	}

	if v := os.Getenv("FREEAPI_LOG_PATH"); v != "" {
		cfg.LogPath = filepath.Clean(v)
	}
	if v := os.Getenv("FREEAPI_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
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
