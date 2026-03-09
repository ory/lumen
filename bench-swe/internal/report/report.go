package report

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aeneasr/lumen/bench-swe/internal/analysis"
	"github.com/aeneasr/lumen/bench-swe/internal/judge"
	"github.com/aeneasr/lumen/bench-swe/internal/metrics"
	"github.com/aeneasr/lumen/bench-swe/internal/runner"
	"github.com/aeneasr/lumen/bench-swe/internal/task"
)

type Config struct {
	ResultsDir  string
	EmbedModel  string
	ClaudeModel string
	Tasks       []task.Task
	Scenarios   []runner.Scenario
	Runs        int
	Verbose     bool
	OutputPath  string
}

type taskResult struct {
	task      task.Task
	scenario  runner.Scenario
	runIndex  int
	metrics   *metrics.Metrics
	judge     *judge.JudgeResult
	patch     string
	lumenUsed bool // true if mcp__lumen__semantic_search was called at least once
}

func Generate(cfg *Config) error {
	results := loadResults(cfg)

	if err := generateSummary(cfg, results); err != nil {
		return fmt.Errorf("summary report: %w", err)
	}
	if err := generateDetail(cfg, results); err != nil {
		return fmt.Errorf("detail report: %w", err)
	}
	return nil
}

func loadResults(cfg *Config) []taskResult {
	totalRuns := cfg.Runs
	if totalRuns < 1 {
		totalRuns = 1
	}

	var results []taskResult
	for _, t := range cfg.Tasks {
		for _, s := range cfg.Scenarios {
			for run := 1; run <= totalRuns; run++ {
				slug := runner.Slug(t.ID, s, run, totalRuns)
				var tr taskResult
				tr.task = t
				tr.scenario = s
				tr.runIndex = run

				metricsPath := filepath.Join(cfg.ResultsDir, slug+"-metrics.json")
				tr.metrics, _ = metrics.LoadFromFile(metricsPath)

				judgePath := filepath.Join(cfg.ResultsDir, slug+"-judge.json")
				tr.judge, _ = judge.LoadResult(judgePath)

				patchPath := filepath.Join(cfg.ResultsDir, slug+"-patch.diff")
				if data, err := os.ReadFile(patchPath); err == nil {
					tr.patch = string(data)
				}

				if s == runner.WithLumen {
					rawPath := filepath.Join(cfg.ResultsDir, slug+"-raw.jsonl")
					tr.lumenUsed, _ = analysis.HasLumenSearch(rawPath)
				}

				results = append(results, tr)
			}
		}
	}
	return results
}

// Template data types for summary report.

type summaryData struct {
	Date               string
	EmbedModel         string
	ClaudeModel        string
	ResultsTableHeader string
	ResultsTableSep    string
	Rows               []summaryRow
	ScenarioAggs       []scenarioAgg
	LangAggs           []langAgg
}

type summaryRow struct {
	ResultsRow string
}

type scenarioAgg struct {
	Name      string
	Perfect   int
	Good      int
	Poor      int
	AvgCost   string
	AvgTime   string
	AvgTokens string
}

type langAgg struct {
	Language     string
	BaselineWins int
	WithLumenWins  int
}

func generateSummary(cfg *Config, results []taskResult) error {
	path := filepath.Join(cfg.ResultsDir, "summary-report.md")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	data := summaryData{
		Date:        time.Now().UTC().Format("2006-01-02 15:04 UTC"),
		EmbedModel:  cfg.EmbedModel,
		ClaudeModel: cfg.ClaudeModel,
	}

	// Build dynamic table header/separator
	var header, sep strings.Builder
	header.WriteString("| Task | Lang | Diff |")
	sep.WriteString("|------|------|------|")
	for _, s := range cfg.Scenarios {
		fmt.Fprintf(&header, " %s Rating |", s)
		sep.WriteString("------------|")
	}
	for _, s := range cfg.Scenarios {
		fmt.Fprintf(&header, " %s Cost |", s)
		sep.WriteString("----------|")
	}
	for _, s := range cfg.Scenarios {
		fmt.Fprintf(&header, " %s Time |", s)
		sep.WriteString("----------|")
	}
	data.ResultsTableHeader = header.String()
	data.ResultsTableSep = sep.String()

	// Build rows (use median metrics and best rating when runs > 1)
	multiRun := cfg.Runs > 1
	for _, t := range cfg.Tasks {
		var row strings.Builder
		fmt.Fprintf(&row, "| %s | %s | %s |", t.ID, t.Language, t.Difficulty)
		for _, s := range cfg.Scenarios {
			runs := findResults(results, t.ID, s)
			if s == runner.WithLumen && allRunsInvalid(runs) {
				row.WriteString(" INVALID |")
				continue
			}
			jr := bestRating(validRuns(runs))
			if jr != nil {
				fmt.Fprintf(&row, " %s |", jr.Rating)
			} else {
				row.WriteString(" — |")
			}
		}
		for _, s := range cfg.Scenarios {
			runs := findResults(results, t.ID, s)
			m := medianMetrics(runs)
			if m != nil {
				label := fmt.Sprintf("$%.4f", m.CostUSD)
				if multiRun {
					label += "†"
				}
				fmt.Fprintf(&row, " %s |", label)
			} else {
				row.WriteString(" — |")
			}
		}
		for _, s := range cfg.Scenarios {
			runs := findResults(results, t.ID, s)
			m := medianMetrics(runs)
			if m != nil {
				label := fmt.Sprintf("%.1fs", float64(m.DurationMS)/1000)
				if multiRun {
					label += "†"
				}
				fmt.Fprintf(&row, " %s |", label)
			} else {
				row.WriteString(" — |")
			}
		}
		data.Rows = append(data.Rows, summaryRow{ResultsRow: row.String()})
	}

	// Scenario aggregates (use median per task, then average across tasks)
	for _, s := range cfg.Scenarios {
		agg := scenarioAgg{Name: string(s)}
		var totalCost float64
		var totalTime int64
		var totalTokens int64
		count := 0
		for _, t := range cfg.Tasks {
			runs := findResults(results, t.ID, s)
			jr := bestRating(validRuns(runs))
			if jr != nil {
				switch jr.Rating {
				case judge.Perfect:
					agg.Perfect++
				case judge.Good:
					agg.Good++
				default:
					agg.Poor++
				}
			}
			m := medianMetrics(validRuns(runs))
			if m != nil {
				totalCost += m.CostUSD
				totalTime += m.DurationMS
				totalTokens += m.InputTokens + m.OutputTokens
				count++
			}
		}
		if count > 0 {
			agg.AvgCost = fmt.Sprintf("$%.4f", totalCost/float64(count))
			agg.AvgTime = fmt.Sprintf("%.1fs", float64(totalTime)/float64(count)/1000)
			agg.AvgTokens = fmt.Sprintf("%d", totalTokens/int64(count))
		} else {
			agg.AvgCost = "—"
			agg.AvgTime = "—"
			agg.AvgTokens = "—"
		}
		data.ScenarioAggs = append(data.ScenarioAggs, agg)
	}

	// Language aggregates
	for _, lang := range uniqueLanguages(cfg.Tasks) {
		wins := map[runner.Scenario]int{}
		for _, t := range cfg.Tasks {
			if t.Language != lang {
				continue
			}
			// Find best rating across scenarios (using best across runs)
			bestR := judge.Poor
			for _, s := range cfg.Scenarios {
				runs := findResults(results, t.ID, s)
				jr := bestRating(runs)
				if jr != nil && ratingRank(jr.Rating) > ratingRank(bestR) {
					bestR = jr.Rating
				}
			}
			// Count how many scenarios achieved the best rating
			var bestScenarios []runner.Scenario
			for _, s := range cfg.Scenarios {
				runs := findResults(results, t.ID, s)
				jr := bestRating(runs)
				if jr != nil && ratingRank(jr.Rating) == ratingRank(bestR) {
					bestScenarios = append(bestScenarios, s)
				}
			}
			// Only count as win if single winner (ties don't count)
			if len(bestScenarios) == 1 {
				wins[bestScenarios[0]]++
			}
		}
		data.LangAggs = append(data.LangAggs, langAgg{
			Language:     lang,
			BaselineWins: wins[runner.Baseline],
			WithLumenWins:  wins[runner.WithLumen],
		})
	}

	if err := summaryTmpl.Execute(f, data); err != nil {
		return err
	}

	fmt.Printf("  Summary: %s\n", path)
	return nil
}

// Template data types for detail report.

type detailData struct {
	Date  string
	Tasks []detailTask
}

type detailTask struct {
	ID              string
	Language        string
	Difficulty      string
	IssueTitle      string
	IssueBodyQuoted string
	MetricsRows     []detailMetricsRow
	ScenarioDetails []detailScenario
}

type detailMetricsRow struct {
	Scenario     string
	Duration     string
	InputTokens  string
	CacheRead    string
	OutputTokens string
	Cost         string
}

type detailScenario struct {
	Scenario    string
	Rating      string
	Explanation string
	Patch       string
}

func generateDetail(cfg *Config, results []taskResult) error {
	path := filepath.Join(cfg.ResultsDir, "detail-report.md")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	data := detailData{
		Date: time.Now().UTC().Format("2006-01-02 15:04 UTC"),
	}

	for _, t := range cfg.Tasks {
		dt := detailTask{
			ID:              t.ID,
			Language:        t.Language,
			Difficulty:      t.Difficulty,
			IssueTitle:      t.IssueTitle,
			IssueBodyQuoted: strings.ReplaceAll(t.IssueBody, "\n", "\n> "),
		}

		for _, s := range cfg.Scenarios {
			runs := findResults(results, t.ID, s)
			for _, r := range runs {
				label := string(s)
				if cfg.Runs > 1 {
					label = fmt.Sprintf("%s run%d", s, r.runIndex)
				}
				row := detailMetricsRow{Scenario: label}
				if r.metrics != nil {
					m := r.metrics
					row.Duration = fmt.Sprintf("%.1fs", float64(m.DurationMS)/1000)
					row.InputTokens = fmt.Sprintf("%d", m.InputTokens)
					row.CacheRead = fmt.Sprintf("%d", m.CacheRead)
					row.OutputTokens = fmt.Sprintf("%d", m.OutputTokens)
					row.Cost = fmt.Sprintf("$%.4f", m.CostUSD)
				} else {
					row.Duration = "—"
					row.InputTokens = "—"
					row.CacheRead = "—"
					row.OutputTokens = "—"
					row.Cost = "—"
				}
				dt.MetricsRows = append(dt.MetricsRows, row)
			}
		}

		for _, s := range cfg.Scenarios {
			runs := findResults(results, t.ID, s)
			for _, r := range runs {
				label := string(s)
				if cfg.Runs > 1 {
					label = fmt.Sprintf("%s run%d", s, r.runIndex)
				}
				ds := detailScenario{Scenario: label}
				if r.scenario == runner.WithLumen && !r.lumenUsed {
					ds.Rating = "INVALID (lumen not used)"
				} else if r.judge != nil {
					ds.Rating = string(r.judge.Rating)
					ds.Explanation = r.judge.Explanation
				}
				ds.Patch = r.patch
				dt.ScenarioDetails = append(dt.ScenarioDetails, ds)
			}
		}

		data.Tasks = append(data.Tasks, dt)
	}

	if err := detailTmpl.Execute(f, data); err != nil {
		return err
	}

	fmt.Printf("  Detail:  %s\n", path)
	return nil
}

func findResult(results []taskResult, taskID string, s runner.Scenario) *taskResult {
	for i := range results {
		if results[i].task.ID == taskID && results[i].scenario == s {
			return &results[i]
		}
	}
	return nil
}

func findResults(results []taskResult, taskID string, s runner.Scenario) []taskResult {
	var out []taskResult
	for _, r := range results {
		if r.task.ID == taskID && r.scenario == s {
			out = append(out, r)
		}
	}
	return out
}

// medianMetrics returns the median metrics across multiple run results.
func medianMetrics(runs []taskResult) *metrics.Metrics {
	var ms []*metrics.Metrics
	for _, r := range runs {
		if r.metrics != nil {
			ms = append(ms, r.metrics)
		}
	}
	if len(ms) == 0 {
		return nil
	}
	if len(ms) == 1 {
		return ms[0]
	}

	// Sort by cost and take median
	sort.Slice(ms, func(i, j int) bool { return ms[i].CostUSD < ms[j].CostUSD })
	med := ms[len(ms)/2]
	return med
}

// allRunsInvalid returns true if all with-lumen runs failed to call semantic search.
func allRunsInvalid(runs []taskResult) bool {
	if len(runs) == 0 {
		return false
	}
	for _, r := range runs {
		if r.lumenUsed {
			return false
		}
	}
	return true
}

// validRuns filters out with-lumen runs where lumen was not used.
func validRuns(runs []taskResult) []taskResult {
	var out []taskResult
	for _, r := range runs {
		if r.scenario != runner.WithLumen || r.lumenUsed {
			out = append(out, r)
		}
	}
	return out
}

// bestRating returns the best judge rating across multiple run results.
func bestRating(runs []taskResult) *judge.JudgeResult {
	var best *judge.JudgeResult
	for _, r := range runs {
		if r.judge == nil {
			continue
		}
		if best == nil || ratingRank(r.judge.Rating) > ratingRank(best.Rating) {
			best = r.judge
		}
	}
	return best
}

func uniqueLanguages(tasks []task.Task) []string {
	seen := map[string]bool{}
	var langs []string
	for _, t := range tasks {
		if !seen[t.Language] {
			seen[t.Language] = true
			langs = append(langs, t.Language)
		}
	}
	return langs
}

func ratingRank(r judge.Rating) int {
	switch r {
	case judge.Perfect:
		return 3
	case judge.Good:
		return 2
	case judge.Poor:
		return 1
	default:
		return 0
	}
}

// GenerateOverview prints and/or writes a compact overview table of results.
// It runs only if cfg.Verbose is true or cfg.OutputPath is non-empty.
func GenerateOverview(cfg *Config) error {
	results := loadResults(cfg)

	var b strings.Builder
	const hdr = "%-30s  %-8s  %-6s  %-10s  %-8s  %-8s  %-7s\n"
	const row = "%-30s  %-8s  %-6s  %-10s  %-8s  %-8s  %-7s\n"
	fmt.Fprintf(&b, hdr, "Task", "Lang", "Diff", "Scenario", "Rating", "Cost", "Time")
	fmt.Fprintf(&b, "%s\n", strings.Repeat("-", 90))

	var totalCost float64
	counts := map[judge.Rating]int{}

	for _, t := range cfg.Tasks {
		for _, s := range cfg.Scenarios {
			runs := findResults(results, t.ID, s)
			jr := bestRating(runs)
			m := medianMetrics(runs)

			rating := "—"
			cost := "—"
			dur := "—"

			if jr != nil {
				rating = string(jr.Rating)
				counts[jr.Rating]++
			}
			if m != nil {
				cost = fmt.Sprintf("$%.4f", m.CostUSD)
				dur = fmt.Sprintf("%.1fs", float64(m.DurationMS)/1000)
				totalCost += m.CostUSD
			}

			fmt.Fprintf(&b, row, t.ID, t.Language, t.Difficulty, string(s), rating, cost, dur)
		}
	}

	fmt.Fprintf(&b, "%s\n", strings.Repeat("-", 90))
	fmt.Fprintf(&b, "Perfect: %d  Good: %d  Poor: %d  Total cost: $%.4f\n",
		counts[judge.Perfect], counts[judge.Good], counts[judge.Poor], totalCost)

	out := b.String()

	if cfg.Verbose {
		fmt.Print("\nOverview:\n\n")
		fmt.Print(out)
	}

	outPath := cfg.OutputPath
	if outPath == "" {
		outPath = "overview.txt"
	}
	if err := os.WriteFile(outPath, []byte(out), 0o644); err != nil {
		return fmt.Errorf("writing overview: %w", err)
	}
	fmt.Printf("  Overview: %s\n", outPath)

	return nil
}
