## Rating: Perfect

Both patches fix the same root cause in `JsonPath.resize()`: new elements in the expanded `indicies` array must be initialized to `-1` rather than `0`. The gold patch creates a new `IntArray` pre-filled with `-1` then copies old values in; the candidate patch uses `copyOf` followed by `fill(-1, oldSize, newSize)` to initialize only the new slots — semantically identical. The test added in the candidate is slightly less thorough (hardcoded depth vs. 20-level parametric) but still exercises the regression correctly.
