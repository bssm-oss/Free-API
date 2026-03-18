package context

import (
	"testing"
)

func TestNewConversation(t *testing.T) {
	store, cleanup := tempDB(t)
	defer cleanup()

	mgr := NewManager(store, 50, "sliding_window", "default system")

	id, err := mgr.NewConversation("")
	if err != nil {
		t.Fatalf("new conv: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty ID")
	}
	if len(id) != 16 { // 8 bytes hex = 16 chars
		t.Errorf("expected 16 char ID, got %d: %s", len(id), id)
	}
}

func TestGetOrContinueNew(t *testing.T) {
	store, cleanup := tempDB(t)
	defer cleanup()

	mgr := NewManager(store, 50, "sliding_window", "sys")

	id, isContinue, err := mgr.GetOrContinue("", false, "")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if isContinue {
		t.Error("expected new conversation, not continue")
	}
	if id == "" {
		t.Error("expected ID")
	}
}

func TestGetOrContinueLast(t *testing.T) {
	store, cleanup := tempDB(t)
	defer cleanup()

	mgr := NewManager(store, 50, "sliding_window", "sys")

	// Create first conversation
	id1, _ := mgr.NewConversation("sys")

	// Continue last should return id1
	id2, isContinue, err := mgr.GetOrContinue("", true, "")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !isContinue {
		t.Error("expected continue=true")
	}
	if id2 != id1 {
		t.Errorf("expected %s, got %s", id1, id2)
	}
}

func TestBuildMessages(t *testing.T) {
	store, cleanup := tempDB(t)
	defer cleanup()

	mgr := NewManager(store, 50, "sliding_window", "You are helpful")
	id, _ := mgr.NewConversation("You are helpful")

	msgs, err := mgr.BuildMessages(id, "hello")
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	// Should have: system + user
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "system" {
		t.Errorf("expected system, got %s", msgs[0].Role)
	}
	if msgs[1].Role != "user" || msgs[1].Content != "hello" {
		t.Errorf("unexpected user msg: %+v", msgs[1])
	}
}

func TestBuildMessagesSlidingWindow(t *testing.T) {
	store, cleanup := tempDB(t)
	defer cleanup()

	mgr := NewManager(store, 4, "sliding_window", "sys") // max 4 history messages
	id, _ := mgr.NewConversation("sys")

	// Add 6 messages (3 exchanges)
	for i := 0; i < 3; i++ {
		mgr.SaveExchange(id, "q", "a", "test", "model", 0, 0)
	}

	msgs, _ := mgr.BuildMessages(id, "new question")

	// system + 4 history (sliding window) + 1 new user = 6
	if len(msgs) != 6 {
		t.Errorf("expected 6 messages with sliding window, got %d", len(msgs))
	}
}

func TestSaveExchangeAutoTitle(t *testing.T) {
	store, cleanup := tempDB(t)
	defer cleanup()

	mgr := NewManager(store, 50, "sliding_window", "sys")
	id, _ := mgr.NewConversation("sys")

	mgr.SaveExchange(id, "What is Go programming?", "Go is...", "groq", "llama", 10, 5)

	conv, _ := store.GetConversation(id)
	if conv.Title == "" {
		t.Error("expected auto-generated title")
	}
}
