package provider

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/bssm-oss/Free-API/internal/logging"
	"github.com/bssm-oss/Free-API/internal/models"
)

// Rotator handles automatic provider rotation on rate limits.
type Rotator struct {
	registry *Registry
}

// NewRotator creates a new rotator with a registry.
func NewRotator(registry *Registry) *Rotator {
	return &Rotator{registry: registry}
}

// Chat tries each provider in priority order until one succeeds.
func (r *Rotator) Chat(ctx context.Context, messages []models.Message, opts models.ChatOptions) (*models.Response, error) {
	providers := r.registry.GetByPriority()
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers configured. Run: freeapi config set <provider>.api_key <key>")
	}

	start := time.Now()
	logging.Debug("provider.rotation_start", map[string]any{
		"mode":           "chat",
		"provider_count": len(providers),
		"message_count":  len(messages),
		"stream":         opts.Stream,
		"model_override": opts.Model,
	})

	var lastErr error
	for _, p := range providers {
		available := p.IsAvailable()
		logging.Debug("provider.attempt", map[string]any{
			"mode":      "chat",
			"provider":  p.Name(),
			"available": available,
			"model":     p.DefaultModel(),
		})
		if !available {
			rl := p.RateLimitStatus()
			logging.Info("provider.skipped", map[string]any{
				"mode":         "chat",
				"provider":     p.Name(),
				"available":    false,
				"rate_limited": rl.IsLimited,
				"reset_at":     rl.ResetAt.Format(time.RFC3339Nano),
			})
			continue
		}

		resp, err := p.Chat(ctx, messages, opts)
		if err != nil {
			var rle *RateLimitError
			if errors.As(err, &rle) {
				logging.Info("provider.rate_limited", map[string]any{
					"mode":        "chat",
					"provider":    p.Name(),
					"retry_after": rle.RetryAfter.Format(time.RFC3339Nano),
					"error":       err.Error(),
				})
				fmt.Fprintf(os.Stderr, "⚠️  %s rate limited, trying next provider...\n", p.Name())
				lastErr = err
				continue
			}
			// Non-rate-limit error - still try next provider
			logging.Error("provider.error", map[string]any{
				"mode":     "chat",
				"provider": p.Name(),
				"error":    err.Error(),
			})
			fmt.Fprintf(os.Stderr, "⚠️  %s error: %v, trying next...\n", p.Name(), err)
			lastErr = err
			continue
		}

		logging.Info("provider.success", map[string]any{
			"mode":         "chat",
			"provider":     p.Name(),
			"elapsed_ms":   time.Since(start).Milliseconds(),
			"response_len": len(resp.Content),
			"tokens_in":    resp.TokensIn,
			"tokens_out":   resp.TokensOut,
		})
		return resp, nil
	}

	if lastErr != nil {
		logging.Error("provider.rotation_failed", map[string]any{
			"mode":       "chat",
			"elapsed_ms": time.Since(start).Milliseconds(),
			"error":      lastErr.Error(),
		})
		return nil, fmt.Errorf("all providers failed. Last error: %w", lastErr)
	}
	logging.Error("provider.rotation_unavailable", map[string]any{
		"mode":       "chat",
		"elapsed_ms": time.Since(start).Milliseconds(),
	})
	return nil, fmt.Errorf("no available providers. Run 'freeapi setup' or set keys with:\n  freeapi config set gemini.api_key <key>\n  Environment vars: GEMINI_API_KEY, GROQ_API_KEY, CEREBRAS_API_KEY, etc.")
}

// ChatStream tries each provider in priority order for streaming.
func (r *Rotator) ChatStream(ctx context.Context, messages []models.Message, opts models.ChatOptions) (<-chan models.StreamChunk, string, error) {
	providers := r.registry.GetByPriority()
	if len(providers) == 0 {
		return nil, "", fmt.Errorf("no providers configured")
	}

	start := time.Now()
	logging.Debug("provider.rotation_start", map[string]any{
		"mode":           "stream",
		"provider_count": len(providers),
		"message_count":  len(messages),
		"stream":         true,
		"model_override": opts.Model,
	})

	var lastErr error
	for _, p := range providers {
		available := p.IsAvailable()
		logging.Debug("provider.attempt", map[string]any{
			"mode":      "stream",
			"provider":  p.Name(),
			"available": available,
			"model":     p.DefaultModel(),
		})
		if !available {
			rl := p.RateLimitStatus()
			logging.Info("provider.skipped", map[string]any{
				"mode":         "stream",
				"provider":     p.Name(),
				"available":    false,
				"rate_limited": rl.IsLimited,
				"reset_at":     rl.ResetAt.Format(time.RFC3339Nano),
			})
			continue
		}

		ch, err := p.ChatStream(ctx, messages, opts)
		if err != nil {
			var rle *RateLimitError
			if errors.As(err, &rle) {
				logging.Info("provider.rate_limited", map[string]any{
					"mode":        "stream",
					"provider":    p.Name(),
					"retry_after": rle.RetryAfter.Format(time.RFC3339Nano),
					"error":       err.Error(),
				})
				fmt.Fprintf(os.Stderr, "⚠️  %s rate limited, trying next provider...\n", p.Name())
				lastErr = err
				continue
			}
			logging.Error("provider.error", map[string]any{
				"mode":     "stream",
				"provider": p.Name(),
				"error":    err.Error(),
			})
			fmt.Fprintf(os.Stderr, "⚠️  %s error: %v, trying next...\n", p.Name(), err)
			lastErr = err
			continue
		}

		logging.Info("provider.success", map[string]any{
			"mode":       "stream",
			"provider":   p.Name(),
			"elapsed_ms": time.Since(start).Milliseconds(),
		})
		return ch, p.Name(), nil
	}

	if lastErr != nil {
		logging.Error("provider.rotation_failed", map[string]any{
			"mode":       "stream",
			"elapsed_ms": time.Since(start).Milliseconds(),
			"error":      lastErr.Error(),
		})
		return nil, "", fmt.Errorf("all providers failed. Last error: %w", lastErr)
	}
	logging.Error("provider.rotation_unavailable", map[string]any{
		"mode":       "stream",
		"elapsed_ms": time.Since(start).Milliseconds(),
	})
	return nil, "", fmt.Errorf("no available providers. Run 'freeapi setup' to configure API keys")
}

// ChatWithProvider uses a specific named provider.
func (r *Rotator) ChatWithProvider(ctx context.Context, providerName string, messages []models.Message, opts models.ChatOptions) (*models.Response, error) {
	p, err := r.registry.GetByName(providerName)
	if err != nil {
		logging.Error("provider.lookup_error", map[string]any{
			"mode":     "chat",
			"provider": providerName,
			"error":    err.Error(),
		})
		return nil, err
	}
	if !p.IsAvailable() {
		logging.Info("provider.skipped", map[string]any{
			"mode":         "chat",
			"provider":     providerName,
			"available":    false,
			"rate_limited": p.RateLimitStatus().IsLimited,
		})
		return nil, fmt.Errorf("%s is not available (no key or rate limited)", providerName)
	}
	return p.Chat(ctx, messages, opts)
}

// ChatStreamWithProvider uses a specific named provider for streaming.
func (r *Rotator) ChatStreamWithProvider(ctx context.Context, providerName string, messages []models.Message, opts models.ChatOptions) (<-chan models.StreamChunk, error) {
	p, err := r.registry.GetByName(providerName)
	if err != nil {
		logging.Error("provider.lookup_error", map[string]any{
			"mode":     "stream",
			"provider": providerName,
			"error":    err.Error(),
		})
		return nil, err
	}
	if !p.IsAvailable() {
		logging.Info("provider.skipped", map[string]any{
			"mode":         "stream",
			"provider":     providerName,
			"available":    false,
			"rate_limited": p.RateLimitStatus().IsLimited,
		})
		return nil, fmt.Errorf("%s is not available", providerName)
	}
	return p.ChatStream(ctx, messages, opts)
}

// Status returns info about all providers.
func (r *Rotator) Status() []ProviderStatus {
	var statuses []ProviderStatus
	for _, p := range r.registry.GetByPriority() {
		available := p.IsAvailable()
		rl := p.RateLimitStatus()
		statuses = append(statuses, ProviderStatus{
			Name:      p.Name(),
			Model:     p.DefaultModel(),
			Available: available,
			RateLimit: rl,
		})
	}
	return statuses
}

// ProviderStatus is a summary of a provider's state.
type ProviderStatus struct {
	Name      string
	Model     string
	Available bool
	RateLimit models.RateLimitInfo
}
