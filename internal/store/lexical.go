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
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/ory/lumen/internal/chunker"
)

const lexicalChunkSchema = `CREATE TABLE IF NOT EXISTS chunk_defs (
 id INTEGER PRIMARY KEY,
 file_revision_id INTEGER NOT NULL REFERENCES file_revisions(id) ON DELETE CASCADE,
 ordinal INTEGER NOT NULL,
 chunk_key TEXT NOT NULL,
 symbol TEXT NOT NULL,
 kind TEXT NOT NULL,
 start_line INTEGER NOT NULL,
 end_line INTEGER NOT NULL,
 references_json TEXT NOT NULL,
 imports_json TEXT NOT NULL,
 exported INTEGER NOT NULL,
 UNIQUE(file_revision_id, ordinal)
)`

func createLexicalSchema(db *sql.DB) error {
	for _, stmt := range []string{
		`CREATE INDEX IF NOT EXISTS idx_chunk_defs_symbol ON chunk_defs(symbol)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS lexical_chunks USING fts5(path, symbol, signature, comments, body, content='', contentless_delete=1, detail=full, columnsize=1, tokenize="unicode61 tokenchars '_'")`,
		`CREATE TRIGGER IF NOT EXISTS lexical_chunk_delete AFTER DELETE ON chunk_defs BEGIN DELETE FROM lexical_chunks WHERE rowid=old.id; END`,
		// Per-term document frequencies for IDF-weighted query coverage.
		`CREATE VIRTUAL TABLE IF NOT EXISTS lexical_vocab USING fts5vocab(lexical_chunks, row)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("create lexical index: %w", err)
		}
	}
	return nil
}

// OpenLexicalCollection opens a model-free, repository-scoped lexical index.
// UseProject selects a worktree; lexical and vector indexes use separate files.
func OpenLexicalCollection(dsn string) (*Store, error) {
	s, err := openCollection(dsn, 0, "lexical")
	if err != nil && IsCorruptionErr(err) && dsn != ":memory:" {
		deleteDBFiles(dsn)
		s, err = openCollection(dsn, 0, "lexical")
	}
	if err != nil {
		return nil, err
	}
	if err := s.UseProject(""); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}

// Lexical reports whether this collection uses BM25 without embeddings.
func (s *Store) Lexical() bool { return s.lexical }

func (s *Store) storeLexicalRevision(path, hash string, chunks []chunker.Chunk) (bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.Exec(`INSERT OR IGNORE INTO file_revisions(relative_path, content_hash, complete) VALUES (?, ?, 0)`, path, hashBlob(hash))
	if err != nil {
		return false, err
	}
	inserted, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	var revisionID int64
	var complete bool
	if err := tx.QueryRow(`SELECT id, complete FROM file_revisions WHERE relative_path=? AND content_hash=?`, path, hashBlob(hash)).Scan(&revisionID, &complete); err != nil {
		return false, err
	}
	if !complete {
		for i, c := range chunks {
			refs, err := json.Marshal(c.References)
			if err != nil {
				return false, err
			}
			imports, err := json.Marshal(c.Imports)
			if err != nil {
				return false, err
			}
			row, err := tx.Exec(`INSERT INTO chunk_defs(file_revision_id, ordinal, chunk_key, symbol, kind, start_line, end_line, references_json, imports_json, exported) VALUES (?,?,?,?,?,?,?,?,?,?)`, revisionID, i, c.ID, c.Symbol, c.Kind, c.StartLine, c.EndLine, string(refs), string(imports), exportedColumn(c))
			if err != nil {
				return false, err
			}
			id, err := row.LastInsertId()
			if err != nil {
				return false, err
			}
			if _, err := tx.Exec(`INSERT INTO lexical_chunks(rowid,path,symbol,signature,comments,body) VALUES (?,?,?,?,?,?)`, id, identifierText(path), identifierText(c.Symbol), identifierText(c.Signature), identifierText(c.Comments), identifierText(c.Content)); err != nil {
				return false, err
			}
		}
		if _, err := tx.Exec(`UPDATE file_revisions SET complete=1 WHERE id=?`, revisionID); err != nil {
			return false, err
		}
	}
	displaced, err := replaceProjectFileTx(tx, s.projectID, path, revisionID)
	if err != nil {
		return false, err
	}
	if err := s.gcRevisionsTx(tx, displaced); err != nil {
		return false, err
	}
	return inserted > 0, tx.Commit()
}

// chunk_defs.exported values.
const (
	exportedNone    = 0
	exportedNamed   = 1
	exportedDefault = 2
)

func exportedColumn(c chunker.Chunk) int {
	switch {
	case c.Exported:
		return exportedNamed
	case c.DefaultExport:
		return exportedDefault
	}
	return exportedNone
}

// identifierText preserves complete identifiers and adds camelCase, acronym,
// snake_case and digit-boundary components. Both indexing and queries use it.
func identifierText(text string) string {
	var tokens []string
	for _, word := range strings.FieldsFunc(text, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' }) {
		tokens = append(tokens, word)
		runes := []rune(word)
		start := 0
		for i := 0; i <= len(runes); i++ {
			boundary := i == len(runes)
			if i < len(runes) {
				boundary = runes[i] == '_' || (i > start && (unicode.IsUpper(runes[i]) && (unicode.IsLower(runes[i-1]) || (i+1 < len(runes) && unicode.IsLower(runes[i+1]) && unicode.IsUpper(runes[i-1]))) || unicode.IsDigit(runes[i]) != unicode.IsDigit(runes[i-1])))
			}
			if boundary {
				if i > start && (start > 0 || i < len(runes)) {
					tokens = append(tokens, string(runes[start:i]))
				}
				start = i
				if i < len(runes) && runes[i] == '_' {
					start++
				}
			}
		}
	}
	return strings.Join(tokens, " ")
}

// LexicalTerms returns the normalized, deduplicated terms (at most 64) that a
// lexical query matches, in query order.
func LexicalTerms(query string) []string {
	words := strings.Fields(strings.ToLower(identifierText(query)))
	seen := make(map[string]bool, len(words))
	terms := make([]string, 0, len(words))
	for _, word := range words {
		if !seen[word] {
			terms = append(terms, word)
			seen[word] = true
		}
		if len(terms) == 64 {
			break
		}
	}
	return terms
}

// LexicalTokens returns the normalized token set indexed for text.
func LexicalTokens(text string) map[string]bool {
	tokens := map[string]bool{}
	for _, word := range strings.Fields(strings.ToLower(identifierText(text))) {
		tokens[word] = true
	}
	return tokens
}

func lexicalMatch(query string) string {
	terms := LexicalTerms(query)
	for i, term := range terms {
		terms[i] = `"` + term + `"`
	}
	return strings.Join(terms, " OR ")
}

const lexicalResultColumns = `pf.relative_path, cd.symbol, cd.kind, cd.start_line, cd.end_line, fr.content_hash, cd.references_json, cd.imports_json, cd.exported`
const lexicalResultJoins = ` JOIN chunk_defs cd ON cd.file_revision_id=pf.file_revision_id JOIN file_revisions fr ON fr.id=pf.file_revision_id `

// SearchLexical ranks symbol-sized candidates within the selected worktree.
// Path filters are literal file or directory paths and apply before the limit.
func (s *Store) SearchLexical(ctx context.Context, query string, limit int, pathFilter string) ([]SearchResult, error) {
	if !s.lexical {
		return nil, fmt.Errorf("SearchLexical requires a lexical collection")
	}
	match := lexicalMatch(query)
	if limit <= 0 || match == "" {
		return nil, nil
	}
	sqlQuery := `SELECT ` + lexicalResultColumns + `, bm25(lexical_chunks, 12, 16, 6, 3, 1) - CASE WHEN lower(cd.symbol)=lower(?) THEN 1000 WHEN lower(pf.relative_path)=lower(?) THEN 500 ELSE 0 END AS score FROM project_files pf ` + lexicalResultJoins + ` JOIN lexical_chunks ON lexical_chunks.rowid=cd.id WHERE lexical_chunks MATCH ? AND pf.project_id=?`
	args := []any{strings.TrimSpace(query), strings.TrimSpace(query), match, s.projectID}
	sqlQuery, args = lexicalPathFilter(sqlQuery, args, pathFilter)
	sqlQuery += ` ORDER BY score, pf.relative_path, cd.start_line, cd.id LIMIT ?`
	args = append(args, limit)
	rows, err := s.reader().QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("lexical search: %w", err)
	}
	return scanLexicalResults(rows)
}

// lexicalPathSeparator matches the native separator used by stored relative
// paths and filepath.Rel filters.
var lexicalPathSeparator = string(filepath.Separator)

func lexicalPathFilter(query string, args []any, path string) (string, []any) {
	path = strings.TrimSuffix(strings.TrimSuffix(path, "/"), lexicalPathSeparator)
	if path != "" {
		query += ` AND (pf.relative_path=? OR substr(pf.relative_path,1,length(?)+1)=? || ?)`
		args = append(args, path, path, path, lexicalPathSeparator)
	}
	return query, args
}

// ListProjectChunks returns structural metadata only for the active worktree.
func (s *Store) ListProjectChunks(pathFilter string) ([]SearchResult, error) {
	if !s.lexical {
		return nil, fmt.Errorf("ListProjectChunks requires a lexical collection")
	}
	query, args := lexicalPathFilter(`SELECT `+lexicalResultColumns+`, 0 FROM project_files pf `+lexicalResultJoins+` WHERE pf.project_id=?`, []any{s.projectID}, pathFilter)
	rows, err := s.reader().Query(query+` ORDER BY pf.relative_path,cd.start_line,cd.id`, args...)
	if err != nil {
		return nil, err
	}
	return scanLexicalResults(rows)
}

func scanLexicalResults(rows *sql.Rows) ([]SearchResult, error) {
	defer func() { _ = rows.Close() }()
	var results []SearchResult
	for rows.Next() {
		var result SearchResult
		var hash []byte
		var refs, imports string
		var exported int
		if err := rows.Scan(&result.FilePath, &result.Symbol, &result.Kind, &result.StartLine, &result.EndLine, &hash, &refs, &imports, &exported, &result.Distance); err != nil {
			return nil, err
		}
		result.Exported, result.DefaultExport = exported == exportedNamed, exported == exportedDefault
		result.ContentHash = hex.EncodeToString(hash)
		if err := json.Unmarshal([]byte(refs), &result.References); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(imports), &result.Imports); err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, rows.Err()
}

// LookupLexicalSymbols retrieves at most 257 candidates for bounded structural
// expansion. Callers must skip expansion when more than 256 rows are returned:
// the truncated set cannot establish whether a reference is unambiguous.
func (s *Store) LookupLexicalSymbols(ctx context.Context, symbols []string, pathFilter string) ([]SearchResult, error) {
	if !s.lexical {
		return nil, fmt.Errorf("LookupLexicalSymbols requires a lexical collection")
	}
	if len(symbols) == 0 {
		return nil, nil
	}
	if len(symbols) > 96 {
		return nil, fmt.Errorf("structural lookup exceeds 96 symbols")
	}
	marks := make([]string, len(symbols))
	args := []any{s.projectID}
	for i, symbol := range symbols {
		marks[i] = "?"
		args = append(args, symbol)
	}
	query := `SELECT ` + lexicalResultColumns + `, 0 FROM project_files pf ` + lexicalResultJoins + ` WHERE pf.project_id=? AND cd.symbol IN (` + strings.Join(marks, ",") + `)`
	query, args = lexicalPathFilter(query, args, pathFilter)
	rows, err := s.reader().QueryContext(ctx, query+` ORDER BY pf.relative_path,cd.start_line,cd.id LIMIT 257`, args...)
	if err != nil {
		return nil, err
	}
	return scanLexicalResults(rows)
}

// LookupLexicalDefaultExports returns the default-exported declarations of one
// file in the active worktree. A module has at most one default export; more
// than one row means the file cannot be resolved unambiguously.
func (s *Store) LookupLexicalDefaultExports(ctx context.Context, path string) ([]SearchResult, error) {
	if !s.lexical {
		return nil, fmt.Errorf("LookupLexicalDefaultExports requires a lexical collection")
	}
	rows, err := s.reader().QueryContext(ctx, `SELECT `+lexicalResultColumns+`, 0 FROM project_files pf `+lexicalResultJoins+` WHERE pf.project_id=? AND pf.relative_path=? AND cd.exported=? ORDER BY cd.start_line, cd.id LIMIT 2`, s.projectID, path, exportedDefault)
	if err != nil {
		return nil, err
	}
	return scanLexicalResults(rows)
}

// LexicalTermWeights returns an inverse document frequency for each term over
// the collection's lexical documents. Rare terms weigh more; a term occurring
// nowhere gets the maximum weight.
func (s *Store) LexicalTermWeights(ctx context.Context, terms []string) (map[string]float64, error) {
	if !s.lexical {
		return nil, fmt.Errorf("LexicalTermWeights requires a lexical collection")
	}
	weights := make(map[string]float64, len(terms))
	if len(terms) == 0 {
		return weights, nil
	}
	var total float64
	if err := s.reader().QueryRowContext(ctx, `SELECT count(*) FROM chunk_defs`).Scan(&total); err != nil {
		return nil, err
	}
	marks := make([]string, len(terms))
	args := make([]any, len(terms))
	for i, term := range terms {
		marks[i] = "?"
		args[i] = term
		weights[term] = math.Log(1 + (total+0.5)/0.5)
	}
	rows, err := s.reader().QueryContext(ctx, `SELECT term, doc FROM lexical_vocab WHERE term IN (`+strings.Join(marks, ",")+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var term string
		var docs float64
		if err := rows.Scan(&term, &docs); err != nil {
			return nil, err
		}
		weights[term] = math.Log(1 + (total-docs+0.5)/(docs+0.5))
	}
	return weights, rows.Err()
}
