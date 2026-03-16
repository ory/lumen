// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package embedder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"

	"github.com/sethvargo/go-retry"
)

// OpenAI implements the Embedder interface using an OpenAI-compatible
// /v1/embeddings endpoint. It works with OpenAI, Voyage AI, Azure OpenAI,
// Together AI, and any other service exposing the same wire format.
type OpenAI struct {
	model      string
	dimensions int
	baseURL    string
	apiKey     string
	client     *http.Client
}

// NewOpenAI creates a new OpenAI-compatible embedder.
// baseURL is the API base URL (e.g. "https://api.openai.com").
// apiKey is the Bearer token for authentication.
func NewOpenAI(model string, dimensions int, baseURL string, apiKey string) (*OpenAI, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required for OpenAI-compatible backend")
	}
	return &OpenAI{
		model:      model,
		dimensions: dimensions,
		baseURL:    baseURL,
		apiKey:     apiKey,
		client: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}, nil
}

// Dimensions returns the embedding vector dimensionality.
func (o *OpenAI) Dimensions() int {
	return o.dimensions
}

// ModelName returns the model name used for embeddings.
func (o *OpenAI) ModelName() string {
	return o.model
}

// openaiEmbedRequest is the JSON body sent to /v1/embeddings.
type openaiEmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// openaiEmbedItem is a single embedding item in the response.
type openaiEmbedItem struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

// openaiEmbedResponse is the JSON body returned from /v1/embeddings.
type openaiEmbedResponse struct {
	Data []openaiEmbedItem `json:"data"`
}

// Embed converts texts into embedding vectors, splitting into batches of 32.
func (o *OpenAI) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	var allVecs [][]float32
	for i := 0; i < len(texts); i += embedBatchSize {
		batch := texts[i:min(i+embedBatchSize, len(texts))]

		vecs, err := o.embedBatch(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("embedding batch starting at %d: %w", i, err)
		}
		allVecs = append(allVecs, vecs...)
	}

	return allVecs, nil
}

// embedBatch sends a single batch of texts to the /v1/embeddings endpoint.
// Retries up to embedMaxRetries times on transient errors (5xx, 429 rate limits,
// network failures), respecting context cancellation between attempts.
func (o *OpenAI) embedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	bodyBytes, err := json.Marshal(openaiEmbedRequest{
		Model: o.model,
		Input: texts,
	})
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	b := retry.NewExponential(100 * time.Millisecond)

	var embedResp openaiEmbedResponse
	err = retry.Do(ctx, retry.WithMaxRetries(embedMaxRetries-1, b), func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/v1/embeddings", bytes.NewReader(bodyBytes))
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+o.apiKey)

		resp, err := o.client.Do(req)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("request failed: %w", err))
		}

		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			return retry.RetryableError(fmt.Errorf("rate limited: status %d", resp.StatusCode))
		}
		if resp.StatusCode >= 500 {
			return retry.RetryableError(fmt.Errorf("server error: status %d", resp.StatusCode))
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
		}
		if readErr != nil {
			return fmt.Errorf("reading response body: %w", readErr)
		}

		return json.Unmarshal(body, &embedResp)
	})
	if err != nil {
		return nil, fmt.Errorf("openai embed: %w", err)
	}

	// Sort by index — OpenAI spec allows out-of-order responses.
	slices.SortFunc(embedResp.Data, func(a, b openaiEmbedItem) int {
		return a.Index - b.Index
	})

	vecs := make([][]float32, len(embedResp.Data))
	for i, item := range embedResp.Data {
		vecs[i] = item.Embedding
	}
	return vecs, nil
}
