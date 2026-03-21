package models

import "time"

// Message represents a single chat message.
type Message struct {
	Role    string `json:"role"` // system, user, assistant
	Content string `json:"content"`
}

// Conversation holds a full conversation with metadata.
type Conversation struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	SystemPrompt string    `json:"system_prompt"`
	Messages     []Message `json:"messages"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ChatOptions holds options for a chat request.
type ChatOptions struct {
	Model       string  `json:"model,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Stream      bool    `json:"stream,omitempty"`
}

// Response holds a provider's chat response.
type Response struct {
	Content      string `json:"content"`
	Model        string `json:"model"`
	Provider     string `json:"provider"`
	TokensIn     int    `json:"tokens_in"`
	TokensOut    int    `json:"tokens_out"`
	FinishReason string `json:"finish_reason"`
}

// StreamChunk represents a single streaming chunk.
type StreamChunk struct {
	Content string
	Done    bool
	Error   error
}

// RateLimitInfo tracks rate limit state for a provider.
type RateLimitInfo struct {
	Remaining int
	ResetAt   time.Time
	IsLimited bool
}

// ProviderConfig holds config for a single provider.
type ProviderConfig struct {
	APIKey    string `yaml:"api_key"`
	AccountID string `yaml:"account_id,omitempty"`
	Model     string `yaml:"model"`
	Priority  int    `yaml:"priority"`
	Enabled   bool   `yaml:"enabled"`
	BaseURL   string `yaml:"base_url,omitempty"`
}
