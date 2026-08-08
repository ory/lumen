// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ory/lumen/internal/chunker"
	sqlite_vec "github.com/ory/lumen/internal/sqlitevec"
)

const sharedSchemaVersion = "1"

// ErrVectorVanished marks a cross-process race where a vector observed during
// MissingChunkInputs was garbage-collected before StoreFileRevision began its
// write transaction. Callers can re-embed the complete file and retry once.
var ErrVectorVanished = errors.New("shared vector vanished")

// CollectionStats describes both project-local references and collection-wide
// physical storage. Chunk references may exceed UniqueVectors because vectors
// are content-addressed across file revisions and projects.
type CollectionStats struct {
	ProjectFiles     int
	ProjectChunks    int
	UniqueVectors    int
	ChunkReferences  int
	SharedReferences int
	DatabaseBytes    int64
	ReclaimableBytes int64
	VectorStorage    string
}

// CleanupStats reports a shared collection garbage-collection pass.
type CleanupStats struct {
	ProjectsRemoved int
	VectorsRemoved  int
	BytesReclaimed  int64
	ProjectsLeft    int
}

func openCollection(dsn string, dimensions int, vectorStorage string) (*Store, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(1)
	for _, pragma := range []string{
		"PRAGMA auto_vacuum=INCREMENTAL",
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-64000",
		"PRAGMA temp_store=MEMORY",
		"PRAGMA busy_timeout=120000",
	} {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("exec %q: %w", pragma, err)
		}
	}
	// Legacy per-worktree databases remain readable during the lazy migration
	// window. New profile paths always create the shared schema; an existing
	// legacy path is upgraded by normal indexing without making it unreadable.
	legacy, err := checkTableExists(db, "files")
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("check files table: %w", err)
	}
	if legacy {
		shared, err := checkTableExists(db, "collection_meta")
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("check collection_meta table: %w", err)
		}
		if !shared {
			_ = db.Close()
			return openStore(dsn, dimensions)
		}
	}
	if err := createCollectionSchema(db, dimensions, vectorStorage); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create shared schema: %w", err)
	}

	s := &Store{
		db:            db,
		dimensions:    dimensions,
		shared:        true,
		vectorStorage: vectorStorage,
		dsn:           dsn,
	}
	if dsn == ":memory:" {
		return s, nil
	}
	readDB, err := sql.Open("sqlite3", dsn)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("open read db: %w", err)
	}
	readDB.SetMaxOpenConns(1)
	for _, pragma := range []string{
		"PRAGMA query_only=ON",
		"PRAGMA foreign_keys=ON",
		"PRAGMA cache_size=-64000",
		"PRAGMA temp_store=MEMORY",
		"PRAGMA busy_timeout=120000",
	} {
		if _, err := readDB.Exec(pragma); err != nil {
			_ = readDB.Close()
			_ = db.Close()
			return nil, fmt.Errorf("read db %q: %w", pragma, err)
		}
	}
	s.readDB = readDB
	return s, nil
}

func createCollectionSchema(db *sql.DB, dimensions int, vectorStorage string) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS collection_meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS projects (
			id INTEGER PRIMARY KEY,
			path TEXT NOT NULL UNIQUE,
			created_at TEXT NOT NULL,
			last_accessed_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS project_meta (
			project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			key TEXT NOT NULL,
			value TEXT NOT NULL,
			PRIMARY KEY(project_id, key)
		) WITHOUT ROWID`,
		`CREATE TABLE IF NOT EXISTS file_revisions (
			id INTEGER PRIMARY KEY,
			relative_path TEXT NOT NULL,
			content_hash BLOB NOT NULL,
			complete INTEGER NOT NULL DEFAULT 0,
			UNIQUE(relative_path, content_hash)
		)`,
		`CREATE TABLE IF NOT EXISTS project_files (
			project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			relative_path TEXT NOT NULL,
			file_revision_id INTEGER NOT NULL REFERENCES file_revisions(id),
			PRIMARY KEY(project_id, relative_path)
		) WITHOUT ROWID`,
		`CREATE INDEX IF NOT EXISTS idx_project_files_revision ON project_files(file_revision_id)`,
		`CREATE TABLE IF NOT EXISTS vector_keys (
			id INTEGER PRIMARY KEY,
			input_hash BLOB NOT NULL UNIQUE
		)`,
		`CREATE TABLE IF NOT EXISTS chunk_defs (
			id INTEGER PRIMARY KEY,
			file_revision_id INTEGER NOT NULL REFERENCES file_revisions(id) ON DELETE CASCADE,
			ordinal INTEGER NOT NULL,
			chunk_key TEXT NOT NULL,
			vector_id INTEGER NOT NULL REFERENCES vector_keys(id),
			symbol TEXT NOT NULL,
			kind TEXT NOT NULL,
			start_line INTEGER NOT NULL,
			end_line INTEGER NOT NULL,
			UNIQUE(file_revision_id, ordinal)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chunk_defs_revision ON chunk_defs(file_revision_id)`,
		`CREATE INDEX IF NOT EXISTS idx_chunk_defs_vector ON chunk_defs(vector_id)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("exec %q: %w", stmt, err)
		}
	}

	want := map[string]string{
		"schema_version": sharedSchemaVersion,
		"vec_dimensions": fmt.Sprintf("%d", dimensions),
		"vector_storage": vectorStorage,
	}
	for key, value := range want {
		var existing string
		err := db.QueryRow(`SELECT value FROM collection_meta WHERE key = ?`, key).Scan(&existing)
		switch {
		case err == sql.ErrNoRows:
			if _, err := db.Exec(`INSERT INTO collection_meta(key, value) VALUES (?, ?)`, key, value); err != nil {
				return err
			}
		case err != nil:
			return err
		case existing != value:
			return fmt.Errorf("collection profile mismatch for %s: stored %q, requested %q", key, existing, value)
		}
	}

	exists, err := checkTableExists(db, "vec_vectors")
	if err != nil {
		return err
	}
	if !exists {
		elementType := "int8"
		if vectorStorage == "float32" {
			elementType = "float"
		}
		stmt := fmt.Sprintf(`CREATE VIRTUAL TABLE vec_vectors USING vec0(
			vector_id INTEGER PRIMARY KEY,
			embedding %s[%d] distance_metric=cosine
		)`, elementType, dimensions)
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("create vec_vectors: %w", err)
		}
	}
	return nil
}

// IsShared reports whether this store uses the repository-scoped schema.
func (s *Store) IsShared() bool { return s.shared }

// UseProject selects (and, if necessary, creates) a project membership in the
// shared collection. Calling it repeatedly for the same path is cheap. Callers
// must serialize membership switches against Store operations; Indexer does so
// through its project lease.
func (s *Store) UseProject(projectPath string) error {
	if !s.shared {
		return nil
	}
	if projectPath == "" {
		projectPath = ":default"
	} else if abs, err := filepath.Abs(projectPath); err == nil {
		projectPath = filepath.Clean(abs)
	}
	if s.projectID != 0 && s.projectPath == projectPath {
		s.stampSharedAccess()
		return nil
	}
	var existingID int64
	if err := s.reader().QueryRow(`SELECT id FROM projects WHERE path = ?`, projectPath).Scan(&existingID); err == nil {
		s.projectID = existingID
		s.projectPath = projectPath
		s.stampSharedAccess()
		return nil
	} else if err != sql.ErrNoRows {
		return fmt.Errorf("select project: %w", err)
	}
	now := nowRFC3339()
	if _, err := s.db.Exec(`INSERT INTO projects(path, created_at, last_accessed_at) VALUES (?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET last_accessed_at = excluded.last_accessed_at`, projectPath, now, now); err != nil {
		return fmt.Errorf("register project: %w", err)
	}
	if err := s.db.QueryRow(`SELECT id FROM projects WHERE path = ?`, projectPath).Scan(&s.projectID); err != nil {
		return fmt.Errorf("select project: %w", err)
	}
	s.projectPath = projectPath
	return nil
}

// stampSharedAccess is best-effort and deliberately uses a short-lived
// connection with the same bounded timeout as legacy access stamping. Search
// must not wait behind a concurrent indexer's write transaction merely to
// refresh lifecycle metadata.
func (s *Store) stampSharedAccess() {
	if s.dsn == "" || s.dsn == ":memory:" {
		_, _ = s.db.Exec(`UPDATE projects SET last_accessed_at = ? WHERE id = ?`, nowRFC3339(), s.projectID)
		return
	}
	db, err := sql.Open("sqlite3", s.dsn)
	if err != nil {
		return
	}
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(fmt.Sprintf("PRAGMA busy_timeout=%d", accessStampBusyTimeoutMS)); err != nil {
		return
	}
	_, _ = db.Exec(`UPDATE projects SET last_accessed_at = ? WHERE id = ?`, nowRFC3339(), s.projectID)
}

func nowRFC3339() string { return time.Now().UTC().Format(time.RFC3339) }

func (s *Store) setProjectMeta(key, value string) error {
	_, err := s.db.Exec(`INSERT INTO project_meta(project_id, key, value) VALUES (?, ?, ?)
		ON CONFLICT(project_id, key) DO UPDATE SET value = excluded.value`, s.projectID, key, value)
	return err
}

func (s *Store) getProjectMeta(key string) (string, error) {
	var value string
	err := s.reader().QueryRow(`SELECT value FROM project_meta WHERE project_id = ? AND key = ?`, s.projectID, key).Scan(&value)
	return value, err
}

func (s *Store) getProjectMetaBatch(keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	if len(keys) == 0 {
		return result, nil
	}
	marks := make([]string, len(keys))
	args := make([]any, 0, len(keys)+1)
	args = append(args, s.projectID)
	for i, key := range keys {
		marks[i] = "?"
		args = append(args, key)
	}
	rows, err := s.reader().Query(`SELECT key, value FROM project_meta WHERE project_id = ? AND key IN (`+strings.Join(marks, ",")+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		result[key] = value
	}
	return result, rows.Err()
}

func hashBlob(hash string) []byte {
	if decoded, err := hex.DecodeString(hash); err == nil {
		return decoded
	}
	return []byte(hash)
}

// EmbeddingInput returns the exact, filepath-aware input whose hash identifies
// a vector in the shared collection.
func EmbeddingInput(c chunker.Chunk) string {
	return "// " + c.FilePath + "\n" + c.Content
}

func embeddingInputHash(c chunker.Chunk) [sha256.Size]byte {
	return sha256.Sum256([]byte(EmbeddingInput(c)))
}

// AttachExistingFileRevision links the selected project to an already indexed
// complete revision. It returns false when the revision has not been seen in
// this collection and must be chunked.
func (s *Store) AttachExistingFileRevision(relativePath, contentHash string) (bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	var revisionID int64
	err = tx.QueryRow(`SELECT id FROM file_revisions WHERE relative_path = ? AND content_hash = ? AND complete = 1`, relativePath, hashBlob(contentHash)).Scan(&revisionID)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	displaced, err := replaceProjectFileTx(tx, s.projectID, relativePath, revisionID)
	if err != nil {
		return false, err
	}
	if err := gcRevisionsTx(tx, displaced); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

// MissingChunkInputs returns the chunk positions whose exact embedding inputs
// are absent from the collection. Callers only need to embed these positions.
func (s *Store) MissingChunkInputs(chunks []chunker.Chunk) ([]int, error) {
	const queryBatchSize = 256
	hashes := make([][sha256.Size]byte, len(chunks))
	present := make(map[[sha256.Size]byte]struct{}, len(chunks))
	for i, c := range chunks {
		hashes[i] = embeddingInputHash(c)
	}
	for start := 0; start < len(hashes); start += queryBatchSize {
		end := min(start+queryBatchSize, len(hashes))
		marks := make([]string, end-start)
		args := make([]any, end-start)
		for i := start; i < end; i++ {
			marks[i-start] = "?"
			args[i-start] = hashes[i][:]
		}
		rows, err := s.reader().Query(`SELECT input_hash FROM vector_keys WHERE input_hash IN (`+strings.Join(marks, ",")+`)`, args...)
		if err != nil {
			return nil, fmt.Errorf("query shared vector inputs: %w", err)
		}
		for rows.Next() {
			var blob []byte
			if err := rows.Scan(&blob); err != nil {
				_ = rows.Close()
				return nil, fmt.Errorf("scan shared vector input: %w", err)
			}
			if len(blob) == sha256.Size {
				var hash [sha256.Size]byte
				copy(hash[:], blob)
				present[hash] = struct{}{}
			}
		}
		if err := errors.Join(rows.Err(), rows.Close()); err != nil {
			return nil, fmt.Errorf("read shared vector inputs: %w", err)
		}
	}
	missing := make([]int, 0, len(chunks))
	for i, hash := range hashes {
		if _, ok := present[hash]; !ok {
			missing = append(missing, i)
		}
	}
	return missing, nil
}

// StoreFileRevision atomically installs a complete revision and updates the
// selected project's membership. vectors is keyed by chunk position and only
// needs entries returned by MissingChunkInputs; concurrent winners are reused.
func (s *Store) StoreFileRevision(relativePath, contentHash string, chunks []chunker.Chunk, vectors map[int][]float32) (bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(`INSERT OR IGNORE INTO file_revisions(relative_path, content_hash, complete) VALUES (?, ?, 0)`, relativePath, hashBlob(contentHash))
	if err != nil {
		return false, err
	}
	inserted, _ := res.RowsAffected()
	var revisionID int64
	if err := tx.QueryRow(`SELECT id FROM file_revisions WHERE relative_path = ? AND content_hash = ?`, relativePath, hashBlob(contentHash)).Scan(&revisionID); err != nil {
		return false, err
	}

	if inserted > 0 {
		for i, c := range chunks {
			h := embeddingInputHash(c)
			var vectorID int64
			err := tx.QueryRow(`SELECT id FROM vector_keys WHERE input_hash = ?`, h[:]).Scan(&vectorID)
			if err == sql.ErrNoRows {
				vec, ok := vectors[i]
				if !ok {
					return false, fmt.Errorf("%w: missing embedding for chunk %d (%s)", ErrVectorVanished, i, c.ID)
				}
				result, err := tx.Exec(`INSERT INTO vector_keys(input_hash) VALUES (?)`, h[:])
				if err != nil {
					return false, err
				}
				vectorID, err = result.LastInsertId()
				if err != nil {
					return false, err
				}
				blob, err := s.serializeVector(vec)
				if err != nil {
					return false, err
				}
				insertSQL := `INSERT INTO vec_vectors(vector_id, embedding) VALUES (?, ?)`
				if s.vectorStorage == "int8" {
					insertSQL = `INSERT INTO vec_vectors(vector_id, embedding) VALUES (?, vec_int8(?))`
				}
				if _, err := tx.Exec(insertSQL, vectorID, blob); err != nil {
					return false, fmt.Errorf("insert shared vector: %w", err)
				}
			} else if err != nil {
				return false, err
			}
			if _, err := tx.Exec(`INSERT INTO chunk_defs(file_revision_id, ordinal, chunk_key, vector_id, symbol, kind, start_line, end_line)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, revisionID, i, c.ID, vectorID, c.Symbol, c.Kind, c.StartLine, c.EndLine); err != nil {
				return false, err
			}
		}
		if _, err := tx.Exec(`UPDATE file_revisions SET complete = 1 WHERE id = ?`, revisionID); err != nil {
			return false, err
		}
	} else {
		var complete bool
		if err := tx.QueryRow(`SELECT complete FROM file_revisions WHERE id = ?`, revisionID).Scan(&complete); err != nil {
			return false, err
		}
		if !complete {
			return false, fmt.Errorf("file revision %s is incomplete", relativePath)
		}
		// A force reindex supplies vectors even for an existing revision. Refresh
		// those physical rows in place while preserving their stable numeric IDs
		// and all sharing relationships.
		for position, vec := range vectors {
			if position < 0 || position >= len(chunks) {
				return false, fmt.Errorf("vector position %d out of range", position)
			}
			h := embeddingInputHash(chunks[position])
			var vectorID int64
			if err := tx.QueryRow(`SELECT id FROM vector_keys WHERE input_hash = ?`, h[:]).Scan(&vectorID); err != nil {
				return false, err
			}
			blob, err := s.serializeVector(vec)
			if err != nil {
				return false, err
			}
			if _, err := tx.Exec(`DELETE FROM vec_vectors WHERE vector_id = ?`, vectorID); err != nil {
				return false, fmt.Errorf("remove shared vector for refresh: %w", err)
			}
			insertSQL := `INSERT INTO vec_vectors(vector_id, embedding) VALUES (?, ?)`
			if s.vectorStorage == "int8" {
				insertSQL = `INSERT INTO vec_vectors(vector_id, embedding) VALUES (?, vec_int8(?))`
			}
			if _, err := tx.Exec(insertSQL, vectorID, blob); err != nil {
				return false, fmt.Errorf("refresh shared vector: %w", err)
			}
		}
	}

	displaced, err := replaceProjectFileTx(tx, s.projectID, relativePath, revisionID)
	if err != nil {
		return false, err
	}
	if err := gcRevisionsTx(tx, displaced); err != nil {
		return false, err
	}
	return inserted > 0, tx.Commit()
}

func replaceProjectFileTx(tx *sql.Tx, projectID int64, path string, revisionID int64) (int64, error) {
	var displaced int64
	err := tx.QueryRow(`SELECT file_revision_id FROM project_files WHERE project_id = ? AND relative_path = ?`, projectID, path).Scan(&displaced)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	_, err = tx.Exec(`INSERT INTO project_files(project_id, relative_path, file_revision_id) VALUES (?, ?, ?)
		ON CONFLICT(project_id, relative_path) DO UPDATE SET file_revision_id = excluded.file_revision_id`, projectID, path, revisionID)
	if err != nil {
		return 0, err
	}
	if displaced == revisionID {
		return 0, nil
	}
	return displaced, nil
}

func (s *Store) serializeVector(vector []float32) ([]byte, error) {
	if len(vector) != s.dimensions {
		return nil, fmt.Errorf("vector dimensions mismatch: got %d, want %d", len(vector), s.dimensions)
	}
	if s.vectorStorage == "float32" {
		return sqlite_vec.SerializeFloat32(vector)
	}
	return quantizeInt8(vector), nil
}

func quantizeInt8(vector []float32) []byte {
	maxAbs := float32(0)
	for _, value := range vector {
		abs := float32(math.Abs(float64(value)))
		if abs > maxAbs {
			maxAbs = abs
		}
	}
	result := make([]byte, len(vector))
	if maxAbs == 0 {
		return result
	}
	for i, value := range vector {
		q := int(math.Round(float64(value / maxAbs * 127)))
		q = max(-127, min(127, q))
		result[i] = byte(int8(q))
	}
	return result
}

func (s *Store) upsertSharedFile(path, hash string) error {
	if hash != "" {
		if attached, err := s.AttachExistingFileRevision(path, hash); err != nil || attached {
			return err
		}
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.Exec(`INSERT OR IGNORE INTO file_revisions(relative_path, content_hash, complete) VALUES (?, ?, 0)`, path, hashBlob(hash))
	if err != nil {
		return err
	}
	var revisionID int64
	if err := tx.QueryRow(`SELECT id FROM file_revisions WHERE relative_path = ? AND content_hash = ?`, path, hashBlob(hash)).Scan(&revisionID); err != nil {
		return err
	}
	displaced, err := replaceProjectFileTx(tx, s.projectID, path, revisionID)
	if err != nil {
		return err
	}
	if err := gcRevisionsTx(tx, displaced); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) insertSharedChunks(chunks []chunker.Chunk, vectors [][]float32) error {
	if len(chunks) != len(vectors) {
		return fmt.Errorf("chunks and vectors length mismatch: %d vs %d", len(chunks), len(vectors))
	}
	byFile := make(map[string][]int)
	for i, c := range chunks {
		byFile[c.FilePath] = append(byFile[c.FilePath], i)
	}
	for path, positions := range byFile {
		var hash []byte
		if err := s.db.QueryRow(`SELECT fr.content_hash FROM project_files pf JOIN file_revisions fr ON fr.id = pf.file_revision_id
			WHERE pf.project_id = ? AND pf.relative_path = ?`, s.projectID, path).Scan(&hash); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("no file revision registered for %q; call UpsertFile first: %w", path, err)
			}
			return fmt.Errorf("read file revision for %q: %w", path, err)
		}
		fileChunks := make([]chunker.Chunk, len(positions))
		fileVectors := make(map[int][]float32, len(positions))
		for i, pos := range positions {
			fileChunks[i] = chunks[pos]
			fileVectors[i] = vectors[pos]
		}
		if _, err := s.StoreFileRevision(path, hex.EncodeToString(hash), fileChunks, fileVectors); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) removeProjectFile(path string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var displaced int64
	err = tx.QueryRow(`SELECT file_revision_id FROM project_files WHERE project_id = ? AND relative_path = ?`, s.projectID, path).Scan(&displaced)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM project_files WHERE project_id = ? AND relative_path = ?`, s.projectID, path); err != nil {
		return err
	}
	if err := gcRevisionsTx(tx, displaced); err != nil {
		return err
	}
	return tx.Commit()
}

func gcRevisionsTx(tx *sql.Tx, revisionIDs ...int64) error {
	seen := make(map[int64]struct{}, len(revisionIDs))
	for _, revisionID := range revisionIDs {
		if revisionID == 0 {
			continue
		}
		if _, ok := seen[revisionID]; ok {
			continue
		}
		seen[revisionID] = struct{}{}
		var referenced bool
		if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM project_files WHERE file_revision_id = ?)`, revisionID).Scan(&referenced); err != nil {
			return err
		}
		if referenced {
			continue
		}
		rows, err := tx.Query(`SELECT DISTINCT vector_id FROM chunk_defs WHERE file_revision_id = ?`, revisionID)
		if err != nil {
			return err
		}
		var vectorIDs []int64
		for rows.Next() {
			var vectorID int64
			if err := rows.Scan(&vectorID); err != nil {
				_ = rows.Close()
				return err
			}
			vectorIDs = append(vectorIDs, vectorID)
		}
		if err := errors.Join(rows.Err(), rows.Close()); err != nil {
			return err
		}
		if _, err := tx.Exec(`DELETE FROM file_revisions WHERE id = ?`, revisionID); err != nil {
			return err
		}
		for _, vectorID := range vectorIDs {
			var used bool
			if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM chunk_defs WHERE vector_id = ?)`, vectorID).Scan(&used); err != nil {
				return err
			}
			if used {
				continue
			}
			if _, err := tx.Exec(`DELETE FROM vec_vectors WHERE vector_id = ?`, vectorID); err != nil {
				return err
			}
			if _, err := tx.Exec(`DELETE FROM vector_keys WHERE id = ?`, vectorID); err != nil {
				return err
			}
		}
	}
	return nil
}

func gcAllUnreferencedTx(tx *sql.Tx) error {
	if _, err := tx.Exec(`DELETE FROM file_revisions WHERE NOT EXISTS (
		SELECT 1 FROM project_files WHERE file_revision_id = file_revisions.id)`); err != nil {
		return err
	}
	rows, err := tx.Query(`SELECT id FROM vector_keys WHERE NOT EXISTS (
		SELECT 1 FROM chunk_defs WHERE vector_id = vector_keys.id)`)
	if err != nil {
		return err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := tx.Exec(`DELETE FROM vec_vectors WHERE vector_id = ?`, id); err != nil {
			return err
		}
		if _, err := tx.Exec(`DELETE FROM vector_keys WHERE id = ?`, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) projectFileHashes() (map[string]string, error) {
	rows, err := s.reader().Query(`SELECT pf.relative_path, fr.content_hash
		FROM project_files pf JOIN file_revisions fr ON fr.id = pf.file_revision_id
		WHERE pf.project_id = ?`, s.projectID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	result := make(map[string]string)
	for rows.Next() {
		var path string
		var hash []byte
		if err := rows.Scan(&path, &hash); err != nil {
			return nil, err
		}
		result[path] = hex.EncodeToString(hash)
	}
	return result, rows.Err()
}

func (s *Store) searchShared(ctx context.Context, queryVec []float32, limit int, maxDistance float64, pathPrefix string) ([]SearchResult, error) {
	if limit <= 0 {
		return nil, nil
	}
	blob, err := s.serializeVector(queryVec)
	if err != nil {
		return nil, fmt.Errorf("serialize query: %w", err)
	}
	var totalVectors int
	if err := s.reader().QueryRow(`SELECT count(*) FROM vector_keys`).Scan(&totalVectors); err != nil {
		return nil, err
	}
	if totalVectors == 0 {
		return nil, nil
	}
	candidates := min(totalVectors, max(limit, 32))
	for {
		results, boundaryDistance, err := s.searchSharedCandidates(ctx, blob, limit, candidates, maxDistance, pathPrefix)
		if err != nil {
			return nil, err
		}
		if candidates >= totalVectors || (len(results) >= limit && results[len(results)-1].Distance < boundaryDistance) {
			return results, nil
		}
		candidates = min(totalVectors, candidates*2)
	}
}

func (s *Store) searchSharedCandidates(ctx context.Context, blob []byte, limit, candidates int, maxDistance float64, pathPrefix string) ([]SearchResult, float64, error) {
	where := []string{"pf.project_id = ?"}
	args := []any{blob, candidates, s.projectID}
	if maxDistance > 0 {
		where = append(where, "knn.distance < ?")
		args = append(args, maxDistance)
	}
	if pathPrefix != "" {
		where = append(where, "(pf.relative_path = ? OR pf.relative_path LIKE ? || '/%')")
		args = append(args, pathPrefix, pathPrefix)
	}
	args = append(args, limit)
	matchExpression := "?"
	if s.vectorStorage == "int8" {
		matchExpression = "vec_int8(?)"
	}
	query := `WITH knn AS (
		SELECT vector_id, distance FROM vec_vectors
		WHERE embedding MATCH ` + matchExpression + ` AND k = ?
	)
	SELECT pf.relative_path, cd.symbol, cd.kind, cd.start_line, cd.end_line, knn.distance,
		max(knn.distance) OVER ()
	FROM knn
	JOIN chunk_defs cd ON cd.vector_id = knn.vector_id
	JOIN project_files pf ON pf.file_revision_id = cd.file_revision_id
	WHERE ` + strings.Join(where, " AND ") + `
	ORDER BY knn.distance, pf.relative_path, cd.start_line, cd.end_line, cd.symbol, cd.id
	LIMIT ?`
	rows, err := s.reader().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("shared search query: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var results []SearchResult
	var boundaryDistance float64
	for rows.Next() {
		var result SearchResult
		if err := rows.Scan(&result.FilePath, &result.Symbol, &result.Kind, &result.StartLine, &result.EndLine, &result.Distance, &boundaryDistance); err != nil {
			return nil, 0, err
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if err := rows.Close(); err != nil {
		return nil, 0, err
	}
	return results, boundaryDistance, nil
}

func (s *Store) sharedStats() (StoreStats, error) {
	stats, err := s.CollectionStats()
	if err != nil {
		return StoreStats{}, err
	}
	return StoreStats{
		TotalFiles:       stats.ProjectFiles,
		TotalChunks:      stats.ProjectChunks,
		UniqueVectors:    stats.UniqueVectors,
		SharedReferences: stats.SharedReferences,
		DatabaseBytes:    stats.DatabaseBytes,
		ReclaimableBytes: stats.ReclaimableBytes,
		VectorStorage:    stats.VectorStorage,
	}, nil
}

// CollectionStats returns project-local counts and physical collection usage.
func (s *Store) CollectionStats() (CollectionStats, error) {
	var stats CollectionStats
	err := s.reader().QueryRow(`SELECT
		(SELECT count(*) FROM project_files WHERE project_id = ?),
		(SELECT count(*) FROM chunk_defs cd JOIN project_files pf ON pf.file_revision_id = cd.file_revision_id WHERE pf.project_id = ?),
		(SELECT count(*) FROM vector_keys),
		(SELECT count(*) FROM chunk_defs cd JOIN project_files pf ON pf.file_revision_id = cd.file_revision_id)`, s.projectID, s.projectID).Scan(
		&stats.ProjectFiles, &stats.ProjectChunks, &stats.UniqueVectors, &stats.ChunkReferences)
	if err != nil {
		return stats, err
	}
	stats.SharedReferences = stats.ChunkReferences - stats.UniqueVectors
	stats.VectorStorage = s.vectorStorage
	var pageCount, pageSize, freelist int64
	if err := s.reader().QueryRow(`PRAGMA page_count`).Scan(&pageCount); err != nil {
		return stats, err
	}
	if err := s.reader().QueryRow(`PRAGMA page_size`).Scan(&pageSize); err != nil {
		return stats, err
	}
	if err := s.reader().QueryRow(`PRAGMA freelist_count`).Scan(&freelist); err != nil {
		return stats, err
	}
	stats.DatabaseBytes = pageCount * pageSize
	stats.ReclaimableBytes = freelist * pageSize
	return stats, nil
}

func (s *Store) sharedTopSymbols(n int) ([]string, error) {
	rows, err := s.reader().Query(`SELECT cd.symbol FROM chunk_defs cd
		JOIN project_files pf ON pf.file_revision_id = cd.file_revision_id
		WHERE pf.project_id = ? GROUP BY cd.symbol ORDER BY count(*) DESC LIMIT ?`, s.projectID, n)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, err
		}
		symbols = append(symbols, symbol)
	}
	return symbols, rows.Err()
}

func (s *Store) sharedHasSentinelFiles() (bool, error) {
	var exists bool
	err := s.reader().QueryRow(`SELECT EXISTS(
		SELECT 1 FROM project_files pf JOIN file_revisions fr ON fr.id = pf.file_revision_id
		WHERE pf.project_id = ? AND fr.complete = 0)`, s.projectID).Scan(&exists)
	return exists, err
}

// CleanupStaleProjects removes memberships not accessed since cutoff, garbage
// collects data no remaining project references, and incrementally reclaims
// free pages. Missing project directories are also considered stale.
func (s *Store) CleanupStaleProjects(cutoff time.Time) (CleanupStats, error) {
	before, err := s.CollectionStats()
	if err != nil {
		return CleanupStats{}, err
	}
	rows, err := s.db.Query(`SELECT p.id, p.path, p.last_accessed_at,
		NOT EXISTS(SELECT 1 FROM project_files pf WHERE pf.project_id = p.id)
		AND NOT EXISTS(SELECT 1 FROM project_meta pm WHERE pm.project_id = p.id)
		FROM projects p`)
	if err != nil {
		return CleanupStats{}, err
	}
	type project struct {
		id     int64
		path   string
		last   string
		unused bool
	}
	var stale []project
	for rows.Next() {
		var p project
		if err := rows.Scan(&p.id, &p.path, &p.last, &p.unused); err != nil {
			_ = rows.Close()
			return CleanupStats{}, err
		}
		accessed, parseErr := time.Parse(time.RFC3339, p.last)
		_, statErr := os.Stat(p.path)
		defaultUnused := p.path == ":default" && p.unused
		if defaultUnused || (p.path != ":default" && (os.IsNotExist(statErr) || parseErr != nil || accessed.Before(cutoff))) {
			stale = append(stale, p)
		}
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return CleanupStats{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return CleanupStats{}, err
	}
	defer func() { _ = tx.Rollback() }()
	for _, p := range stale {
		if _, err := tx.Exec(`DELETE FROM projects WHERE id = ?`, p.id); err != nil {
			return CleanupStats{}, err
		}
	}
	if err := gcAllUnreferencedTx(tx); err != nil {
		return CleanupStats{}, err
	}
	if err := tx.Commit(); err != nil {
		return CleanupStats{}, err
	}
	after, err := s.CollectionStats()
	if err != nil {
		return CleanupStats{}, err
	}
	_, _ = s.db.Exec(`PRAGMA incremental_vacuum`)
	if s.dsn != ":memory:" {
		_, _ = s.db.Exec(`PRAGMA wal_checkpoint(PASSIVE)`)
	}
	final, _ := s.CollectionStats()
	var projectsLeft int
	_ = s.reader().QueryRow(`SELECT count(*) FROM projects`).Scan(&projectsLeft)
	return CleanupStats{
		ProjectsRemoved: len(stale),
		VectorsRemoved:  before.UniqueVectors - after.UniqueVectors,
		BytesReclaimed:  max(int64(0), before.DatabaseBytes-final.DatabaseBytes),
		ProjectsLeft:    projectsLeft,
	}, nil
}

// CleanupCollectionAt opens dbPath only when it is a shared collection and
// performs project-aware cleanup. The boolean is false for legacy indexes.
func CleanupCollectionAt(dbPath string, cutoff time.Time) (CleanupStats, bool, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return CleanupStats{}, false, err
	}
	defer func() {
		if db != nil {
			_ = db.Close()
		}
	}()
	var shared bool
	if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = 'collection_meta')`).Scan(&shared); err != nil {
		return CleanupStats{}, false, err
	}
	if !shared {
		return CleanupStats{}, false, nil
	}
	var dimensions int
	var storage string
	if err := db.QueryRow(`SELECT value FROM collection_meta WHERE key = 'vec_dimensions'`).Scan(&dimensions); err != nil {
		return CleanupStats{}, true, err
	}
	if err := db.QueryRow(`SELECT value FROM collection_meta WHERE key = 'vector_storage'`).Scan(&storage); err != nil {
		return CleanupStats{}, true, err
	}
	_ = db.Close()
	db = nil
	s, err := openCollection(dbPath, dimensions, storage)
	if err != nil {
		return CleanupStats{}, true, err
	}
	defer func() { _ = s.Close() }()
	stats, err := s.CleanupStaleProjects(cutoff)
	return stats, true, err
}
