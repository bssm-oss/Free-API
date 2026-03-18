package provider

import (
	"fmt"
	"sort"

	"github.com/heodongun/freeapi/internal/config"
	"github.com/heodongun/freeapi/internal/models"
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
	r.providers = append(r.providers, cliProviders...)

	// 2. API-based providers sorted by priority
	type entry struct {
		name string
		cfg  models.ProviderConfig
	}
	var entries []entry
	for name, pcfg := range cfg.Providers {
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
			"HTTP-Referer": "https://github.com/heodongun/freeapi",
			"X-Title":      "freeapi",
		}
		return NewOpenAICompat("openrouter", cfg.APIKey, cfg.BaseURL, cfg.Model, headers)

	case "github":
		return NewOpenAICompat("github", cfg.APIKey, cfg.BaseURL, cfg.Model, nil)

	case "cohere":
		return NewCohere(cfg.APIKey, cfg.BaseURL, cfg.Model)

	case "cloudflare":
		// Cloudflare uses a different URL pattern - skip for now if no account_id
		return nil

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
