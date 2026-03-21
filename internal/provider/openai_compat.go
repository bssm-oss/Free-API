package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bssm-oss/Free-API/internal/models"
)

// OpenAICompatProvider handles all OpenAI-compatible APIs.
// Groq, Cerebras, Mistral, OpenRouter, GitHub Models all use this.
type OpenAICompatProvider struct {
	name         string
	apiKey       string
	baseURL      string
	defaultModel string
	extraHeaders map[string]string

	mu        sync.Mutex
	rateLimit models.RateLimitInfo
}

type openaiRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	Stream      bool            `json:"stream,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	MaxTokens   *int            `json:"max_tokens,omitempty"`
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

type openaiStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

type openaiErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error"`
}

func NewOpenAICompat(name, apiKey, baseURL, defaultModel string, extraHeaders map[string]string) *OpenAICompatProvider {
	return &OpenAICompatProvider{
		name:         name,
		apiKey:       apiKey,
		baseURL:      strings.TrimRight(baseURL, "/"),
		defaultModel: defaultModel,
		extraHeaders: extraHeaders,
	}
}

func (p *OpenAICompatProvider) Name() string { return p.name }

func (p *OpenAICompatProvider) DefaultModel() string { return p.defaultModel }

func (p *OpenAICompatProvider) IsAvailable() bool {
	if p.apiKey == "" || p.baseURL == "" {
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

func (p *OpenAICompatProvider) RateLimitStatus() models.RateLimitInfo {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.rateLimit
}

func (p *OpenAICompatProvider) MarkRateLimited(info models.RateLimitInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rateLimit = info
}

func (p *OpenAICompatProvider) Chat(ctx context.Context, messages []models.Message, opts models.ChatOptions) (*models.Response, error) {
	model := opts.Model
	if model == "" {
		model = p.defaultModel
	}

	oaiMsgs := make([]openaiMessage, len(messages))
	for i, m := range messages {
		oaiMsgs[i] = openaiMessage{Role: m.Role, Content: m.Content}
	}

	reqBody := openaiRequest{
		Model:    model,
		Messages: oaiMsgs,
		Stream:   false,
	}
	if opts.Temperature > 0 {
		reqBody.Temperature = &opts.Temperature
	}
	if opts.MaxTokens > 0 {
		reqBody.MaxTokens = &opts.MaxTokens
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	for k, v := range p.extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := SharedClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: request failed: %w", p.name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		resetAt := p.handleRateLimit(resp)
		return nil, &RateLimitError{Provider: p.name, RetryAfter: resetAt}
	}

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		var errResp openaiErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("%s: %s", p.name, errResp.Error.Message)
		}
		return nil, fmt.Errorf("%s: HTTP %d: %s", p.name, resp.StatusCode, string(respBody))
	}

	var oaiResp openaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&oaiResp); err != nil {
		return nil, fmt.Errorf("%s: decode response: %w", p.name, err)
	}

	if len(oaiResp.Choices) == 0 {
		return nil, fmt.Errorf("%s: empty response", p.name)
	}

	return &models.Response{
		Content:      oaiResp.Choices[0].Message.Content,
		Model:        oaiResp.Model,
		Provider:     p.name,
		TokensIn:     oaiResp.Usage.PromptTokens,
		TokensOut:    oaiResp.Usage.CompletionTokens,
		FinishReason: oaiResp.Choices[0].FinishReason,
	}, nil
}

func (p *OpenAICompatProvider) ChatStream(ctx context.Context, messages []models.Message, opts models.ChatOptions) (<-chan models.StreamChunk, error) {
	model := opts.Model
	if model == "" {
		model = p.defaultModel
	}

	oaiMsgs := make([]openaiMessage, len(messages))
	for i, m := range messages {
		oaiMsgs[i] = openaiMessage{Role: m.Role, Content: m.Content}
	}

	reqBody := openaiRequest{
		Model:    model,
		Messages: oaiMsgs,
		Stream:   true,
	}
	if opts.Temperature > 0 {
		reqBody.Temperature = &opts.Temperature
	}
	if opts.MaxTokens > 0 {
		reqBody.MaxTokens = &opts.MaxTokens
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	for k, v := range p.extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := StreamClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: request failed: %w", p.name, err)
	}

	if resp.StatusCode == 429 {
		resp.Body.Close()
		resetAt := p.handleRateLimit(resp)
		return nil, &RateLimitError{Provider: p.name, RetryAfter: resetAt}
	}

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("%s: HTTP %d: %s", p.name, resp.StatusCode, string(respBody))
	}

	ch := make(chan models.StreamChunk, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				ch <- models.StreamChunk{Done: true}
				return
			}

			var chunk openaiStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				ch <- models.StreamChunk{Content: chunk.Choices[0].Delta.Content}
			}
		}
		if err := scanner.Err(); err != nil {
			ch <- models.StreamChunk{Error: err}
		}
	}()

	return ch, nil
}

func (p *OpenAICompatProvider) handleRateLimit(resp *http.Response) time.Time {
	resetAt := time.Now().Add(60 * time.Second) // default 60s

	if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
		if secs, err := strconv.Atoi(retryAfter); err == nil {
			resetAt = time.Now().Add(time.Duration(secs) * time.Second)
		}
	}

	if remaining := resp.Header.Get("X-Ratelimit-Remaining-Requests"); remaining == "0" {
		if resetStr := resp.Header.Get("X-Ratelimit-Reset-Requests"); resetStr != "" {
			if d, err := time.ParseDuration(resetStr); err == nil {
				resetAt = time.Now().Add(d)
			}
		}
	}

	p.MarkRateLimited(models.RateLimitInfo{
		Remaining: 0,
		ResetAt:   resetAt,
		IsLimited: true,
	})
	return resetAt
}

// RateLimitError signals a 429 response.
type RateLimitError struct {
	Provider   string
	RetryAfter time.Time
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("%s: rate limited until %s", e.Provider, e.RetryAfter.Format(time.RFC3339))
}
