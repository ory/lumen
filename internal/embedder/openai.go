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
	"strings"
	"time"

	"github.com/sethvargo/go-retry"
)

// OpenAI implements the Embedder interface using an OpenAI-compatible
// /v1/embeddings endpoint. It works with OpenAI itself and with any other
// service (internal gateways, proxies) exposing the same wire format.
type OpenAI struct {
	model      string
	dimensions int
	baseURL    string
	apiKey     string
	client     *http.Client
}

// NewOpenAI creates a new OpenAI-compatible embedder.
// baseURL is the API base URL (e.g. "https://api.openai.com", or
// "https://api.openai.com/v1" — a trailing "/v1" is stripped so callers
// following either convention don't end up with "/v1/v1/embeddings").
// apiKey is the Bearer token for authentication; when empty, no
// Authorization header is sent, since some internal gateways trust network
// position rather than a token.
func NewOpenAI(model string, dimensions int, baseURL string, apiKey string) (*OpenAI, error) {
	client := &http.Client{Timeout: 10 * time.Minute}
	if apiKey != "" {
		client.CheckRedirect = refuseHTTPSDowngrade
	}
	return &OpenAI{
		model:      model,
		dimensions: dimensions,
		baseURL:    normalizeBaseURL(baseURL),
		apiKey:     apiKey,
		client:     client,
	}, nil
}

// maxRedirects mirrors net/http's default limit, which a custom
// CheckRedirect replaces.
const maxRedirects = 10

// refuseHTTPSDowngrade is the redirect policy for requests carrying a bearer
// token. net/http forwards Authorization on same-host redirects regardless of
// scheme, so following an https→http hop would send the token in plaintext
// and bypass the config-time https requirement. Plain http origins (loopback
// gateways) are unaffected.
func refuseHTTPSDowngrade(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return fmt.Errorf("stopped after %d redirects", maxRedirects)
	}
	for _, prev := range via {
		if prev.URL.Scheme == "https" && req.URL.Scheme != "https" {
			return fmt.Errorf("refusing redirect from https to %s", req.URL.Redacted())
		}
	}
	return nil
}

// normalizeBaseURL strips a trailing slash and a trailing "/v1" so the
// embed/health-probe code can append "/v1/..." exactly once regardless of
// whether the configured host already includes the version prefix.
func normalizeBaseURL(raw string) string {
	trimmed := strings.TrimRight(raw, "/")
	trimmed = strings.TrimSuffix(trimmed, "/v1")
	return strings.TrimRight(trimmed, "/")
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
// Retries up to embedMaxRetries times on transient errors (5xx, 429 rate
// limits, network failures), respecting context cancellation between
// attempts.
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
		if o.apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+o.apiKey)
		}

		resp, err := o.client.Do(req)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("request failed: %w", err))
		}

		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			return retry.RetryableError(&EmbedError{StatusCode: resp.StatusCode, Message: string(body)})
		}
		if resp.StatusCode != http.StatusOK {
			return &EmbedError{StatusCode: resp.StatusCode, Message: string(body)}
		}
		if readErr != nil {
			return fmt.Errorf("reading response body: %w", readErr)
		}

		return json.Unmarshal(body, &embedResp)
	})
	if err != nil {
		// Identify the failing server by base URL rather than a hardcoded
		// "openai" label — this embedder also backs the LMStudio wrapper, so
		// a static backend name here would mislabel LM Studio failures.
		return nil, fmt.Errorf("embed request to %s: %w", o.baseURL, err)
	}

	// The OpenAI spec allows out-of-order responses but guarantees one item
	// per input, indexed 0..len(texts)-1. Validate that explicitly instead of
	// trusting response order/length — a misbehaving gateway that drops,
	// duplicates, or mis-sizes an item must fail loudly rather than silently
	// misalign embeddings with their source texts.
	if len(embedResp.Data) != len(texts) {
		return nil, fmt.Errorf("embed response: got %d embeddings, want %d", len(embedResp.Data), len(texts))
	}
	vecs := make([][]float32, len(texts))
	seen := make([]bool, len(texts))
	for _, item := range embedResp.Data {
		if item.Index < 0 || item.Index >= len(texts) {
			return nil, fmt.Errorf("embed response: index %d out of range [0,%d)", item.Index, len(texts))
		}
		if seen[item.Index] {
			return nil, fmt.Errorf("embed response: duplicate index %d", item.Index)
		}
		if len(item.Embedding) != o.dimensions {
			return nil, fmt.Errorf("embed response: item %d has %d dimensions, want %d", item.Index, len(item.Embedding), o.dimensions)
		}
		seen[item.Index] = true
		vecs[item.Index] = item.Embedding
	}
	return vecs, nil
}
