//go:build !windows

package indexlock_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/ory/lumen/internal/indexlock"
)

func TestLockPathForDB(t *testing.T) {
	got := indexlock.LockPathForDB("/data/lumen/abc123/index.db")
	want := "/data/lumen/abc123/index.db.lock"
	if got != want {
		t.Errorf("LockPathForDB = %q, want %q", got, want)
	}
}

// TestTryAcquire_Free verifies acquiring a lock on a fresh path succeeds.
func TestTryAcquire_Free(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "index.db.lock")
	lock, err := indexlock.TryAcquire(lockPath)
	if err != nil {
		t.Fatalf("TryAcquire: unexpected error: %v", err)
	}
	if lock == nil {
		t.Fatal("TryAcquire: expected non-nil lock on free path")
	}
	lock.Release()
}

// TestTryAcquire_ReleasedIsReacquirable verifies that after Release the
// lock can be acquired again by the same process.
func TestTryAcquire_ReleasedIsReacquirable(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "index.db.lock")

	lock1, err := indexlock.TryAcquire(lockPath)
	if err != nil || lock1 == nil {
		t.Fatalf("first TryAcquire failed: err=%v lock=%v", err, lock1)
	}
	lock1.Release()

	lock2, err := indexlock.TryAcquire(lockPath)
	if err != nil {
		t.Fatalf("second TryAcquire: unexpected error: %v", err)
	}
	if lock2 == nil {
		t.Fatal("second TryAcquire: expected non-nil lock after Release")
	}
	lock2.Release()
}

// TestIsHeld_Free verifies IsHeld returns false when no lock is held.
func TestIsHeld_Free(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "index.db.lock")
	// First create the file (TryAcquire creates it), then release.
	lock, err := indexlock.TryAcquire(lockPath)
	if err != nil || lock == nil {
		t.Fatalf("TryAcquire: err=%v lock=%v", err, lock)
	}
	lock.Release()
	if indexlock.IsHeld(lockPath) {
		t.Error("IsHeld: expected false on free lock path")
	}
}

// TestIsHeld_NonexistentFile verifies IsHeld returns false when the lock file
// does not exist yet (no indexer has ever run for this project).
func TestIsHeld_NonexistentFile(t *testing.T) {
	// Directory does not exist — IsHeld must not panic; returns false.
	lockPath := filepath.Join(t.TempDir(), "nosuchdir", "index.db.lock")
	if indexlock.IsHeld(lockPath) {
		t.Error("IsHeld: expected false for non-existent path")
	}
}

// TestIsHeld_FileNotFound_InExistingDir verifies IsHeld returns false when the
// lock file does not exist but its parent directory does — the common cold-start
// case (project was indexed before, background indexer never ran).
// It also verifies IsHeld does NOT create the lock file as a side effect.
func TestIsHeld_FileNotFound_InExistingDir(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "index.db.lock") // dir exists, file does not
	if indexlock.IsHeld(lockPath) {
		t.Error("IsHeld: expected false when lock file doesn't exist in existing dir")
	}
	if _, err := os.Stat(lockPath); !errors.Is(err, os.ErrNotExist) {
		t.Error("IsHeld: must not create the lock file as a side effect")
	}
}

// TestReleaseNil verifies Release does not panic on nil.
func TestReleaseNil(t *testing.T) {
	var l *indexlock.Lock
	l.Release() // must not panic
}

// TestIsHeld_HeldByOtherProcess spawns a subprocess that holds the exclusive
// lock and verifies IsHeld returns true from the parent.
//
// The subprocess is this same test binary re-invoked with LUMEN_TEST_LOCK_HOLDER=1.
// It signals readiness by writing one byte to stdout (captured via a pipe).
func TestIsHeld_HeldByOtherProcess(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "index.db.lock")

	// Wire up a pipe so we know exactly when the subprocess has acquired the lock.
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestLockHolder_Helper")
	cmd.Env = append(os.Environ(),
		"LUMEN_TEST_LOCK_PATH="+lockPath,
		"LUMEN_TEST_LOCK_HOLDER=1",
	)
	cmd.Stdout = pw // helper writes ready byte here
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		_ = pr.Close()
		t.Fatalf("start subprocess: %v", err)
	}
	_ = pw.Close() // parent only reads

	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()

	// Wait for the ready byte with a timeout.
	readDone := make(chan error, 1)
	go func() {
		buf := make([]byte, 1)
		_, err := pr.Read(buf)
		_ = pr.Close()
		readDone <- err
	}()

	select {
	case readErr := <-readDone:
		if readErr != nil {
			t.Fatalf("waiting for subprocess ready signal: %v", readErr)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for subprocess to acquire lock")
	}

	if !indexlock.IsHeld(lockPath) {
		t.Error("IsHeld: expected true while subprocess holds exclusive lock")
	}
}

// TestLockHolder_Helper is the subprocess entry point for
// TestIsHeld_HeldByOtherProcess. It acquires the lock, writes one ready byte,
// then sleeps until killed by the parent.
func TestLockHolder_Helper(t *testing.T) {
	if os.Getenv("LUMEN_TEST_LOCK_HOLDER") != "1" {
		t.Skip("helper only runs when invoked as subprocess")
	}
	lockPath := os.Getenv("LUMEN_TEST_LOCK_PATH")
	lock, err := indexlock.TryAcquire(lockPath)
	if err != nil || lock == nil {
		t.Fatalf("helper: TryAcquire failed: err=%v lock=%v", err, lock)
	}
	// Signal readiness to parent.
	_, _ = os.Stdout.Write([]byte{1})
	// Hold lock until the parent kills this process.
	time.Sleep(30 * time.Second)
}
