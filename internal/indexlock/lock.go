// Package indexlock provides advisory file locking for lumen index operations.
// It uses github.com/gofrs/flock which wraps flock(2) on Unix and LockFileEx
// on Windows. The OS automatically releases the lock when the holding process
// terminates for any reason — including SIGKILL / TerminateProcess — making
// stale locks impossible on all platforms.
package indexlock

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/gofrs/flock"
)

// LockPathForDB returns the advisory lock file path for a given index DB path.
// The lock file lives alongside the DB in the same directory.
func LockPathForDB(dbPath string) string {
	return dbPath + ".lock"
}

// LockPathForProject returns a stable per-project lock alongside a shared
// collection. Different worktrees therefore do not serialize one another.
func LockPathForProject(dbPath, projectPath string) string {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(projectPath)))
	return dbPath + ".project-" + hash[:16] + ".lock"
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

// Acquire waits for an exclusive lock on lockPath or for ctx to be cancelled.
// The OS releases the lock if the process exits, so a crashed holder cannot
// leave a stale logical lock behind.
func Acquire(ctx context.Context, lockPath string) (*Lock, error) {
	fl := flock.New(lockPath)
	locked, err := fl.TryLockContext(ctx, 25*time.Millisecond)
	if err != nil {
		_ = fl.Close()
		return nil, err
	}
	if !locked {
		_ = fl.Close()
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return nil, errors.New("lock not acquired")
	}
	return &Lock{fl: fl}, nil
}

// TryAcquireShared takes a shared lock. Indexers hold this on the collection
// guard while taking an exclusive per-project lock; cleanup takes an exclusive
// probe and therefore never runs during active indexing.
func TryAcquireShared(lockPath string) (*Lock, error) {
	fl := flock.New(lockPath)
	locked, err := fl.TryRLock()
	if err != nil {
		return nil, err
	}
	if !locked {
		return nil, nil
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

// IsAnyHeld reports whether lockPath has either shared or exclusive holders.
// It is used by destructive collection maintenance.
func IsAnyHeld(lockPath string) bool {
	if _, err := os.Stat(lockPath); err != nil {
		return false
	}
	fl := flock.New(lockPath)
	locked, err := fl.TryLock()
	if err != nil || !locked {
		return true
	}
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
