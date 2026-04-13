## Rating: Perfect

Both patches fix the same root cause: when `resize()` expands the `indicies` array, the new elements must be initialized to `-1` rather than the default `0`. The gold patch uses `IntArray(newSize) { -1 }` with `copyInto`, while the candidate uses `copyOf` and then explicitly sets the new elements to `-1` in a loop — both produce identical results. The candidate omits the test addition from the gold patch, but the core logic fix is functionally equivalent.
