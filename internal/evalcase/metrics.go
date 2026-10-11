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

package evalcase

import (
	"fmt"
	"sort"
	"strings"
)

// Metrics aggregates outcomes. MRR uses the reciprocal direct rank of the
// first target (0 when missing within the returned results).
type Metrics struct {
	Group           string
	N               int
	Hit3            int
	Hit5            int
	Hit10           int
	ReciprocalRank  float64
	Pass            int
	RelatedExpected int
	RelatedFound    int
}

func (m *Metrics) add(o Outcome) {
	m.N++
	if o.Rank > 0 {
		if o.Rank <= 3 {
			m.Hit3++
		}
		if o.Rank <= 5 {
			m.Hit5++
		}
		if o.Rank <= 10 {
			m.Hit10++
		}
		m.ReciprocalRank += 1 / float64(o.Rank)
	}
	if o.Pass {
		m.Pass++
	}
	m.RelatedExpected += o.RelatedExpected
	m.RelatedFound += o.RelatedFound
}

// MRR is the mean reciprocal rank.
func (m Metrics) MRR() float64 {
	if m.N == 0 {
		return 0
	}
	return m.ReciprocalRank / float64(m.N)
}

// Summarize groups outcomes by key and adds an "all" row.
func Summarize(outcomes []Outcome, key func(Outcome) string) []Metrics {
	groups := map[string]*Metrics{}
	total := &Metrics{Group: "all"}
	for _, o := range outcomes {
		k := key(o)
		if groups[k] == nil {
			groups[k] = &Metrics{Group: k}
		}
		groups[k].add(o)
		total.add(o)
	}
	out := make([]Metrics, 0, len(groups)+1)
	for _, m := range groups {
		out = append(out, *m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Group < out[j].Group })
	return append(out, *total)
}

// Table renders metrics as a Markdown table.
func Table(title string, rows []Metrics) string {
	var b strings.Builder
	fmt.Fprintf(&b, "| %s | n | pass | hit@3 | hit@5 | hit@10 | MRR | related |\n|---|---|---|---|---|---|---|---|\n", title)
	for _, m := range rows {
		related := "-"
		if m.RelatedExpected > 0 {
			related = fmt.Sprintf("%d/%d", m.RelatedFound, m.RelatedExpected)
		}
		fmt.Fprintf(&b, "| %s | %d | %d | %s | %s | %s | %.3f | %s |\n", m.Group, m.N, m.Pass, ratio(m.Hit3, m.N), ratio(m.Hit5, m.N), ratio(m.Hit10, m.N), m.MRR(), related)
	}
	return b.String()
}

func ratio(a, n int) string {
	if n == 0 {
		return "-"
	}
	return fmt.Sprintf("%.2f", float64(a)/float64(n))
}

// Report renders every outcome plus per-language, per-kind and per-category
// summaries. Supported failures and diagnostic misses are listed explicitly.
func Report(mode string, outcomes []Outcome) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Retrieval catalog: %s\n\n", mode)
	b.WriteString(Table("category", Summarize(outcomes, func(o Outcome) string { return o.Category })))
	b.WriteString("\n")
	b.WriteString(Table("language", Summarize(outcomes, func(o Outcome) string { return o.Language })))
	b.WriteString("\n")
	b.WriteString(Table("kind", Summarize(outcomes, func(o Outcome) string { return o.Kind })))
	b.WriteString("\n| case | category | rank | max | pass | detail |\n|---|---|---|---|---|---|\n")
	for _, o := range outcomes {
		rank := "miss"
		if o.Rank > 0 {
			rank = fmt.Sprint(o.Rank)
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %d | %v | %s |\n", o.ID, o.Category, rank, o.MaxRank, o.Pass, strings.ReplaceAll(o.Detail, "|", "\\|"))
	}
	return b.String()
}
