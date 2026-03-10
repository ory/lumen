package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/aeneasr/lumen/bench-swe/internal/analysis"
)

var (
	extractJSON       bool
	extractSearchOnly bool
)

var extractCmd = &cobra.Command{
	Use:   "extract <raw-jsonl-file>",
	Short: "Extract tool calls from a Claude CLI raw JSONL file",
	Long:  "Parses a *-raw.jsonl file and prints all tool calls in sequence, highlighting semantic_search calls with query and result preview.",
	Args:  cobra.ExactArgs(1),
	RunE:  runExtract,
}

func init() {
	extractCmd.Flags().BoolVar(&extractJSON, "json", false, "Output machine-readable JSON")
	extractCmd.Flags().BoolVar(&extractSearchOnly, "search-only", false, "Only show mcp__lumen__semantic_search calls")
}

func runExtract(_ *cobra.Command, args []string) error {
	rawPath := args[0]

	calls, err := analysis.ExtractToolCalls(rawPath)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", rawPath, err)
	}

	if extractSearchOnly {
		var filtered []analysis.ToolCall
		for _, c := range calls {
			if c.Name == "mcp__lumen__semantic_search" {
				filtered = append(filtered, c)
			}
		}
		calls = filtered
	}

	if extractJSON {
		return outputJSON(calls)
	}
	return outputText(calls)
}

func outputJSON(calls []analysis.ToolCall) error {
	searchCount := 0
	for _, c := range calls {
		if c.Name == "mcp__lumen__semantic_search" {
			searchCount++
		}
	}

	out := struct {
		TotalCalls  int                 `json:"total_calls"`
		SearchCalls int                 `json:"search_calls"`
		Calls       []analysis.ToolCall `json:"calls"`
	}{
		TotalCalls:  len(calls),
		SearchCalls: searchCount,
		Calls:       calls,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func outputText(calls []analysis.ToolCall) error {
	searchCount := 0

	for _, c := range calls {
		isSearch := c.Name == "mcp__lumen__semantic_search"
		if isSearch {
			searchCount++
		}

		fmt.Printf("[%d] %s", c.Index, c.Name)
		if c.ID != "" {
			fmt.Printf(" (%s)", c.ID)
		}
		fmt.Println()

		if isSearch {
			query, _ := c.Input["query"].(string)
			searchPath := c.SearchPath()
			fmt.Printf("     Query: %s\n", query)
			if searchPath != "" {
				fmt.Printf("     Path:  %s\n", searchPath)
			}
			if c.Result != "" {
				preview := c.Result
				if len(preview) > 500 {
					preview = preview[:500] + "..."
				}
				// Show first few lines
				lines := strings.SplitN(preview, "\n", 6)
				fmt.Println("     Results:")
				for _, line := range lines {
					fmt.Printf("       %s\n", line)
				}
			}
		}
		fmt.Println()
	}

	// Summary
	fmt.Println("---")
	fmt.Printf("Total tool calls: %d\n", len(calls))
	fmt.Printf("Semantic search calls: %d\n", searchCount)
	if len(calls) == 0 {
		fmt.Println("WARNING: No tool calls found. Check JSONL parsing logic!")
	}

	return nil
}
