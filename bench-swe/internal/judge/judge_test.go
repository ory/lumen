package judge

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseJudgeOutput(t *testing.T) {
	tests := []struct {
		name            string
		output          string
		wantRating      Rating
		wantFiles       bool
		wantLogic       bool
		wantExplanation string
		wantErr         bool
	}{
		{
			name:            "valid JSON Perfect",
			output:          "{\"rating\":\"Perfect\",\"files_correct\":true,\"logic_equivalent\":true}\nThe candidate patch is identical to the gold patch.",
			wantRating:      Perfect,
			wantFiles:       true,
			wantLogic:       true,
			wantExplanation: "The candidate patch is identical to the gold patch.",
		},
		{
			name:            "valid JSON Good",
			output:          "{\"rating\":\"Good\",\"files_correct\":true,\"logic_equivalent\":false}\nDifferent approach but correct.",
			wantRating:      Good,
			wantFiles:       true,
			wantLogic:       false,
			wantExplanation: "Different approach but correct.",
		},
		{
			name:            "valid JSON Poor",
			output:          "{\"rating\":\"Poor\",\"files_correct\":false,\"logic_equivalent\":false}\nDoes not fix the issue.",
			wantRating:      Poor,
			wantFiles:       false,
			wantLogic:       false,
			wantExplanation: "Does not fix the issue.",
		},
		{
			name:       "fallback first word PERFECT",
			output:     "Perfect: the patch fixes everything correctly.",
			wantRating: Perfect,
		},
		{
			name:       "fallback first word GOOD",
			output:     "Good: a valid approach to fixing the bug.",
			wantRating: Good,
		},
		{
			name:       "fallback keyword mid-sentence is not matched",
			output:     "The patch is PERFECT and fixes everything.",
			wantRating: Poor,
		},
		{
			name:       "invalid JSON no keyword",
			output:     "This does not work at all, completely wrong.",
			wantRating: Poor,
		},
		{
			name:       "empty output falls back to Poor",
			output:     "  ",
			wantRating: Poor,
		},
		{
			name:       "JSON wrapped in markdown fences",
			output:     "```json\n{\"rating\":\"Perfect\",\"files_correct\":true,\"logic_equivalent\":true}\n```",
			wantRating: Perfect,
			wantFiles:  true,
			wantLogic:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseJudgeOutput(tt.output)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Rating != tt.wantRating {
				t.Errorf("Rating = %q, want %q", result.Rating, tt.wantRating)
			}
			if tt.wantFiles && !result.FilesCorrect {
				t.Error("FilesCorrect = false, want true")
			}
			if tt.wantLogic && !result.LogicEquivalent {
				t.Error("LogicEquivalent = false, want true")
			}
			if tt.wantExplanation != "" && result.Explanation != tt.wantExplanation {
				t.Errorf("Explanation = %q, want %q", result.Explanation, tt.wantExplanation)
			}
		})
	}
}

func TestParseJudgeOutput_SingleLineNoExplanation(t *testing.T) {
	result, err := parseJudgeOutput(`{"rating":"Good","files_correct":true,"logic_equivalent":false}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Explanation != "" {
		t.Errorf("Explanation = %q, want empty", result.Explanation)
	}
}

func TestSaveAndLoadResult(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "judge.json")
	mdPath := filepath.Join(dir, "judge.md")

	original := &JudgeResult{
		Rating:          Perfect,
		FilesCorrect:    true,
		LogicEquivalent: true,
		Explanation:     "Identical to gold patch.",
	}

	if err := SaveResult(original, jsonPath, mdPath); err != nil {
		t.Fatalf("SaveResult: %v", err)
	}

	// Load JSON
	loaded, err := LoadResult(jsonPath)
	if err != nil {
		t.Fatalf("LoadResult: %v", err)
	}

	if loaded.Rating != original.Rating {
		t.Errorf("Rating = %q, want %q", loaded.Rating, original.Rating)
	}
	if loaded.FilesCorrect != original.FilesCorrect {
		t.Errorf("FilesCorrect = %v, want %v", loaded.FilesCorrect, original.FilesCorrect)
	}
	if loaded.LogicEquivalent != original.LogicEquivalent {
		t.Errorf("LogicEquivalent = %v, want %v", loaded.LogicEquivalent, original.LogicEquivalent)
	}
	// Explanation has json:"-" so it won't survive round-trip
	if loaded.Explanation != "" {
		t.Errorf("Explanation = %q, want empty (json:\"-\" tag)", loaded.Explanation)
	}

	// Verify .md file contains rating and explanation
	mdData, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("reading md: %v", err)
	}
	mdContent := string(mdData)
	if !strings.Contains(mdContent, "Perfect") {
		t.Error("md file does not contain rating")
	}
	if !strings.Contains(mdContent, "Identical to gold patch.") {
		t.Error("md file does not contain explanation")
	}
}

func TestJudge_EmptyPatch(t *testing.T) {
	result, err := Judge(context.Background(), "title", "body", "gold patch", "", "baseline")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Rating != Poor {
		t.Errorf("Rating = %q, want %q", result.Rating, Poor)
	}
	if result.Explanation != "No patch produced." {
		t.Errorf("Explanation = %q, want %q", result.Explanation, "No patch produced.")
	}
}

func TestJudge_WhitespacePatch(t *testing.T) {
	result, err := Judge(context.Background(), "title", "body", "gold patch", "   \n\t  ", "baseline")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Rating != Poor {
		t.Errorf("Rating = %q, want %q", result.Rating, Poor)
	}
}

func TestLoadResult_MissingFile(t *testing.T) {
	_, err := LoadResult("/nonexistent/judge.json")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoadResult_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadResult(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSaveResult_JSONDoesNotContainExplanation(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "judge.json")
	mdPath := filepath.Join(dir, "judge.md")

	original := &JudgeResult{
		Rating:      Good,
		Explanation: "Some explanation text",
	}

	if err := SaveResult(original, jsonPath, mdPath); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["explanation"]; ok {
		t.Error("JSON file should not contain explanation field (json:\"-\" tag)")
	}
}
