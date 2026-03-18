package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/heodongun/freeapi/internal/config"
	"github.com/heodongun/freeapi/internal/models"
	"github.com/heodongun/freeapi/internal/provider"
	"github.com/spf13/cobra"
)

var providersCmd = &cobra.Command{
	Use:   "providers",
	Short: "Manage LLM providers",
}

var providersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured providers",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		registry := provider.NewRegistry(cfg)
		rotator := provider.NewRotator(registry)
		statuses := rotator.Status()

		fmt.Println("Providers:")
		fmt.Println()
		for i, s := range statuses {
			status := "✅"
			hint := ""
			isCLI := strings.HasSuffix(s.Name, "-cli")

			if !s.Available {
				status = "❌"
				if envVars, ok := config.EnvVarMap[s.Name]; ok {
					hint = fmt.Sprintf("  set %s", envVars[0])
				}
			}
			if s.RateLimit.IsLimited {
				status = "⏳"
				hint = fmt.Sprintf("  resets %s", s.RateLimit.ResetAt.Format("15:04:05"))
			}

			ptype := "API"
			if isCLI {
				ptype = "CLI"
			}
			fmt.Printf("  %d. %s %-14s [%s]  %-30s%s\n", i+1, status, s.Name, ptype, s.Model, hint)
		}

		avail := registry.Available()
		fmt.Printf("\n  %d/%d providers available\n", len(avail), registry.Count())
		if len(avail) == 0 {
			fmt.Println("\n  Quick start: freeapi setup")
		}
		return nil
	},
}

var providersStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show detailed provider status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		registry := provider.NewRegistry(cfg)
		rotator := provider.NewRotator(registry)
		statuses := rotator.Status()

		for _, s := range statuses {
			fmt.Printf("%-12s  available=%v", s.Name, s.Available)
			if s.RateLimit.IsLimited {
				fmt.Printf("  rate_limited=true  resets=%s", s.RateLimit.ResetAt.Format(time.RFC3339))
			}
			fmt.Println()
		}
		return nil
	},
}

var providersTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test connectivity to all providers",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		registry := provider.NewRegistry(cfg)

		fmt.Println("Testing providers...")
		fmt.Println()

		testMsg := []models.Message{
			{Role: "user", Content: "Say 'ok' and nothing else."},
		}
		opts := models.ChatOptions{MaxTokens: 10}

		for _, p := range registry.GetByPriority() {
			fmt.Printf("  %-12s ", p.Name())
			if !p.IsAvailable() {
				fmt.Println("⏭️  skipped (no API key)")
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			resp, err := p.Chat(ctx, testMsg, opts)
			cancel()

			if err != nil {
				fmt.Printf("❌ %v\n", err)
			} else {
				fmt.Printf("✅ model=%s response=%q\n", resp.Model, truncate(resp.Content, 30))
			}
		}
		return nil
	},
}

func init() {
	providersCmd.AddCommand(providersListCmd)
	providersCmd.AddCommand(providersStatusCmd)
	providersCmd.AddCommand(providersTestCmd)
	rootCmd.AddCommand(providersCmd)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
