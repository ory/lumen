# SWE-Bench Detail Report

Generated: 2026-04-13 10:01 UTC

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
| **baseline** | 350.1s | 305 | 2276925 | 15511 | $0.5606 |
| **with-lumen** | 231.4s | 271 | 2143284 | 13896 | $0.3516 |

### baseline

**Rating: Good**

```diff
diff --git a/formats/json/commonMain/src/kotlinx/serialization/json/internal/JsonPath.kt b/formats/json/commonMain/src/kotlinx/serialization/json/internal/JsonPath.kt
index 14e70a4..9725ad8 100644
--- a/formats/json/commonMain/src/kotlinx/serialization/json/internal/JsonPath.kt
+++ b/formats/json/commonMain/src/kotlinx/serialization/json/internal/JsonPath.kt
@@ -132,9 +132,13 @@ internal class JsonPath {
     private fun prettyString(it: Any?) = (it as? SerialDescriptor)?.serialName ?: it.toString()
 
     private fun resize() {
+        val oldSize = indicies.size
         val newSize = currentDepth * 2
         currentObjectPath = currentObjectPath.copyOf(newSize)
         indicies = indicies.copyOf(newSize)
+        for (i in oldSize until newSize) {
+            indicies[i] = -1
+        }
     }
 
     override fun toString(): String = getPath()
```

### with-lumen

**Rating: Perfect**

```diff
diff --git a/formats/json/commonMain/src/kotlinx/serialization/json/internal/JsonPath.kt b/formats/json/commonMain/src/kotlinx/serialization/json/internal/JsonPath.kt
index 14e70a4..708f2ac 100644
--- a/formats/json/commonMain/src/kotlinx/serialization/json/internal/JsonPath.kt
+++ b/formats/json/commonMain/src/kotlinx/serialization/json/internal/JsonPath.kt
@@ -134,7 +134,11 @@ internal class JsonPath {
     private fun resize() {
         val newSize = currentDepth * 2
         currentObjectPath = currentObjectPath.copyOf(newSize)
-        indicies = indicies.copyOf(newSize)
+        val newIndices = indicies.copyOf(newSize)
+        for (i in indicies.size until newSize) {
+            newIndices[i] = -1
+        }
+        indicies = newIndices
     }
 
     override fun toString(): String = getPath()
```


