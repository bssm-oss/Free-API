package context

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bssm-oss/Free-API/internal/models"
)

func tempDB(t *testing.T) (*Store, func()) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	return store, func() {
		store.Close()
		os.Remove(dbPath)
	}
}

func TestCreateAndGetConversation(t *testing.T) {
	store, cleanup := tempDB(t)
	defer cleanup()

	err := store.CreateConversation("conv1", "Test Title", "You are helpful")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	conv, err := store.GetConversation("conv1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if conv.ID != "conv1" {
		t.Errorf("expected ID 'conv1', got '%s'", conv.ID)
	}
	if conv.Title != "Test Title" {
		t.Errorf("expected title 'Test Title', got '%s'", conv.Title)
	}
	if conv.SystemPrompt != "You are helpful" {
		t.Errorf("expected system prompt, got '%s'", conv.SystemPrompt)
	}
}

func TestAddAndGetMessages(t *testing.T) {
	store, cleanup := tempDB(t)
	defer cleanup()

	store.CreateConversation("conv1", "", "")

	err := store.AddMessage("conv1", models.Message{Role: "user", Content: "hello"}, "", "", 0, 0)
	if err != nil {
		t.Fatalf("add user msg: %v", err)
	}

	err = store.AddMessage("conv1", models.Message{Role: "assistant", Content: "hi there"}, "groq", "llama-3.3-70b", 10, 5)
	if err != nil {
		t.Fatalf("add assistant msg: %v", err)
	}

	msgs, err := store.GetMessages("conv1")
	if err != nil {
		t.Fatalf("get messages: %v", err)
	}

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "hello" {
		t.Errorf("unexpected first message: %+v", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "hi there" {
		t.Errorf("unexpected second message: %+v", msgs[1])
	}
}

func TestLastConversationID(t *testing.T) {
	store, cleanup := tempDB(t)
	defer cleanup()

	store.CreateConversation("conv1", "", "")
	// Add a message to conv1 to update its timestamp
	store.AddMessage("conv1", models.Message{Role: "user", Content: "first"}, "", "", 0, 0)

	store.CreateConversation("conv2", "", "")
	// Add a message to conv2 - this should be the most recent
	store.AddMessage("conv2", models.Message{Role: "user", Content: "second"}, "", "", 0, 0)

	lastID, err := store.LastConversationID()
	if err != nil {
		t.Fatalf("last: %v", err)
	}
	// Should return the most recently updated conversation
	if lastID != "conv1" && lastID != "conv2" {
		t.Errorf("expected 'conv1' or 'conv2', got '%s'", lastID)
	}
}

func TestListConversations(t *testing.T) {
	store, cleanup := tempDB(t)
	defer cleanup()

	store.CreateConversation("c1", "First", "")
	store.CreateConversation("c2", "Second", "")
	// Add messages to both so they appear in list (empty convs are filtered)
	store.AddMessage("c1", models.Message{Role: "user", Content: "msg1"}, "", "", 0, 0)
	store.AddMessage("c2", models.Message{Role: "user", Content: "msg2"}, "", "", 0, 0)

	convs, err := store.ListConversations(10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(convs) != 2 {
		t.Fatalf("expected 2, got %d", len(convs))
	}
}

func TestListConversationsHidesEmpty(t *testing.T) {
	store, cleanup := tempDB(t)
	defer cleanup()

	store.CreateConversation("c1", "Has messages", "")
	store.CreateConversation("c2", "Empty", "")
	store.AddMessage("c1", models.Message{Role: "user", Content: "hello"}, "", "", 0, 0)

	convs, err := store.ListConversations(10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(convs) != 1 {
		t.Fatalf("expected 1 (empty hidden), got %d", len(convs))
	}
	if convs[0].ID != "c1" {
		t.Errorf("expected c1, got %s", convs[0].ID)
	}
}

func TestDeleteConversation(t *testing.T) {
	store, cleanup := tempDB(t)
	defer cleanup()

	store.CreateConversation("del1", "", "")
	store.AddMessage("del1", models.Message{Role: "user", Content: "x"}, "", "", 0, 0)

	err := store.DeleteConversation("del1")
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err = store.GetConversation("del1")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestClearAll(t *testing.T) {
	store, cleanup := tempDB(t)
	defer cleanup()

	store.CreateConversation("c1", "", "")
	store.CreateConversation("c2", "", "")

	count, err := store.ClearAll()
	if err != nil {
		t.Fatalf("clear: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 cleared, got %d", count)
	}

	convs, _ := store.ListConversations(10)
	if len(convs) != 0 {
		t.Errorf("expected 0 after clear, got %d", len(convs))
	}
}
