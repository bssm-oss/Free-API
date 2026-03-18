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

	"github.com/heodongun/freeapi/internal/models"
)

// GeminiProvider handles Google Gemini API (non-OpenAI format).
type GeminiProvider struct {
	apiKey       string
	baseURL      string
	defaultModel string

	mu        sync.Mutex
	rateLimit models.RateLimitInfo
}

type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	SystemInstruction *geminiContent        `json:"systemInstruction,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
	ModelVersion string `json:"modelVersion"`
}

type geminiStreamChunkResp struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
}

func NewGemini(apiKey, baseURL, defaultModel string) *GeminiProvider {
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	return &GeminiProvider{
		apiKey:       apiKey,
		baseURL:      strings.TrimRight(baseURL, "/"),
		defaultModel: defaultModel,
	}
}

func (p *GeminiProvider) Name() string        { return "gemini" }
func (p *GeminiProvider) DefaultModel() string { return p.defaultModel }

func (p *GeminiProvider) IsAvailable() bool {
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

func (p *GeminiProvider) RateLimitStatus() models.RateLimitInfo {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.rateLimit
}

func (p *GeminiProvider) MarkRateLimited(info models.RateLimitInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rateLimit = info
}

func (p *GeminiProvider) Chat(ctx context.Context, messages []models.Message, opts models.ChatOptions) (*models.Response, error) {
	model := opts.Model
	if model == "" {
		model = p.defaultModel
	}

	reqBody := p.buildRequest(messages)
	body, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, model, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := SharedClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		resetAt := time.Now().Add(60 * time.Second)
		p.MarkRateLimited(models.RateLimitInfo{
			IsLimited: true,
			ResetAt:   resetAt,
		})
		return nil, &RateLimitError{Provider: "gemini", RetryAfter: resetAt}
	}

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var gResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&gResp); err != nil {
		return nil, fmt.Errorf("gemini: decode: %w", err)
	}

	if len(gResp.Candidates) == 0 || len(gResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini: empty response")
	}

	var contentParts []string
	for _, part := range gResp.Candidates[0].Content.Parts {
		contentParts = append(contentParts, part.Text)
	}

	return &models.Response{
		Content:      strings.Join(contentParts, ""),
		Model:        gResp.ModelVersion,
		Provider:     "gemini",
		TokensIn:     gResp.UsageMetadata.PromptTokenCount,
		TokensOut:    gResp.UsageMetadata.CandidatesTokenCount,
		FinishReason: gResp.Candidates[0].FinishReason,
	}, nil
}

func (p *GeminiProvider) ChatStream(ctx context.Context, messages []models.Message, opts models.ChatOptions) (<-chan models.StreamChunk, error) {
	model := opts.Model
	if model == "" {
		model = p.defaultModel
	}

	reqBody := p.buildRequest(messages)
	body, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse&key=%s", p.baseURL, model, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := StreamClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini: request failed: %w", err)
	}

	if resp.StatusCode == 429 {
		resp.Body.Close()
		resetAt := time.Now().Add(60 * time.Second)
		p.MarkRateLimited(models.RateLimitInfo{
			IsLimited: true,
			ResetAt:   resetAt,
		})
		return nil, &RateLimitError{Provider: "gemini", RetryAfter: resetAt}
	}

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("gemini: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	ch := make(chan models.StreamChunk, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")

			var chunk geminiStreamChunkResp
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			if len(chunk.Candidates) > 0 && len(chunk.Candidates[0].Content.Parts) > 0 {
				for _, part := range chunk.Candidates[0].Content.Parts {
					if part.Text != "" {
						ch <- models.StreamChunk{Content: part.Text}
					}
				}
			}
		}
		ch <- models.StreamChunk{Done: true}
	}()

	return ch, nil
}

func (p *GeminiProvider) buildRequest(messages []models.Message) geminiRequest {
	var req geminiRequest
	var contents []geminiContent

	for _, m := range messages {
		if m.Role == "system" {
			req.SystemInstruction = &geminiContent{
				Parts: []geminiPart{{Text: m.Content}},
			}
			continue
		}

		role := m.Role
		if role == "assistant" {
			role = "model"
		}

		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		})
	}

	req.Contents = contents
	return req
}
