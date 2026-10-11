package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ory/lumen/internal/config"
	"github.com/spf13/cobra"
)

func offlineLexicalConfig(t *testing.T) *config.ConfigService {
	t.Helper()
	t.Setenv("LUMEN_RETRIEVAL", "")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	// An invalid unrelated embedding profile must not block default startup.
	t.Setenv("LUMEN_EMBED_MODEL", "unavailable-model")
	t.Setenv("LUMEN_EMBED_DIMS", "0")
	t.Setenv("OLLAMA_HOST", "http://127.0.0.1:1")
	t.Setenv("LM_STUDIO_HOST", "http://127.0.0.1:1")
	t.Setenv("LUMEN_VECTOR_STORAGE", "not-a-vector-type")
	cfg, err := config.NewConfigService("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Retrieval() != "bm25" || newEmbedder(cfg) != nil {
		t.Fatal("default constructed embedding backend")
	}
	return cfg
}

func TestDefaultBM25MCPWithoutEmbeddingService(t *testing.T) {
	cfg := offlineLexicalConfig(t)
	root := t.TempDir()
	root, _ = filepath.EvalSymlinks(root)
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package fixture\n// duplicate payment prevention\nfunc PreventDuplicates(){reserve()}\nfunc reserve(){}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	ic := &indexerCache{cfg: cfg}
	defer ic.Close()
	result, _, err := ic.handleSemanticSearch(context.Background(), &mcp.CallToolRequest{}, SemanticSearchInput{Query: "duplicate payment", Path: root, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	for _, want := range []string{"retrieval=bm25", "PreventDuplicates", "related-to=\"main.go:PreventDuplicates\""} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q: %s", want, text)
		}
	}
	if strings.Contains(text, "symbol=\"reserve\" kind=\"function\" score=") {
		t.Fatalf("related fabricated score: %s", text)
	}
	baseline, _, err := ic.handleSemanticSearch(context.Background(), &mcp.CallToolRequest{}, SemanticSearchInput{Query: "duplicate payment", Path: root, Limit: 1, LexicalOnly: true, Summary: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(baseline.Content[0].(*mcp.TextContent).Text, "related-to=") {
		t.Fatal("baseline expanded")
	}
	if _, _, err := ic.handleHealthCheck(context.Background(), nil, HealthCheckInput{}); err != nil {
		t.Fatal(err)
	}
	status, _, err := ic.handleIndexStatus(context.Background(), nil, IndexStatusInput{Path: root})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(status.Content[0].(*mcp.TextContent).Text, "Unique lexical entries:") {
		t.Fatal("wrong status units")
	}
	// Even summary-only queries validate the revision while freshness TTL is warm.
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package fixture\nfunc Changed() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	stale, _, err := ic.handleSemanticSearch(context.Background(), &mcp.CallToolRequest{}, SemanticSearchInput{Query: "duplicate payment", Path: root, Summary: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stale.Content[0].(*mcp.TextContent).Text, "<result:chunk") {
		t.Fatal("stale summary leaked old symbol location")
	}
	minScore := 0.2
	if _, _, err := ic.handleSemanticSearch(context.Background(), &mcp.CallToolRequest{}, SemanticSearchInput{Query: "payment", Path: root, MinScore: &minScore}); err == nil {
		t.Fatal("silently applied cosine min_score")
	}
}

func TestDefaultBM25CLIAndAncestorProfile(t *testing.T) {
	cfg := offlineLexicalConfig(t)
	root := t.TempDir()
	root, _ = filepath.EvalSymlinks(root)
	child := filepath.Join(root, "src")
	if err := os.MkdirAll(child, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, "main.go"), []byte("package fixture\nfunc RefreshAccessToken() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := &cobra.Command{}
	cmd.Flags().String("retrieval", "", "")
	cmd.Flags().String("model", "", "")
	cmd.Flags().String("backend", "", "")
	cmd.Flags().Bool("force", false, "")
	if err := runIndex(cmd, []string{root}); err != nil {
		t.Fatal(err)
	}
	db := configuredDBPath(cfg, root, "lumen-bm25-v1")
	if _, err := os.Stat(db); err != nil {
		t.Fatal(err)
	}
	got, _, err := resolveIndexRootWithPath(child, "", func(project string) string { return configuredDBPath(cfg, project, "lumen-bm25-v1") })
	if err != nil || got != root {
		t.Fatalf("ancestor %q %v", got, err)
	}
	spawned := ""
	msg := generateSessionContextInternalWithDirective("search", child, func(string, string) string { return "" }, func(p string) { spawned = p })
	if spawned != "" || !strings.Contains(msg, "ready") {
		t.Fatalf("hook did not reuse lexical ancestor: %q spawned=%q", msg, spawned)
	}
	// Profiles with nondefault chunk settings must also resolve ancestors.
	t.Setenv("LUMEN_MAX_CHUNK_TOKENS", "1024")
	custom, err := config.NewConfigService("")
	if err != nil {
		t.Fatal(err)
	}
	customPath := configuredDBPath(custom, root, "")
	if err := os.MkdirAll(filepath.Dir(customPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(customPath, nil, 0644); err != nil {
		t.Fatal(err)
	}
	got = findAncestorIndexWithPath(child, func(p string) string { return configuredDBPath(custom, p, "") })
	if got != root {
		t.Fatalf("custom profile ancestor: %s", got)
	}
}

// Warnings computed before the BM25 branch must reach the agent, as they do
// in semantic output.
func TestFormatLexicalResultsKeepsWarnings(t *testing.T) {
	text := formatSearchResults("/p", SemanticSearchOutput{Retrieval: "bm25", SeedWarning: "seeded from sibling worktree", StaleWarning: "index is refreshing"})
	for _, want := range []string{"Warning: seeded from sibling worktree", "Warning: index is refreshing"} {
		if !strings.Contains(text, want) {
			t.Errorf("BM25 output lacks %q:\n%s", want, text)
		}
	}
}
