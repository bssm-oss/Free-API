package provider

import (
	"context"

	"github.com/heodongun/freeapi/internal/models"
)

// Provider is the interface all LLM providers must implement.
type Provider interface {
	// Name returns the provider identifier.
	Name() string

	// Chat sends messages and returns a response.
	Chat(ctx context.Context, messages []models.Message, opts models.ChatOptions) (*models.Response, error)

	// ChatStream sends messages and returns a streaming channel.
	ChatStream(ctx context.Context, messages []models.Message, opts models.ChatOptions) (<-chan models.StreamChunk, error)

	// IsAvailable returns true if the provider can accept requests.
	IsAvailable() bool

	// RateLimitStatus returns current rate limit info.
	RateLimitStatus() models.RateLimitInfo

	// MarkRateLimited marks this provider as rate-limited until resetAt.
	MarkRateLimited(info models.RateLimitInfo)

	// DefaultModel returns the configured default model.
	DefaultModel() string
}
