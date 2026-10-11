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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTargetMatches(t *testing.T) {
	r := Result{File: "a/b.go", Symbol: "DB.Close", StartLine: 10, EndLine: 20, Content: "func (db *DB) Close() error { return db.head.Close() }"}
	for name, tc := range map[string]struct {
		target Target
		want   bool
	}{
		"symbol":            {Target{File: "a/b.go", Symbol: "DB.Close"}, true},
		"symbol suffix":     {Target{File: "a/b.go", Symbol: "Close"}, true},
		"symbol partial":    {Target{File: "a/b.go", Symbol: "B.Close"}, false},
		"other file":        {Target{File: "a/c.go", Symbol: "DB.Close"}, false},
		"line inside":       {Target{File: "a/b.go", Line: 15}, true},
		"line outside":      {Target{File: "a/b.go", Line: 21}, false},
		"contains all":      {Target{File: "a/b.go", Contains: []string{"db.head", "Close()"}}, true},
		"contains missing":  {Target{File: "a/b.go", Contains: []string{"db.head", "Rollback"}}, false},
		"symbol + contains": {Target{File: "a/b.go", Symbol: "DB.Close", Contains: []string{"nope"}}, false},
	} {
		if got := tc.target.Matches(r); got != tc.want {
			t.Errorf("%s: Matches = %v, want %v", name, got, tc.want)
		}
	}
}

func TestJudge(t *testing.T) {
	results := []Result{
		{File: "mock.go", Symbol: "MockQuerier.Close", Content: "x"},
		{File: "db.go", Symbol: "DB.Close", Content: "y"},
		{File: "helper.go", Symbol: "helper", Content: "z", Related: true},
	}
	c := Case{ID: "c", MaxRank: 3, Targets: []Target{{File: "db.go", Symbol: "DB.Close"}}}
	if o := Judge(c, results); !o.Pass || o.Rank != 2 {
		t.Fatalf("plain case = %+v", o)
	}
	c.Distractors = []Target{{File: "mock.go", Symbol: "Close"}}
	if o := Judge(c, results); o.Pass || o.DistractorRank != 1 {
		t.Fatalf("distractor above target must fail: %+v", o)
	}
	c.Distractors = nil
	c.Related = []Target{{File: "helper.go", Symbol: "helper"}}
	if o := Judge(c, results); !o.Pass || o.RelatedFound != 1 {
		t.Fatalf("related case = %+v", o)
	}
	// Related snippets never count as direct hits.
	c = Case{ID: "r", MaxRank: 3, Targets: []Target{{File: "helper.go", Symbol: "helper"}}}
	if o := Judge(c, results); o.Pass || o.Rank != 0 {
		t.Fatalf("related snippet counted as direct hit: %+v", o)
	}
	c.MaxRank = 1
	c.Targets = []Target{{File: "db.go", Symbol: "DB.Close"}}
	if o := Judge(c, results); o.Pass || !strings.Contains(o.Detail, "rank 2 > 1") {
		t.Fatalf("rank budget not enforced: %+v", o)
	}
	none := Case{ID: "n", Expect: "none"}
	if o := Judge(none, nil); !o.Pass {
		t.Fatalf("expect none with no results = %+v", o)
	}
	if o := Judge(none, results[2:]); !o.Pass {
		t.Fatalf("expect none ignores related context = %+v", o)
	}
	if o := Judge(none, results); o.Pass {
		t.Fatalf("expect none with results = %+v", o)
	}
}

func TestJudgeStepRequiresPriorEvidence(t *testing.T) {
	c := Case{ID: "m", Steps: []Step{
		{Query: "duplicate payment", MaxRank: 5, Targets: []Target{{File: "pay.ts", Symbol: "processPayment"}}},
		{Query: "reserveSlot", MaxRank: 5, Requires: []string{"reserveSlot"}, Targets: []Target{{File: "res.ts", Symbol: "reserveSlot"}}},
	}}
	hit := []Result{{File: "res.ts", Symbol: "reserveSlot", Content: "export function reserveSlot() {}"}}
	if o := JudgeStep(c, 1, []Result{{Content: "import { reserveSlot as claim }"}}, hit); !o.Pass || o.ID != "m#2" {
		t.Fatalf("step = %+v", o)
	}
	if o := JudgeStep(c, 1, []Result{{Content: "unrelated"}}, hit); o.Pass || len(o.RequiresMissing) != 1 {
		t.Fatalf("step without evidence must fail: %+v", o)
	}
}

func TestLoadValidates(t *testing.T) {
	for name, body := range map[string]string{
		"unknown field":    `[{"id":"a","corpus":"c","language":"go","kind":"task","category":"supported","query":"q","targets":[{"file":"f","symbol":"s"}],"bogus":1}]`,
		"bad category":     `[{"id":"a","corpus":"c","language":"go","kind":"task","category":"maybe","query":"q","targets":[{"file":"f","symbol":"s"}]}]`,
		"file-only target": `[{"id":"a","corpus":"c","language":"go","kind":"task","category":"supported","query":"q","targets":[{"file":"f"}]}]`,
		"no targets":       `[{"id":"a","corpus":"c","language":"go","kind":"task","category":"supported","query":"q"}]`,
		"duplicate":        `[{"id":"a","corpus":"c","language":"go","kind":"task","category":"supported","query":"q","targets":[{"file":"f","symbol":"s"}]},{"id":"a","corpus":"c","language":"go","kind":"task","category":"supported","query":"q","targets":[{"file":"f","symbol":"s"}]}]`,
	} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "x.json"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(dir); err == nil {
			t.Errorf("%s: Load accepted invalid catalog", name)
		}
	}
	dir := t.TempDir()
	body := `[{"id":"a","corpus":"c","language":"go","kind":"language","category":"supported","query":"q","targets":[{"file":"f","symbol":"s"}]}]`
	if err := os.WriteFile(filepath.Join(dir, "x.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cases, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cases[0].MaxRank != LanguageRank {
		t.Fatalf("default language rank = %d", cases[0].MaxRank)
	}
}

func TestSummarize(t *testing.T) {
	rows := Summarize([]Outcome{{Language: "go", Rank: 1, Pass: true}, {Language: "go", Rank: 4}, {Language: "ts"}}, func(o Outcome) string { return o.Language })
	all := rows[len(rows)-1]
	if all.N != 3 || all.Hit3 != 1 || all.Hit5 != 2 || all.Pass != 1 {
		t.Fatalf("all = %+v", all)
	}
	if got := all.MRR(); got < 0.416 || got > 0.417 {
		t.Fatalf("MRR = %v", got)
	}
}

// TestCatalogIsValid loads the checked-in catalog and verifies every target
// refers to an existing corpus file whose source contains the judged text.
// Judgments are frozen against source, independent of any retrieval output.
func TestCatalogIsValid(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	cases, err := Load(filepath.Join(root, "retrieval", "cases"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatal("empty catalog")
	}
	check := func(c Case, ts []Target) {
		for _, tg := range ts {
			data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(c.Corpus), filepath.FromSlash(tg.File)))
			if err != nil {
				t.Errorf("%s: %v", c.ID, err)
				continue
			}
			src := string(data)
			for _, s := range tg.Contains {
				if !strings.Contains(src, s) {
					t.Errorf("%s: %s does not contain %q", c.ID, tg.File, s)
				}
			}
			if n := strings.Count(src, "\n") + 1; tg.Line > n {
				t.Errorf("%s: %s has %d lines, target line %d", c.ID, tg.File, n, tg.Line)
			}
		}
	}
	for _, c := range cases {
		check(c, c.Targets)
		check(c, c.Distractors)
		check(c, c.Related)
		for _, s := range c.Steps {
			check(c, s.Targets)
		}
	}
}
