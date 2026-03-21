// Package indexlock provides advisory file locking for lumen index operations.
// It uses github.com/gofrs/flock which wraps flock(2) on Unix and LockFileEx
// on Windows. The OS automatically releases the lock when the holding process
// terminates for any reason — including SIGKILL / TerminateProcess — making
// stale locks impossible on all platforms.
package indexlock

import (
	"os"

	"github.com/gofrs/flock"
)

// LockPathForDB returns the advisory lock file path for a given index DB path.
// The lock file lives alongside the DB in the same directory.
func LockPathForDB(dbPath string) string {
	return dbPath + ".lock"
}

// Lock is an exclusive advisory lock held on an index lock file.
// Release it when indexing is complete. Safe to call Release on a nil Lock.
type Lock struct {
	fl *flock.Flock
}

// TryAcquire attempts to acquire an exclusive non-blocking lock on lockPath.
//
//   - Returns (lock, nil) on success — the caller owns the lock.
//   - Returns (nil, nil) when another process already holds the lock (normal
//     case: a background indexer is already running — callers should skip).
//   - Returns (nil, err) only on unexpected OS errors (e.g. permissions).
func TryAcquire(lockPath string) (*Lock, error) {
	fl := flock.New(lockPath)
	locked, err := fl.TryLock()
	if err != nil {
		return nil, err
	}
	if !locked {
		return nil, nil // another process holds the lock
	}
	return &Lock{fl: fl}, nil
}

// IsHeld reports whether another process currently holds an exclusive lock on
// lockPath. Returns true on any error (fail-closed: callers skip work rather
// than risk concurrent writes). Does NOT create the lock file — if it doesn't
// exist, no process holds it.
func IsHeld(lockPath string) bool {
	if _, err := os.Stat(lockPath); err != nil {
		return false // file doesn't exist → no indexer running
	}
	fl := flock.New(lockPath)
	locked, err := fl.TryRLock()
	if err != nil {
		// Could not acquire shared lock → exclusive lock is held by another process.
		return true
	}
	if !locked {
		// TryRLock returned false without error → exclusive lock held.
		return true
	}
	// Shared lock succeeded → no exclusive lock held. Release immediately.
	_ = fl.Unlock()
	return false
}

// Release releases the exclusive lock and closes the underlying file.
// Safe to call on a nil *Lock.
func (l *Lock) Release() {
	if l == nil || l.fl == nil {
		return
	}
	_ = l.fl.Unlock()
}
