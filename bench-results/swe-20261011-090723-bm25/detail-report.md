# SWE-Bench Detail Report

Generated: 2026-10-11 07:33 UTC

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
| **baseline** | 94.0s | 14 | 180630 | 1966 | $0.1237 |
| **with-lumen** | 50.0s | 16 | 216576 | 2470 | $0.1343 |

### baseline

**Rating: Good**

```diff
diff --git a/src/jv_aux.c b/src/jv_aux.c
index 47a622e..e8c5be1 100644
--- a/src/jv_aux.c
+++ b/src/jv_aux.c
@@ -276,7 +276,10 @@ static jv jv_dels(jv t, jv keys) {
     jv starts = jv_array(), ends = jv_array();
     jv_array_foreach(keys, i, key) {
       if (jv_get_kind(key) == JV_KIND_NUMBER) {
-        if (jv_number_value(key) < 0) {
+        if (isnan(jv_number_value(key))) {
+          // NaN is not a valid index, nothing to delete
+          jv_free(key);
+        } else if (jv_number_value(key) < 0) {
           neg_keys = jv_array_append(neg_keys, key);
         } else {
           nonneg_keys = jv_array_append(nonneg_keys, key);
@@ -444,7 +447,8 @@ static jv delpaths_sorted(jv object, jv paths, int start) {
     int delkey = jv_array_length(jv_array_get(jv_copy(paths), i)) == start + 1;
     jv key = jv_array_get(jv_array_get(jv_copy(paths), i), start);
     while (j < jv_array_length(jv_copy(paths)) &&
-           jv_equal(jv_copy(key), jv_array_get(jv_array_get(jv_copy(paths), j), start)))
+           (j == i || // always advance, as jv_equal(nan, nan) is false
+            jv_equal(jv_copy(key), jv_array_get(jv_array_get(jv_copy(paths), j), start))))
       j++;
     // if i <= entry < j, then entry starts with key
     if (delkey) {
diff --git a/tests/jq.test b/tests/jq.test
index 3729cc4..8f3e927 100644
--- a/tests/jq.test
+++ b/tests/jq.test
@@ -2542,3 +2542,12 @@ true
 reduce range(10001) as $_ ([];[.]) | tojson | contains("<skipped: too deep>")
 null
 true
+
+# Deleting with nan as an array index is a no-op
+del(.[nan])
+[1,2,3]
+[1,2,3]
+
+del(.[nan,nan])
+[1,2,3]
+[1,2,3]
```

### with-lumen

**Rating: INVALID (lumen not used)**

```diff
diff --git a/src/jv_aux.c b/src/jv_aux.c
index 47a622e..3118f80 100644
--- a/src/jv_aux.c
+++ b/src/jv_aux.c
@@ -276,7 +276,11 @@ static jv jv_dels(jv t, jv keys) {
     jv starts = jv_array(), ends = jv_array();
     jv_array_foreach(keys, i, key) {
       if (jv_get_kind(key) == JV_KIND_NUMBER) {
-        if (jv_number_value(key) < 0) {
+        double dkey = jv_number_value(key);
+        if (isnan(dkey)) {
+          // NaN is not a valid index, nothing to delete
+          jv_free(key);
+        } else if (dkey < 0) {
           neg_keys = jv_array_append(neg_keys, key);
         } else {
           nonneg_keys = jv_array_append(nonneg_keys, key);
@@ -439,7 +443,9 @@ jv jv_getpath(jv root, jv path) {
 static jv delpaths_sorted(jv object, jv paths, int start) {
   jv delkeys = jv_array();
   for (int i=0; i<jv_array_length(jv_copy(paths));) {
-    int j = i;
+    // entry i starts with key by definition; this also guarantees progress
+    // when key is NaN, for which jv_equal(key, key) is false
+    int j = i + 1;
     assert(jv_array_length(jv_array_get(jv_copy(paths), i)) > start);
     int delkey = jv_array_length(jv_array_get(jv_copy(paths), i)) == start + 1;
     jv key = jv_array_get(jv_array_get(jv_copy(paths), i), start);
diff --git a/tests/jq.test b/tests/jq.test
index 3729cc4..71dcf5f 100644
--- a/tests/jq.test
+++ b/tests/jq.test
@@ -2542,3 +2542,11 @@ true
 reduce range(10001) as $_ ([];[.]) | tojson | contains("<skipped: too deep>")
 null
 true
+
+del(.[nan])
+[1,2,3]
+[1,2,3]
+
+del(.[nan,nan])
+[1,2,3]
+[1,2,3]
```

---

## cpp-hard [cpp]

**Issue:** Consider adding a C API

> There is a well-defined binary interface between top-level functions such as `fmt::format` and their type-erased counterparts like `fmt::vformat`. This interface relies only on basic C++ features and should be compatible with C (except for the overload returning `std::string` obviously). As a result, it may be possible to provide a C API that builds argument lists and invokes the type-erased functions.
> 
> Unlike the `printf` family of functions, such an API could be fully type-safe, including for runtime format strings, and would not rely on encoding type information in the format string itself.
> 
> It would be great to have a C-compatible interface so that C projects and other languages with C FFI can use fmt's formatting capabilities without needing a C++ compiler.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 156.8s | 20 | 329156 | 11574 | $0.2758 |
| **with-lumen** | 145.3s | 28 | 575370 | 16972 | $0.4102 |

### baseline

**Rating: Good**

```diff
diff --git a/CMakeLists.txt b/CMakeLists.txt
index 53afc51..d0b088b 100644
--- a/CMakeLists.txt
+++ b/CMakeLists.txt
@@ -299,10 +299,10 @@ endfunction()
 
 # Define the fmt library, its includes and the needed defines.
 set(FMT_HEADERS)
-add_headers(FMT_HEADERS args.h base.h chrono.h color.h compile.h core.h format.h
-                        format-inl.h os.h ostream.h printf.h ranges.h std.h
+add_headers(FMT_HEADERS args.h base.h chrono.h color.h compile.h core.h fmt-c.h
+                        format.h format-inl.h os.h ostream.h printf.h ranges.h std.h
                         xchar.h)
-set(FMT_SOURCES src/format.cc)
+set(FMT_SOURCES src/format.cc src/fmt-c.cc)
 
 add_module_library(fmt src/fmt.cc FALLBACK
                    ${FMT_SOURCES} ${FMT_HEADERS} README.md ChangeLog.md
diff --git a/include/fmt/fmt-c.h b/include/fmt/fmt-c.h
new file mode 100644
index 0000000..d3b585b
--- /dev/null
+++ b/include/fmt/fmt-c.h
@@ -0,0 +1,207 @@
+// Formatting library for C++ - C API
+//
+// Copyright (c) 2012 - present, Victor Zverovich
+// All rights reserved.
+//
+// For the license information refer to format.h.
+
+#ifndef FMT_C_H_
+#define FMT_C_H_
+
+#include <stdio.h>
+#include <stddef.h>
+
+#ifdef __cplusplus
+extern "C" {
+#endif
+
+typedef enum {
+  fmt_type_int,
+  fmt_type_uint,
+  fmt_type_long_long,
+  fmt_type_ulong_long,
+  fmt_type_double,
+  fmt_type_long_double,
+  fmt_type_char,
+  fmt_type_bool,
+  fmt_type_string,
+  fmt_type_pointer
+} fmt_type;
+
+// A type-tagged format argument.
+typedef struct {
+  fmt_type type;
+  union {
+    int int_value;
+    unsigned uint_value;
+    long long long_long_value;
+    unsigned long long ulong_long_value;
+    double double_value;
+    long double long_double_value;
+    char char_value;
+    int bool_value;
+    const char* string_value;
+    const void* pointer_value;
+  } value;
+} fmt_arg;
+
+static inline fmt_arg fmt_from_int(int x) {
+  fmt_arg a;
+  a.type = fmt_type_int;
+  a.value.int_value = x;
+  return a;
+}
+static inline fmt_arg fmt_from_uint(unsigned x) {
+  fmt_arg a;
+  a.type = fmt_type_uint;
+  a.value.uint_value = x;
+  return a;
+}
+static inline fmt_arg fmt_from_long_long(long long x) {
+  fmt_arg a;
+  a.type = fmt_type_long_long;
+  a.value.long_long_value = x;
+  return a;
+}
+static inline fmt_arg fmt_from_ulong_long(unsigned long long x) {
+  fmt_arg a;
+  a.type = fmt_type_ulong_long;
+  a.value.ulong_long_value = x;
+  return a;
+}
+static inline fmt_arg fmt_from_double(double x) {
+  fmt_arg a;
+  a.type = fmt_type_double;
+  a.value.double_value = x;
+  return a;
+}
+static inline fmt_arg fmt_from_long_double(long double x) {
+  fmt_arg a;
+  a.type = fmt_type_long_double;
+  a.value.long_double_value = x;
+  return a;
+}
+static inline fmt_arg fmt_from_char(char x) {
+  fmt_arg a;
+  a.type = fmt_type_char;
+  a.value.char_value = x;
+  return a;
+}
+static inline fmt_arg fmt_from_bool(int x) {
+  fmt_arg a;
+  a.type = fmt_type_bool;
+  a.value.bool_value = x != 0;
+  return a;
+}
+static inline fmt_arg fmt_from_string(const char* x) {
+  fmt_arg a;
+  a.type = fmt_type_string;
+  a.value.string_value = x;
+  return a;
+}
+static inline fmt_arg fmt_from_pointer(const void* x) {
+  fmt_arg a;
+  a.type = fmt_type_pointer;
+  a.value.pointer_value = x;
+  return a;
+}
+
+// Formats `args` according to the format string `fmt` into `buffer` of `size`
+// bytes. The output is truncated if it doesn't fit and is null-terminated
+// unless `size` is zero. Returns the number of characters that would have been
+// written given enough space (excluding the terminating null) or a negative
+// value on error (e.g. an invalid format string).
+int fmt_vformat(char* buffer, size_t size, const char* fmt,
+                const fmt_arg* args, size_t num_args);
+
+// Formats `args` according to the format string `fmt` and writes the result to
+// the stream `f`. Returns the number of characters written or a negative value
+// on error.
+int fmt_vfprint(FILE* f, const char* fmt, const fmt_arg* args,
+                size_t num_args);
+
+// Implementation details: `args[0]` is the format string.
+int fmt_format_packed(char* buffer, size_t size, const fmt_arg* args,
+                      size_t num_args);
+int fmt_fprint_packed(FILE* f, const fmt_arg* args, size_t num_args);
+
+#ifdef __cplusplus
+}  // extern "C"
+#endif
+
+#ifndef __cplusplus
+
+#define FMT_MAKE_ARG(x)                                       \
+  _Generic((x),                                               \
+      _Bool: fmt_from_bool,                                   \
+      char: fmt_from_char,                                    \
+      signed char: fmt_from_int,                              \
+      short: fmt_from_int,                                    \
+      int: fmt_from_int,                                      \
+      long: fmt_from_long_long,                               \
+      long long: fmt_from_long_long,                          \
+      unsigned char: fmt_from_uint,                           \
+      unsigned short: fmt_from_uint,                          \
+      unsigned: fmt_from_uint,                                \
+      unsigned long: fmt_from_ulong_long,                     \
+      unsigned long long: fmt_from_ulong_long,                \
+      float: fmt_from_double,                                 \
+      double: fmt_from_double,                                \
+      long double: fmt_from_long_double,                      \
+      char*: fmt_from_string,                                 \
+      const char*: fmt_from_string,                           \
+      void*: fmt_from_pointer,                                \
+      const void*: fmt_from_pointer)(x)
+
+#define FMT_NARGS_(_1, _2, _3, _4, _5, _6, _7, _8, _9, _10, _11, _12, _13, \
+                   _14, _15, _16, n, ...)                                  \
+  n
+#define FMT_NARGS(...)                                                       \
+  FMT_NARGS_(__VA_ARGS__, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, \
+             1, 0)
+
+#define FMT_MAP_1(x) FMT_MAKE_ARG(x)
+#define FMT_MAP_2(x, ...) FMT_MAKE_ARG(x), FMT_MAP_1(__VA_ARGS__)
+#define FMT_MAP_3(x, ...) FMT_MAKE_ARG(x), FMT_MAP_2(__VA_ARGS__)
+#define FMT_MAP_4(x, ...) FMT_MAKE_ARG(x), FMT_MAP_3(__VA_ARGS__)
+#define FMT_MAP_5(x, ...) FMT_MAKE_ARG(x), FMT_MAP_4(__VA_ARGS__)
+#define FMT_MAP_6(x, ...) FMT_MAKE_ARG(x), FMT_MAP_5(__VA_ARGS__)
+#define FMT_MAP_7(x, ...) FMT_MAKE_ARG(x), FMT_MAP_6(__VA_ARGS__)
+#define FMT_MAP_8(x, ...) FMT_MAKE_ARG(x), FMT_MAP_7(__VA_ARGS__)
+#define FMT_MAP_9(x, ...) FMT_MAKE_ARG(x), FMT_MAP_8(__VA_ARGS__)
+#define FMT_MAP_10(x, ...) FMT_MAKE_ARG(x), FMT_MAP_9(__VA_ARGS__)
+#define FMT_MAP_11(x, ...) FMT_MAKE_ARG(x), FMT_MAP_10(__VA_ARGS__)
+#define FMT_MAP_12(x, ...) FMT_MAKE_ARG(x), FMT_MAP_11(__VA_ARGS__)
+#define FMT_MAP_13(x, ...) FMT_MAKE_ARG(x), FMT_MAP_12(__VA_ARGS__)
+#define FMT_MAP_14(x, ...) FMT_MAKE_ARG(x), FMT_MAP_13(__VA_ARGS__)
+#define FMT_MAP_15(x, ...) FMT_MAKE_ARG(x), FMT_MAP_14(__VA_ARGS__)
+#define FMT_MAP_16(x, ...) FMT_MAKE_ARG(x), FMT_MAP_15(__VA_ARGS__)
+
+#define FMT_CONCAT_(a, b) a##b
+#define FMT_CONCAT(a, b) FMT_CONCAT_(a, b)
+#define FMT_MAP(...) FMT_CONCAT(FMT_MAP_, FMT_NARGS(__VA_ARGS__))(__VA_ARGS__)
+
+// Formats arguments according to the format string `fmt` into `buffer`; see
+// fmt_vformat. The format string must be the first argument after `size`:
+//
+//   char buf[100];
+//   fmt_format(buf, sizeof(buf), "The answer is {}", 42);
+//
+// At most 15 formatting arguments are supported.
+#define fmt_format(buffer, size, ...)                                  \
+  fmt_format_packed(buffer, size, (const fmt_arg[]){FMT_MAP(__VA_ARGS__)}, \
+                    FMT_NARGS(__VA_ARGS__))
+
+// Formats arguments according to the format string `fmt` and writes the result
+// to the stream `f`; see fmt_vfprint.
+#define fmt_fprint(f, ...) \
+  fmt_fprint_packed(f, (const fmt_arg[]){FMT_MAP(__VA_ARGS__)}, \
+                    FMT_NARGS(__VA_ARGS__))
+
+// Formats arguments according to the format string `fmt` and writes the result
+// to stdout.
+#define fmt_print(...) fmt_fprint(stdout, __VA_ARGS__)
+
+#endif  // __cplusplus
+
+#endif  // FMT_C_H_
diff --git a/src/fmt-c.cc b/src/fmt-c.cc
new file mode 100644
index 0000000..4a291c2
--- /dev/null
+++ b/src/fmt-c.cc
@@ -0,0 +1,77 @@
+// Formatting library for C++ - C API
+//
+// Copyright (c) 2012 - present, Victor Zverovich
+// All rights reserved.
+//
+// For the license information refer to format.h.
+
+#include "fmt/fmt-c.h"
+
+#include "fmt/args.h"
+#include "fmt/format.h"
+
+namespace {
+
+using arg_store = fmt::dynamic_format_arg_store<fmt::format_context>;
+
+void push_args(arg_store& store, const fmt_arg* args, size_t num_args) {
+  for (size_t i = 0; i < num_args; ++i) {
+    const fmt_arg& a = args[i];
+    switch (a.type) {
+    case fmt_type_int:        store.push_back(a.value.int_value); break;
+    case fmt_type_uint:       store.push_back(a.value.uint_value); break;
+    case fmt_type_long_long:  store.push_back(a.value.long_long_value); break;
+    case fmt_type_ulong_long: store.push_back(a.value.ulong_long_value); break;
+    case fmt_type_double:     store.push_back(a.value.double_value); break;
+    case fmt_type_long_double:
+      store.push_back(a.value.long_double_value);
+      break;
+    case fmt_type_char:    store.push_back(a.value.char_value); break;
+    case fmt_type_bool:    store.push_back(a.value.bool_value != 0); break;
+    case fmt_type_string:  store.push_back(a.value.string_value); break;
+    case fmt_type_pointer: store.push_back(a.value.pointer_value); break;
+    }
+  }
+}
+
+}  // namespace
+
+extern "C" {
+
+int fmt_vformat(char* buffer, size_t size, const char* fmt,
+                const fmt_arg* args, size_t num_args) {
+  try {
+    arg_store store;
+    push_args(store, args, num_args);
+    auto result = fmt::vformat_to_n(buffer, size, fmt, store);
+    if (size != 0) buffer[result.size < size ? result.size : size - 1] = '\0';
+    return static_cast<int>(result.size);
+  } catch (...) {
+    return -1;
+  }
+}
+
+int fmt_vfprint(FILE* f, const char* fmt, const fmt_arg* args,
+                size_t num_args) {
+  try {
+    arg_store store;
+    push_args(store, args, num_args);
+    auto s = fmt::vformat(fmt, store);
+    if (fwrite(s.data(), 1, s.size(), f) != s.size()) return -1;
+    return static_cast<int>(s.size());
+  } catch (...) {
+    return -1;
+  }
+}
+
+int fmt_format_packed(char* buffer, size_t size, const fmt_arg* args,
+                      size_t num_args) {
+  return fmt_vformat(buffer, size, args[0].value.string_value, args + 1,
+                     num_args - 1);
+}
+
+int fmt_fprint_packed(FILE* f, const fmt_arg* args, size_t num_args) {
+  return fmt_vfprint(f, args[0].value.string_value, args + 1, num_args - 1);
+}
+
+}  // extern "C"
diff --git a/test/CMakeLists.txt b/test/CMakeLists.txt
index 3f7d24a..13320bc 100644
--- a/test/CMakeLists.txt
+++ b/test/CMakeLists.txt
@@ -50,6 +50,7 @@ add_fmt_test(base-test)
 add_fmt_test(assert-test)
 add_fmt_test(chrono-test)
 add_fmt_test(color-test)
+add_fmt_test(fmt-c-test)
 add_fmt_test(gtest-extra-test)
 add_fmt_test(format-test mock-allocator.h)
 if (MSVC)
diff --git a/test/fmt-c-test.cc b/test/fmt-c-test.cc
new file mode 100644
index 0000000..4c1c51d
--- /dev/null
+++ b/test/fmt-c-test.cc
@@ -0,0 +1,36 @@
+// Formatting library for C++ - tests of the C API
+//
+// Copyright (c) 2012 - present, Victor Zverovich
+// All rights reserved.
+//
+// For the license information refer to format.h.
+
+#include "fmt/fmt-c.h"
+
+#include <string.h>
+
+#include "gtest/gtest.h"
+
+TEST(fmt_c_test, format) {
+  char buf[64];
+  fmt_arg args[] = {fmt_from_int(42), fmt_from_string("hi"),
+                    fmt_from_double(1.25), fmt_from_char('c'),
+                    fmt_from_bool(1), fmt_from_ulong_long(7)};
+  int n = fmt_vformat(buf, sizeof(buf), "{} {} {:.2f} {} {} {}", args, 6);
+  EXPECT_EQ(n, 19);
+  EXPECT_STREQ(buf, "42 hi 1.25 c true 7");
+}
+
+TEST(fmt_c_test, truncation) {
+  char buf[4];
+  fmt_arg arg = fmt_from_int(123456);
+  EXPECT_EQ(fmt_vformat(buf, sizeof(buf), "{}", &arg, 1), 6);
+  EXPECT_STREQ(buf, "123");
+}
+
+TEST(fmt_c_test, error) {
+  char buf[16];
+  fmt_arg arg = fmt_from_string("x");
+  EXPECT_LT(fmt_vformat(buf, sizeof(buf), "{:d}", &arg, 1), 0);
+  EXPECT_LT(fmt_vformat(buf, sizeof(buf), "{} {}", &arg, 1), 0);
+}
```

### with-lumen

**Rating: INVALID (lumen not used)**

```diff
diff --git a/CMakeLists.txt b/CMakeLists.txt
index 53afc51..7e53746 100644
--- a/CMakeLists.txt
+++ b/CMakeLists.txt
@@ -299,10 +299,10 @@ endfunction()
 
 # Define the fmt library, its includes and the needed defines.
 set(FMT_HEADERS)
-add_headers(FMT_HEADERS args.h base.h chrono.h color.h compile.h core.h format.h
-                        format-inl.h os.h ostream.h printf.h ranges.h std.h
-                        xchar.h)
-set(FMT_SOURCES src/format.cc)
+add_headers(FMT_HEADERS args.h base.h chrono.h color.h compile.h core.h fmt-c.h
+                        format.h format-inl.h os.h ostream.h printf.h ranges.h
+                        std.h xchar.h)
+set(FMT_SOURCES src/format.cc src/fmt-c.cc)
 
 add_module_library(fmt src/fmt.cc FALLBACK
                    ${FMT_SOURCES} ${FMT_HEADERS} README.md ChangeLog.md
diff --git a/include/fmt/fmt-c.h b/include/fmt/fmt-c.h
new file mode 100644
index 0000000..6a7f1c2
--- /dev/null
+++ b/include/fmt/fmt-c.h
@@ -0,0 +1,240 @@
+/*
+ * Formatting library for C - the C API
+ *
+ * Copyright (c) 2012 - present, Victor Zverovich
+ * All rights reserved.
+ *
+ * For the license information refer to format.h.
+ */
+
+#ifndef FMT_FMT_C_H_
+#define FMT_FMT_C_H_
+
+#include <stdbool.h>
+#include <stddef.h>
+#include <stdio.h>
+
+#if defined(FMT_SHARED) && defined(_WIN32)
+#  define FMT_C_API __declspec(dllimport)
+#elif defined(FMT_LIB_EXPORT) && defined(_WIN32)
+#  define FMT_C_API __declspec(dllexport)
+#elif defined(FMT_LIB_EXPORT) && defined(__GNUC__)
+#  define FMT_C_API __attribute__((visibility("default")))
+#else
+#  define FMT_C_API
+#endif
+
+#ifdef __cplusplus
+extern "C" {
+#endif
+
+/* The type of a formatting argument. */
+typedef enum {
+  fmt_type_int,
+  fmt_type_uint,
+  fmt_type_bool,
+  fmt_type_char,
+  fmt_type_double,
+  fmt_type_long_double,
+  fmt_type_cstring,
+  fmt_type_pointer
+} fmt_type;
+
+/*
+ * A formatting argument. Use `fmt_from_*` functions or `fmt_make_arg` to
+ * create it. Strings and pointers are not copied, so the referenced data must
+ * outlive the formatting call.
+ */
+typedef struct {
+  fmt_type type;
+  union {
+    long long int_value;
+    unsigned long long uint_value;
+    bool bool_value;
+    char char_value;
+    double double_value;
+    long double long_double_value;
+    const char* cstring_value;
+    const void* pointer_value;
+  } value;
+} fmt_arg;
+
+static inline fmt_arg fmt_from_int(long long x) {
+  fmt_arg a;
+  a.type = fmt_type_int;
+  a.value.int_value = x;
+  return a;
+}
+static inline fmt_arg fmt_from_uint(unsigned long long x) {
+  fmt_arg a;
+  a.type = fmt_type_uint;
+  a.value.uint_value = x;
+  return a;
+}
+static inline fmt_arg fmt_from_bool(bool x) {
+  fmt_arg a;
+  a.type = fmt_type_bool;
+  a.value.bool_value = x;
+  return a;
+}
+static inline fmt_arg fmt_from_char(char x) {
+  fmt_arg a;
+  a.type = fmt_type_char;
+  a.value.char_value = x;
+  return a;
+}
+static inline fmt_arg fmt_from_double(double x) {
+  fmt_arg a;
+  a.type = fmt_type_double;
+  a.value.double_value = x;
+  return a;
+}
+static inline fmt_arg fmt_from_long_double(long double x) {
+  fmt_arg a;
+  a.type = fmt_type_long_double;
+  a.value.long_double_value = x;
+  return a;
+}
+static inline fmt_arg fmt_from_cstring(const char* x) {
+  fmt_arg a;
+  a.type = fmt_type_cstring;
+  a.value.cstring_value = x;
+  return a;
+}
+static inline fmt_arg fmt_from_pointer(const void* x) {
+  fmt_arg a;
+  a.type = fmt_type_pointer;
+  a.value.pointer_value = x;
+  return a;
+}
+
+/*
+ * Formats `args` according to the format string `fmt` and writes the output to
+ * `buffer`, writing at most `size` bytes including the terminating null
+ * character (nothing is written if `size` is zero). Returns the number of
+ * characters that would have been written if the buffer was large enough,
+ * excluding the terminating null character, or a negative value on error
+ * (e.g. an invalid format string).
+ */
+FMT_C_API int fmt_vformat(char* buffer, size_t size, const char* fmt,
+                          const fmt_arg* args, size_t num_args);
+
+/*
+ * Formats `args` according to `fmt` and writes the output to the stream `f`.
+ * Returns the number of characters written or a negative value on error.
+ */
+FMT_C_API int fmt_vfprint(FILE* f, const char* fmt, const fmt_arg* args,
+                          size_t num_args);
+
+#ifndef __cplusplus
+
+/* Creates a `fmt_arg` from a value of a supported type (requires C11). */
+#define fmt_make_arg(x)            \
+  _Generic((x),                    \
+      bool: fmt_from_bool,         \
+      char: fmt_from_char,         \
+      signed char: fmt_from_int,   \
+      short: fmt_from_int,         \
+      int: fmt_from_int,           \
+      long: fmt_from_int,          \
+      long long: fmt_from_int,     \
+      unsigned char: fmt_from_uint,       \
+      unsigned short: fmt_from_uint,      \
+      unsigned: fmt_from_uint,            \
+      unsigned long: fmt_from_uint,       \
+      unsigned long long: fmt_from_uint,  \
+      float: fmt_from_double,             \
+      double: fmt_from_double,            \
+      long double: fmt_from_long_double,  \
+      char*: fmt_from_cstring,            \
+      const char*: fmt_from_cstring,      \
+      void*: fmt_from_pointer,            \
+      const void*: fmt_from_pointer)(x)
+
+#define FMT_C_CAT_(a, b) a##b
+#define FMT_C_CAT(a, b) FMT_C_CAT_(a, b)
+
+/* Argument counting and mapping, supports up to 16 formatting arguments. */
+#define FMT_C_NARGS_(_1, _2, _3, _4, _5, _6, _7, _8, _9, _10, _11, _12, _13, _14, _15, _16, _17, n, ...) n
+#define FMT_C_NARGS(...) FMT_C_NARGS_(__VA_ARGS__, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0)
+
+#define FMT_C_MAP_1(f, a) f(a)
+#define FMT_C_MAP_2(f, a, ...) f(a), FMT_C_MAP_1(f, __VA_ARGS__)
+#define FMT_C_MAP_3(f, a, ...) f(a), FMT_C_MAP_2(f, __VA_ARGS__)
+#define FMT_C_MAP_4(f, a, ...) f(a), FMT_C_MAP_3(f, __VA_ARGS__)
+#define FMT_C_MAP_5(f, a, ...) f(a), FMT_C_MAP_4(f, __VA_ARGS__)
+#define FMT_C_MAP_6(f, a, ...) f(a), FMT_C_MAP_5(f, __VA_ARGS__)
+#define FMT_C_MAP_7(f, a, ...) f(a), FMT_C_MAP_6(f, __VA_ARGS__)
+#define FMT_C_MAP_8(f, a, ...) f(a), FMT_C_MAP_7(f, __VA_ARGS__)
+#define FMT_C_MAP_9(f, a, ...) f(a), FMT_C_MAP_8(f, __VA_ARGS__)
+#define FMT_C_MAP_10(f, a, ...) f(a), FMT_C_MAP_9(f, __VA_ARGS__)
+#define FMT_C_MAP_11(f, a, ...) f(a), FMT_C_MAP_10(f, __VA_ARGS__)
+#define FMT_C_MAP_12(f, a, ...) f(a), FMT_C_MAP_11(f, __VA_ARGS__)
+#define FMT_C_MAP_13(f, a, ...) f(a), FMT_C_MAP_12(f, __VA_ARGS__)
+#define FMT_C_MAP_14(f, a, ...) f(a), FMT_C_MAP_13(f, __VA_ARGS__)
+#define FMT_C_MAP_15(f, a, ...) f(a), FMT_C_MAP_14(f, __VA_ARGS__)
+#define FMT_C_MAP_16(f, a, ...) f(a), FMT_C_MAP_15(f, __VA_ARGS__)
+#define FMT_C_MAP(f, ...) \
+  FMT_C_CAT(FMT_C_MAP_, FMT_C_NARGS(__VA_ARGS__))(f, __VA_ARGS__)
+
+#define FMT_C_ARGS_1(fmt) fmt, NULL, 0
+#define FMT_C_ARGS_2(fmt, ...) \
+  fmt, (fmt_arg[]){FMT_C_MAP(fmt_make_arg, __VA_ARGS__)}, 1
+#define FMT_C_ARGS_3(fmt, ...) \
+  fmt, (fmt_arg[]){FMT_C_MAP(fmt_make_arg, __VA_ARGS__)}, 2
+#define FMT_C_ARGS_4(fmt, ...) \
+  fmt, (fmt_arg[]){FMT_C_MAP(fmt_make_arg, __VA_ARGS__)}, 3
+#define FMT_C_ARGS_5(fmt, ...) \
+  fmt, (fmt_arg[]){FMT_C_MAP(fmt_make_arg, __VA_ARGS__)}, 4
+#define FMT_C_ARGS_6(fmt, ...) \
+  fmt, (fmt_arg[]){FMT_C_MAP(fmt_make_arg, __VA_ARGS__)}, 5
+#define FMT_C_ARGS_7(fmt, ...) \
+  fmt, (fmt_arg[]){FMT_C_MAP(fmt_make_arg, __VA_ARGS__)}, 6
+#define FMT_C_ARGS_8(fmt, ...) \
+  fmt, (fmt_arg[]){FMT_C_MAP(fmt_make_arg, __VA_ARGS__)}, 7
+#define FMT_C_ARGS_9(fmt, ...) \
+  fmt, (fmt_arg[]){FMT_C_MAP(fmt_make_arg, __VA_ARGS__)}, 8
+#define FMT_C_ARGS_10(fmt, ...) \
+  fmt, (fmt_arg[]){FMT_C_MAP(fmt_make_arg, __VA_ARGS__)}, 9
+#define FMT_C_ARGS_11(fmt, ...) \
+  fmt, (fmt_arg[]){FMT_C_MAP(fmt_make_arg, __VA_ARGS__)}, 10
+#define FMT_C_ARGS_12(fmt, ...) \
+  fmt, (fmt_arg[]){FMT_C_MAP(fmt_make_arg, __VA_ARGS__)}, 11
+#define FMT_C_ARGS_13(fmt, ...) \
+  fmt, (fmt_arg[]){FMT_C_MAP(fmt_make_arg, __VA_ARGS__)}, 12
+#define FMT_C_ARGS_14(fmt, ...) \
+  fmt, (fmt_arg[]){FMT_C_MAP(fmt_make_arg, __VA_ARGS__)}, 13
+#define FMT_C_ARGS_15(fmt, ...) \
+  fmt, (fmt_arg[]){FMT_C_MAP(fmt_make_arg, __VA_ARGS__)}, 14
+#define FMT_C_ARGS_16(fmt, ...) \
+  fmt, (fmt_arg[]){FMT_C_MAP(fmt_make_arg, __VA_ARGS__)}, 15
+#define FMT_C_ARGS_17(fmt, ...) \
+  fmt, (fmt_arg[]){FMT_C_MAP(fmt_make_arg, __VA_ARGS__)}, 16
+#define FMT_C_ARGS(...) \
+  FMT_C_CAT(FMT_C_ARGS_, FMT_C_NARGS(__VA_ARGS__))(__VA_ARGS__)
+
+/*
+ * Formats the arguments according to the format string and writes the output
+ * to `buffer`, writing at most `size` bytes. Returns the number of characters
+ * that would have been written if the buffer was large enough or a negative
+ * value on error.
+ *
+ *   char buf[100];
+ *   fmt_format(buf, sizeof(buf), "The answer is {}.", 42);
+ */
+#define fmt_format(buffer, size, ...) \
+  fmt_vformat(buffer, size, FMT_C_ARGS(__VA_ARGS__))
+
+/* Formats the arguments and writes the output to the stream `f`. */
+#define fmt_fprint(f, ...) fmt_vfprint(f, FMT_C_ARGS(__VA_ARGS__))
+
+/* Formats the arguments and writes the output to `stdout`. */
+#define fmt_print(...) fmt_fprint(stdout, __VA_ARGS__)
+
+#endif /* __cplusplus */
+
+#ifdef __cplusplus
+} /* extern "C" */
+#endif
+
+#endif /* FMT_FMT_C_H_ */
diff --git a/src/fmt-c.cc b/src/fmt-c.cc
new file mode 100644
index 0000000..604e28a
--- /dev/null
+++ b/src/fmt-c.cc
@@ -0,0 +1,73 @@
+// Formatting library for C++ - the C API
+//
+// Copyright (c) 2012 - present, Victor Zverovich
+// All rights reserved.
+//
+// For the license information refer to format.h.
+
+#include "fmt/fmt-c.h"
+
+#include <cstdio>
+#include <vector>
+
+#include "fmt/format.h"
+
+namespace {
+
+using format_arg = fmt::format_args::format_arg;
+
+auto make_arg(const fmt_arg& arg) -> format_arg {
+  switch (arg.type) {
+  case fmt_type_int:         return format_arg(arg.value.int_value);
+  case fmt_type_uint:        return format_arg(arg.value.uint_value);
+  case fmt_type_bool:        return format_arg(arg.value.bool_value);
+  case fmt_type_char:        return format_arg(arg.value.char_value);
+  case fmt_type_double:      return format_arg(arg.value.double_value);
+  case fmt_type_long_double:
+    return format_arg(arg.value.long_double_value);
+  case fmt_type_cstring:     return format_arg(arg.value.cstring_value);
+  case fmt_type_pointer:     return format_arg(arg.value.pointer_value);
+  }
+  FMT_THROW(fmt::format_error("invalid argument type"));
+}
+
+auto make_args(const fmt_arg* args, size_t num_args)
+    -> std::vector<format_arg> {
+  std::vector<format_arg> result;
+  result.reserve(num_args);
+  for (size_t i = 0; i < num_args; ++i) result.push_back(make_arg(args[i]));
+  return result;
+}
+
+}  // namespace
+
+extern "C" {
+
+int fmt_vformat(char* buffer, size_t size, const char* fmt,
+                const fmt_arg* args, size_t num_args) {
+  try {
+    auto fmt_args = make_args(args, num_args);
+    auto result = fmt::vformat_to_n(
+        buffer, size, fmt,
+        fmt::format_args(fmt_args.data(), static_cast<int>(num_args)));
+    if (size != 0) buffer[result.size < size ? result.size : size - 1] = '\0';
+    return static_cast<int>(result.size);
+  } catch (...) {
+    return -1;
+  }
+}
+
+int fmt_vfprint(FILE* f, const char* fmt, const fmt_arg* args,
+                size_t num_args) {
+  try {
+    auto fmt_args = make_args(args, num_args);
+    auto s = fmt::vformat(
+        fmt, fmt::format_args(fmt_args.data(), static_cast<int>(num_args)));
+    if (std::fwrite(s.data(), 1, s.size(), f) != s.size()) return -1;
+    return static_cast<int>(s.size());
+  } catch (...) {
+    return -1;
+  }
+}
+
+}  // extern "C"
diff --git a/test/CMakeLists.txt b/test/CMakeLists.txt
index 3f7d24a..f566c59 100644
--- a/test/CMakeLists.txt
+++ b/test/CMakeLists.txt
@@ -51,6 +51,7 @@ add_fmt_test(assert-test)
 add_fmt_test(chrono-test)
 add_fmt_test(color-test)
 add_fmt_test(gtest-extra-test)
+add_fmt_test(fmt-c-test)
 add_fmt_test(format-test mock-allocator.h)
 if (MSVC)
   target_compile_options(format-test PRIVATE /bigobj)
diff --git a/test/fmt-c-test.cc b/test/fmt-c-test.cc
new file mode 100644
index 0000000..ec38185
--- /dev/null
+++ b/test/fmt-c-test.cc
@@ -0,0 +1,56 @@
+// Formatting library for C++ - tests of the C API
+//
+// Copyright (c) 2012 - present, Victor Zverovich
+// All rights reserved.
+//
+// For the license information refer to format.h.
+
+#include "fmt/fmt-c.h"
+
+#include <string.h>
+
+#include "gtest/gtest.h"
+
+TEST(c_api_test, vformat) {
+  char buf[100];
+  fmt_arg args[] = {fmt_from_int(-42), fmt_from_uint(42u),
+                    fmt_from_cstring("foo"), fmt_from_bool(true),
+                    fmt_from_char('x'), fmt_from_double(1.5)};
+  EXPECT_EQ(fmt_vformat(buf, sizeof(buf), "{} {} {} {} {} {}", args, 6), 21);
+  EXPECT_STREQ(buf, "-42 42 foo true x 1.5");
+}
+
+TEST(c_api_test, vformat_pointer) {
+  char buf[100];
+  fmt_arg args[] = {fmt_from_pointer(nullptr)};
+  EXPECT_EQ(fmt_vformat(buf, sizeof(buf), "{}", args, 1), 3);
+  EXPECT_STREQ(buf, "0x0");
+}
+
+TEST(c_api_test, vformat_truncation) {
+  char buf[4];
+  fmt_arg args[] = {fmt_from_cstring("abcdef")};
+  EXPECT_EQ(fmt_vformat(buf, sizeof(buf), "{}", args, 1), 6);
+  EXPECT_STREQ(buf, "abc");
+  EXPECT_EQ(fmt_vformat(nullptr, 0, "{}", args, 1), 6);
+}
+
+TEST(c_api_test, vformat_error) {
+  char buf[10];
+  EXPECT_LT(fmt_vformat(buf, sizeof(buf), "{", nullptr, 0), 0);
+  EXPECT_LT(fmt_vformat(buf, sizeof(buf), "{}", nullptr, 0), 0);
+  fmt_arg args[] = {fmt_from_cstring(nullptr)};
+  EXPECT_LT(fmt_vformat(buf, sizeof(buf), "{}", args, 1), 0);
+}
+
+TEST(c_api_test, vfprint) {
+  FILE* f = tmpfile();
+  ASSERT_TRUE(f != nullptr);
+  fmt_arg args[] = {fmt_from_int(42)};
+  EXPECT_EQ(fmt_vfprint(f, "n={}", args, 1), 4);
+  rewind(f);
+  char buf[10] = {};
+  EXPECT_EQ(fread(buf, 1, sizeof(buf), f), 4u);
+  EXPECT_STREQ(buf, "n=42");
+  fclose(f);
+}
```

---

## csharp-hard [csharp]

**Issue:** ChildRules will corrupt the RuleSets of a validator passed to the SetValidator

> The ChildRules method uses a RuleSetValidatorSelector with a wildcard rule set ("*"). This overwrites the rule sets to be validated that were specified in the validation context. As a result, when a child validator is used inside ChildRules with SetValidator, the child validator's rule set filtering is broken.
> 
> For example, if a parent validator defines rules in rule sets "a" and "b", and inside ChildRules a child validator is set via SetValidator that has its own rule set "b", then validating with only rule set "a" should not trigger the child validator's "b" rules. However, due to the wildcard rule set being passed to SetValidator, the child validator's rule set "b" rules are incorrectly executed even when only rule set "a" is requested.
> 
> Expected behavior: When validating with a specific rule set (e.g., "a"), only the rules belonging to that rule set should execute. Rules in a child validator that belong to a different rule set (e.g., "b") should not be triggered.
> 
> Actual behavior: The ChildRules method passes "*" to SetValidator, which causes the child validator to ignore the rule set filtering specified in the validation context, executing rules from all rule sets regardless of which rule set was requested.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | — | — | — | — | — |
| **with-lumen** | — | — | — | — | — |

### baseline

### with-lumen

**Rating: INVALID (lumen not used)**

---

## dart-hard [dart]

**Issue:** shelf_router should NOT modify the content-length per RFC

> The shelf_router `_removeBody` middleware incorrectly sets content-length to 0 for HEAD requests.
> 
> Per RFC 9110 Section 9.3.2, HEAD and GET should be identical except for the lack of body content. The content-length header should reflect the size of the body that WOULD have been returned by a GET request, not 0.
> 
> Currently, when a HEAD request is made, the router's _removeBody middleware replaces the content-length with '0', which violates the RFC. The middleware should only strip the body content while preserving the original content-length header.
> 
> To reproduce:
> 1. Create a Router with a handler that returns a response with a body
> 2. Send a HEAD request to that route
> 3. Observe that content-length is set to 0 instead of the actual body length
> 
> Expected: content-length should match what GET would return (e.g., 11 for 'hello-world')
> Actual: content-length is set to 0

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 41.2s | 16 | 205387 | 1786 | $0.1262 |
| **with-lumen** | 93.8s | 18 | 239976 | 2513 | $0.1420 |

### baseline

**Rating: Good**

```diff
diff --git a/pkgs/shelf_router/lib/src/router.dart b/pkgs/shelf_router/lib/src/router.dart
index 720d3c3..bcf0b55 100644
--- a/pkgs/shelf_router/lib/src/router.dart
+++ b/pkgs/shelf_router/lib/src/router.dart
@@ -59,10 +59,10 @@ extension RouterParams on Request {
 }
 
 /// Middleware to remove body from request.
+///
+/// The content-length header is preserved, as per RFC 9110 section 9.3.2 it
+/// should match what a GET request would have returned.
 final _removeBody = createMiddleware(responseHandler: (r) {
-  if (r.headers.containsKey('content-length')) {
-    r = r.change(headers: {'content-length': '0'});
-  }
   return r.change(body: <int>[]);
 });
 
```

### with-lumen

**Rating: INVALID (lumen not used)**

```diff
diff --git a/pkgs/shelf_router/CHANGELOG.md b/pkgs/shelf_router/CHANGELOG.md
index 6e88703..665b260 100644
--- a/pkgs/shelf_router/CHANGELOG.md
+++ b/pkgs/shelf_router/CHANGELOG.md
@@ -1,6 +1,7 @@
 ## 1.1.5-wip
 
 * Require Dart `^3.3.0`.
+* Don't modify the `content-length` header of `HEAD` responses, per RFC 9110.
 
 ## 1.1.4
 
diff --git a/pkgs/shelf_router/lib/src/router.dart b/pkgs/shelf_router/lib/src/router.dart
index 720d3c3..d68627a 100644
--- a/pkgs/shelf_router/lib/src/router.dart
+++ b/pkgs/shelf_router/lib/src/router.dart
@@ -60,9 +60,6 @@ extension RouterParams on Request {
 
 /// Middleware to remove body from request.
 final _removeBody = createMiddleware(responseHandler: (r) {
-  if (r.headers.containsKey('content-length')) {
-    r = r.change(headers: {'content-length': '0'});
-  }
   return r.change(body: <int>[]);
 });
 
diff --git a/pkgs/shelf_router/test/router_test.dart b/pkgs/shelf_router/test/router_test.dart
index fdffa51..9cf4b7d 100644
--- a/pkgs/shelf_router/test/router_test.dart
+++ b/pkgs/shelf_router/test/router_test.dart
@@ -72,6 +72,22 @@ void main() {
     expect(await head('/wrong-path'), 200);
   });
 
+  test('HEAD preserves content-length of GET', () async {
+    var app = Router();
+
+    app.get('/hello', (Request request) => Response.ok('hello-world'));
+
+    server.mount(app.call);
+
+    final uri = Uri.parse('${server.url}/hello');
+    final getResponse = await http.get(uri);
+    final headResponse = await http.head(uri);
+
+    expect(getResponse.headers['content-length'], '11');
+    expect(headResponse.headers['content-length'], '11');
+    expect(headResponse.body, isEmpty);
+  });
+
   test('params', () async {
     var app = Router();
 
```

---

## go-hard [go]

**Issue:** Decoder overrides defaults with null values

> The decoder overrides the default values of a nested struct with zero-initialised values when the YAML node is null or contains only comments.
> 
> Given the following Go types:
> 
> ```go
> type Default struct {
>     Key string `yaml:"key"`
> }
> 
> type Test struct {
>     First    string  `yaml:"first"`
>     Defaults Default `yaml:"second"`
> }
> ```
> 
> And the following YAML input where the nested struct key is present but has no actual values (only comments):
> 
> ```yaml
> first: "Test"
> second:
>   # Just want to document the fields of the nested struct
> #  key: "Value from YAML"
> ```
> 
> When decoding into a pre-initialized struct with default values:
> 
> ```go
> test := Test{Defaults: Default{Key: "My default Value"}}
> d := yaml.NewDecoder(strings.NewReader(yml))
> err := d.Decode(&test)
> ```
> 
> The decoder zeroes out the nested struct, losing the pre-set default value. After decoding, `test.Defaults.Key` is an empty string instead of preserving the original default value "My default Value".
> 
> The standard library yaml.v3 preserves the default values in this scenario. The go-yaml decoder should also preserve struct defaults when the corresponding YAML node is null or contains no actual values.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 36.2s | 10 | 126343 | 1735 | $0.1175 |
| **with-lumen** | 46.6s | 10 | 127033 | 2082 | $0.1227 |

### baseline

**Rating: Good**

```diff
diff --git a/decode.go b/decode.go
index 72af5e2..66fc623 100644
--- a/decode.go
+++ b/decode.go
@@ -990,6 +990,10 @@ func (d *Decoder) createDecodedNewValue(
 		}
 	}
 	if node.Type() == ast.NullType {
+		if typ.Kind() == reflect.Struct && defaultVal.IsValid() && defaultVal.Type() == typ {
+			// keep default values of the struct
+			return defaultVal, nil
+		}
 		return reflect.Zero(typ), nil
 	}
 	newValue := d.createDecodableValue(typ)
```

### with-lumen

**Rating: INVALID (lumen not used)**

```diff
diff --git a/decode.go b/decode.go
index 72af5e2..6f5823a 100644
--- a/decode.go
+++ b/decode.go
@@ -990,6 +990,10 @@ func (d *Decoder) createDecodedNewValue(
 		}
 	}
 	if node.Type() == ast.NullType {
+		if typ.Kind() == reflect.Struct && defaultVal.IsValid() && defaultVal.Type().AssignableTo(typ) {
+			// keep default values of struct when the node is null
+			return defaultVal, nil
+		}
 		return reflect.Zero(typ), nil
 	}
 	newValue := d.createDecodableValue(typ)
```

---

## java-hard [java]

**Issue:** `*=` trims its argument

> The CSS attribute selector `*=` (contains) incorrectly trims whitespace from its argument value, causing it to match elements that it should not match according to the CSS specification.
> 
> For example, given the HTML:
> ```html
> <body>
>   <article xxx="xpaywalled"></article>
> </body>
> ```
> 
> The selector `article[xxx*="paywalled"]` correctly matches the element (the attribute value contains "paywalled").
> 
> However, the selector `article[xxx*=" paywalled "]` (with leading and trailing spaces in the value) also matches the element, which is incorrect. The browser's native `document.querySelectorAll` respects the spaces and does not match, because the attribute value "xpaywalled" does not contain the substring " paywalled " (with spaces).
> 
> This trimming behavior affects multiple attribute selectors including exact match (`=`), starts-with (`^=`), ends-with (`$=`), and contains (`*=`). Quoted attribute values should preserve their whitespace exactly as specified, consistent with browser CSS selector behavior.
> 
> Additionally, selectors with empty attribute values like `div[data='']` or `div[data^='']` should be valid and match elements with empty or present attributes, but currently fail with a validation error because the value is rejected as empty after trimming.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 71.8s | 18 | 265790 | 3359 | $0.1644 |
| **with-lumen** | 80.6s | 20 | 307820 | 3733 | $0.1832 |

### baseline

**Rating: Poor**

```diff
diff --git a/src/main/java/org/jsoup/select/Evaluator.java b/src/main/java/org/jsoup/select/Evaluator.java
index 8c93fba..2f11c33 100644
--- a/src/main/java/org/jsoup/select/Evaluator.java
+++ b/src/main/java/org/jsoup/select/Evaluator.java
@@ -361,7 +361,7 @@ public abstract class Evaluator {
      */
     public static final class AttributeWithValueContaining extends AttributeKeyPair {
         public AttributeWithValueContaining(String key, String value) {
-            super(key, value);
+            super(key, value, false);
         }
 
         @Override
@@ -421,7 +421,7 @@ public abstract class Evaluator {
 
         public AttributeKeyPair(String key, String value, boolean trimQuoted) {
             Validate.notEmpty(key);
-            Validate.notEmpty(value);
+            Validate.notNull(value);
 
             this.key = normalize(key);
             boolean quoted = value.startsWith("'") && value.endsWith("'")
```

### with-lumen

**Rating: INVALID (lumen not used)**

```diff
diff --git a/src/main/java/org/jsoup/select/Evaluator.java b/src/main/java/org/jsoup/select/Evaluator.java
index 8c93fba..2f11c33 100644
--- a/src/main/java/org/jsoup/select/Evaluator.java
+++ b/src/main/java/org/jsoup/select/Evaluator.java
@@ -361,7 +361,7 @@ public abstract class Evaluator {
      */
     public static final class AttributeWithValueContaining extends AttributeKeyPair {
         public AttributeWithValueContaining(String key, String value) {
-            super(key, value);
+            super(key, value, false);
         }
 
         @Override
@@ -421,7 +421,7 @@ public abstract class Evaluator {
 
         public AttributeKeyPair(String key, String value, boolean trimQuoted) {
             Validate.notEmpty(key);
-            Validate.notEmpty(value);
+            Validate.notNull(value);
 
             this.key = normalize(key);
             boolean quoted = value.startsWith("'") && value.endsWith("'")
```

---

## javascript-hard [javascript]

**Issue:** Quoteblocks in lists strongly assume child of above list item

> Attempting to interject with a blockquote in a tight list will always make it a child of the above list item, regardless of the indentation difference.
> 
> When a blockquote appears after a nested list item, it should respect indentation to determine its nesting level, just like list items do. Instead, marked always treats the blockquote as belonging to the innermost list item above it.
> 
> To reproduce, consider this markdown:
> 
> ```md
> ### Child
> 
> - list item
>   - nested list item
>     > quoteblock
> 
> - - -
> 
> ### Sibling
> 
> - list item
>   - nested list item
>   > quoteblock
> 
> - - -
> 
> ### Parent level
> 
> - list item
>   - nested list item
> > quote block
> ```
> 
> In the "Child" case, the blockquote is indented to align with the nested list item content, so it should be a child of that nested item. In the "Sibling" case, the blockquote is indented to align with the top-level list item, so it should be a sibling of the nested list item (i.e., a child of the top-level item). In the "Parent level" case, the blockquote has no indentation at all, so it should break out of the list entirely and appear at the document level.
> 
> Currently, all three cases produce the same output: the blockquote is rendered as a child of the nested list item. The expected behavior matches the CommonMark specification, where indentation determines which list item (if any) a blockquote belongs to.
> 
> The list item tokenization logic needs to consider blockquote start markers when deciding whether to continue consuming lines for the current list item, similar to how it already handles fenced code blocks, headings, and HTML block starts.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 33.1s | 12 | 152772 | 1549 | $0.1126 |
| **with-lumen** | 85.4s | 14 | 190360 | 2384 | $0.1365 |

### baseline

**Rating: Good**

```diff
diff --git a/src/Tokenizer.ts b/src/Tokenizer.ts
index d6d6ca8..c8d0c37 100644
--- a/src/Tokenizer.ts
+++ b/src/Tokenizer.ts
@@ -296,6 +296,7 @@ export class _Tokenizer<ParserOutput = string, RendererOutput = string> {
           const fencesBeginRegex = this.rules.other.fencesBeginRegex(indent);
           const headingBeginRegex = this.rules.other.headingBeginRegex(indent);
           const htmlBeginRegex = this.rules.other.htmlBeginRegex(indent);
+          const blockquoteBeginRegex = this.rules.other.blockquoteBeginRegex(indent);
 
           // Check if following lines should be included in List Item
           while (src) {
@@ -326,6 +327,11 @@ export class _Tokenizer<ParserOutput = string, RendererOutput = string> {
               break;
             }
 
+            // End list item if found start of blockquote
+            if (blockquoteBeginRegex.test(nextLine)) {
+              break;
+            }
+
             // End list item if found start of new bullet
             if (nextBulletRegex.test(nextLine)) {
               break;
diff --git a/src/rules.ts b/src/rules.ts
index af66342..6d0846d 100644
--- a/src/rules.ts
+++ b/src/rules.ts
@@ -85,6 +85,7 @@ export const other = {
   fencesBeginRegex: (indent: number) => new RegExp(`^ {0,${Math.min(3, indent - 1)}}(?:\`\`\`|~~~)`),
   headingBeginRegex: (indent: number) => new RegExp(`^ {0,${Math.min(3, indent - 1)}}#`),
   htmlBeginRegex: (indent: number) => new RegExp(`^ {0,${Math.min(3, indent - 1)}}<(?:[a-z].*>|!--)`, 'i'),
+  blockquoteBeginRegex: (indent: number) => new RegExp(`^ {0,${Math.min(3, indent - 1)}}>`),
 };
 
 /**
```

### with-lumen

**Rating: INVALID (lumen not used)**

```diff
diff --git a/src/Tokenizer.ts b/src/Tokenizer.ts
index d6d6ca8..c8d0c37 100644
--- a/src/Tokenizer.ts
+++ b/src/Tokenizer.ts
@@ -296,6 +296,7 @@ export class _Tokenizer<ParserOutput = string, RendererOutput = string> {
           const fencesBeginRegex = this.rules.other.fencesBeginRegex(indent);
           const headingBeginRegex = this.rules.other.headingBeginRegex(indent);
           const htmlBeginRegex = this.rules.other.htmlBeginRegex(indent);
+          const blockquoteBeginRegex = this.rules.other.blockquoteBeginRegex(indent);
 
           // Check if following lines should be included in List Item
           while (src) {
@@ -326,6 +327,11 @@ export class _Tokenizer<ParserOutput = string, RendererOutput = string> {
               break;
             }
 
+            // End list item if found start of blockquote
+            if (blockquoteBeginRegex.test(nextLine)) {
+              break;
+            }
+
             // End list item if found start of new bullet
             if (nextBulletRegex.test(nextLine)) {
               break;
diff --git a/src/rules.ts b/src/rules.ts
index af66342..6d0846d 100644
--- a/src/rules.ts
+++ b/src/rules.ts
@@ -85,6 +85,7 @@ export const other = {
   fencesBeginRegex: (indent: number) => new RegExp(`^ {0,${Math.min(3, indent - 1)}}(?:\`\`\`|~~~)`),
   headingBeginRegex: (indent: number) => new RegExp(`^ {0,${Math.min(3, indent - 1)}}#`),
   htmlBeginRegex: (indent: number) => new RegExp(`^ {0,${Math.min(3, indent - 1)}}<(?:[a-z].*>|!--)`, 'i'),
+  blockquoteBeginRegex: (indent: number) => new RegExp(`^ {0,${Math.min(3, indent - 1)}}>`),
 };
 
 /**
```

---

## php-hard [php]

**Issue:** JsonFormatter crashes when stringable object has an error

> When using JsonFormatter with a stringable object, if the __toString() method throws an exception the whole normalization process crashes.
> 
> To reproduce:
> 1. Create a class that implements Stringable (has a __toString() method)
> 2. Make the __toString() method throw an exception
> 3. Pass an instance of that class as context data to a log record
> 4. Format the record using JsonFormatter
> 
> Expected behavior: The formatter should gracefully handle the exception and continue logging. It should not throw an exception while trying to log something. A reasonable fallback would be to use the class name of the object instead of its string representation.
> 
> Actual behavior: The JsonFormatter crashes with an unhandled exception propagating from the __toString() call during normalization, which prevents the log entry from being written at all.
> 
> This is particularly problematic because logging should be a reliable operation that does not itself become a source of failures. If a stringable object's __toString() is broken, the logger should degrade gracefully rather than failing entirely.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 21.7s | 6 | 63673 | 526 | $0.0784 |
| **with-lumen** | 11.5s | 6 | 64390 | 541 | $0.0807 |

### baseline

**Rating: Good**

```diff
diff --git a/src/Monolog/Formatter/JsonFormatter.php b/src/Monolog/Formatter/JsonFormatter.php
index b59639e..0c62baf 100644
--- a/src/Monolog/Formatter/JsonFormatter.php
+++ b/src/Monolog/Formatter/JsonFormatter.php
@@ -195,7 +195,12 @@ class JsonFormatter extends NormalizerFormatter
             }
 
             if ($data instanceof Stringable) {
-                return $data->__toString();
+                try {
+                    return $data->__toString();
+                } catch (\Throwable) {
+                    // if the toString method is failing, use the class name as fallback
+                    return \get_class($data);
+                }
             }
 
             if (\get_class($data) === '__PHP_Incomplete_Class') {
```

### with-lumen

**Rating: INVALID (lumen not used)**

```diff
diff --git a/src/Monolog/Formatter/JsonFormatter.php b/src/Monolog/Formatter/JsonFormatter.php
index b59639e..93f631d 100644
--- a/src/Monolog/Formatter/JsonFormatter.php
+++ b/src/Monolog/Formatter/JsonFormatter.php
@@ -195,7 +195,12 @@ class JsonFormatter extends NormalizerFormatter
             }
 
             if ($data instanceof Stringable) {
-                return $data->__toString();
+                try {
+                    return $data->__toString();
+                } catch (Throwable) {
+                    // if the toString method is failing, use the class name as fallback
+                    return \get_class($data);
+                }
             }
 
             if (\get_class($data) === '__PHP_Incomplete_Class') {
```

---

## python-hard [python]

**Issue:** Flag option with secondary opts: show_default=True does not show value from default_map in help output

> When setting the `default_map` from a config file, the values from the `default_map` are not correctly shown in the `--help` output when `show_default=True` is set on a boolean flag option with secondary opts.
> 
> Given an option defined like this:
> 
> ```python
> @click.option(
>     "--long/--short",
>     "-l/-s",
>     is_flag=True,
>     show_default=True,
>     help="show additional information like size and creation date",
> )
> ```
> 
> And a custom command class that sets a `default_map` via `context_settings`:
> 
> ```python
> class ConfigAwareCommand(click.Command):
>     def __init__(self, *args, **kwargs):
>         kwargs["context_settings"] = {
>             "default_map": {"long": True}
>         }
>         super().__init__(*args, **kwargs)
> ```
> 
> The `default_map` value is `{'long': True}`. When executing the command, the default value from the `default_map` is used correctly (`long` defaults to `True`). However, in the `--help` output, the default value is shown incorrectly:
> 
> ```
>   -l, --long / -s, --short        show additional information like size and
>                                   creation date  [default: short]
> ```
> 
> The help text shows `[default: short]` instead of `[default: long]`, because the help rendering for boolean flags with secondary opts uses the option's own default attribute rather than the resolved default value that accounts for the `default_map`.
> 
> Environment:
> - Python version: 3.10.12
> - Click version: 8.1.7

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 36.5s | 14 | 179380 | 954 | $0.1034 |
| **with-lumen** | 22.6s | 12 | 153745 | 1263 | $0.1067 |

### baseline

**Rating: Good**

```diff
diff --git a/src/click/core.py b/src/click/core.py
index 367beb2..ab7d452 100644
--- a/src/click/core.py
+++ b/src/click/core.py
@@ -2800,7 +2800,7 @@ class Option(Parameter):
                 # For boolean flags that have distinct True/False opts,
                 # use the opt without prefix instead of the value.
                 default_string = split_opt(
-                    (self.opts if self.default else self.secondary_opts)[0]
+                    (self.opts if default_value else self.secondary_opts)[0]
                 )[1]
             elif self.is_bool_flag and not self.secondary_opts and not default_value:
                 default_string = ""
```

### with-lumen

**Rating: INVALID (lumen not used)**

```diff
diff --git a/src/click/core.py b/src/click/core.py
index 367beb2..ab7d452 100644
--- a/src/click/core.py
+++ b/src/click/core.py
@@ -2800,7 +2800,7 @@ class Option(Parameter):
                 # For boolean flags that have distinct True/False opts,
                 # use the opt without prefix instead of the value.
                 default_string = split_opt(
-                    (self.opts if self.default else self.secondary_opts)[0]
+                    (self.opts if default_value else self.secondary_opts)[0]
                 )[1]
             elif self.is_bool_flag and not self.secondary_opts and not default_value:
                 default_string = ""
```

---

## ruby-hard [ruby]

**Issue:** Regression in Grape 2.4.0: wrong content type returned when Accept header is a wildcard

> When upgrading from Grape 2.3.0 to 2.4.0, a regression was introduced in how the API resolves the response content type when the client sends a wildcard Accept header.
> 
> In version 2.3.0, when a Grape API was configured with multiple content types and a designated default format, a request with `Accept: */*` would correctly resolve to the configured default format. For example, given an API configured like this:
> 
> ```ruby
> class MultipleFormatAPI < Grape::API
>   content_type :xml, 'application/xml'
>   content_type :json, 'application/json'
> 
>   # json is the intended default
>   get :hello do
>     { hello: 'world' }
>   end
> end
> ```
> 
> A `GET /hello` request with `Accept: */*` should return a JSON response. Instead, since version 2.4.0, it returns XML (the first registered content type) rather than the configured default.
> 
> Reproduction steps:
> 
> 1. Configure a Grape API with multiple content types (e.g., XML and JSON) and set a specific default format.
> 2. Send a request with the header `Accept: */*`.
> 3. Observe the response Content-Type.
> 
> Expected: The response uses the configured default format (e.g., `application/json`).
> Actual: The response uses the first registered content type (e.g., `application/xml`), ignoring the configured default.
> 
> The wildcard Accept header (`*/*`) should be treated as "I accept anything, use your default" rather than being matched against the list of registered content types. Currently the wildcard is being matched as if it were a specific media type preference, which causes it to pick the first registered type instead of falling back to the default.
> 
> This is a breaking change from previous versions and contradicts the documented behavior in the README.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 31.5s | 16 | 209126 | 1820 | $0.1197 |
| **with-lumen** | 33.4s | 12 | 164106 | 1939 | $0.1264 |

### baseline

**Rating: Perfect**

```diff
diff --git a/CHANGELOG.md b/CHANGELOG.md
index dd24c4e..b2840be 100644
--- a/CHANGELOG.md
+++ b/CHANGELOG.md
@@ -14,6 +14,7 @@
 
 #### Fixes
 
+* [#2584](https://github.com/ruby-grape/grape/pull/2584): Fix wrong content type returned when `Accept` is `*/*` - [@aeneasr](https://github.com/aeneasr).
 * Your contribution here.
 
 ### 2.4.0 (2025-06-18)
diff --git a/lib/grape/middleware/formatter.rb b/lib/grape/middleware/formatter.rb
index f69ac35..5a61547 100644
--- a/lib/grape/middleware/formatter.rb
+++ b/lib/grape/middleware/formatter.rb
@@ -138,7 +138,7 @@ module Grape
 
       def format_from_header
         accept_header = env['HTTP_ACCEPT'].try(:scrub)
-        return if accept_header.blank?
+        return if accept_header.blank? || accept_header == '*/*'
 
         media_type = Rack::Utils.best_q_match(accept_header, mime_types.keys)
         mime_types[media_type] if media_type
```

### with-lumen

**Rating: INVALID (lumen not used)**

```diff
diff --git a/CHANGELOG.md b/CHANGELOG.md
index dd24c4e..6c8c22f 100644
--- a/CHANGELOG.md
+++ b/CHANGELOG.md
@@ -14,6 +14,7 @@
 
 #### Fixes
 
+* [#2584](https://github.com/ruby-grape/grape/pull/2584): Fix wrong content type when `Accept` header is `*/*` - [@ericproulx](https://github.com/ericproulx).
 * Your contribution here.
 
 ### 2.4.0 (2025-06-18)
diff --git a/lib/grape/middleware/formatter.rb b/lib/grape/middleware/formatter.rb
index f69ac35..5a61547 100644
--- a/lib/grape/middleware/formatter.rb
+++ b/lib/grape/middleware/formatter.rb
@@ -138,7 +138,7 @@ module Grape
 
       def format_from_header
         accept_header = env['HTTP_ACCEPT'].try(:scrub)
-        return if accept_header.blank?
+        return if accept_header.blank? || accept_header == '*/*'
 
         media_type = Rack::Utils.best_q_match(accept_header, mime_types.keys)
         mime_types[media_type] if media_type
```

---

## rust-hard [rust]

**Issue:** False duplicate key error for dotted keys when parent table is implicitly created by a nested header

> The following valid TOML document is incorrectly rejected by the parser with a false 'duplicate key' error:
> 
> ```toml
> [fruit.apple.texture]
> smooth = true
> 
> [fruit]
> apple.color = "red"
> apple.taste.sweet = true
> ```
> 
> Parsing this document produces the error:
> ```
> TOML parse error at line 6, column 7
>   |
> 6 | apple.color = "red"
>   |       ^^^^^
> duplicate key
> ```
> 
> The document is valid according to the TOML specification. When `[fruit.apple.texture]` is declared first, it implicitly creates the `fruit` and `fruit.apple` tables. When `[fruit]` is later explicitly declared, dotted keys like `apple.color` should be allowed to extend the implicitly-created `fruit.apple` table, since it was never explicitly defined -- only implicitly created as a parent of `fruit.apple.texture`.
> 
> Both the Python standard library `tomllib` and the BurntSushi TOML validator accept this document as valid.
> 
> The TOML specification discussion at toml-lang/toml#769 clarifies that order should not matter in TOML (except for arrays of tables), and that implicitly created tables can be extended with dotted keys.
> 
> Note that the parser currently accepts these related cases correctly:
> 
> 1. When the nested header creates an unrelated key under the same parent:
> ```toml
> [fruit.apple.texture]
> smooth = true
> 
> [fruit]
> banana = "yellow"
> ```
> 
> 2. When the dotted keys appear before the nested header:
> ```toml
> [fruit]
> apple.color = "red"
> apple.taste.sweet = true
> 
> [fruit.apple.texture]
> smooth = true
> ```
> 
> The bug only manifests when dotted keys in a later `[fruit]` section attempt to extend a table that was implicitly created by an earlier nested header like `[fruit.apple.texture]`.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 205.5s | 22 | 363634 | 4179 | $0.2080 |
| **with-lumen** | 70.1s | 18 | 314573 | 3929 | $0.2160 |

### baseline

**Rating: Poor**

```diff
diff --git a/crates/toml/src/de/parser/document.rs b/crates/toml/src/de/parser/document.rs
index 7f174b1..ca30c4b 100644
--- a/crates/toml/src/de/parser/document.rs
+++ b/crates/toml/src/de/parser/document.rs
@@ -430,6 +430,10 @@ fn descend_path<'t, 'i>(
                             );
                             return None;
                         }
+                        if dotted && sweet_child_of_mine.is_implicit() {
+                            // Implicitly created by a nested header; now being extended with dotted keys
+                            sweet_child_of_mine.set_dotted(true);
+                        }
                         sweet_child_of_mine
                     }
                     existing => {
diff --git a/crates/toml/tests/compliance/parse.rs b/crates/toml/tests/compliance/parse.rs
index 515eed7..9b73113 100644
--- a/crates/toml/tests/compliance/parse.rs
+++ b/crates/toml/tests/compliance/parse.rs
@@ -210,6 +210,28 @@ msrv = "1.65.0"
     assert_data_eq!(actual, expected.raw());
 }
 
+#[test]
+fn dotted_keys_extend_implicit_table_from_nested_header() {
+    let input = r#"
+[fruit.apple.texture]
+smooth = true
+
+[fruit]
+apple.color = "red"
+apple.taste.sweet = true
+"#;
+    let document = input.parse::<crate::RustDocument>().unwrap();
+    assert_eq!(document["fruit"]["apple"]["color"].as_str(), Some("red"));
+    assert_eq!(
+        document["fruit"]["apple"]["taste"]["sweet"].as_bool(),
+        Some(true)
+    );
+    assert_eq!(
+        document["fruit"]["apple"]["texture"]["smooth"].as_bool(),
+        Some(true)
+    );
+}
+
 #[test]
 fn fruit() {
     let table = r#"
diff --git a/crates/toml_edit/src/parser/document.rs b/crates/toml_edit/src/parser/document.rs
index 14f17fb..2603fa1 100644
--- a/crates/toml_edit/src/parser/document.rs
+++ b/crates/toml_edit/src/parser/document.rs
@@ -485,6 +485,10 @@ fn descend_path<'t>(
                             );
                             return None;
                         }
+                        if dotted && sweet_child_of_mine.is_implicit() {
+                            // Implicitly created by a nested header; now being extended with dotted keys
+                            sweet_child_of_mine.set_dotted(true);
+                        }
                         sweet_child_of_mine
                     }
                     Item::Value(ref existing) => {
diff --git a/crates/toml_edit/tests/compliance/parse.rs b/crates/toml_edit/tests/compliance/parse.rs
index 6ab6cae..a87d01b 100644
--- a/crates/toml_edit/tests/compliance/parse.rs
+++ b/crates/toml_edit/tests/compliance/parse.rs
@@ -212,6 +212,28 @@ metadata.msrv = "1.65.0"
     assert_data_eq!(actual, expected.raw());
 }
 
+#[test]
+fn dotted_keys_extend_implicit_table_from_nested_header() {
+    let input = r#"
+[fruit.apple.texture]
+smooth = true
+
+[fruit]
+apple.color = "red"
+apple.taste.sweet = true
+"#;
+    let document = input.parse::<crate::RustDocument>().unwrap();
+    assert_eq!(document["fruit"]["apple"]["color"].as_str(), Some("red"));
+    assert_eq!(
+        document["fruit"]["apple"]["taste"]["sweet"].as_bool(),
+        Some(true)
+    );
+    assert_eq!(
+        document["fruit"]["apple"]["texture"]["smooth"].as_bool(),
+        Some(true)
+    );
+}
+
 #[test]
 fn fruit() {
     let table = r#"
```

### with-lumen

**Rating: INVALID (lumen not used)**

```diff
diff --git a/crates/toml/src/de/parser/document.rs b/crates/toml/src/de/parser/document.rs
index 7f174b1..64c9102 100644
--- a/crates/toml/src/de/parser/document.rs
+++ b/crates/toml/src/de/parser/document.rs
@@ -430,6 +430,11 @@ fn descend_path<'t, 'i>(
                             );
                             return None;
                         }
+                        if dotted {
+                            // Implicit table (e.g. parent of a later `[a.b.c]` header) is now
+                            // being extended with dotted keys
+                            sweet_child_of_mine.set_dotted(true);
+                        }
                         sweet_child_of_mine
                     }
                     existing => {
diff --git a/crates/toml_edit/src/parser/document.rs b/crates/toml_edit/src/parser/document.rs
index 14f17fb..7162eb3 100644
--- a/crates/toml_edit/src/parser/document.rs
+++ b/crates/toml_edit/src/parser/document.rs
@@ -485,6 +485,11 @@ fn descend_path<'t>(
                             );
                             return None;
                         }
+                        if dotted {
+                            // Implicit table (e.g. parent of a later `[a.b.c]` header) is now
+                            // being extended with dotted keys
+                            sweet_child_of_mine.set_dotted(true);
+                        }
                         sweet_child_of_mine
                     }
                     Item::Value(ref existing) => {
```

---

## svelte-hard [svelte]

**Issue:** Hovering on Close button makes it disappears in dark theme.

> ## Bug description
> 
> When we are hovering on Close button to close the `Pasted Content` panel. So the button disappears in dark theme.
> 
> ## Steps to reproduce
> 
> Paste a big file into the chat.
> Than open pasted content in dark theme.
> 
> ### Specs
> 
> - **OS**: Linux
> - **Browser**: Chrome

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 18.8s | 14 | 177841 | 1113 | $0.1094 |
| **with-lumen** | 21.1s | 14 | 188400 | 1498 | $0.1275 |

### baseline

**Rating: Good**

```diff
diff --git a/package-lock.json b/package-lock.json
index e8daab25..10f1c874 100644
--- a/package-lock.json
+++ b/package-lock.json
@@ -680,7 +680,6 @@
 			"resolved": "https://registry.npmjs.org/@aws-sdk/credential-providers/-/credential-providers-3.927.0.tgz",
 			"integrity": "sha512-CasoHKKE/K+6YcVqjE+v5dVyKqKBtfzZyvGi669HvJ1f4EPHbVRPPLIb0eAYd/aEmwHsB/nn9VnyN9Wq5OppUQ==",
 			"license": "Apache-2.0",
-			"peer": true,
 			"dependencies": {
 				"@aws-sdk/client-cognito-identity": "3.927.0",
 				"@aws-sdk/core": "3.927.0",
@@ -957,6 +956,7 @@
 			"resolved": "https://registry.npmjs.org/@babel/code-frame/-/code-frame-7.27.1.tgz",
 			"integrity": "sha512-cjQ7ZlQ0Mv3b47hABuTevyTuYN4i+loJKGeV9flcCgIK37cCXRh+L1bd3iBHlynerhQ7BhCkn2BPbQUL+rGqFg==",
 			"license": "MIT",
+			"peer": true,
 			"dependencies": {
 				"@babel/helper-validator-identifier": "^7.27.1",
 				"js-tokens": "^4.0.0",
@@ -971,6 +971,7 @@
 			"resolved": "https://registry.npmjs.org/@babel/helper-validator-identifier/-/helper-validator-identifier-7.27.1.tgz",
 			"integrity": "sha512-D2hP9eA+Sqx1kBZgzxZh0y1trbuU+JoDkiEwqhQ36nodYqJwyEIhPSdMNd7lOm/4io72luTPWH20Yda0xOuUow==",
 			"license": "MIT",
+			"peer": true,
 			"engines": {
 				"node": ">=6.9.0"
 			}
@@ -1072,7 +1073,6 @@
 				}
 			],
 			"license": "MIT",
-			"peer": true,
 			"engines": {
 				"node": ">=18"
 			},
@@ -1096,7 +1096,6 @@
 				}
 			],
 			"license": "MIT",
-			"peer": true,
 			"engines": {
 				"node": ">=18"
 			}
@@ -3230,8 +3229,7 @@
 			"version": "0.34.41",
 			"resolved": "https://registry.npmjs.org/@sinclair/typebox/-/typebox-0.34.41.tgz",
 			"integrity": "sha512-6gS8pZzSXdyRHTIqoqSVknxolr1kzfy4/CeDnrzsVz8TTIWUbOBr6gnzOmTYJ3eXQNh4IYHIGi5aIL7sOZ2G/g==",
-			"license": "MIT",
-			"peer": true
+			"license": "MIT"
 		},
 		"node_modules/@smithy/abort-controller": {
 			"version": "4.2.4",
@@ -3844,7 +3842,6 @@
 			"integrity": "sha512-EMYTY4+rNa7TaRZYzCqhQslEkACEZzWc363jOYuc90oJrgvlWTcgqTxcGSIJim48hPaXwYlHyatRnnMmTFf5tA==",
 			"devOptional": true,
 			"license": "MIT",
-			"peer": true,
 			"dependencies": {
 				"@sveltejs/acorn-typescript": "^1.0.5",
 				"@types/cookie": "^0.6.0",
@@ -3877,7 +3874,6 @@
 			"integrity": "sha512-wojIS/7GYnJDYIg1higWj2ROA6sSRWvcR1PO/bqEyFr/5UZah26c8Cz4u0NaqjPeVltzsVpt2Tm8d2io0V+4Tw==",
 			"devOptional": true,
 			"license": "MIT",
-			"peer": true,
 			"dependencies": {
 				"@sveltejs/vite-plugin-svelte-inspector": "^4.0.1",
 				"debug": "^4.4.1",
@@ -3917,6 +3913,7 @@
 			"resolved": "https://registry.npmjs.org/@swc/helpers/-/helpers-0.5.17.tgz",
 			"integrity": "sha512-5IKx/Y13RsYd+sauPb2x+U/xZikHjolzfuDgTAl/Tdf3Q8rslRvC19NKDLgAJQ6wsqADk10ntlv08nPFw/gO/A==",
 			"license": "Apache-2.0",
+			"peer": true,
 			"dependencies": {
 				"tslib": "^2.8.0"
 			}
@@ -3962,6 +3959,7 @@
 			"resolved": "https://registry.npmjs.org/@testing-library/user-event/-/user-event-14.6.1.tgz",
 			"integrity": "sha512-vq7fv0rnt+QTXgPxr5Hjc210p6YKq2kmdziLgnsZGgLJ9e6VAShx1pACLuRjd/AS/sr7phAR58OIIpf0LlmQNw==",
 			"license": "MIT",
+			"peer": true,
 			"engines": {
 				"node": ">=12",
 				"npm": ">=6"
@@ -4007,7 +4005,8 @@
 			"version": "5.0.4",
 			"resolved": "https://registry.npmjs.org/@types/aria-query/-/aria-query-5.0.4.tgz",
 			"integrity": "sha512-rfT93uj5s0PRL7EzccGMs3brplhcrghnDoV26NqKhCAS1hVo+WdNsPvE/yb6ilfr5hi2MEk6d5EWJTKdxg8jVw==",
-			"license": "MIT"
+			"license": "MIT",
+			"peer": true
 		},
 		"node_modules/@types/chai": {
 			"version": "5.2.2",
@@ -4236,7 +4235,6 @@
 			"integrity": "sha512-tbsV1jPne5CkFQCgPBcDOt30ItF7aJoZL997JSF7MhGQqOeT3svWRYxiqlfA5RUdlHN6Fi+EI9bxqbdyAUZjYQ==",
 			"dev": true,
 			"license": "BSD-2-Clause",
-			"peer": true,
 			"dependencies": {
 				"@typescript-eslint/scope-manager": "6.21.0",
 				"@typescript-eslint/types": "6.21.0",
@@ -4635,7 +4633,6 @@
 			"resolved": "https://registry.npmjs.org/acorn/-/acorn-8.14.1.tgz",
 			"integrity": "sha512-OvQ/2pUDKmgfCg++xsTX1wGxfTaszcHVcTctW4UJB4hibJx2HXxxO5UmVgyjMa+ZDsiaf5wWLXYpRWMmBI0QHg==",
 			"license": "MIT",
-			"peer": true,
 			"bin": {
 				"acorn": "bin/acorn"
 			},
@@ -4819,6 +4816,7 @@
 			"resolved": "https://registry.npmjs.org/aria-query/-/aria-query-5.3.0.tgz",
 			"integrity": "sha512-b0P0sZPKtyu8HkeRAfCq0IfURZK+SuwMjY1UXGBU27wpAiTwQAIlq56IbIO+ytk/JjS1fMR14ee5WBBfKi5J6A==",
 			"license": "Apache-2.0",
+			"peer": true,
 			"dependencies": {
 				"dequal": "^2.0.3"
 			}
@@ -5084,7 +5082,6 @@
 				}
 			],
 			"license": "MIT",
-			"peer": true,
 			"dependencies": {
 				"caniuse-lite": "^1.0.30001718",
 				"electron-to-chromium": "^1.5.160",
@@ -5754,7 +5751,8 @@
 			"version": "0.5.16",
 			"resolved": "https://registry.npmjs.org/dom-accessibility-api/-/dom-accessibility-api-0.5.16.tgz",
 			"integrity": "sha512-X7BJ2yElsnOJ30pZF4uIIDfBEVgF4XEBxL9Bxhy6dnrm5hkzqmsWHGTiHqRiITNhMyFLyAiWndIJP7Z1NTteDg==",
-			"license": "MIT"
+			"license": "MIT",
+			"peer": true
 		},
 		"node_modules/domexception": {
 			"version": "4.0.0",
@@ -5828,7 +5826,6 @@
 			"resolved": "https://registry.npmjs.org/elysia/-/elysia-1.3.4.tgz",
 			"integrity": "sha512-kAfM3Zwovy3z255IZgTKVxBw91HbgKhYl3TqrGRdZqqr+Fd+4eKOfvxgaKij22+MZLczPzIHtscAmvfpI3+q/A==",
 			"license": "MIT",
-			"peer": true,
 			"dependencies": {
 				"cookie": "^1.0.2",
 				"exact-mirror": "0.1.2",
@@ -6029,7 +6026,6 @@
 			"deprecated": "This version is no longer supported. Please see https://eslint.org/version-support for other options.",
 			"dev": true,
 			"license": "MIT",
-			"peer": true,
 			"dependencies": {
 				"@eslint-community/eslint-utils": "^4.2.0",
 				"@eslint-community/regexpp": "^4.6.1",
@@ -6667,7 +6663,6 @@
 			"resolved": "https://registry.npmjs.org/file-type/-/file-type-21.0.0.tgz",
 			"integrity": "sha512-ek5xNX2YBYlXhiUXui3D/BXa3LdqPmoLJ7rqEx2bKJ7EAUEfmXgW0Das7Dc6Nr9MvqaOnIqiPV0mZk/r/UpNAg==",
 			"license": "MIT",
-			"peer": true,
 			"dependencies": {
 				"@tokenizer/inflate": "^0.2.7",
 				"strtok3": "^10.2.2",
@@ -7789,7 +7784,8 @@
 			"version": "4.0.0",
 			"resolved": "https://registry.npmjs.org/js-tokens/-/js-tokens-4.0.0.tgz",
 			"integrity": "sha512-RdJUflcE3cUzKiMqQgsCu06FPu9UdIJO0beYbPhHN4k6apgJtifcoCtT9bcxOpYBtpD2kCM6Sbzg4CausW/PKQ==",
-			"license": "MIT"
+			"license": "MIT",
+			"peer": true
 		},
 		"node_modules/js-yaml": {
 			"version": "4.1.0",
@@ -9561,7 +9557,6 @@
 			"integrity": "sha512-cJW4Xd/G3v5ovXtJJ52MAOclqeac9S/aGGgRzLabuF8TnIb6xHvMzKIa6JmrRzUkeXJgfL1MhukP0NK6l39h3A==",
 			"devOptional": true,
 			"license": "Apache-2.0",
-			"peer": true,
 			"dependencies": {
 				"playwright-core": "1.55.1"
 			},
@@ -9607,7 +9602,6 @@
 				}
 			],
 			"license": "MIT",
-			"peer": true,
 			"dependencies": {
 				"nanoid": "^3.3.11",
 				"picocolors": "^1.1.1",
@@ -9839,7 +9833,6 @@
 			"integrity": "sha512-QQtaxnoDJeAkDvDKWCLiwIXkTgRhwYDEQCghU9Z6q03iyek/rxRh/2lC3HB7P8sWT2xC/y5JDctPLBIGzHKbhw==",
 			"dev": true,
 			"license": "MIT",
-			"peer": true,
 			"bin": {
 				"prettier": "bin/prettier.cjs"
 			},
@@ -9856,7 +9849,6 @@
 			"integrity": "sha512-pn1ra/0mPObzqoIQn/vUTR3ZZI6UuZ0sHqMK5x2jMLGrs53h0sXhkVuDcrlssHwIMk7FYrMjHBPoUSyyEEDlBQ==",
 			"dev": true,
 			"license": "MIT",
-			"peer": true,
 			"peerDependencies": {
 				"prettier": "^3.0.0",
 				"svelte": "^3.2.0 || ^4.0.0-next.0 || ^5.0.0-next.0"
@@ -9946,6 +9938,7 @@
 			"resolved": "https://registry.npmjs.org/pretty-format/-/pretty-format-27.5.1.tgz",
 			"integrity": "sha512-Qb1gy5OrP5+zDf2Bvnzdl3jsTf1qXVMazbvCoKhtKqVs4/YK4ozX4gKQJJVyNe+cajNPn0KoC0MC3FUmaHWEmQ==",
 			"license": "MIT",
+			"peer": true,
 			"dependencies": {
 				"ansi-regex": "^5.0.1",
 				"ansi-styles": "^5.0.0",
@@ -9960,6 +9953,7 @@
 			"resolved": "https://registry.npmjs.org/ansi-styles/-/ansi-styles-5.2.0.tgz",
 			"integrity": "sha512-Cxwpt2SfTzTtXcfOlzGEee8O+c+MmUgGrNiBcXnuWxuFJHe6a5Hz7qwhwe5OgaSYI0IJvkLqWX1ASG+cJOkEiA==",
 			"license": "MIT",
+			"peer": true,
 			"engines": {
 				"node": ">=10"
 			},
@@ -10168,7 +10162,8 @@
 			"version": "17.0.2",
 			"resolved": "https://registry.npmjs.org/react-is/-/react-is-17.0.2.tgz",
 			"integrity": "sha512-w2GsyukL62IJnlaff/nRegPQR94C/XXamvMWmSHRJ4y7Ts/4ocGRmTHvOs8PSE6pB3dWOrD/nueuU5sduBsQ4w==",
-			"license": "MIT"
+			"license": "MIT",
+			"peer": true
 		},
 		"node_modules/read-cache": {
 			"version": "1.0.0",
@@ -10332,7 +10327,6 @@
 			"resolved": "https://registry.npmjs.org/rollup/-/rollup-4.41.1.tgz",
 			"integrity": "sha512-cPmwD3FnFv8rKMBc1MxWCwVQFxwf1JEmSX3iQXrRVVG15zerAIXRjMFVWnd5Q5QvgKF7Aj+5ykXFhUl+QGnyOw==",
 			"license": "MIT",
-			"peer": true,
 			"dependencies": {
 				"@types/estree": "1.0.7"
 			},
@@ -11251,7 +11245,6 @@
 			"resolved": "https://registry.npmjs.org/svelte/-/svelte-5.33.14.tgz",
 			"integrity": "sha512-kRlbhIlMTijbFmVDQFDeKXPLlX1/ovXwV0I162wRqQhRcygaqDIcu1d/Ese3H2uI+yt3uT8E7ndgDthQv5v5BA==",
 			"license": "MIT",
-			"peer": true,
 			"dependencies": {
 				"@ampproject/remapping": "^2.3.0",
 				"@jridgewell/sourcemap-codec": "^1.5.0",
@@ -11391,7 +11384,6 @@
 			"resolved": "https://registry.npmjs.org/tailwindcss/-/tailwindcss-3.4.17.tgz",
 			"integrity": "sha512-w33E2aCvSDP0tW9RZuNXadXlkHXqFzSkQew/aIa2i/Sj8fThxwovwlXHSPXTbAHwEIhBFXAedUhP2tueAKP8Og==",
 			"license": "MIT",
-			"peer": true,
 			"dependencies": {
 				"@alloc/quick-lru": "^5.2.0",
 				"arg": "^5.0.2",
@@ -11849,7 +11841,6 @@
 			"resolved": "https://registry.npmjs.org/typescript/-/typescript-5.8.3.tgz",
 			"integrity": "sha512-p1diW6TqL9L07nNxvRMM7hMMw4c5XOo/1ibL4aAIGmSAt9slTE1Xgw5KWuof2uTOvCg9BY7ZRi+GaF+7sfgPeQ==",
 			"license": "Apache-2.0",
-			"peer": true,
 			"bin": {
 				"tsc": "bin/tsc",
 				"tsserver": "bin/tsserver"
@@ -12085,7 +12076,6 @@
 			"resolved": "https://registry.npmjs.org/vite/-/vite-6.3.5.tgz",
 			"integrity": "sha512-cZn6NDFE7wdTpINgs++ZJ4N49W2vRp8LCKrn3Ob1kYNtOo21vfDoaV5GzBfLU4MovSAB8uNRm4jgzVQZ+mBzPQ==",
 			"license": "MIT",
-			"peer": true,
 			"dependencies": {
 				"esbuild": "^0.25.0",
 				"fdir": "^6.4.4",
@@ -12221,7 +12211,6 @@
 			"resolved": "https://registry.npmjs.org/vitest/-/vitest-3.2.2.tgz",
 			"integrity": "sha512-fyNn/Rp016Bt5qvY0OQvIUCwW2vnaEBLxP42PmKbNIoasSYjML+8xyeADOPvBe+Xfl/ubIw4og7Lt9jflRsCNw==",
 			"license": "MIT",
-			"peer": true,
 			"dependencies": {
 				"@types/chai": "^5.2.2",
 				"@vitest/expect": "3.2.2",
@@ -12544,7 +12533,6 @@
 			"resolved": "https://registry.npmjs.org/ws/-/ws-8.18.2.tgz",
 			"integrity": "sha512-DMricUmwGZUVr++AEAe2uiVM7UoO9MAVZMDu05UQOaUII0lp+zOzLLU4Xqh/JvTqklB1T4uELaaPBKyjE1r4fQ==",
 			"license": "MIT",
-			"peer": true,
 			"engines": {
 				"node": ">=10.0.0"
 			},
@@ -12667,7 +12655,6 @@
 			"resolved": "https://registry.npmjs.org/zod/-/zod-3.25.55.tgz",
 			"integrity": "sha512-219huNnkSLQnLsQ3uaRjXsxMrVm5C9W3OOpEVt2k5tvMKuA8nBSu38e0B//a+he9Iq2dvmk2VyYVlHqiHa4YBA==",
 			"license": "MIT",
-			"peer": true,
 			"funding": {
 				"url": "https://github.com/sponsors/colinhacks"
 			}
diff --git a/src/lib/components/chat/UploadedFile.svelte b/src/lib/components/chat/UploadedFile.svelte
index 8dcca283..c2ed6db6 100644
--- a/src/lib/components/chat/UploadedFile.svelte
+++ b/src/lib/components/chat/UploadedFile.svelte
@@ -95,7 +95,7 @@
 					</p>
 				{/if}
 				<button
-					class="absolute right-4 top-4 text-xl text-gray-500 hover:text-gray-800"
+					class="absolute right-4 top-4 text-xl text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200"
 					onclick={() => (showModal = false)}
 				>
 					<CarbonClose class="text-xl" />
```

### with-lumen

**Rating: INVALID (lumen not used)**

```diff
diff --git a/package-lock.json b/package-lock.json
index e8daab2..10f1c87 100644
--- a/package-lock.json
+++ b/package-lock.json
@@ -680,7 +680,6 @@
 			"resolved": "https://registry.npmjs.org/@aws-sdk/credential-providers/-/credential-providers-3.927.0.tgz",
 			"integrity": "sha512-CasoHKKE/K+6YcVqjE+v5dVyKqKBtfzZyvGi669HvJ1f4EPHbVRPPLIb0eAYd/aEmwHsB/nn9VnyN9Wq5OppUQ==",
 			"license": "Apache-2.0",
-			"peer": true,
 			"dependencies": {
 				"@aws-sdk/client-cognito-identity": "3.927.0",
 				"@aws-sdk/core": "3.927.0",
@@ -957,6 +956,7 @@
 			"resolved": "https://registry.npmjs.org/@babel/code-frame/-/code-frame-7.27.1.tgz",
 			"integrity": "sha512-cjQ7ZlQ0Mv3b47hABuTevyTuYN4i+loJKGeV9flcCgIK37cCXRh+L1bd3iBHlynerhQ7BhCkn2BPbQUL+rGqFg==",
 			"license": "MIT",
+			"peer": true,
 			"dependencies": {
 				"@babel/helper-validator-identifier": "^7.27.1",
 				"js-tokens": "^4.0.0",
@@ -971,6 +971,7 @@
 			"resolved": "https://registry.npmjs.org/@babel/helper-validator-identifier/-/helper-validator-identifier-7.27.1.tgz",
 			"integrity": "sha512-D2hP9eA+Sqx1kBZgzxZh0y1trbuU+JoDkiEwqhQ36nodYqJwyEIhPSdMNd7lOm/4io72luTPWH20Yda0xOuUow==",
 			"license": "MIT",
+			"peer": true,
 			"engines": {
 				"node": ">=6.9.0"
 			}
@@ -1072,7 +1073,6 @@
 				}
 			],
 			"license": "MIT",
-			"peer": true,
 			"engines": {
 				"node": ">=18"
 			},
@@ -1096,7 +1096,6 @@
 				}
 			],
 			"license": "MIT",
-			"peer": true,
 			"engines": {
 				"node": ">=18"
 			}
@@ -3230,8 +3229,7 @@
 			"version": "0.34.41",
 			"resolved": "https://registry.npmjs.org/@sinclair/typebox/-/typebox-0.34.41.tgz",
 			"integrity": "sha512-6gS8pZzSXdyRHTIqoqSVknxolr1kzfy4/CeDnrzsVz8TTIWUbOBr6gnzOmTYJ3eXQNh4IYHIGi5aIL7sOZ2G/g==",
-			"license": "MIT",
-			"peer": true
+			"license": "MIT"
 		},
 		"node_modules/@smithy/abort-controller": {
 			"version": "4.2.4",
@@ -3844,7 +3842,6 @@
 			"integrity": "sha512-EMYTY4+rNa7TaRZYzCqhQslEkACEZzWc363jOYuc90oJrgvlWTcgqTxcGSIJim48hPaXwYlHyatRnnMmTFf5tA==",
 			"devOptional": true,
 			"license": "MIT",
-			"peer": true,
 			"dependencies": {
 				"@sveltejs/acorn-typescript": "^1.0.5",
 				"@types/cookie": "^0.6.0",
@@ -3877,7 +3874,6 @@
 			"integrity": "sha512-wojIS/7GYnJDYIg1higWj2ROA6sSRWvcR1PO/bqEyFr/5UZah26c8Cz4u0NaqjPeVltzsVpt2Tm8d2io0V+4Tw==",
 			"devOptional": true,
 			"license": "MIT",
-			"peer": true,
 			"dependencies": {
 				"@sveltejs/vite-plugin-svelte-inspector": "^4.0.1",
 				"debug": "^4.4.1",
@@ -3917,6 +3913,7 @@
 			"resolved": "https://registry.npmjs.org/@swc/helpers/-/helpers-0.5.17.tgz",
 			"integrity": "sha512-5IKx/Y13RsYd+sauPb2x+U/xZikHjolzfuDgTAl/Tdf3Q8rslRvC19NKDLgAJQ6wsqADk10ntlv08nPFw/gO/A==",
 			"license": "Apache-2.0",
+			"peer": true,
 			"dependencies": {
 				"tslib": "^2.8.0"
 			}
@@ -3962,6 +3959,7 @@
 			"resolved": "https://registry.npmjs.org/@testing-library/user-event/-/user-event-14.6.1.tgz",
 			"integrity": "sha512-vq7fv0rnt+QTXgPxr5Hjc210p6YKq2kmdziLgnsZGgLJ9e6VAShx1pACLuRjd/AS/sr7phAR58OIIpf0LlmQNw==",
 			"license": "MIT",
+			"peer": true,
 			"engines": {
 				"node": ">=12",
 				"npm": ">=6"
@@ -4007,7 +4005,8 @@
 			"version": "5.0.4",
 			"resolved": "https://registry.npmjs.org/@types/aria-query/-/aria-query-5.0.4.tgz",
 			"integrity": "sha512-rfT93uj5s0PRL7EzccGMs3brplhcrghnDoV26NqKhCAS1hVo+WdNsPvE/yb6ilfr5hi2MEk6d5EWJTKdxg8jVw==",
-			"license": "MIT"
+			"license": "MIT",
+			"peer": true
 		},
 		"node_modules/@types/chai": {
 			"version": "5.2.2",
@@ -4236,7 +4235,6 @@
 			"integrity": "sha512-tbsV1jPne5CkFQCgPBcDOt30ItF7aJoZL997JSF7MhGQqOeT3svWRYxiqlfA5RUdlHN6Fi+EI9bxqbdyAUZjYQ==",
 			"dev": true,
 			"license": "BSD-2-Clause",
-			"peer": true,
 			"dependencies": {
 				"@typescript-eslint/scope-manager": "6.21.0",
 				"@typescript-eslint/types": "6.21.0",
@@ -4635,7 +4633,6 @@
 			"resolved": "https://registry.npmjs.org/acorn/-/acorn-8.14.1.tgz",
 			"integrity": "sha512-OvQ/2pUDKmgfCg++xsTX1wGxfTaszcHVcTctW4UJB4hibJx2HXxxO5UmVgyjMa+ZDsiaf5wWLXYpRWMmBI0QHg==",
 			"license": "MIT",
-			"peer": true,
 			"bin": {
 				"acorn": "bin/acorn"
 			},
@@ -4819,6 +4816,7 @@
 			"resolved": "https://registry.npmjs.org/aria-query/-/aria-query-5.3.0.tgz",
 			"integrity": "sha512-b0P0sZPKtyu8HkeRAfCq0IfURZK+SuwMjY1UXGBU27wpAiTwQAIlq56IbIO+ytk/JjS1fMR14ee5WBBfKi5J6A==",
 			"license": "Apache-2.0",
+			"peer": true,
 			"dependencies": {
 				"dequal": "^2.0.3"
 			}
@@ -5084,7 +5082,6 @@
 				}
 			],
 			"license": "MIT",
-			"peer": true,
 			"dependencies": {
 				"caniuse-lite": "^1.0.30001718",
 				"electron-to-chromium": "^1.5.160",
@@ -5754,7 +5751,8 @@
 			"version": "0.5.16",
 			"resolved": "https://registry.npmjs.org/dom-accessibility-api/-/dom-accessibility-api-0.5.16.tgz",
 			"integrity": "sha512-X7BJ2yElsnOJ30pZF4uIIDfBEVgF4XEBxL9Bxhy6dnrm5hkzqmsWHGTiHqRiITNhMyFLyAiWndIJP7Z1NTteDg==",
-			"license": "MIT"
+			"license": "MIT",
+			"peer": true
 		},
 		"node_modules/domexception": {
 			"version": "4.0.0",
@@ -5828,7 +5826,6 @@
 			"resolved": "https://registry.npmjs.org/elysia/-/elysia-1.3.4.tgz",
 			"integrity": "sha512-kAfM3Zwovy3z255IZgTKVxBw91HbgKhYl3TqrGRdZqqr+Fd+4eKOfvxgaKij22+MZLczPzIHtscAmvfpI3+q/A==",
 			"license": "MIT",
-			"peer": true,
 			"dependencies": {
 				"cookie": "^1.0.2",
 				"exact-mirror": "0.1.2",
@@ -6029,7 +6026,6 @@
 			"deprecated": "This version is no longer supported. Please see https://eslint.org/version-support for other options.",
 			"dev": true,
 			"license": "MIT",
-			"peer": true,
 			"dependencies": {
 				"@eslint-community/eslint-utils": "^4.2.0",
 				"@eslint-community/regexpp": "^4.6.1",
@@ -6667,7 +6663,6 @@
 			"resolved": "https://registry.npmjs.org/file-type/-/file-type-21.0.0.tgz",
 			"integrity": "sha512-ek5xNX2YBYlXhiUXui3D/BXa3LdqPmoLJ7rqEx2bKJ7EAUEfmXgW0Das7Dc6Nr9MvqaOnIqiPV0mZk/r/UpNAg==",
 			"license": "MIT",
-			"peer": true,
 			"dependencies": {
 				"@tokenizer/inflate": "^0.2.7",
 				"strtok3": "^10.2.2",
@@ -7789,7 +7784,8 @@
 			"version": "4.0.0",
 			"resolved": "https://registry.npmjs.org/js-tokens/-/js-tokens-4.0.0.tgz",
 			"integrity": "sha512-RdJUflcE3cUzKiMqQgsCu06FPu9UdIJO0beYbPhHN4k6apgJtifcoCtT9bcxOpYBtpD2kCM6Sbzg4CausW/PKQ==",
-			"license": "MIT"
+			"license": "MIT",
+			"peer": true
 		},
 		"node_modules/js-yaml": {
 			"version": "4.1.0",
@@ -9561,7 +9557,6 @@
 			"integrity": "sha512-cJW4Xd/G3v5ovXtJJ52MAOclqeac9S/aGGgRzLabuF8TnIb6xHvMzKIa6JmrRzUkeXJgfL1MhukP0NK6l39h3A==",
 			"devOptional": true,
 			"license": "Apache-2.0",
-			"peer": true,
 			"dependencies": {
 				"playwright-core": "1.55.1"
 			},
@@ -9607,7 +9602,6 @@
 				}
 			],
 			"license": "MIT",
-			"peer": true,
 			"dependencies": {
 				"nanoid": "^3.3.11",
 				"picocolors": "^1.1.1",
@@ -9839,7 +9833,6 @@
 			"integrity": "sha512-QQtaxnoDJeAkDvDKWCLiwIXkTgRhwYDEQCghU9Z6q03iyek/rxRh/2lC3HB7P8sWT2xC/y5JDctPLBIGzHKbhw==",
 			"dev": true,
 			"license": "MIT",
-			"peer": true,
 			"bin": {
 				"prettier": "bin/prettier.cjs"
 			},
@@ -9856,7 +9849,6 @@
 			"integrity": "sha512-pn1ra/0mPObzqoIQn/vUTR3ZZI6UuZ0sHqMK5x2jMLGrs53h0sXhkVuDcrlssHwIMk7FYrMjHBPoUSyyEEDlBQ==",
 			"dev": true,
 			"license": "MIT",
-			"peer": true,
 			"peerDependencies": {
 				"prettier": "^3.0.0",
 				"svelte": "^3.2.0 || ^4.0.0-next.0 || ^5.0.0-next.0"
@@ -9946,6 +9938,7 @@
 			"resolved": "https://registry.npmjs.org/pretty-format/-/pretty-format-27.5.1.tgz",
 			"integrity": "sha512-Qb1gy5OrP5+zDf2Bvnzdl3jsTf1qXVMazbvCoKhtKqVs4/YK4ozX4gKQJJVyNe+cajNPn0KoC0MC3FUmaHWEmQ==",
 			"license": "MIT",
+			"peer": true,
 			"dependencies": {
 				"ansi-regex": "^5.0.1",
 				"ansi-styles": "^5.0.0",
@@ -9960,6 +9953,7 @@
 			"resolved": "https://registry.npmjs.org/ansi-styles/-/ansi-styles-5.2.0.tgz",
 			"integrity": "sha512-Cxwpt2SfTzTtXcfOlzGEee8O+c+MmUgGrNiBcXnuWxuFJHe6a5Hz7qwhwe5OgaSYI0IJvkLqWX1ASG+cJOkEiA==",
 			"license": "MIT",
+			"peer": true,
 			"engines": {
 				"node": ">=10"
 			},
@@ -10168,7 +10162,8 @@
 			"version": "17.0.2",
 			"resolved": "https://registry.npmjs.org/react-is/-/react-is-17.0.2.tgz",
 			"integrity": "sha512-w2GsyukL62IJnlaff/nRegPQR94C/XXamvMWmSHRJ4y7Ts/4ocGRmTHvOs8PSE6pB3dWOrD/nueuU5sduBsQ4w==",
-			"license": "MIT"
+			"license": "MIT",
+			"peer": true
 		},
 		"node_modules/read-cache": {
 			"version": "1.0.0",
@@ -10332,7 +10327,6 @@
 			"resolved": "https://registry.npmjs.org/rollup/-/rollup-4.41.1.tgz",
 			"integrity": "sha512-cPmwD3FnFv8rKMBc1MxWCwVQFxwf1JEmSX3iQXrRVVG15zerAIXRjMFVWnd5Q5QvgKF7Aj+5ykXFhUl+QGnyOw==",
 			"license": "MIT",
-			"peer": true,
 			"dependencies": {
 				"@types/estree": "1.0.7"
 			},
@@ -11251,7 +11245,6 @@
 			"resolved": "https://registry.npmjs.org/svelte/-/svelte-5.33.14.tgz",
 			"integrity": "sha512-kRlbhIlMTijbFmVDQFDeKXPLlX1/ovXwV0I162wRqQhRcygaqDIcu1d/Ese3H2uI+yt3uT8E7ndgDthQv5v5BA==",
 			"license": "MIT",
-			"peer": true,
 			"dependencies": {
 				"@ampproject/remapping": "^2.3.0",
 				"@jridgewell/sourcemap-codec": "^1.5.0",
@@ -11391,7 +11384,6 @@
 			"resolved": "https://registry.npmjs.org/tailwindcss/-/tailwindcss-3.4.17.tgz",
 			"integrity": "sha512-w33E2aCvSDP0tW9RZuNXadXlkHXqFzSkQew/aIa2i/Sj8fThxwovwlXHSPXTbAHwEIhBFXAedUhP2tueAKP8Og==",
 			"license": "MIT",
-			"peer": true,
 			"dependencies": {
 				"@alloc/quick-lru": "^5.2.0",
 				"arg": "^5.0.2",
@@ -11849,7 +11841,6 @@
 			"resolved": "https://registry.npmjs.org/typescript/-/typescript-5.8.3.tgz",
 			"integrity": "sha512-p1diW6TqL9L07nNxvRMM7hMMw4c5XOo/1ibL4aAIGmSAt9slTE1Xgw5KWuof2uTOvCg9BY7ZRi+GaF+7sfgPeQ==",
 			"license": "Apache-2.0",
-			"peer": true,
 			"bin": {
 				"tsc": "bin/tsc",
 				"tsserver": "bin/tsserver"
@@ -12085,7 +12076,6 @@
 			"resolved": "https://registry.npmjs.org/vite/-/vite-6.3.5.tgz",
 			"integrity": "sha512-cZn6NDFE7wdTpINgs++ZJ4N49W2vRp8LCKrn3Ob1kYNtOo21vfDoaV5GzBfLU4MovSAB8uNRm4jgzVQZ+mBzPQ==",
 			"license": "MIT",
-			"peer": true,
 			"dependencies": {
 				"esbuild": "^0.25.0",
 				"fdir": "^6.4.4",
@@ -12221,7 +12211,6 @@
 			"resolved": "https://registry.npmjs.org/vitest/-/vitest-3.2.2.tgz",
 			"integrity": "sha512-fyNn/Rp016Bt5qvY0OQvIUCwW2vnaEBLxP42PmKbNIoasSYjML+8xyeADOPvBe+Xfl/ubIw4og7Lt9jflRsCNw==",
 			"license": "MIT",
-			"peer": true,
 			"dependencies": {
 				"@types/chai": "^5.2.2",
 				"@vitest/expect": "3.2.2",
@@ -12544,7 +12533,6 @@
 			"resolved": "https://registry.npmjs.org/ws/-/ws-8.18.2.tgz",
 			"integrity": "sha512-DMricUmwGZUVr++AEAe2uiVM7UoO9MAVZMDu05UQOaUII0lp+zOzLLU4Xqh/JvTqklB1T4uELaaPBKyjE1r4fQ==",
 			"license": "MIT",
-			"peer": true,
 			"engines": {
 				"node": ">=10.0.0"
 			},
@@ -12667,7 +12655,6 @@
 			"resolved": "https://registry.npmjs.org/zod/-/zod-3.25.55.tgz",
 			"integrity": "sha512-219huNnkSLQnLsQ3uaRjXsxMrVm5C9W3OOpEVt2k5tvMKuA8nBSu38e0B//a+he9Iq2dvmk2VyYVlHqiHa4YBA==",
 			"license": "MIT",
-			"peer": true,
 			"funding": {
 				"url": "https://github.com/sponsors/colinhacks"
 			}
diff --git a/src/lib/components/chat/UploadedFile.svelte b/src/lib/components/chat/UploadedFile.svelte
index 8dcca28..c2ed6db 100644
--- a/src/lib/components/chat/UploadedFile.svelte
+++ b/src/lib/components/chat/UploadedFile.svelte
@@ -95,7 +95,7 @@
 					</p>
 				{/if}
 				<button
-					class="absolute right-4 top-4 text-xl text-gray-500 hover:text-gray-800"
+					class="absolute right-4 top-4 text-xl text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200"
 					onclick={() => (showModal = false)}
 				>
 					<CarbonClose class="text-xl" />
```

---

## swift-hard [swift]

**Issue:** Fix crash on trailing-comma generic arguments

> ## Bug description
> 
> Sourcery crashes with a fatal error when parsing Swift code that contains trailing commas in generic type parameters. This is valid syntax in Swift 6.2+, but causes the parser to create empty generic arguments which then crash on force-unwrap operations.
> 
> ## Steps to reproduce
> 
> 1. Create a Swift file with a generic type that has a trailing comma:
> ```swift
> let asyncTaskData: OrderedDictionary<
>     String,
>     (status: Result<Int, Error>, messages: [String]),
> >
> ```
> 
> 2. Run Sourcery on this file
> 3. Observe crash: `Fatal error: Unexpectedly found nil while unwrapping an Optional value`
> 
> ## Expected behavior
> 
> Sourcery should handle trailing commas in generic arguments gracefully and not crash. Empty generic parameters created by trailing commas should be filtered out during parsing.
> 
> ## Environment
> 
> - Valid Swift 6.2+ syntax
> - Affects both SwiftSyntax-based and string-based parsing paths

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 78.8s | 22 | 325850 | 3548 | $0.1806 |
| **with-lumen** | 96.0s | 16 | 212090 | 2267 | $0.1386 |

### baseline

**Rating: Good**

```diff
diff --git a/Package.resolved b/Package.resolved
index 97abb9a..8533566 100644
--- a/Package.resolved
+++ b/Package.resolved
@@ -95,8 +95,8 @@
       "kind" : "remoteSourceControl",
       "location" : "https://github.com/apple/swift-argument-parser.git",
       "state" : {
-        "revision" : "8f4d2753f0e4778c76d5f05ad16c74f707390531",
-        "version" : "1.2.3"
+        "revision" : "6a52f3251125d74daf04fcbd5e6f08a75d074382",
+        "version" : "1.8.2"
       }
     },
     {
@@ -138,28 +138,28 @@
     {
       "identity" : "swift-driver",
       "kind" : "remoteSourceControl",
-      "location" : "https://github.com/art-divin/swift-driver.git",
+      "location" : "https://github.com/swiftlang/swift-driver.git",
       "state" : {
-        "revision" : "1c07ced84c1dfc1f9c3253dcbaa216fc9c76ee25",
-        "version" : "1.0.1"
+        "branch" : "release/6.2",
+        "revision" : "aedacc6c1583db4f2989a367e3c41968558a5b8e"
       }
     },
     {
       "identity" : "swift-llbuild",
       "kind" : "remoteSourceControl",
-      "location" : "https://github.com/art-divin/swift-llbuild.git",
+      "location" : "https://github.com/swiftlang/swift-llbuild.git",
       "state" : {
-        "revision" : "783aec21649a6c47d1a8314db4144bdceb11df30",
-        "version" : "1.0.0"
+        "branch" : "release/6.2",
+        "revision" : "073dff55529d7c4ecbd615ab5f5ac52ae5b380da"
       }
     },
     {
       "identity" : "swift-package-manager",
       "kind" : "remoteSourceControl",
-      "location" : "https://github.com/art-divin/swift-package-manager.git",
+      "location" : "https://github.com/swiftlang/swift-package-manager.git",
       "state" : {
-        "revision" : "7df9321541e544d711dd93f5b97dd4e8bf7e100e",
-        "version" : "1.0.8"
+        "branch" : "swift-6.2-RELEASE",
+        "revision" : "3cf1cb66ebde00dab737dcebad96218e2017530d"
       }
     },
     {
@@ -167,8 +167,8 @@
       "kind" : "remoteSourceControl",
       "location" : "https://github.com/swiftlang/swift-syntax.git",
       "state" : {
-        "revision" : "0687f71944021d616d34d922343dcef086855920",
-        "version" : "600.0.1"
+        "branch" : "release/6.2",
+        "revision" : "5a87516fc3dddbd23cb76358eb489915ee86b444"
       }
     },
     {
@@ -180,13 +180,22 @@
         "version" : "1.1.1"
       }
     },
+    {
+      "identity" : "swift-toolchain-sqlite",
+      "kind" : "remoteSourceControl",
+      "location" : "https://github.com/swiftlang/swift-toolchain-sqlite",
+      "state" : {
+        "revision" : "d9b11cb79071d5ab992ff47dfd2e8d6e19418d97",
+        "version" : "1.0.13"
+      }
+    },
     {
       "identity" : "swift-tools-support-core",
       "kind" : "remoteSourceControl",
-      "location" : "https://github.com/art-divin/swift-tools-support-core.git",
+      "location" : "https://github.com/swiftlang/swift-tools-support-core.git",
       "state" : {
-        "revision" : "930e82e5ae2432c71fe05f440b5d778285270bdb",
-        "version" : "1.0.0"
+        "branch" : "release/6.2",
+        "revision" : "5a993c8848487c934d53ffceef8e1cad0a241dc1"
       }
     },
     {
diff --git a/SourceryFramework/Sources/Parsing/String+TypeInference.swift b/SourceryFramework/Sources/Parsing/String+TypeInference.swift
index 8263e8b..ed5b634 100644
--- a/SourceryFramework/Sources/Parsing/String+TypeInference.swift
+++ b/SourceryFramework/Sources/Parsing/String+TypeInference.swift
@@ -243,7 +243,7 @@ extension String {
         }
         
         let body = trimmed[trimmed.index(after: start)..<trimmed.index(before: trimmed.endIndex)]
-        return GenericType(name: String(trimmed[..<start]), typeParameters: String(body).commaSeparated().map({ value in
+        return GenericType(name: String(trimmed[..<start]), typeParameters: String(body).commaSeparated().filter({ !$0.trimmed.isEmpty }).map({ value in
             let stripped = value.stripped()
             return GenericTypeParameter(typeName: stripped.inferType ?? TypeName(stripped))
         }))
diff --git a/SourceryFramework/Sources/Parsing/SwiftSyntax/AST/GenericType+SwiftSyntax.swift b/SourceryFramework/Sources/Parsing/SwiftSyntax/AST/GenericType+SwiftSyntax.swift
index 1c59fa8..e2f35c7 100644
--- a/SourceryFramework/Sources/Parsing/SwiftSyntax/AST/GenericType+SwiftSyntax.swift
+++ b/SourceryFramework/Sources/Parsing/SwiftSyntax/AST/GenericType+SwiftSyntax.swift
@@ -9,14 +9,18 @@ extension GenericType {
         let parameters = node.arguments.map { argument -> GenericTypeParameter? in
             switch argument.argument {
             case .type(let type):
+                // trailing comma produces an empty argument, skip it
+                guard !type.trimmedDescription.isEmpty else { return nil }
                 return GenericTypeParameter(typeName: TypeName(type))
             default: // case .expr
                 return nil
             }
         }.compactMap({ $0 })
         #else
-        let parameters = node.arguments.map { argument in
-            GenericTypeParameter(typeName: TypeName(argument.argument))
+        let parameters = node.arguments.compactMap { argument -> GenericTypeParameter? in
+            // trailing comma produces an empty argument, skip it
+            guard !argument.argument.trimmedDescription.isEmpty else { return nil }
+            return GenericTypeParameter(typeName: TypeName(argument.argument))
         }
         #endif
 
```

### with-lumen

**Rating: INVALID (lumen not used)**

```diff
diff --git a/Package.resolved b/Package.resolved
index 97abb9a..8533566 100644
--- a/Package.resolved
+++ b/Package.resolved
@@ -95,8 +95,8 @@
       "kind" : "remoteSourceControl",
       "location" : "https://github.com/apple/swift-argument-parser.git",
       "state" : {
-        "revision" : "8f4d2753f0e4778c76d5f05ad16c74f707390531",
-        "version" : "1.2.3"
+        "revision" : "6a52f3251125d74daf04fcbd5e6f08a75d074382",
+        "version" : "1.8.2"
       }
     },
     {
@@ -138,28 +138,28 @@
     {
       "identity" : "swift-driver",
       "kind" : "remoteSourceControl",
-      "location" : "https://github.com/art-divin/swift-driver.git",
+      "location" : "https://github.com/swiftlang/swift-driver.git",
       "state" : {
-        "revision" : "1c07ced84c1dfc1f9c3253dcbaa216fc9c76ee25",
-        "version" : "1.0.1"
+        "branch" : "release/6.2",
+        "revision" : "aedacc6c1583db4f2989a367e3c41968558a5b8e"
       }
     },
     {
       "identity" : "swift-llbuild",
       "kind" : "remoteSourceControl",
-      "location" : "https://github.com/art-divin/swift-llbuild.git",
+      "location" : "https://github.com/swiftlang/swift-llbuild.git",
       "state" : {
-        "revision" : "783aec21649a6c47d1a8314db4144bdceb11df30",
-        "version" : "1.0.0"
+        "branch" : "release/6.2",
+        "revision" : "073dff55529d7c4ecbd615ab5f5ac52ae5b380da"
       }
     },
     {
       "identity" : "swift-package-manager",
       "kind" : "remoteSourceControl",
-      "location" : "https://github.com/art-divin/swift-package-manager.git",
+      "location" : "https://github.com/swiftlang/swift-package-manager.git",
       "state" : {
-        "revision" : "7df9321541e544d711dd93f5b97dd4e8bf7e100e",
-        "version" : "1.0.8"
+        "branch" : "swift-6.2-RELEASE",
+        "revision" : "3cf1cb66ebde00dab737dcebad96218e2017530d"
       }
     },
     {
@@ -167,8 +167,8 @@
       "kind" : "remoteSourceControl",
       "location" : "https://github.com/swiftlang/swift-syntax.git",
       "state" : {
-        "revision" : "0687f71944021d616d34d922343dcef086855920",
-        "version" : "600.0.1"
+        "branch" : "release/6.2",
+        "revision" : "5a87516fc3dddbd23cb76358eb489915ee86b444"
       }
     },
     {
@@ -180,13 +180,22 @@
         "version" : "1.1.1"
       }
     },
+    {
+      "identity" : "swift-toolchain-sqlite",
+      "kind" : "remoteSourceControl",
+      "location" : "https://github.com/swiftlang/swift-toolchain-sqlite",
+      "state" : {
+        "revision" : "d9b11cb79071d5ab992ff47dfd2e8d6e19418d97",
+        "version" : "1.0.13"
+      }
+    },
     {
       "identity" : "swift-tools-support-core",
       "kind" : "remoteSourceControl",
-      "location" : "https://github.com/art-divin/swift-tools-support-core.git",
+      "location" : "https://github.com/swiftlang/swift-tools-support-core.git",
       "state" : {
-        "revision" : "930e82e5ae2432c71fe05f440b5d778285270bdb",
-        "version" : "1.0.0"
+        "branch" : "release/6.2",
+        "revision" : "5a993c8848487c934d53ffceef8e1cad0a241dc1"
       }
     },
     {
diff --git a/SourceryFramework/Sources/Parsing/String+TypeInference.swift b/SourceryFramework/Sources/Parsing/String+TypeInference.swift
index 8263e8b..6ca6d1b 100644
--- a/SourceryFramework/Sources/Parsing/String+TypeInference.swift
+++ b/SourceryFramework/Sources/Parsing/String+TypeInference.swift
@@ -243,8 +243,9 @@ extension String {
         }
         
         let body = trimmed[trimmed.index(after: start)..<trimmed.index(before: trimmed.endIndex)]
-        return GenericType(name: String(trimmed[..<start]), typeParameters: String(body).commaSeparated().map({ value in
+        return GenericType(name: String(trimmed[..<start]), typeParameters: String(body).commaSeparated().compactMap({ value in
             let stripped = value.stripped()
+            guard !stripped.isEmpty else { return nil }
             return GenericTypeParameter(typeName: stripped.inferType ?? TypeName(stripped))
         }))
     }
diff --git a/SourceryFramework/Sources/Parsing/SwiftSyntax/AST/GenericType+SwiftSyntax.swift b/SourceryFramework/Sources/Parsing/SwiftSyntax/AST/GenericType+SwiftSyntax.swift
index 1c59fa8..5f38440 100644
--- a/SourceryFramework/Sources/Parsing/SwiftSyntax/AST/GenericType+SwiftSyntax.swift
+++ b/SourceryFramework/Sources/Parsing/SwiftSyntax/AST/GenericType+SwiftSyntax.swift
@@ -13,11 +13,11 @@ extension GenericType {
             default: // case .expr
                 return nil
             }
-        }.compactMap({ $0 })
+        }.compactMap({ $0 }).filter({ !$0.typeName.name.isEmpty })
         #else
         let parameters = node.arguments.map { argument in
             GenericTypeParameter(typeName: TypeName(argument.argument))
-        }
+        }.filter({ !$0.typeName.name.isEmpty })
         #endif
 
         self.init(name: name, typeParameters: parameters)
```

---

## typescript-hard [typescript]

**Issue:** Negative flag (--no-) negation does not propagate to aliases or main option

> When using the `--no-` prefix to negate a boolean flag that has aliases, the negation is not correctly propagated between the alias and the main option name.
> 
> For example, given a boolean argument `verbose` with default `true` and alias `v`:
> 
> - Running with `--no-v` sets `v=false` but leaves `verbose=true` (the default), because the propagation logic only copies values when the target is `undefined`. Since `verbose` has a default of `true`, it is not `undefined`, so the negated value from the alias never overwrites it.
> 
> - Similarly, running with `--no-verbose` sets `verbose=false` but does not propagate the negation to the alias `v`.
> 
> Reproduction:
> 
> ```ts
> import { parseRawArgs } from 'citty';
> 
> // Case 1: --no-alias should propagate to main option
> const result1 = parseRawArgs(['--no-v'], {
>   boolean: ['verbose'],
>   alias: { v: ['verbose'] },
>   default: { verbose: true },
> });
> console.log(result1);
> // Actual:   { _: [], v: false, verbose: true }
> // Expected: { _: [], v: false, verbose: false }
> 
> // Case 2: --no-main should propagate to aliases
> const result2 = parseRawArgs(['--no-verbose'], {
>   boolean: ['verbose'],
>   alias: { v: ['verbose'] },
>   default: { verbose: true },
> });
> console.log(result2);
> // Actual:   { _: [], verbose: false }  (v is missing/undefined)
> // Expected: { _: [], v: false, verbose: false }
> ```
> 
> The root cause is in the argument parser's handling of negated flags. When a `--no-` prefixed flag is encountered, the parser sets only the literal flag name to `false` without resolving it through the alias mapping. The negation needs to be resolved bidirectionally: if the negated name is an alias, the main option should also be set to `false`; if the negated name is the main option, all its aliases should also be set to `false`. This must happen before the general alias propagation step, which skips values that are already defined (non-undefined), so default values block the propagation of negated flags.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | — | — | — | — | — |
| **with-lumen** | — | — | — | — | — |

### baseline

### with-lumen

**Rating: INVALID (lumen not used)**


