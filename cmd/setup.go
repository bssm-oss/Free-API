package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/bssm-oss/Free-API/internal/config"
	"github.com/bssm-oss/Free-API/internal/provider"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Auto-detect CLIs and configure API keys",
	RunE:  runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	fmt.Println("🔧 freeapi setup")
	fmt.Println()

	// 1. Auto-detect installed AI CLIs
	fmt.Println("━━━ Installed AI CLIs ━━━")
	clis := provider.DetectCLIs()
	if len(clis) > 0 {
		for _, c := range clis {
			fmt.Printf("  ✅ %s → %s\n", c.Name(), c.DefaultModel())
		}
		fmt.Printf("\n  %d CLI(s) detected — these work without API keys!\n", len(clis))
	} else {
		fmt.Println("  No AI CLIs found.")
		fmt.Println()
		fmt.Println("  Install one with:")
		fmt.Println("    npm i -g @anthropic-ai/claude-code   # Claude")
		fmt.Println("    npm i -g @anthropic-ai/claude-code   # Gemini (if not installed)")
		fmt.Println("    brew install codex                    # Codex")
	}
	fmt.Println()

	// 2. Show not-installed CLIs that could be added
	fmt.Println("━━━ Available to Install ━━━")
	installable := []struct {
		name    string
		bin     string
		install string
	}{
		{"Gemini CLI", "gemini", "npm i -g @anthropic-ai/claude-code  # or brew install gemini"},
		{"Claude CLI", "claude", "npm i -g @anthropic-ai/claude-code"},
		{"Codex CLI", "codex", "brew install codex"},
		{"Copilot CLI", "copilot", "gh extension install github/gh-copilot"},
		{"OpenCode", "opencode", "go install github.com/opencode-ai/opencode@latest"},
	}
	hasUninstalled := false
	for _, item := range installable {
		if _, err := exec.LookPath(item.bin); err != nil {
			fmt.Printf("  ❌ %s — %s\n", item.name, item.install)
			hasUninstalled = true
		}
	}
	if !hasUninstalled {
		fmt.Println("  All known CLIs already installed!")
	}
	fmt.Println()

	// 3. Check API-based providers
	fmt.Println("━━━ API Providers (optional) ━━━")
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	apiProviders := []struct {
		name    string
		display string
		url     string
	}{
		{"gemini", "Google Gemini API", "https://ai.google.dev"},
		{"groq", "Groq", "https://console.groq.com/keys"},
		{"cerebras", "Cerebras", "https://cloud.cerebras.ai/"},
		{"openrouter", "OpenRouter", "https://openrouter.ai/keys"},
		{"github", "GitHub Models", "GITHUB_TOKEN from gh auth token"},
	}

	hasKey := 0
	for _, p := range apiProviders {
		pcfg := cfg.Providers[p.name]
		if pcfg.APIKey != "" {
			fmt.Printf("  ✅ %s — configured\n", p.display)
			hasKey++
		} else {
			fmt.Printf("  ○  %s — %s\n", p.display, p.url)
		}
	}

	if hasKey == 0 && len(clis) == 0 {
		fmt.Println("\n  ⚠️  No providers available! Set up at least one API key:")
	}
	fmt.Println()

	// 4. Ask if they want to set API keys
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Set up API keys now? (y/N): ")
	if scanner.Scan() && strings.ToLower(strings.TrimSpace(scanner.Text())) == "y" {
		fmt.Println()
		changed := false
		for _, p := range apiProviders {
			pcfg := cfg.Providers[p.name]
			if pcfg.APIKey != "" {
				continue // already set
			}
			fmt.Printf("  %s key (Enter to skip): ", p.display)
			if !scanner.Scan() {
				break
			}
			key := strings.TrimSpace(scanner.Text())
			if key != "" {
				pcfg.APIKey = key
				cfg.Providers[p.name] = pcfg
				changed = true
				fmt.Println("  ✅ Set!")
			}
		}
		if changed {
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Println("\n✅ Saved!")
		}
	}

	// 5. Summary
	registry := provider.NewRegistry(cfg)
	total := registry.Count()
	avail := len(registry.Available())
	fmt.Printf("\n🎉 Ready! %d/%d providers available. Try: freeapi \"hello\"\n", avail, total)

	return nil
}
