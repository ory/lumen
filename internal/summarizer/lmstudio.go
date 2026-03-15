package summarizer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LMStudioSummarizer calls the LM Studio /v1/chat/completions endpoint.
type LMStudioSummarizer struct {
	model   string
	baseURL string
	client  *http.Client
}

// NewLMStudio creates a new LMStudioSummarizer.
func NewLMStudio(model, baseURL string) *LMStudioSummarizer {
	return &LMStudioSummarizer{
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Minute},
	}
}

type lmstudioChatRequest struct {
	Model    string                `json:"model"`
	Messages []lmstudioChatMessage `json:"messages"`
}

type lmstudioChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type lmstudioChatResponse struct {
	Choices []struct {
		Message lmstudioChatMessage `json:"message"`
	} `json:"choices"`
}

func (s *LMStudioSummarizer) chat(ctx context.Context, prompt string) (string, error) {
	reqBody := lmstudioChatRequest{
		Model:    s.model,
		Messages: []lmstudioChatMessage{{Role: "user", Content: prompt}},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("lmstudio chat request: %w", err)
	}
	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("lmstudio chat: status %d: %s", resp.StatusCode, string(body))
	}
	if readErr != nil {
		return "", fmt.Errorf("read lmstudio response: %w", readErr)
	}

	var chatResp lmstudioChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("unmarshal lmstudio response: %w", err)
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("lmstudio returned no choices")
	}
	return chatResp.Choices[0].Message.Content, nil
}

// SummarizeChunk generates a natural-language summary for a code chunk.
func (s *LMStudioSummarizer) SummarizeChunk(ctx context.Context, chunk ChunkInfo) (string, error) {
	return s.chat(ctx, chunkPrompt(chunk))
}

// SummarizeFile generates a file-level summary from its chunk summaries.
func (s *LMStudioSummarizer) SummarizeFile(ctx context.Context, chunkSummaries []string) (string, error) {
	return s.chat(ctx, filePrompt(chunkSummaries))
}
