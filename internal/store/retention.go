// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package store

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"
)

// SetRevisionCachePolicy sets a collection-wide logical byte budget and TTL.
// The budget counts vectors conservatively once per retained revision, plus
// chunk metadata; SQLite page/WAL overhead is not included. Zero disables it.
func (s *Store) SetRevisionCachePolicy(bytes int64, ttl time.Duration) error {
	if bytes < 0 || ttl < 0 {
		return fmt.Errorf("cache budget and TTL must be nonnegative")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for key, value := range map[string]int64{"revision_cache_bytes": bytes, "revision_cache_seconds": int64(ttl / time.Second)} {
		if _, err := tx.Exec(`INSERT INTO collection_meta(key,value) VALUES (?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, strconv.FormatInt(value, 10)); err != nil {
			return err
		}
	}
	if err := pruneRevisionCacheTx(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func gcRevisionsTx(tx *sql.Tx, revisionIDs ...int64) error {
	for _, id := range revisionIDs {
		if id == 0 {
			continue
		}
		_, err := tx.Exec(`INSERT OR IGNORE INTO revision_cache(revision_id, unused_at, bytes)
   SELECT fr.id, ?, 128 + length(fr.relative_path) + COALESCE((
    SELECT sum(96 + length(cd.chunk_key) + length(cd.symbol) + length(cd.kind) +
      CAST((SELECT value FROM collection_meta WHERE key='vec_dimensions') AS INTEGER) *
      CASE (SELECT value FROM collection_meta WHERE key='vector_storage') WHEN 'float32' THEN 4 ELSE 1 END)
    FROM chunk_defs cd WHERE cd.file_revision_id=fr.id),0)
   FROM file_revisions fr WHERE fr.id=? AND fr.complete=1
    AND NOT EXISTS(SELECT 1 FROM project_files WHERE file_revision_id=fr.id)`, time.Now().Unix(), id)
		if err != nil {
			return err
		}
		var retained bool
		if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM revision_cache WHERE revision_id=?)`, id).Scan(&retained); err != nil {
			return err
		}
		if !retained {
			if err := deleteRevisionsTx(tx, id); err != nil {
				return err
			}
		}
	}
	return pruneRevisionCacheTx(tx)
}

func pruneRevisionCacheTx(tx *sql.Tx) error {
	budget, ttl := int64(256<<20), int64((7*24*time.Hour)/time.Second)
	for key, target := range map[string]*int64{"revision_cache_bytes": &budget, "revision_cache_seconds": &ttl} {
		var value string
		err := tx.QueryRow(`SELECT value FROM collection_meta WHERE key=?`, key).Scan(&value)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return err
		}
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed < 0 {
			return fmt.Errorf("invalid %s: %q", key, value)
		}
		*target = parsed
	}
	rows, err := tx.Query(`SELECT revision_id, unused_at, bytes FROM revision_cache ORDER BY unused_at DESC, revision_id DESC`)
	if err != nil {
		return err
	}
	var remove []int64
	var retained int64
	now := time.Now().Unix()
	for rows.Next() {
		var id, unused, size int64
		if err := rows.Scan(&id, &unused, &size); err != nil {
			_ = rows.Close()
			return err
		}
		if ttl == 0 || now-unused >= ttl || unused > now || size > budget-retained {
			remove = append(remove, id)
		} else {
			retained += size
		}
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return err
	}
	return deleteRevisionsTx(tx, remove...)
}
