package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServerChatEndpoint(t *testing.T) {
	stateDir := t.TempDir()
	binDir := t.TempDir()
	homeDir := t.TempDir()

	scriptPath := filepath.Join(binDir, "gemini")
	content := strings.ReplaceAll(fakeCLIRecorderScript("gemini", "gemini"), "__STATE_DIR__", stateDir)
	if err := os.WriteFile(scriptPath, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake gemini: %v", err)
	}

	t.Setenv("HOME", homeDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	mux := newServerMux()

	firstReq := httptest.NewRequest(http.MethodPost, "/freeapi/chat", strings.NewReader(`{"message":"hello"}`))
	firstReq.Header.Set("Content-Type", "application/json")
	firstResp := httptest.NewRecorder()
	mux.ServeHTTP(firstResp, firstReq)

	if firstResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", firstResp.Code, firstResp.Body.String())
	}

	var firstBody serverChatResponse
	if err := json.NewDecoder(firstResp.Body).Decode(&firstBody); err != nil {
		t.Fatalf("decode first response: %v", err)
	}
	if firstBody.Content != "gemini-response-1" {
		t.Fatalf("unexpected first content: %q", firstBody.Content)
	}
	if firstBody.Response != "gemini-response-1" {
		t.Fatalf("unexpected first response: %q", firstBody.Response)
	}
	if firstBody.Provider != "gemini-cli" {
		t.Fatalf("unexpected provider: %q", firstBody.Provider)
	}
	if firstBody.ConversationID == "" {
		t.Fatal("expected conversation ID")
	}

	secondReq := httptest.NewRequest(
		http.MethodPost,
		"/freeapi/chat",
		strings.NewReader(`{"message":"follow up","conversation_id":"`+firstBody.ConversationID+`"}`),
	)
	secondReq.Header.Set("Content-Type", "application/json")
	secondResp := httptest.NewRecorder()
	mux.ServeHTTP(secondResp, secondReq)

	if secondResp.Code != http.StatusOK {
		t.Fatalf("expected 200 on second request, got %d: %s", secondResp.Code, secondResp.Body.String())
	}

	prompt := readFile(t, filepath.Join(stateDir, "gemini_prompt_2.txt"))
	for _, want := range []string{
		"User:\nhello",
		"Assistant:\ngemini-response-1",
		"User:\nfollow up",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("second prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestServerChatRejectsInvalidMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/freeapi/chat", nil)
	resp := httptest.NewRecorder()

	newServerMux().ServeHTTP(resp, req)

	if resp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.Code)
	}
}
