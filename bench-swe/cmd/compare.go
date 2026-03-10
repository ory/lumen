package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/aeneasr/lumen/bench-swe/internal/judge"
	"github.com/aeneasr/lumen/bench-swe/internal/metrics"
)

var compareCmd = &cobra.Command{
	Use:   "compare <dir1> <dir2>",
	Short: "Compare two benchmark result directories",
	Args:  cobra.ExactArgs(2),
	RunE:  runCompare,
}

func runCompare(_ *cobra.Command, args []string) error {
	dir1, dir2 := args[0], args[1]

	// Discover all task-scenario pairs from metrics files
	pairs1 := discoverPairs(dir1)
	pairs2 := discoverPairs(dir2)

	// Merge keys
	allKeys := map[string]bool{}
	for k := range pairs1 {
		allKeys[k] = true
	}
	for k := range pairs2 {
		allKeys[k] = true
	}

	fmt.Printf("# Benchmark Comparison\n\n")
	fmt.Printf("- **Run 1**: `%s`\n", filepath.Base(dir1))
	fmt.Printf("- **Run 2**: `%s`\n\n", filepath.Base(dir2))

	fmt.Println("## Per-Task Deltas")
	fmt.Println()
	fmt.Println("| Task-Scenario | R1 Rating | R2 Rating | Cost Delta | Time Delta | Token Delta |")
	fmt.Println("|---------------|-----------|-----------|------------|------------|-------------|")

	var improvements, regressions, unchanged int
	var totalCostDelta float64
	var totalTimeDelta int64
	var totalTokenDelta int64
	compared := 0

	for key := range allKeys {
		m1 := pairs1[key]
		m2 := pairs2[key]

		r1Rating, r2Rating := "—", "—"
		j1Path := filepath.Join(dir1, key+"-judge.json")
		if j1, err := judge.LoadResult(j1Path); err == nil {
			r1Rating = string(j1.Rating)
		}
		j2Path := filepath.Join(dir2, key+"-judge.json")
		if j2, err := judge.LoadResult(j2Path); err == nil {
			r2Rating = string(j2.Rating)
		}

		costDelta, timeDelta, tokenDelta := "—", "—", "—"
		if m1 != nil && m2 != nil {
			cd := m2.CostUSD - m1.CostUSD
			td := m2.DurationMS - m1.DurationMS
			tkd := (m2.InputTokens + m2.OutputTokens) - (m1.InputTokens + m1.OutputTokens)

			totalCostDelta += cd
			totalTimeDelta += td
			totalTokenDelta += tkd
			compared++

			costDelta = formatDelta(cd, m1.CostUSD, "$%.4f")
			timeDelta = formatDeltaTime(td, m1.DurationMS)
			tokenDelta = formatDeltaInt(tkd, m1.InputTokens+m1.OutputTokens)
		}

		// Track quality changes
		if r1Rating != "—" && r2Rating != "—" {
			r1Rank := ratingToRank(r1Rating)
			r2Rank := ratingToRank(r2Rating)
			if r2Rank > r1Rank {
				improvements++
			} else if r2Rank < r1Rank {
				regressions++
			} else {
				unchanged++
			}
		}

		fmt.Printf("| %s | %s | %s | %s | %s | %s |\n",
			key, r1Rating, r2Rating, costDelta, timeDelta, tokenDelta)
	}

	fmt.Println()
	fmt.Println("## Summary")
	fmt.Println()
	fmt.Printf("- **Quality**: %d improvements, %d regressions, %d unchanged\n",
		improvements, regressions, unchanged)
	if compared > 0 {
		avgCost := totalCostDelta / float64(compared)
		avgTime := float64(totalTimeDelta) / float64(compared) / 1000
		avgTokens := totalTokenDelta / int64(compared)
		fmt.Printf("- **Avg Cost Delta**: $%.4f per task\n", avgCost)
		fmt.Printf("- **Avg Time Delta**: %.1fs per task\n", avgTime)
		fmt.Printf("- **Avg Token Delta**: %d per task\n", avgTokens)
	}

	return nil
}

func discoverPairs(dir string) map[string]*metrics.Metrics {
	result := map[string]*metrics.Metrics{}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "-metrics.json") {
			key := strings.TrimSuffix(e.Name(), "-metrics.json")
			m, err := metrics.LoadFromFile(filepath.Join(dir, e.Name()))
			if err == nil {
				result[key] = m
			}
		}
	}
	return result
}

func formatDelta(delta, base float64, fmtStr string) string {
	sign := "+"
	if delta < 0 {
		sign = ""
	}
	pct := 0.0
	if base > 0 {
		pct = delta / base * 100
	}
	return fmt.Sprintf("%s"+fmtStr+" (%+.0f%%)", sign, delta, pct)
}

func formatDeltaTime(deltaMS, baseMS int64) string {
	delta := float64(deltaMS) / 1000
	base := float64(baseMS) / 1000
	sign := "+"
	if delta < 0 {
		sign = ""
	}
	pct := 0.0
	if base > 0 {
		pct = delta / base * 100
	}
	return fmt.Sprintf("%s%.1fs (%+.0f%%)", sign, delta, pct)
}

func formatDeltaInt(delta, base int64) string {
	sign := "+"
	if delta < 0 {
		sign = ""
	}
	pct := 0.0
	if base > 0 {
		pct = float64(delta) / float64(base) * 100
	}
	return fmt.Sprintf("%s%d (%+.0f%%)", sign, delta, pct)
}

func ratingToRank(s string) int {
	switch s {
	case "Perfect":
		return 3
	case "Good":
		return 2
	case "Poor":
		return 1
	default:
		return 0
	}
}
