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
	configureLocalizedHelp(os.Args[1:])

	// Check if first arg looks like a message (not a subcommand)
	if len(os.Args) > 1 {
		first := os.Args[1]
		if !strings.HasPrefix(first, "-") && !isSubcommand(first) {
			if suggestion, ok := likelySubcommandTypo(first); ok {
				err := fmt.Errorf("unknown command %q. Did you mean %q? If you meant to send it as a prompt, use: freeapi chat %q", first, suggestion, first)
				fmt.Fprintln(os.Stderr, err)
				return err
			}
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
	for _, sc := range knownSubcommands() {
		if s == sc {
			return true
		}
	}
	return false
}

func knownSubcommands() []string {
	return []string{"chat", "config", "providers", "history", "version", "help", "completion", "setup", "models", "export", "scan", "server"}
}

func likelySubcommandTypo(input string) (string, bool) {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return "", false
	}

	best := ""
	bestDistance := 99
	for _, sc := range knownSubcommands() {
		dist := levenshteinDistance(input, sc)
		if dist < bestDistance {
			bestDistance = dist
			best = sc
		}
		if len(input) >= 4 && strings.HasPrefix(sc, input) && len(sc)-len(input) <= 2 {
			return sc, true
		}
	}

	return best, bestDistance <= 1
}

func levenshteinDistance(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(a); i++ {
		curr := make([]int, len(b)+1)
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			curr[j] = minInt(
				curr[j-1]+1,
				prev[j]+1,
				prev[j-1]+cost,
			)
		}
		prev = curr
	}

	return prev[len(b)]
}

func minInt(values ...int) int {
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

// shortID safely truncates an ID to 8 chars for display.
func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}
