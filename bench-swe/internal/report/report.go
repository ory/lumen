package report

import (
	"fmt"
	"math"
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
	toolCalls int  // total tool call count from raw JSONL
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

				rawPath := filepath.Join(cfg.ResultsDir, slug+"-raw.jsonl")
				if calls, err := analysis.ExtractToolCalls(rawPath); err == nil {
					tr.toolCalls = len(calls)
					if s == runner.WithLumen {
						for _, tc := range calls {
							if tc.Name == "mcp__lumen__semantic_search" {
								tr.lumenUsed = true
								break
							}
						}
					}
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
	Language      string
	BaselineWins  int
	WithLumenWins int
}

func generateSummary(cfg *Config, results []taskResult) error {
	path := filepath.Join(cfg.ResultsDir, "summary-report.md")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	data := summaryData{
		Date:        time.Now().UTC().Format("2006-01-02 15:04 UTC"),
		EmbedModel:  cfg.EmbedModel,
		ClaudeModel: cfg.ClaudeModel,
	}

	// Build dynamic table header/separator
	var header, sep strings.Builder
	header.WriteString("| Task | Lang |")
	sep.WriteString("|------|------|")
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
		fmt.Fprintf(&row, "| %s | %s |", t.ID, t.Language)
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
			Language:      lang,
			BaselineWins:  wins[runner.Baseline],
			WithLumenWins: wins[runner.WithLumen],
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
	defer func() { _ = f.Close() }()

	data := detailData{
		Date: time.Now().UTC().Format("2006-01-02 15:04 UTC"),
	}

	for _, t := range cfg.Tasks {
		dt := detailTask{
			ID:              t.ID,
			Language:        t.Language,
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

// totalTokens returns the total token count (input + output + cache read).
func totalTokens(m *metrics.Metrics) int64 {
	if m == nil {
		return 0
	}
	return m.InputTokens + m.OutputTokens + m.CacheRead
}

func ratingName(rank int) string {
	switch rank {
	case 3:
		return "Perfect"
	case 2:
		return "Good"
	case 1:
		return "Poor"
	default:
		return "None"
	}
}

func pctDiff(label, better, worse string, base, cmp float64) string {
	if base == 0 {
		return ""
	}
	pct := (base - cmp) / base * 100
	if pct >= 0 {
		return fmt.Sprintf("%-12s%.4g (baseline) vs %.4g (lumen) → %.0f%% %s\n", label, base, cmp, pct, better)
	}
	return fmt.Sprintf("%-12s%.4g (baseline) vs %.4g (lumen) → %.0f%% %s\n", label, base, cmp, -pct, worse)
}

// GenerateOverview prints and/or writes a compact overview table of results.
// Output is appended to the file (not overwritten) so that multiple runs
// can be compared side-by-side.
func GenerateOverview(cfg *Config) error {
	results := loadResults(cfg)

	var b strings.Builder

	// Run marker for distinguishing appended runs
	fmt.Fprintf(&b, "=== Run: %s | embed=%s | claude=%s ===\n\n",
		time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
		cfg.EmbedModel,
		cfg.ClaudeModel,
	)

	const (
		hdr = "%-30s  %-8s  %-10s  %-4s  %-8s  %-8s  %-7s  %-13s  %-11s  %-5s\n"
		row = "%-30s  %-8s  %-10s  %-4s  %-8s  %-8s  %-7s  %-13s  %-11s  %-5s\n"
		sep = 120
	)
	fmt.Fprintf(&b, hdr, "Task", "Lang", "Scenario", "Run", "Rating", "Cost", "Time", "Total Tokens", "Tool Calls", "Valid")
	fmt.Fprintf(&b, "%s\n", strings.Repeat("-", sep))

	var totalCost float64
	counts := map[judge.Rating]int{}
	invalidCount := 0

	// Comparison accumulators: worst baseline vs best with-lumen
	var (
		blMaxCost   float64
		blMaxTime   int64
		blMaxTokens int64
		blMaxCalls  int
		blWorstRank = 4 // higher than any real rank, so first match is always "worse"
		hasBaseline bool

		lmMinCost   = math.MaxFloat64
		lmMinTime   = int64(math.MaxInt64)
		lmMinTokens = int64(math.MaxInt64)
		lmMinCalls  = math.MaxInt
		lmBestRank  int
		hasLumen    bool
	)

	for _, t := range cfg.Tasks {
		for _, s := range cfg.Scenarios {
			runs := findResults(results, t.ID, s)
			for _, r := range runs {
				invalid := r.scenario == runner.WithLumen && !r.lumenUsed
				valid := "YES"
				if invalid {
					valid = "NO"
					invalidCount++
				}

				rating := "—"
				cost := "—"
				dur := "—"
				tokens := "—"
				calls := "—"
				runCol := fmt.Sprintf("%d", r.runIndex)

				if invalid {
					rating = "INVALID"
				} else if r.judge != nil {
					rating = string(r.judge.Rating)
					counts[r.judge.Rating]++
				}
				if r.metrics != nil {
					cost = fmt.Sprintf("$%.4f", r.metrics.CostUSD)
					dur = fmt.Sprintf("%.1fs", float64(r.metrics.DurationMS)/1000)
					totalCost += r.metrics.CostUSD
					tokens = fmt.Sprintf("%d", totalTokens(r.metrics))
				}
				if r.toolCalls > 0 {
					calls = fmt.Sprintf("%d", r.toolCalls)
				}

				fmt.Fprintf(&b, row, t.ID, t.Language, string(s), runCol, rating, cost, dur, tokens, calls, valid)

				// Accumulate comparison data
				if r.scenario == runner.Baseline && r.metrics != nil && !invalid {
					hasBaseline = true
					if r.metrics.CostUSD > blMaxCost {
						blMaxCost = r.metrics.CostUSD
					}
					if r.metrics.DurationMS > blMaxTime {
						blMaxTime = r.metrics.DurationMS
					}
					tk := totalTokens(r.metrics)
					if tk > blMaxTokens {
						blMaxTokens = tk
					}
					if r.toolCalls > blMaxCalls {
						blMaxCalls = r.toolCalls
					}
					if r.judge != nil {
						rank := ratingRank(r.judge.Rating)
						if rank < blWorstRank {
							blWorstRank = rank
						}
					}
				}
				if r.scenario == runner.WithLumen && r.lumenUsed && r.metrics != nil {
					hasLumen = true
					if r.metrics.CostUSD < lmMinCost {
						lmMinCost = r.metrics.CostUSD
					}
					if r.metrics.DurationMS < lmMinTime {
						lmMinTime = r.metrics.DurationMS
					}
					tk := totalTokens(r.metrics)
					if tk < lmMinTokens {
						lmMinTokens = tk
					}
					if r.toolCalls < lmMinCalls {
						lmMinCalls = r.toolCalls
					}
					if r.judge != nil {
						rank := ratingRank(r.judge.Rating)
						if rank > lmBestRank {
							lmBestRank = rank
						}
					}
				}
			}
		}
	}

	fmt.Fprintf(&b, "%s\n", strings.Repeat("-", sep))
	fmt.Fprintf(&b, "Perfect: %d  Good: %d  Poor: %d  Invalid: %d  Total cost: $%.4f\n",
		counts[judge.Perfect], counts[judge.Good], counts[judge.Poor], invalidCount, totalCost)

	// Comparison summary: worst baseline vs best with-lumen
	if hasBaseline && hasLumen {
		fmt.Fprintf(&b, "\n--- Comparison: worst baseline vs best with-lumen ---\n")
		fmt.Fprint(&b, pctDiff("Cost:", "cheaper", "more expensive",
			blMaxCost, lmMinCost))
		fmt.Fprint(&b, pctDiff("Time:", "faster", "slower",
			float64(blMaxTime)/1000, float64(lmMinTime)/1000))
		fmt.Fprint(&b, pctDiff("Tokens:", "fewer tokens", "more tokens",
			float64(blMaxTokens), float64(lmMinTokens)))
		fmt.Fprint(&b, pctDiff("Tool calls:", "fewer tool calls", "more tool calls",
			float64(blMaxCalls), float64(lmMinCalls)))
		fmt.Fprintf(&b, "%-12s%s (baseline) vs %s (lumen)\n",
			"Rating:", ratingName(blWorstRank), ratingName(lmBestRank))
	}

	fmt.Fprintln(&b) // trailing blank line between appended runs

	out := b.String()

	if cfg.Verbose {
		fmt.Print("\nOverview:\n\n")
		fmt.Print(out)
	}

	outPath := cfg.OutputPath
	if outPath == "" {
		outPath = "overview.txt"
	}
	f, err := os.OpenFile(outPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening overview file: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(out); err != nil {
		return fmt.Errorf("writing overview: %w", err)
	}
	fmt.Printf("  Overview: %s\n", outPath)

	return nil
}
