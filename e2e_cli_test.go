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

//go:build e2e

package main

import (
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"
	"github.com/ory/lumen/internal/config"
)

// gitSampleProject copies testdata/sample-project into a temp directory and
// initialises a git repo so that RepoRoot resolves to the temp dir itself
// (not the parent lumen repo).
func gitSampleProject(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	// Resolve symlinks so the path matches what the CLI computes via
	// git rev-parse --show-toplevel + filepath.EvalSymlinks (e.g. on macOS
	// /tmp → /private/tmp).
	if resolved, err := filepath.EvalSymlinks(tmp); err == nil {
		tmp = resolved
	}
	copyDir(t, sampleProjectPath(t), tmp)
	for _, args := range [][]string{
		{"init"},
		{"add", "."},
		{"commit", "-m", "init", "--allow-empty"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmp
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
	return tmp
}

// runCLI runs the lumen binary with the given args and returns stdout, stderr, and any error.
func runCLI(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	dataHome := t.TempDir()
	return runCLIWithDataHome(t, dataHome, args...)
}

// runCLIWithDataHome runs the lumen binary using a specific XDG_DATA_HOME.
func runCLIWithDataHome(t *testing.T, dataHome string, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		ollamaHost = "http://localhost:11434"
	}

	cmd := exec.Command(serverBinary, args...)
	cmd.Env = []string{
		"OLLAMA_HOST=" + ollamaHost,
		"LUMEN_EMBED_MODEL=all-minilm",
		"XDG_DATA_HOME=" + dataHome,
		"HOME=" + os.Getenv("HOME"),
		"PATH=" + os.Getenv("PATH"),
	}

	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

func TestE2E_CLI_IndexIdempotent(t *testing.T) {
	t.Parallel()
	projectPath := gitSampleProject(t)
	dataHome := t.TempDir()

	// First index.
	_, _, err := runCLIWithDataHome(t, dataHome, "index", projectPath)
	if err != nil {
		t.Fatalf("first index failed: %v", err)
	}

	// Second index (no changes) should say up to date.
	stdout, _, err := runCLIWithDataHome(t, dataHome, "index", projectPath)
	if err != nil {
		t.Fatalf("second index failed: %v", err)
	}
	if !strings.Contains(stdout, "up to date") {
		t.Errorf("expected 'up to date' on second index, got: %s", stdout)
	}
}

func TestE2E_CLI_IndexForceReindex(t *testing.T) {
	t.Parallel()
	projectPath := gitSampleProject(t)
	dataHome := t.TempDir()

	// First index.
	_, _, err := runCLIWithDataHome(t, dataHome, "index", projectPath)
	if err != nil {
		t.Fatalf("first index failed: %v", err)
	}

	// Force reindex should re-index even with no changes.
	stdout, _, err := runCLIWithDataHome(t, dataHome, "index", "--force", projectPath)
	if err != nil {
		t.Fatalf("force index failed: %v", err)
	}
	if !strings.Contains(stdout, "Done.") {
		t.Errorf("expected 'Done.' on force reindex, got: %s", stdout)
	}
	if strings.Contains(stdout, "up to date") {
		t.Error("force reindex should not say 'up to date'")
	}
}

// openIndexDB opens the SQLite index database for a given project path and dataHome.
func openIndexDB(t *testing.T, dataHome, projectPath string) *sql.DB {
	t.Helper()
	sqlite_vec.Auto()
	dbPath := config.DBPathForProjectBase(dataHome, projectPath, "all-minilm")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open index db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestE2E_CLI_SQLVerifySchema(t *testing.T) {
	t.Parallel()
	projectPath := gitSampleProject(t)
	dataHome := t.TempDir()

	// Index first.
	_, stderr, err := runCLIWithDataHome(t, dataHome, "index", projectPath)
	if err != nil {
		t.Fatalf("index failed: %v\nstderr: %s", err, stderr)
	}

	db := openIndexDB(t, dataHome, projectPath)

	// --- Verify tables exist ---
	expectedTables := []string{"files", "chunks", "project_meta", "vec_chunks"}
	for _, table := range expectedTables {
		var count int
		err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name = ?", table).Scan(&count)
		if err != nil {
			t.Fatalf("query sqlite_master for %s: %v", table, err)
		}
		if count == 0 {
			t.Errorf("expected table %q to exist", table)
		}
	}

	// --- Verify files table ---
	var fileCount int
	if err := db.QueryRow("SELECT count(*) FROM files").Scan(&fileCount); err != nil {
		t.Fatalf("count files: %v", err)
	}
	if fileCount != 6 {
		t.Errorf("expected 6 files, got %d", fileCount)
	}

	// All file paths should end in .go and have valid hashes.
	rows, err := db.Query("SELECT path, hash FROM files ORDER BY path")
	if err != nil {
		t.Fatalf("query files: %v", err)
	}
	defer rows.Close()
	var filePaths []string
	for rows.Next() {
		var path, hash string
		if err := rows.Scan(&path, &hash); err != nil {
			t.Fatalf("scan file row: %v", err)
		}
		filePaths = append(filePaths, path)
		if !strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, ".svelte") {
			t.Errorf("file path should end in .go or .svelte: %s", path)
		}
		if len(hash) != 64 { // SHA-256 hex
			t.Errorf("file hash should be 64 hex chars, got %d: %s", len(hash), hash)
		}
	}

	// Check expected filenames are present.
	joined := strings.Join(filePaths, "\n")
	for _, name := range []string{"auth.go", "config.go", "database.go", "handler.go", "models.go"} {
		if !strings.Contains(joined, name) {
			t.Errorf("expected %s in file paths", name)
		}
	}

	// --- Verify chunks table ---
	var chunkCount int
	if err := db.QueryRow("SELECT count(*) FROM chunks").Scan(&chunkCount); err != nil {
		t.Fatalf("count chunks: %v", err)
	}
	if chunkCount < 15 {
		t.Errorf("expected at least 15 chunks (fixture has ~20), got %d", chunkCount)
	}

	// Every chunk should reference a valid file and have sensible fields.
	chunkRows, err := db.Query(`
		SELECT c.id, c.file_path, c.symbol, c.kind, c.start_line, c.end_line
		FROM chunks c
		JOIN files f ON c.file_path = f.path
		ORDER BY c.file_path, c.start_line
	`)
	if err != nil {
		t.Fatalf("query chunks: %v", err)
	}
	defer chunkRows.Close()

	kindCounts := make(map[string]int)
	for chunkRows.Next() {
		var id, filePath, symbol, kind string
		var startLine, endLine int
		if err := chunkRows.Scan(&id, &filePath, &symbol, &kind, &startLine, &endLine); err != nil {
			t.Fatalf("scan chunk row: %v", err)
		}
		if id == "" {
			t.Error("chunk id should not be empty")
		}
		if symbol == "" {
			t.Error("chunk symbol should not be empty")
		}
		if !validChunkKinds[kind] {
			t.Errorf("invalid chunk kind %q for symbol %s", kind, symbol)
		}
		if startLine <= 0 {
			t.Errorf("chunk %s: start_line should be > 0, got %d", symbol, startLine)
		}
		if endLine < startLine {
			t.Errorf("chunk %s: end_line (%d) < start_line (%d)", symbol, endLine, startLine)
		}
		kindCounts[kind]++
	}

	// Should have a mix of kinds.
	if kindCounts["function"] == 0 {
		t.Error("expected at least one function chunk")
	}
	if kindCounts["type"] == 0 {
		t.Error("expected at least one type chunk")
	}
	if kindCounts["interface"] == 0 {
		t.Error("expected at least one interface chunk")
	}

	// --- Verify vec_chunks has same count as chunks ---
	var vecCount int
	if err := db.QueryRow("SELECT count(*) FROM vec_chunks").Scan(&vecCount); err != nil {
		t.Fatalf("count vec_chunks: %v", err)
	}
	if vecCount != chunkCount {
		t.Errorf("vec_chunks count (%d) should match chunks count (%d)", vecCount, chunkCount)
	}

	// --- Verify project_meta ---
	meta := make(map[string]string)
	metaRows, err := db.Query("SELECT key, value FROM project_meta")
	if err != nil {
		t.Fatalf("query project_meta: %v", err)
	}
	defer metaRows.Close()
	for metaRows.Next() {
		var key, value string
		if err := metaRows.Scan(&key, &value); err != nil {
			t.Fatalf("scan meta row: %v", err)
		}
		meta[key] = value
	}

	// root_hash should be a SHA-256 hex string.
	if rh, ok := meta["root_hash"]; !ok {
		t.Error("project_meta missing root_hash")
	} else if len(rh) != 64 {
		t.Errorf("root_hash should be 64 hex chars, got %d", len(rh))
	}

	// vec_dimensions should be "384" (all-minilm).
	if vd, ok := meta["vec_dimensions"]; !ok {
		t.Error("project_meta missing vec_dimensions")
	} else if vd != "384" {
		t.Errorf("expected vec_dimensions=384, got %s", vd)
	}

	// embedding_model should be "all-minilm".
	if em, ok := meta["embedding_model"]; !ok {
		t.Error("project_meta missing embedding_model")
	} else if em != "all-minilm" {
		t.Errorf("expected embedding_model=all-minilm, got %s", em)
	}

	// --- Verify no orphan chunks (chunks without matching files) ---
	var orphans int
	if err := db.QueryRow(`
		SELECT count(*) FROM chunks c
		LEFT JOIN files f ON c.file_path = f.path
		WHERE f.path IS NULL
	`).Scan(&orphans); err != nil {
		t.Fatalf("query orphan chunks: %v", err)
	}
	if orphans != 0 {
		t.Errorf("found %d orphan chunks (no matching file)", orphans)
	}

	// --- Verify specific known symbols exist ---
	knownSymbols := map[string]string{
		"ValidateToken":  "function",
		"HandleHealth":   "function",
		"QueryUsers":     "function",
		"User":           "type",
		"UserRepository": "interface",
	}
	for symbol, expectedKind := range knownSymbols {
		var kind string
		err := db.QueryRow("SELECT kind FROM chunks WHERE symbol = ?", symbol).Scan(&kind)
		if err == sql.ErrNoRows {
			t.Errorf("expected symbol %q to exist in chunks", symbol)
		} else if err != nil {
			t.Fatalf("query symbol %s: %v", symbol, err)
		} else if kind != expectedKind {
			t.Errorf("symbol %s: expected kind=%s, got %s", symbol, expectedKind, kind)
		}
	}
}

func TestE2E_CLI_SQLVerifyKNN(t *testing.T) {
	t.Parallel()
	projectPath := gitSampleProject(t)
	dataHome := t.TempDir()

	// Index the project.
	_, stderr, err := runCLIWithDataHome(t, dataHome, "index", projectPath)
	if err != nil {
		t.Fatalf("index failed: %v\nstderr: %s", err, stderr)
	}

	db := openIndexDB(t, dataHome, projectPath)

	// Grab the embedding vector of ValidateToken from vec_chunks.
	var tokenID string
	if err := db.QueryRow("SELECT id FROM chunks WHERE symbol = 'ValidateToken'").Scan(&tokenID); err != nil {
		t.Fatalf("get ValidateToken id: %v", err)
	}

	// Use ValidateToken's own vector as the query vector — it should be
	// the top result (distance ≈ 0, score ≈ 1).
	var vecBlob []byte
	if err := db.QueryRow("SELECT embedding FROM vec_chunks WHERE id = ?", tokenID).Scan(&vecBlob); err != nil {
		t.Fatalf("get ValidateToken embedding: %v", err)
	}

	// Run raw KNN query.
	rows, err := db.Query(`
		SELECT c.symbol, c.kind, v.distance
		FROM vec_chunks v
		JOIN chunks c ON v.id = c.id
		WHERE v.embedding MATCH ?
		AND v.k = 5
		ORDER BY v.distance
		LIMIT 5
	`, vecBlob)
	if err != nil {
		t.Fatalf("KNN query: %v", err)
	}
	defer rows.Close()

	type knnResult struct {
		symbol   string
		kind     string
		distance float64
	}
	var results []knnResult
	for rows.Next() {
		var r knnResult
		if err := rows.Scan(&r.symbol, &r.kind, &r.distance); err != nil {
			t.Fatalf("scan KNN row: %v", err)
		}
		results = append(results, r)
	}

	if len(results) == 0 {
		t.Fatal("KNN query returned no results")
	}

	// First result should be ValidateToken itself with distance ≈ 0.
	if results[0].symbol != "ValidateToken" {
		t.Errorf("expected first KNN result to be ValidateToken, got %s", results[0].symbol)
	}
	if results[0].distance > 0.01 {
		t.Errorf("self-similarity distance should be ≈ 0, got %f", results[0].distance)
	}

	// All distances should be non-negative and ordered ascending.
	for i, r := range results {
		if r.distance < 0 {
			t.Errorf("result[%d] %s: distance should be >= 0, got %f", i, r.symbol, r.distance)
		}
		if i > 0 && r.distance < results[i-1].distance {
			t.Errorf("results not ordered by distance: %s(%f) < %s(%f)",
				r.symbol, r.distance, results[i-1].symbol, results[i-1].distance)
		}
	}

	// Scores (1 - distance) should all be in (0, 1].
	for i, r := range results {
		score := 1.0 - r.distance
		if score <= 0 || score > 1 {
			t.Errorf("result[%d] %s: score should be in (0, 1], got %f", i, r.symbol, score)
		}
	}
}

func TestE2E_CLI_SQLVerifyIncremental(t *testing.T) {
	t.Parallel()
	dataHome := t.TempDir()
	tmpDir := t.TempDir()
	copyDir(t, sampleProjectPath(t), tmpDir)

	// Index.
	_, _, err := runCLIWithDataHome(t, dataHome, "index", tmpDir)
	if err != nil {
		t.Fatalf("first index failed: %v", err)
	}

	db := openIndexDB(t, dataHome, tmpDir)

	// Count initial state.
	var initialFiles, initialChunks int
	db.QueryRow("SELECT count(*) FROM files").Scan(&initialFiles)
	db.QueryRow("SELECT count(*) FROM chunks").Scan(&initialChunks)

	if initialFiles != 6 {
		t.Fatalf("expected 6 initial files, got %d", initialFiles)
	}

	// Get initial root hash.
	var hash1 string
	db.QueryRow("SELECT value FROM project_meta WHERE key = 'root_hash'").Scan(&hash1)

	// Add a new file.
	newCode := "package project\n\n// Shutdown stops the server.\nfunc Shutdown() error { return nil }\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "shutdown.go"), []byte(newCode), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Close DB before re-indexing (index opens its own connection).
	db.Close()

	// Re-index.
	_, _, err = runCLIWithDataHome(t, dataHome, "index", tmpDir)
	if err != nil {
		t.Fatalf("second index failed: %v", err)
	}

	db = openIndexDB(t, dataHome, tmpDir)

	// Verify file count increased.
	var newFileCount int
	db.QueryRow("SELECT count(*) FROM files").Scan(&newFileCount)
	if newFileCount != 7 {
		t.Errorf("expected 7 files after adding one, got %d", newFileCount)
	}

	// Verify chunk count increased.
	var newChunkCount int
	db.QueryRow("SELECT count(*) FROM chunks").Scan(&newChunkCount)
	if newChunkCount <= initialChunks {
		t.Errorf("expected more chunks after adding file: before=%d, after=%d", initialChunks, newChunkCount)
	}

	// Verify new symbol exists.
	var shutdownExists int
	db.QueryRow("SELECT count(*) FROM chunks WHERE symbol = 'Shutdown'").Scan(&shutdownExists)
	if shutdownExists == 0 {
		t.Error("expected Shutdown symbol in chunks after adding file")
	}

	// Verify vec_chunks stayed in sync.
	var vecCount, chunkCount int
	db.QueryRow("SELECT count(*) FROM vec_chunks").Scan(&vecCount)
	db.QueryRow("SELECT count(*) FROM chunks").Scan(&chunkCount)
	if vecCount != chunkCount {
		t.Errorf("vec_chunks (%d) out of sync with chunks (%d)", vecCount, chunkCount)
	}

	// Root hash should have changed.
	var hash2 string
	db.QueryRow("SELECT value FROM project_meta WHERE key = 'root_hash'").Scan(&hash2)
	if hash2 == hash1 {
		t.Error("root_hash should change after adding a file")
	}

	// Delete a file.
	db.Close()
	os.Remove(filepath.Join(tmpDir, "database.go"))

	_, _, err = runCLIWithDataHome(t, dataHome, "index", tmpDir)
	if err != nil {
		t.Fatalf("third index failed: %v", err)
	}

	db = openIndexDB(t, dataHome, tmpDir)

	// Verify file removed from files table.
	var dbFileExists int
	db.QueryRow("SELECT count(*) FROM files WHERE path LIKE '%database.go'").Scan(&dbFileExists)
	if dbFileExists != 0 {
		t.Error("database.go should be removed from files table after deletion")
	}

	// Verify QueryUsers chunks are gone.
	var queryUsersExists int
	db.QueryRow("SELECT count(*) FROM chunks WHERE symbol = 'QueryUsers'").Scan(&queryUsersExists)
	if queryUsersExists != 0 {
		t.Error("QueryUsers chunks should be removed after deleting database.go")
	}

	// Verify no orphan chunks.
	var orphans int
	db.QueryRow(`
		SELECT count(*) FROM chunks c
		LEFT JOIN files f ON c.file_path = f.path
		WHERE f.path IS NULL
	`).Scan(&orphans)
	if orphans != 0 {
		t.Errorf("found %d orphan chunks after file deletion", orphans)
	}
}

// TestE2E_CLI_AncestorReuse_Index verifies that `lumen index` from a non-git
// subdirectory reuses the parent's existing on-disk index via findAncestorIndex.
func TestE2E_CLI_AncestorReuse_Index(t *testing.T) {
	t.Parallel()
	dataHome := t.TempDir()

	// Copy sample-project to a plain (non-git) temp dir.
	dir := t.TempDir()
	copyDir(t, sampleProjectPath(t), dir)
	subDir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Index the parent directory.
	_, _, err := runCLIWithDataHome(t, dataHome, "index", dir)
	if err != nil {
		t.Fatalf("first index failed: %v", err)
	}

	// Index from sub/ — should find ancestor's DB and report up-to-date.
	stdout, _, err := runCLIWithDataHome(t, dataHome, "index", subDir)
	if err != nil {
		t.Fatalf("sub index failed: %v", err)
	}
	if !strings.Contains(stdout, "up to date") {
		t.Errorf("expected 'up to date' from ancestor reuse, got stdout: %s", stdout)
	}
}

// TestE2E_CLI_AncestorReuse_Search verifies that `lumen search` from a non-git
// subdirectory reuses the parent's on-disk index for search results.
// Uses --cwd to set the index root explicitly, while -p scopes the search.
func TestE2E_CLI_AncestorReuse_Search(t *testing.T) {
	t.Parallel()
	dataHome := t.TempDir()

	// Copy sample-project to a plain (non-git) temp dir.
	dir := t.TempDir()
	copyDir(t, sampleProjectPath(t), dir)
	subDir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Seed the parent index.
	_, _, err := runCLIWithDataHome(t, dataHome, "index", dir)
	if err != nil {
		t.Fatalf("index failed: %v", err)
	}

	// Search with --cwd pointing at sub/ (triggers ancestor resolution for
	// indexRoot) and -p pointing at dir (full scope, no pathPrefix filtering).
	stdout, _, err := runCLIWithDataHome(t, dataHome, "search", "--cwd", subDir, "-p", dir, "token validation")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if !strings.Contains(stdout, `symbol=`) {
		t.Fatalf("expected at least one result from ancestor index, got stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "ValidateToken") {
		t.Errorf("expected ValidateToken in search results from ancestor index, got stdout: %s", stdout)
	}
}

// TestE2E_CLI_AncestorReuse_NearestWins verifies that when multiple ancestor
// directories have indexes, the nearest one is used.
func TestE2E_CLI_AncestorReuse_NearestWins(t *testing.T) {
	t.Parallel()
	dataHome := t.TempDir()

	// Create grandparent with canonical files.
	grandparent := t.TempDir()
	pkgDir := filepath.Join(grandparent, "pkg")
	apiDir := filepath.Join(grandparent, "api")
	for _, d := range []string{pkgDir, apiDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "server.go"), []byte(`package pkg

// StartServer starts the main server loop.
func StartServer() {}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "handler.go"), []byte(`package api

// HandleLogin processes login requests.
func HandleLogin() {}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create parent with a different file.
	parent := filepath.Join(grandparent, "parent")
	libDir := filepath.Join(parent, "lib")
	childDir := filepath.Join(parent, "child")
	for _, d := range []string{libDir, childDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(libDir, "worker.go"), []byte(`package lib

// RunWorker executes the background worker.
func RunWorker() {}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Index both levels.
	if _, _, err := runCLIWithDataHome(t, dataHome, "index", grandparent); err != nil {
		t.Fatalf("grandparent index failed: %v", err)
	}
	if _, _, err := runCLIWithDataHome(t, dataHome, "index", parent); err != nil {
		t.Fatalf("parent index failed: %v", err)
	}

	// Search from child/ — nearest ancestor is parent/, which has RunWorker.
	// Use --cwd child/ so indexRoot resolves to parent/ via ancestor walking,
	// and -p parent/ for full scope (no pathPrefix filtering).
	stdout, _, err := runCLIWithDataHome(t, dataHome, "search", "--cwd", childDir, "-p", parent, "run worker")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if !strings.Contains(stdout, `symbol="RunWorker"`) {
		t.Errorf("expected RunWorker from nearest ancestor (parent), got stdout: %s", stdout)
	}
}
