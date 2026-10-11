package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNewOutputBudget(t *testing.T) {
	tests := []struct {
		in       int
		capBytes int
		wantErr  bool
	}{
		{in: 0, capBytes: 16000},
		{in: -1, capBytes: -1},
		{in: 256, capBytes: 1024},
		{in: 4000, capBytes: 16000},
		{in: 255, wantErr: true},
		{in: 1, wantErr: true},
		{in: -2, wantErr: true},
		{in: math.MinInt, wantErr: true},
		{in: math.MaxInt, capBytes: math.MaxInt},
		{in: math.MaxInt/bytesPerToken + 1, capBytes: math.MaxInt},
	}
	for _, tt := range tests {
		b, err := newOutputBudget(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("newOutputBudget(%d) err = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if tt.wantErr {
			continue
		}
		got := b.capBytes
		if b.unlimited() {
			got = -1
		}
		if got != tt.capBytes {
			t.Errorf("newOutputBudget(%d) cap = %d, want %d", tt.in, got, tt.capBytes)
		}
	}
}

func TestValidateSearchInput_MaxTokens(t *testing.T) {
	dir := t.TempDir()
	for _, tt := range []struct {
		maxTokens int
		wantErr   bool
	}{{0, false}, {-1, false}, {256, false}, {255, true}, {1, true}, {-2, true}} {
		input := SemanticSearchInput{Query: "q", Path: dir, MaxTokens: tt.maxTokens}
		if err := validateSearchInput(&input); (err != nil) != tt.wantErr {
			t.Errorf("max_tokens=%d: err = %v, wantErr %v", tt.maxTokens, err, tt.wantErr)
		}
	}
}

func TestSearchCmd_MaxTokensValidatedBeforeSearch(t *testing.T) {
	if f := searchCmd.Flags().Lookup("max-tokens"); f == nil || f.DefValue != "0" {
		t.Fatalf("max-tokens flag = %+v, want default 0", f)
	}
	t.Cleanup(func() { _ = searchCmd.Flags().Set("max-tokens", "0") })
	if err := searchCmd.Flags().Set("max-tokens", "100"); err != nil {
		t.Fatal(err)
	}
	err := runSearch(searchCmd, []string{"q"})
	if err == nil || !strings.Contains(err.Error(), "max_tokens") {
		t.Fatalf("runSearch err = %v, want max_tokens validation error", err)
	}
}

func mustBudget(t *testing.T, maxTokens int) outputBudget {
	t.Helper()
	b, err := newOutputBudget(maxTokens)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func lines(n, width int, prefix string) string {
	var b strings.Builder
	for i := range n {
		if i > 0 {
			b.WriteByte('\n')
		}
		line := fmt.Sprintf("%s line %d ", prefix, i)
		b.WriteString(line + strings.Repeat("x", max(0, width-len(line))))
	}
	return b.String()
}

func bm25Output(n, contentLines int) SemanticSearchOutput {
	out := SemanticSearchOutput{Retrieval: "bm25"}
	for i := range n {
		out.Results = append(out.Results, SearchResultItem{
			FilePath:  fmt.Sprintf("/p/pkg/file%02d.go", i),
			Symbol:    fmt.Sprintf("Func%02d", i),
			Kind:      "function",
			StartLine: 10 * i,
			EndLine:   10*i + contentLines,
			Score:     float32(100 - i),
			Content:   lines(contentLines, 60, fmt.Sprintf("f%02d", i)),
		})
	}
	return out
}

func semanticOutput(n, contentLines int) SemanticSearchOutput {
	out := bm25Output(n, contentLines)
	out.Retrieval = ""
	for i := range out.Results {
		out.Results[i].Score = 0.9 - float32(i)*0.01
		// Two chunks per file exercise the semantic formatter's grouping.
		out.Results[i].FilePath = fmt.Sprintf("/p/pkg/file%02d.go", i/2)
	}
	return out
}

var chunkHeaderRe = regexp.MustCompile(`(?m)^  <result:chunk [^\n]*>$`)

func checkBounded(t *testing.T, text string, b outputBudget, trailing int) {
	t.Helper()
	if !b.unlimited() && len(text)+trailing > b.capBytes {
		t.Fatalf("rendered %d bytes + %d trailing exceeds cap %d", len(text), trailing, b.capBytes)
	}
	// Output stays structurally intact: every opened element is closed.
	if strings.Count(text, "<result:file ") != strings.Count(text, "</result:file>") ||
		strings.Count(text, "<result:chunk ") != strings.Count(text, "</result:chunk>") {
		t.Fatalf("unbalanced result markup:\n%s", text)
	}
	for _, h := range chunkHeaderRe.FindAllString(text, -1) {
		if strings.Contains(h, "truncated=") && !strings.HasSuffix(h, ` truncated="true">`) {
			t.Fatalf("truncated attribute is not last: %s", h)
		}
	}
}

func TestBoundedSearchText_FitsIsByteIdentical(t *testing.T) {
	for name, out := range map[string]SemanticSearchOutput{
		"bm25":     bm25Output(8, 5),
		"semantic": semanticOutput(8, 5),
		"empty":    {Retrieval: "bm25"},
		"none":     {},
		"stale":    {StaleWarning: staleIndexWarning},
	} {
		want := formatSearchResults("/p", out)
		if got := boundedSearchText("/p", out, mustBudget(t, 0), 1); got != want {
			t.Errorf("%s default differs from unbounded output:\n%s\n---\n%s", name, got, want)
		}
	}
	big := bm25Output(50, 200)
	want := formatSearchResults("/p", big)
	if len(want) < 100_000 {
		t.Fatalf("fixture too small: %d", len(want))
	}
	if got := boundedSearchText("/p", big, mustBudget(t, -1), 1); got != want {
		t.Fatal("unlimited output differs from unbounded formatter")
	}
	sem := semanticOutput(50, 200)
	if got := boundedSearchText("/p", sem, mustBudget(t, -1), 0); got != formatSearchResults("/p", sem) {
		t.Fatal("unlimited semantic output differs from unbounded formatter")
	}
}

var footerRe = regexp.MustCompile(`Output capped at max_tokens=(\d+) \(~(\d+) bytes\): (\d+) of (\d+) results shown, (\d+) snippets truncated, (\d+) results omitted\.`)

func footerCounts(t *testing.T, text string) (shown, total, truncated, omitted int) {
	t.Helper()
	m := footerRe.FindStringSubmatch(text)
	if m == nil {
		t.Fatalf("missing budget footer:\n%s", text)
	}
	_, _ = fmt.Sscan(m[3], &shown)
	_, _ = fmt.Sscan(m[4], &total)
	_, _ = fmt.Sscan(m[5], &truncated)
	_, _ = fmt.Sscan(m[6], &omitted)
	return
}

func TestBoundedSearchText_PreservesBreadth(t *testing.T) {
	for name, out := range map[string]SemanticSearchOutput{"bm25": bm25Output(20, 40), "semantic": semanticOutput(20, 40)} {
		t.Run(name, func(t *testing.T) {
			b := mustBudget(t, 1000)
			text := boundedSearchText("/p", out, b, 0)
			checkBounded(t, text, b, 0)
			for _, r := range out.Results {
				if !strings.Contains(text, `symbol="`+r.Symbol+`"`) {
					t.Errorf("locator for %s dropped", r.Symbol)
				}
				// Every hit keeps its original source range.
				if !strings.Contains(text, fmt.Sprintf(`line-start="%d" line-end="%d"`, r.StartLine, r.EndLine)) {
					t.Errorf("source range of %s changed", r.Symbol)
				}
			}
			shown, total, truncated, omitted := footerCounts(t, text)
			if shown != 20 || total != 20 || omitted != 0 || truncated == 0 {
				t.Fatalf("footer = %d/%d truncated=%d omitted=%d", shown, total, truncated, omitted)
			}
			if got := strings.Count(text, `truncated="true"`); got != truncated {
				t.Fatalf("%d truncated attributes, footer says %d", got, truncated)
			}
			if !strings.Contains(text, "Found 20 results") {
				t.Fatal("header must count rendered results")
			}
			// Snippets that received content are whole-line prefixes.
			for _, r := range out.Results {
				if strings.Contains(text, r.Content+"\n") {
					continue
				}
				first, _, _ := strings.Cut(r.Content, "\n")
				if strings.Contains(text, first[:20]) && !strings.Contains(text, first+"\n") {
					t.Errorf("%s content cut inside a line", r.Symbol)
				}
			}
		})
	}
}

func TestBoundedSearchText_WaterFillFavorsSmallSnippets(t *testing.T) {
	out := bm25Output(5, 3)
	out.Results[0].Content = lines(400, 60, "huge")
	b := mustBudget(t, 1000)
	text := boundedSearchText("/p", out, b, 0)
	checkBounded(t, text, b, 0)
	for _, r := range out.Results[1:] {
		if !strings.Contains(text, r.Content+"\n  </result:chunk>") {
			t.Errorf("small snippet %s not kept whole", r.Symbol)
		}
	}
	if !strings.Contains(text, `symbol="Func00" kind="function" score="100" truncated="true">`+"\nhuge line 0") {
		t.Errorf("large snippet should be marked truncated and keep a prefix:\n%s", text)
	}
	if _, _, truncated, _ := footerCounts(t, text); truncated != 1 {
		t.Fatalf("truncated = %d, want 1", truncated)
	}
	// Leftover bytes from line rounding go to the truncated snippet, so the
	// unused budget is below one line plus the footer slack.
	if slack := b.capBytes - len(text); slack > 200 {
		t.Errorf("left %d bytes unused", slack)
	}
}

func TestBoundedSearchText_MetadataOnlyPrefix(t *testing.T) {
	out := bm25Output(1000, 0)
	for i := range out.Results {
		out.Results[i].Content = ""
	}
	b := mustBudget(t, 256)
	for _, trailing := range []int{0, 1} {
		text := boundedSearchText("/p", out, b, trailing)
		checkBounded(t, text, b, trailing)
		shown, total, truncated, omitted := footerCounts(t, text)
		if shown == 0 || total != 1000 || truncated != 0 || shown+omitted != total {
			t.Fatalf("footer = %d/%d truncated=%d omitted=%d", shown, total, truncated, omitted)
		}
		if !strings.Contains(text, fmt.Sprintf("Found %d results", shown)) {
			t.Fatal("header must count rendered results")
		}
		// The retained results are the rank prefix.
		for i := range shown {
			if !strings.Contains(text, fmt.Sprintf(`symbol="Func%02d"`, i)) {
				t.Fatalf("rank %d missing from prefix", i)
			}
		}
		if strings.Contains(text, fmt.Sprintf(`symbol="Func%02d"`, shown)) {
			t.Fatal("result beyond prefix rendered")
		}
	}
}

func TestBoundedSearchText_SummaryAndMaxLines(t *testing.T) {
	out := bm25Output(30, 3)
	for i := range out.Results {
		// max_lines already applied by the caller: the marker counts as content.
		out.Results[i].Content += "\n..."
	}
	b := mustBudget(t, 256)
	text := boundedSearchText("/p", out, b, 1)
	checkBounded(t, text, b, 1)
	if strings.Contains(text, "...\n  </result:chunk>") && strings.Count(text, `truncated="true"`) == 0 {
		t.Fatal("expected cuts")
	}
}

func TestBoundedSearchText_UTF8AndInvalidBytes(t *testing.T) {
	for name, content := range map[string]string{
		"cjk":     strings.Repeat("漢字かな😀", 20000),
		"invalid": strings.Repeat("\xff\xfe\x80abc", 20000),
		"crlf":    strings.Repeat("line\r\n", 5000),
	} {
		t.Run(name, func(t *testing.T) {
			out := bm25Output(3, 1)
			out.Results[0].Content = content
			b := mustBudget(t, 300)
			text := boundedSearchText("/p", out, b, 0)
			checkBounded(t, text, b, 0)
			if !utf8.ValidString(text) {
				t.Fatal("bounded output is not valid UTF-8")
			}
			// Invalid bytes arrive as one U+FFFD each, as JSON would deliver them.
			if name != "crlf" && !strings.Contains(text, `truncated="true">`+"\n"+runePrefix(validUTF8(content), 200)) {
				t.Fatalf("oversized first line lost its floor snippet:\n%s", text)
			}
		})
	}
}

func TestBoundedSearchText_EnormousSingleLine(t *testing.T) {
	out := SemanticSearchOutput{Retrieval: "bm25", Results: []SearchResultItem{{
		FilePath: "/p/dist/app.min.js", Symbol: "bundle", Kind: "file", StartLine: 1, EndLine: 1, Score: 3,
		Content: strings.Repeat("a=1;", 50_000),
	}}}
	b := mustBudget(t, 0)
	text := boundedSearchText("/p", out, b, 1)
	checkBounded(t, text, b, 1)
	if !strings.Contains(text, strings.Repeat("a=1;", 64)) {
		t.Fatal("expected a rune-safe prefix of the oversized line")
	}
	if b.capBytes-len(text) > 300 {
		t.Fatalf("enormous line wasted %d bytes", b.capBytes-len(text))
	}
}

func TestBoundedSearchText_NoFragmentsBelowOneLine(t *testing.T) {
	// Shares are smaller than one 250-byte line: snippets are either whole
	// lines or locator-only, never a fragment of a short line.
	out := bm25Output(5, 2)
	for i := range out.Results {
		out.Results[i].Content = lines(2, 250, fmt.Sprintf("hit%d", i))
	}
	b := mustBudget(t, 400)
	text := boundedSearchText("/p", out, b, 0)
	checkBounded(t, text, b, 0)
	for _, r := range out.Results {
		for _, line := range strings.Split(r.Content, "\n") {
			if strings.Contains(text, line[:40]) && !strings.Contains(text, line+"\n") {
				t.Fatalf("fragment of %s rendered:\n%s", r.Symbol, text)
			}
		}
	}
	if _, _, truncated, _ := footerCounts(t, text); truncated == 0 {
		t.Fatal("expected truncation")
	}
}

func TestBoundedSearchText_EscapedAndLongAttributes(t *testing.T) {
	out := bm25Output(40, 10)
	for i := range out.Results {
		out.Results[i].Symbol = strings.Repeat("operator<<&\"", 20)
		out.Results[i].FilePath = fmt.Sprintf("/p/a&b/%s%d.go", strings.Repeat("<dir>", 30), i)
	}
	sem := out
	sem.Retrieval = ""
	sem.Results[0].Symbol = strings.Repeat("A+B+", 400)
	for _, o := range []SemanticSearchOutput{out, sem} {
		for _, tokens := range []int{256, 1000, 4000} {
			b := mustBudget(t, tokens)
			text := boundedSearchText("/p", o, b, 1)
			checkBounded(t, text, b, 1)
			footerCounts(t, text)
		}
	}
}

func TestBoundedSearchText_RelatedContext(t *testing.T) {
	base := bm25Output(3, 2)
	related := SearchResultItem{FilePath: "/p/pkg/helper.go", Symbol: "helper", Kind: "function", StartLine: 1, EndLine: 3, RelatedTo: "pkg/file00.go:Func00", Relation: "direct_call", Content: lines(3, 60, "helper")}
	base.Results = append(base.Results, related)

	full := formatSearchResults("/p", base)
	b := outputBudget{maxTokens: 1, capBytes: len(full) + 300}
	fit := boundedSearchText("/p", base, b, 0)
	if fit != full {
		t.Fatal("output that fits must be unchanged")
	}

	// Tighten so the related snippet cannot fit whole: it is dropped and
	// direct content stays whole.
	b = outputBudget{maxTokens: 1, capBytes: len(full) - 20}
	text := boundedSearchText("/p", base, b, 0)
	checkBounded(t, text, b, 0)
	if strings.Contains(text, "related-to=") {
		t.Fatal("related snippet should be dropped before direct content is cut")
	}
	if strings.Contains(text, `truncated="true"`) {
		t.Fatal("direct content cut while related context could be dropped instead")
	}
	if shown, _, _, omitted := footerCounts(t, text); shown != 3 || omitted != 1 {
		t.Fatalf("shown=%d omitted=%d", shown, omitted)
	}

	// Space for a second, unrelated direct hit's content but not the
	// related one keeps provenance intact on retained related items.
	big := bm25Output(3, 2)
	big.Results = append(big.Results, related)
	big.Results[2].Content = lines(200, 60, "tail")
	b = mustBudget(t, 1000)
	text = boundedSearchText("/p", big, b, 0)
	checkBounded(t, text, b, 0)
	if strings.Contains(text, "related-to=") {
		t.Fatal("related context kept although a direct hit was truncated")
	}

	// A related snippet whose parent was omitted is never rendered.
	orphan := bm25Output(1, 0)
	orphan.Results[0].Symbol = strings.Repeat("S", 2000)
	orphan.Results[0].Content = ""
	orphan.Results = append(orphan.Results, SearchResultItem{FilePath: "/p/x.go", Symbol: "x", Kind: "function", StartLine: 1, EndLine: 1, RelatedTo: "pkg/file00.go:" + strings.Repeat("S", 2000), Relation: "direct_call", Content: "x"})
	b = mustBudget(t, 256)
	text = boundedSearchText("/p", orphan, b, 0)
	checkBounded(t, text, b, 0)
	if strings.Contains(text, "related-to=") {
		t.Fatal("orphaned related snippet rendered")
	}

	// With two related snippets and room for only the first, the first is
	// kept whole, in order, with its provenance.
	second := related
	second.Symbol, second.FilePath, second.Content = "other", "/p/pkg/other.go", lines(20, 60, "other")
	keep := bm25Output(3, 2)
	keep.Results = append(keep.Results, related, second)
	full = formatSearchResults("/p", keep)
	b = outputBudget{maxTokens: 1, capBytes: len(full) - 100}
	text = boundedSearchText("/p", keep, b, 0)
	checkBounded(t, text, b, 0)
	if !strings.Contains(text, `symbol="helper" kind="function" related-to="pkg/file00.go:Func00" relation="direct_call">`+"\n"+related.Content+"\n") {
		t.Fatalf("first related snippet should be kept whole with provenance:\n%s", text)
	}
	if strings.Contains(text, `symbol="other"`) {
		t.Fatal("second related snippet should be omitted")
	}
	if shown, _, truncated, omitted := footerCounts(t, text); shown != 4 || omitted != 1 || truncated != 0 {
		t.Fatalf("shown=%d omitted=%d truncated=%d", shown, omitted, truncated)
	}
}

func TestBoundedSearchText_MissingContentIsNotTruncated(t *testing.T) {
	out := semanticOutput(10, 30)
	out.Results[1].Content = "" // file vanished: fillSnippets leaves no content
	b := mustBudget(t, 300)
	text := boundedSearchText("/p", out, b, 0)
	checkBounded(t, text, b, 0)
	if strings.Contains(text, `symbol="Func01" kind="function" score="0.89" truncated="true"`) {
		t.Fatal("naturally empty content marked truncated")
	}
}

func TestBoundedSearchText_Warnings(t *testing.T) {
	huge := "seed failed for " + strings.Repeat("/very/long/path", 500)
	for name, out := range map[string]SemanticSearchOutput{
		"stale-only":    {StaleWarning: staleIndexWarning, SeedWarning: huge},
		"bm25-empty":    {Retrieval: "bm25", SeedWarning: huge, StaleWarning: staleIndexWarning},
		"bm25-results":  func() SemanticSearchOutput { o := bm25Output(5, 20); o.SeedWarning = huge; return o }(),
		"semantic-hint": {SeedWarning: huge, FilteredHint: strings.Repeat("hint ", 1000)},
	} {
		t.Run(name, func(t *testing.T) {
			b := mustBudget(t, 256)
			text := boundedSearchText("/p", out, b, 1)
			checkBounded(t, text, b, 1)
			if !strings.Contains(text, "Warning: seed failed for /very/long/path") {
				t.Fatalf("seed warning lost:\n%s", text)
			}
			if out.StaleWarning != "" && !strings.Contains(text, "Warning: "+staleIndexWarning) {
				t.Fatalf("short stale warning should stay whole:\n%s", text)
			}
			if len(out.Results) == 0 && strings.Contains(text, "Output capped") {
				t.Fatal("empty results must not carry a footer")
			}
		})
	}
}

func TestBoundedSearchText_SemanticAllOmittedIsNotNoResults(t *testing.T) {
	out := semanticOutput(2, 1)
	for i := range out.Results {
		out.Results[i].Symbol = strings.Repeat("Sym", 600)
	}
	b := mustBudget(t, 256)
	text := boundedSearchText("/p", out, b, 0)
	checkBounded(t, text, b, 0)
	if strings.HasPrefix(text, "No results found") {
		t.Fatalf("budget omission reported as no results:\n%s", text)
	}
	if shown, total, _, omitted := footerCounts(t, text); shown != 0 || total != 2 || omitted != 2 {
		t.Fatalf("shown=%d total=%d omitted=%d", shown, total, omitted)
	}
}

func TestBoundedSearchText_InvariantSweep(t *testing.T) {
	for _, n := range []int{1, 2, 7, 25, 120} {
		for _, contentLines := range []int{0, 1, 9, 80} {
			for _, tokens := range []int{256, 257, 600, 1000, 4000} {
				for _, out := range []SemanticSearchOutput{bm25Output(n, contentLines), semanticOutput(n, contentLines)} {
					for _, trailing := range []int{0, 1} {
						b := mustBudget(t, tokens)
						text := boundedSearchText("/p", out, b, trailing)
						checkBounded(t, text, b, trailing)
						if strings.Contains(text, "Output capped") {
							shown, total, truncated, omitted := footerCounts(t, text)
							if total != n || shown+omitted != n || strings.Count(text, `truncated="true"`) != truncated {
								t.Fatalf("n=%d lines=%d tokens=%d: footer %d/%d t=%d o=%d", n, contentLines, tokens, shown, total, truncated, omitted)
							}
						} else if text != formatSearchResults("/p", out) {
							t.Fatalf("n=%d lines=%d tokens=%d: changed output without footer", n, contentLines, tokens)
						}
					}
				}
			}
		}
	}
}

func TestHandleSemanticSearch_MaxTokensBM25(t *testing.T) {
	cfg := offlineLexicalConfig(t)
	root := t.TempDir()
	root, _ = filepath.EvalSymlinks(root)
	for i := range 12 {
		var src strings.Builder
		fmt.Fprintf(&src, "package fixture\n\n// PaymentLedger%d reconciles payment ledger entries.\nfunc PaymentLedger%d() {\n", i, i)
		for j := range 60 {
			fmt.Fprintf(&src, "\t_ = \"payment ledger reconciliation step %d of %d\"\n", j, i)
		}
		src.WriteString("}\n")
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("f%02d.go", i)), []byte(src.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ic := &indexerCache{cfg: cfg}
	defer ic.Close()
	search := func(maxTokens int) string {
		t.Helper()
		res, _, err := ic.handleSemanticSearch(context.Background(), &mcp.CallToolRequest{}, SemanticSearchInput{Query: "payment ledger reconciliation", Path: root, Limit: 12, MaxTokens: maxTokens})
		if err != nil {
			t.Fatal(err)
		}
		return res.Content[0].(*mcp.TextContent).Text
	}
	unlimited := search(-1)
	if len(unlimited) <= 1024 || strings.Contains(unlimited, "Output capped") {
		t.Fatalf("unlimited output unexpectedly small or capped (%d bytes)", len(unlimited))
	}
	bounded := search(256)
	if len(bounded) > 1024 {
		t.Fatalf("bounded output %d bytes exceeds 1024", len(bounded))
	}
	if !strings.Contains(bounded, "Output capped at max_tokens=256") {
		t.Fatalf("missing footer:\n%s", bounded)
	}
	if _, _, err := ic.handleSemanticSearch(context.Background(), &mcp.CallToolRequest{}, SemanticSearchInput{Query: "payment", Path: root, MaxTokens: 10}); err == nil {
		t.Fatal("max_tokens below minimum accepted")
	}
}

func TestMinimalBudgetTextFitsMinimumBudget(t *testing.T) {
	floor := minMaxTokens*bytesPerToken - 1 // CLI reserves the final newline
	for _, retrieval := range []string{"bm25", ""} {
		out := SemanticSearchOutput{
			Retrieval: retrieval, Reindexed: true, IndexedFiles: math.MaxInt,
			SeedWarning: strings.Repeat("w", 1<<20), StaleWarning: staleIndexWarning,
			Results: []SearchResultItem{{FilePath: "/p/" + strings.Repeat("d/", 1<<16), Symbol: "S", Content: "x"}},
			budget:  &budgetReport{maxTokens: math.MaxInt, capBytes: math.MaxInt, shown: math.MaxInt, total: math.MaxInt, truncated: math.MaxInt, omitted: math.MaxInt},
		}
		text := minimalBudgetText("/p", out)
		if len(text) > floor {
			t.Fatalf("%q minimal text is %d bytes, above the %d-byte minimum budget", retrieval, len(text), floor)
		}
		if strings.HasPrefix(text, "No results found") || !strings.Contains(text, "Output capped") || strings.Contains(text, "<result:") {
			t.Fatalf("minimal text malformed:\n%s", text)
		}
	}
}

func TestBoundedSearchText_WarningAndMetadataExhaustion(t *testing.T) {
	// Warnings and locators each alone exceed the budget: warnings are cut
	// fairly, results omitted whole, and markup stays closed.
	for _, retrieval := range []string{"bm25", ""} {
		out := bm25Output(50, 5)
		out.Retrieval = retrieval
		for i := range out.Results {
			out.Results[i].Symbol = strings.Repeat("Σ", 300+i)
		}
		out.Results[0].Symbol = "Small"
		out.SeedWarning = strings.Repeat("seed ", 4000)
		out.StaleWarning = strings.Repeat("stale ", 4000)
		out.Reindexed, out.IndexedFiles = true, 123456789
		b := mustBudget(t, 256)
		text := boundedSearchText("/p", out, b, 1)
		checkBounded(t, text, b, 1)
		if !strings.Contains(text, "Warning: seed seed") || !strings.Contains(text, "Warning: stale stale") {
			t.Fatalf("warnings must stay visible:\n%s", text)
		}
		if shown, total, _, omitted := footerCounts(t, text); total != 50 || shown+omitted != 50 {
			t.Fatalf("shown=%d omitted=%d", shown, omitted)
		}
	}
}

// invalidOutput carries invalid UTF-8 in every rendered field. encoding/json,
// and therefore MCP, replaces each invalid byte with the 3-byte U+FFFD.
func invalidOutput(retrieval string) SemanticSearchOutput {
	out := bm25Output(6, 3)
	out.Retrieval = retrieval
	out.SeedWarning = "seed \xff\xfe warning"
	out.StaleWarning = "stale \x80"
	for i := range out.Results {
		out.Results[i].FilePath = fmt.Sprintf("/p/bad\xff/f%d.go", i)
		out.Results[i].Symbol = "Sym\xc3"
		out.Results[i].Kind = "function\xff"
		out.Results[i].Content = strings.Repeat("\xff", 4000) + "\n" + strings.Repeat("ok \xfe\n", 50)
	}
	out.Results[0].Content = strings.Repeat("\xff", 16000)
	return out
}

func jsonRoundTrip(t *testing.T, text string) string {
	t.Helper()
	wire, err := json.Marshal(struct {
		Text string `json:"text"`
	}{text})
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(wire, &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded.Text
}

func TestBoundedSearchText_InvalidUTF8SurvivesJSON(t *testing.T) {
	for _, retrieval := range []string{"bm25", ""} {
		for _, tokens := range []int{256, 1000, 4000} {
			out := invalidOutput(retrieval)
			source := slices.Clone(out.Results)
			b := mustBudget(t, tokens)
			text := boundedSearchText("/p", out, b, 0)
			received := jsonRoundTrip(t, text)
			if received != text {
				t.Fatalf("bounded text is not already in its JSON-decoded form (%d -> %d bytes)", len(text), len(received))
			}
			checkBounded(t, received, b, 0)
			if !utf8.ValidString(received) {
				t.Fatal("received text is not valid UTF-8")
			}
			// One U+FFFD per invalid byte, as encoding/json does.
			if !strings.Contains(received, "seed \uFFFD\uFFFD warning") {
				t.Fatalf("warning replacement differs from encoding/json:\n%s", received)
			}
			if !slices.EqualFunc(source, out.Results, func(a, b SearchResultItem) bool { return a == b }) {
				t.Fatal("source results were mutated")
			}
		}
	}
	// Unlimited output stays raw: -1 is the old output.
	out := invalidOutput("bm25")
	if got := boundedSearchText("/p", out, mustBudget(t, -1), 0); got != formatSearchResults("/p", out) {
		t.Fatal("unlimited output changed")
	}
}

func TestBoundedSearchText_InvalidUTF8ThroughMCPTransport(t *testing.T) {
	cfg := offlineLexicalConfig(t)
	root := t.TempDir()
	root, _ = filepath.EvalSymlinks(root)
	for i := range 4 {
		src := fmt.Sprintf("def ledger_blob_%d():\n    \"\"\"LedgerBlob holds binary ledger bytes.\"\"\"\n", i)
		for range 40 {
			src += "    x = 'ledger " + strings.Repeat("\xff", 30) + "'\n"
		}
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("blob%d.py", i)), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ic := &indexerCache{cfg: cfg}
	defer ic.Close()
	ctx := context.Background()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	if _, err := newMCPServer(ic).Connect(ctx, serverTransport, nil); err != nil {
		t.Fatal(err)
	}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = session.Close() }()
	for _, tokens := range []int{1000, 4000} {
		res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "semantic_search", Arguments: map[string]any{"query": "ledger blob", "path": root, "max_tokens": tokens}})
		if err != nil || res.IsError {
			t.Fatalf("call: %v %+v", err, res)
		}
		received := res.Content[0].(*mcp.TextContent).Text
		b := mustBudget(t, tokens)
		checkBounded(t, received, b, 0)
		if !utf8.ValidString(received) || !strings.Contains(received, "ledger_blob_") || !strings.Contains(received, "\uFFFD") {
			t.Fatalf("received %d bytes, valid=%v:\n%s", len(received), utf8.ValidString(received), received)
		}
	}
}

func TestValidUTF8MatchesJSON(t *testing.T) {
	for _, s := range []string{
		"", "plain", "漢字😀", "\xff", "\xff\xfe\x80", "a\xe6\xbcb", "\xed\xa0\x80", "\xf0\x9f\x98", "ok\xc3",
		strings.Repeat("\xff", 16000), " <&> ",
	} {
		if got, want := validUTF8(s), jsonRoundTrip(t, s); got != want {
			t.Errorf("validUTF8(%q) = %q, encoding/json delivers %q", s, got, want)
		}
	}
}
