# SWE-Bench Detail Report

Generated: 2026-03-11 11:41 UTC

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
| **baseline** | 370.7s | 35 | 1326942 | 15506 | $1.1025 |
| **with-lumen** | 359.1s | 29 | 1019261 | 22056 | $1.0141 |

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
index 0000000..958230d
--- /dev/null
+++ b/include/fmt/c.h
@@ -0,0 +1,120 @@
+// Formatting library for C++ - C API
+//
+// Copyright (c) 2012 - present, Victor Zverovich
+// All rights reserved.
+//
+// For the license information refer to format.h.
+
+/**
+ * A C API for the {fmt} library that provides type-safe formatting without
+ * requiring a C++ compiler. It builds on the type-erased `fmt::vformat`
+ * interface and `fmt::dynamic_format_arg_store`.
+ *
+ * Unlike printf, this API is fully type-safe: type information is encoded in
+ * the argument store, not in the format string itself.
+ *
+ * Example:
+ *
+ *     fmt_arg_store* args = fmt_arg_store_create();
+ *     fmt_arg_store_push_int(args, 42);
+ *     fmt_arg_store_push_cstring(args, "world");
+ *     char* result = fmt_vformat("answer={} name={}", args);
+ *     // result == "answer=42 name=world"
+ *     fmt_free(result);
+ *     fmt_arg_store_destroy(args);
+ */
+
+#ifndef FMT_C_H_
+#define FMT_C_H_
+
+#include <stddef.h>
+
+// Export macro for shared library builds.
+#ifndef FMT_C_API
+#  if defined(_WIN32) && !defined(FMT_HEADER_ONLY)
+#    if defined(FMT_LIB_EXPORT)
+#      define FMT_C_API __declspec(dllexport)
+#    elif defined(FMT_SHARED)
+#      define FMT_C_API __declspec(dllimport)
+#    else
+#      define FMT_C_API
+#    endif
+#  elif defined(FMT_LIB_EXPORT) || defined(FMT_SHARED)
+#    define FMT_C_API __attribute__((visibility("default")))
+#  else
+#    define FMT_C_API
+#  endif
+#endif
+
+#ifdef __cplusplus
+extern "C" {
+#endif
+
+/**
+ * An opaque handle to a dynamic argument store. Use `fmt_arg_store_create` to
+ * create one and `fmt_arg_store_destroy` to release it.
+ */
+typedef struct fmt_arg_store_s fmt_arg_store;
+
+/** Creates a new, empty argument store. Returns NULL on allocation failure. */
+FMT_C_API fmt_arg_store* fmt_arg_store_create(void);
+
+/** Destroys an argument store created by `fmt_arg_store_create`. */
+FMT_C_API void fmt_arg_store_destroy(fmt_arg_store* store);
+
+/**
+ * Pushes a typed value into the argument store.
+ * For `push_cstring`, the pointed-to string must remain valid until
+ * `fmt_vformat` is called. For `push_string`, the string is copied.
+ */
+FMT_C_API void fmt_arg_store_push_int(fmt_arg_store* store, int value);
+FMT_C_API void fmt_arg_store_push_uint(fmt_arg_store* store,
+                                       unsigned int value);
+FMT_C_API void fmt_arg_store_push_long_long(fmt_arg_store* store,
+                                            long long value);
+FMT_C_API void fmt_arg_store_push_ulong_long(fmt_arg_store* store,
+                                             unsigned long long value);
+FMT_C_API void fmt_arg_store_push_float(fmt_arg_store* store, float value);
+FMT_C_API void fmt_arg_store_push_double(fmt_arg_store* store, double value);
+FMT_C_API void fmt_arg_store_push_long_double(fmt_arg_store* store,
+                                              long double value);
+/** Pushes a boolean value (0 = false, non-zero = true). */
+FMT_C_API void fmt_arg_store_push_bool(fmt_arg_store* store, int value);
+FMT_C_API void fmt_arg_store_push_char(fmt_arg_store* store, char value);
+/**
+ * Pushes a null-terminated C string. The pointer must remain valid until
+ * `fmt_vformat` is called.
+ */
+FMT_C_API void fmt_arg_store_push_cstring(fmt_arg_store* store,
+                                          const char* value);
+/**
+ * Pushes a string of the given length. The string is copied into the store, so
+ * the pointer does not need to remain valid after this call.
+ */
+FMT_C_API void fmt_arg_store_push_string(fmt_arg_store* store,
+                                         const char* value, size_t size);
+/** Pushes a pointer value (formatted as a hexadecimal address). */
+FMT_C_API void fmt_arg_store_push_pointer(fmt_arg_store* store,
+                                          const void* value);
+
+/**
+ * Formats a string according to `fmt` using the arguments in `store`.
+ *
+ * Returns a null-terminated heap-allocated string on success, or NULL if
+ * formatting failed (e.g. invalid format string) or memory allocation failed.
+ * The caller must release the result with `fmt_free`.
+ *
+ * The `store` argument may be NULL (equivalent to an empty store).
+ */
+FMT_C_API char* fmt_vformat(const char* fmt, fmt_arg_store* store);
+
+/**
+ * Releases memory returned by `fmt_vformat`. Passing NULL is a no-op.
+ */
+FMT_C_API void fmt_free(void* ptr);
+
+#ifdef __cplusplus
+}  // extern "C"
+#endif
+
+#endif  // FMT_C_H_
diff --git a/src/fmt-c.cc b/src/fmt-c.cc
new file mode 100644
index 0000000..9655333
--- /dev/null
+++ b/src/fmt-c.cc
@@ -0,0 +1,117 @@
+// Formatting library for C++ - C API implementation
+//
+// Copyright (c) 2012 - present, Victor Zverovich
+// All rights reserved.
+//
+// For the license information refer to format.h.
+
+#include "fmt/c.h"
+
+#include <cstdlib>
+#include <cstring>
+#include <new>
+
+#include "fmt/args.h"
+
+struct fmt_arg_store_s {
+  fmt::dynamic_format_arg_store<fmt::format_context> store;
+};
+
+extern "C" {
+
+FMT_C_API fmt_arg_store* fmt_arg_store_create(void) {
+  FMT_TRY { return new (std::nothrow) fmt_arg_store_s; }
+  FMT_CATCH(...) { return nullptr; }
+}
+
+FMT_C_API void fmt_arg_store_destroy(fmt_arg_store* store) { delete store; }
+
+FMT_C_API void fmt_arg_store_push_int(fmt_arg_store* store, int value) {
+  if (store) FMT_TRY { store->store.push_back(value); }
+  FMT_CATCH(...) {}
+}
+
+FMT_C_API void fmt_arg_store_push_uint(fmt_arg_store* store,
+                                       unsigned int value) {
+  if (store) FMT_TRY { store->store.push_back(value); }
+  FMT_CATCH(...) {}
+}
+
+FMT_C_API void fmt_arg_store_push_long_long(fmt_arg_store* store,
+                                            long long value) {
+  if (store) FMT_TRY { store->store.push_back(value); }
+  FMT_CATCH(...) {}
+}
+
+FMT_C_API void fmt_arg_store_push_ulong_long(fmt_arg_store* store,
+                                             unsigned long long value) {
+  if (store) FMT_TRY { store->store.push_back(value); }
+  FMT_CATCH(...) {}
+}
+
+FMT_C_API void fmt_arg_store_push_float(fmt_arg_store* store, float value) {
+  if (store) FMT_TRY { store->store.push_back(value); }
+  FMT_CATCH(...) {}
+}
+
+FMT_C_API void fmt_arg_store_push_double(fmt_arg_store* store, double value) {
+  if (store) FMT_TRY { store->store.push_back(value); }
+  FMT_CATCH(...) {}
+}
+
+FMT_C_API void fmt_arg_store_push_long_double(fmt_arg_store* store,
+                                              long double value) {
+  if (store) FMT_TRY { store->store.push_back(value); }
+  FMT_CATCH(...) {}
+}
+
+FMT_C_API void fmt_arg_store_push_bool(fmt_arg_store* store, int value) {
+  if (store) FMT_TRY { store->store.push_back(value != 0); }
+  FMT_CATCH(...) {}
+}
+
+FMT_C_API void fmt_arg_store_push_char(fmt_arg_store* store, char value) {
+  if (store) FMT_TRY { store->store.push_back(value); }
+  FMT_CATCH(...) {}
+}
+
+FMT_C_API void fmt_arg_store_push_cstring(fmt_arg_store* store,
+                                          const char* value) {
+  if (store) FMT_TRY { store->store.push_back(value); }
+  FMT_CATCH(...) {}
+}
+
+FMT_C_API void fmt_arg_store_push_string(fmt_arg_store* store,
+                                         const char* value, size_t size) {
+  if (store && value)
+    FMT_TRY { store->store.push_back(std::string(value, size)); }
+  FMT_CATCH(...) {}
+}
+
+FMT_C_API void fmt_arg_store_push_pointer(fmt_arg_store* store,
+                                          const void* value) {
+  if (store) FMT_TRY { store->store.push_back(value); }
+  FMT_CATCH(...) {}
+}
+
+FMT_C_API char* fmt_vformat(const char* format, fmt_arg_store* store) {
+  if (!format) return nullptr;
+  FMT_TRY {
+    std::string result;
+    if (store) {
+      result = fmt::vformat(format, store->store);
+    } else {
+      fmt::dynamic_format_arg_store<fmt::format_context> empty;
+      result = fmt::vformat(format, empty);
+    }
+    char* buf = static_cast<char*>(std::malloc(result.size() + 1));
+    if (!buf) return nullptr;
+    std::memcpy(buf, result.c_str(), result.size() + 1);
+    return buf;
+  }
+  FMT_CATCH(...) { return nullptr; }
+}
+
+FMT_C_API void fmt_free(void* ptr) { std::free(ptr); }
+
+}  // extern "C"
diff --git a/test/CMakeLists.txt b/test/CMakeLists.txt
index 3f7d24a..eb0654d 100644
--- a/test/CMakeLists.txt
+++ b/test/CMakeLists.txt
@@ -46,6 +46,7 @@ if (FMT_MODULE)
 endif ()
 
 add_fmt_test(args-test)
+add_fmt_test(c-api-test)
 add_fmt_test(base-test)
 add_fmt_test(assert-test)
 add_fmt_test(chrono-test)
diff --git a/test/c-api-test.cc b/test/c-api-test.cc
new file mode 100644
index 0000000..fca20d2
--- /dev/null
+++ b/test/c-api-test.cc
@@ -0,0 +1,192 @@
+// Formatting library for C++ - C API tests
+//
+// Copyright (c) 2012 - present, Victor Zverovich
+// All rights reserved.
+//
+// For the license information refer to format.h.
+
+#include "fmt/c.h"
+
+#include <string.h>
+
+#include "gtest/gtest.h"
+
+TEST(c_api_test, empty_format) {
+  fmt_arg_store* args = fmt_arg_store_create();
+  char* result = fmt_vformat("no args", args);
+  ASSERT_NE(result, nullptr);
+  EXPECT_STREQ(result, "no args");
+  fmt_free(result);
+  fmt_arg_store_destroy(args);
+}
+
+TEST(c_api_test, null_store) {
+  char* result = fmt_vformat("hello", nullptr);
+  ASSERT_NE(result, nullptr);
+  EXPECT_STREQ(result, "hello");
+  fmt_free(result);
+}
+
+TEST(c_api_test, push_int) {
+  fmt_arg_store* args = fmt_arg_store_create();
+  fmt_arg_store_push_int(args, 42);
+  char* result = fmt_vformat("{}", args);
+  ASSERT_NE(result, nullptr);
+  EXPECT_STREQ(result, "42");
+  fmt_free(result);
+  fmt_arg_store_destroy(args);
+}
+
+TEST(c_api_test, push_uint) {
+  fmt_arg_store* args = fmt_arg_store_create();
+  fmt_arg_store_push_uint(args, 42u);
+  char* result = fmt_vformat("{}", args);
+  ASSERT_NE(result, nullptr);
+  EXPECT_STREQ(result, "42");
+  fmt_free(result);
+  fmt_arg_store_destroy(args);
+}
+
+TEST(c_api_test, push_long_long) {
+  fmt_arg_store* args = fmt_arg_store_create();
+  fmt_arg_store_push_long_long(args, -9000000000LL);
+  char* result = fmt_vformat("{}", args);
+  ASSERT_NE(result, nullptr);
+  EXPECT_STREQ(result, "-9000000000");
+  fmt_free(result);
+  fmt_arg_store_destroy(args);
+}
+
+TEST(c_api_test, push_ulong_long) {
+  fmt_arg_store* args = fmt_arg_store_create();
+  fmt_arg_store_push_ulong_long(args, 9000000000ULL);
+  char* result = fmt_vformat("{}", args);
+  ASSERT_NE(result, nullptr);
+  EXPECT_STREQ(result, "9000000000");
+  fmt_free(result);
+  fmt_arg_store_destroy(args);
+}
+
+TEST(c_api_test, push_double) {
+  fmt_arg_store* args = fmt_arg_store_create();
+  fmt_arg_store_push_double(args, 1.5);
+  char* result = fmt_vformat("{}", args);
+  ASSERT_NE(result, nullptr);
+  EXPECT_STREQ(result, "1.5");
+  fmt_free(result);
+  fmt_arg_store_destroy(args);
+}
+
+TEST(c_api_test, push_float) {
+  fmt_arg_store* args = fmt_arg_store_create();
+  fmt_arg_store_push_float(args, 1.5f);
+  char* result = fmt_vformat("{}", args);
+  ASSERT_NE(result, nullptr);
+  EXPECT_STREQ(result, "1.5");
+  fmt_free(result);
+  fmt_arg_store_destroy(args);
+}
+
+TEST(c_api_test, push_bool_true) {
+  fmt_arg_store* args = fmt_arg_store_create();
+  fmt_arg_store_push_bool(args, 1);
+  char* result = fmt_vformat("{}", args);
+  ASSERT_NE(result, nullptr);
+  EXPECT_STREQ(result, "true");
+  fmt_free(result);
+  fmt_arg_store_destroy(args);
+}
+
+TEST(c_api_test, push_bool_false) {
+  fmt_arg_store* args = fmt_arg_store_create();
+  fmt_arg_store_push_bool(args, 0);
+  char* result = fmt_vformat("{}", args);
+  ASSERT_NE(result, nullptr);
+  EXPECT_STREQ(result, "false");
+  fmt_free(result);
+  fmt_arg_store_destroy(args);
+}
+
+TEST(c_api_test, push_char) {
+  fmt_arg_store* args = fmt_arg_store_create();
+  fmt_arg_store_push_char(args, 'A');
+  char* result = fmt_vformat("{}", args);
+  ASSERT_NE(result, nullptr);
+  EXPECT_STREQ(result, "A");
+  fmt_free(result);
+  fmt_arg_store_destroy(args);
+}
+
+TEST(c_api_test, push_cstring) {
+  fmt_arg_store* args = fmt_arg_store_create();
+  fmt_arg_store_push_cstring(args, "hello");
+  char* result = fmt_vformat("{}", args);
+  ASSERT_NE(result, nullptr);
+  EXPECT_STREQ(result, "hello");
+  fmt_free(result);
+  fmt_arg_store_destroy(args);
+}
+
+TEST(c_api_test, push_string_with_length) {
+  fmt_arg_store* args = fmt_arg_store_create();
+  const char* str = "hello world";
+  fmt_arg_store_push_string(args, str, 5);  // only "hello"
+  char* result = fmt_vformat("{}", args);
+  ASSERT_NE(result, nullptr);
+  EXPECT_STREQ(result, "hello");
+  fmt_free(result);
+  fmt_arg_store_destroy(args);
+}
+
+TEST(c_api_test, push_pointer) {
+  fmt_arg_store* args = fmt_arg_store_create();
+  fmt_arg_store_push_pointer(args, nullptr);
+  char* result = fmt_vformat("{}", args);
+  ASSERT_NE(result, nullptr);
+  EXPECT_STREQ(result, "0x0");
+  fmt_free(result);
+  fmt_arg_store_destroy(args);
+}
+
+TEST(c_api_test, multiple_args) {
+  fmt_arg_store* args = fmt_arg_store_create();
+  fmt_arg_store_push_int(args, 42);
+  fmt_arg_store_push_cstring(args, "world");
+  fmt_arg_store_push_double(args, 1.5);
+  char* result = fmt_vformat("{} {} {}", args);
+  ASSERT_NE(result, nullptr);
+  EXPECT_STREQ(result, "42 world 1.5");
+  fmt_free(result);
+  fmt_arg_store_destroy(args);
+}
+
+TEST(c_api_test, format_spec) {
+  fmt_arg_store* args = fmt_arg_store_create();
+  fmt_arg_store_push_int(args, 255);
+  char* result = fmt_vformat("{:x}", args);
+  ASSERT_NE(result, nullptr);
+  EXPECT_STREQ(result, "ff");
+  fmt_free(result);
+  fmt_arg_store_destroy(args);
+}
+
+TEST(c_api_test, invalid_format_returns_null) {
+  fmt_arg_store* args = fmt_arg_store_create();
+  // Too many placeholders for the number of arguments
+  char* result = fmt_vformat("{} {}", args);
+  EXPECT_EQ(result, nullptr);
+  fmt_arg_store_destroy(args);
+}
+
+TEST(c_api_test, push_string_is_copied) {
+  fmt_arg_store* args = fmt_arg_store_create();
+  char buf[] = "original";
+  fmt_arg_store_push_string(args, buf, strlen(buf));
+  // Mutate buf after pushing; result should still show "original"
+  buf[0] = 'X';
+  char* result = fmt_vformat("{}", args);
+  ASSERT_NE(result, nullptr);
+  EXPECT_STREQ(result, "original");
+  fmt_free(result);
+  fmt_arg_store_destroy(args);
+}
```

### with-lumen

**Rating: Good**

```diff
diff --git a/CMakeLists.txt b/CMakeLists.txt
index 53afc51..e44394e 100644
--- a/CMakeLists.txt
+++ b/CMakeLists.txt
@@ -299,10 +299,10 @@ endfunction()
 
 # Define the fmt library, its includes and the needed defines.
 set(FMT_HEADERS)
-add_headers(FMT_HEADERS args.h base.h chrono.h color.h compile.h core.h format.h
-                        format-inl.h os.h ostream.h printf.h ranges.h std.h
-                        xchar.h)
-set(FMT_SOURCES src/format.cc)
+add_headers(FMT_HEADERS args.h base.h c_api.h chrono.h color.h compile.h core.h
+                        format.h format-inl.h os.h ostream.h printf.h ranges.h
+                        std.h xchar.h)
+set(FMT_SOURCES src/format.cc src/c_api.cc)
 
 add_module_library(fmt src/fmt.cc FALLBACK
                    ${FMT_SOURCES} ${FMT_HEADERS} README.md ChangeLog.md
diff --git a/include/fmt/c_api.h b/include/fmt/c_api.h
new file mode 100644
index 0000000..21c5ec9
--- /dev/null
+++ b/include/fmt/c_api.h
@@ -0,0 +1,140 @@
+// Formatting library for C++ - C API
+//
+// Copyright (c) 2012 - present, Victor Zverovich
+// All rights reserved.
+//
+// For the license information refer to format.h.
+
+#ifndef FMT_C_API_H_
+#define FMT_C_API_H_
+
+#include <stddef.h>
+
+#if !defined(FMT_HEADER_ONLY) && defined(_WIN32)
+#  if defined(FMT_LIB_EXPORT)
+#    define FMT_C_API __declspec(dllexport)
+#  elif defined(FMT_SHARED)
+#    define FMT_C_API __declspec(dllimport)
+#  endif
+#elif defined(FMT_LIB_EXPORT) || defined(FMT_SHARED)
+#  define FMT_C_API __attribute__((visibility("default")))
+#endif
+#ifndef FMT_C_API
+#  define FMT_C_API
+#endif
+
+#ifdef __cplusplus
+extern "C" {
+#endif
+
+/**
+ * The type of a formatting argument passed to the C API.
+ *
+ * Unlike printf, the type is encoded explicitly in the argument struct rather
+ * than in the format string, which makes the interface type-safe even for
+ * runtime format strings.
+ */
+enum fmt_type {
+  FMT_INT,
+  FMT_UINT,
+  FMT_LONG_LONG,
+  FMT_ULONG_LONG,
+  FMT_BOOL,
+  FMT_CHAR,
+  FMT_FLOAT,
+  FMT_DOUBLE,
+  FMT_LONG_DOUBLE,
+  FMT_CSTRING,
+  FMT_STRING,
+  FMT_POINTER
+};
+
+/** A string argument with explicit length. */
+struct fmt_string_value {
+  const char* data;
+  size_t size;
+};
+
+/**
+ * A type-tagged formatting argument.
+ *
+ * Unlike printf format specifiers, the type is stored in the `type` field
+ * rather than encoded in the format string. This makes the API type-safe even
+ * for runtime format strings and avoids undefined behavior from type mismatches.
+ */
+struct fmt_arg {
+  enum fmt_type type;
+  union {
+    int int_value;
+    unsigned uint_value;
+    long long long_long_value;
+    unsigned long long ulong_long_value;
+    int bool_value; /**< 0 for false, non-zero for true */
+    char char_value;
+    float float_value;
+    double double_value;
+    long double long_double_value;
+    const char* cstring_value;
+    struct fmt_string_value string_value;
+    const void* pointer_value;
+  } value;
+};
+
+/**
+ * Formats arguments according to the format string and writes the result into
+ * the provided buffer.
+ *
+ * Behaves like snprintf: if `buf` is not null and `buf_size` is greater than
+ * zero, writes up to `buf_size - 1` characters followed by a null terminator.
+ * Returns the total number of characters that would be written if the buffer
+ * were large enough, not counting the null terminator, or -1 on error.
+ *
+ * **Example**:
+ *
+ *     char buf[64];
+ *     struct fmt_arg args[] = {FMT_ARG_INT(42), FMT_ARG_CSTRING("world")};
+ *     fmt_format_to(buf, sizeof(buf), "{} {}", args, 2);
+ *     // buf == "42 world"
+ */
+FMT_C_API int fmt_format_to(char* buf, size_t buf_size, const char* fmt,
+                             const struct fmt_arg* args, int num_args);
+
+/**
+ * Formats arguments according to the format string and writes the result to
+ * the standard output. Returns the number of characters written, or -1 on
+ * error.
+ */
+FMT_C_API int fmt_print(const char* fmt, const struct fmt_arg* args,
+                        int num_args);
+
+#ifdef __cplusplus
+}
+#endif
+
+/* Convenience macros for constructing fmt_arg values (C99). */
+#define FMT_ARG_INT(v) \
+  {FMT_INT, {.int_value = (v)}}
+#define FMT_ARG_UINT(v) \
+  {FMT_UINT, {.uint_value = (v)}}
+#define FMT_ARG_LONG_LONG(v) \
+  {FMT_LONG_LONG, {.long_long_value = (v)}}
+#define FMT_ARG_ULONG_LONG(v) \
+  {FMT_ULONG_LONG, {.ulong_long_value = (v)}}
+#define FMT_ARG_BOOL(v) \
+  {FMT_BOOL, {.bool_value = (int)(v)}}
+#define FMT_ARG_CHAR(v) \
+  {FMT_CHAR, {.char_value = (v)}}
+#define FMT_ARG_FLOAT(v) \
+  {FMT_FLOAT, {.float_value = (v)}}
+#define FMT_ARG_DOUBLE(v) \
+  {FMT_DOUBLE, {.double_value = (v)}}
+#define FMT_ARG_LONG_DOUBLE(v) \
+  {FMT_LONG_DOUBLE, {.long_double_value = (v)}}
+#define FMT_ARG_CSTRING(v) \
+  {FMT_CSTRING, {.cstring_value = (v)}}
+#define FMT_ARG_STRING(d, s) \
+  {FMT_STRING, {.string_value = {(d), (s)}}}
+#define FMT_ARG_POINTER(v) \
+  {FMT_POINTER, {.pointer_value = (v)}}
+
+#endif  // FMT_C_API_H_
diff --git a/src/c_api.cc b/src/c_api.cc
new file mode 100644
index 0000000..f2713d5
--- /dev/null
+++ b/src/c_api.cc
@@ -0,0 +1,90 @@
+// Formatting library for C++ - C API implementation
+//
+// Copyright (c) 2012 - present, Victor Zverovich
+// All rights reserved.
+//
+// For the license information refer to format.h.
+
+#include "fmt/c_api.h"
+
+#include <cstring>  // memcpy
+#include <vector>
+
+#define FMT_HEADER_ONLY
+#include "fmt/format.h"
+
+namespace {
+
+auto to_fmt_arg(const fmt_arg& arg) -> fmt::basic_format_arg<fmt::context> {
+  switch (arg.type) {
+  case FMT_INT:
+    return fmt::basic_format_arg<fmt::context>(arg.value.int_value);
+  case FMT_UINT:
+    return fmt::basic_format_arg<fmt::context>(arg.value.uint_value);
+  case FMT_LONG_LONG:
+    return fmt::basic_format_arg<fmt::context>(arg.value.long_long_value);
+  case FMT_ULONG_LONG:
+    return fmt::basic_format_arg<fmt::context>(arg.value.ulong_long_value);
+  case FMT_BOOL:
+    return fmt::basic_format_arg<fmt::context>(
+        static_cast<bool>(arg.value.bool_value));
+  case FMT_CHAR:
+    return fmt::basic_format_arg<fmt::context>(arg.value.char_value);
+  case FMT_FLOAT:
+    return fmt::basic_format_arg<fmt::context>(arg.value.float_value);
+  case FMT_DOUBLE:
+    return fmt::basic_format_arg<fmt::context>(arg.value.double_value);
+  case FMT_LONG_DOUBLE:
+    return fmt::basic_format_arg<fmt::context>(arg.value.long_double_value);
+  case FMT_CSTRING:
+    return fmt::basic_format_arg<fmt::context>(arg.value.cstring_value);
+  case FMT_STRING:
+    return fmt::basic_format_arg<fmt::context>(fmt::string_view(
+        arg.value.string_value.data, arg.value.string_value.size));
+  case FMT_POINTER:
+    return fmt::basic_format_arg<fmt::context>(arg.value.pointer_value);
+  }
+  return fmt::basic_format_arg<fmt::context>();
+}
+
+}  // namespace
+
+extern "C" {
+
+int fmt_format_to(char* buf, size_t buf_size, const char* fmt_str,
+                  const fmt_arg* args, int num_args) {
+  FMT_TRY {
+    std::vector<fmt::basic_format_arg<fmt::context>> cpp_args;
+    cpp_args.reserve(static_cast<size_t>(num_args));
+    for (int i = 0; i < num_args; ++i) cpp_args.push_back(to_fmt_arg(args[i]));
+    auto fa =
+        fmt::basic_format_args<fmt::context>(cpp_args.data(), num_args);
+    auto result = fmt::memory_buffer();
+    fmt::detail::vformat_to(result, fmt::string_view(fmt_str), fa);
+    if (buf && buf_size > 0) {
+      size_t copy_size =
+          result.size() < buf_size - 1 ? result.size() : buf_size - 1;
+      std::memcpy(buf, result.data(), copy_size);
+      buf[copy_size] = '\0';
+    }
+    return static_cast<int>(result.size());
+  }
+  FMT_CATCH(...) { return -1; }
+}
+
+int fmt_print(const char* fmt_str, const fmt_arg* args, int num_args) {
+  FMT_TRY {
+    std::vector<fmt::basic_format_arg<fmt::context>> cpp_args;
+    cpp_args.reserve(static_cast<size_t>(num_args));
+    for (int i = 0; i < num_args; ++i) cpp_args.push_back(to_fmt_arg(args[i]));
+    auto fa =
+        fmt::basic_format_args<fmt::context>(cpp_args.data(), num_args);
+    auto result = fmt::memory_buffer();
+    fmt::detail::vformat_to(result, fmt::string_view(fmt_str), fa);
+    std::fwrite(result.data(), 1, result.size(), stdout);
+    return static_cast<int>(result.size());
+  }
+  FMT_CATCH(...) { return -1; }
+}
+
+}  // extern "C"
diff --git a/test/CMakeLists.txt b/test/CMakeLists.txt
index 3f7d24a..eb0654d 100644
--- a/test/CMakeLists.txt
+++ b/test/CMakeLists.txt
@@ -46,6 +46,7 @@ if (FMT_MODULE)
 endif ()
 
 add_fmt_test(args-test)
+add_fmt_test(c-api-test)
 add_fmt_test(base-test)
 add_fmt_test(assert-test)
 add_fmt_test(chrono-test)
diff --git a/test/c-api-test.cc b/test/c-api-test.cc
new file mode 100644
index 0000000..b637832
--- /dev/null
+++ b/test/c-api-test.cc
@@ -0,0 +1,140 @@
+// Formatting library for C++ - C API tests
+//
+// Copyright (c) 2012 - present, Victor Zverovich
+// All rights reserved.
+//
+// For the license information refer to format.h.
+
+#include "fmt/c_api.h"
+
+#include <string.h>
+
+#include "gtest/gtest.h"
+
+TEST(c_api_test, no_args) {
+  char buf[64];
+  int n = fmt_format_to(buf, sizeof(buf), "hello", nullptr, 0);
+  EXPECT_EQ(5, n);
+  EXPECT_STREQ("hello", buf);
+}
+
+TEST(c_api_test, int_arg) {
+  char buf[64];
+  struct fmt_arg args[] = {FMT_ARG_INT(42)};
+  int n = fmt_format_to(buf, sizeof(buf), "{}", args, 1);
+  EXPECT_EQ(2, n);
+  EXPECT_STREQ("42", buf);
+}
+
+TEST(c_api_test, multiple_args) {
+  char buf[64];
+  struct fmt_arg args[] = {FMT_ARG_INT(1), FMT_ARG_CSTRING("foo"),
+                           FMT_ARG_DOUBLE(3.14)};
+  fmt_format_to(buf, sizeof(buf), "{} {} {:.2f}", args, 3);
+  EXPECT_STREQ("1 foo 3.14", buf);
+}
+
+TEST(c_api_test, uint_arg) {
+  char buf[32];
+  struct fmt_arg args[] = {FMT_ARG_UINT(42u)};
+  fmt_format_to(buf, sizeof(buf), "{}", args, 1);
+  EXPECT_STREQ("42", buf);
+}
+
+TEST(c_api_test, long_long_arg) {
+  char buf[32];
+  struct fmt_arg args[] = {FMT_ARG_LONG_LONG(1234567890123LL)};
+  fmt_format_to(buf, sizeof(buf), "{}", args, 1);
+  EXPECT_STREQ("1234567890123", buf);
+}
+
+TEST(c_api_test, ulong_long_arg) {
+  char buf[32];
+  struct fmt_arg args[] = {FMT_ARG_ULONG_LONG(9876543210ULL)};
+  fmt_format_to(buf, sizeof(buf), "{}", args, 1);
+  EXPECT_STREQ("9876543210", buf);
+}
+
+TEST(c_api_test, bool_arg) {
+  char buf[16];
+  struct fmt_arg args[] = {FMT_ARG_BOOL(1)};
+  fmt_format_to(buf, sizeof(buf), "{}", args, 1);
+  EXPECT_STREQ("true", buf);
+
+  struct fmt_arg args2[] = {FMT_ARG_BOOL(0)};
+  fmt_format_to(buf, sizeof(buf), "{}", args2, 1);
+  EXPECT_STREQ("false", buf);
+}
+
+TEST(c_api_test, char_arg) {
+  char buf[16];
+  struct fmt_arg args[] = {FMT_ARG_CHAR('A')};
+  fmt_format_to(buf, sizeof(buf), "{}", args, 1);
+  EXPECT_STREQ("A", buf);
+}
+
+TEST(c_api_test, float_arg) {
+  char buf[32];
+  struct fmt_arg args[] = {FMT_ARG_FLOAT(1.5f)};
+  fmt_format_to(buf, sizeof(buf), "{}", args, 1);
+  EXPECT_STREQ("1.5", buf);
+}
+
+TEST(c_api_test, double_arg) {
+  char buf[32];
+  struct fmt_arg args[] = {FMT_ARG_DOUBLE(2.5)};
+  fmt_format_to(buf, sizeof(buf), "{}", args, 1);
+  EXPECT_STREQ("2.5", buf);
+}
+
+TEST(c_api_test, cstring_arg) {
+  char buf[32];
+  struct fmt_arg args[] = {FMT_ARG_CSTRING("hello")};
+  fmt_format_to(buf, sizeof(buf), "{}", args, 1);
+  EXPECT_STREQ("hello", buf);
+}
+
+TEST(c_api_test, string_arg) {
+  char buf[32];
+  const char* s = "hello world";
+  struct fmt_arg args[] = {FMT_ARG_STRING(s, 5)};
+  fmt_format_to(buf, sizeof(buf), "{}", args, 1);
+  EXPECT_STREQ("hello", buf);
+}
+
+TEST(c_api_test, pointer_arg) {
+  char buf[32];
+  void* p = nullptr;
+  struct fmt_arg args[] = {FMT_ARG_POINTER(p)};
+  fmt_format_to(buf, sizeof(buf), "{}", args, 1);
+  EXPECT_STREQ("0x0", buf);
+}
+
+TEST(c_api_test, returns_required_size) {
+  // buf_size is too small; returns what would be needed
+  char buf[4];
+  struct fmt_arg args[] = {FMT_ARG_INT(12345)};
+  int n = fmt_format_to(buf, sizeof(buf), "{}", args, 1);
+  EXPECT_EQ(5, n);
+  EXPECT_STREQ("123", buf);  // truncated but null-terminated
+}
+
+TEST(c_api_test, null_buf_returns_size) {
+  struct fmt_arg args[] = {FMT_ARG_CSTRING("hello")};
+  int n = fmt_format_to(nullptr, 0, "{}", args, 1);
+  EXPECT_EQ(5, n);
+}
+
+TEST(c_api_test, invalid_format_returns_minus_one) {
+  char buf[32];
+  struct fmt_arg args[] = {FMT_ARG_INT(1)};
+  int n = fmt_format_to(buf, sizeof(buf), "{", args, 1);
+  EXPECT_EQ(-1, n);
+}
+
+TEST(c_api_test, format_spec) {
+  char buf[32];
+  struct fmt_arg args[] = {FMT_ARG_INT(255)};
+  fmt_format_to(buf, sizeof(buf), "{:#x}", args, 1);
+  EXPECT_STREQ("0xff", buf);
+}
```


