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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ory/lumen/internal/config"
)

// ProbeServer verifies that an embedding service is reachable and that the
// configured model is actually loaded. A listening LM Studio or Ollama process
// without the requested model cannot serve embeddings and must not be selected
// as a healthy failover target.
func ProbeServer(ctx context.Context, srv config.ServerConfig) error {
	endpoint := strings.TrimRight(srv.Host, "/") + "/api/tags"
	if srv.Backend == config.BackendLMStudio {
		endpoint = strings.TrimRight(srv.Host, "/") + "/v1/models"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create health request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("service unreachable: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("service returned HTTP %d", resp.StatusCode)
	}

	var available []string
	switch srv.Backend {
	case config.BackendOllama:
		var body struct {
			Models []struct {
				Name  string `json:"name"`
				Model string `json:"model"`
			} `json:"models"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return fmt.Errorf("decode model list: %w", err)
		}
		for _, model := range body.Models {
			available = append(available, model.Name, model.Model)
		}
	case config.BackendLMStudio:
		var body struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return fmt.Errorf("decode model list: %w", err)
		}
		for _, model := range body.Data {
			available = append(available, model.ID)
		}
	default:
		return fmt.Errorf("unknown backend %q", srv.Backend)
	}

	for _, model := range available {
		if model == srv.Model || strings.TrimSuffix(model, ":latest") == strings.TrimSuffix(srv.Model, ":latest") {
			return nil
		}
	}
	return fmt.Errorf("configured model %q is not loaded", srv.Model)
}
