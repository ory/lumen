## Rating: Good

The candidate patch fixes the NPE by null-checking `contextElement` before calling `ownerDocument()`, which is logically equivalent to the gold patch's approach. However, it only modifies the main source file and omits the test file addition, and uses a ternary operator instead of wrapping the block in an `if` statement. The fix correctly prevents the NPE in the same scenario, but the missing test means `files_correct` is false.
