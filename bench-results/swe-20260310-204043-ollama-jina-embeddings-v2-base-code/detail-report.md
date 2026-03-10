# SWE-Bench Detail Report

Generated: 2026-03-10 19:43 UTC

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
| **baseline** | 51.5s | 12 | 248768 | 1936 | $0.1857 |
| **with-lumen** | 34.0s | 6 | 65649 | 796 | $0.1359 |

### baseline

**Rating: Good**

```diff
diff --git a/src/Monolog/Formatter/JsonFormatter.php b/src/Monolog/Formatter/JsonFormatter.php
index b59639e..ef838dc 100644
--- a/src/Monolog/Formatter/JsonFormatter.php
+++ b/src/Monolog/Formatter/JsonFormatter.php
@@ -11,6 +11,7 @@
 
 namespace Monolog\Formatter;
 
+use Monolog\Utils;
 use Stringable;
 use Throwable;
 use Monolog\LogRecord;
@@ -195,7 +196,11 @@ class JsonFormatter extends NormalizerFormatter
             }
 
             if ($data instanceof Stringable) {
-                return $data->__toString();
+                try {
+                    return $data->__toString();
+                } catch (\Throwable) {
+                    return '[object ' . Utils::getClass($data) . ']';
+                }
             }
 
             if (\get_class($data) === '__PHP_Incomplete_Class') {
```

### with-lumen

**Rating: Good**

```diff
diff --git a/src/Monolog/Formatter/JsonFormatter.php b/src/Monolog/Formatter/JsonFormatter.php
index b59639e..3f8f094 100644
--- a/src/Monolog/Formatter/JsonFormatter.php
+++ b/src/Monolog/Formatter/JsonFormatter.php
@@ -195,7 +195,11 @@ class JsonFormatter extends NormalizerFormatter
             }
 
             if ($data instanceof Stringable) {
-                return $data->__toString();
+                try {
+                    return $data->__toString();
+                } catch (\Throwable) {
+                    return '[object ' . \get_class($data) . ']';
+                }
             }
 
             if (\get_class($data) === '__PHP_Incomplete_Class') {
```


