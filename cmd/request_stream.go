package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/bssm-oss/Free-API/internal/models"
	"github.com/bssm-oss/Free-API/internal/provider"
)

const defaultChatTimeoutSeconds = 120

type streamRequestConfig struct {
	providerOverride string
	raw              bool
	spinnerMessage   string
}

func executeStreamRequest(ctx context.Context, rotator *provider.Rotator, messages []models.Message, opts models.ChatOptions, cfg streamRequestConfig) (string, string, error) {
	opts.Stream = true
	spinner := startRequestSpinner(shouldShowSpinner(cfg.raw), cfg.spinnerMessage)
	defer spinner.Stop()

	var ch <-chan models.StreamChunk
	var providerName string
	var err error

	if cfg.providerOverride != "" {
		ch, err = rotator.ChatStreamWithProvider(ctx, cfg.providerOverride, messages, opts)
		providerName = cfg.providerOverride
	} else {
		ch, providerName, err = rotator.ChatStream(ctx, messages, opts)
	}
	if err != nil {
		return "", "", err
	}

	var fullContent strings.Builder
	spinnerStopped := false
	for chunk := range ch {
		if chunk.Error != nil {
			if !spinnerStopped {
				spinner.Stop()
				spinnerStopped = true
			}
			if fullContent.Len() > 0 {
				return fullContent.String(), providerName, fmt.Errorf("stream interrupted: %w", chunk.Error)
			}
			return "", providerName, chunk.Error
		}
		if chunk.Done {
			break
		}
		if !spinnerStopped {
			spinner.Stop()
			spinnerStopped = true
		}
		fmt.Print(chunk.Content)
		fullContent.WriteString(chunk.Content)
	}
	if !spinnerStopped {
		spinner.Stop()
	}

	return fullContent.String(), providerName, nil
}
