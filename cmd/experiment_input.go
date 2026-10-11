// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"slices"

	"github.com/ory/lumen/internal/embedder"
	"github.com/spf13/cobra"
)

// The experiment accepts explicit relevance judgments rather than inventing
// synthetic quality claims from vector distances alone.
type inputDataset struct {
	Chunks []struct {
		Path string `json:"path"`
		Body string `json:"body"`
	} `json:"chunks"`
	Queries []struct {
		Text          string   `json:"text"`
		RelevantPaths []string `json:"relevant_paths"`
	} `json:"queries"`
}

type inputMetrics struct {
	Mode         string  `json:"mode"`
	UniqueInputs int     `json:"unique_inputs"`
	Recall       float64 `json:"recall_at_k"`
	MRR          float64 `json:"mrr_at_k"`
}

func init() {
	command := &cobra.Command{Use: "experiment-input <dataset.json>", Short: "Compare path-aware and body-only embeddings against labeled queries", Args: cobra.ExactArgs(1), RunE: runInputExperiment}
	command.Flags().Int("k", 10, "number of retrieved chunks to evaluate")
	command.Flags().String("model", "", "embedding model")
	command.Flags().String("backend", "", "embedding backend")
	rootCmd.AddCommand(command)
}

func runInputExperiment(cmd *cobra.Command, args []string) error {
	data, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	var dataset inputDataset
	if err := json.Unmarshal(data, &dataset); err != nil {
		return err
	}
	k, _ := cmd.Flags().GetInt("k")
	if k < 1 || len(dataset.Chunks) == 0 || len(dataset.Queries) == 0 {
		return fmt.Errorf("positive k, chunks, and labeled queries are required")
	}
	paths := map[string]bool{}
	for _, chunk := range dataset.Chunks {
		if chunk.Path == "" || chunk.Body == "" {
			return fmt.Errorf("each chunk requires path and body")
		}
		paths[chunk.Path] = true
	}
	for _, query := range dataset.Queries {
		if query.Text == "" || len(query.RelevantPaths) == 0 {
			return fmt.Errorf("each query requires text and relevant_paths")
		}
		for _, path := range query.RelevantPaths {
			if !paths[path] {
				return fmt.Errorf("relevance label %q has no corpus chunk", path)
			}
		}
	}
	cfg, err := loadConfigWithFlags(cmd)
	if err != nil {
		return err
	}
	emb := newEmbedder(cfg)
	model, dimensions := emb.ModelName(), emb.Dimensions()
	ctx := embedder.BackgroundContext(cmd.Context())
	encode := func(texts []string) ([][]float32, error) {
		var result [][]float32
		for start := 0; start < len(texts); start += 32 {
			batch := texts[start:min(start+32, len(texts))]
			vectors, err := emb.Embed(ctx, batch)
			if err != nil {
				return nil, err
			}
			if emb.ModelName() != model || emb.Dimensions() != dimensions {
				return nil, fmt.Errorf("embedding profile changed during experiment; retry with a single model")
			}
			if len(vectors) != len(batch) {
				return nil, fmt.Errorf("backend returned an unexpected vector count")
			}
			result = append(result, vectors...)
		}
		return result, nil
	}
	queries := make([]string, len(dataset.Queries))
	for i, q := range dataset.Queries {
		queries[i] = q.Text
	}
	queryVectors, err := encode(queries)
	if err != nil {
		return err
	}
	var report []inputMetrics
	for _, mode := range []string{"path", "body"} {
		positions := make([]int, len(dataset.Chunks))
		unique := map[string]int{}
		var inputs []string
		for i, chunk := range dataset.Chunks {
			text := chunk.Body
			if mode == "path" {
				text = "// " + chunk.Path + "\n" + text
			}
			position, ok := unique[text]
			if !ok {
				position = len(inputs)
				unique[text] = position
				inputs = append(inputs, text)
			}
			positions[i] = position
		}
		vectors, err := encode(inputs)
		if err != nil {
			return err
		}
		metrics := inputMetrics{Mode: mode, UniqueInputs: len(inputs)}
		for qi, query := range dataset.Queries {
			type match struct {
				path    string
				score   float64
				ordinal int
			}
			matches := make([]match, len(dataset.Chunks))
			for i, chunk := range dataset.Chunks {
				score, err := cosineScore(queryVectors[qi], vectors[positions[i]])
				if err != nil {
					return err
				}
				matches[i] = match{chunk.Path, score, i}
			}
			slices.SortFunc(matches, func(a, b match) int {
				if a.score > b.score {
					return -1
				}
				if a.score < b.score {
					return 1
				}
				if a.path < b.path {
					return -1
				}
				if a.path > b.path {
					return 1
				}
				return a.ordinal - b.ordinal
			})
			relevant := map[string]bool{}
			for _, path := range query.RelevantPaths {
				relevant[path] = true
			}
			found := map[string]bool{}
			rr := 0.0
			for rank, m := range matches[:min(k, len(matches))] {
				if relevant[m.path] {
					found[m.path] = true
					if rr == 0 {
						rr = 1 / float64(rank+1)
					}
				}
			}
			metrics.Recall += float64(len(found)) / float64(len(relevant))
			metrics.MRR += rr
		}
		metrics.Recall /= float64(len(dataset.Queries))
		metrics.MRR /= float64(len(dataset.Queries))
		report = append(report, metrics)
	}
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(struct {
		Model   string         `json:"model"`
		K       int            `json:"k"`
		Results []inputMetrics `json:"results"`
	}{model, k, report})
}

func cosineScore(a, b []float32) (float64, error) {
	if len(a) == 0 || len(a) != len(b) {
		return 0, fmt.Errorf("incompatible embedding dimensions")
	}
	var dot, aa, bb float64
	for i, x := range a {
		y := float64(b[i])
		v := float64(x)
		dot += v * y
		aa += v * v
		bb += y * y
	}
	if aa == 0 || bb == 0 {
		return 0, fmt.Errorf("backend returned a zero vector")
	}
	score := dot / math.Sqrt(aa*bb)
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return 0, fmt.Errorf("backend returned non-finite vectors")
	}
	return score, nil
}
