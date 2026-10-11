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

// No e2e build tag: the result parser shared by the E2E suites is unit-tested
// without building the server or contacting an embedding service.

package main

import "testing"

func TestParseSearchOutputScoreForms(t *testing.T) {
	text := `Found 5 results [retrieval=bm25; direct scores are weighted lexical ranks, not cosine similarity] (indexed 3 files):

<result:file filename="a.go">
  <result:chunk line-start="1" line-end="3" symbol="Exact" kind="function" score="1000">
func Exact() {}
  </result:chunk>
</result:file>

<result:file filename="b.go">
  <result:chunk line-start="4" line-end="9" symbol="Dec" kind="method" score="12.3457">
body
  </result:chunk>
</result:file>

<result:file filename="c.go">
  <result:chunk line-start="2" line-end="2" symbol="Exp" kind="var" score="1.23457e+06">
x
  </result:chunk>
</result:file>

<result:file filename="d.go">
  <result:chunk line-start="2" line-end="2" symbol="Neg" kind="var" score="-0.25">
y
  </result:chunk>
</result:file>

<result:file filename="billing/reservation.ts">
  <result:chunk line-start="1" line-end="1" symbol="operator&lt;&lt;" kind="function" related-to="billing/payment.ts:processPayment" relation="direct_call">
export function reserveSlot() {}
  </result:chunk>
</result:file>
`
	out, err := parseSearchOutput(text)
	if err != nil {
		t.Fatal(err)
	}
	if out.Retrieval != "bm25" || !out.Reindexed || out.IndexedFiles != 3 {
		t.Fatalf("header = %+v", out)
	}
	want := []struct {
		symbol   string
		score    float32
		hasScore bool
	}{{"Exact", 1000, true}, {"Dec", 12.3457, true}, {"Exp", 1.23457e+06, true}, {"Neg", -0.25, true}, {"operator<<", 0, false}}
	if len(out.Results) != len(want) {
		t.Fatalf("parsed %d results, want %d: %+v", len(out.Results), len(want), out.Results)
	}
	for i, w := range want {
		r := out.Results[i]
		if r.Symbol != w.symbol || r.Score != w.score || r.HasScore != w.hasScore {
			t.Errorf("result %d = %+v, want %+v", i, r, w)
		}
	}
	rel := out.Results[4]
	if rel.RelatedTo != "billing/payment.ts:processPayment" || rel.Relation != "direct_call" || rel.Content != "export function reserveSlot() {}" {
		t.Errorf("related = %+v", rel)
	}
}

func TestParseSearchOutputSemanticAndEmpty(t *testing.T) {
	out, err := parseSearchOutput("Found 1 results:\n\n<result:file filename=\"x.go\">\n  <result:chunk line-start=\"1\" line-end=\"2\" symbol=\"X\" kind=\"function\" score=\"0.66\">\nfunc X() {}\n  </result:chunk>\n</result:file>\n")
	if err != nil {
		t.Fatal(err)
	}
	if out.Retrieval != "" || len(out.Results) != 1 || out.Results[0].Score != 0.66 || !out.Results[0].HasScore {
		t.Fatalf("semantic = %+v", out)
	}
	out, err = parseSearchOutput("No results found. (indexed 7 files)")
	if err != nil || len(out.Results) != 0 || out.IndexedFiles != 7 {
		t.Fatalf("empty = %+v, %v", out, err)
	}
	out, err = parseSearchOutput("Found 0 results [retrieval=bm25; direct scores are weighted lexical ranks, not cosine similarity]:\n")
	if err != nil || len(out.Results) != 0 || out.Retrieval != "bm25" {
		t.Fatalf("bm25 empty = %+v, %v", out, err)
	}
}

func TestParseSearchOutputRejectsMalformedChunks(t *testing.T) {
	for _, header := range []string{
		`  <result:chunk line-start="1" line-end="2" symbol="X" kind="function" score="NaN">`,
		`  <result:chunk line-start="1" line-end="2" symbol="X" kind="function" score="Inf">`,
		`  <result:chunk line-start="1" line-end="2" symbol="X" kind="function">`,
		`  <result:chunk line-start="1" line-end="2" symbol="X" kind="function" related-to="a">`,
	} {
		if _, err := parseSearchOutput("Found 1 results:\n<result:file filename=\"x.go\">\n" + header + "\n  </result:chunk>\n</result:file>\n"); err == nil {
			t.Errorf("accepted malformed header %s", header)
		}
	}
}

func TestParseSearchOutputBudget(t *testing.T) {
	text := `Found 2 results [retrieval=bm25; direct scores are weighted lexical ranks, not cosine similarity]:

<result:file filename="a.go">
  <result:chunk line-start="1" line-end="30" symbol="A" kind="function" score="12.5" truncated="true">
func A() {
  </result:chunk>
</result:file>

<result:file filename="b.go">
  <result:chunk line-start="4" line-end="9" symbol="B" kind="method" score="3">
body
  </result:chunk>
</result:file>

Output capped at max_tokens=1000 (~4000 bytes): 2 of 5 results shown, 1 snippets truncated, 3 results omitted. Read the listed line ranges directly for full code, or raise max_tokens (-1 = unlimited).`
	out, err := parseSearchOutput(text)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Results) != 2 || !out.Results[0].Truncated || out.Results[1].Truncated || out.Results[0].EndLine != 30 || out.Results[0].Content != "func A() {" || out.Results[1].Content != "body" {
		t.Fatalf("results = %+v", out.Results)
	}
	if out.Truncated != 1 || out.Omitted != 3 {
		t.Fatalf("budget counts = truncated %d omitted %d", out.Truncated, out.Omitted)
	}

	out, err = parseSearchOutput("Found 1 results:\n\n<result:file filename=\"x.go\">\n  <result:chunk line-start=\"1\" line-end=\"2\" symbol=\"X\" kind=\"function\" related-to=\"y.go:Y\" relation=\"direct_call\" truncated=\"true\">\nfunc X() {}\n  </result:chunk>\n</result:file>")
	if err != nil || len(out.Results) != 1 || !out.Results[0].Truncated || out.Results[0].RelatedTo != "y.go:Y" || out.Truncated != 0 {
		t.Fatalf("related truncated = %+v, %v", out, err)
	}

	for _, header := range []string{
		`  <result:chunk line-start="1" line-end="2" symbol="X" kind="function" truncated="true" score="1">`,
		`  <result:chunk line-start="1" line-end="2" symbol="X" kind="function" score="1" truncated="false">`,
		`  <result:chunk line-start="1" line-end="2" symbol="X" kind="function" truncated="true">`,
	} {
		if _, err := parseSearchOutput("Found 1 results:\n<result:file filename=\"x.go\">\n" + header + "\n  </result:chunk>\n</result:file>\n"); err == nil {
			t.Errorf("accepted malformed header %s", header)
		}
	}
}
