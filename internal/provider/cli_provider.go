package provider

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bssm-oss/Free-API/internal/logging"
	"github.com/bssm-oss/Free-API/internal/models"
)

var cliProviderMaxRuntime = 10 * time.Second
var cliProviderTimeoutByName = map[string]time.Duration{
	"codex-cli":  20 * time.Second,
	"gemini-cli": 30 * time.Second,
}
var cliProviderCooldown = 10 * time.Minute

// CLIProvider wraps an external AI CLI tool as a provider.
type CLIProvider struct {
	name    string
	binPath string
	args    func(prompt string) []string // builds CLI args from prompt

	mu        sync.Mutex
	rateLimit models.RateLimitInfo
}

// CLIProviderConfig defines how to invoke a CLI tool.
type CLIProviderConfig struct {
	Name    string
	BinPath string
	Args    func(prompt string) []string
}

// KnownCLIs returns configs for all known AI CLI tools.
func KnownCLIs() []CLIProviderConfig {
	return []CLIProviderConfig{
		{
			Name: "gemini-cli",
			Args: func(prompt string) []string {
				return []string{"--yolo", "--prompt", prompt, "--output-format", "text"}
			},
		},
		{
			Name: "claude-cli",
			Args: func(prompt string) []string {
				return []string{"--dangerously-skip-permissions", "--print", prompt}
			},
		},
		{
			Name: "codex-cli",
			Args: func(prompt string) []string {
				return []string{"exec", "--skip-git-repo-check", "--ephemeral", "--full-auto", prompt}
			},
		},
		{
			Name: "copilot-cli",
			Args: func(prompt string) []string {
				return []string{"-p", prompt, "--allow-all-tools"}
			},
		},
		{
			Name: "opencode-cli",
			Args: func(prompt string) []string {
				return []string{"run", prompt}
			},
		},
	}
}

// BinNames maps CLI provider names to their binary names.
var BinNames = map[string]string{
	"gemini-cli":   "gemini",
	"claude-cli":   "claude",
	"codex-cli":    "codex",
	"copilot-cli":  "copilot",
	"opencode-cli": "opencode",
}

var defaultCLIPriorities = map[string]int{
	"codex-cli":    10,
	"gemini-cli":   20,
	"claude-cli":   30,
	"copilot-cli":  40,
	"opencode-cli": 50,
}

func IsKnownCLI(name string) bool {
	_, ok := BinNames[name]
	return ok
}

func DefaultCLIPriority(name string) int {
	if priority, ok := defaultCLIPriorities[name]; ok {
		return priority
	}
	return 100
}

// DetectCLIs finds installed AI CLIs and returns providers for them.
func DetectCLIs() []Provider {
	var providers []Provider

	for _, cfg := range KnownCLIs() {
		binName := BinNames[cfg.Name]
		binPath, err := exec.LookPath(binName)
		if err != nil {
			continue // not installed
		}

		providers = append(providers, &CLIProvider{
			name:    cfg.Name,
			binPath: binPath,
			args:    cfg.Args,
		})
	}

	return providers
}

func NewCLIProvider(name, binPath string, args func(string) []string) *CLIProvider {
	return &CLIProvider{
		name:    name,
		binPath: binPath,
		args:    args,
	}
}

func (p *CLIProvider) Name() string         { return p.name }
func (p *CLIProvider) DefaultModel() string { return filepath.Base(p.binPath) }

func (p *CLIProvider) IsAvailable() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	if p.rateLimit.IsLimited && now.Before(p.rateLimit.ResetAt) {
		return false
	}

	if cooldownUntil := loadProviderCooldown(p.name); !cooldownUntil.IsZero() {
		if now.Before(cooldownUntil) {
			p.rateLimit = models.RateLimitInfo{
				IsLimited: true,
				ResetAt:   cooldownUntil,
			}
			return false
		}
		clearProviderCooldown(p.name)
	}

	if p.rateLimit.IsLimited && !p.rateLimit.ResetAt.IsZero() && !now.Before(p.rateLimit.ResetAt) {
		p.rateLimit = models.RateLimitInfo{}
	}
	if p.rateLimit.IsLimited && p.rateLimit.ResetAt.IsZero() {
		return false
	}
	p.rateLimit.IsLimited = false
	return true
}

func (p *CLIProvider) RateLimitStatus() models.RateLimitInfo {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.rateLimit
}

func (p *CLIProvider) MarkRateLimited(info models.RateLimitInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rateLimit = info
	if info.IsLimited && !info.ResetAt.IsZero() {
		persistProviderCooldown(p.name, info.ResetAt)
		return
	}
	clearProviderCooldown(p.name)
}

func (p *CLIProvider) Chat(ctx context.Context, messages []models.Message, opts models.ChatOptions) (*models.Response, error) {
	prompt := extractPrompt(messages)

	execCtx := ctx
	cancel := func() {}
	runtimeLimit := cliRuntimeLimit(p.name)
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > runtimeLimit {
		execCtx, cancel = context.WithTimeout(ctx, runtimeLimit)
	}
	defer cancel()

	args := p.args(prompt)
	cmd := exec.CommandContext(execCtx, p.binPath, args...)
	workDir := cliWorkspaceDir(p.name)
	if dir := cliWorkspaceDir(p.name); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err == nil {
			cmd.Dir = dir
		}
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	logging.Debug("provider.cli_exec_start", map[string]any{
		"provider":      p.name,
		"bin_path":      p.binPath,
		"arg_count":     len(args),
		"message_count": len(messages),
		"prompt_len":    len(prompt),
		"runtime_ms":    runtimeLimit.Milliseconds(),
		"work_dir":      workDir,
		"stream":        opts.Stream,
		"model":         opts.Model,
	})

	err := cmd.Run()
	if err != nil {
		errOutput := stderr.String() + stdout.String()
		if execCtx.Err() == context.DeadlineExceeded {
			resetAt := time.Now().Add(cliProviderCooldown)
			p.MarkRateLimited(models.RateLimitInfo{
				IsLimited: true,
				ResetAt:   resetAt,
			})
			logging.Error("provider.cli_timeout", map[string]any{
				"provider":       p.name,
				"elapsed_ms":     time.Since(start).Milliseconds(),
				"runtime_ms":     runtimeLimit.Milliseconds(),
				"cooldown_until": resetAt.Format(time.RFC3339Nano),
			})
			return nil, fmt.Errorf("%s: timed out after %s", p.name, runtimeLimit)
		}
		if isQuotaError(errOutput) {
			resetAt := time.Now().Add(60 * time.Second)
			p.MarkRateLimited(models.RateLimitInfo{
				IsLimited: true,
				ResetAt:   resetAt,
			})
			logging.Info("provider.cli_rate_limited", map[string]any{
				"provider":       p.name,
				"elapsed_ms":     time.Since(start).Milliseconds(),
				"cooldown_until": resetAt.Format(time.RFC3339Nano),
				"output_len":     len(strings.TrimSpace(errOutput)),
			})
			return nil, &RateLimitError{Provider: p.name, RetryAfter: resetAt}
		}
		logging.Error("provider.cli_exec_error", map[string]any{
			"provider":   p.name,
			"elapsed_ms": time.Since(start).Milliseconds(),
			"output_len": len(strings.TrimSpace(errOutput)),
			"error":      err.Error(),
		})
		return nil, fmt.Errorf("%s: %v\n%s", p.name, err, strings.TrimSpace(errOutput))
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		output = strings.TrimSpace(stderr.String())
	}
	if output == "" {
		logging.Error("provider.cli_empty_response", map[string]any{
			"provider":   p.name,
			"elapsed_ms": time.Since(start).Milliseconds(),
		})
		return nil, fmt.Errorf("%s: empty response", p.name)
	}

	p.MarkRateLimited(models.RateLimitInfo{})
	logging.Info("provider.cli_exec_success", map[string]any{
		"provider":     p.name,
		"elapsed_ms":   time.Since(start).Milliseconds(),
		"response_len": len(output),
	})

	return &models.Response{
		Content:  output,
		Provider: p.name,
		Model:    p.name,
	}, nil
}

func cliRuntimeLimit(name string) time.Duration {
	if limit, ok := cliProviderTimeoutByName[name]; ok {
		return limit
	}
	return cliProviderMaxRuntime
}

func cliWorkspaceDir(name string) string {
	switch name {
	case "claude-cli", "codex-cli", "copilot-cli", "opencode-cli":
		return filepath.Join(os.TempDir(), "freeapi-cli-workspaces", name)
	default:
		return ""
	}
}

func (p *CLIProvider) ChatStream(ctx context.Context, messages []models.Message, opts models.ChatOptions) (<-chan models.StreamChunk, error) {
	// CLI providers don't support streaming, return full response as single chunk
	resp, err := p.Chat(ctx, messages, opts)
	if err != nil {
		return nil, err
	}

	ch := make(chan models.StreamChunk, 2)
	go func() {
		defer close(ch)
		ch <- models.StreamChunk{Content: resp.Content}
		ch <- models.StreamChunk{Done: true}
	}()
	return ch, nil
}

func extractPrompt(messages []models.Message) string {
	var prompt strings.Builder
	var sawConversation bool

	for _, m := range messages {
		content := strings.TrimSpace(m.Content)
		if content == "" {
			continue
		}

		switch m.Role {
		case "system":
			if prompt.Len() > 0 {
				prompt.WriteString("\n\n")
			}
			prompt.WriteString("System instructions:\n")
			prompt.WriteString(content)
		case "user":
			if !sawConversation {
				if prompt.Len() > 0 {
					prompt.WriteString("\n\n")
				}
				prompt.WriteString("Conversation:\n")
				sawConversation = true
			}
			prompt.WriteString("User:\n")
			prompt.WriteString(content)
			prompt.WriteString("\n\n")
		case "assistant":
			if !sawConversation {
				if prompt.Len() > 0 {
					prompt.WriteString("\n\n")
				}
				prompt.WriteString("Conversation:\n")
				sawConversation = true
			}
			prompt.WriteString("Assistant:\n")
			prompt.WriteString(content)
			prompt.WriteString("\n\n")
		}
	}

	if sawConversation {
		prompt.WriteString("Assistant:")
	}

	return strings.TrimSpace(prompt.String())
}

func isQuotaError(output string) bool {
	lower := strings.ToLower(output)
	quotaKeywords := []string{
		"rate limit", "ratelimit", "quota exceeded", "too many requests",
		"429", "resource exhausted", "capacity", "throttl",
	}
	for _, kw := range quotaKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}
