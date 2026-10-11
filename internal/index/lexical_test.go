package index

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ory/lumen/internal/merkle"
)

func lexicalFixture(t *testing.T, files map[string]string) (*Indexer, string) {
	t.Helper()
	root := t.TempDir()
	for path, body := range files {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	idx, err := NewLexicalIndexer(filepath.Join(t.TempDir(), "index.db"), 512, root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = idx.Close() })
	if _, err := idx.Index(context.Background(), root, false, nil); err != nil {
		t.Fatal(err)
	}
	return idx, root
}

func TestLexicalFreshnessAndBaseline(t *testing.T) {
	idx, root := lexicalFixture(t, map[string]string{"main.go": "package fixture\n// Prevent duplicate payment processing.\nfunc ProcessPayment() { reserve() }\nfunc reserve() {}\n"})
	ctx := context.Background()
	direct, err := idx.SearchLexical(ctx, root, "duplicate payment", 1, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(direct) != 1 || direct[0].Symbol != "ProcessPayment" {
		t.Fatalf("direct: %+v", direct)
	}
	expanded, err := idx.SearchLexical(ctx, root, "duplicate payment", 1, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(expanded) != 2 || expanded[1].Symbol != "reserve" || expanded[1].RelatedTo != "main.go:ProcessPayment" {
		t.Fatalf("expanded: %+v", expanded)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package fixture\nfunc Different() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	stale, err := idx.SearchLexical(ctx, root, "duplicate payment", 1, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 0 {
		t.Fatalf("stale snippets returned: %+v", stale)
	}
	if err := os.Remove(filepath.Join(root, "main.go")); err != nil {
		t.Fatal(err)
	}
	stale, err = idx.SearchLexical(ctx, root, "duplicate payment", 1, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 0 {
		t.Fatalf("deleted snippets returned: %+v", stale)
	}
}

func TestLexicalLocalImportsAndFilters(t *testing.T) {
	for _, language := range []string{"go", "typescript"} {
		t.Run(language, func(t *testing.T) {
			var files map[string]string
			if language == "go" {
				files = map[string]string{"go.mod": "module example.test/payments\n", "main.go": "package payments\nimport db \"example.test/payments/storage\"\n// prevent duplicate payment processing\nfunc ProcessPayment() { db.Reserve() }\n", "storage/store.go": "package storage\nfunc Reserve() {}\n"}
			} else {
				files = map[string]string{"main.ts": "import { reserve as save } from './storage/store';\n// prevent duplicate payment processing\nexport function ProcessPayment() { save(); }\n", "storage/store.ts": "export function reserve() {}\n"}
			}
			idx, root := lexicalFixture(t, files)
			hits, err := idx.SearchLexical(context.Background(), root, "duplicate payment", 1, "", true)
			if err != nil {
				t.Fatal(err)
			}
			if len(hits) != 2 || hits[1].RelatedTo == "" || !strings.HasPrefix(hits[1].FilePath, "storage/") {
				t.Fatalf("import expansion: %+v", hits)
			}
			hits, err = idx.SearchLexical(context.Background(), root, "duplicate payment", 1, "main", true)
			if err != nil {
				t.Fatal(err)
			}
			if len(hits) != 0 {
				t.Fatalf("path component isolation: %+v", hits)
			}
		})
	}
}

func TestLexicalExpansionBudgetCyclesAndAmbiguity(t *testing.T) {
	source := "package fixture\n// transaction prevent duplicates\nfunc Entry(){ first(); second(); third(); huge(); ambiguous() }\nfunc first(){Entry()}\nfunc second(){}\nfunc third(){}\nfunc ambiguous(){}\nfunc ambiguous(){}\nfunc huge(){\n/*" + strings.Repeat("x", 6100) + "*/\n}\n"
	idx, root := lexicalFixture(t, map[string]string{"main.go": source})
	hits, err := idx.SearchLexical(context.Background(), root, "transaction duplicates", 1, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 3 {
		t.Fatalf("wanted direct+2 related: %+v", hits)
	}
	bytes := 0
	for _, r := range hits[1:] {
		bytes += len(r.Content)
		if r.Symbol == "ambiguous" || r.Symbol == "huge" || r.Symbol == "Entry" {
			t.Fatalf("unsafe expansion: %+v", r)
		}
	}
	if bytes > 6000 {
		t.Fatal("exceeded hard budget")
	}
}

func TestLexicalDoesNotInventExportOrShadowedEdges(t *testing.T) {
	cases := map[string]map[string]string{
		"module import shadow":   {"main.py": "def reserve():\n    pass\n\nfrom other import reserve\n# duplicate payment prevention\ndef pay():\n    reserve()\n"},
		"export alias collision": {"main.ts": "import {reserve} from './store';\n// duplicate payment prevention\nexport function pay(){ reserve(); }\n", "store.ts": "function reserve() {}\nfunction actual() {}\nexport {actual as reserve};\n"},
		"reexport collision":     {"main.ts": "import {reserve} from './store';\n// duplicate payment prevention\nexport function pay(){ reserve(); }\n", "store.ts": "function reserve() {}\nexport {reserve} from './other';\n", "other.ts": "export function reserve() {}\n"},
		"catch shadow":           {"main.ts": "function reserve() {}\n// duplicate payment prevention\nexport function pay(){ try { throw new Error(); } catch(reserve) { reserve(); } }\n"},
		"local import shadow":    {"main.py": "def reserve():\n    pass\n\n# duplicate payment prevention\ndef pay():\n    from other import reserve\n    reserve()\n"},
	}
	for name, files := range cases {
		t.Run(name, func(t *testing.T) {
			idx, root := lexicalFixture(t, files)
			hits, err := idx.SearchLexical(context.Background(), root, "duplicate payment prevention", 1, "", true)
			if err != nil {
				t.Fatal(err)
			}
			if len(hits) != 1 || hits[0].RelatedTo != "" {
				t.Fatalf("invented edge: %+v", hits)
			}
		})
	}
}

func TestLexicalUnchunkableFileStaysFresh(t *testing.T) {
	idx, root := lexicalFixture(t, map[string]string{
		"main.go":   "package fixture\nfunc Valid() {}\n",
		"broken.go": "package fixture\nfunc (\n",
	})
	fresh, err := idx.IsFresh(root)
	if err != nil {
		t.Fatal(err)
	}
	if !fresh {
		t.Fatal("index with one unchunkable file must be stamped fresh")
	}
	reindexed, _, err := idx.EnsureFresh(context.Background(), root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if reindexed {
		t.Fatal("unchanged tree with an unchunkable file was reindexed")
	}
	// Fixing the file changes its content hash, so it is indexed again.
	if err := os.WriteFile(filepath.Join(root, "broken.go"), []byte("package fixture\nfunc Repaired() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if reindexed, _, err = idx.EnsureFresh(context.Background(), root, nil); err != nil || !reindexed {
		t.Fatalf("repaired file not reindexed: %v %v", reindexed, err)
	}
	got, err := idx.SearchLexical(context.Background(), root, "Repaired", 1, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Symbol != "Repaired" {
		t.Fatalf("repaired symbol: %+v", got)
	}
}

func TestLexicalSkipsFileThatBecomesUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores file permissions")
	}
	idx, root := lexicalFixture(t, map[string]string{"main.go": "package fixture\nfunc Valid() {}\n"})
	locked := filepath.Join(root, "locked.go")
	if err := os.WriteFile(locked, []byte("package fixture\nfunc Locked() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	tree, err := merkle.BuildTree(root, makeSkip(root))
	if err != nil {
		t.Fatal(err)
	}
	// Simulate permissions changing between the scan and the read.
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0644) })
	stats, err := idx.indexLexicalWithTree(context.Background(), root, false, tree, nil)
	if err != nil {
		t.Fatalf("unreadable file aborted lexical indexing: %v", err)
	}
	if stats.FilesSkipped != 1 {
		t.Fatalf("FilesSkipped = %d, want 1", stats.FilesSkipped)
	}
	// The skipped file was never stored, so this tree must not be stamped fresh:
	// once it is readable again the next pass retries it.
	if stamped, err := idx.store.GetMeta("root_hash"); err != nil || stamped == tree.RootHash {
		t.Fatalf("tree with a transiently unreadable file stamped fresh: %q %v", stamped, err)
	}
	if err := os.Chmod(locked, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := idx.EnsureFresh(context.Background(), root, nil); err != nil {
		t.Fatal(err)
	}
	got, err := idx.SearchLexical(context.Background(), root, "Locked", 1, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Symbol != "Locked" {
		t.Fatalf("readable-again file not indexed: %+v", got)
	}
}

func TestGoModulePathReadsOnce(t *testing.T) {
	reads := 0
	module := goModulePath(func(string) ([]byte, error) {
		reads++
		return []byte("// comment\nmodule \"example.test/payments\"\n"), nil
	}, "/repo")
	for range 3 {
		if got := module(); got != "example.test/payments" {
			t.Fatalf("module = %q", got)
		}
	}
	if reads != 1 {
		t.Fatalf("go.mod read %d times, want 1", reads)
	}
	missing := goModulePath(func(string) ([]byte, error) { return nil, os.ErrNotExist }, "/repo")
	if got := missing(); got != "" {
		t.Fatalf("missing go.mod module = %q", got)
	}
}

func TestLexicalSplitsOversizedChunks(t *testing.T) {
	// A single generated declaration thousands of lines long must not become
	// one BM25 document: the MCP search result would exceed the agent's tool
	// output budget (observed: 9,411-line chunk, 763k-char response).
	var body strings.Builder
	body.WriteString("package fixture\n\nvar generated = `\n")
	for i := 0; i < 3000; i++ {
		body.WriteString("// generated line with some stable filler text here\n")
	}
	body.WriteString("`\n")
	idx, _ := lexicalFixture(t, map[string]string{"gen.go": body.String()})
	chunks, err := idx.store.ListProjectChunks("")
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected the oversized declaration to be split, got %d chunk(s)", len(chunks))
	}
	for _, c := range chunks {
		if span := c.EndLine - c.StartLine + 1; span > 100 {
			t.Errorf("chunk %s:%d-%d spans %d lines; want <= 100 (maxChunkTokens=512)", c.FilePath, c.StartLine, c.EndLine, span)
		}
	}
}
