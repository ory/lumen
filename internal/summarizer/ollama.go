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

// OllamaSummarizer calls the Ollama /api/chat endpoint.
type OllamaSummarizer struct {
	model   string
	baseURL string
	client  *http.Client
}

// NewOllama creates a new OllamaSummarizer.
func NewOllama(model, baseURL string) *OllamaSummarizer {
	return &OllamaSummarizer{
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Minute},
	}
}

type ollamaChatRequest struct {
	Model    string              `json:"model"`
	Messages []ollamaChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
}

type ollamaChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResponse struct {
	Message ollamaChatMessage `json:"message"`
}

func (s *OllamaSummarizer) chat(ctx context.Context, prompt string) (string, error) {
	reqBody := ollamaChatRequest{
		Model:    s.model,
		Messages: []ollamaChatMessage{{Role: "user", Content: prompt}},
		Stream:   false,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/api/chat", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama chat request: %w", err)
	}
	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama chat: status %d: %s", resp.StatusCode, string(body))
	}
	if readErr != nil {
		return "", fmt.Errorf("read ollama response: %w", readErr)
	}

	var chatResp ollamaChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("unmarshal ollama response: %w", err)
	}
	return chatResp.Message.Content, nil
}

// SummarizeChunk generates a natural-language summary for a code chunk.
func (s *OllamaSummarizer) SummarizeChunk(ctx context.Context, chunk ChunkInfo) (string, error) {
	return s.chat(ctx, chunkPrompt(chunk))
}

// SummarizeFile generates a file-level summary from its chunk summaries.
func (s *OllamaSummarizer) SummarizeFile(ctx context.Context, chunkSummaries []string) (string, error) {
	return s.chat(ctx, filePrompt(chunkSummaries))
}
