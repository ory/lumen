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

// This file intentionally has no e2e build tag so the snapshot matcher is
// unit-tested without requiring Ollama.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var langSnapshotResultPattern = regexp.MustCompile(`^(.+):(\d+)-(\d+)  (.+)$`)

type langResultIdentity struct {
	filePath string
	symbol   string
	kind     string
}

type langSnapshotResult struct {
	identity  langResultIdentity
	startLine int
	endLine   int
}

func (r langSnapshotResult) String() string {
	return fmt.Sprintf("%s:%d-%d  %s (%s)", r.identity.filePath, r.startLine, r.endLine, r.identity.symbol, r.identity.kind)
}

type parsedLangSnapshot struct {
	declaredCount int
	results       []langSnapshotResult
}

func parseLangSnapshot(snapshot string) (parsedLangSnapshot, error) {
	trimmed := strings.TrimRight(snapshot, "\r\n")
	if trimmed == "" {
		return parsedLangSnapshot{}, fmt.Errorf("snapshot is empty")
	}
	lines := strings.Split(trimmed, "\n")

	const countPrefix = "results: "
	header := strings.TrimSuffix(lines[0], "\r")
	if !strings.HasPrefix(header, countPrefix) {
		return parsedLangSnapshot{}, fmt.Errorf("malformed snapshot header %q", header)
	}

	declaredCount, err := strconv.Atoi(strings.TrimPrefix(header, countPrefix))
	if err != nil || declaredCount < 0 {
		return parsedLangSnapshot{}, fmt.Errorf("malformed snapshot result count %q", strings.TrimPrefix(header, countPrefix))
	}
	if got := len(lines) - 1; got != declaredCount {
		return parsedLangSnapshot{}, fmt.Errorf("snapshot declares %d results but contains %d result lines", declaredCount, got)
	}

	results := make([]langSnapshotResult, 0, declaredCount)
	for i, rawLine := range lines[1:] {
		result, err := parseLangSnapshotResult(strings.TrimSuffix(rawLine, "\r"))
		if err != nil {
			return parsedLangSnapshot{}, fmt.Errorf("malformed snapshot result line %d: %w", i+2, err)
		}
		results = append(results, result)
	}

	return parsedLangSnapshot{declaredCount: declaredCount, results: results}, nil
}

func parseLangSnapshotResult(line string) (langSnapshotResult, error) {
	parts := langSnapshotResultPattern.FindStringSubmatch(line)
	if parts == nil {
		return langSnapshotResult{}, fmt.Errorf("expected location and identity separated by two spaces: %q", line)
	}

	startLine, _ := strconv.Atoi(parts[2])
	endLine, _ := strconv.Atoi(parts[3])
	if startLine <= 0 || endLine < startLine {
		return langSnapshotResult{}, fmt.Errorf("invalid line range %d-%d", startLine, endLine)
	}

	description := parts[4]
	kindStart := strings.LastIndex(description, " (")
	if kindStart < 1 || !strings.HasSuffix(description, ")") {
		return langSnapshotResult{}, fmt.Errorf("invalid symbol and kind %q", description)
	}

	identity := langResultIdentity{
		filePath: parts[1],
		symbol:   description[:kindStart],
		kind:     description[kindStart+2 : len(description)-1],
	}
	if strings.TrimSpace(identity.filePath) == "" || strings.TrimSpace(identity.symbol) == "" || strings.TrimSpace(identity.kind) == "" {
		return langSnapshotResult{}, fmt.Errorf("file, symbol, and kind must be non-empty: %q", line)
	}

	return langSnapshotResult{identity: identity, startLine: startLine, endLine: endLine}, nil
}

func compareLangSnapshot(snapshot string, actual []searchResultItem) error {
	expected, err := parseLangSnapshot(snapshot)
	if err != nil {
		return err
	}

	actualResults := make([]langSnapshotResult, 0, len(actual))
	for i, result := range actual {
		if err := validateLangResult(result); err != nil {
			return fmt.Errorf("invalid actual result %d: %w", i+1, err)
		}
		actualResults = append(actualResults, langSnapshotResult{
			identity: langResultIdentity{
				filePath: result.FilePath,
				symbol:   result.Symbol,
				kind:     result.Kind,
			},
			startLine: result.StartLine,
			endLine:   result.EndLine,
		})
	}

	expectedMatches, actualMatches := matchLangSnapshotResults(expected.results, actualResults)
	matchedCount := 0
	for _, matched := range expectedMatches {
		if matched {
			matchedCount++
		}
	}
	requiredMatches := (len(expected.results) + 1) / 2
	if matchedCount >= requiredMatches {
		return nil
	}

	unmatchedExpected := unmatchedLangResults(expected.results, expectedMatches)
	unmatchedActual := unmatchedLangResults(actualResults, actualMatches)

	var message strings.Builder
	message.WriteString("language snapshot comparison failed:\n")
	fmt.Fprintf(&message, "expected result count: %d\n", expected.declaredCount)
	fmt.Fprintf(&message, "actual result count: %d\n", len(actual))
	fmt.Fprintf(&message, "matched rows: %d/%d (required at least %d)\n", matchedCount, len(expected.results), requiredMatches)
	writeResultList(&message, "unmatched expected rows", unmatchedExpected)
	writeResultList(&message, "unmatched actual rows", unmatchedActual)
	return fmt.Errorf("%s", strings.TrimSuffix(message.String(), "\n"))
}

func matchLangSnapshotResults(expected, actual []langSnapshotResult) ([]bool, []bool) {
	// Use augmenting paths to find a maximum one-to-one matching. A greedy
	// matcher can undercount when one broad range overlaps several narrower ones.
	actualToExpected := make([]int, len(actual))
	for i := range actualToExpected {
		actualToExpected[i] = -1
	}

	var augment func(int, []bool) bool
	augment = func(expectedIndex int, seenActual []bool) bool {
		for actualIndex := range actual {
			if seenActual[actualIndex] || !langSnapshotResultsMatch(expected[expectedIndex], actual[actualIndex]) {
				continue
			}
			seenActual[actualIndex] = true
			if actualToExpected[actualIndex] == -1 || augment(actualToExpected[actualIndex], seenActual) {
				actualToExpected[actualIndex] = expectedIndex
				return true
			}
		}
		return false
	}

	for expectedIndex := range expected {
		augment(expectedIndex, make([]bool, len(actual)))
	}

	expectedMatches := make([]bool, len(expected))
	actualMatches := make([]bool, len(actual))
	for actualIndex, expectedIndex := range actualToExpected {
		if expectedIndex >= 0 {
			expectedMatches[expectedIndex] = true
			actualMatches[actualIndex] = true
		}
	}
	return expectedMatches, actualMatches
}

func langSnapshotResultsMatch(expected, actual langSnapshotResult) bool {
	return expected.identity == actual.identity &&
		expected.startLine <= actual.endLine && actual.startLine <= expected.endLine
}

func validateLangResult(result searchResultItem) error {
	switch {
	case strings.TrimSpace(result.FilePath) == "":
		return fmt.Errorf("file path is empty")
	case strings.TrimSpace(result.Symbol) == "":
		return fmt.Errorf("symbol is empty")
	case strings.TrimSpace(result.Kind) == "":
		return fmt.Errorf("kind is empty")
	case result.StartLine <= 0:
		return fmt.Errorf("start line must be positive, got %d", result.StartLine)
	case result.EndLine < result.StartLine:
		return fmt.Errorf("end line %d is before start line %d", result.EndLine, result.StartLine)
	default:
		return nil
	}
}

func unmatchedLangResults(results []langSnapshotResult, matches []bool) []string {
	unmatched := make([]string, 0)
	for i, result := range results {
		if !matches[i] {
			unmatched = append(unmatched, result.String())
		}
	}
	return unmatched
}

func writeResultList(message *strings.Builder, label string, results []string) {
	fmt.Fprintf(message, "%s (%d):\n", label, len(results))
	for _, result := range results {
		fmt.Fprintf(message, "  - %s\n", result)
	}
}

func TestCompareLangSnapshot(t *testing.T) {
	t.Parallel()

	expected := "" +
		"results: 4\n" +
		"alpha.go:10-20  Alpha (function)\n" +
		"beta.go:30-40  Beta (type)\n" +
		"gamma.go:50-60  Gamma (method)\n" +
		"delta.go:70-80  Delta (variable)\n\n"
	baseline := []searchResultItem{
		{FilePath: "alpha.go", Symbol: "Alpha", Kind: "function", StartLine: 10, EndLine: 20},
		{FilePath: "beta.go", Symbol: "Beta", Kind: "type", StartLine: 30, EndLine: 40},
		{FilePath: "gamma.go", Symbol: "Gamma", Kind: "method", StartLine: 50, EndLine: 60},
		{FilePath: "delta.go", Symbol: "Delta", Kind: "variable", StartLine: 70, EndLine: 80},
	}

	tests := []struct {
		name    string
		actual  []searchResultItem
		wantErr string
	}{
		{name: "exact match", actual: baseline},
		{
			name: "overlapping line range drift",
			actual: []searchResultItem{
				{FilePath: "alpha.go", Symbol: "Alpha", Kind: "function", StartLine: 1, EndLine: 12},
				{FilePath: "beta.go", Symbol: "Beta", Kind: "type", StartLine: 35, EndLine: 45},
				{FilePath: "gamma.go", Symbol: "Gamma", Kind: "method", StartLine: 45, EndLine: 55},
				{FilePath: "delta.go", Symbol: "Delta", Kind: "variable", StartLine: 75, EndLine: 100},
			},
		},
		{
			name: "disjoint ranges with matching identities",
			actual: []searchResultItem{
				{FilePath: "alpha.go", Symbol: "Alpha", Kind: "function", StartLine: 1, EndLine: 9},
				{FilePath: "beta.go", Symbol: "Beta", Kind: "type", StartLine: 41, EndLine: 49},
				{FilePath: "gamma.go", Symbol: "Gamma", Kind: "method", StartLine: 61, EndLine: 69},
				{FilePath: "delta.go", Symbol: "Delta", Kind: "variable", StartLine: 81, EndLine: 90},
			},
			wantErr: "matched rows: 0/4 (required at least 2)",
		},
		{
			name: "fifty percent boundary",
			actual: []searchResultItem{
				baseline[0], baseline[1],
				{FilePath: "new-one.go", Symbol: "NewOne", Kind: "function", StartLine: 1, EndLine: 1},
				{FilePath: "new-two.go", Symbol: "NewTwo", Kind: "type", StartLine: 2, EndLine: 2},
			},
		},
		{
			name:   "fewer results at threshold",
			actual: baseline[:2],
		},
		{
			name:    "fewer results below threshold",
			actual:  baseline[:1],
			wantErr: "matched rows: 1/4 (required at least 2)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := compareLangSnapshot(expected, tt.actual)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("compareLangSnapshot() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("compareLangSnapshot() error = %v, want error containing %q", err, tt.wantErr)
			}
			if !strings.Contains(err.Error(), "expected result count: 4") || !strings.Contains(err.Error(), "actual result count:") {
				t.Fatalf("compareLangSnapshot() error does not report result counts: %v", err)
			}
			if !strings.Contains(err.Error(), "unmatched expected rows") || !strings.Contains(err.Error(), "unmatched actual rows") {
				t.Fatalf("compareLangSnapshot() error does not report unmatched rows: %v", err)
			}
		})
	}
}

func TestCompareLangSnapshotPreservesDuplicateRows(t *testing.T) {
	t.Parallel()

	const expected = "" +
		"results: 4\n" +
		"alpha.go:10-20  Alpha (function)\n" +
		"alpha.go:30-40  Alpha (function)\n" +
		"alpha.go:50-60  Alpha (function)\n" +
		"alpha.go:70-80  Alpha (function)\n\n"

	t.Run("one actual row matches at most one expected row", func(t *testing.T) {
		actual := []searchResultItem{
			{FilePath: "alpha.go", Symbol: "Alpha", Kind: "function", StartLine: 1, EndLine: 100},
		}
		err := compareLangSnapshot(expected, actual)
		if err == nil || !strings.Contains(err.Error(), "matched rows: 1/4 (required at least 2)") {
			t.Fatalf("compareLangSnapshot() error = %v, want one matched row", err)
		}
	})

	t.Run("separate actual rows satisfy the threshold", func(t *testing.T) {
		actual := []searchResultItem{
			{FilePath: "alpha.go", Symbol: "Alpha", Kind: "function", StartLine: 15, EndLine: 35},
			{FilePath: "alpha.go", Symbol: "Alpha", Kind: "function", StartLine: 55, EndLine: 75},
		}
		if err := compareLangSnapshot(expected, actual); err != nil {
			t.Fatalf("compareLangSnapshot() error = %v", err)
		}
	})
}

func TestCompareLangSnapshotRoundsRequiredRowsUp(t *testing.T) {
	t.Parallel()

	const expected = "" +
		"results: 3\n" +
		"alpha.go:10-20  Alpha (function)\n" +
		"beta.go:30-40  Beta (type)\n" +
		"gamma.go:50-60  Gamma (method)\n\n"

	err := compareLangSnapshot(expected, []searchResultItem{
		{FilePath: "alpha.go", Symbol: "Alpha", Kind: "function", StartLine: 10, EndLine: 20},
	})
	if err == nil || !strings.Contains(err.Error(), "matched rows: 1/3 (required at least 2)") {
		t.Fatalf("compareLangSnapshot() error = %v, want ceiling threshold of two rows", err)
	}

	if err := compareLangSnapshot(expected, []searchResultItem{
		{FilePath: "alpha.go", Symbol: "Alpha", Kind: "function", StartLine: 10, EndLine: 20},
		{FilePath: "beta.go", Symbol: "Beta", Kind: "type", StartLine: 30, EndLine: 40},
	}); err != nil {
		t.Fatalf("compareLangSnapshot() error at ceiling threshold = %v", err)
	}
}

func TestCompareLangSnapshotUsesMaximumRangeMatching(t *testing.T) {
	t.Parallel()

	const expected = "" +
		"results: 4\n" +
		"alpha.go:10-20  Alpha (function)\n" +
		"alpha.go:20-30  Alpha (function)\n" +
		"beta.go:40-50  Beta (type)\n" +
		"gamma.go:60-70  Gamma (method)\n\n"
	actual := []searchResultItem{
		{FilePath: "alpha.go", Symbol: "Alpha", Kind: "function", StartLine: 10, EndLine: 30},
		{FilePath: "alpha.go", Symbol: "Alpha", Kind: "function", StartLine: 10, EndLine: 19},
	}

	if err := compareLangSnapshot(expected, actual); err != nil {
		t.Fatalf("compareLangSnapshot() error = %v", err)
	}
}

func TestCompareLangSnapshotParsesPathsWithConsecutiveSpaces(t *testing.T) {
	t.Parallel()

	const expected = "results: 1\ndir/two  spaces.go:10-20  Alpha (function)\n"
	actual := []searchResultItem{
		{FilePath: "dir/two  spaces.go", Symbol: "Alpha", Kind: "function", StartLine: 12, EndLine: 18},
	}

	if err := compareLangSnapshot(expected, actual); err != nil {
		t.Fatalf("compareLangSnapshot() error = %v", err)
	}
}

func TestParseCommittedLangSnapshots(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir(langSnapshotDirectory)
	if err != nil {
		t.Fatalf("failed to read snapshot directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "TestLang_") {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			snapshot, err := os.ReadFile(filepath.Join(langSnapshotDirectory, entry.Name()))
			if err != nil {
				t.Fatalf("failed to read snapshot: %v", err)
			}
			if _, err := parseLangSnapshot(string(snapshot)); err != nil {
				t.Fatalf("parseLangSnapshot() error = %v", err)
			}
		})
	}
}

func TestCompareLangSnapshotRejectsMalformedSnapshots(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		snapshot string
		wantErr  string
	}{
		{name: "empty", snapshot: "\n", wantErr: "snapshot is empty"},
		{name: "invalid header", snapshot: "result count: 1\nalpha.go:1-1  Alpha (function)\n", wantErr: "malformed snapshot header"},
		{name: "invalid count", snapshot: "results: many\n", wantErr: "malformed snapshot result count"},
		{name: "declared count mismatch", snapshot: "results: 2\nalpha.go:1-1  Alpha (function)\n", wantErr: "declares 2 results but contains 1"},
		{name: "invalid result", snapshot: "results: 1\nnot a result\n", wantErr: "malformed snapshot result line 2"},
		{name: "invalid line range", snapshot: "results: 1\nalpha.go:2-1  Alpha (function)\n", wantErr: "invalid line range 2-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := compareLangSnapshot(tt.snapshot, nil)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("compareLangSnapshot() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestCompareLangSnapshotRejectsInvalidActualResults(t *testing.T) {
	t.Parallel()

	const expected = "results: 1\nalpha.go:1-1  Alpha (function)\n\n"
	valid := searchResultItem{FilePath: "alpha.go", Symbol: "Alpha", Kind: "function", StartLine: 1, EndLine: 1}
	tests := []struct {
		name    string
		mutate  func(*searchResultItem)
		wantErr string
	}{
		{name: "missing file", mutate: func(result *searchResultItem) { result.FilePath = "" }, wantErr: "file path is empty"},
		{name: "missing symbol", mutate: func(result *searchResultItem) { result.Symbol = "" }, wantErr: "symbol is empty"},
		{name: "missing kind", mutate: func(result *searchResultItem) { result.Kind = "" }, wantErr: "kind is empty"},
		{name: "non-positive start line", mutate: func(result *searchResultItem) { result.StartLine = 0 }, wantErr: "start line must be positive"},
		{name: "reversed line range", mutate: func(result *searchResultItem) { result.EndLine = 0 }, wantErr: "is before start line"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := valid
			tt.mutate(&result)
			err := compareLangSnapshot(expected, []searchResultItem{result})
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("compareLangSnapshot() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}
