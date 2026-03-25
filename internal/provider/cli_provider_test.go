package provider

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestCLIProviderChatUsesPerProviderTimeout(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "slow-cli")
	script := "#!/bin/sh\nsleep 1\nprintf 'too late\\n'\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	oldTimeout := cliProviderMaxRuntime
	cliProviderMaxRuntime = 50 * time.Millisecond
	defer func() { cliProviderMaxRuntime = oldTimeout }()

	p := NewCLIProvider("slow-cli", bin, func(prompt string) []string {
		return []string{prompt}
	})

	_, err := p.Chat(context.Background(), []models.Message{{Role: "user", Content: "hi"}}, models.ChatOptions{})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout error, got: %v", err)
	}
}
