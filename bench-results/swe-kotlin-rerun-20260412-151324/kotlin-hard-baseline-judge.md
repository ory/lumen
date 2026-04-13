## Rating: Good

Both patches fix the same root cause: after `resize()`, the newly allocated portion of `indicies` must be initialized to `-1` rather than `0`. The gold patch creates a new `IntArray(newSize) { -1 }` and copies old values in, while the candidate uses `copyOf` then fills the new portion with `-1` via `fill(oldSize, newSize)` — both produce the same final state. The test added in the candidate is slightly less robust (uses hardcoded JSON strings at fixed depths rather than the gold's programmatic 20-level nesting), but it still exercises the bug fix correctly.
