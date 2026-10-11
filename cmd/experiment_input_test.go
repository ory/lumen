// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInputExperimentMeasuresRetrievalAndReuse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embed" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"models":[{"name":"experiment"}]}`))
			return
		}
		var request struct {
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		vectors := make([][]float32, len(request.Input))
		for i, text := range request.Input {
			if strings.Contains(text, "cat") {
				vectors[i] = []float32{1, 0}
			} else {
				vectors[i] = []float32{0, 1}
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"embeddings": vectors})
	}))
	defer server.Close()
	command := flagsTestHarness(t, fmt.Sprintf("servers:\n  - backend: ollama\n    host: %s\n    model: experiment\n    dims: 2\n", server.URL))
	command.Flags().Int("k", 1, "")
	command.SetContext(context.Background())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	dataset := filepath.Join(t.TempDir(), "dataset.json")
	if err := os.WriteFile(dataset, []byte(`{"chunks":[{"path":"a.go","body":"cat"},{"path":"b.go","body":"cat"},{"path":"c.go","body":"dog"}],"queries":[{"text":"cat","relevant_paths":["a.go"]}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	command.SetOut(&output)
	if err := runInputExperiment(command, []string{dataset}); err != nil {
		t.Fatal(err)
	}
	var report struct {
		Results []inputMetrics `json:"results"`
	}
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Results) != 2 || report.Results[0].UniqueInputs != 3 || report.Results[1].UniqueInputs != 2 {
		t.Fatalf("reuse metrics: %s", output.String())
	}
	for _, result := range report.Results {
		if result.Recall != 1 || result.MRR != 1 {
			t.Fatalf("retrieval metrics: %+v", result)
		}
	}
}
