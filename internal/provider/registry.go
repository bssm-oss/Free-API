package provider

import (
	"fmt"
	"sort"

	"github.com/bssm-oss/Free-API/internal/config"
	"github.com/bssm-oss/Free-API/internal/models"
)

// Registry manages all providers sorted by priority.
type Registry struct {
	providers []Provider
}

// NewRegistry creates providers from config + auto-detected CLIs.
func NewRegistry(cfg *config.Config) *Registry {
	r := &Registry{}

	// 1. CLI-based providers first (no API key needed, already installed)
	cliProviders := DetectCLIs()
	sort.Slice(cliProviders, func(i, j int) bool {
		return configuredPriority(cfg, cliProviders[i].Name(), DefaultCLIPriority(cliProviders[i].Name())) <
			configuredPriority(cfg, cliProviders[j].Name(), DefaultCLIPriority(cliProviders[j].Name()))
	})
	for _, p := range cliProviders {
		if providerEnabled(cfg, p.Name(), true) {
			r.providers = append(r.providers, p)
		}
	}

	// 2. API-based providers sorted by priority
	type entry struct {
		name string
		cfg  models.ProviderConfig
	}
	var entries []entry
	for name, pcfg := range cfg.Providers {
		if IsKnownCLI(name) {
			continue
		}
		if pcfg.Enabled {
			entries = append(entries, entry{name, pcfg})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].cfg.Priority < entries[j].cfg.Priority
	})

	for _, e := range entries {
		p := createProvider(e.name, e.cfg)
		if p != nil {
			r.providers = append(r.providers, p)
		}
	}

	return r
}

func providerEnabled(cfg *config.Config, name string, defaultValue bool) bool {
	if cfg == nil {
		return defaultValue
	}
	if pcfg, ok := cfg.Providers[name]; ok {
		return pcfg.Enabled
	}
	return defaultValue
}

func configuredPriority(cfg *config.Config, name string, defaultValue int) int {
	if cfg == nil {
		return defaultValue
	}
	if pcfg, ok := cfg.Providers[name]; ok && pcfg.Priority > 0 {
		return pcfg.Priority
	}
	return defaultValue
}

func createProvider(name string, cfg models.ProviderConfig) Provider {
	switch name {
	case "gemini":
		return NewGemini(cfg.APIKey, cfg.BaseURL, cfg.Model)

	case "groq":
		return NewOpenAICompat("groq", cfg.APIKey, cfg.BaseURL, cfg.Model, nil)

	case "cerebras":
		return NewOpenAICompat("cerebras", cfg.APIKey, cfg.BaseURL, cfg.Model, nil)

	case "mistral":
		return NewOpenAICompat("mistral", cfg.APIKey, cfg.BaseURL, cfg.Model, nil)

	case "openrouter":
		headers := map[string]string{
			"HTTP-Referer": "https://github.com/bssm-oss/Free-API",
			"X-Title":      "freeapi",
		}
		return NewOpenAICompat("openrouter", cfg.APIKey, cfg.BaseURL, cfg.Model, headers)

	case "github":
		return NewOpenAICompat("github", cfg.APIKey, cfg.BaseURL, cfg.Model, nil)

	case "cohere":
		return NewCohere(cfg.APIKey, cfg.BaseURL, cfg.Model)

	case "cloudflare":
		return NewCloudflare(cfg.APIKey, cfg.AccountID, cfg.BaseURL, cfg.Model)

	default:
		return nil
	}
}

// GetByPriority returns providers sorted by priority.
func (r *Registry) GetByPriority() []Provider {
	return r.providers
}

// GetByName returns a specific provider.
func (r *Registry) GetByName(name string) (Provider, error) {
	for _, p := range r.providers {
		if p.Name() == name {
			return p, nil
		}
	}
	return nil, fmt.Errorf("provider not found: %s", name)
}

// Available returns only currently available providers.
func (r *Registry) Available() []Provider {
	var avail []Provider
	for _, p := range r.providers {
		if p.IsAvailable() {
			avail = append(avail, p)
		}
	}
	return avail
}

// Count returns total number of registered providers.
func (r *Registry) Count() int {
	return len(r.providers)
}
