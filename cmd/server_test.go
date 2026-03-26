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

func TestServerRootRedirectsToSwagger(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()

	newServerMux().ServeHTTP(resp, req)

	if resp.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d", resp.Code)
	}
	if got := resp.Header().Get("Location"); got != "/swagger" {
		t.Fatalf("expected redirect to /swagger, got %q", got)
	}
}

func TestSwaggerUIPageServesHTML(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/swagger", nil)
	resp := httptest.NewRecorder()

	newServerMux().ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if ctype := resp.Header().Get("Content-Type"); !strings.Contains(ctype, "text/html") {
		t.Fatalf("expected HTML content-type, got %q", ctype)
	}
	body := resp.Body.String()
	for _, want := range []string{"SwaggerUIBundle", "/openapi.json", "freeapi API Docs"} {
		if !strings.Contains(body, want) {
			t.Fatalf("swagger page missing %q:\n%s", want, body)
		}
	}
}

func TestOpenAPISpecServesJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	resp := httptest.NewRecorder()

	newServerMux().ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var spec map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&spec); err != nil {
		t.Fatalf("decode spec: %v", err)
	}

	if got := spec["openapi"]; got != "3.0.3" {
		t.Fatalf("expected openapi 3.0.3, got %#v", got)
	}

	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		t.Fatalf("expected paths object in spec, got %#v", spec["paths"])
	}
	if _, ok := paths["/freeapi/chat"]; !ok {
		t.Fatalf("expected /freeapi/chat path in spec, got %#v", paths)
	}
	if _, ok := paths["/healthz"]; !ok {
		t.Fatalf("expected /healthz path in spec, got %#v", paths)
	}
}

func TestServerChatRejectsInvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/freeapi/chat", strings.NewReader(`{"bad":true}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	newServerMux().ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "invalid JSON body") {
		t.Fatalf("expected invalid JSON error, got %s", resp.Body.String())
	}
}
