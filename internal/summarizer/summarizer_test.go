package summarizer_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ory/lumen/internal/summarizer"
)

func ollamaChatHandler(t *testing.T, wantSubstring string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/chat" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		var req struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if len(req.Messages) == 0 {
			t.Error("expected at least one message")
		}
		userContent := req.Messages[len(req.Messages)-1].Content
		if !strings.Contains(userContent, wantSubstring) {
			t.Errorf("expected prompt to contain %q, got:\n%s", wantSubstring, userContent)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]string{"content": "This function does X."},
		})
	}
}

func TestOllamaSummarizer_SummarizeChunk(t *testing.T) {
	srv := httptest.NewServer(ollamaChatHandler(t, "MyFunc"))
	defer srv.Close()

	s := summarizer.NewOllama("qwen2.5-coder:7b", srv.URL)
	result, err := s.SummarizeChunk(context.Background(), summarizer.ChunkInfo{
		Kind:    "function",
		Symbol:  "MyFunc",
		Content: "func MyFunc() {}",
	})
	if err != nil {
		t.Fatalf("SummarizeChunk error: %v", err)
	}
	if result != "This function does X." {
		t.Fatalf("unexpected result: %q", result)
	}
}

func TestOllamaSummarizer_SummarizeFile(t *testing.T) {
	srv := httptest.NewServer(ollamaChatHandler(t, "chunk summary 1"))
	defer srv.Close()

	s := summarizer.NewOllama("qwen2.5-coder:7b", srv.URL)
	result, err := s.SummarizeFile(context.Background(), []string{"chunk summary 1", "chunk summary 2"})
	if err != nil {
		t.Fatalf("SummarizeFile error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty result")
	}
}

func TestOllamaSummarizer_ServerError_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	s := summarizer.NewOllama("qwen2.5-coder:7b", srv.URL)
	_, err := s.SummarizeChunk(context.Background(), summarizer.ChunkInfo{Kind: "function", Symbol: "F", Content: "f()"})
	if err == nil {
		t.Fatal("expected error from 500 response")
	}
}

func lmstudioChatHandler(t *testing.T, wantSubstring string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		var req struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if len(req.Messages) == 0 {
			t.Error("expected at least one message, got empty slice")
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		userContent := req.Messages[len(req.Messages)-1].Content
		if !strings.Contains(userContent, wantSubstring) {
			t.Errorf("expected prompt to contain %q, got:\n%s", wantSubstring, userContent)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": "This function does Y."}},
			},
		})
	}
}

func TestLMStudioSummarizer_SummarizeChunk(t *testing.T) {
	srv := httptest.NewServer(lmstudioChatHandler(t, "AnotherFunc"))
	defer srv.Close()

	s := summarizer.NewLMStudio("qwen2.5-coder:7b", srv.URL)
	result, err := s.SummarizeChunk(context.Background(), summarizer.ChunkInfo{
		Kind:    "method",
		Symbol:  "AnotherFunc",
		Content: "func (r *Recv) AnotherFunc() {}",
	})
	if err != nil {
		t.Fatalf("SummarizeChunk error: %v", err)
	}
	if result != "This function does Y." {
		t.Fatalf("unexpected result: %q", result)
	}
}

func TestLMStudioSummarizer_SummarizeFile(t *testing.T) {
	srv := httptest.NewServer(lmstudioChatHandler(t, "handles auth"))
	defer srv.Close()

	s := summarizer.NewLMStudio("qwen2.5-coder:7b", srv.URL)
	result, err := s.SummarizeFile(context.Background(), []string{"handles auth", "validates tokens"})
	if err != nil {
		t.Fatalf("SummarizeFile error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty result")
	}
}

func TestLMStudioSummarizer_ServerError_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	s := summarizer.NewLMStudio("qwen2.5-coder:7b", srv.URL)
	_, err := s.SummarizeChunk(context.Background(), summarizer.ChunkInfo{Kind: "function", Symbol: "F", Content: "f()"})
	if err == nil {
		t.Fatal("expected error from 500 response")
	}
}
