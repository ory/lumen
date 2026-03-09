package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// identRe matches camelCase, PascalCase, and snake_case identifiers of 4+ chars.
var identRe = regexp.MustCompile(`[A-Za-z][A-Za-z0-9_]{3,}`)

type Task struct {
	ID             string   `json:"id"`
	Language       string   `json:"language"`
	Difficulty     string   `json:"difficulty"`
	Repo           string   `json:"repo"`
	BaseCommit     string   `json:"base_commit"`
	FixCommit      string   `json:"fix_commit"`
	IssueURL       string   `json:"issue_url"`
	IssueTitle     string   `json:"issue_title"`
	IssueBody      string   `json:"issue_body"`
	GoldPatchFile  string   `json:"gold_patch_file"`
	ExpectedFiles  []string `json:"expected_files"`
	SetupCommands  []string `json:"setup_commands"`
	TestCommand    string   `json:"test_command"`
	TimeoutSeconds int      `json:"timeout_s"`
}

func (t *Task) applyDefaults() {
	if t.TimeoutSeconds == 0 {
		t.TimeoutSeconds = 900
	}
}

func (t Task) Validate(baseDir string) error {
	if t.ID == "" {
		return fmt.Errorf("task missing id")
	}
	if t.Repo == "" {
		return fmt.Errorf("task %s: missing repo", t.ID)
	}
	if t.BaseCommit == "" {
		return fmt.Errorf("task %s: missing base_commit", t.ID)
	}
	if t.IssueBody == "" {
		return fmt.Errorf("task %s: missing issue_body", t.ID)
	}
	if t.GoldPatchFile == "" {
		return fmt.Errorf("task %s: missing gold_patch_file", t.ID)
	}
	patchPath := filepath.Join(baseDir, t.GoldPatchFile)
	if _, err := os.Stat(patchPath); err != nil {
		return fmt.Errorf("task %s: gold patch not found at %s", t.ID, patchPath)
	}
	return nil
}

func (t Task) GoldPatch(baseDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(baseDir, t.GoldPatchFile))
	if err != nil {
		return "", fmt.Errorf("reading gold patch for %s: %w", t.ID, err)
	}
	return string(data), nil
}

// GrepScore returns a greppability score in [0,1] and the identifiers from
// the gold patch that appear verbatim in the issue title or body.
// A score near 1 means Claude can locate the fix by grepping without using
// semantic search, making the task a poor lumen benchmark.
func (t Task) GrepScore(patchContent string) (float64, []string) {
	ids := patchIdentifiers(patchContent)
	if len(ids) == 0 {
		return 0, nil
	}
	lower := strings.ToLower(t.IssueTitle + " " + t.IssueBody)
	var leaked []string
	for _, id := range ids {
		if strings.Contains(lower, strings.ToLower(id)) {
			leaked = append(leaked, id)
		}
	}
	return float64(len(leaked)) / float64(len(ids)), leaked
}

// patchIdentifiers extracts file basenames and function/type names from a
// unified diff by parsing diff headers and hunk context lines.
func patchIdentifiers(patch string) []string {
	var ids []string
	seen := map[string]bool{}
	add := func(s string) {
		if s != "" && !seen[s] {
			ids = append(ids, s)
			seen[s] = true
		}
	}
	for _, line := range strings.Split(patch, "\n") {
		// "diff --git a/path/to/file.go b/path/to/file.go"
		if strings.HasPrefix(line, "diff --git ") {
			if parts := strings.Fields(line); len(parts) >= 3 {
				add(filepath.Base(strings.TrimPrefix(parts[2], "a/")))
			}
			continue
		}
		// "@@ -N,N +N,N @@ func FooBar(...)" — context after second @@
		if strings.HasPrefix(line, "@@ ") {
			if i := strings.Index(line[3:], "@@"); i >= 0 {
				ctx := strings.TrimSpace(line[3+i+2:])
				for _, tok := range identRe.FindAllString(ctx, -1) {
					add(tok)
				}
			}
		}
	}
	return ids
}

func LoadTasks(tasksDir string, langs []string, difficulty string) ([]Task, error) {
	var tasks []Task

	langSet := make(map[string]bool, len(langs))
	for _, l := range langs {
		langSet[l] = true
	}

	err := filepath.Walk(tasksDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		var t Task
		if err := json.Unmarshal(data, &t); err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}
		t.applyDefaults()

		if len(langSet) > 0 && !langSet[t.Language] {
			return nil
		}
		if difficulty != "" && t.Difficulty != difficulty {
			return nil
		}

		tasks = append(tasks, t)
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks found in %s (langs=%v, difficulty=%q)", tasksDir, langs, difficulty)
	}
	return tasks, nil
}
