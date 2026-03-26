package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoggerWritesJSONLines(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "freeapi.log")
	Configure(logPath, "debug")

	Info("chat.start", map[string]any{
		"provider":    "gemini-cli",
		"message_len": 5,
	})

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}

	text := string(data)
	for _, want := range []string{
		`"event":"chat.start"`,
		`"level":"info"`,
		`"provider":"gemini-cli"`,
		`"message_len":5`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("log missing %q:\n%s", want, text)
		}
	}
}

func TestLoggerFiltersByLevel(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "freeapi.log")
	Configure(logPath, "error")

	Info("chat.start", map[string]any{"provider": "gemini-cli"})
	Error("chat.error", map[string]any{"provider": "gemini-cli"})

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}

	text := string(data)
	if strings.Contains(text, `"event":"chat.start"`) {
		t.Fatalf("info log should not be written at error level:\n%s", text)
	}
	if !strings.Contains(text, `"event":"chat.error"`) {
		t.Fatalf("error log missing:\n%s", text)
	}
}
