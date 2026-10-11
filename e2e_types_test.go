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

// This file intentionally has no e2e build tag because the snapshot matcher
// unit tests share the black-box search result type without requiring Ollama.

package main

import (
	"fmt"
	"html"
	"math"
	"regexp"
	"strconv"
	"strings"
)

const langSnapshotDirectory = "testdata/snapshots"

// searchResultItem is shared by the black-box E2E tests and the snapshot
// matcher unit tests. It mirrors the semantic search tool's result shape while
// remaining decoupled from the production command types.
type searchResultItem struct {
	FilePath  string  `json:"file_path"`
	Symbol    string  `json:"symbol"`
	Kind      string  `json:"kind"`
	StartLine int     `json:"start_line"`
	EndLine   int     `json:"end_line"`
	Score     float32 `json:"score"`
	// HasScore is false for BM25 related context, which carries provenance
	// (RelatedTo, Relation) instead of a relevance score.
	HasScore  bool   `json:"-"`
	RelatedTo string `json:"related_to,omitempty"`
	Relation  string `json:"relation,omitempty"`
	Content   string `json:"content,omitempty"`
}

type semanticSearchOutput struct {
	Retrieval    string             `json:"retrieval,omitempty"`
	Results      []searchResultItem `json:"results"`
	Reindexed    bool               `json:"reindexed"`
	IndexedFiles int                `json:"indexed_files,omitempty"`
}

// fileRe matches <result:file filename="..."> opening tags.
var fileRe = regexp.MustCompile(`^<result:file filename="([^"]+)">$`)

// chunkRe matches <result:chunk ...> opening tags. Direct hits carry a score
// in any finite Go %g/%f form (semantic "0.66", BM25 "%.6g" forms such as
// "1000", "12.3457" or "1.23457e+06"); BM25 related context instead carries
// related-to and relation attributes and no score:
//
//	<result:chunk line-start="10" line-end="19" symbol="ValidateToken" kind="function" score="0.66">
//	<result:chunk line-start="3" line-end="9" symbol="reserve" kind="function" related-to="a.go:Checkout" relation="direct_call">
var chunkRe = regexp.MustCompile(`^\s*<result:chunk line-start="(\d+)" line-end="(\d+)" symbol="([^"]*)" kind="([^"]*)"(?: score="([-+]?(?:\d+\.?\d*|\.\d+)(?:[eE][-+]?\d+)?)"| related-to="([^"]*)" relation="([^"]*)")>$`)

var retrievalRe = regexp.MustCompile(`\[retrieval=(\w+)`)

// SearchOutput mode reported in the result header ("bm25" or "semantic"; empty
// for historical semantic output without a retrieval tag).
func searchRetrieval(text string) string {
	if m := retrievalRe.FindStringSubmatch(text); m != nil {
		return m[1]
	}
	return ""
}

// parseSearchOutput parses both semantic and BM25 result text. It fails on
// malformed chunk headers rather than silently dropping results.
func parseSearchOutput(text string) (semanticSearchOutput, error) {
	var out semanticSearchOutput
	out.Retrieval = searchRetrieval(text)

	// Parse "(indexed M files)" first — present in both "Found N results" and "No results found." output.
	if m := regexp.MustCompile(`\(indexed (\d+) files\)`).FindStringSubmatch(text); m != nil {
		out.IndexedFiles, _ = strconv.Atoi(m[1])
		out.Reindexed = true
	}
	if strings.HasPrefix(text, "No results found") {
		return out, nil
	}

	lines := strings.Split(text, "\n")
	currentFile := ""
	inChunk := false
	var chunkItem searchResultItem
	var contentLines []string

	flushChunk := func() {
		if !inChunk {
			return
		}
		chunkItem.Content = strings.TrimSpace(strings.Join(contentLines, "\n"))
		out.Results = append(out.Results, chunkItem)
		inChunk = false
		contentLines = nil
	}

	for _, line := range lines {
		if m := fileRe.FindStringSubmatch(line); m != nil {
			flushChunk()
			currentFile = html.UnescapeString(m[1])
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "<result:chunk ") && !inChunk {
			m := chunkRe.FindStringSubmatch(line)
			if m == nil {
				return out, fmt.Errorf("malformed result chunk header: %q", line)
			}
			startLine, _ := strconv.Atoi(m[1])
			endLine, _ := strconv.Atoi(m[2])
			chunkItem = searchResultItem{
				FilePath:  currentFile,
				Symbol:    html.UnescapeString(m[3]),
				Kind:      html.UnescapeString(m[4]),
				StartLine: startLine,
				EndLine:   endLine,
				RelatedTo: html.UnescapeString(m[6]),
				Relation:  m[7],
			}
			if m[5] != "" {
				score, err := strconv.ParseFloat(m[5], 64)
				if err != nil || math.IsInf(score, 0) || math.IsNaN(score) {
					return out, fmt.Errorf("non-finite score %q", m[5])
				}
				chunkItem.Score = float32(score)
				chunkItem.HasScore = true
			}
			inChunk = true
			continue
		}
		if inChunk {
			trimmed := strings.TrimPrefix(line, "  ")
			if trimmed == "</result:chunk>" {
				flushChunk()
				continue
			}
			contentLines = append(contentLines, trimmed)
		}
	}
	flushChunk()

	return out, nil
}
