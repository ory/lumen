//go:build !windows

// Package indexlock provides advisory file locking for lumen index operations.
// It uses flock(2) so the OS automatically releases the lock when the holding
// process terminates for any reason — including SIGKILL — making stale locks
// impossible.
//
// Windows is not supported (flock is Unix-only). No-op stubs in
// lock_windows.go allow the package to compile there, but locking is disabled.
package indexlock

import (
	"os"
	"syscall"
)

// LockPathForDB returns the advisory lock file path for a given index DB path.
// The lock file lives alongside the DB in the same directory.
func LockPathForDB(dbPath string) string {
	return dbPath + ".lock"
}

// Lock is an exclusive advisory lock held on an index lock file.
// Release it when indexing is complete. Safe to call Release on a nil Lock.
type Lock struct {
	f *os.File
}

// TryAcquire attempts to acquire an exclusive non-blocking flock on lockPath.
//
//   - Returns (lock, nil) on success — the caller owns the lock.
//   - Returns (nil, nil) when another process already holds the lock (normal
//     case: a background indexer is already running — callers should skip).
//   - Returns (nil, err) only on unexpected OS errors (e.g. permissions).
func TryAcquire(lockPath string) (*Lock, error) {
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if err == syscall.EWOULDBLOCK {
			return nil, nil // another process holds the lock
		}
		return nil, err
	}
	return &Lock{f: f}, nil
}

// IsHeld reports whether another process currently holds an exclusive lock on
// lockPath. Returns false on any error (fail-open: callers proceed normally).
// Does NOT create the lock file — if it doesn't exist, no process holds it.
func IsHeld(lockPath string) bool {
	f, err := os.OpenFile(lockPath, os.O_RDONLY, 0)
	if err != nil {
		return false // ENOENT or permission error → no indexer running
	}
	defer func() { _ = f.Close() }()
	// A non-blocking shared lock succeeds only when no exclusive lock is held.
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_SH|syscall.LOCK_NB); err != nil {
		return err == syscall.EWOULDBLOCK
	}
	// Release the shared lock immediately — we only wanted to probe.
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return false
}

// Release releases the exclusive lock and closes the underlying file.
// Safe to call on a nil *Lock.
func (l *Lock) Release() {
	if l == nil || l.f == nil {
		return
	}
	_ = syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	_ = l.f.Close()
}
