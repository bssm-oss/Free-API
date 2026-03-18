package cmd

import (
	"fmt"
	"strings"

	"github.com/heodongun/freeapi/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage freeapi configuration",
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		// Mask API keys for display
		for name, p := range cfg.Providers {
			if p.APIKey != "" {
				if len(p.APIKey) > 8 {
					p.APIKey = p.APIKey[:4] + "..." + p.APIKey[len(p.APIKey)-4:]
				} else {
					p.APIKey = "***"
				}
				cfg.Providers[name] = p
			}
		}

		data, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Long: `Set a configuration value. Use dot notation for nested keys.

Examples:
  freeapi config set gemini.api_key "AIza..."
  freeapi config set groq.api_key "gsk_..."
  freeapi config set gemini.enabled false
  freeapi config set default_system_prompt "You are a coding assistant."`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		key, value := args[0], args[1]
		parts := strings.SplitN(key, ".", 2)

		if len(parts) == 2 {
			providerName := parts[0]
			field := parts[1]

			p, ok := cfg.Providers[providerName]
			if !ok {
				return fmt.Errorf("unknown provider: %s", providerName)
			}

			switch field {
			case "api_key":
				p.APIKey = value
			case "model":
				p.Model = value
			case "enabled":
				p.Enabled = value == "true"
			default:
				return fmt.Errorf("unknown field: %s", field)
			}
			cfg.Providers[providerName] = p
		} else {
			switch key {
			case "default_system_prompt":
				cfg.DefaultSystemPrompt = value
			case "context_strategy":
				cfg.ContextStrategy = value
			default:
				return fmt.Errorf("unknown key: %s", key)
			}
		}

		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Printf("Set %s = %s\n", key, value)
		return nil
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create default config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.DefaultConfig()
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Printf("Config created at %s\n", config.ConfigPath())
		return nil
	},
}

func init() {
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configInitCmd)
	rootCmd.AddCommand(configCmd)
}
