package cmd

import (
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	// defaultMaxTokens applies when max_tokens is omitted or 0.
	defaultMaxTokens = 4000
	minMaxTokens     = 256
	// bytesPerToken estimates tokens from UTF-8 bytes, matching the related
	// context budget. The byte ceiling is exact; token counts are not.
	bytesPerToken = 4
	// contentFloor is the shortest cut of an oversized first line worth
	// rendering; shorter shares leave the hit locator-only.
	contentFloor = 256
	// truncatedAttr is appended last to chunk headers whose content the
	// output budget shortened.
	truncatedAttr = ` truncated="true"`
)

// outputBudget caps the rendered search text at capBytes UTF-8 bytes.
type outputBudget struct {
	maxTokens int // -1 is unlimited
	capBytes  int
}

// newOutputBudget validates max_tokens: 0 selects the default, -1 disables
// the cap, and anything else must be at least minMaxTokens.
func newOutputBudget(maxTokens int) (outputBudget, error) {
	switch {
	case maxTokens == -1:
		return outputBudget{maxTokens: -1}, nil
	case maxTokens == 0:
		maxTokens = defaultMaxTokens
	case maxTokens < minMaxTokens:
		return outputBudget{}, fmt.Errorf("max_tokens must be 0 (default %d), -1 (unlimited) or at least %d, got %d", defaultMaxTokens, minMaxTokens, maxTokens)
	}
	if maxTokens > math.MaxInt/bytesPerToken {
		return outputBudget{maxTokens: maxTokens, capBytes: math.MaxInt}, nil
	}
	return outputBudget{maxTokens: maxTokens, capBytes: maxTokens * bytesPerToken}, nil
}

func (b outputBudget) unlimited() bool { return b.maxTokens < 0 }

// budgetReport is rendered as a footer when the budget cut the output.
type budgetReport struct {
	maxTokens, capBytes              int
	shown, total, truncated, omitted int
}

func writeBudgetFooter(b *strings.Builder, r *budgetReport) {
	if r == nil {
		return
	}
	if !strings.HasSuffix(b.String(), "\n") {
		b.WriteByte('\n')
	}
	fmt.Fprintf(b, "\nOutput capped at max_tokens=%d (~%d bytes): %d of %d results shown, %d snippets truncated, %d results omitted. Read the listed line ranges directly for full code, or raise max_tokens (-1 = unlimited).",
		r.maxTokens, r.capBytes, r.shown, r.total, r.truncated, r.omitted)
}

// boundedSearchText renders out like formatSearchResults within budget,
// reserving trailing bytes the caller appends (the CLI's final newline).
// Output that fits is returned unchanged. Otherwise every direct hit whose
// locator fits is kept in rank order, snippets share the remaining bytes
// max-min fairly as whole-line prefixes, and related context is kept whole
// only after all direct content.
func boundedSearchText(projectPath string, out SemanticSearchOutput, budget outputBudget, trailing int) string {
	if budget.unlimited() {
		return formatSearchResults(projectPath, out)
	}
	// Size the text the caller receives: MCP's JSON encoding turns each
	// invalid byte into a 3-byte U+FFFD, so replace them before measuring.
	projectPath, out = wireUTF8(projectPath, out)
	text := formatSearchResults(projectPath, out)
	if len(text) <= budget.capBytes-trailing {
		return text
	}
	limit := budget.capBytes - trailing
	size := func(o SemanticSearchOutput) int { return len(formatSearchResults(projectPath, o)) }

	var direct, related []SearchResultItem
	for _, r := range out.Results {
		if r.RelatedTo != "" {
			related = append(related, r)
		} else {
			direct = append(direct, r)
		}
	}
	total := len(out.Results)
	out.Results = nil
	if total > 0 {
		// Size with the widest footer the final counts can produce.
		out.budget = &budgetReport{maxTokens: budget.maxTokens, capBytes: budget.capBytes, shown: total, total: total, truncated: total, omitted: total}
	}
	out = fitWarnings(out, limit, size)

	// Longest rank prefix of direct hits whose locators fit, sized as if
	// every snippet were truncated so any content split stays within budget.
	locators := make([]SearchResultItem, len(direct))
	for i, r := range direct {
		r.truncated = r.Content != ""
		r.Content = ""
		locators[i] = r
	}
	kept := sort.Search(len(locators)+1, func(n int) bool {
		o := out
		o.Results = locators[:n]
		return size(o) > limit
	}) - 1
	kept = max(kept, 0)
	o := out
	o.Results = locators[:kept]
	results := slices.Clone(o.Results)
	contents := make([]string, kept)
	for i := range contents {
		contents[i] = direct[i].Content
	}
	packed, cut := packContents(contents, limit-size(o))
	truncated := 0
	for i := range results {
		results[i].Content = packed[i]
		results[i].truncated = cut[i]
		if cut[i] {
			truncated++
		}
	}

	if truncated == 0 {
		for _, r := range related {
			if !hasParent(r, results[:kept]) {
				continue
			}
			o.Results = append(slices.Clone(results), r)
			if size(o) <= limit {
				results = o.Results
			}
		}
	}

	out.Results = results
	if out.budget != nil {
		out.budget = &budgetReport{maxTokens: budget.maxTokens, capBytes: budget.capBytes, shown: len(results), total: total, truncated: truncated, omitted: total - len(results)}
		if truncated == 0 && len(results) == total {
			out.budget = nil
		}
	}
	text = formatSearchResults(projectPath, out)
	if len(text) > limit {
		// Unreachable: results are only kept when the rendering with the
		// widest footer fits. Should a formatter change shape, fall back to
		// the bare header and footer, which always fits the minimum budget.
		text = minimalBudgetText(projectPath, out)
	}
	return text
}

// minimalBudgetText renders only the header and budget footer: no results,
// warnings or snippets. TestMinimalBudgetTextFitsMinimumBudget proves it fits
// the smallest accepted budget with the widest possible counts.
func minimalBudgetText(projectPath string, out SemanticSearchOutput) string {
	if out.budget != nil {
		r := *out.budget
		r.truncated, r.omitted, r.shown = 0, r.total, 0
		out.budget = &r
	}
	out.Results, out.SeedWarning, out.StaleWarning, out.FilteredHint = nil, "", "", ""
	return formatSearchResults(projectPath, out)
}

// wireUTF8 replaces every invalid UTF-8 byte in the rendered fields with
// U+FFFD, exactly as encoding/json does, without mutating out.Results.
// Valid text is returned unchanged.
func wireUTF8(projectPath string, out SemanticSearchOutput) (string, SemanticSearchOutput) {
	projectPath = validUTF8(projectPath)
	for _, w := range []*string{&out.SeedWarning, &out.StaleWarning, &out.FilteredHint} {
		*w = validUTF8(*w)
	}
	cloned := false
	for i, r := range out.Results {
		fixed := r
		for _, f := range []*string{&fixed.FilePath, &fixed.Symbol, &fixed.Kind, &fixed.RelatedTo, &fixed.Relation, &fixed.Content} {
			*f = validUTF8(*f)
		}
		if fixed == r {
			continue
		}
		if !cloned {
			out.Results = slices.Clone(out.Results)
			cloned = true
		}
		out.Results[i] = fixed
	}
	return projectPath, out
}

// validUTF8 replaces each invalid byte (not each invalid run, unlike
// strings.ToValidUTF8) with U+FFFD, matching encoding/json.
func validUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + len(s)/2)
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			b.WriteRune(utf8.RuneError)
		} else {
			b.WriteString(s[i : i+size])
		}
		i += size
	}
	return b.String()
}

// hasParent reports whether related context's parent direct hit was kept.
// RelatedTo is "<index-relative path>:<symbol>".
func hasParent(r SearchResultItem, direct []SearchResultItem) bool {
	for _, d := range direct {
		file, ok := strings.CutSuffix(r.RelatedTo, ":"+d.Symbol)
		if ok && (filepath.ToSlash(d.FilePath) == file || strings.HasSuffix(filepath.ToSlash(d.FilePath), "/"+file)) {
			return true
		}
	}
	return false
}

// fitWarnings cuts warnings max-min fairly until the mandatory header,
// warnings and footer fit, keeping each warning's beginning.
func fitWarnings(out SemanticSearchOutput, limit int, size func(SemanticSearchOutput) int) SemanticSearchOutput {
	const marker = "..."
	warnings := []*string{&out.SeedWarning, &out.StaleWarning, &out.FilteredHint}
	for range 3 {
		excess := size(out) - limit
		if excess <= 0 {
			break
		}
		lens := make([]int, len(warnings))
		sum := 0
		for i, w := range warnings {
			lens[i] = len(*w)
			sum += lens[i]
		}
		level := waterLevel(lens, max(sum-excess, 0))
		for _, w := range warnings {
			if len(*w) > level {
				*w = runePrefix(*w, max(level-len(marker), 0)) + marker
			}
		}
	}
	return out
}

// packContents splits avail bytes among snippets. Each snippet with content
// costs len+1 (its trailing newline); every snippet was sized as truncated, so
// one kept whole refunds len(truncatedAttr). Shares are max-min fair, cut to
// whole-line prefixes, and rounding leftovers are handed out in rank order.
func packContents(contents []string, avail int) ([]string, []bool) {
	n := len(contents)
	packed := make([]string, n)
	cut := make([]bool, n)
	want := make([]int, n)
	need, refund := 0, 0
	for i, c := range contents {
		if c != "" {
			want[i] = len(c) + 1
			need += want[i]
			refund += len(truncatedAttr)
		}
	}
	if need <= avail+refund {
		copy(packed, contents)
		return packed, cut
	}

	level := waterLevel(want, avail)
	used := make([]int, n)
	leftover := avail
	for i, c := range contents {
		if c == "" {
			continue
		}
		if want[i] <= level {
			packed[i], used[i] = c, want[i]
			leftover += len(truncatedAttr) - want[i]
			continue
		}
		packed[i] = contentPrefix(c, level-1)
		used[i] = contentCost(packed[i])
		cut[i] = true
		leftover -= used[i]
	}
	for i, c := range contents {
		if !cut[i] {
			continue
		}
		give := used[i] + leftover
		if want[i] <= give+len(truncatedAttr) {
			packed[i], cut[i] = c, false
			leftover = give + len(truncatedAttr) - want[i]
			continue
		}
		packed[i] = contentPrefix(c, give-1)
		used[i] = contentCost(packed[i])
		leftover = give - used[i]
	}
	return packed, cut
}

func contentCost(s string) int {
	if s == "" {
		return 0
	}
	return len(s) + 1
}

// waterLevel is the largest per-snippet share such that the sum of
// min(want, level) fits avail.
func waterLevel(want []int, avail int) int {
	sorted := slices.Sorted(slices.Values(want))
	for i, w := range sorted {
		rest := len(sorted) - i
		if w*rest > avail {
			return avail / rest
		}
		avail -= w
	}
	return math.MaxInt
}

// contentPrefix returns the longest whole-line prefix of c within limit
// bytes. A first line longer than contentFloor is cut at a rune boundary if
// at least contentFloor bytes fit; otherwise nothing useful fits.
func contentPrefix(c string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if len(c) <= limit {
		return c
	}
	if j := strings.LastIndexByte(c[:limit+1], '\n'); j > 0 {
		return c[:j]
	}
	first := strings.IndexByte(c, '\n')
	if first < 0 {
		first = len(c)
	}
	if first > contentFloor && limit >= contentFloor {
		return runePrefix(c, min(limit, first))
	}
	return ""
}

// runePrefix cuts s to at most n bytes without splitting a valid UTF-8
// sequence. Text reaching it is already valid; invalid bytes never panic.
func runePrefix(s string, n int) string {
	if n >= len(s) {
		return s
	}
	i := n
	for back := 0; i > 0 && back < utf8.UTFMax-1 && !utf8.RuneStart(s[i]); back++ {
		i--
	}
	if !utf8.RuneStart(s[i]) {
		i = n
	}
	return s[:i]
}
