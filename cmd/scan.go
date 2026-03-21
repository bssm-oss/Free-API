package cmd

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan for installed AI CLIs",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🔍 Scanning for AI CLIs...")
		fmt.Println()

		tools := []struct {
			name    string
			bin     string
			install string
		}{
			{"Gemini CLI", "gemini", "npm install -g @google/gemini-cli  OR  brew install gemini-cli"},
			{"Claude Code", "claude", "npm i -g @anthropic-ai/claude-code"},
			{"Codex CLI", "codex", "brew install codex  OR  cargo install codex-cli"},
			{"Copilot CLI", "copilot", "install GitHub Copilot CLI"},
			{"OpenCode", "opencode", "curl -fsSL https://opencode.ai/install | bash"},
		}

		installed := 0
		for _, t := range tools {
			path, err := exec.LookPath(t.bin)
			if err == nil {
				fmt.Printf("  ✅ %-14s %s\n", t.name, path)
				installed++
			} else {
				fmt.Printf("  ❌ %-14s install: %s\n", t.name, t.install)
			}
		}

		fmt.Printf("\n  %d/%d installed\n", installed, len(tools))
		if installed > 0 {
			fmt.Println("\n  These work immediately with: freeapi \"your question\"")
		}
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
}
