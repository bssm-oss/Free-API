package provider

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bssm-oss/Free-API/internal/models"
)

var cliProviderMaxRuntime = 20 * time.Second

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
				return []string{"--yolo", prompt}
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
				return []string{"exec", "--full-auto", prompt}
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
	"gemini-cli":   10,
	"claude-cli":   20,
	"codex-cli":    30,
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
	if p.rateLimit.IsLimited && time.Now().Before(p.rateLimit.ResetAt) {
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
}

func (p *CLIProvider) Chat(ctx context.Context, messages []models.Message, opts models.ChatOptions) (*models.Response, error) {
	// Build prompt from messages - use last user message
	prompt := extractPrompt(messages)

	execCtx := ctx
	cancel := func() {}
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > cliProviderMaxRuntime {
		execCtx, cancel = context.WithTimeout(ctx, cliProviderMaxRuntime)
	}
	defer cancel()

	args := p.args(prompt)
	cmd := exec.CommandContext(execCtx, p.binPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Check if it's a rate limit or quota error
		errOutput := stderr.String() + stdout.String()
		if execCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("%s: timed out after %s", p.name, cliProviderMaxRuntime)
		}
		if isQuotaError(errOutput) {
			resetAt := time.Now().Add(60 * time.Second)
			p.MarkRateLimited(models.RateLimitInfo{
				IsLimited: true,
				ResetAt:   resetAt,
			})
			return nil, &RateLimitError{Provider: p.name, RetryAfter: resetAt}
		}
		return nil, fmt.Errorf("%s: %v\n%s", p.name, err, strings.TrimSpace(errOutput))
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		output = strings.TrimSpace(stderr.String())
	}
	if output == "" {
		return nil, fmt.Errorf("%s: empty response", p.name)
	}

	return &models.Response{
		Content:  output,
		Provider: p.name,
		Model:    p.name,
	}, nil
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
