package context

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/bssm-oss/Free-API/internal/models"
)

// Manager handles conversation context with trimming strategies.
type Manager struct {
	store        *Store
	maxMessages  int
	strategy     string
	systemPrompt string
}

// NewManager creates a context manager.
func NewManager(store *Store, maxMessages int, strategy, systemPrompt string) *Manager {
	if maxMessages <= 0 {
		maxMessages = 50
	}
	if strategy == "" {
		strategy = "sliding_window"
	}
	return &Manager{
		store:        store,
		maxMessages:  maxMessages,
		strategy:     strategy,
		systemPrompt: systemPrompt,
	}
}

// NewConversation creates a new conversation and returns its ID.
func (m *Manager) NewConversation(systemPrompt string) (string, error) {
	id := generateID()
	if systemPrompt == "" {
		systemPrompt = m.systemPrompt
	}
	if err := m.store.CreateConversation(id, "", systemPrompt); err != nil {
		return "", err
	}
	return id, nil
}

// GetOrContinue returns the conversation ID to use.
// If continueLastConv is true, returns the last conversation ID.
// If convID is specified, returns that. Otherwise creates a new one.
func (m *Manager) GetOrContinue(convID string, continueLastConv bool, systemPrompt string) (string, bool, error) {
	if convID != "" {
		resolvedID, err := m.store.ResolveConversationID(convID)
		if err != nil {
			return "", false, err
		}
		return resolvedID, true, nil
	}

	if continueLastConv {
		id, err := m.store.LastConversationID()
		if err != nil {
			// No prior conversation, create new
			newID, err := m.NewConversation(systemPrompt)
			return newID, false, err
		}
		return id, true, nil
	}

	// New conversation
	id, err := m.NewConversation(systemPrompt)
	return id, false, err
}

// BuildMessages returns the message history for an API call.
// Includes system prompt + conversation history + new user message.
func (m *Manager) BuildMessages(convID, userInput string) ([]models.Message, error) {
	conv, err := m.store.GetConversation(convID)
	if err != nil {
		return nil, err
	}

	var msgs []models.Message

	// System prompt
	sysPrompt := conv.SystemPrompt
	if sysPrompt == "" {
		sysPrompt = m.systemPrompt
	}
	if sysPrompt != "" {
		msgs = append(msgs, models.Message{Role: "system", Content: sysPrompt})
	}

	// History (apply sliding window)
	history := conv.Messages
	if m.strategy == "sliding_window" && len(history) > m.maxMessages {
		history = history[len(history)-m.maxMessages:]
	}
	msgs = append(msgs, history...)

	// New user message
	msgs = append(msgs, models.Message{Role: "user", Content: userInput})

	return msgs, nil
}

// SaveExchange stores the user message and assistant response atomically.
func (m *Manager) SaveExchange(convID, userInput, assistantOutput, providerName, modelName string, tokensIn, tokensOut int) error {
	tx, err := m.store.DB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Save user message
	if _, err := tx.Exec(
		`INSERT INTO messages (conversation_id, role, content, provider, model, tokens_in, tokens_out) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		convID, "user", userInput, "", "", 0, 0,
	); err != nil {
		return err
	}

	// Save assistant response
	if _, err := tx.Exec(
		`INSERT INTO messages (conversation_id, role, content, provider, model, tokens_in, tokens_out) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		convID, "assistant", assistantOutput, providerName, modelName, tokensIn, tokensOut,
	); err != nil {
		return err
	}

	// Update timestamp
	if _, err := tx.Exec(`UPDATE conversations SET updated_at = CURRENT_TIMESTAMP WHERE id = ?`, convID); err != nil {
		return err
	}

	// Auto-set title from first user message if empty
	var title string
	tx.QueryRow(`SELECT title FROM conversations WHERE id = ?`, convID).Scan(&title)
	if title == "" {
		t := userInput
		if len(t) > 60 {
			t = t[:60] + "..."
		}
		tx.Exec(`UPDATE conversations SET title = ? WHERE id = ?`, t, convID)
	}

	return tx.Commit()
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
