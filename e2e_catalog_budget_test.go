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

//go:build e2e

package main

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ory/lumen/internal/evalcase"
)

// budgetStats aggregates the rendered size and budget cuts of one catalog run.
type budgetStats struct {
	sizes                      []int
	capped, truncated, omitted int
	overCap                    int
}

func (s *budgetStats) observe(t *testing.T, capBytes int) func(string) {
	return func(text string) {
		s.sizes = append(s.sizes, len(text))
		if capBytes > 0 && len(text) > capBytes {
			s.overCap++
		}
		out, err := parseSearchOutput(text)
		if err != nil {
			t.Errorf("unparseable bounded output: %v", err)
			return
		}
		if out.Truncated > 0 || out.Omitted > 0 {
			s.capped++
		}
		s.truncated += out.Truncated
		s.omitted += out.Omitted
	}
}

func percentile(sorted []int, p int) int {
	if len(sorted) == 0 {
		return 0
	}
	return sorted[min(len(sorted)-1, len(sorted)*p/100)]
}

// TestE2E_Catalog_BM25Budget measures visible evidence under the output
// budget. Budgeting runs after ranking, so ranks cannot change; it can only
// shorten or omit content a judgment reads (contains, multistep requires).
// Flips against the unlimited run at the same limit are reported, not
// enforced; only the hard byte ceiling and parseability are asserted.
func TestE2E_Catalog_BM25Budget(t *testing.T) {
	t.Parallel()
	session, trap, _ := startBM25Server(t)
	open := func(string) *mcp.ClientSession { return session }
	configs := []struct {
		name     string
		extra    map[string]any
		capBytes int
		baseline string
	}{
		{"limit10-unlimited", map[string]any{"max_tokens": -1}, 0, ""},
		{"limit10-default", nil, 16000, "limit10-unlimited"},
		{"limit10-max_tokens1000", map[string]any{"max_tokens": 1000}, 4000, "limit10-unlimited"},
		{"limit20-unlimited", map[string]any{"limit": 20, "max_tokens": -1}, 0, ""},
		{"limit20-default", map[string]any{"limit": 20}, 16000, "limit20-unlimited"},
	}
	runs := map[string]map[string]evalcase.Outcome{}
	var b strings.Builder
	b.WriteString("# BM25 output budget (executable MCP)\n\n")
	b.WriteString("| config | supported pass | diagnostic pass | calls | p50 B | p90 B | max B | mean est. tokens | capped calls | truncated | omitted | over cap |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	var flips strings.Builder
	for _, c := range configs {
		var stats budgetStats
		outcomes := runCatalog(t, open, c.extra, false, stats.observe(t, c.capBytes))
		byID := map[string]evalcase.Outcome{}
		pass := map[string][2]int{}
		for _, o := range outcomes {
			byID[o.ID] = o
			p := pass[o.Category]
			p[1]++
			if o.Pass {
				p[0]++
			}
			pass[o.Category] = p
		}
		runs[c.name] = byID
		if stats.overCap > 0 {
			t.Errorf("%s: %d responses exceeded the %d-byte ceiling", c.name, stats.overCap, c.capBytes)
		}
		sorted := slices.Sorted(slices.Values(stats.sizes))
		total := 0
		for _, n := range sorted {
			total += n
		}
		mean := 0
		if len(sorted) > 0 {
			mean = total / len(sorted) / 4
		}
		fmt.Fprintf(&b, "| %s | %d/%d | %d/%d | %d | %d | %d | %d | %d | %d | %d | %d | %d |\n", c.name,
			pass[evalcase.Supported][0], pass[evalcase.Supported][1], pass[evalcase.Diagnostic][0], pass[evalcase.Diagnostic][1],
			len(sorted), percentile(sorted, 50), percentile(sorted, 90), percentile(sorted, 100), mean, stats.capped, stats.truncated, stats.omitted, stats.overCap)
		if c.baseline == "" {
			continue
		}
		base := runs[c.baseline]
		var lost, gained []string
		for id, o := range byID {
			if bo, ok := base[id]; ok && bo.Pass != o.Pass {
				entry := fmt.Sprintf("%s (%s): %s", id, o.Category, o.Detail)
				if bo.Pass {
					lost = append(lost, entry)
				} else {
					gained = append(gained, entry)
				}
			}
		}
		slices.Sort(lost)
		slices.Sort(gained)
		fmt.Fprintf(&flips, "\n## %s vs %s\n\npass→fail: %d, fail→pass: %d\n\n", c.name, c.baseline, len(lost), len(gained))
		for _, e := range lost {
			fmt.Fprintf(&flips, "- pass→fail %s\n", e)
		}
		for _, e := range gained {
			fmt.Fprintf(&flips, "- fail→pass %s\n", e)
		}
	}
	b.WriteString("\nBytes are rendered UTF-8 text; estimated tokens = bytes / 4.\n")
	b.WriteString(flips.String())
	writeCatalogReport(t, "bm25-budget", b.String())
	t.Log("\n" + b.String())
	trap.assertUnused(t)
}
