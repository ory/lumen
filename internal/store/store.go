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

// Package store manages SQLite storage for code chunks and their embedding vectors.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3" // register sqlite3 driver

	"github.com/ory/lumen/internal/chunker"
	sqlite_vec "github.com/ory/lumen/internal/sqlitevec"
)

// MetaLastAccessedAt is the project_meta key holding the RFC3339 UTC timestamp
// of the last time this index was opened. `lumen clean` reads it to decide
// whether an index is still in use.
const MetaLastAccessedAt = "last_accessed_at"

// accessStampBusyTimeoutMS bounds how long opening a store waits for the write
// lock to record its access time before giving up on the stamp.
const accessStampBusyTimeoutMS = 250

func init() {
	if err := sqlite_vec.Auto(); err != nil {
		panic(err)
	}
}

// IsCorruptionErr reports whether err indicates SQLite database corruption.
// These are the canonical SQLite error messages for an unrecoverable on-disk
// data problem; the only safe recovery is to delete the database and rebuild.
func IsCorruptionErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "database disk image is malformed") ||
		strings.Contains(msg, "disk I/O error")
}

// deleteDBFiles removes the SQLite database file and its WAL/SHM sidecars.
// Errors are silently ignored — the file may already be gone or unwritable.
func deleteDBFiles(path string) {
	for _, suffix := range []string{"", "-wal", "-shm"} {
		_ = os.Remove(path + suffix)
	}
}

// SearchResult represents a single result from a vector search.
type SearchResult struct {
	FilePath  string
	Symbol    string
	Kind      string
	StartLine int
	EndLine   int
	Distance  float64
}

// StoreStats holds aggregate statistics about the store contents.
type StoreStats struct { //nolint:revive // StoreStats is intentionally named to avoid ambiguity at call sites
	TotalFiles       int
	TotalChunks      int
	UniqueVectors    int
	SharedReferences int
	DatabaseBytes    int64
	ReclaimableBytes int64
	VectorStorage    string
}

// Store manages SQLite + sqlite-vec storage for code chunks and their
// embedding vectors. A shared Store's selected project is mutable; callers
// must serialize UseProject with all operations that depend on that selection.
type Store struct {
	db            *sql.DB
	readDB        *sql.DB // separate read-only connection; nil for :memory: databases
	dimensions    int
	shared        bool
	vectorStorage string
	projectPath   string
	projectID     int64
	dsn           string
}

// New opens (or creates) a SQLite database at dsn, enables WAL mode and
// foreign keys, and creates the schema tables if they do not exist.
// dimensions specifies the size of the embedding vectors.
//
// If the database file is corrupted (SQLite returns a corruption error during
// open or schema setup), New deletes the file and its WAL/SHM sidecars and
// retries once from a clean state. In-memory databases (dsn == ":memory:")
// are never deleted.
func New(dsn string, dimensions int) (*Store, error) {
	s, err := openStore(dsn, dimensions)
	if err != nil && IsCorruptionErr(err) && dsn != ":memory:" {
		deleteDBFiles(dsn)
		s, err = openStore(dsn, dimensions)
	}
	if err != nil {
		return s, err
	}
	if s.shared && s.projectID == 0 {
		if err := s.UseProject(""); err != nil {
			_ = s.Close()
			return nil, err
		}
	}
	if s.shared {
		s.stampSharedAccess()
	} else {
		s.stampAccess(dsn)
	}
	return s, nil
}

// NewCollection opens a repository-scoped shared collection and selects the
// project membership identified by projectPath. vectorStorage must be int8 or
// float32. Multiple Store instances may safely select different worktrees in
// the same database. If schema setup detects corruption, on-disk database and
// sidecar files are removed and creation is retried once.
func NewCollection(dsn string, dimensions int, vectorStorage, projectPath string) (*Store, error) {
	if vectorStorage != "int8" && vectorStorage != "float32" {
		return nil, fmt.Errorf("unsupported vector storage %q", vectorStorage)
	}
	s, err := openCollection(dsn, dimensions, vectorStorage)
	if err != nil && IsCorruptionErr(err) && dsn != ":memory:" {
		deleteDBFiles(dsn)
		s, err = openCollection(dsn, dimensions, vectorStorage)
	}
	if err != nil {
		return nil, err
	}
	if err := s.UseProject(projectPath); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}

// stampAccess records the current time as this index's last access so
// `lumen clean` can tell indexes that are still in use apart from abandoned
// ones.
//
// The write runs on a throwaway connection with a short busy timeout instead of
// the store's own connection: a concurrently running indexer holds the SQLite
// write lock for the duration of every insert batch, and the sqlite3 driver does
// not honour context cancellation while waiting out busy_timeout — so reusing
// the store's 120s timeout would stall opening (and therefore search) for
// minutes. Failing to stamp is harmless: the indexer stamps its own open, and
// `clean` falls back to last_indexed_at.
func (s *Store) stampAccess(dsn string) {
	value := time.Now().UTC().Format(time.RFC3339)
	if dsn == ":memory:" {
		// Nothing else can hold the lock on a private in-memory database, and a
		// second connection would open a different one.
		_ = s.SetMeta(MetaLastAccessedAt, value)
		return
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return
	}
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(fmt.Sprintf("PRAGMA busy_timeout=%d", accessStampBusyTimeoutMS)); err != nil {
		return
	}
	_, _ = db.Exec(
		`INSERT INTO project_meta (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		MetaLastAccessedAt, value,
	)
}

func openStore(dsn string, dimensions int) (*Store, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(1)

	// Enable WAL mode, foreign keys, and write-performance settings.
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-64000",
		"PRAGMA temp_store=MEMORY",
		"PRAGMA busy_timeout=120000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("exec %q: %w", p, err)
		}
	}
	// A low-level caller may open a database already upgraded to the shared
	// schema (for example metadata tooling during a rolling upgrade). Return a
	// shared view instead of attempting to overlay the legacy tables.
	shared, err := checkTableExists(db, "collection_meta")
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("check collection_meta table: %w", err)
	}
	if shared {
		var storage string
		if err := db.QueryRow(`SELECT value FROM collection_meta WHERE key = 'vector_storage'`).Scan(&storage); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("read vector_storage: %w", err)
		}
		_ = db.Close()
		return openCollection(dsn, dimensions, storage)
	}

	if err := createSchema(db, dimensions); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	s := &Store{db: db, dimensions: dimensions}

	// Open a separate read connection for file-based databases. In WAL mode,
	// readers do not block writers and vice versa, so Search and other read
	// operations can proceed concurrently with indexing.
	// :memory: databases cannot share state across connections.
	//
	// We deliberately do NOT use ?mode=ro (SQLITE_OPEN_READONLY) because on
	// Linux a read-only connection cannot always mmap the WAL shared-memory
	// file, causing it to silently fall back to reading only the main
	// database file (missing uncommitted WAL data). Instead we open a
	// normal connection and set PRAGMA query_only=ON, which prevents writes
	// at the SQL level while keeping full WAL read visibility.
	if dsn != ":memory:" {
		readDB, err := sql.Open("sqlite3", dsn)
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("open read db: %w", err)
		}
		readDB.SetMaxOpenConns(1)

		// WAL journal mode is inherited from the database file (set by the
		// write connection), so we don't set it here. We configure
		// query_only to prevent accidental writes and tune performance.
		readPragmas := []string{
			"PRAGMA query_only=ON",
			"PRAGMA cache_size=-64000",
			"PRAGMA temp_store=MEMORY",
			"PRAGMA busy_timeout=120000",
		}
		for _, p := range readPragmas {
			if _, err := readDB.Exec(p); err != nil {
				_ = readDB.Close()
				_ = db.Close()
				return nil, fmt.Errorf("read db %q: %w", p, err)
			}
		}
		s.readDB = readDB
	}

	return s, nil
}

// reader returns the read-only database connection when available (file-based
// databases), falling back to the primary write connection for in-memory databases.
func (s *Store) reader() *sql.DB {
	if s.readDB != nil {
		return s.readDB
	}
	return s.db
}

func createSchema(db *sql.DB, dimensions int) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS files (
			path TEXT PRIMARY KEY,
			hash TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS project_meta (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS chunks (
			id         TEXT PRIMARY KEY,
			file_path  TEXT NOT NULL REFERENCES files(path),
			symbol     TEXT NOT NULL,
			kind       TEXT NOT NULL,
			start_line INTEGER NOT NULL,
			end_line   INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chunks_file_path ON chunks(file_path)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("exec %q: %w", s, err)
		}
	}

	// Handle vec_chunks dimension mismatch: if the table exists with
	// different dimensions, drop it and all associated data so it gets
	// recreated with the correct size.
	if err := ensureVecDimensions(db, dimensions); err != nil {
		return err
	}

	return nil
}

// ensureVecDimensions creates the vec_chunks virtual table, or recreates it
// if the existing table has a different number of dimensions.
func ensureVecDimensions(db *sql.DB, dimensions int) error {
	tableExists, err := checkTableExists(db, "vec_chunks")
	if err != nil {
		return err
	}

	if !tableExists {
		return createVecTable(db, dimensions)
	}

	storedDims, err := getStoredDimensions(db)
	if err == nil && storedDims == dimensions {
		return nil
	}

	return resetAndRecreateVecTable(db, dimensions)
}

func checkTableExists(db *sql.DB, tableName string) (bool, error) {
	var exists bool
	err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&exists)
	return exists, err
}

func createVecTable(db *sql.DB, dimensions int) error {
	createVec := fmt.Sprintf(
		`CREATE VIRTUAL TABLE IF NOT EXISTS vec_chunks USING vec0(
			id TEXT PRIMARY KEY,
			embedding float[%d] distance_metric=cosine
		)`, dimensions)

	if _, err := db.Exec(createVec); err != nil {
		return fmt.Errorf("create vec_chunks: %w", err)
	}
	return storeDimensions(db, dimensions)
}

func getStoredDimensions(db *sql.DB) (int, error) {
	var dims int
	err := db.QueryRow("SELECT value FROM project_meta WHERE key = 'vec_dimensions'").Scan(&dims)
	return dims, err
}

func storeDimensions(db *sql.DB, dimensions int) error {
	_, err := db.Exec(
		`INSERT INTO project_meta (key, value) VALUES ('vec_dimensions', ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		fmt.Sprintf("%d", dimensions),
	)
	if err != nil {
		return fmt.Errorf("store vec_dimensions: %w", err)
	}
	return nil
}

func resetAndRecreateVecTable(db *sql.DB, dimensions int) error {
	// Drop the virtual table first — cannot be wrapped in a transaction in SQLite.
	if _, err := db.Exec("DROP TABLE IF EXISTS vec_chunks"); err != nil {
		return fmt.Errorf("drop vec_chunks: %w", err)
	}

	// Clear regular tables atomically so a crash leaves them consistent.
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin reset tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, stmt := range []string{
		"DELETE FROM chunks",
		"DELETE FROM files",
		"DELETE FROM project_meta",
	} {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("reset %q: %w", stmt, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit reset: %w", err)
	}

	return createVecTable(db, dimensions)
}

// ReadMetaAt opens the SQLite database at dbPath read-only and returns the
// project_meta values for keys in a single open. Missing keys are absent from
// the returned map. It is safe to call on databases created by older binaries
// with a different schema version; the function performs no migrations, holds
// no write lock, and — unlike New — does not count as an index access, so
// scanning metadata never keeps an abandoned index alive. Returns an error only
// when the file cannot be opened (missing file, permission denied) or when the
// project_meta table is missing (the file is not a lumen index).
func ReadMetaAt(dbPath string, keys ...string) (map[string]string, error) {
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("stat %s: %w", dbPath, err)
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", dbPath, err)
	}
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA query_only=ON"); err != nil {
		return nil, fmt.Errorf("set query_only: %w", err)
	}

	return queryMeta(db, keys)
}

// SetMeta upserts a key-value pair in the project_meta table.
func (s *Store) SetMeta(key, value string) error {
	if s.shared {
		return s.setProjectMeta(key, value)
	}
	_, err := s.db.Exec(
		`INSERT INTO project_meta (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	return err
}

// GetMeta retrieves a value from the project_meta table by key.
// It uses the read-only connection when available for concurrency with writes.
func (s *Store) GetMeta(key string) (string, error) {
	if s.shared {
		return s.getProjectMeta(key)
	}
	var val string
	err := s.reader().QueryRow("SELECT value FROM project_meta WHERE key = ?", key).Scan(&val)
	if err != nil {
		return "", err
	}
	return val, nil
}

// GetMetaBatch retrieves multiple key-value pairs from project_meta in one query.
// Missing keys are absent from the returned map. Uses the read-only connection
// when available for concurrency with writes.
func (s *Store) GetMetaBatch(keys []string) (map[string]string, error) {
	if s.shared {
		return s.getProjectMetaBatch(keys)
	}
	return queryMeta(s.reader(), keys)
}

// queryMeta fetches keys from project_meta in a single query. Missing keys are
// absent from the returned map.
func queryMeta(db *sql.DB, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	if len(keys) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(keys))
	args := make([]any, len(keys))
	for i, k := range keys {
		placeholders[i] = "?"
		args[i] = k
	}
	query := fmt.Sprintf(
		"SELECT key, value FROM project_meta WHERE key IN (%s)",
		strings.Join(placeholders, ","),
	)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query project_meta: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, fmt.Errorf("scan project_meta: %w", err)
		}
		result[k] = v
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("query project_meta: %w", err)
	}
	return result, nil
}

// UpsertFile inserts or updates a file path and its content hash.
func (s *Store) UpsertFile(path, hash string) error {
	if s.shared {
		return s.upsertSharedFile(path, hash)
	}
	_, err := s.db.Exec(
		`INSERT INTO files (path, hash) VALUES (?, ?)
		 ON CONFLICT(path) DO UPDATE SET hash = excluded.hash`,
		path, hash,
	)
	return err
}

// InsertChunks inserts a batch of chunks and their corresponding embedding
// vectors into the chunks and vec_chunks tables within a single transaction.
// Precondition: caller must have called DeleteFileChunks for every file path
// present in chunks before calling this function. vec_chunks does not support
// INSERT OR REPLACE (sqlite-vec virtual table limitation), so duplicate IDs
// would cause an error. The deduplication loop below handles within-batch
// duplicates only.
func (s *Store) InsertChunks(chunks []chunker.Chunk, vectors [][]float32) error {
	if s.shared {
		return s.insertSharedChunks(chunks, vectors)
	}
	if len(chunks) != len(vectors) {
		return fmt.Errorf("chunks and vectors length mismatch: %d vs %d", len(chunks), len(vectors))
	}

	chunks, vectors = deduplicateChunks(chunks, vectors)
	return s.insertChunksInTransaction(chunks, vectors)
}

func deduplicateChunks(chunks []chunker.Chunk, vectors [][]float32) ([]chunker.Chunk, [][]float32) {
	seen := make(map[string]bool, len(chunks))
	deduped := make([]chunker.Chunk, 0, len(chunks))
	dedupedVecs := make([][]float32, 0, len(vectors))
	for i := range len(chunks) {
		if !seen[chunks[i].ID] {
			seen[chunks[i].ID] = true
			deduped = append(deduped, chunks[i])
			dedupedVecs = append(dedupedVecs, vectors[i])
		}
	}
	return deduped, dedupedVecs
}

func (s *Store) insertChunksInTransaction(chunks []chunker.Chunk, vectors [][]float32) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	chunkStmt, err := tx.Prepare(
		`INSERT OR REPLACE INTO chunks (id, file_path, symbol, kind, start_line, end_line)
		 VALUES (?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("prepare chunk insert: %w", err)
	}
	defer func() { _ = chunkStmt.Close() }()

	vecStmt, err := tx.Prepare(
		`INSERT OR REPLACE INTO vec_chunks (id, embedding) VALUES (?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("prepare vec insert: %w", err)
	}
	defer func() { _ = vecStmt.Close() }()

	for i, c := range chunks {
		if err := insertChunkAndVector(chunkStmt, vecStmt, c, vectors[i], i); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func insertChunkAndVector(chunkStmt, vecStmt interface {
	Exec(...interface{}) (sql.Result, error)
}, c chunker.Chunk, vec []float32, idx int) error {
	if _, err := chunkStmt.Exec(c.ID, c.FilePath, c.Symbol, c.Kind, c.StartLine, c.EndLine); err != nil {
		return fmt.Errorf("insert chunk %s: %w", c.ID, err)
	}
	blob, err := sqlite_vec.SerializeFloat32(vec)
	if err != nil {
		return fmt.Errorf("serialize vector %d: %w", idx, err)
	}
	if _, err := vecStmt.Exec(c.ID, blob); err != nil {
		return fmt.Errorf("insert vec %s: %w", c.ID, err)
	}
	return nil
}

// DeleteFileChunks removes all chunks (and their vectors) associated with the
// given file path, then removes the file record itself.
func (s *Store) DeleteFileChunks(filePath string) error {
	if s.shared {
		return s.removeProjectFile(filePath)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Delete vec_chunks entries for chunks belonging to this file.
	_, err = tx.Exec(
		`DELETE FROM vec_chunks WHERE id IN (SELECT id FROM chunks WHERE file_path = ?)`,
		filePath,
	)
	if err != nil {
		return fmt.Errorf("delete vec_chunks: %w", err)
	}

	// Delete chunks.
	_, err = tx.Exec(`DELETE FROM chunks WHERE file_path = ?`, filePath)
	if err != nil {
		return fmt.Errorf("delete chunks: %w", err)
	}

	// Delete file record.
	_, err = tx.Exec(`DELETE FROM files WHERE path = ?`, filePath)
	if err != nil {
		return fmt.Errorf("delete file: %w", err)
	}

	return tx.Commit()
}

// Search performs a KNN vector search and returns the closest chunks.
// If maxDistance > 0, results with distance >= maxDistance are excluded.
// If pathPrefix != "", only chunks whose file_path equals pathPrefix or
// starts with pathPrefix+"/" are returned; the KNN candidate count is
// inflated to compensate for the post-JOIN filter.
//
// Search uses a dedicated read-only database connection (when available) so
// that it can execute concurrently with write operations on the primary
// connection (e.g. during indexing). The provided context is used for
// query cancellation.
func (s *Store) Search(ctx context.Context, queryVec []float32, limit int, maxDistance float64, pathPrefix string) ([]SearchResult, error) {
	if s.shared {
		return s.searchShared(ctx, queryVec, limit, maxDistance, pathPrefix)
	}
	blob, err := sqlite_vec.SerializeFloat32(queryVec)
	if err != nil {
		return nil, fmt.Errorf("serialize query: %w", err)
	}

	// When filtering by path prefix we fetch more KNN candidates so the
	// post-JOIN filter still returns enough results.
	knn := limit
	if pathPrefix != "" {
		knn = min(limit*3, 300)
	}

	// Build WHERE clauses dynamically.
	whereClauses := []string{"v.embedding MATCH ?", "v.k = ?"}
	args := []any{blob, knn}

	if maxDistance > 0 {
		whereClauses = append(whereClauses, "v.distance < ?")
		args = append(args, maxDistance)
	}
	if pathPrefix != "" {
		whereClauses = append(whereClauses, "(c.file_path = ? OR c.file_path LIKE ? || '/%')")
		args = append(args, pathPrefix, pathPrefix)
	}
	args = append(args, limit)

	query := fmt.Sprintf(`
		SELECT c.file_path, c.symbol, c.kind, c.start_line, c.end_line, v.distance
		FROM vec_chunks v
		JOIN chunks c ON v.id = c.id
		WHERE %s
		ORDER BY v.distance
		LIMIT ?
	`, strings.Join(whereClauses, "\n\t\tAND "))

	rows, err := s.reader().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.FilePath, &r.Symbol, &r.Kind, &r.StartLine, &r.EndLine, &r.Distance); err != nil {
			return nil, fmt.Errorf("scan result: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return results, nil
}

// GetFileHashes returns a map of file path to content hash for all tracked files.
func (s *Store) GetFileHashes() (map[string]string, error) {
	if s.shared {
		return s.projectFileHashes()
	}
	rows, err := s.db.Query("SELECT path, hash FROM files")
	if err != nil {
		return nil, fmt.Errorf("query files: %w", err)
	}
	defer func() { _ = rows.Close() }()

	hashes := make(map[string]string)
	for rows.Next() {
		var path, hash string
		if err := rows.Scan(&path, &hash); err != nil {
			return nil, fmt.Errorf("scan file: %w", err)
		}
		hashes[path] = hash
	}
	return hashes, rows.Err()
}

// Stats returns aggregate statistics about the store contents in one query.
// Uses the read-only connection when available for concurrency with writes.
func (s *Store) Stats() (StoreStats, error) {
	if s.shared {
		return s.sharedStats()
	}
	var stats StoreStats
	err := s.reader().QueryRow(
		`SELECT (SELECT count(*) FROM files), (SELECT count(*) FROM chunks)`,
	).Scan(&stats.TotalFiles, &stats.TotalChunks)
	if err != nil {
		return stats, fmt.Errorf("stats query: %w", err)
	}
	return stats, nil
}

// TopSymbols returns the n most frequently occurring symbol names in the store.
func (s *Store) TopSymbols(n int) ([]string, error) {
	if s.shared {
		return s.sharedTopSymbols(n)
	}
	rows, err := s.reader().Query(
		"SELECT symbol FROM chunks GROUP BY symbol ORDER BY count(*) DESC LIMIT ?", n,
	)
	if err != nil {
		return nil, fmt.Errorf("top symbols query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var symbols []string
	for rows.Next() {
		var sym string
		if err := rows.Scan(&sym); err != nil {
			return nil, fmt.Errorf("scan symbol: %w", err)
		}
		symbols = append(symbols, sym)
	}
	return symbols, rows.Err()
}

// HasSentinelFiles reports whether any files have an empty hash, indicating
// they were registered but never fully indexed (interrupted run).
// Uses the read-only connection when available for consistency with other
// read methods (GetMeta, Stats, Search).
func (s *Store) HasSentinelFiles() (bool, error) {
	if s.shared {
		return s.sharedHasSentinelFiles()
	}
	var exists bool
	err := s.reader().QueryRow("SELECT EXISTS(SELECT 1 FROM files WHERE hash = '')").Scan(&exists)
	return exists, err
}

// Analyze runs ANALYZE on the database so the query planner has up-to-date
// statistics. Call once after a full index pass, not after every batch.
func (s *Store) Analyze() {
	_, _ = s.db.Exec("ANALYZE")
}

// Close closes the underlying database connections.
func (s *Store) Close() error {
	if s.readDB != nil {
		_ = s.readDB.Close()
	}
	return s.db.Close()
}
