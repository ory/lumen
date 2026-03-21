//go:build windows

package cmd

// spawnBackgroundIndexer is a no-op on Windows. Windows is not supported.
func spawnBackgroundIndexer(_ string) {}
