package judge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aeneasr/lumen/bench-swe/internal/agent"
	"github.com/aeneasr/lumen/bench-swe/internal/metrics"
)

type Rating string

const (
	Poor    Rating = "Poor"
	Good    Rating = "Good"
	Perfect Rating = "Perfect"
)

type JudgeResult struct {
	Rating          Rating `json:"rating"`
	FilesCorrect    bool   `json:"files_correct"`
	LogicEquivalent bool   `json:"logic_equivalent"`
	Explanation     string `json:"-"`
}

func Judge(ctx context.Context, issueTitle, issueBody, goldPatch, candidatePatch, scenario string) (*JudgeResult, error) {
	return JudgeWithAgent(ctx, agent.Config{Provider: "claude", Transport: "cli", Auth: "subscription", Model: "claude-sonnet-4-6"}, issueTitle, issueBody, goldPatch, candidatePatch, scenario)
}

func JudgeWithAgent(ctx context.Context, cfg agent.Config, issueTitle, issueBody, goldPatch, candidatePatch, scenario string) (*JudgeResult, error) {
	if strings.TrimSpace(candidatePatch) == "" {
		return &JudgeResult{
			Rating:      Poor,
			Explanation: "No patch produced.",
		}, nil
	}

	prompt := fmt.Sprintf(`You are judging a code fix against the known correct human-written patch.

Issue: %s

%s

=== Gold patch (correct human fix) ===
%s

=== Candidate patch (%s scenario) ===
%s

Rate the candidate as one of:
- **Perfect**: Fixes the issue correctly. Same or equivalent logic to gold patch.
- **Good**: Addresses the issue correctly but with a different valid approach.
- **Poor**: Wrong, incomplete, does not compile, or does not fix the issue.

On the FIRST line, respond with EXACTLY this JSON (no markdown fencing):
{"rating": "Poor|Good|Perfect", "files_correct": true|false, "logic_equivalent": true|false}

Then write 2-3 sentences explaining your reasoning.`, issueTitle, issueBody, goldPatch, scenario, candidatePatch)

	dir, err := os.MkdirTemp("", "bench-judge-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	cmd, cleanup, err := agent.Command(ctx, agent.Request{Config: cfg, CWD: dir, ReadOnly: true, Prompt: prompt + "\nDo not call tools; judge only the supplied patches."})
	if err != nil {
		return nil, err
	}
	defer cleanup()
	rawPath := filepath.Join(dir, "judge.jsonl")
	raw, err := os.Create(rawPath)
	if err != nil {
		return nil, err
	}
	cmd.Stdout = raw
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()
	_ = raw.Close()
	if err != nil {
		return nil, fmt.Errorf("running %s judge: %w: %s", cfg.Provider, err, strings.TrimSpace(stderr.String()))
	}
	answer, err := metrics.ExtractAnswer(rawPath)
	if err != nil {
		return nil, err
	}
	return parseJudgeOutput(answer)
}

func parseJudgeOutput(output string) (*JudgeResult, error) {
	lines := strings.SplitN(strings.TrimSpace(output), "\n", 2)
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty judge output")
	}

	var result JudgeResult
	// Strip markdown fences that Claude often wraps JSON in
	cleaned := strings.TrimSpace(output)
	if strings.HasPrefix(cleaned, "```") {
		// Remove opening fence (```json or ```)
		if idx := strings.Index(cleaned, "\n"); idx >= 0 {
			cleaned = cleaned[idx+1:]
		}
		// Remove closing fence
		if idx := strings.LastIndex(cleaned, "```"); idx >= 0 {
			cleaned = cleaned[:idx]
		}
		cleaned = strings.TrimSpace(cleaned)
		// Re-split after stripping fences
		lines = strings.SplitN(cleaned, "\n", 2)
	}
	jsonLine := strings.TrimSpace(lines[0])
	if err := json.Unmarshal([]byte(jsonLine), &result); err != nil {
		// Try to extract rating from first word of output
		firstWord := strings.ToUpper(strings.SplitN(strings.TrimSpace(output), " ", 2)[0])
		// Also strip punctuation/colon from first word
		firstWord = strings.TrimRight(firstWord, ":.,;!")
		switch firstWord {
		case "PERFECT":
			result.Rating = Perfect
		case "GOOD":
			result.Rating = Good
		default:
			result.Rating = Poor
		}
	}

	if len(lines) > 1 {
		result.Explanation = strings.TrimSpace(lines[1])
	}

	return &result, nil
}

func SaveResult(result *JudgeResult, jsonPath, mdPath string) error {
	// Save JSON
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	if err := os.WriteFile(jsonPath, data, 0o644); err != nil {
		return err
	}

	// Save explanation markdown
	md := fmt.Sprintf("## Rating: %s\n\n%s\n", result.Rating, result.Explanation)
	return os.WriteFile(mdPath, []byte(md), 0o644)
}

func LoadResult(jsonPath string) (*JudgeResult, error) {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, err
	}
	var r JudgeResult
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}
