package analysis

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aeneasr/lumen/bench-swe/internal/runner"
	"github.com/aeneasr/lumen/bench-swe/internal/task"
)

// withLumenRawFiles returns all raw JSONL paths for the with-lumen scenario
// of a task, supporting both single-run (slug-raw.jsonl) and multi-run
// (slug-run1-raw.jsonl, slug-run2-raw.jsonl, ...) naming conventions.
func withLumenRawFiles(resultsDir, taskID string) []string {
	slug := taskID + "-" + string(runner.WithLumen)
	// Try single-run path first
	single := filepath.Join(resultsDir, slug+"-raw.jsonl")
	if _, err := os.Stat(single); err == nil {
		return []string{single}
	}
	// Glob multi-run paths
	matches, _ := filepath.Glob(filepath.Join(resultsDir, slug+"-run*-raw.jsonl"))
	return matches
}

type searchCall struct {
	Query   string `json:"query"`
	CWD     string `json:"cwd"`
	Results string // raw XML/text result
}

type langStats struct {
	Language      string
	TotalTasks    int
	TotalRuns     int
	InvalidRuns   int // with-lumen runs where mcp__lumen__semantic_search was never called
	TotalQueries  int
	FilesHit      int
	FilesMissed   int
	TotalResults  int
	NoiseResults  int
	MissedFiles   []string
}

func Analyze(resultsDir, tasksBaseDir string, tasks []task.Task) error {
	stats := map[string]*langStats{}

	for _, t := range tasks {
		if stats[t.Language] == nil {
			stats[t.Language] = &langStats{Language: t.Language}
		}
		ls := stats[t.Language]
		ls.TotalTasks++

		goldPatch, err := t.GoldPatch(tasksBaseDir)
		if err != nil {
			continue
		}
		goldFiles := extractPatchFiles(goldPatch)

		// Analyze with-lumen scenario across all runs (single or multi-run naming)
		rawPaths := withLumenRawFiles(resultsDir, t.ID)
		for _, rawPath := range rawPaths {
			ls.TotalRuns++

			calls, err := extractSearchCalls(rawPath)
			if err != nil {
				continue
			}

			if len(calls) == 0 {
				ls.InvalidRuns++
				continue
			}

			ls.TotalQueries += len(calls)

			// Check which gold files were found in search results
			for _, gf := range goldFiles {
				found := false
				for _, call := range calls {
					if strings.Contains(call.Results, gf) {
						found = true
						break
					}
				}
				if found {
					ls.FilesHit++
				} else {
					ls.FilesMissed++
					ls.MissedFiles = append(ls.MissedFiles, fmt.Sprintf("%s: %s", t.ID, gf))
				}
			}

			// Count results that don't reference gold files (noise)
			for _, call := range calls {
				resultFiles := extractResultFiles(call.Results)
				ls.TotalResults += len(resultFiles)
				for _, rf := range resultFiles {
					isGold := false
					for _, gf := range goldFiles {
						if strings.HasSuffix(rf, gf) || strings.HasSuffix(gf, rf) {
							isGold = true
							break
						}
					}
					if !isGold {
						ls.NoiseResults++
					}
				}
			}
		}
	}

	return writeAnalysis(resultsDir, stats)
}

func extractSearchCalls(rawPath string) ([]searchCall, error) {
	allCalls, err := ExtractToolCalls(rawPath)
	if err != nil {
		return nil, err
	}

	var calls []searchCall
	for _, tc := range allCalls {
		if tc.Name == "mcp__lumen__semantic_search" {
			query, _ := tc.Input["query"].(string)
			calls = append(calls, searchCall{Query: query, CWD: tc.SearchPath(), Results: tc.Result})
		}
	}
	return calls, nil
}

func extractPatchFiles(patch string) []string {
	var files []string
	for _, line := range strings.Split(patch, "\n") {
		if strings.HasPrefix(line, "+++ b/") {
			files = append(files, strings.TrimPrefix(line, "+++ b/"))
		}
	}
	return files
}

func extractResultFiles(results string) []string {
	var files []string
	// Parse XML-style results: <result:file filename="...">
	for _, line := range strings.Split(results, "\n") {
		if idx := strings.Index(line, `filename="`); idx >= 0 {
			rest := line[idx+len(`filename="`):]
			if end := strings.Index(rest, `"`); end >= 0 {
				files = append(files, rest[:end])
			}
		}
	}
	return files
}

func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func writeAnalysis(resultsDir string, stats map[string]*langStats) error {
	path := filepath.Join(resultsDir, "chunker-analysis.md")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := func(format string, args ...any) {
		fmt.Fprintf(f, format+"\n", args...)
	}

	w("# Chunker Analysis")
	w("")

	// Sort languages
	var langs []string
	for lang := range stats {
		langs = append(langs, lang)
	}
	sort.Strings(langs)

	for _, lang := range langs {
		ls := stats[lang]
		w("## %s", capitalizeFirst(lang))
		w("")

		totalFiles := ls.FilesHit + ls.FilesMissed
		hitRate := 0.0
		if totalFiles > 0 {
			hitRate = float64(ls.FilesHit) / float64(totalFiles) * 100
		}
		noiseRate := 0.0
		if ls.TotalResults > 0 {
			noiseRate = float64(ls.NoiseResults) / float64(ls.TotalResults) * 100
		}

		w("- **Tasks**: %d", ls.TotalTasks)
		w("- **Runs**: %d (%d valid, %d invalid — no lumen search used)", ls.TotalRuns, ls.TotalRuns-ls.InvalidRuns, ls.InvalidRuns)
		w("- **Queries**: %d", ls.TotalQueries)
		w("- **Hit Rate**: %.0f%% (%d/%d gold-patch files found)", hitRate, ls.FilesHit, totalFiles)
		w("- **Noise Rate**: %.0f%% (%d/%d results not in gold patch)", noiseRate, ls.NoiseResults, ls.TotalResults)
		w("")

		if len(ls.MissedFiles) > 0 {
			w("### Missed Files")
			w("")
			for _, mf := range ls.MissedFiles {
				w("- `%s`", mf)
			}
			w("")
			w("**Note**: Review these misses to identify general chunker improvement patterns.")
			w("Recommendations must apply to ANY codebase, not just these test repos.")
			w("")
		}
	}

	fmt.Printf("  Analysis: %s\n", path)
	return nil
}
