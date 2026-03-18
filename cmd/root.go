package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "freeapi [message]",
	Short: "Free LLM API aggregator CLI",
	Long: `freeapi - A CLI tool that aggregates free LLM APIs with automatic rotation.

When one provider hits rate limits, freeapi automatically switches to the next.
Supports Gemini, Groq, Cerebras, Mistral, OpenRouter, Cohere, GitHub Models, and Cloudflare.

Conversation context is maintained via SQLite for seamless multi-turn chats.

Quick usage:
  freeapi "Explain Go interfaces"     # direct chat
  freeapi chat -c "Tell me more"      # continue conversation
  freeapi                              # interactive REPL mode`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			// Direct chat mode: freeapi "hello"
			return chatCmd.RunE(cmd, args)
		}
		// No args + no subcommand = interactive REPL
		return runInteractive()
	},
}

func Execute() error {
	// Check if first arg looks like a message (not a subcommand)
	if len(os.Args) > 1 {
		first := os.Args[1]
		if !strings.HasPrefix(first, "-") && !isSubcommand(first) {
			// Treat as direct chat
			rootCmd.SetArgs(os.Args[1:])
		}
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}

func isSubcommand(s string) bool {
	subcommands := []string{"chat", "config", "providers", "history", "version", "help", "completion", "setup", "models", "export", "scan"}
	for _, sc := range subcommands {
		if s == sc {
			return true
		}
	}
	return false
}

// shortID safely truncates an ID to 8 chars for display.
func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}
