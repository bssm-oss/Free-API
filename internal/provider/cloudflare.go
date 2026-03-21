package provider

import "fmt"

// CloudflareProvider wraps Cloudflare Workers AI's OpenAI-compatible endpoint.
type CloudflareProvider struct {
	*OpenAICompatProvider
	accountID string
}

func NewCloudflare(apiKey, accountID, baseURL, defaultModel string) *CloudflareProvider {
	if baseURL == "" && accountID != "" {
		baseURL = fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/ai/v1", accountID)
	}

	return &CloudflareProvider{
		OpenAICompatProvider: NewOpenAICompat("cloudflare", apiKey, baseURL, defaultModel, nil),
		accountID:            accountID,
	}
}

func (p *CloudflareProvider) IsAvailable() bool {
	if p.baseURL == "" {
		return false
	}
	return p.OpenAICompatProvider.IsAvailable()
}
