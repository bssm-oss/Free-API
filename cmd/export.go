package cmd

import (
	"fmt"
	"strings"

	appctx "github.com/bssm-oss/Free-API/internal/context"
	"github.com/bssm-oss/Free-API/internal/config"
	"github.com/bssm-oss/Free-API/internal/models"
	"github.com/spf13/cobra"
)

var exportFormat string

var exportCmd = &cobra.Command{
	Use:   "export [conversation-id]",
	Short: "Export a conversation to markdown or text",
	Long: `Export a conversation to stdout. Use --format to choose output format.

Examples:
  freeapi export abc123                    # export as markdown
  freeapi export abc123 --format text      # export as plain text
  freeapi export abc123 > conversation.md  # save to file`,
	Args: cobra.ExactArgs(1),
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

		// Find conversation
		convID, err := findConvByPrefix(store, args[0])
		if err != nil {
			return err
		}

		conv, err := store.GetConversation(convID)
		if err != nil {
			return fmt.Errorf("conversation not found: %s", args[0])
		}

		switch exportFormat {
		case "text":
			return exportText(conv)
		default:
			return exportMarkdown(conv)
		}
	},
}

func init() {
	exportCmd.Flags().StringVar(&exportFormat, "format", "markdown", "Output format: markdown, text")
	rootCmd.AddCommand(exportCmd)
}

func exportMarkdown(conv *models.Conversation) error {
	title := conv.Title
	if title == "" {
		title = "Conversation"
	}

	fmt.Printf("# %s\n\n", title)
	fmt.Printf("**ID**: `%s`  \n", shortID(conv.ID))
	fmt.Printf("**Created**: %s  \n\n", conv.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Println("---")
	fmt.Println()

	for _, m := range conv.Messages {
		switch m.Role {
		case "system":
			fmt.Printf("*System: %s*\n\n", m.Content)
		case "user":
			fmt.Printf("## 👤 You\n\n%s\n\n", m.Content)
		case "assistant":
			fmt.Printf("## 🤖 Assistant\n\n%s\n\n", m.Content)
		}
	}
	return nil
}

func exportText(conv *models.Conversation) error {
	for _, m := range conv.Messages {
		switch m.Role {
		case "system":
			// skip
		case "user":
			fmt.Printf("You: %s\n\n", m.Content)
		case "assistant":
			fmt.Printf("AI: %s\n\n", m.Content)
		}
	}
	return nil
}

func findConvByPrefix(store *appctx.Store, prefix string) (string, error) {
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
