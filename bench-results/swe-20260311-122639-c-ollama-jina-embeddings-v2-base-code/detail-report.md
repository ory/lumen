# SWE-Bench Detail Report

Generated: 2026-03-11 11:33 UTC

---

## c-hard [c]

**Issue:** Fix infinite loop and undefined behavior in `del(.[nan])`

> When using the `del` builtin with `nan` as an array index, jq enters an infinite loop and never terminates. For example, the expression `[1,2,3] | del(.[nan])` hangs indefinitely instead of returning a result.
> 
> The root cause is that `nan` is a valid jq value of kind JV_KIND_NUMBER, but NaN comparisons violate normal numeric ordering assumptions. Specifically:
> 
> 1. In the `jv_dels` function (which handles deletion of array elements by index), when the index is NaN, the code attempts to classify it as negative or non-negative. Since NaN is neither less than zero nor greater than or equal to zero, NaN indices fall through to the string/object key deletion path where they do not belong, leading to undefined behavior.
> 
> 2. In the `delpaths_sorted` function, a `while` loop uses `jv_equal` to group consecutive paths with matching leading keys. When the key is NaN, `jv_equal(nan, nan)` returns false (per IEEE 754 semantics), so the loop counter `j` never advances past `i`, causing an infinite loop.
> 
> Expected behavior:
> - `[1,2,3] | del(.[nan])` should return `[1,2,3]` unchanged (NaN is not a valid array index, so no element should be deleted).
> - `[1,2,3] | del(.[nan,nan])` should similarly return `[1,2,3]` unchanged.
> - The operation should complete in finite time without hanging or crashing.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 128.5s | 20 | 465919 | 7475 | $0.3709 |
| **with-lumen** | 100.6s | 27 | 371408 | 4839 | $0.2691 |

### baseline

**Rating: Good**

```diff
diff --git a/src/jv_aux.c b/src/jv_aux.c
index 47a622e..c74ac80 100644
--- a/src/jv_aux.c
+++ b/src/jv_aux.c
@@ -276,7 +276,11 @@ static jv jv_dels(jv t, jv keys) {
     jv starts = jv_array(), ends = jv_array();
     jv_array_foreach(keys, i, key) {
       if (jv_get_kind(key) == JV_KIND_NUMBER) {
-        if (jv_number_value(key) < 0) {
+        double keyval = jv_number_value(key);
+        if (isnan(keyval)) {
+          jv_free(key);
+          // NaN is not a valid array index; skip it
+        } else if (keyval < 0) {
           neg_keys = jv_array_append(neg_keys, key);
         } else {
           nonneg_keys = jv_array_append(nonneg_keys, key);
@@ -439,10 +443,13 @@ jv jv_getpath(jv root, jv path) {
 static jv delpaths_sorted(jv object, jv paths, int start) {
   jv delkeys = jv_array();
   for (int i=0; i<jv_array_length(jv_copy(paths));) {
-    int j = i;
     assert(jv_array_length(jv_array_get(jv_copy(paths), i)) > start);
     int delkey = jv_array_length(jv_array_get(jv_copy(paths), i)) == start + 1;
     jv key = jv_array_get(jv_array_get(jv_copy(paths), i), start);
+    // j always starts past i (paths[i] is unconditionally in group [i,j))
+    // This also handles NaN keys where jv_equal(nan,nan)==false would otherwise
+    // prevent j from advancing past i, causing an infinite loop.
+    int j = i + 1;
     while (j < jv_array_length(jv_copy(paths)) &&
            jv_equal(jv_copy(key), jv_array_get(jv_array_get(jv_copy(paths), j), start)))
       j++;
```

### with-lumen

**Rating: Poor**

```diff
diff --git a/src/jv_aux.c b/src/jv_aux.c
index 47a622e..18394d9 100644
--- a/src/jv_aux.c
+++ b/src/jv_aux.c
@@ -276,7 +276,9 @@ static jv jv_dels(jv t, jv keys) {
     jv starts = jv_array(), ends = jv_array();
     jv_array_foreach(keys, i, key) {
       if (jv_get_kind(key) == JV_KIND_NUMBER) {
-        if (jv_number_value(key) < 0) {
+        if (jvp_number_is_nan(key)) {
+          jv_free(key); // NaN is not a valid array index; skip
+        } else if (jv_number_value(key) < 0) {
           neg_keys = jv_array_append(neg_keys, key);
         } else {
           nonneg_keys = jv_array_append(nonneg_keys, key);
@@ -444,7 +446,7 @@ static jv delpaths_sorted(jv object, jv paths, int start) {
     int delkey = jv_array_length(jv_array_get(jv_copy(paths), i)) == start + 1;
     jv key = jv_array_get(jv_array_get(jv_copy(paths), i), start);
     while (j < jv_array_length(jv_copy(paths)) &&
-           jv_equal(jv_copy(key), jv_array_get(jv_array_get(jv_copy(paths), j), start)))
+           jv_identical(jv_copy(key), jv_array_get(jv_array_get(jv_copy(paths), j), start)))
       j++;
     // if i <= entry < j, then entry starts with key
     if (delkey) {
```


