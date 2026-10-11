// Copyright 2026 Aeneas Rekkas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package store

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	scanCacheInfix  = ".scan-"
	scanCacheSuffix = ".json"
)

// ScanCachePath returns the disposable freshness-scan sidecar for one
// worktree root of the collection at dbPath.
func ScanCachePath(dbPath, projectRoot string) string {
	return fmt.Sprintf("%s%s%x%s", dbPath, scanCacheInfix, sha256.Sum256([]byte(projectRoot)), scanCacheSuffix)
}

// removeOrphanScanCaches deletes completed scan sidecars whose worktree no
// longer has a membership. Only exact "<db>.scan-<sha256 hex>.json" names are
// considered: in-progress temporary files, lock files (which must keep their
// inode for flock), and unknown files are left alone. Callers must hold the
// exclusive collection guard.
func removeOrphanScanCaches(dbPath string, projectPaths []string) (int, error) {
	surviving := make(map[string]bool, len(projectPaths))
	for _, path := range projectPaths {
		surviving[filepath.Base(ScanCachePath(dbPath, path))] = true
	}
	entries, err := os.ReadDir(filepath.Dir(dbPath))
	if err != nil {
		return 0, err
	}
	prefix := filepath.Base(dbPath) + scanCacheInfix
	var removed int
	var errs []error
	for _, entry := range entries {
		name := entry.Name()
		if !entry.Type().IsRegular() || surviving[name] || !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, scanCacheSuffix) {
			continue
		}
		digest := strings.TrimSuffix(strings.TrimPrefix(name, prefix), scanCacheSuffix)
		if len(digest) != 2*sha256.Size || strings.ToLower(digest) != digest {
			continue
		}
		if _, err := hex.DecodeString(digest); err != nil {
			continue
		}
		if err := os.Remove(filepath.Join(filepath.Dir(dbPath), name)); err != nil && !os.IsNotExist(err) {
			errs = append(errs, err)
			continue
		}
		removed++
	}
	return removed, errors.Join(errs...)
}
