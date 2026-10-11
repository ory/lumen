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

// Package evalcase loads the checked-in retrieval case catalog under
// testdata/retrieval/cases and judges search results against it. The same
// catalog drives the offline BM25 tests and the executable E2E suites, so
// relevance judgments exist exactly once.
//
// Judgments are source-grounded: a target names a file plus content that must
// appear inside a returned span (and optionally a symbol or line). They never
// depend on how a retrieval mode splits or merges chunks.
package evalcase

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// Categories.
const (
	// Supported cases must meet MaxRank (and related/distractor expectations).
	Supported = "supported"
	// Diagnostic cases are measured and reported but never fail a run: they
	// document vocabulary mismatches and unsupported inference honestly.
	Diagnostic = "diagnostic"
)

// Default rank budgets by case kind.
const (
	ControlRank  = 3
	TaskRank     = 5
	LanguageRank = 10
)

// Target identifies source that a result must cover. File is relative to the
// corpus root using forward slashes. At least one of Symbol, Contains or Line
// must be set so a judgment never reduces to "anything in this file".
type Target struct {
	File     string   `json:"file"`
	Symbol   string   `json:"symbol,omitempty"`
	Contains []string `json:"contains,omitempty"`
	Line     int      `json:"line,omitempty"`
}

// Step is one query of a multistep exploration. Requires lists identifiers
// that must have been present in content retrieved by the preceding step: the
// follow-up query is only legitimate if an agent could have read them.
type Step struct {
	Query    string   `json:"query"`
	Path     string   `json:"path,omitempty"`
	MaxRank  int      `json:"max_rank,omitempty"`
	Requires []string `json:"requires,omitempty"`
	Targets  []Target `json:"targets"`
}

// Case is one catalog entry.
type Case struct {
	ID       string `json:"id"`
	Corpus   string `json:"corpus"`
	Language string `json:"language"`
	// Kind is the task family: exact, control, task, language, vocabulary,
	// multistep, structural or syntax.
	Kind     string `json:"kind"`
	Category string `json:"category"`
	Origin   string `json:"origin"`
	Query    string `json:"query,omitempty"`
	Path     string `json:"path,omitempty"`
	MaxRank  int    `json:"max_rank,omitempty"`
	// Expect "none" asserts that the query returns no direct results.
	Expect      string   `json:"expect,omitempty"`
	Targets     []Target `json:"targets,omitempty"`
	Distractors []Target `json:"distractors,omitempty"`
	// Related targets must appear as one-hop related context (not direct hits).
	Related []Target `json:"related,omitempty"`
	Steps   []Step   `json:"steps,omitempty"`
	Note    string   `json:"note,omitempty"`
}

// Result is a retrieval-mode-neutral search result.
type Result struct {
	File      string // corpus-relative, forward slashes
	Symbol    string
	Kind      string
	StartLine int
	EndLine   int
	Content   string
	Related   bool
}

// Load reads every *.json file under dir, validates and returns the cases
// sorted by id.
func Load(dir string) ([]Case, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	var all []Case
	ids := map[string]string{}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var cases []Case
		dec := json.NewDecoder(strings.NewReader(string(data)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&cases); err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
		for _, c := range cases {
			if prev, ok := ids[c.ID]; ok {
				return nil, fmt.Errorf("%s: duplicate case id %q (also in %s)", filepath.Base(f), c.ID, prev)
			}
			ids[c.ID] = filepath.Base(f)
			if err := c.validate(); err != nil {
				return nil, fmt.Errorf("%s: case %q: %w", filepath.Base(f), c.ID, err)
			}
			all = append(all, c.withDefaults())
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	return all, nil
}

func (c Case) validate() error {
	if c.ID == "" || c.Corpus == "" || c.Language == "" || c.Kind == "" {
		return errors.New("id, corpus, language and kind are required")
	}
	if c.Category != Supported && c.Category != Diagnostic {
		return fmt.Errorf("category must be %q or %q", Supported, Diagnostic)
	}
	if len(c.Steps) > 0 {
		if c.Query != "" || len(c.Targets) > 0 {
			return errors.New("multistep cases put queries and targets in steps")
		}
		for i, s := range c.Steps {
			if s.Query == "" || len(s.Targets) == 0 {
				return fmt.Errorf("step %d needs a query and targets", i+1)
			}
			if i == 0 && len(s.Requires) > 0 {
				return errors.New("the first step cannot require prior results")
			}
			if err := validTargets(s.Targets); err != nil {
				return err
			}
		}
		return nil
	}
	if c.Query == "" {
		return errors.New("query is required")
	}
	if c.Expect == "none" {
		if len(c.Targets) > 0 {
			return errors.New(`expect "none" cannot have targets`)
		}
		return nil
	}
	if len(c.Targets) == 0 {
		return errors.New("at least one target is required")
	}
	for _, ts := range [][]Target{c.Targets, c.Distractors, c.Related} {
		if err := validTargets(ts); err != nil {
			return err
		}
	}
	return nil
}

func validTargets(ts []Target) error {
	for _, t := range ts {
		if t.File == "" {
			return errors.New("target file is required")
		}
		if t.Symbol == "" && len(t.Contains) == 0 && t.Line == 0 {
			return fmt.Errorf("target %s needs a symbol, contains or line", t.File)
		}
	}
	return nil
}

func defaultRank(kind string) int {
	switch kind {
	case "exact", "control", "syntax":
		return ControlRank
	case "language":
		return LanguageRank
	default:
		return TaskRank
	}
}

func (c Case) withDefaults() Case {
	if c.MaxRank == 0 {
		c.MaxRank = defaultRank(c.Kind)
	}
	for i := range c.Steps {
		if c.Steps[i].MaxRank == 0 {
			c.Steps[i].MaxRank = TaskRank
		}
	}
	return c
}

// Matches reports whether r covers t.
func (t Target) Matches(r Result) bool {
	if r.File != t.File {
		return false
	}
	if t.Symbol != "" && r.Symbol != t.Symbol && !strings.HasSuffix(r.Symbol, "."+t.Symbol) {
		return false
	}
	if t.Line > 0 && (t.Line < r.StartLine || t.Line > r.EndLine) {
		return false
	}
	for _, s := range t.Contains {
		if !strings.Contains(r.Content, s) {
			return false
		}
	}
	return true
}

// Outcome is the judgment for one case or step.
type Outcome struct {
	ID       string
	Language string
	Kind     string
	Category string
	Query    string
	// Rank is the 1-based direct rank of the first target, 0 when missing.
	Rank    int
	MaxRank int
	// DistractorRank is the best rank of any distractor ranked above the target.
	DistractorRank  int
	RelatedExpected int
	RelatedFound    int
	// RequiresMissing lists identifiers a multistep query used that the previous
	// step never surfaced.
	RequiresMissing []string
	Pass            bool
	Detail          string
}

// Judge evaluates a single-query case.
func Judge(c Case, results []Result) Outcome {
	o := Outcome{ID: c.ID, Language: c.Language, Kind: c.Kind, Category: c.Category, Query: c.Query, MaxRank: c.MaxRank}
	direct := directResults(results)
	if c.Expect == "none" {
		o.Pass = len(direct) == 0
		o.Rank = 1
		if !o.Pass {
			o.Rank = 0
			o.Detail = fmt.Sprintf("expected no direct results, got %d", len(direct))
		}
		return o
	}
	o.Rank = rankOf(c.Targets, direct)
	for i, r := range direct {
		if o.Rank > 0 && i+1 >= o.Rank {
			break
		}
		if slices.ContainsFunc(c.Distractors, func(t Target) bool { return t.Matches(r) }) {
			o.DistractorRank = i + 1
			break
		}
	}
	o.RelatedExpected = len(c.Related)
	for _, t := range c.Related {
		if slices.ContainsFunc(results, func(r Result) bool { return r.Related && t.Matches(r) }) {
			o.RelatedFound++
		}
	}
	o.Pass = o.Rank > 0 && o.Rank <= c.MaxRank && o.DistractorRank == 0 && o.RelatedFound == o.RelatedExpected
	o.Detail = describe(o, direct)
	return o
}

// JudgeStep evaluates one step of a multistep case. prior is everything the
// previous step returned (direct and related).
func JudgeStep(c Case, i int, prior, results []Result) Outcome {
	s := c.Steps[i]
	o := Outcome{ID: fmt.Sprintf("%s#%d", c.ID, i+1), Language: c.Language, Kind: c.Kind, Category: c.Category, Query: s.Query, MaxRank: s.MaxRank}
	for _, req := range s.Requires {
		if !slices.ContainsFunc(prior, func(r Result) bool { return strings.Contains(r.Content, req) }) {
			o.RequiresMissing = append(o.RequiresMissing, req)
		}
	}
	direct := directResults(results)
	o.Rank = rankOf(s.Targets, direct)
	o.Pass = o.Rank > 0 && o.Rank <= s.MaxRank && len(o.RequiresMissing) == 0
	o.Detail = describe(o, direct)
	return o
}

func directResults(results []Result) []Result {
	var direct []Result
	for _, r := range results {
		if !r.Related {
			direct = append(direct, r)
		}
	}
	return direct
}

func rankOf(targets []Target, direct []Result) int {
	for i, r := range direct {
		if slices.ContainsFunc(targets, func(t Target) bool { return t.Matches(r) }) {
			return i + 1
		}
	}
	return 0
}

func describe(o Outcome, direct []Result) string {
	var b strings.Builder
	switch {
	case o.Rank == 0:
		b.WriteString("target missing")
	case o.Rank > o.MaxRank:
		fmt.Fprintf(&b, "target rank %d > %d", o.Rank, o.MaxRank)
	}
	if o.DistractorRank > 0 {
		fmt.Fprintf(&b, "; distractor above target at rank %d", o.DistractorRank)
	}
	if o.RelatedFound < o.RelatedExpected {
		fmt.Fprintf(&b, "; related %d/%d", o.RelatedFound, o.RelatedExpected)
	}
	if len(o.RequiresMissing) > 0 {
		fmt.Fprintf(&b, "; prior step never surfaced %v", o.RequiresMissing)
	}
	if b.Len() == 0 {
		return ""
	}
	b.WriteString("; top:")
	for i, r := range direct[:min(5, len(direct))] {
		fmt.Fprintf(&b, " %d=%s:%d-%d %s", i+1, r.File, r.StartLine, r.EndLine, r.Symbol)
	}
	return strings.TrimPrefix(b.String(), "; ")
}
