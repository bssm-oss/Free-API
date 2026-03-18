package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	appctx "github.com/heodongun/freeapi/internal/context"
	"github.com/heodongun/freeapi/internal/config"
	"github.com/heodongun/freeapi/internal/models"
	"github.com/heodongun/freeapi/internal/provider"
	"github.com/spf13/cobra"
)

var (
	chatContinue  bool
	chatCID       string
	chatProvider  string
	chatModel     string
	chatSystemMsg string
	chatNoStream  bool
	chatRaw       bool
	chatTimeout   int
)

var chatCmd = &cobra.Command{
	Use:   "chat [message]",
	Short: "Send a message to an LLM",
	Long: `Send a message and get a response. Automatically selects the best available free provider.

Examples:
  freeapi chat "Explain Go interfaces"
  freeapi chat -c "Tell me more"
  freeapi chat --provider gemini "Hello"
  freeapi chat --raw "Just output" > output.txt
  echo "analyze this" | freeapi chat`,
	RunE: runChat,
}

func init() {
	chatCmd.Flags().BoolVarP(&chatContinue, "continue", "c", false, "Continue last conversation")
	chatCmd.Flags().StringVar(&chatCID, "cid", "", "Continue a specific conversation by ID")
	chatCmd.Flags().StringVarP(&chatProvider, "provider", "p", "", "Use a specific provider")
	chatCmd.Flags().StringVarP(&chatModel, "model", "m", "", "Use a specific model")
	chatCmd.Flags().StringVarP(&chatSystemMsg, "system", "s", "", "Set system message")
	chatCmd.Flags().BoolVar(&chatNoStream, "no-stream", false, "Disable streaming output")
	chatCmd.Flags().BoolVar(&chatRaw, "raw", false, "Raw output only (no metadata to stderr)")
	chatCmd.Flags().IntVar(&chatTimeout, "timeout", 120, "Request timeout in seconds")
	rootCmd.AddCommand(chatCmd)
}

func runChat(cmd *cobra.Command, args []string) error {
	// Collect input
	var input string
	if len(args) > 0 {
		input = strings.Join(args, " ")
	}

	stat, err := os.Stdin.Stat()
	if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		reader := bufio.NewReader(os.Stdin)
		piped, err := io.ReadAll(reader)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		pipeStr := strings.TrimSpace(string(piped))
		if pipeStr != "" {
			if input != "" {
				input = pipeStr + "\n\n" + input
			} else {
				input = pipeStr
			}
		}
	}

	if input == "" {
		return fmt.Errorf("no message provided. Usage: freeapi chat \"your message\"")
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Init SQLite store
	dbPath := cfg.DBPath
	if dbPath == "" {
		dbPath = config.DefaultDBPath()
	}
	store, err := appctx.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer store.Close()

	// Context manager
	sysPrompt := cfg.DefaultSystemPrompt
	if chatSystemMsg != "" {
		sysPrompt = chatSystemMsg
	}
	mgr := appctx.NewManager(store, cfg.MaxContextMessages, cfg.ContextStrategy, sysPrompt)

	// Get or create conversation
	convID, isContinue, err := mgr.GetOrContinue(chatCID, chatContinue, sysPrompt)
	if err != nil {
		return err
	}

	if isContinue && !chatRaw {
		fmt.Fprintf(os.Stderr, "📎 Continuing conversation %s\n", shortID(convID))
	}

	// Build messages with history
	messages, err := mgr.BuildMessages(convID, input)
	if err != nil {
		return err
	}

	// Init provider rotator
	registry := provider.NewRegistry(cfg)
	rotator := provider.NewRotator(registry)

	opts := models.ChatOptions{
		Model:  chatModel,
		Stream: !chatNoStream,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(chatTimeout)*time.Second)
	defer cancel()

	// Streaming or non-streaming
	if chatNoStream {
		return runNonStream(ctx, rotator, messages, opts, mgr, convID, input)
	}
	return runStream(ctx, rotator, messages, opts, mgr, convID, input)
}

func runStream(ctx context.Context, rotator *provider.Rotator, messages []models.Message, opts models.ChatOptions, mgr *appctx.Manager, convID, input string) error {
	opts.Stream = true
	start := time.Now()

	var ch <-chan models.StreamChunk
	var providerName string
	var err error

	if chatProvider != "" {
		ch, err = rotator.ChatStreamWithProvider(ctx, chatProvider, messages, opts)
		providerName = chatProvider
	} else {
		ch, providerName, err = rotator.ChatStream(ctx, messages, opts)
	}
	if err != nil {
		return err
	}

	var fullContent strings.Builder
	for chunk := range ch {
		if chunk.Error != nil {
			if fullContent.Len() > 0 {
				fmt.Fprintf(os.Stderr, "\n⚠️  Stream interrupted: %v\n", chunk.Error)
				break
			}
			return chunk.Error
		}
		if chunk.Done {
			break
		}
		fmt.Print(chunk.Content)
		fullContent.WriteString(chunk.Content)
	}
	fmt.Println()

	elapsed := time.Since(start)

	// Save to history
	if fullContent.Len() > 0 {
		if err := mgr.SaveExchange(convID, input, fullContent.String(), providerName, "", 0, 0); err != nil && !chatRaw {
			fmt.Fprintf(os.Stderr, "⚠️  Failed to save conversation: %v\n", err)
		}
	}

	if !chatRaw {
		fmt.Fprintf(os.Stderr, "💬 [%s] %.1fs conv=%s\n", providerName, elapsed.Seconds(), shortID(convID))
	}
	return nil
}

func runNonStream(ctx context.Context, rotator *provider.Rotator, messages []models.Message, opts models.ChatOptions, mgr *appctx.Manager, convID, input string) error {
	opts.Stream = false
	start := time.Now()

	var resp *models.Response
	var err error

	if chatProvider != "" {
		resp, err = rotator.ChatWithProvider(ctx, chatProvider, messages, opts)
	} else {
		resp, err = rotator.Chat(ctx, messages, opts)
	}
	if err != nil {
		return err
	}

	elapsed := time.Since(start)

	fmt.Print(resp.Content)
	if !strings.HasSuffix(resp.Content, "\n") {
		fmt.Println()
	}

	// Save to history
	if err := mgr.SaveExchange(convID, input, resp.Content, resp.Provider, resp.Model, resp.TokensIn, resp.TokensOut); err != nil && !chatRaw {
		fmt.Fprintf(os.Stderr, "⚠️  Failed to save conversation: %v\n", err)
	}

	if !chatRaw {
		fmt.Fprintf(os.Stderr, "💬 [%s/%s] %.1fs tokens=%d/%d conv=%s\n",
			resp.Provider, resp.Model, elapsed.Seconds(), resp.TokensIn, resp.TokensOut, shortID(convID))
	}
	return nil
}
