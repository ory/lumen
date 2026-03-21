//go:build windows

// Package indexlock provides advisory file locking for lumen index operations.
// Windows is not supported. These stubs allow cross-compilation but provide
// no mutual-exclusion guarantees.
package indexlock

// LockPathForDB returns the lock file path alongside the DB.
func LockPathForDB(dbPath string) string { return dbPath + ".lock" }

// Lock is a no-op on Windows.
type Lock struct{}

// TryAcquire always succeeds on Windows (no-op).
func TryAcquire(_ string) (*Lock, error) { return &Lock{}, nil }

// IsHeld always returns false on Windows (no-op).
func IsHeld(_ string) bool { return false }

// Release is a no-op on Windows.
func (l *Lock) Release() {}
