package cmd

import (
	"fmt"
	"strings"

	appctx "github.com/bssm-oss/Free-API/internal/context"
	"github.com/bssm-oss/Free-API/internal/config"
	"github.com/spf13/cobra"
)

var historyLimit int

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Manage conversation history",
}

var historyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent conversations",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		dbPath := cfg.DBPath
		if dbPath == "" {
			dbPath = config.DefaultDBPath()
		}

		store, err := appctx.NewStore(dbPath)
		if err != nil {
			return err
		}
		defer store.Close()

		convs, err := store.ListConversations(historyLimit)
		if err != nil {
			return err
		}

		if len(convs) == 0 {
			fmt.Println("No conversations yet.")
			return nil
		}

		fmt.Println("Recent conversations:")
		fmt.Println()
		for _, c := range convs {
			title := c.Title
			if title == "" {
				title = truncate(c.FirstMessage, 50)
			}
			if title == "" {
				title = "(empty)"
			}
			fmt.Printf("  %s  %s  [%d msgs]  %s\n",
				shortID(c.ID), c.UpdatedAt.Format("2006-01-02 15:04"), c.MessageCount, title)
		}
		return nil
	},
}

var historyShowCmd = &cobra.Command{
	Use:   "show [id]",
	Short: "Show a conversation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		dbPath := cfg.DBPath
		if dbPath == "" {
			dbPath = config.DefaultDBPath()
		}

		store, err := appctx.NewStore(dbPath)
		if err != nil {
			return err
		}
		defer store.Close()

		// Find conversation by prefix match
		convID, err := findConversation(store, args[0])
		if err != nil {
			return err
		}

		conv, err := store.GetConversation(convID)
		if err != nil {
			return fmt.Errorf("conversation not found: %s", args[0])
		}

		fmt.Printf("Conversation: %s\n", shortID(conv.ID))
		if conv.Title != "" {
			fmt.Printf("Title: %s\n", conv.Title)
		}
		fmt.Printf("Created: %s\n", conv.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println(strings.Repeat("─", 60))

		for _, m := range conv.Messages {
			role := m.Role
			switch role {
			case "user":
				role = "👤 You"
			case "assistant":
				role = "🤖 AI"
			case "system":
				role = "⚙️  System"
			}
			fmt.Printf("\n%s:\n%s\n", role, m.Content)
		}
		return nil
	},
}

var historyDeleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a conversation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		dbPath := cfg.DBPath
		if dbPath == "" {
			dbPath = config.DefaultDBPath()
		}

		store, err := appctx.NewStore(dbPath)
		if err != nil {
			return err
		}
		defer store.Close()

		convID, err := findConversation(store, args[0])
		if err != nil {
			return err
		}

		if err := store.DeleteConversation(convID); err != nil {
			return err
		}
		fmt.Printf("Deleted conversation %s\n", shortID(convID))
		return nil
	},
}

var historyClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Delete all conversations",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		dbPath := cfg.DBPath
		if dbPath == "" {
			dbPath = config.DefaultDBPath()
		}

		store, err := appctx.NewStore(dbPath)
		if err != nil {
			return err
		}
		defer store.Close()

		count, err := store.ClearAll()
		if err != nil {
			return err
		}
		fmt.Printf("Deleted %d conversations.\n", count)
		return nil
	},
}

func init() {
	historyListCmd.Flags().IntVarP(&historyLimit, "limit", "n", 20, "Number of conversations to show")
	historyCmd.AddCommand(historyListCmd)
	historyCmd.AddCommand(historyShowCmd)
	historyCmd.AddCommand(historyDeleteCmd)
	historyCmd.AddCommand(historyClearCmd)
	rootCmd.AddCommand(historyCmd)
}

// findConversation finds a conversation by ID prefix.
func findConversation(store *appctx.Store, prefix string) (string, error) {
	convs, err := store.ListConversations(100)
	if err != nil {
		return "", err
	}

	for _, c := range convs {
		if strings.HasPrefix(c.ID, prefix) {
			return c.ID, nil
		}
	}
	return "", fmt.Errorf("conversation not found: %s", prefix)
}
