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
	"database/sql"
	"fmt"
	"strings"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3" // register sqlite3 driver

	"github.com/ory/lumen/internal/chunker"
)

func init() {
	sqlite_vec.Auto()
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
	TotalFiles  int
	TotalChunks int
}

// Store manages SQLite + sqlite-vec storage for code chunks and their
// embedding vectors.
type Store struct {
	db         *sql.DB
	dimensions int
}

// New opens (or creates) a SQLite database at dsn, enables WAL mode and
// foreign keys, and creates the schema tables if they do not exist.
// dimensions specifies the size of the embedding vectors.
func New(dsn string, dimensions int) (*Store, error) {
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
		"PRAGMA busy_timeout=5000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("exec %q: %w", p, err)
		}
	}

	if err := createSchema(db, dimensions); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return &Store{db: db, dimensions: dimensions}, nil
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
	stmts := []string{
		"DROP TABLE IF EXISTS vec_chunks",
		"DELETE FROM chunks",
		"DELETE FROM files",
		"DELETE FROM project_meta",
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("reset for dimension change %q: %w", s, err)
		}
	}

	createVec := fmt.Sprintf(
		`CREATE VIRTUAL TABLE IF NOT EXISTS vec_chunks USING vec0(
			id TEXT PRIMARY KEY,
			embedding float[%d] distance_metric=cosine
		)`, dimensions)

	if _, err := db.Exec(createVec); err != nil {
		return fmt.Errorf("recreate vec_chunks: %w", err)
	}

	return storeDimensions(db, dimensions)
}

// SetMeta upserts a key-value pair in the project_meta table.
func (s *Store) SetMeta(key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO project_meta (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	return err
}

// GetMeta retrieves a value from the project_meta table by key.
func (s *Store) GetMeta(key string) (string, error) {
	var val string
	err := s.db.QueryRow("SELECT value FROM project_meta WHERE key = ?", key).Scan(&val)
	if err != nil {
		return "", err
	}
	return val, nil
}

// GetMetaBatch retrieves multiple key-value pairs from project_meta in one query.
// Missing keys are absent from the returned map.
func (s *Store) GetMetaBatch(keys []string) (map[string]string, error) {
	if len(keys) == 0 {
		return map[string]string{}, nil
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
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query meta batch: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string]string, len(keys))
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, fmt.Errorf("scan meta: %w", err)
		}
		result[k] = v
	}
	return result, rows.Err()
}

// UpsertFile inserts or updates a file path and its content hash.
func (s *Store) UpsertFile(path, hash string) error {
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
		`INSERT INTO vec_chunks (id, embedding) VALUES (?, ?)`,
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
func (s *Store) Search(queryVec []float32, limit int, maxDistance float64, pathPrefix string) ([]SearchResult, error) {
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

	rows, err := s.db.Query(query, args...)
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
func (s *Store) Stats() (StoreStats, error) {
	var stats StoreStats
	err := s.db.QueryRow(
		`SELECT (SELECT count(*) FROM files), (SELECT count(*) FROM chunks)`,
	).Scan(&stats.TotalFiles, &stats.TotalChunks)
	if err != nil {
		return stats, fmt.Errorf("stats query: %w", err)
	}
	return stats, nil
}

// TopSymbols returns the n most frequently occurring symbol names in the store.
func (s *Store) TopSymbols(n int) ([]string, error) {
	rows, err := s.db.Query(
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

// Analyze runs ANALYZE on the database so the query planner has up-to-date
// statistics. Call once after a full index pass, not after every batch.
func (s *Store) Analyze() {
	_, _ = s.db.Exec("ANALYZE")
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}
