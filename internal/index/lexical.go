package index

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ory/lumen/internal/chunker"
	"github.com/ory/lumen/internal/merkle"
	"github.com/ory/lumen/internal/store"
)

// NewLexicalIndexer opens a model-independent collection without an embedder.
func NewLexicalIndexer(dsn string, maxChunkTokens int, projectPath string) (*Indexer, error) {
	s, err := store.OpenLexicalCollection(dsn)
	if err != nil {
		return nil, err
	}
	if projectPath != "" {
		if err := s.UseProject(projectPath); err != nil {
			_ = s.Close()
			return nil, err
		}
	}
	return &Indexer{store: s, chunker: chunker.NewMultiChunker(chunker.DefaultLanguages(maxChunkTokens)), maxChunkTokens: maxChunkTokens, dsn: dsn, vectorStorage: "lexical", projectPath: projectPath}, nil
}

func (idx *Indexer) Lexical() bool { return idx.emb == nil }

func (idx *Indexer) indexLexicalWithTree(ctx context.Context, projectDir string, force bool, tree *merkle.Tree, progress ProgressFunc) (Stats, error) {
	stats := Stats{TotalFiles: len(tree.Files)}
	old, err := idx.store.GetFileHashes()
	if err != nil {
		return stats, err
	}
	added, removed, modified := merkle.Diff(&merkle.Tree{Files: old}, tree)
	stats.FilesAdded, stats.FilesRemoved, stats.FilesModified = len(added), len(removed), len(modified)
	paths := append(added, modified...)
	if force {
		paths = nil
		for path := range tree.Files {
			paths = append(paths, path)
		}
	}
	slices.Sort(paths)
	slices.Sort(removed)
	stats.FilesChanged = len(paths) + len(removed)
	// Files that could not be read are not stored. Leaving the tree unstamped
	// makes the next pass retry them; unchunkable content is deterministic and
	// is stored as an empty revision instead.
	retry := false
	for _, path := range removed {
		if err := idx.store.DeleteFileChunks(path); err != nil {
			return stats, err
		}
	}
	for i, path := range paths {
		if err := ctx.Err(); err != nil {
			return stats, err
		}
		if progress != nil {
			progress(i, len(paths), "Indexing "+path)
		}
		if !force {
			reused, err := idx.store.AttachExistingFileRevision(path, tree.Files[path])
			if err != nil {
				return stats, err
			}
			if reused {
				stats.IndexedFiles++
				continue
			}
		}
		data, err := os.ReadFile(filepath.Join(projectDir, path))
		if err != nil {
			if os.IsPermission(err) {
				stats.FilesSkipped++
				retry = true
				if idx.logger != nil {
					idx.logger.Warn("skipping unreadable file", "path", path, "error", err)
				}
				continue
			}
			return stats, err
		}
		// The scan and read may race an editor. Never attach chunks to the wrong revision.
		hash := fmt.Sprintf("%x", sha256.Sum256(data))
		if hash != tree.Files[path] {
			return stats, fmt.Errorf("file changed while indexing: %s; retry indexing", path)
		}
		if isBinaryContent(data) {
			if err := idx.store.DeleteFileChunks(path); err != nil {
				return stats, err
			}
			continue
		}
		chunks, err := idx.chunker.Chunk(path, data)
		if err != nil {
			stats.FilesSkipped++
			if idx.logger != nil {
				idx.logger.Warn("skipping unchunkable file", "path", path, "error", err)
			}
			// Record the revision without chunks so an unchanged tree stays fresh
			// and stale chunks from an earlier revision are not attached.
			if _, err := idx.store.StoreFileRevision(path, hash, nil, nil); err != nil {
				return stats, err
			}
			continue
		}
		// Keep symbol-sized lexical documents; no embedding context window is required.
		created, err := idx.store.StoreFileRevision(path, hash, chunks, nil)
		if err != nil {
			return stats, err
		}
		if created {
			stats.ChunksCreated += len(chunks)
		}
		stats.IndexedFiles++
	}
	for k, v := range map[string]string{"retrieval": "bm25", "project_path": projectDir, "last_indexed_at": time.Now().UTC().Format(time.RFC3339), "total_files": strconv.Itoa(stats.TotalFiles), "vector_storage": "lexical"} {
		if err := idx.store.SetMeta(k, v); err != nil {
			return stats, err
		}
	}
	// Do not stamp a scan with unread files as fresh.
	if !retry {
		if err := idx.store.SetMeta("root_hash", tree.RootHash); err != nil {
			return stats, err
		}
	}
	if progress != nil {
		progress(len(paths), len(paths), "Lexical indexing complete")
	}
	return stats, nil
}

// LexicalResult contains a validated current-worktree snippet. RelatedTo marks
// explanatory context and has no relevance score of its own.
type LexicalResult struct {
	store.SearchResult
	Content   string
	RelatedTo string
	Relation  string
}

// SearchLexical returns direct matches first, then at most two one-hop related
// snippets. The related context budget is hard-capped at 6000 UTF-8 bytes
// (1500 estimated tokens at four bytes/token), without truncating symbols.
func (idx *Indexer) SearchLexical(ctx context.Context, projectDir, query string, limit int, pathPrefix string, expand bool) ([]LexicalResult, error) {
	release, err := idx.lockProject(projectDir)
	if err != nil {
		return nil, err
	}
	defer release()
	if limit <= 0 {
		return nil, nil
	}
	hits, err := idx.store.SearchLexical(ctx, query, limit*3, pathPrefix)
	if err != nil {
		return nil, err
	}
	type fileData struct {
		hash  string
		lines []string
	}
	files := map[string]fileData{}
	// Search-invariant lookups are resolved at most once per call.
	resolveRoot := sync.OnceValues(func() (string, error) { return filepath.EvalSymlinks(projectDir) })
	goModule := goModulePath(os.ReadFile, projectDir)
	read := func(r store.SearchResult) (string, bool) {
		f, ok := files[r.FilePath]
		if !ok {
			path := filepath.Join(projectDir, r.FilePath)
			info, e := os.Lstat(path)
			if e != nil || !info.Mode().IsRegular() {
				files[r.FilePath] = fileData{}
				return "", false
			}
			resolved, e := filepath.EvalSymlinks(path)
			if e != nil {
				return "", false
			}
			root, e := resolveRoot()
			if e != nil {
				return "", false
			}
			rel, e := filepath.Rel(root, resolved)
			if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return "", false
			}
			data, e := os.ReadFile(path)
			if e != nil {
				files[r.FilePath] = fileData{}
				return "", false
			}
			f = fileData{fmt.Sprintf("%x", sha256.Sum256(data)), strings.Split(string(data), "\n")}
			files[r.FilePath] = f
		}
		if f.hash == "" || f.hash != r.ContentHash || r.StartLine < 1 || r.EndLine < r.StartLine || r.EndLine > len(f.lines) {
			return "", false
		}
		return strings.Join(f.lines[r.StartLine-1:r.EndLine], "\n"), true
	}
	key := func(r store.SearchResult) string { return fmt.Sprintf("%s:%d:%d", r.FilePath, r.StartLine, r.EndLine) }
	seen := map[string]bool{}
	var out []LexicalResult
	for _, r := range hits {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if seen[key(r)] {
			continue
		}
		content, ok := read(r)
		if !ok {
			continue
		}
		seen[key(r)] = true
		out = append(out, LexicalResult{SearchResult: r, Content: content})
		if len(out) == limit {
			break
		}
	}
	if !expand || len(out) == 0 {
		return out, nil
	}
	names := map[string]bool{}
	for _, parent := range out[:min(3, len(out))] {
		for _, ref := range parent.References[:min(32, len(parent.References))] {
			name := ref
			if dot := strings.LastIndexByte(name, '.'); dot >= 0 {
				name = name[dot+1:]
			}
			names[name] = true
			for _, imp := range parent.Imports {
				local, target, ok := strings.Cut(imp, "=")
				if !ok || local != ref {
					continue
				}
				_, symbol, ok := strings.Cut(target, "#")
				if ok && symbol != "default" {
					names[symbol] = true
				}
			}
		}
	}
	symbols := make([]string, 0, len(names))
	for name := range names {
		symbols = append(symbols, name)
	}
	slices.Sort(symbols)
	if len(symbols) > 96 {
		return out, nil
	}
	all, err := idx.store.LookupLexicalSymbols(ctx, symbols, pathPrefix)
	if err != nil {
		return nil, err
	}
	if len(all) > 256 {
		return out, nil
	}
	bySymbol := map[string][]store.SearchResult{}
	for _, r := range all {
		bySymbol[r.Symbol] = append(bySymbol[r.Symbol], r)
	}
	direct := append([]LexicalResult(nil), out...)
	budget := 6000
	extras := 0
	for _, parent := range direct[:min(3, len(direct))] {
		for _, ref := range parent.References[:min(32, len(parent.References))] {
			candidates := resolveLexicalReference(projectDir, goModule, parent, ref, bySymbol)

			if len(candidates) != 1 {
				continue
			}
			r := candidates[0]
			if seen[key(r)] {
				continue
			}
			content, ok := read(r)
			if !ok || len(content) > budget {
				continue
			}
			out = append(out, LexicalResult{SearchResult: r, Content: content, RelatedTo: parent.FilePath + ":" + parent.Symbol, Relation: "direct_call"})
			seen[key(r)] = true
			budget -= len(content)
			extras++
			if extras == 2 {
				return out, nil
			}
		}
	}
	return out, nil
}

// Resolve only direct, syntax-bound local targets. Module aliases, reexports,
// implicit receiver dispatch and nonlocal dependencies are deliberately skipped.
func resolveLexicalReference(root string, goModule func() string, parent LexicalResult, ref string, bySymbol map[string][]store.SearchResult) []store.SearchResult {
	var candidates []store.SearchResult
	for _, r := range bySymbol[ref] {
		if r.FilePath == parent.FilePath && r.Kind == "function" {
			candidates = append(candidates, r)
		}
	}
	if len(candidates) > 0 {
		return candidates
	}
	for _, imp := range parent.Imports {
		local, target, ok := strings.Cut(imp, "=")
		if !ok {
			continue
		}
		if spec, symbol, js := strings.Cut(target, "#"); js {
			if local != ref || symbol == "default" || (!strings.HasPrefix(spec, "./") && !strings.HasPrefix(spec, "../")) {
				continue
			}
			base := filepath.Clean(filepath.Join(filepath.Dir(parent.FilePath), filepath.FromSlash(spec)))
			if base == ".." || strings.HasPrefix(base, ".."+string(filepath.Separator)) {
				continue
			}
			// Require exactly one physical candidate among conventional module paths.
			paths := []string{base}
			if filepath.Ext(base) == "" {
				for _, ext := range []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".mts"} {
					paths = append(paths, base+ext, filepath.Join(base, "index"+ext))
				}
			}
			existing := ""
			ambiguous := false
			for _, path := range paths {
				info, err := os.Stat(filepath.Join(root, path))
				if err == nil && info.Mode().IsRegular() {
					if existing != "" {
						ambiguous = true
						break
					}
					existing = path
				}
			}
			if ambiguous || existing == "" {
				continue
			}
			for _, r := range bySymbol[symbol] {
				if r.FilePath == existing && r.Kind == "function" && r.Exported {
					candidates = append(candidates, r)
				}
			}
		} else if filepath.Ext(parent.FilePath) == ".go" && strings.HasPrefix(ref, local+".") {
			symbol := strings.TrimPrefix(ref, local+".")
			module := goModule()
			if module == "" || (target != module && !strings.HasPrefix(target, module+"/")) {
				continue
			}
			dir := strings.TrimPrefix(strings.TrimPrefix(target, module), "/")
			if dir == "" {
				dir = "."
			}
			for _, r := range bySymbol[symbol] {
				if filepath.Dir(r.FilePath) == filepath.FromSlash(dir) && r.Kind == "function" && r.Exported && !strings.HasSuffix(r.FilePath, "_test.go") {
					candidates = append(candidates, r)
				}
			}
		}
	}
	return candidates
}

// goModulePath returns a lazy, memoized reader of the module path declared in
// root/go.mod. It returns "" when go.mod is missing or declares no module.
func goModulePath(readFile func(string) ([]byte, error), root string) func() string {
	return sync.OnceValue(func() string {
		mod, err := readFile(filepath.Join(root, "go.mod"))
		if err != nil {
			return ""
		}
		for _, line := range strings.Split(string(mod), "\n") {
			fields := strings.Fields(line)
			if len(fields) == 2 && fields[0] == "module" {
				return strings.Trim(fields[1], "\"")
			}
		}
		return ""
	})
}
