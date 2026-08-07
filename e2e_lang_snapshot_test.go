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

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

var langSnapshotLocationPattern = regexp.MustCompile(`^(.+):(\d+)-(\d+)$`)

type langResultIdentity struct {
	filePath string
	symbol   string
	kind     string
}

func (i langResultIdentity) String() string {
	return fmt.Sprintf("%s  %s (%s)", i.filePath, i.symbol, i.kind)
}

type parsedLangSnapshot struct {
	declaredCount int
	identities    map[langResultIdentity]struct{}
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

	identities := make(map[langResultIdentity]struct{}, declaredCount)
	for i, rawLine := range lines[1:] {
		identity, err := parseLangSnapshotResult(strings.TrimSuffix(rawLine, "\r"))
		if err != nil {
			return parsedLangSnapshot{}, fmt.Errorf("malformed snapshot result line %d: %w", i+2, err)
		}
		identities[identity] = struct{}{}
	}

	return parsedLangSnapshot{declaredCount: declaredCount, identities: identities}, nil
}

func parseLangSnapshotResult(line string) (langResultIdentity, error) {
	parts := strings.SplitN(line, "  ", 2)
	if len(parts) != 2 {
		return langResultIdentity{}, fmt.Errorf("expected location and identity separated by two spaces: %q", line)
	}

	location := langSnapshotLocationPattern.FindStringSubmatch(parts[0])
	if location == nil {
		return langResultIdentity{}, fmt.Errorf("invalid location %q", parts[0])
	}
	startLine, _ := strconv.Atoi(location[2])
	endLine, _ := strconv.Atoi(location[3])
	if startLine <= 0 || endLine < startLine {
		return langResultIdentity{}, fmt.Errorf("invalid line range %d-%d", startLine, endLine)
	}

	description := parts[1]
	kindStart := strings.LastIndex(description, " (")
	if kindStart < 1 || !strings.HasSuffix(description, ")") {
		return langResultIdentity{}, fmt.Errorf("invalid symbol and kind %q", description)
	}

	identity := langResultIdentity{
		filePath: location[1],
		symbol:   description[:kindStart],
		kind:     description[kindStart+2 : len(description)-1],
	}
	if strings.TrimSpace(identity.filePath) == "" || strings.TrimSpace(identity.symbol) == "" || strings.TrimSpace(identity.kind) == "" {
		return langResultIdentity{}, fmt.Errorf("file, symbol, and kind must be non-empty: %q", line)
	}

	return identity, nil
}

func compareLangSnapshot(snapshot string, actual []searchResultItem) error {
	expected, err := parseLangSnapshot(snapshot)
	if err != nil {
		return err
	}

	actualIdentities := make(map[langResultIdentity]struct{}, len(actual))
	for i, result := range actual {
		if err := validateLangResult(result); err != nil {
			return fmt.Errorf("invalid actual result %d: %w", i+1, err)
		}
		actualIdentities[langResultIdentity{
			filePath: result.FilePath,
			symbol:   result.Symbol,
			kind:     result.Kind,
		}] = struct{}{}
	}

	overlap := 0
	for identity := range expected.identities {
		if _, ok := actualIdentities[identity]; ok {
			overlap++
		}
	}
	requiredOverlap := (len(expected.identities) + 1) / 2
	countMatches := len(actual) == expected.declaredCount
	if countMatches && overlap >= requiredOverlap {
		return nil
	}

	missing := identityDifference(expected.identities, actualIdentities)
	unexpected := identityDifference(actualIdentities, expected.identities)

	var message strings.Builder
	message.WriteString("language snapshot comparison failed:\n")
	if !countMatches {
		fmt.Fprintf(&message, "actual result count: got %d, want %d\n", len(actual), expected.declaredCount)
	}
	fmt.Fprintf(&message, "identity overlap: %d/%d (required at least %d)\n", overlap, len(expected.identities), requiredOverlap)
	writeIdentityList(&message, "missing identities", missing)
	writeIdentityList(&message, "unexpected identities", unexpected)
	return fmt.Errorf("%s", strings.TrimSuffix(message.String(), "\n"))
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

func identityDifference(left, right map[langResultIdentity]struct{}) []string {
	difference := make([]string, 0)
	for identity := range left {
		if _, ok := right[identity]; !ok {
			difference = append(difference, identity.String())
		}
	}
	slices.Sort(difference)
	return difference
}

func writeIdentityList(message *strings.Builder, label string, identities []string) {
	fmt.Fprintf(message, "%s (%d):\n", label, len(identities))
	for _, identity := range identities {
		fmt.Fprintf(message, "  - %s\n", identity)
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
			name: "line range drift",
			actual: []searchResultItem{
				{FilePath: "alpha.go", Symbol: "Alpha", Kind: "function", StartLine: 1, EndLine: 2},
				{FilePath: "beta.go", Symbol: "Beta", Kind: "type", StartLine: 3, EndLine: 4},
				{FilePath: "gamma.go", Symbol: "Gamma", Kind: "method", StartLine: 5, EndLine: 6},
				{FilePath: "delta.go", Symbol: "Delta", Kind: "variable", StartLine: 7, EndLine: 8},
			},
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
			name: "below threshold",
			actual: []searchResultItem{
				baseline[0],
				{FilePath: "new-one.go", Symbol: "NewOne", Kind: "function", StartLine: 1, EndLine: 1},
				{FilePath: "new-two.go", Symbol: "NewTwo", Kind: "type", StartLine: 2, EndLine: 2},
				{FilePath: "new-three.go", Symbol: "NewThree", Kind: "method", StartLine: 3, EndLine: 3},
			},
			wantErr: "identity overlap: 1/4 (required at least 2)",
		},
		{
			name:    "count mismatch",
			actual:  baseline[:3],
			wantErr: "actual result count: got 3, want 4",
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
			if !strings.Contains(err.Error(), "missing identities") || !strings.Contains(err.Error(), "unexpected identities") {
				t.Fatalf("compareLangSnapshot() error does not report identity differences: %v", err)
			}
		})
	}
}

func TestCompareLangSnapshotNormalizesDuplicateIdentities(t *testing.T) {
	t.Parallel()

	const expected = "" +
		"results: 4\n" +
		"alpha.go:10-20  Alpha (function)\n" +
		"alpha.go:15-25  Alpha (function)\n" +
		"beta.go:30-40  Beta (type)\n" +
		"gamma.go:50-60  Gamma (method)\n\n"
	actual := []searchResultItem{
		{FilePath: "alpha.go", Symbol: "Alpha", Kind: "function", StartLine: 1, EndLine: 2},
		{FilePath: "alpha.go", Symbol: "Alpha", Kind: "function", StartLine: 3, EndLine: 4},
		{FilePath: "beta.go", Symbol: "Beta", Kind: "type", StartLine: 5, EndLine: 6},
		{FilePath: "new.go", Symbol: "New", Kind: "variable", StartLine: 7, EndLine: 8},
	}

	if err := compareLangSnapshot(expected, actual); err != nil {
		t.Fatalf("compareLangSnapshot() error = %v", err)
	}
}

func TestParseCommittedLangSnapshots(t *testing.T) {
	t.Parallel()

	snapshotDirectory := filepath.Join("testdata", "snapshots")
	entries, err := os.ReadDir(snapshotDirectory)
	if err != nil {
		t.Fatalf("failed to read snapshot directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "TestLang_") {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			snapshot, err := os.ReadFile(filepath.Join(snapshotDirectory, entry.Name()))
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
