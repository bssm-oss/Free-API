package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bssm-oss/Free-API/internal/models"
)

func TestCohereChatStreamParsesContentDeltas(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected auth header: %q", got)
		}

		var req cohereRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !req.Stream {
			t.Fatal("expected streaming request")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: content-delta\n")
		fmt.Fprint(w, "data: {\"type\":\"content-delta\",\"delta\":{\"message\":{\"content\":{\"text\":\"hello \"}}}}\n\n")
		fmt.Fprint(w, "event: content-delta\n")
		fmt.Fprint(w, "data: {\"type\":\"content-delta\",\"delta\":{\"message\":{\"content\":{\"text\":\"world\"}}}}\n\n")
		fmt.Fprint(w, "event: message-end\n")
		fmt.Fprint(w, "data: {\"type\":\"message-end\"}\n\n")
	}))
	defer server.Close()

	p := NewCohere("test-key", server.URL, "command-r-plus")
	ch, err := p.ChatStream(context.Background(), []models.Message{{Role: "user", Content: "hi"}}, models.ChatOptions{})
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}

	var content strings.Builder
	for chunk := range ch {
		if chunk.Error != nil {
			t.Fatalf("unexpected chunk error: %v", chunk.Error)
		}
		if chunk.Done {
			break
		}
		content.WriteString(chunk.Content)
	}

	if got := content.String(); got != "hello world" {
		t.Fatalf("expected streamed content %q, got %q", "hello world", got)
	}
}
