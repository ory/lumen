# SWE-Bench Detail Report

Generated: 2026-03-10 21:50 UTC

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
| **baseline** | 474.0s | 986 | 1104736 | 22610 | $1.1736 |
| **with-lumen** | 409.7s | 55 | 2309230 | 20087 | $1.4039 |

### baseline

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
index 0000000..0fad581
--- /dev/null
+++ b/include/fmt/c.h
@@ -0,0 +1,113 @@
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
+#include <stddef.h>  // size_t
+
+// Visibility macros for shared library support.
+#if defined(_WIN32)
+#  if defined(FMT_LIB_EXPORT)
+#    define FMT_C_API __declspec(dllexport)
+#  elif defined(FMT_SHARED)
+#    define FMT_C_API __declspec(dllimport)
+#  else
+#    define FMT_C_API
+#  endif
+#elif defined(FMT_LIB_EXPORT) || defined(FMT_SHARED)
+#  if defined(__GNUC__) || defined(__clang__)
+#    define FMT_C_API __attribute__((visibility("default")))
+#  else
+#    define FMT_C_API
+#  endif
+#else
+#  define FMT_C_API
+#endif
+
+#ifdef __cplusplus
+extern "C" {
+#endif
+
+/**
+ * An opaque handle to a dynamic argument list used to build argument vectors
+ * for type-erased formatting functions.
+ *
+ * Use `fmt_args_create` to create an instance and `fmt_args_destroy` to
+ * release it.
+ */
+typedef struct fmt_args fmt_args;
+
+/** Creates a new empty argument list. Returns NULL on allocation failure. */
+FMT_C_API fmt_args* fmt_args_create(void);
+
+/** Destroys an argument list created with `fmt_args_create`. */
+FMT_C_API void fmt_args_destroy(fmt_args* a);
+
+/** Adds an `int` argument to the list. */
+FMT_C_API void fmt_args_push_int(fmt_args* a, int v);
+
+/** Adds an `unsigned int` argument to the list. */
+FMT_C_API void fmt_args_push_uint(fmt_args* a, unsigned v);
+
+/** Adds a `long long` argument to the list. */
+FMT_C_API void fmt_args_push_long_long(fmt_args* a, long long v);
+
+/** Adds an `unsigned long long` argument to the list. */
+FMT_C_API void fmt_args_push_ulong_long(fmt_args* a, unsigned long long v);
+
+/** Adds a `double` argument to the list. */
+FMT_C_API void fmt_args_push_double(fmt_args* a, double v);
+
+/**
+ * Adds a Boolean argument (0 = false, any other value = true) to the list.
+ */
+FMT_C_API void fmt_args_push_bool(fmt_args* a, int v);
+
+/**
+ * Adds a null-terminated C string argument to the list. The string is copied
+ * into the argument list.
+ */
+FMT_C_API void fmt_args_push_cstring(fmt_args* a, const char* v);
+
+/**
+ * Adds a string argument with explicit length to the list. The string is
+ * copied into the argument list and does not need to be null-terminated.
+ */
+FMT_C_API void fmt_args_push_string(fmt_args* a, const char* v, size_t n);
+
+/** Adds a pointer argument to the list. */
+FMT_C_API void fmt_args_push_pointer(fmt_args* a, const void* v);
+
+/**
+ * Formats a string according to `fmt` (of length `fmt_len`) using the
+ * arguments in `a`.
+ *
+ * Returns a newly allocated null-terminated string that the caller must
+ * release with `fmt_free`, or NULL if formatting fails (e.g. invalid format
+ * string or allocation failure).
+ *
+ * **Example**:
+ *
+ *     fmt_args* a = fmt_args_create();
+ *     fmt_args_push_int(a, 42);
+ *     char* s = fmt_vformat("The answer is {}.", 17, a);
+ *     // s == "The answer is 42."
+ *     fmt_free(s);
+ *     fmt_args_destroy(a);
+ */
+FMT_C_API char* fmt_vformat(const char* fmt, size_t fmt_len,
+                             const fmt_args* a);
+
+/** Releases a string returned by `fmt_vformat`. */
+FMT_C_API void fmt_free(void* p);
+
+#ifdef __cplusplus
+}  // extern "C"
+#endif
+
+#endif  // FMT_C_H_
diff --git a/src/c.cc b/src/c.cc
new file mode 100644
index 0000000..cb5a7c2
--- /dev/null
+++ b/src/c.cc
@@ -0,0 +1,86 @@
+// Formatting library for C++ - C API implementation
+//
+// Copyright (c) 2012 - present, Victor Zverovich
+// All rights reserved.
+//
+// For the license information refer to format.h.
+
+#include "fmt/c.h"
+
+#include "fmt/args.h"
+
+#include <stdlib.h>
+#include <string.h>
+
+#include <string>
+
+struct fmt_args {
+  fmt::dynamic_format_arg_store<fmt::format_context> store;
+};
+
+extern "C" {
+
+fmt_args* fmt_args_create(void) {
+  try {
+    return new fmt_args();
+  } catch (...) {
+    return nullptr;
+  }
+}
+
+void fmt_args_destroy(fmt_args* a) { delete a; }
+
+void fmt_args_push_int(fmt_args* a, int v) {
+  if (a) a->store.push_back(v);
+}
+
+void fmt_args_push_uint(fmt_args* a, unsigned v) {
+  if (a) a->store.push_back(v);
+}
+
+void fmt_args_push_long_long(fmt_args* a, long long v) {
+  if (a) a->store.push_back(v);
+}
+
+void fmt_args_push_ulong_long(fmt_args* a, unsigned long long v) {
+  if (a) a->store.push_back(v);
+}
+
+void fmt_args_push_double(fmt_args* a, double v) {
+  if (a) a->store.push_back(v);
+}
+
+void fmt_args_push_bool(fmt_args* a, int v) {
+  if (a) a->store.push_back(static_cast<bool>(v));
+}
+
+void fmt_args_push_cstring(fmt_args* a, const char* v) {
+  if (a) a->store.push_back(v);
+}
+
+void fmt_args_push_string(fmt_args* a, const char* v, size_t n) {
+  if (a) a->store.push_back(std::string(v, n));
+}
+
+void fmt_args_push_pointer(fmt_args* a, const void* v) {
+  if (a) a->store.push_back(v);
+}
+
+char* fmt_vformat(const char* fmt_str, size_t fmt_len, const fmt_args* a) {
+  if (!fmt_str || !a) return nullptr;
+  try {
+    auto result =
+        fmt::vformat(fmt::string_view(fmt_str, fmt_len), a->store);
+    char* buf = static_cast<char*>(malloc(result.size() + 1));
+    if (!buf) return nullptr;
+    memcpy(buf, result.data(), result.size());
+    buf[result.size()] = '\0';
+    return buf;
+  } catch (...) {
+    return nullptr;
+  }
+}
+
+void fmt_free(void* p) { free(p); }
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
index 0000000..ab6d899
--- /dev/null
+++ b/test/c-api-test.cc
@@ -0,0 +1,165 @@
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
+TEST(c_api_test, create_destroy) {
+  fmt_args* a = fmt_args_create();
+  ASSERT_NE(nullptr, a);
+  fmt_args_destroy(a);
+}
+
+TEST(c_api_test, format_no_args) {
+  fmt_args* a = fmt_args_create();
+  char* s = fmt_vformat("hello", 5, a);
+  ASSERT_NE(nullptr, s);
+  EXPECT_STREQ("hello", s);
+  fmt_free(s);
+  fmt_args_destroy(a);
+}
+
+TEST(c_api_test, format_int) {
+  fmt_args* a = fmt_args_create();
+  fmt_args_push_int(a, 42);
+  char* s = fmt_vformat("{}", 2, a);
+  ASSERT_NE(nullptr, s);
+  EXPECT_STREQ("42", s);
+  fmt_free(s);
+  fmt_args_destroy(a);
+}
+
+TEST(c_api_test, format_uint) {
+  fmt_args* a = fmt_args_create();
+  fmt_args_push_uint(a, 42u);
+  char* s = fmt_vformat("{}", 2, a);
+  ASSERT_NE(nullptr, s);
+  EXPECT_STREQ("42", s);
+  fmt_free(s);
+  fmt_args_destroy(a);
+}
+
+TEST(c_api_test, format_long_long) {
+  fmt_args* a = fmt_args_create();
+  fmt_args_push_long_long(a, -1234567890123LL);
+  char* s = fmt_vformat("{}", 2, a);
+  ASSERT_NE(nullptr, s);
+  EXPECT_STREQ("-1234567890123", s);
+  fmt_free(s);
+  fmt_args_destroy(a);
+}
+
+TEST(c_api_test, format_ulong_long) {
+  fmt_args* a = fmt_args_create();
+  fmt_args_push_ulong_long(a, 1234567890123ULL);
+  char* s = fmt_vformat("{}", 2, a);
+  ASSERT_NE(nullptr, s);
+  EXPECT_STREQ("1234567890123", s);
+  fmt_free(s);
+  fmt_args_destroy(a);
+}
+
+TEST(c_api_test, format_double) {
+  fmt_args* a = fmt_args_create();
+  fmt_args_push_double(a, 1.5);
+  char* s = fmt_vformat("{}", 2, a);
+  ASSERT_NE(nullptr, s);
+  EXPECT_STREQ("1.5", s);
+  fmt_free(s);
+  fmt_args_destroy(a);
+}
+
+TEST(c_api_test, format_bool) {
+  fmt_args* a = fmt_args_create();
+  fmt_args_push_bool(a, 1);
+  fmt_args_push_bool(a, 0);
+  char* s = fmt_vformat("{} {}", 5, a);
+  ASSERT_NE(nullptr, s);
+  EXPECT_STREQ("true false", s);
+  fmt_free(s);
+  fmt_args_destroy(a);
+}
+
+TEST(c_api_test, format_cstring) {
+  fmt_args* a = fmt_args_create();
+  fmt_args_push_cstring(a, "world");
+  char* s = fmt_vformat("hello, {}!", 10, a);
+  ASSERT_NE(nullptr, s);
+  EXPECT_STREQ("hello, world!", s);
+  fmt_free(s);
+  fmt_args_destroy(a);
+}
+
+TEST(c_api_test, format_string_with_length) {
+  fmt_args* a = fmt_args_create();
+  // "hello world" but only push "hello" (5 chars, no null term needed)
+  fmt_args_push_string(a, "hello world", 5);
+  char* s = fmt_vformat("{}", 2, a);
+  ASSERT_NE(nullptr, s);
+  EXPECT_STREQ("hello", s);
+  fmt_free(s);
+  fmt_args_destroy(a);
+}
+
+TEST(c_api_test, format_pointer) {
+  fmt_args* a = fmt_args_create();
+  fmt_args_push_pointer(a, nullptr);
+  char* s = fmt_vformat("{}", 2, a);
+  ASSERT_NE(nullptr, s);
+  // The exact format is implementation-defined, but should not be empty.
+  EXPECT_GT(strlen(s), 0u);
+  fmt_free(s);
+  fmt_args_destroy(a);
+}
+
+TEST(c_api_test, format_multiple_args) {
+  fmt_args* a = fmt_args_create();
+  fmt_args_push_int(a, 1);
+  fmt_args_push_cstring(a, "two");
+  fmt_args_push_double(a, 3.0);
+  char* s = fmt_vformat("{} {} {}", 8, a);
+  ASSERT_NE(nullptr, s);
+  EXPECT_STREQ("1 two 3", s);
+  fmt_free(s);
+  fmt_args_destroy(a);
+}
+
+TEST(c_api_test, format_spec) {
+  fmt_args* a = fmt_args_create();
+  fmt_args_push_int(a, 42);
+  char* s = fmt_vformat("{:08}", 5, a);
+  ASSERT_NE(nullptr, s);
+  EXPECT_STREQ("00000042", s);
+  fmt_free(s);
+  fmt_args_destroy(a);
+}
+
+TEST(c_api_test, invalid_format_returns_null) {
+  fmt_args* a = fmt_args_create();
+  // Too few arguments for the format string.
+  char* s = fmt_vformat("{} {}", 5, a);
+  EXPECT_EQ(nullptr, s);
+  fmt_args_destroy(a);
+}
+
+TEST(c_api_test, cstring_is_copied) {
+  fmt_args* a = fmt_args_create();
+  {
+    char buf[] = "hello";
+    fmt_args_push_cstring(a, buf);
+    // Overwrite buf after pushing - the stored copy should be unaffected.
+    buf[0] = 'X';
+  }
+  char* s = fmt_vformat("{}", 2, a);
+  ASSERT_NE(nullptr, s);
+  EXPECT_STREQ("hello", s);
+  fmt_free(s);
+  fmt_args_destroy(a);
+}
```

### with-lumen

**Rating: Good**

```diff
diff --git a/CMakeLists.txt b/CMakeLists.txt
index 53afc51..a297980 100644
--- a/CMakeLists.txt
+++ b/CMakeLists.txt
@@ -299,9 +299,9 @@ endfunction()
 
 # Define the fmt library, its includes and the needed defines.
 set(FMT_HEADERS)
-add_headers(FMT_HEADERS args.h base.h chrono.h color.h compile.h core.h format.h
-                        format-inl.h os.h ostream.h printf.h ranges.h std.h
-                        xchar.h)
+add_headers(FMT_HEADERS args.h base.h c-api.h chrono.h color.h compile.h core.h
+                        format.h format-inl.h os.h ostream.h printf.h ranges.h
+                        std.h xchar.h)
 set(FMT_SOURCES src/format.cc)
 
 add_module_library(fmt src/fmt.cc FALLBACK
@@ -383,6 +383,23 @@ target_include_directories(fmt-header-only
   $<BUILD_INTERFACE:${PROJECT_SOURCE_DIR}/include>
   $<INSTALL_INTERFACE:${FMT_INC_DIR}>)
 
+# The fmt-c library provides a C-compatible API.
+add_library(fmt-c src/fmt-c.cc)
+add_library(fmt::fmt-c ALIAS fmt-c)
+target_link_libraries(fmt-c PUBLIC fmt)
+target_include_directories(fmt-c ${FMT_SYSTEM_HEADERS_ATTRIBUTE} BEFORE PUBLIC
+  $<BUILD_INTERFACE:${PROJECT_SOURCE_DIR}/include>
+  $<INSTALL_INTERFACE:${FMT_INC_DIR}>)
+set_target_properties(fmt-c PROPERTIES
+  VERSION ${FMT_VERSION} SOVERSION ${CPACK_PACKAGE_VERSION_MAJOR}
+  DEBUG_POSTFIX "${FMT_DEBUG_POSTFIX}"
+  COMPILE_PDB_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}"
+  COMPILE_PDB_NAME "fmt-c"
+  COMPILE_PDB_NAME_DEBUG "fmt-c${FMT_DEBUG_POSTFIX}")
+if (BUILD_SHARED_LIBS)
+  target_compile_definitions(fmt-c PRIVATE FMT_LIB_EXPORT INTERFACE FMT_SHARED)
+endif ()
+
 # Install targets.
 if (FMT_INSTALL)
   include(CMakePackageConfigHelpers)
@@ -422,7 +439,7 @@ if (FMT_INSTALL)
     ${project_config}
     INSTALL_DESTINATION ${FMT_CMAKE_DIR})
 
-  set(INSTALL_TARGETS fmt fmt-header-only)
+  set(INSTALL_TARGETS fmt fmt-header-only fmt-c)
 
   set(INSTALL_FILE_SET)
   if (FMT_USE_CMAKE_MODULES)
diff --git a/include/fmt/c-api.h b/include/fmt/c-api.h
new file mode 100644
index 0000000..6f9f565
--- /dev/null
+++ b/include/fmt/c-api.h
@@ -0,0 +1,104 @@
+// Formatting library for C++ - C API
+//
+// Copyright (c) 2012 - present, Victor Zverovich
+// All rights reserved.
+//
+// For the license information refer to format.h.
+
+// A C API for fmt that allows C code and languages with C FFI to use the
+// library's formatting capabilities. Unlike the printf family of functions,
+// this API is type-safe and doesn't encode type information in the format
+// string.
+
+#ifndef FMT_C_API_H_
+#define FMT_C_API_H_
+
+#include <stddef.h>  // size_t
+#include <stdio.h>   // FILE
+
+#if defined(FMT_LIB_EXPORT) || defined(FMT_SHARED)
+#  if defined(_WIN32)
+#    if defined(FMT_LIB_EXPORT)
+#      define FMT_C_API_EXPORT __declspec(dllexport)
+#    else
+#      define FMT_C_API_EXPORT __declspec(dllimport)
+#    endif
+#  else
+#    define FMT_C_API_EXPORT __attribute__((visibility("default")))
+#  endif
+#else
+#  define FMT_C_API_EXPORT
+#endif
+
+#ifdef __cplusplus
+extern "C" {
+#endif
+
+// An opaque handle to a list of format arguments.
+typedef struct fmt_args fmt_args;
+
+// Creates a format argument store. The returned handle must be destroyed with
+// fmt_args_destroy after use.
+FMT_C_API_EXPORT fmt_args* fmt_args_create(void);
+
+// Destroys a format argument store created with fmt_args_create.
+FMT_C_API_EXPORT void fmt_args_destroy(fmt_args* args);
+
+// Pushes an int argument.
+FMT_C_API_EXPORT void fmt_args_push_int(fmt_args* args, int value);
+
+// Pushes an unsigned int argument.
+FMT_C_API_EXPORT void fmt_args_push_uint(fmt_args* args, unsigned int value);
+
+// Pushes a long long argument.
+FMT_C_API_EXPORT void fmt_args_push_long_long(fmt_args* args, long long value);
+
+// Pushes an unsigned long long argument.
+FMT_C_API_EXPORT void fmt_args_push_ulong_long(fmt_args* args,
+                                                unsigned long long value);
+
+// Pushes a double argument.
+FMT_C_API_EXPORT void fmt_args_push_double(fmt_args* args, double value);
+
+// Pushes a long double argument.
+FMT_C_API_EXPORT void fmt_args_push_long_double(fmt_args* args,
+                                                 long double value);
+
+// Pushes a bool argument (non-zero value is true).
+FMT_C_API_EXPORT void fmt_args_push_bool(fmt_args* args, int value);
+
+// Pushes a char argument.
+FMT_C_API_EXPORT void fmt_args_push_char(fmt_args* args, char value);
+
+// Pushes a null-terminated C string argument.
+FMT_C_API_EXPORT void fmt_args_push_cstring(fmt_args* args, const char* str);
+
+// Pushes a string argument with explicit length (may contain null bytes).
+FMT_C_API_EXPORT void fmt_args_push_string(fmt_args* args, const char* str,
+                                            size_t size);
+
+// Pushes a pointer argument.
+FMT_C_API_EXPORT void fmt_args_push_pointer(fmt_args* args, const void* ptr);
+
+// Formats a string using the given arguments and returns a newly allocated
+// null-terminated result. Returns NULL on error.
+// The caller must free the returned string with fmt_free().
+FMT_C_API_EXPORT char* fmt_vformat(const char* fmt, size_t fmt_size,
+                                    fmt_args* args);
+
+// Frees memory allocated by fmt_vformat.
+FMT_C_API_EXPORT void fmt_free(void* ptr);
+
+// Prints a formatted string to stdout.
+FMT_C_API_EXPORT void fmt_vprint(const char* fmt, size_t fmt_size,
+                                  fmt_args* args);
+
+// Prints a formatted string to a file.
+FMT_C_API_EXPORT void fmt_vfprint(FILE* f, const char* fmt, size_t fmt_size,
+                                   fmt_args* args);
+
+#ifdef __cplusplus
+}
+#endif
+
+#endif  // FMT_C_API_H_
diff --git a/src/fmt-c.cc b/src/fmt-c.cc
new file mode 100644
index 0000000..3b6315c
--- /dev/null
+++ b/src/fmt-c.cc
@@ -0,0 +1,108 @@
+// Formatting library for C++ - C API implementation
+//
+// Copyright (c) 2012 - present, Victor Zverovich
+// All rights reserved.
+//
+// For the license information refer to format.h.
+
+#include "fmt/c-api.h"
+
+#include <cstdlib>  // std::malloc, std::free
+#include <cstring>  // std::memcpy
+
+#include "fmt/args.h"
+
+struct fmt_args {
+  fmt::dynamic_format_arg_store<fmt::format_context> store;
+};
+
+extern "C" {
+
+fmt_args* fmt_args_create(void) {
+  try {
+    return new fmt_args();
+  } catch (...) {
+    return nullptr;
+  }
+}
+
+void fmt_args_destroy(fmt_args* args) { delete args; }
+
+void fmt_args_push_int(fmt_args* args, int value) {
+  if (args) args->store.push_back(value);
+}
+
+void fmt_args_push_uint(fmt_args* args, unsigned int value) {
+  if (args) args->store.push_back(value);
+}
+
+void fmt_args_push_long_long(fmt_args* args, long long value) {
+  if (args) args->store.push_back(value);
+}
+
+void fmt_args_push_ulong_long(fmt_args* args, unsigned long long value) {
+  if (args) args->store.push_back(value);
+}
+
+void fmt_args_push_double(fmt_args* args, double value) {
+  if (args) args->store.push_back(value);
+}
+
+void fmt_args_push_long_double(fmt_args* args, long double value) {
+  if (args) args->store.push_back(value);
+}
+
+void fmt_args_push_bool(fmt_args* args, int value) {
+  if (args) args->store.push_back(static_cast<bool>(value));
+}
+
+void fmt_args_push_char(fmt_args* args, char value) {
+  if (args) args->store.push_back(value);
+}
+
+void fmt_args_push_cstring(fmt_args* args, const char* str) {
+  if (args) args->store.push_back(str);
+}
+
+void fmt_args_push_string(fmt_args* args, const char* str, size_t size) {
+  if (args) args->store.push_back(fmt::string_view(str, size));
+}
+
+void fmt_args_push_pointer(fmt_args* args, const void* ptr) {
+  if (args) args->store.push_back(ptr);
+}
+
+char* fmt_vformat(const char* fmt_str, size_t fmt_size, fmt_args* args) {
+  if (!args) return nullptr;
+  try {
+    auto result =
+        fmt::vformat(fmt::string_view(fmt_str, fmt_size), args->store);
+    auto* buf = static_cast<char*>(std::malloc(result.size() + 1));
+    if (!buf) return nullptr;
+    std::memcpy(buf, result.c_str(), result.size() + 1);
+    return buf;
+  } catch (...) {
+    return nullptr;
+  }
+}
+
+void fmt_free(void* ptr) { std::free(ptr); }
+
+void fmt_vprint(const char* fmt_str, size_t fmt_size, fmt_args* args) {
+  if (!args) return;
+  try {
+    fmt::vprint(fmt::string_view(fmt_str, fmt_size), args->store);
+  } catch (...) {
+  }
+}
+
+void fmt_vfprint(FILE* f, const char* fmt_str, size_t fmt_size,
+                 fmt_args* args) {
+  if (!f || !args) return;
+  try {
+    fmt::vprint(f, fmt::string_view(fmt_str, fmt_size), args->store);
+  } catch (...) {
+  }
+}
+
+}  // extern "C"
diff --git a/test/CMakeLists.txt b/test/CMakeLists.txt
index 3f7d24a..4b06751 100644
--- a/test/CMakeLists.txt
+++ b/test/CMakeLists.txt
@@ -85,6 +85,14 @@ target_compile_definitions(enforce-checks-test PRIVATE
 add_executable(perf-sanity perf-sanity.cc)
 target_link_libraries(perf-sanity fmt::fmt)
 
+# C API test - compiled as C to verify the header is C-compatible.
+enable_language(C)
+add_executable(c-api-test c-api-test.c)
+target_include_directories(c-api-test PRIVATE
+  $<BUILD_INTERFACE:${PROJECT_SOURCE_DIR}/include>)
+target_link_libraries(c-api-test fmt-c)
+add_test(NAME c-api-test COMMAND c-api-test)
+
 if (FMT_MODULE)
   # The tests need {fmt} to be compiled as traditional library
   # because of visibility of implementation details.
diff --git a/test/c-api-test.c b/test/c-api-test.c
new file mode 100644
index 0000000..3ed2ad8
--- /dev/null
+++ b/test/c-api-test.c
@@ -0,0 +1,130 @@
+// Formatting library for C++ - C API test
+//
+// Copyright (c) 2012 - present, Victor Zverovich
+// All rights reserved.
+//
+// For the license information refer to format.h.
+
+// This test file is compiled as C to verify the C API is usable from C code.
+
+#include <assert.h>
+#include <stdio.h>
+#include <string.h>
+
+#include "fmt/c-api.h"
+
+static void test_format_int(void) {
+  fmt_args* args = fmt_args_create();
+  assert(args != NULL);
+  fmt_args_push_int(args, 42);
+  char* result = fmt_vformat("{}", 2, args);
+  assert(result != NULL);
+  assert(strcmp(result, "42") == 0);
+  fmt_free(result);
+  fmt_args_destroy(args);
+}
+
+static void test_format_double(void) {
+  fmt_args* args = fmt_args_create();
+  assert(args != NULL);
+  fmt_args_push_double(args, 1.5);
+  char* result = fmt_vformat("{}", 2, args);
+  assert(result != NULL);
+  assert(strcmp(result, "1.5") == 0);
+  fmt_free(result);
+  fmt_args_destroy(args);
+}
+
+static void test_format_string(void) {
+  fmt_args* args = fmt_args_create();
+  assert(args != NULL);
+  fmt_args_push_cstring(args, "hello");
+  char* result = fmt_vformat("{}", 2, args);
+  assert(result != NULL);
+  assert(strcmp(result, "hello") == 0);
+  fmt_free(result);
+  fmt_args_destroy(args);
+}
+
+static void test_format_multiple_args(void) {
+  fmt_args* args = fmt_args_create();
+  assert(args != NULL);
+  fmt_args_push_int(args, 1);
+  fmt_args_push_cstring(args, "two");
+  fmt_args_push_double(args, 3.0);
+  char* result = fmt_vformat("{} {} {}", 8, args);
+  assert(result != NULL);
+  assert(strcmp(result, "1 two 3") == 0);
+  fmt_free(result);
+  fmt_args_destroy(args);
+}
+
+static void test_format_bool(void) {
+  fmt_args* args = fmt_args_create();
+  assert(args != NULL);
+  fmt_args_push_bool(args, 1);
+  char* result = fmt_vformat("{}", 2, args);
+  assert(result != NULL);
+  assert(strcmp(result, "true") == 0);
+  fmt_free(result);
+  fmt_args_destroy(args);
+}
+
+static void test_format_char(void) {
+  fmt_args* args = fmt_args_create();
+  assert(args != NULL);
+  fmt_args_push_char(args, 'A');
+  char* result = fmt_vformat("{}", 2, args);
+  assert(result != NULL);
+  assert(strcmp(result, "A") == 0);
+  fmt_free(result);
+  fmt_args_destroy(args);
+}
+
+static void test_format_string_with_length(void) {
+  fmt_args* args = fmt_args_create();
+  assert(args != NULL);
+  fmt_args_push_string(args, "hello world", 5);
+  char* result = fmt_vformat("{}", 2, args);
+  assert(result != NULL);
+  assert(strcmp(result, "hello") == 0);
+  fmt_free(result);
+  fmt_args_destroy(args);
+}
+
+static void test_format_pointer(void) {
+  fmt_args* args = fmt_args_create();
+  assert(args != NULL);
+  fmt_args_push_pointer(args, NULL);
+  char* result = fmt_vformat("{}", 2, args);
+  assert(result != NULL);
+  /* Just verify it produces some output - exact format depends on platform */
+  assert(strlen(result) > 0);
+  fmt_free(result);
+  fmt_args_destroy(args);
+}
+
+static void test_format_specs(void) {
+  fmt_args* args = fmt_args_create();
+  assert(args != NULL);
+  fmt_args_push_int(args, 42);
+  char* result = fmt_vformat("{:>10}", 6, args);
+  assert(result != NULL);
+  assert(strcmp(result, "        42") == 0);
+  fmt_free(result);
+  fmt_args_destroy(args);
+}
+
+int main(void) {
+  test_format_int();
+  test_format_double();
+  test_format_string();
+  test_format_multiple_args();
+  test_format_bool();
+  test_format_char();
+  test_format_string_with_length();
+  test_format_pointer();
+  test_format_specs();
+  printf("All C API tests passed.\n");
+  return 0;
+}
```


