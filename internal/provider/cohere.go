package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bssm-oss/Free-API/internal/models"
)

// CohereProvider handles Cohere's chat API.
type CohereProvider struct {
	apiKey       string
	baseURL      string
	defaultModel string

	mu        sync.Mutex
	rateLimit models.RateLimitInfo
}

type cohereRequest struct {
	Model    string          `json:"model"`
	Messages []cohereMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type cohereMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type cohereResponse struct {
	Message struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"message"`
	FinishReason string `json:"finish_reason"`
	Usage        struct {
		Tokens struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"tokens"`
	} `json:"usage"`
}

type cohereStreamEvent struct {
	Type  string `json:"type"`
	Delta struct {
		Message struct {
			Content struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"message"`
	} `json:"delta"`
}

func NewCohere(apiKey, baseURL, defaultModel string) *CohereProvider {
	if baseURL == "" {
		baseURL = "https://api.cohere.ai/v2"
	}
	return &CohereProvider{
		apiKey:       apiKey,
		baseURL:      baseURL,
		defaultModel: defaultModel,
	}
}

func (p *CohereProvider) Name() string         { return "cohere" }
func (p *CohereProvider) DefaultModel() string { return p.defaultModel }

func (p *CohereProvider) IsAvailable() bool {
	if p.apiKey == "" {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.rateLimit.IsLimited && time.Now().Before(p.rateLimit.ResetAt) {
		return false
	}
	p.rateLimit.IsLimited = false
	return true
}

func (p *CohereProvider) RateLimitStatus() models.RateLimitInfo {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.rateLimit
}

func (p *CohereProvider) MarkRateLimited(info models.RateLimitInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rateLimit = info
}

func (p *CohereProvider) Chat(ctx context.Context, messages []models.Message, opts models.ChatOptions) (*models.Response, error) {
	model := opts.Model
	if model == "" {
		model = p.defaultModel
	}

	cMsgs := make([]cohereMessage, len(messages))
	for i, m := range messages {
		cMsgs[i] = cohereMessage{Role: m.Role, Content: m.Content}
	}

	reqBody := cohereRequest{
		Model:    model,
		Messages: cMsgs,
		Stream:   false,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := SharedClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cohere: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		p.MarkRateLimited(models.RateLimitInfo{
			IsLimited: true,
			ResetAt:   time.Now().Add(60 * time.Second),
		})
		return nil, &RateLimitError{Provider: "cohere", RetryAfter: p.rateLimit.ResetAt}
	}

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cohere: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var cResp cohereResponse
	if err := json.NewDecoder(resp.Body).Decode(&cResp); err != nil {
		return nil, fmt.Errorf("cohere: decode: %w", err)
	}

	var content string
	for _, c := range cResp.Message.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}

	return &models.Response{
		Content:      content,
		Model:        model,
		Provider:     "cohere",
		TokensIn:     cResp.Usage.Tokens.InputTokens,
		TokensOut:    cResp.Usage.Tokens.OutputTokens,
		FinishReason: cResp.FinishReason,
	}, nil
}

func (p *CohereProvider) ChatStream(ctx context.Context, messages []models.Message, opts models.ChatOptions) (<-chan models.StreamChunk, error) {
	model := opts.Model
	if model == "" {
		model = p.defaultModel
	}

	cMsgs := make([]cohereMessage, len(messages))
	for i, m := range messages {
		cMsgs[i] = cohereMessage{Role: m.Role, Content: m.Content}
	}

	reqBody := cohereRequest{
		Model:    model,
		Messages: cMsgs,
		Stream:   true,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := StreamClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cohere: request failed: %w", err)
	}

	if resp.StatusCode == 429 {
		resp.Body.Close()
		p.MarkRateLimited(models.RateLimitInfo{
			IsLimited: true,
			ResetAt:   time.Now().Add(60 * time.Second),
		})
		return nil, &RateLimitError{Provider: "cohere", RetryAfter: p.rateLimit.ResetAt}
	}

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("cohere: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	ch := make(chan models.StreamChunk, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		done := false

		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			var event cohereStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			switch event.Type {
			case "content-delta":
				if text := event.Delta.Message.Content.Text; text != "" {
					ch <- models.StreamChunk{Content: text}
				}
			case "message-end":
				done = true
				ch <- models.StreamChunk{Done: true}
				return
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- models.StreamChunk{Error: err}
			return
		}
		if !done {
			ch <- models.StreamChunk{Done: true}
		}
	}()

	return ch, nil
}
