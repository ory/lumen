# SWE-Bench Detail Report

Generated: 2026-03-11 12:50 UTC

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
| **baseline** | 132.5s | 18 | 279064 | 5544 | $0.2200 |
| **with-lumen** | 104.8s | 13 | 168436 | 4750 | $0.2304 |

### baseline

**Rating: Good**

```diff
diff --git a/src/jv_aux.c b/src/jv_aux.c
index 47a622e..52166cf 100644
--- a/src/jv_aux.c
+++ b/src/jv_aux.c
@@ -276,7 +276,10 @@ static jv jv_dels(jv t, jv keys) {
     jv starts = jv_array(), ends = jv_array();
     jv_array_foreach(keys, i, key) {
       if (jv_get_kind(key) == JV_KIND_NUMBER) {
-        if (jv_number_value(key) < 0) {
+        double keyval = jv_number_value(key);
+        if (isnan(keyval)) {
+          jv_free(key); // NaN is not a valid array index; skip it
+        } else if (keyval < 0) {
           neg_keys = jv_array_append(neg_keys, key);
         } else {
           nonneg_keys = jv_array_append(nonneg_keys, key);
@@ -439,10 +442,11 @@ jv jv_getpath(jv root, jv path) {
 static jv delpaths_sorted(jv object, jv paths, int start) {
   jv delkeys = jv_array();
   for (int i=0; i<jv_array_length(jv_copy(paths));) {
-    int j = i;
     assert(jv_array_length(jv_array_get(jv_copy(paths), i)) > start);
     int delkey = jv_array_length(jv_array_get(jv_copy(paths), i)) == start + 1;
     jv key = jv_array_get(jv_array_get(jv_copy(paths), i), start);
+    // Start j at i+1: paths[i] always matches key; also handles NaN where jv_equal(nan,nan)==false
+    int j = i + 1;
     while (j < jv_array_length(jv_copy(paths)) &&
            jv_equal(jv_copy(key), jv_array_get(jv_array_get(jv_copy(paths), j), start)))
       j++;
```

### with-lumen

**Rating: INVALID (lumen not used)**

```diff
diff --git a/src/jv_aux.c b/src/jv_aux.c
index 47a622e..88310a1 100644
--- a/src/jv_aux.c
+++ b/src/jv_aux.c
@@ -276,7 +276,10 @@ static jv jv_dels(jv t, jv keys) {
     jv starts = jv_array(), ends = jv_array();
     jv_array_foreach(keys, i, key) {
       if (jv_get_kind(key) == JV_KIND_NUMBER) {
-        if (jv_number_value(key) < 0) {
+        double keyval = jv_number_value(key);
+        if (isnan(keyval)) {
+          jv_free(key); // NaN is not a valid array index; skip it
+        } else if (keyval < 0) {
           neg_keys = jv_array_append(neg_keys, key);
         } else {
           nonneg_keys = jv_array_append(nonneg_keys, key);
@@ -439,10 +442,13 @@ jv jv_getpath(jv root, jv path) {
 static jv delpaths_sorted(jv object, jv paths, int start) {
   jv delkeys = jv_array();
   for (int i=0; i<jv_array_length(jv_copy(paths));) {
-    int j = i;
     assert(jv_array_length(jv_array_get(jv_copy(paths), i)) > start);
     int delkey = jv_array_length(jv_array_get(jv_copy(paths), i)) == start + 1;
     jv key = jv_array_get(jv_array_get(jv_copy(paths), i), start);
+    // Start j at i+1: paths[i] always belongs to the group [i, j).
+    // Starting at i+1 (rather than i) avoids an infinite loop when key is NaN,
+    // since jv_equal(nan, nan) returns false per IEEE 754 semantics.
+    int j = i + 1;
     while (j < jv_array_length(jv_copy(paths)) &&
            jv_equal(jv_copy(key), jv_array_get(jv_array_get(jv_copy(paths), j), start)))
       j++;
```


