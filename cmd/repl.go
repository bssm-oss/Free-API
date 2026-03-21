package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bssm-oss/Free-API/internal/config"
	appctx "github.com/bssm-oss/Free-API/internal/context"
	"github.com/bssm-oss/Free-API/internal/models"
	"github.com/bssm-oss/Free-API/internal/provider"
)

func runInteractive() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	dbPath := cfg.DBPath
	if dbPath == "" {
		dbPath = config.DefaultDBPath()
	}
	store, err := appctx.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer store.Close()

	registry := provider.NewRegistry(cfg)
	rotator := provider.NewRotator(registry)

	// Show status
	avail := registry.Available()
	fmt.Printf("freeapi v%s — %d/%d providers ready\n", Version, len(avail), registry.Count())
	if len(avail) == 0 {
		fmt.Println("⚠️  No providers available. Run: freeapi setup")
		fmt.Println("   Or configure credentials with: freeapi config set <provider>.api_key <key>")
		return nil
	}
	for _, s := range rotator.Status() {
		if s.Available {
			fmt.Printf("  ✅ %s (%s)\n", s.Name, s.Model)
		}
	}

	mgr := appctx.NewManager(store, cfg.MaxContextMessages, cfg.ContextStrategy, cfg.DefaultSystemPrompt)
	var convID string // lazy-init on first message

	fmt.Printf("\n💬 freeapi REPL. Type /help for commands, Ctrl+C to exit.\n\n")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

	for {
		fmt.Print("you> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Handle REPL commands
		switch {
		case input == "/help":
			printREPLHelp()
			continue
		case input == "/new":
			convID = "" // will be created on next message
			fmt.Println("📝 New conversation started.")
			fmt.Println()
			continue
		case input == "/status":
			for _, s := range rotator.Status() {
				st := "✅"
				if !s.Available {
					st = "❌"
				}
				fmt.Printf("  %s %s (%s)\n", st, s.Name, s.Model)
			}
			fmt.Println()
			continue
		case input == "/quit" || input == "/exit":
			fmt.Println("Bye!")
			return nil
		case input == "/id":
			fmt.Printf("Conversation: %s\n\n", convID)
			continue
		case input == "/last":
			lastID, err := store.LastConversationID()
			if err != nil {
				fmt.Println("No previous conversations.")
				continue
			}
			if lastID == convID {
				fmt.Println("Already on the latest conversation.")
				continue
			}
			convID = lastID
			fmt.Printf("📎 Switched to conversation [%s]\n\n", shortID(convID))
			continue
		case input == "/history":
			convs, err := store.ListConversations(10)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				continue
			}
			for _, c := range convs {
				marker := "  "
				if c.ID == convID {
					marker = "→ "
				}
				title := c.Title
				if title == "" && c.FirstMessage != "" {
					title = c.FirstMessage
					if len(title) > 40 {
						title = title[:40] + "..."
					}
				}
				fmt.Printf("%s%s  [%d msgs]  %s\n", marker, shortID(c.ID), c.MessageCount, title)
			}
			fmt.Println()
			continue
		}

		// Lazy-init conversation on first message
		if convID == "" {
			convID, _, err = mgr.GetOrContinue("", false, cfg.DefaultSystemPrompt)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				continue
			}
		}

		// Build messages
		messages, err := mgr.BuildMessages(convID, input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		// Stream response (5 min timeout)
		opts := models.ChatOptions{Stream: true}
		fullContent, providerName := doStream(rotator, messages, opts)

		// Save
		if fullContent != "" {
			if err := mgr.SaveExchange(convID, input, fullContent, providerName, "", 0, 0); err != nil {
				fmt.Fprintf(os.Stderr, "⚠️  Save failed: %v\n", err)
			}
		}

		if providerName != "" {
			fmt.Fprintf(os.Stderr, "[%s] ", providerName)
		}
	}

	return scanner.Err()
}

func printREPLHelp() {
	fmt.Print(`
Commands:
  /new       Start a new conversation
  /last      Switch to last conversation
  /history   List recent conversations
  /status    Show provider status
  /id        Show current conversation ID
  /help      Show this help
  /quit      Exit

Just type your message to chat!
`)
}

func doStream(rotator *provider.Rotator, messages []models.Message, opts models.ChatOptions) (string, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ch, providerName, err := rotator.ChatStream(ctx, messages, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		return "", ""
	}

	fmt.Print("\n")
	var fullContent strings.Builder
	for chunk := range ch {
		if chunk.Error != nil {
			fmt.Fprintf(os.Stderr, "\nStream error: %v\n", chunk.Error)
			break
		}
		if chunk.Done {
			break
		}
		fmt.Print(chunk.Content)
		fullContent.WriteString(chunk.Content)
	}
	fmt.Printf("\n\n")

	return fullContent.String(), providerName
}
