package provider

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bssm-oss/Free-API/internal/models"
)

// mockProvider implements Provider for testing.
type mockProvider struct {
	name      string
	model     string
	available bool
	response  *models.Response
	err       error
	rateLimit models.RateLimitInfo
}

func (m *mockProvider) Name() string         { return m.name }
func (m *mockProvider) DefaultModel() string { return m.model }
func (m *mockProvider) IsAvailable() bool    { return m.available }

func (m *mockProvider) Chat(ctx context.Context, messages []models.Message, opts models.ChatOptions) (*models.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *mockProvider) ChatStream(ctx context.Context, messages []models.Message, opts models.ChatOptions) (<-chan models.StreamChunk, error) {
	if m.err != nil {
		return nil, m.err
	}
	ch := make(chan models.StreamChunk, 2)
	go func() {
		defer close(ch)
		ch <- models.StreamChunk{Content: m.response.Content}
		ch <- models.StreamChunk{Done: true}
	}()
	return ch, nil
}

func (m *mockProvider) RateLimitStatus() models.RateLimitInfo { return m.rateLimit }
func (m *mockProvider) MarkRateLimited(info models.RateLimitInfo) {
	m.rateLimit = info
	m.available = false
}

func TestRotatorUsesFirstAvailable(t *testing.T) {
	reg := &Registry{
		providers: []Provider{
			&mockProvider{name: "first", available: true, response: &models.Response{Content: "hello", Provider: "first"}},
			&mockProvider{name: "second", available: true, response: &models.Response{Content: "world", Provider: "second"}},
		},
	}
	rotator := NewRotator(reg)

	resp, err := rotator.Chat(context.Background(), []models.Message{{Role: "user", Content: "hi"}}, models.ChatOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Provider != "first" {
		t.Errorf("expected provider 'first', got '%s'", resp.Provider)
	}
}

func TestRotatorFallsBackOnRateLimit(t *testing.T) {
	reg := &Registry{
		providers: []Provider{
			&mockProvider{
				name:      "limited",
				available: true,
				err:       &RateLimitError{Provider: "limited", RetryAfter: time.Now().Add(60 * time.Second)},
			},
			&mockProvider{name: "backup", available: true, response: &models.Response{Content: "from backup", Provider: "backup"}},
		},
	}
	rotator := NewRotator(reg)

	resp, err := rotator.Chat(context.Background(), []models.Message{{Role: "user", Content: "hi"}}, models.ChatOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Provider != "backup" {
		t.Errorf("expected provider 'backup', got '%s'", resp.Provider)
	}
}

func TestRotatorReturnsErrorWhenAllFail(t *testing.T) {
	reg := &Registry{
		providers: []Provider{
			&mockProvider{name: "a", available: true, err: fmt.Errorf("fail a")},
			&mockProvider{name: "b", available: true, err: fmt.Errorf("fail b")},
		},
	}
	rotator := NewRotator(reg)

	_, err := rotator.Chat(context.Background(), []models.Message{{Role: "user", Content: "hi"}}, models.ChatOptions{})
	if err == nil {
		t.Fatal("expected error when all providers fail")
	}
}

func TestRotatorSkipsUnavailable(t *testing.T) {
	reg := &Registry{
		providers: []Provider{
			&mockProvider{name: "down", available: false},
			&mockProvider{name: "up", available: true, response: &models.Response{Content: "ok", Provider: "up"}},
		},
	}
	rotator := NewRotator(reg)

	resp, err := rotator.Chat(context.Background(), []models.Message{{Role: "user", Content: "hi"}}, models.ChatOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Provider != "up" {
		t.Errorf("expected provider 'up', got '%s'", resp.Provider)
	}
}

func TestRotatorNoProviders(t *testing.T) {
	reg := &Registry{}
	rotator := NewRotator(reg)

	_, err := rotator.Chat(context.Background(), []models.Message{{Role: "user", Content: "hi"}}, models.ChatOptions{})
	if err == nil {
		t.Fatal("expected error with no providers")
	}
}

func TestRotatorStream(t *testing.T) {
	reg := &Registry{
		providers: []Provider{
			&mockProvider{name: "stream", available: true, response: &models.Response{Content: "streamed", Provider: "stream"}},
		},
	}
	rotator := NewRotator(reg)

	ch, name, err := rotator.ChatStream(context.Background(), []models.Message{{Role: "user", Content: "hi"}}, models.ChatOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "stream" {
		t.Errorf("expected provider 'stream', got '%s'", name)
	}

	var content string
	for chunk := range ch {
		if chunk.Done {
			break
		}
		content += chunk.Content
	}
	if content != "streamed" {
		t.Errorf("expected 'streamed', got '%s'", content)
	}
}
