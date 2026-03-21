package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bssm-oss/Free-API/internal/models"
)

func TestNewCloudflareBuildsOpenAICompatBaseURL(t *testing.T) {
	p := NewCloudflare("token", "acct-123", "", "@cf/meta/llama")

	if !p.IsAvailable() {
		t.Fatal("expected cloudflare provider with token and account to be available")
	}

	want := "https://api.cloudflare.com/client/v4/accounts/acct-123/ai/v1"
	if p.baseURL != want {
		t.Fatalf("expected baseURL %q, got %q", want, p.baseURL)
	}
}

func TestNewCloudflareAllowsExplicitBaseURLWithoutAccountID(t *testing.T) {
	p := NewCloudflare("token", "", "https://example.com/custom", "@cf/meta/llama")

	if !p.IsAvailable() {
		t.Fatal("expected explicit baseURL to make cloudflare provider available")
	}
}

func TestNewCloudflareUnavailableWithoutRouteInfo(t *testing.T) {
	p := NewCloudflare("token", "", "", "@cf/meta/llama")

	if p.IsAvailable() {
		t.Fatal("expected cloudflare provider without account ID or base URL to be unavailable")
	}
}

func TestCloudflareChatUsesOpenAICompatEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("unexpected auth header: %q", got)
		}

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req["model"] != "@cf/meta/llama" {
			t.Fatalf("unexpected model: %#v", req["model"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}],
			"model":"@cf/meta/llama",
			"usage":{"prompt_tokens":3,"completion_tokens":1}
		}`))
	}))
	defer server.Close()

	p := NewCloudflare("token", "", server.URL, "@cf/meta/llama")
	resp, err := p.Chat(context.Background(), []models.Message{{Role: "user", Content: "hi"}}, models.ChatOptions{})
	if err != nil {
		t.Fatalf("cloudflare chat: %v", err)
	}
	if resp.Content != "ok" {
		t.Fatalf("expected response content %q, got %q", "ok", resp.Content)
	}
	if resp.Provider != "cloudflare" {
		t.Fatalf("expected provider cloudflare, got %q", resp.Provider)
	}
}
