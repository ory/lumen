package index

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ory/lumen/internal/config"
	"github.com/ory/lumen/internal/embedder"
)

// These fixtures and judgments deliberately include an unbridgeable vocabulary
// mismatch. This is a reproducible smoke evaluation, not a representative corpus.
var retrievalQueries = []struct{ name, query, target, contextTarget string }{
	{"exact_symbol", "refreshAccessToken", "refreshAccessToken", ""},
	{"path", "session/refresh.go", "refreshAccessToken", ""},
	{"comment", "prevent duplicate payment processing", "processPayment", ""},
	{"imported_helper", "processPayment", "processPayment", "reserveSlot"},
	{"vocabulary_mismatch", "stop repeated invoice charges", "processPayment", ""},
}

func retrievalCorpus(t testing.TB) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"session/refresh.go":     "package session\n// refreshAccessToken renews expired credentials.\nfunc refreshAccessToken() string { return \"credential\" }\n",
		"billing/payment.ts":     "import { reserveSlot as claim } from './reservation';\n// Prevent duplicate payment processing.\nexport function processPayment(id: string) { return claim(id); }\n",
		"billing/reservation.ts": "export function reserveSlot(key: string) { return ledger.insertIfAbsent(key); }\n",
		"ui/theme.ts":            "export function chooseTheme() { return 'dark'; }\n",
		"cache/entries.go":       "package cache\nfunc ClearEntries() {}\n",
	}
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func retrievalLexicalIndex(t testing.TB) (*Indexer, string, time.Duration, int64) {
	t.Helper()
	root := retrievalCorpus(t)
	idx, err := NewLexicalIndexer(filepath.Join(t.TempDir(), "index.db"), 512, root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = idx.Close() })
	start := time.Now()
	if _, err = idx.Index(context.Background(), root, false, nil); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	status, err := idx.Status(root)
	if err != nil {
		t.Fatal(err)
	}
	return idx, root, elapsed, status.DatabaseBytes
}

func TestRetrievalEvaluation(t *testing.T) {
	idx, root, build, bytes := retrievalLexicalIndex(t)
	t.Logf("corpus_files=5 queries=%d index_build=%s database_bytes=%d (SQLite pages, excludes WAL)", len(retrievalQueries), build, bytes)
	for _, expand := range []bool{false, true} {
		hits, rr, contextHits, extras := 0, 0.0, 0, 0
		var queryTime time.Duration
		for _, q := range retrievalQueries {
			start := time.Now()
			results, err := idx.SearchLexical(context.Background(), root, q.query, 3, "", expand)
			queryTime += time.Since(start)
			if err != nil {
				t.Fatal(err)
			}
			rank, targetRank, extraBytes, relatedCount := 0, 0, 0, 0
			foundContext := false
			for _, r := range results {
				if r.RelatedTo == "" {
					rank++
					if r.Symbol == q.target && targetRank == 0 {
						targetRank = rank
					}
				} else {
					extraBytes += len(r.Content)
					relatedCount++
				}
				if q.contextTarget != "" && r.Symbol == q.contextTarget {
					foundContext = true
				}
			}
			if targetRank > 0 {
				hits++
				rr += 1.0 / float64(targetRank)
			}
			if foundContext {
				contextHits++
			}
			if extraBytes > 6000 || relatedCount > 2 {
				t.Fatalf("context budget exceeded: %d bytes, %d snippets", extraBytes, relatedCount)
			}
			extras += extraBytes
			// The vocabulary-mismatch probe documents a known BM25 limit; every
			// other case must hit, and expansion must surface its context.
			if q.name != "vocabulary_mismatch" && (targetRank == 0 || targetRank > 3) {
				t.Errorf("expand=%t case=%s: target %s not in top 3", expand, q.name, q.target)
			}
			if expand && q.contextTarget != "" && !foundContext {
				t.Errorf("case=%s: context target %s not surfaced", q.name, q.contextTarget)
			}
			t.Logf("expand=%t case=%s direct_rank=%d context_target_found=%t extra_bytes=%d", expand, q.name, targetRank, foundContext, extraBytes)
		}
		t.Logf("expand=%t direct_hit@3=%.3f direct_MRR@3=%.3f context_target_recall=%.3f mean_query=%s extra_bytes=%d", expand, float64(hits)/float64(len(retrievalQueries)), rr/float64(len(retrievalQueries)), float64(contextHits), queryTime/time.Duration(len(retrievalQueries)), extras)
	}
}

func BenchmarkRetrieval(b *testing.B) {
	idx, root, build, bytes := retrievalLexicalIndex(b)
	for _, expand := range []bool{false, true} {
		b.Run(fmt.Sprintf("expand=%t", expand), func(b *testing.B) {
			var extraBytes int
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				q := retrievalQueries[n%len(retrievalQueries)]
				results, err := idx.SearchLexical(context.Background(), root, q.query, 3, "", expand)
				if err != nil {
					b.Fatal(err)
				}
				for _, r := range results {
					if r.RelatedTo != "" {
						extraBytes += len(r.Content)
					}
				}
			}
			b.StopTimer()
			b.ReportMetric(float64(bytes), "index-bytes")
			b.ReportMetric(float64(build.Nanoseconds())/1e6, "build-ms")
			b.ReportMetric(float64(extraBytes)/float64(b.N), "extra-bytes/op")
		})
	}
}

// Opt-in only: uses the real configured model, never synthetic vectors.
func TestRetrievalEvaluationSemantic(t *testing.T) {
	if os.Getenv("LUMEN_BENCH_SEMANTIC") != "1" {
		t.Skip("semantic comparison unavailable: set LUMEN_BENCH_SEMANTIC=1 with a configured embedding service")
	}
	cfg, err := config.NewConfigService(config.DefaultConfigPath(), config.WithRetrieval("semantic"))
	if err != nil {
		t.Fatal(err)
	}
	defer cfg.Stop()
	emb := embedder.NewFailoverEmbedder(cfg)
	root := retrievalCorpus(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	probe, err := emb.Embed(ctx, []string{"retrieval evaluation"})
	if err != nil {
		t.Skipf("semantic comparison unavailable: %v", err)
	}
	if len(probe) != 1 {
		t.Fatal("embedding service returned an invalid probe")
	}
	idx, err := NewIndexerForProject(filepath.Join(t.TempDir(), "index.db"), emb, 512, cfg.VectorStorage(), root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = idx.Close() }()
	start := time.Now()
	if _, err := idx.Index(ctx, root, false, nil); err != nil {
		t.Fatal(err)
	}
	build := time.Since(start)
	status, err := idx.Status(root)
	if err != nil {
		t.Fatal(err)
	}
	hits, rr, contextHits := 0, 0.0, 0
	start = time.Now()
	for _, q := range retrievalQueries {
		vectors, err := emb.Embed(ctx, []string{q.query})
		if err != nil {
			t.Fatal(err)
		}
		results, err := idx.Search(ctx, root, vectors[0], 3, 0, "")
		if err != nil {
			t.Fatal(err)
		}
		for n, r := range results {
			if r.Symbol == q.target {
				hits++
				rr += 1.0 / float64(n+1)
				break
			}
		}
		for _, r := range results {
			if q.contextTarget != "" && r.Symbol == q.contextTarget {
				contextHits++
				break
			}
		}
	}
	t.Logf("semantic model=%s dimensions=%d index_build=%s database_bytes=%d direct_hit@3=%.3f direct_MRR@3=%.3f context_target_recall=%.3f mean_query_including_embedding=%s extra_bytes=0", emb.ModelName(), emb.Dimensions(), build, status.DatabaseBytes, float64(hits)/float64(len(retrievalQueries)), rr/float64(len(retrievalQueries)), float64(contextHits), time.Since(start)/time.Duration(len(retrievalQueries)))
}
