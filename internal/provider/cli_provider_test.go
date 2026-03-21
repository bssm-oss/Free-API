package provider

import (
	"strings"
	"testing"

	"github.com/bssm-oss/Free-API/internal/models"
)

func TestExtractPromptIncludesFullConversation(t *testing.T) {
	prompt := extractPrompt([]models.Message{
		{Role: "system", Content: "Respond concisely."},
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi"},
		{Role: "user", Content: "What did I just say?"},
	})

	expectedParts := []string{
		"System instructions:\nRespond concisely.",
		"Conversation:",
		"User:\nHello",
		"Assistant:\nHi",
		"User:\nWhat did I just say?",
	}
	for _, part := range expectedParts {
		if !strings.Contains(prompt, part) {
			t.Fatalf("expected prompt to include %q, got:\n%s", part, prompt)
		}
	}

	if !strings.HasSuffix(prompt, "Assistant:") {
		t.Fatalf("expected prompt to end with assistant cue, got:\n%s", prompt)
	}
}
