package context

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bssm-oss/Free-API/internal/logging"
	"github.com/bssm-oss/Free-API/internal/models"
	_ "modernc.org/sqlite"
)

// Store manages conversation persistence in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore opens or creates the SQLite database.
func NewStore(dbPath string) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		logging.Error("context.store_dir_error", map[string]any{
			"db_path": dbPath,
			"error":   err.Error(),
		})
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		logging.Error("context.store_open_error", map[string]any{
			"db_path": dbPath,
			"error":   err.Error(),
		})
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		db.Close()
		logging.Error("context.store_pragma_error", map[string]any{
			"db_path": dbPath,
			"pragma":  "busy_timeout",
			"error":   err.Error(),
		})
		return nil, fmt.Errorf("set busy_timeout: %w", err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
		if !strings.Contains(err.Error(), "SQLITE_BUSY") {
			db.Close()
			logging.Error("context.store_pragma_error", map[string]any{
				"db_path": dbPath,
				"pragma":  "journal_mode",
				"error":   err.Error(),
			})
			return nil, fmt.Errorf("set journal_mode: %w", err)
		}
	}

	if err := migrate(db); err != nil {
		db.Close()
		logging.Error("context.store_migrate_error", map[string]any{
			"db_path": dbPath,
			"error":   err.Error(),
		})
		return nil, fmt.Errorf("migrate db: %w", err)
	}

	logging.Debug("context.store_ready", map[string]any{
		"db_path": dbPath,
	})
	return &Store{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS conversations (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL DEFAULT '',
			system_prompt TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			conversation_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			provider TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL DEFAULT '',
			tokens_in INTEGER NOT NULL DEFAULT 0,
			tokens_out INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_messages_conv ON messages(conversation_id);
	`)
	return err
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying database for transaction use.
func (s *Store) DB() *sql.DB {
	return s.db
}

// CreateConversation creates a new conversation and returns its ID.
func (s *Store) CreateConversation(id, title, systemPrompt string) error {
	_, err := s.db.Exec(
		`INSERT INTO conversations (id, title, system_prompt) VALUES (?, ?, ?)`,
		id, title, systemPrompt,
	)
	return err
}

// AddMessage adds a message to a conversation.
func (s *Store) AddMessage(convID string, msg models.Message, provider, model string, tokensIn, tokensOut int) error {
	_, err := s.db.Exec(
		`INSERT INTO messages (conversation_id, role, content, provider, model, tokens_in, tokens_out)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		convID, msg.Role, msg.Content, provider, model, tokensIn, tokensOut,
	)
	if err != nil {
		return err
	}

	// Update conversation timestamp and title
	_, err = s.db.Exec(
		`UPDATE conversations SET updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		convID,
	)
	return err
}

// UpdateTitle updates the conversation title.
func (s *Store) UpdateTitle(convID, title string) error {
	_, err := s.db.Exec(`UPDATE conversations SET title = ? WHERE id = ?`, title, convID)
	return err
}

// GetMessages returns all messages for a conversation.
func (s *Store) GetMessages(convID string) ([]models.Message, error) {
	rows, err := s.db.Query(
		`SELECT role, content FROM messages WHERE conversation_id = ? ORDER BY id ASC`,
		convID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []models.Message
	for rows.Next() {
		var m models.Message
		if err := rows.Scan(&m.Role, &m.Content); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// GetConversation returns a conversation with its messages.
func (s *Store) GetConversation(convID string) (*models.Conversation, error) {
	var conv models.Conversation
	err := s.db.QueryRow(
		`SELECT id, title, system_prompt, created_at, updated_at FROM conversations WHERE id = ?`,
		convID,
	).Scan(&conv.ID, &conv.Title, &conv.SystemPrompt, &conv.CreatedAt, &conv.UpdatedAt)
	if err != nil {
		return nil, err
	}

	conv.Messages, err = s.GetMessages(convID)
	if err != nil {
		return nil, err
	}

	return &conv, nil
}

// LastConversationID returns the most recent conversation ID.
func (s *Store) LastConversationID() (string, error) {
	var id string
	err := s.db.QueryRow(
		`SELECT c.id
		 FROM conversations c
		 ORDER BY
		   COALESCE((SELECT MAX(id) FROM messages WHERE conversation_id = c.id), 0) DESC,
		   c.updated_at DESC,
		   c.created_at DESC,
		   c.id DESC
		 LIMIT 1`,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("no conversations found")
	}
	return id, err
}

// ListConversations returns recent conversations.
func (s *Store) ListConversations(limit int) ([]ConversationSummary, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.Query(`
		SELECT c.id, c.title, c.updated_at,
			   (SELECT COUNT(*) FROM messages WHERE conversation_id = c.id) as msg_count,
			   (SELECT content FROM messages WHERE conversation_id = c.id AND role = 'user' ORDER BY id ASC LIMIT 1) as first_msg
		FROM conversations c
		WHERE (SELECT COUNT(*) FROM messages WHERE conversation_id = c.id) > 0
		ORDER BY
			(SELECT MAX(id) FROM messages WHERE conversation_id = c.id) DESC,
			c.updated_at DESC,
			c.created_at DESC,
			c.id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convs []ConversationSummary
	for rows.Next() {
		var cs ConversationSummary
		var firstMsg sql.NullString
		if err := rows.Scan(&cs.ID, &cs.Title, &cs.UpdatedAt, &cs.MessageCount, &firstMsg); err != nil {
			return nil, err
		}
		if firstMsg.Valid {
			cs.FirstMessage = firstMsg.String
		}
		convs = append(convs, cs)
	}
	return convs, rows.Err()
}

// ResolveConversationID resolves an exact or unique prefix match.
func (s *Store) ResolveConversationID(prefix string) (string, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return "", fmt.Errorf("conversation ID is required")
	}

	rows, err := s.db.Query(
		`SELECT id FROM conversations WHERE id LIKE ? ORDER BY id ASC LIMIT 2`,
		prefix+"%",
	)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var matches []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "", err
		}
		if id == prefix {
			return id, nil
		}
		matches = append(matches, id)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("conversation not found: %s", prefix)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("conversation prefix is ambiguous: %s", prefix)
	}
}

// DeleteConversation removes a conversation and its messages atomically.
func (s *Store) DeleteConversation(convID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM messages WHERE conversation_id = ?`, convID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM conversations WHERE id = ?`, convID); err != nil {
		return err
	}
	return tx.Commit()
}

// ClearAll removes all conversations and messages atomically.
func (s *Store) ClearAll() (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var count int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM conversations`).Scan(&count); err != nil {
		return 0, err
	}

	if _, err := tx.Exec(`DELETE FROM messages`); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(`DELETE FROM conversations`); err != nil {
		return 0, err
	}
	return count, tx.Commit()
}

// ConversationSummary is a brief view of a conversation.
type ConversationSummary struct {
	ID           string
	Title        string
	UpdatedAt    time.Time
	MessageCount int
	FirstMessage string
}
