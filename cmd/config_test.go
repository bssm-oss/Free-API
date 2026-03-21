package cmd

import (
	"testing"

	"github.com/bssm-oss/Free-API/internal/config"
	"github.com/bssm-oss/Free-API/internal/provider"
)

func TestApplyConfigValueSupportsCLIProviders(t *testing.T) {
	cfg := config.DefaultConfig()

	if err := applyConfigValue(cfg, "codex-cli.enabled", "false"); err != nil {
		t.Fatalf("apply cli enabled: %v", err)
	}
	if err := applyConfigValue(cfg, "codex-cli.priority", "99"); err != nil {
		t.Fatalf("apply cli priority: %v", err)
	}

	pcfg, ok := cfg.Providers["codex-cli"]
	if !ok {
		t.Fatal("expected codex-cli config entry to be created")
	}
	if pcfg.Enabled {
		t.Fatal("expected codex-cli to be disabled")
	}
	if pcfg.Priority != 99 {
		t.Fatalf("expected codex-cli priority 99, got %d", pcfg.Priority)
	}
}

func TestApplyConfigValueSupportsProductSettings(t *testing.T) {
	cfg := config.DefaultConfig()

	if err := applyConfigValue(cfg, "max_context_messages", "100"); err != nil {
		t.Fatalf("set max_context_messages: %v", err)
	}
	if err := applyConfigValue(cfg, "db_path", "/tmp/freeapi.db"); err != nil {
		t.Fatalf("set db_path: %v", err)
	}
	if err := applyConfigValue(cfg, "cloudflare.account_id", "acct-123"); err != nil {
		t.Fatalf("set cloudflare account_id: %v", err)
	}

	if cfg.MaxContextMessages != 100 {
		t.Fatalf("expected max_context_messages=100, got %d", cfg.MaxContextMessages)
	}
	if cfg.DBPath != "/tmp/freeapi.db" {
		t.Fatalf("unexpected db_path: %q", cfg.DBPath)
	}
	if cfg.Providers["cloudflare"].AccountID != "acct-123" {
		t.Fatalf("unexpected cloudflare account ID: %q", cfg.Providers["cloudflare"].AccountID)
	}
}

func TestApplyConfigValueRejectsUnsupportedCLIField(t *testing.T) {
	cfg := config.DefaultConfig()

	err := applyConfigValue(cfg, "codex-cli.api_key", "nope")
	if err == nil {
		t.Fatal("expected unsupported cli field error")
	}
}

func TestApplyConfigValueSeedsCLIDefaultPriority(t *testing.T) {
	cfg := config.DefaultConfig()

	if err := applyConfigValue(cfg, "gemini-cli.enabled", "true"); err != nil {
		t.Fatalf("apply config value: %v", err)
	}

	if got := cfg.Providers["gemini-cli"].Priority; got != provider.DefaultCLIPriority("gemini-cli") {
		t.Fatalf("expected seeded cli priority %d, got %d", provider.DefaultCLIPriority("gemini-cli"), got)
	}
}
