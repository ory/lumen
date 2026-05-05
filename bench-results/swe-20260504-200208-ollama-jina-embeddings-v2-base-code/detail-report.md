# SWE-Bench Detail Report

Generated: 2026-05-05 00:32 UTC

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
| **baseline** | — | — | — | — | — |
| **with-lumen** | — | — | — | — | — |

### baseline

### with-lumen

**Rating: INVALID (lumen not used)**

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
| **baseline** | 374.1s | 177 | 1582347 | 14422 | $1.1953 |
| **with-lumen** | 769.7s | 370 | 2464011 | 12380 | $0.3724 |

### baseline

**Rating: Good**

```diff
diff --git a/CMakeLists.txt b/CMakeLists.txt
index 53afc51..c34d565 100644
--- a/CMakeLists.txt
+++ b/CMakeLists.txt
@@ -299,10 +299,10 @@ endfunction()
 
 # Define the fmt library, its includes and the needed defines.
 set(FMT_HEADERS)
-add_headers(FMT_HEADERS args.h base.h chrono.h color.h compile.h core.h format.h
+add_headers(FMT_HEADERS args.h base.h c.h chrono.h color.h compile.h core.h format.h
                         format-inl.h os.h ostream.h printf.h ranges.h std.h
                         xchar.h)
-set(FMT_SOURCES src/format.cc)
+set(FMT_SOURCES src/format.cc src/fmt-c.cc)
 
 add_module_library(fmt src/fmt.cc FALLBACK
                    ${FMT_SOURCES} ${FMT_HEADERS} README.md ChangeLog.md
diff --git a/include/fmt/c.h b/include/fmt/c.h
new file mode 100644
index 0000000..aed2bc0
--- /dev/null
+++ b/include/fmt/c.h
@@ -0,0 +1,93 @@
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
+#include <stddef.h>
+
+/**
+ * \file c.h
+ * \brief A type-safe C API for the {fmt} library.
+ *
+ * Unlike printf, type information is not encoded in the format string.
+ * Instead, each argument is pushed with an explicit typed function, making
+ * the API safe for use from C and languages with a C FFI.
+ *
+ * Example:
+ *
+ *   fmt_args_t* a = fmt_args_new();
+ *   fmt_push_int(a, 42);
+ *   fmt_push_cstr(a, "world");
+ *   char buf[256];
+ *   fmt_format_to(buf, sizeof(buf), "The answer is {} and hello {}", a);
+ *   fmt_args_free(a);
+ */
+
+#ifdef __cplusplus
+extern "C" {
+#endif
+
+/** Opaque handle to a list of formatting arguments. */
+typedef struct fmt_args_s fmt_args_t;
+
+/** Allocates a new, empty argument list. Must be freed with fmt_args_free(). */
+fmt_args_t* fmt_args_new(void);
+
+/** Frees an argument list previously created with fmt_args_new(). */
+void fmt_args_free(fmt_args_t* args);
+
+/** Appends a signed integer argument. */
+void fmt_push_int(fmt_args_t* args, int value);
+
+/** Appends an unsigned integer argument. */
+void fmt_push_uint(fmt_args_t* args, unsigned int value);
+
+/** Appends a signed long long argument. */
+void fmt_push_long_long(fmt_args_t* args, long long value);
+
+/** Appends an unsigned long long argument. */
+void fmt_push_ulong_long(fmt_args_t* args, unsigned long long value);
+
+/** Appends a double argument. */
+void fmt_push_double(fmt_args_t* args, double value);
+
+/** Appends a boolean argument (zero is false, non-zero is true). */
+void fmt_push_bool(fmt_args_t* args, int value);
+
+/** Appends a null-terminated C string argument. */
+void fmt_push_cstr(fmt_args_t* args, const char* value);
+
+/** Appends a string argument given as a data pointer and length. */
+void fmt_push_str(fmt_args_t* args, const char* data, size_t size);
+
+/** Appends a pointer argument (formatted as a hex address). */
+void fmt_push_pointer(fmt_args_t* args, const void* value);
+
+/**
+ * Formats arguments according to the format string and writes the result into
+ * buf.
+ *
+ * Behaves like snprintf: returns the total number of characters that would
+ * have been written (excluding the null terminator), even if the buffer was
+ * too small. Returns -1 on error (e.g. invalid format string).
+ *
+ * The output is always null-terminated when buf is non-null and n > 0.
+ *
+ * @param buf     Output buffer (may be NULL when n == 0).
+ * @param n       Size of the output buffer in bytes.
+ * @param fmt_str Null-terminated format string using {fmt} syntax.
+ * @param args    Argument list built with fmt_push_* functions.
+ * @return        Number of characters written (excluding NUL), or -1 on error.
+ */
+int fmt_format_to(char* buf, size_t n, const char* fmt_str, fmt_args_t* args);
+
+#ifdef __cplusplus
+}
+#endif
+
+#endif  /* FMT_C_H_ */
diff --git a/src/fmt-c.cc b/src/fmt-c.cc
new file mode 100644
index 0000000..2e07f31
--- /dev/null
+++ b/src/fmt-c.cc
@@ -0,0 +1,79 @@
+// Formatting library for C++ - C API implementation
+//
+// Copyright (c) 2012 - present, Victor Zverovich
+// All rights reserved.
+//
+// For the license information refer to format.h.
+
+#include "fmt/c.h"
+
+#include <string.h>  // memcpy
+
+#include "fmt/args.h"
+#include "fmt/format.h"
+
+struct fmt_args_s {
+  fmt::dynamic_format_arg_store<fmt::context> store;
+};
+
+extern "C" {
+
+fmt_args_t* fmt_args_new(void) { return new fmt_args_t; }
+
+void fmt_args_free(fmt_args_t* args) { delete args; }
+
+void fmt_push_int(fmt_args_t* args, int value) {
+  args->store.push_back(value);
+}
+
+void fmt_push_uint(fmt_args_t* args, unsigned int value) {
+  args->store.push_back(value);
+}
+
+void fmt_push_long_long(fmt_args_t* args, long long value) {
+  args->store.push_back(value);
+}
+
+void fmt_push_ulong_long(fmt_args_t* args, unsigned long long value) {
+  args->store.push_back(value);
+}
+
+void fmt_push_double(fmt_args_t* args, double value) {
+  args->store.push_back(value);
+}
+
+void fmt_push_bool(fmt_args_t* args, int value) {
+  args->store.push_back(value != 0);
+}
+
+void fmt_push_cstr(fmt_args_t* args, const char* value) {
+  args->store.push_back(value);
+}
+
+void fmt_push_str(fmt_args_t* args, const char* data, size_t size) {
+  args->store.push_back(fmt::string_view(data, size));
+}
+
+void fmt_push_pointer(fmt_args_t* args, const void* value) {
+  args->store.push_back(value);
+}
+
+int fmt_format_to(char* buf, size_t n, const char* fmt_str,
+                  fmt_args_t* args) {
+  try {
+    auto memory = fmt::memory_buffer();
+    fmt::detail::vformat_to(memory, fmt::string_view(fmt_str),
+                            fmt::format_args(args->store));
+    size_t len = memory.size();
+    if (buf && n > 0) {
+      size_t copy_len = len < n - 1 ? len : n - 1;
+      memcpy(buf, memory.data(), copy_len);
+      buf[copy_len] = '\0';
+    }
+    return static_cast<int>(len);
+  } catch (...) {
+    return -1;
+  }
+}
+
+}  // extern "C"
```

### with-lumen

**Rating: Good**

```diff
diff --git a/CMakeLists.txt b/CMakeLists.txt
index 53afc51..7c16441 100644
--- a/CMakeLists.txt
+++ b/CMakeLists.txt
@@ -299,10 +299,10 @@ endfunction()
 
 # Define the fmt library, its includes and the needed defines.
 set(FMT_HEADERS)
-add_headers(FMT_HEADERS args.h base.h chrono.h color.h compile.h core.h format.h
-                        format-inl.h os.h ostream.h printf.h ranges.h std.h
-                        xchar.h)
-set(FMT_SOURCES src/format.cc)
+add_headers(FMT_HEADERS args.h base.h c.h chrono.h color.h compile.h core.h
+                        format.h format-inl.h os.h ostream.h printf.h ranges.h
+                        std.h xchar.h)
+set(FMT_SOURCES src/format.cc src/c.cc)
 
 add_module_library(fmt src/fmt.cc FALLBACK
                    ${FMT_SOURCES} ${FMT_HEADERS} README.md ChangeLog.md
diff --git a/include/fmt/c.h b/include/fmt/c.h
new file mode 100644
index 0000000..a202ae3
--- /dev/null
+++ b/include/fmt/c.h
@@ -0,0 +1,85 @@
+// Formatting library for C - C API
+//
+// Copyright (c) 2012 - present, Victor Zverovich
+// All rights reserved.
+//
+// For the license information refer to format.h.
+
+#ifndef FMT_C_H_
+#define FMT_C_H_
+
+#ifdef __cplusplus
+extern "C" {
+#endif
+
+#include <stddef.h>
+#include <stdint.h>
+#include <stdlib.h>
+#include <string.h>
+
+// Opaque handle for a format argument store.
+// This is a typedef for void* to maintain C compatibility.
+typedef void* fmt_arg_store;
+
+// Creates a new argument store. Must be freed with fmt_free_arg_store.
+fmt_arg_store fmt_create_arg_store(void);
+
+// Frees an argument store created with fmt_create_arg_store.
+void fmt_free_arg_store(fmt_arg_store store);
+
+// Adds a signed integer argument to the store.
+void fmt_add_int(fmt_arg_store store, int value);
+
+// Adds an unsigned integer argument to the store.
+void fmt_add_uint(fmt_arg_store store, unsigned int value);
+
+// Adds a signed long integer argument to the store.
+void fmt_add_long(fmt_arg_store store, long value);
+
+// Adds an unsigned long integer argument to the store.
+void fmt_add_ulong(fmt_arg_store store, unsigned long value);
+
+// Adds a signed long long integer argument to the store.
+void fmt_add_long_long(fmt_arg_store store, long long value);
+
+// Adds an unsigned long long integer argument to the store.
+void fmt_add_ulong_long(fmt_arg_store store, unsigned long long value);
+
+// Adds a floating-point argument to the store.
+void fmt_add_double(fmt_arg_store store, double value);
+
+// Adds a floating-point argument (long double) to the store.
+void fmt_add_long_double(fmt_arg_store store, long double value);
+
+// Adds a boolean argument to the store.
+void fmt_add_bool(fmt_arg_store store, int value);
+
+// Adds a character argument to the store.
+void fmt_add_char(fmt_arg_store store, char value);
+
+// Adds a C string argument to the store.
+// The string is copied into the store.
+void fmt_add_string(fmt_arg_store store, const char* value);
+
+// Adds a string argument with explicit length to the store.
+// The string is copied into the store.
+void fmt_add_string_len(fmt_arg_store store, const char* value, size_t len);
+
+// Adds a pointer argument to the store.
+void fmt_add_pointer(fmt_arg_store store, const void* value);
+
+// Formats a string according to the format specification and arguments.
+// Returns a newly allocated string that must be freed with free().
+// Returns NULL on error.
+char* fmt_format(const char* format, fmt_arg_store args);
+
+// Formats a string with explicit format length.
+// Returns a newly allocated string that must be freed with free().
+// Returns NULL on error.
+char* fmt_format_len(const char* format, size_t format_len, fmt_arg_store args);
+
+#ifdef __cplusplus
+}
+#endif
+
+#endif  // FMT_C_H_
diff --git a/src/c.cc b/src/c.cc
new file mode 100644
index 0000000..9740b04
--- /dev/null
+++ b/src/c.cc
@@ -0,0 +1,171 @@
+// Formatting library for C - C API implementation
+//
+// Copyright (c) 2012 - present, Victor Zverovich
+// All rights reserved.
+//
+// For the license information refer to format.h.
+
+#include "fmt/c.h"
+
+#include <cstring>
+
+#include "fmt/args.h"
+#include "fmt/format.h"
+
+extern "C" {
+
+// Opaque wrapper around fmt::dynamic_format_arg_store
+struct fmt_arg_store_impl {
+  fmt::dynamic_format_arg_store<fmt::format_context> store;
+};
+
+fmt_arg_store fmt_create_arg_store(void) {
+  try {
+    return new fmt_arg_store_impl();
+  } catch (...) {
+    return nullptr;
+  }
+}
+
+void fmt_free_arg_store(fmt_arg_store store) {
+  delete static_cast<fmt_arg_store_impl*>(store);
+}
+
+void fmt_add_int(fmt_arg_store store, int value) {
+  if (!store) return;
+  try {
+    static_cast<fmt_arg_store_impl*>(store)->store.push_back(value);
+  } catch (...) {
+  }
+}
+
+void fmt_add_uint(fmt_arg_store store, unsigned int value) {
+  if (!store) return;
+  try {
+    static_cast<fmt_arg_store_impl*>(store)->store.push_back(value);
+  } catch (...) {
+  }
+}
+
+void fmt_add_long(fmt_arg_store store, long value) {
+  if (!store) return;
+  try {
+    static_cast<fmt_arg_store_impl*>(store)->store.push_back(value);
+  } catch (...) {
+  }
+}
+
+void fmt_add_ulong(fmt_arg_store store, unsigned long value) {
+  if (!store) return;
+  try {
+    static_cast<fmt_arg_store_impl*>(store)->store.push_back(value);
+  } catch (...) {
+  }
+}
+
+void fmt_add_long_long(fmt_arg_store store, long long value) {
+  if (!store) return;
+  try {
+    static_cast<fmt_arg_store_impl*>(store)->store.push_back(value);
+  } catch (...) {
+  }
+}
+
+void fmt_add_ulong_long(fmt_arg_store store, unsigned long long value) {
+  if (!store) return;
+  try {
+    static_cast<fmt_arg_store_impl*>(store)->store.push_back(value);
+  } catch (...) {
+  }
+}
+
+void fmt_add_double(fmt_arg_store store, double value) {
+  if (!store) return;
+  try {
+    static_cast<fmt_arg_store_impl*>(store)->store.push_back(value);
+  } catch (...) {
+  }
+}
+
+void fmt_add_long_double(fmt_arg_store store, long double value) {
+  if (!store) return;
+  try {
+    static_cast<fmt_arg_store_impl*>(store)->store.push_back(value);
+  } catch (...) {
+  }
+}
+
+void fmt_add_bool(fmt_arg_store store, int value) {
+  if (!store) return;
+  try {
+    static_cast<fmt_arg_store_impl*>(store)->store.push_back(value != 0);
+  } catch (...) {
+  }
+}
+
+void fmt_add_char(fmt_arg_store store, char value) {
+  if (!store) return;
+  try {
+    static_cast<fmt_arg_store_impl*>(store)->store.push_back(value);
+  } catch (...) {
+  }
+}
+
+void fmt_add_string(fmt_arg_store store, const char* value) {
+  if (!store || !value) return;
+  try {
+    static_cast<fmt_arg_store_impl*>(store)->store.push_back(
+        fmt::string_view(value));
+  } catch (...) {
+  }
+}
+
+void fmt_add_string_len(fmt_arg_store store, const char* value, size_t len) {
+  if (!store || !value) return;
+  try {
+    static_cast<fmt_arg_store_impl*>(store)->store.push_back(
+        fmt::string_view(value, len));
+  } catch (...) {
+  }
+}
+
+void fmt_add_pointer(fmt_arg_store store, const void* value) {
+  if (!store) return;
+  try {
+    static_cast<fmt_arg_store_impl*>(store)->store.push_back(value);
+  } catch (...) {
+  }
+}
+
+char* fmt_format(const char* format, fmt_arg_store args) {
+  if (!format || !args) return nullptr;
+  try {
+    auto& store = static_cast<fmt_arg_store_impl*>(args)->store;
+    auto result = fmt::vformat(fmt::string_view(format), store);
+    char* output = static_cast<char*>(malloc(result.size() + 1));
+    if (!output) return nullptr;
+    std::memcpy(output, result.data(), result.size());
+    output[result.size()] = '\0';
+    return output;
+  } catch (...) {
+    return nullptr;
+  }
+}
+
+char* fmt_format_len(const char* format, size_t format_len, fmt_arg_store args) {
+  if (!format || !args) return nullptr;
+  try {
+    auto& store = static_cast<fmt_arg_store_impl*>(args)->store;
+    auto result =
+        fmt::vformat(fmt::string_view(format, format_len), store);
+    char* output = static_cast<char*>(malloc(result.size() + 1));
+    if (!output) return nullptr;
+    std::memcpy(output, result.data(), result.size());
+    output[result.size()] = '\0';
+    return output;
+  } catch (...) {
+    return nullptr;
+  }
+}
+
+}  // extern "C"
diff --git a/test/CMakeLists.txt b/test/CMakeLists.txt
index 3f7d24a..2c4e120 100644
--- a/test/CMakeLists.txt
+++ b/test/CMakeLists.txt
@@ -47,6 +47,7 @@ endif ()
 
 add_fmt_test(args-test)
 add_fmt_test(base-test)
+add_fmt_test(c-test)
 add_fmt_test(assert-test)
 add_fmt_test(chrono-test)
 add_fmt_test(color-test)
diff --git a/test/c-example.c b/test/c-example.c
new file mode 100644
index 0000000..bf67079
--- /dev/null
+++ b/test/c-example.c
@@ -0,0 +1,36 @@
+// Example C program demonstrating the fmt C API
+
+#include <fmt/c.h>
+#include <stdio.h>
+#include <stdlib.h>
+
+int main(void) {
+  // Create an argument store
+  fmt_arg_store args = fmt_create_arg_store();
+  if (!args) {
+    fprintf(stderr, "Failed to create argument store\n");
+    return 1;
+  }
+
+  // Add arguments to the store
+  fmt_add_string(args, "World");
+  fmt_add_int(args, 42);
+  fmt_add_double(args, 3.14159);
+
+  // Format a string
+  char* result = fmt_format("Hello, {}! The answer is {} and pi is {}", args);
+  if (!result) {
+    fprintf(stderr, "Failed to format string\n");
+    fmt_free_arg_store(args);
+    return 1;
+  }
+
+  // Print the result
+  printf("%s\n", result);
+
+  // Clean up
+  free(result);
+  fmt_free_arg_store(args);
+
+  return 0;
+}
diff --git a/test/c-test.cc b/test/c-test.cc
new file mode 100644
index 0000000..ebef0fc
--- /dev/null
+++ b/test/c-test.cc
@@ -0,0 +1,134 @@
+// Test C API
+// Tests are written in C++ to use Google Test, but they test the C API.
+
+#include <cstring>
+
+#include "fmt/c.h"
+#include "gtest/gtest.h"
+
+TEST(CAPITest, IntegerFormatting) {
+  auto args = fmt_create_arg_store();
+  ASSERT_NE(nullptr, args);
+
+  fmt_add_int(args, 42);
+  char* result = fmt_format("The answer is {}", args);
+  EXPECT_STREQ("The answer is 42", result);
+  free(result);
+
+  fmt_free_arg_store(args);
+}
+
+TEST(CAPITest, MultipleArguments) {
+  auto args = fmt_create_arg_store();
+  ASSERT_NE(nullptr, args);
+
+  fmt_add_int(args, 1);
+  fmt_add_int(args, 2);
+  fmt_add_int(args, 3);
+  char* result = fmt_format("{}, {}, {}", args);
+  EXPECT_STREQ("1, 2, 3", result);
+  free(result);
+
+  fmt_free_arg_store(args);
+}
+
+TEST(CAPITest, StringFormatting) {
+  auto args = fmt_create_arg_store();
+  ASSERT_NE(nullptr, args);
+
+  fmt_add_string(args, "world");
+  char* result = fmt_format("Hello, {}!", args);
+  EXPECT_STREQ("Hello, world!", result);
+  free(result);
+
+  fmt_free_arg_store(args);
+}
+
+TEST(CAPITest, FloatFormatting) {
+  auto args = fmt_create_arg_store();
+  ASSERT_NE(nullptr, args);
+
+  fmt_add_double(args, 3.14);
+  char* result = fmt_format("Value: {}", args);
+  EXPECT_STREQ("Value: 3.14", result);
+  free(result);
+
+  fmt_free_arg_store(args);
+}
+
+TEST(CAPITest, BoolFormatting) {
+  auto args = fmt_create_arg_store();
+  ASSERT_NE(nullptr, args);
+
+  fmt_add_bool(args, 1);
+  fmt_add_bool(args, 0);
+  char* result = fmt_format("{} {}", args);
+  EXPECT_STREQ("true false", result);
+  free(result);
+
+  fmt_free_arg_store(args);
+}
+
+TEST(CAPITest, MixedTypes) {
+  auto args = fmt_create_arg_store();
+  ASSERT_NE(nullptr, args);
+
+  fmt_add_string(args, "answer");
+  fmt_add_int(args, 42);
+  fmt_add_double(args, 3.14);
+  char* result = fmt_format("{}: {} ({})", args);
+  EXPECT_STREQ("answer: 42 (3.14)", result);
+  free(result);
+
+  fmt_free_arg_store(args);
+}
+
+TEST(CAPITest, StringWithLength) {
+  auto args = fmt_create_arg_store();
+  ASSERT_NE(nullptr, args);
+
+  fmt_add_string_len(args, "hello world", 5);
+  char* result = fmt_format("{}", args);
+  EXPECT_STREQ("hello", result);
+  free(result);
+
+  fmt_free_arg_store(args);
+}
+
+TEST(CAPITest, PointerFormatting) {
+  auto args = fmt_create_arg_store();
+  ASSERT_NE(nullptr, args);
+
+  int x = 42;
+  fmt_add_pointer(args, &x);
+  char* result = fmt_format("Pointer: {}", args);
+  EXPECT_TRUE(std::strstr(result, "Pointer: 0x") != nullptr);
+  free(result);
+
+  fmt_free_arg_store(args);
+}
+
+TEST(CAPITest, CharFormatting) {
+  auto args = fmt_create_arg_store();
+  ASSERT_NE(nullptr, args);
+
+  fmt_add_char(args, 'A');
+  char* result = fmt_format("Char: {}", args);
+  EXPECT_STREQ("Char: A", result);
+  free(result);
+
+  fmt_free_arg_store(args);
+}
+
+TEST(CAPITest, LongTypes) {
+  auto args = fmt_create_arg_store();
+  ASSERT_NE(nullptr, args);
+
+  fmt_add_long(args, 1000000L);
+  fmt_add_ulong(args, 2000000UL);
+  char* result = fmt_format("{} {}", args);
+  EXPECT_STREQ("1000000 2000000", result);
+  free(result);
+
+  fmt_free_arg_store(args);
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


