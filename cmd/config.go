package cmd

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bssm-oss/Free-API/internal/config"
	"github.com/bssm-oss/Free-API/internal/models"
	"github.com/bssm-oss/Free-API/internal/provider"
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
  freeapi config set codex-cli.enabled false
  freeapi config set max_context_messages 100
  freeapi config set default_system_prompt "You are a coding assistant."`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadRaw()
		if err != nil {
			return err
		}

		key, value := args[0], args[1]
		if err := applyConfigValue(cfg, key, value); err != nil {
			return err
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

func applyConfigValue(cfg *config.Config, key, value string) error {
	parts := strings.SplitN(key, ".", 2)

	if len(parts) == 2 {
		providerName := parts[0]
		field := parts[1]

		p, ok := cfg.Providers[providerName]
		if !ok {
			if !provider.IsKnownCLI(providerName) {
				return fmt.Errorf("unknown provider: %s", providerName)
			}
			p = models.ProviderConfig{
				Enabled:  true,
				Priority: provider.DefaultCLIPriority(providerName),
			}
		}

		if provider.IsKnownCLI(providerName) {
			switch field {
			case "enabled":
				parsed, err := parseBoolValue(value)
				if err != nil {
					return fmt.Errorf("%s: %w", key, err)
				}
				p.Enabled = parsed
			case "priority":
				parsed, err := strconv.Atoi(value)
				if err != nil || parsed <= 0 {
					return fmt.Errorf("%s: must be a positive integer", key)
				}
				p.Priority = parsed
			default:
				return fmt.Errorf("unknown field for CLI provider %s: %s", providerName, field)
			}
			cfg.Providers[providerName] = p
			return nil
		}

		switch field {
		case "api_key":
			p.APIKey = value
		case "account_id":
			p.AccountID = value
		case "model":
			p.Model = value
		case "enabled":
			parsed, err := parseBoolValue(value)
			if err != nil {
				return fmt.Errorf("%s: %w", key, err)
			}
			p.Enabled = parsed
		case "priority":
			parsed, err := strconv.Atoi(value)
			if err != nil || parsed <= 0 {
				return fmt.Errorf("%s: must be a positive integer", key)
			}
			p.Priority = parsed
		case "base_url":
			p.BaseURL = value
		default:
			return fmt.Errorf("unknown field: %s", field)
		}
		cfg.Providers[providerName] = p
		return nil
	}

	switch key {
	case "default_system_prompt":
		cfg.DefaultSystemPrompt = value
	case "context_strategy":
		if value != "sliding_window" {
			return fmt.Errorf("unsupported context_strategy: %s", value)
		}
		cfg.ContextStrategy = value
	case "max_context_messages":
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			return fmt.Errorf("%s: must be a positive integer", key)
		}
		cfg.MaxContextMessages = parsed
	case "db_path":
		if value == "" {
			cfg.DBPath = ""
			return nil
		}
		cfg.DBPath = filepath.Clean(value)
	default:
		return fmt.Errorf("unknown key: %s", key)
	}

	return nil
}

func parseBoolValue(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes", "y", "on":
		return true, nil
	case "false", "0", "no", "n", "off":
		return false, nil
	default:
		return false, fmt.Errorf("must be a boolean")
	}
}
