# SWE-Bench Detail Report

Generated: 2026-04-12 13:25 UTC

---

## kotlin-hard [kotlin]

**Issue:** Unexpected token for data object in a deep json structure gives ArrayIndexOutOfBoundsException

> **Describe the bug**
> 
> When the JSON parser encounters an unexpected token while parsing a `data object` deep in a JSON structure (roughly 8 or more levels of nesting), it throws `ArrayIndexOutOfBoundsException` instead of the usual `SerializationException` with a helpful message pointing at the offending path.
> 
> For shallow structures, the parser produces a clean error such as:
> 
>     Unexpected JSON token at offset N: Expected start of the object '{', but had 'f' instead at path: $.bar.bar.baz
> 
> Once the nesting depth crosses a threshold, the same kind of malformed input instead surfaces as:
> 
>     java.lang.ArrayIndexOutOfBoundsException: Index 0 out of bounds for length 0
> 
> with no indication of where in the JSON the problem occurred.
> 
> **To Reproduce**
> 
> ```kotlin
> package com.foo.bar
> 
> import kotlinx.serialization.Serializable
> import kotlinx.serialization.SerializationException
> import kotlinx.serialization.json.Json
> import org.junit.jupiter.api.Assertions.assertEquals
> import org.junit.jupiter.api.Test
> import org.junit.jupiter.api.assertThrows
> 
> class TestDeepJsonFailure {
>     @Test
>     fun `test deep failure`() {
>         val exception = assertThrows<SerializationException> {
>             Json.decodeFromString<Foo>("""{"bar": {"bar": {"bar": {"bar": {"bar": {"bar": {"baz": false}}}}}}}""")
>         }
> 
>         assertEquals(
>             """
>                 Unexpected JSON token at offset 56: Expected start of the object '{', but had 'f' instead at path: ${'$'}.bar.bar.bar.bar.bar.bar.baz
>                 JSON input: {"bar": {"bar": {"bar": {"bar": {"bar": {"bar": {"baz": false}}}}}}}
>             """.trimIndent(),
>             exception.message
>         )
>     }
> 
>     @Test
>     fun `test deeper failure`() {
>         val exception = assertThrows<ArrayIndexOutOfBoundsException> {
>             Json.decodeFromString<Foo>("""{"bar": {"bar": {"bar": {"bar": {"bar": {"bar": {"bar": {"baz": false}}}}}}}}""")
>         }
> 
>         assertEquals(
>             "Index 0 out of bounds for length 0",
>             exception.message
>         )
>     }
> }
> 
> @Serializable
> data class Foo(val bar: Foo?, val baz: Baz?)
> 
> @Serializable
> data object Baz
> ```
> 
> The first test passes (shallow malformed input produces a `SerializationException` with a readable path), while the second test documents the broken behaviour: for a slightly deeper but otherwise equivalent malformed input the decoder throws an `ArrayIndexOutOfBoundsException` with the useless message `Index 0 out of bounds for length 0`.
> 
> **Expected behavior**
> 
> The parser should fail with the same kind of `SerializationException` carrying a readable error message and path, regardless of how deeply the malformed `data object` is nested. Users should never see an `ArrayIndexOutOfBoundsException` escape from `Json.decodeFromString`.
> 
> **Environment**
>  - Kotlin version: 2.1.10
>  - kotlinx.serialization version: 1.8.0
>  - Kotlin platforms: JVM
> 

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 308.3s | 41 | 745371 | 14799 | $0.7397 |
| **with-lumen** | 195.8s | 31 | 505250 | 11776 | $0.5098 |

### baseline

**Rating: Good**

```diff
diff --git a/formats/json-tests/commonTest/src/kotlinx/serialization/JsonPathTest.kt b/formats/json-tests/commonTest/src/kotlinx/serialization/JsonPathTest.kt
index 8d31ba2..3ed80bc 100644
--- a/formats/json-tests/commonTest/src/kotlinx/serialization/JsonPathTest.kt
+++ b/formats/json-tests/commonTest/src/kotlinx/serialization/JsonPathTest.kt
@@ -156,6 +156,25 @@ class JsonPathTest : JsonTestBase() {
         expectPath(expectedPath) { json.decodeFromString(Sealed.serializer(), malformed) }
     }
 
+    @Serializable
+    data class FooNested(val bar: FooNested?, val baz: BazObject?)
+
+    @Serializable
+    data object BazObject
+
+    @Test
+    fun testDeepNestingWithDataObject() {
+        // Shallow nesting (6 levels): produces a readable SerializationException
+        expectPath("$.bar.bar.bar.bar.bar.bar.baz") {
+            Json.decodeFromString<FooNested>("""{"bar": {"bar": {"bar": {"bar": {"bar": {"bar": {"baz": false}}}}}}}""")
+        }
+        // Deep nesting (7 levels): previously threw ArrayIndexOutOfBoundsException because
+        // indicies[] elements were zero-initialized after resize instead of -1
+        expectPath("$.bar.bar.bar.bar.bar.bar.bar.baz") {
+            Json.decodeFromString<FooNested>("""{"bar": {"bar": {"bar": {"bar": {"bar": {"bar": {"bar": {"baz": false}}}}}}}}""")
+        }
+    }
+
     private inline fun expectPath(path: String, block: () -> Unit) {
         val message = runCatching { block() }
             .exceptionOrNull()!!.message!!
diff --git a/formats/json/commonMain/src/kotlinx/serialization/json/internal/JsonPath.kt b/formats/json/commonMain/src/kotlinx/serialization/json/internal/JsonPath.kt
index 14e70a4..0607669 100644
--- a/formats/json/commonMain/src/kotlinx/serialization/json/internal/JsonPath.kt
+++ b/formats/json/commonMain/src/kotlinx/serialization/json/internal/JsonPath.kt
@@ -134,7 +134,9 @@ internal class JsonPath {
     private fun resize() {
         val newSize = currentDepth * 2
         currentObjectPath = currentObjectPath.copyOf(newSize)
+        val oldSize = indicies.size
         indicies = indicies.copyOf(newSize)
+        indicies.fill(-1, oldSize, newSize)
     }
 
     override fun toString(): String = getPath()
```

### with-lumen

**Rating: Perfect**

```diff
diff --git a/formats/json-tests/commonTest/src/kotlinx/serialization/JsonPathTest.kt b/formats/json-tests/commonTest/src/kotlinx/serialization/JsonPathTest.kt
index 8d31ba2..3ac1035 100644
--- a/formats/json-tests/commonTest/src/kotlinx/serialization/JsonPathTest.kt
+++ b/formats/json-tests/commonTest/src/kotlinx/serialization/JsonPathTest.kt
@@ -156,6 +156,24 @@ class JsonPathTest : JsonTestBase() {
         expectPath(expectedPath) { json.decodeFromString(Sealed.serializer(), malformed) }
     }
 
+    @Serializable
+    data class FooWithDataObject(val bar: FooWithDataObject?, val baz: DataObjectBaz?)
+
+    @Serializable
+    data object DataObjectBaz
+
+    @Test
+    fun testDeepNestingWithDataObject() {
+        // Regression test: deeply nested data object should produce SerializationException with
+        // a readable path, not ArrayIndexOutOfBoundsException caused by uninitialized indicies
+        // after JsonPath.resize().
+        expectPath("$.bar.bar.bar.bar.bar.bar.bar.baz") {
+            Json.decodeFromString<FooWithDataObject>(
+                """{"bar": {"bar": {"bar": {"bar": {"bar": {"bar": {"bar": {"baz": false}}}}}}}}"""
+            )
+        }
+    }
+
     private inline fun expectPath(path: String, block: () -> Unit) {
         val message = runCatching { block() }
             .exceptionOrNull()!!.message!!
diff --git a/formats/json/commonMain/src/kotlinx/serialization/json/internal/JsonPath.kt b/formats/json/commonMain/src/kotlinx/serialization/json/internal/JsonPath.kt
index 14e70a4..0607669 100644
--- a/formats/json/commonMain/src/kotlinx/serialization/json/internal/JsonPath.kt
+++ b/formats/json/commonMain/src/kotlinx/serialization/json/internal/JsonPath.kt
@@ -134,7 +134,9 @@ internal class JsonPath {
     private fun resize() {
         val newSize = currentDepth * 2
         currentObjectPath = currentObjectPath.copyOf(newSize)
+        val oldSize = indicies.size
         indicies = indicies.copyOf(newSize)
+        indicies.fill(-1, oldSize, newSize)
     }
 
     override fun toString(): String = getPath()
```


