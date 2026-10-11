package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ory/lumen/internal/index"
)

func lexicalSearchItems(ctx context.Context, idx *index.Indexer, root, searchPath, query string, limit int, minScore *float64, expand, summary bool, maxLines int) ([]SearchResultItem, error) {
	if minScore != nil && *minScore > 0 {
		return nil, fmt.Errorf("min_score is a cosine threshold and is unsupported for BM25; omit it or select retrieval=semantic")
	}
	prefix := ""
	if searchPath != root {
		rel, err := filepath.Rel(root, searchPath)
		if err != nil {
			return nil, err
		}
		prefix = rel
	}
	searchCtx, cancel := context.WithTimeout(ctx, defaultSearchTimeout)
	defer cancel()
	hits, err := idx.SearchLexical(searchCtx, root, query, limit, prefix, expand)
	if err != nil {
		return nil, err
	}
	items := make([]SearchResultItem, 0, len(hits))
	for _, r := range hits {
		item := SearchResultItem{FilePath: filepath.Join(root, r.FilePath), Symbol: r.Symbol, Kind: r.Kind, StartLine: r.StartLine, EndLine: r.EndLine, RelatedTo: r.RelatedTo, Relation: r.Relation}
		if r.RelatedTo == "" {
			item.Score = float32(-r.Distance)
		}
		if !summary {
			item.Content = r.Content
			if maxLines > 0 {
				lines := strings.Split(item.Content, "\n")
				if len(lines) > maxLines {
					item.Content = strings.Join(lines[:maxLines], "\n") + "\n..."
				}
			}
		}
		items = append(items, item)
	}
	return items, nil
}

// Keep direct hits in retrieval order and append related context with provenance.
func formatLexicalResults(projectPath string, out SemanticSearchOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Found %d results [retrieval=bm25; direct scores are weighted lexical ranks, not cosine similarity]", len(out.Results))
	if out.Reindexed {
		fmt.Fprintf(&b, " (indexed %d files)", out.IndexedFiles)
	}
	for _, warning := range []string{out.SeedWarning, out.StaleWarning} {
		if warning != "" {
			fmt.Fprintf(&b, "\nWarning: %s", warning)
		}
	}
	b.WriteString(":\n")
	for _, r := range out.Results {
		rel, err := filepath.Rel(projectPath, r.FilePath)
		if err != nil {
			rel = r.FilePath
		}
		fmt.Fprintf(&b, "\n<result:file filename=\"%s\">\n", xmlEscaper.Replace(rel))
		fmt.Fprintf(&b, "  <result:chunk line-start=\"%d\" line-end=\"%d\" symbol=\"%s\" kind=\"%s\"", r.StartLine, r.EndLine, xmlEscaper.Replace(r.Symbol), xmlEscaper.Replace(r.Kind))
		if r.RelatedTo != "" {
			fmt.Fprintf(&b, " related-to=\"%s\" relation=\"%s\"", xmlEscaper.Replace(r.RelatedTo), r.Relation)
		} else {
			fmt.Fprintf(&b, " score=\"%.6g\"", r.Score)
		}
		if r.truncated {
			b.WriteString(truncatedAttr)
		}
		b.WriteString(">\n")
		if r.Content != "" {
			b.WriteString(r.Content)
			b.WriteByte('\n')
		}
		b.WriteString("  </result:chunk>\n</result:file>\n")
	}
	writeBudgetFooter(&b, out.budget)
	return b.String()
}

// Preserve semantic zero scores while omitting a score for related context.
func (r SearchResultItem) MarshalJSON() ([]byte, error) {
	type plain SearchResultItem
	if r.RelatedTo == "" {
		return json.Marshal(plain(r))
	}
	return json.Marshal(struct {
		plain
		Score *float32 `json:"score,omitempty"`
	}{plain: plain(r)})
}
