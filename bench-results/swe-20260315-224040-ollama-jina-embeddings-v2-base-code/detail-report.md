# SWE-Bench Detail Report

Generated: 2026-03-15 21:43 UTC

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
| **baseline** | 66.8s | 218 | 1139206 | 6125 | $0.1760 |
| **with-lumen** | 74.0s | 174 | 935076 | 7126 | $0.1964 |

### baseline

**Rating: Perfect**

```diff
diff --git a/src/Monolog/Formatter/JsonFormatter.php b/src/Monolog/Formatter/JsonFormatter.php
index b59639e..00a1deb 100644
--- a/src/Monolog/Formatter/JsonFormatter.php
+++ b/src/Monolog/Formatter/JsonFormatter.php
@@ -195,7 +195,12 @@ class JsonFormatter extends NormalizerFormatter
             }
 
             if ($data instanceof Stringable) {
-                return $data->__toString();
+                try {
+                    return $data->__toString();
+                } catch (Throwable) {
+                    // If __toString() throws an exception, use the class name as fallback
+                    return $data::class;
+                }
             }
 
             if (\get_class($data) === '__PHP_Incomplete_Class') {
diff --git a/tests/Monolog/Formatter/JsonFormatterTest.php b/tests/Monolog/Formatter/JsonFormatterTest.php
index 56cef22..ba73215 100644
--- a/tests/Monolog/Formatter/JsonFormatterTest.php
+++ b/tests/Monolog/Formatter/JsonFormatterTest.php
@@ -338,6 +338,24 @@ class JsonFormatterTest extends MonologTestCase
             $record
         );
     }
+
+    public function testFormatStringableObjectWithThrowingToString()
+    {
+        $formatter = new JsonFormatter();
+
+        $record = $formatter->format($this->getRecord(
+            Level::Debug,
+            'Testing',
+            channel: 'test',
+            datetime: new \DateTimeImmutable('2022-02-22 00:00:00'),
+            context: [
+                'throwingStringable' => new TestJsonNormWithThrowingToString,
+            ],
+        ));
+
+        $parsed = json_decode($record, true);
+        $this->assertSame('Monolog\Formatter\TestJsonNormWithThrowingToString', $parsed['context']['throwingStringable']);
+    }
 }
 
 class TestJsonNormPublic
@@ -370,3 +388,11 @@ class TestJsonNormWithToString
         return 'stringified';
     }
 }
+
+class TestJsonNormWithThrowingToString
+{
+    public function __toString()
+    {
+        throw new \RuntimeException('Failed to convert to string');
+    }
+}
```

### with-lumen

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
diff --git a/tests/Monolog/Formatter/JsonFormatterTest.php b/tests/Monolog/Formatter/JsonFormatterTest.php
index 56cef22..f05023b 100644
--- a/tests/Monolog/Formatter/JsonFormatterTest.php
+++ b/tests/Monolog/Formatter/JsonFormatterTest.php
@@ -338,6 +338,27 @@ class JsonFormatterTest extends MonologTestCase
             $record
         );
     }
+
+    public function testFormatObjectWithThrowingToString()
+    {
+        $formatter = new JsonFormatter();
+
+        $record = $formatter->format($this->getRecord(
+            Level::Debug,
+            'Testing',
+            channel: 'test',
+            datetime: new \DateTimeImmutable('2022-02-22 00:00:00'),
+            context: [
+                'stringableThrows' => new TestJsonNormWithToStringThrows,
+            ],
+        ));
+
+        // The formatter should gracefully handle the exception and use the class name as fallback
+        $this->assertSame(
+            '{"message":"Testing","context":{"stringableThrows":"Monolog\\\\Formatter\\\\TestJsonNormWithToStringThrows"},"level":100,"level_name":"DEBUG","channel":"test","datetime":"2022-02-22T00:00:00+00:00","extra":{}}'."\n",
+            $record
+        );
+    }
 }
 
 class TestJsonNormPublic
@@ -370,3 +391,11 @@ class TestJsonNormWithToString
         return 'stringified';
     }
 }
+
+class TestJsonNormWithToStringThrows implements \Stringable
+{
+    public function __toString(): string
+    {
+        throw new \RuntimeException('__toString failed');
+    }
+}
```


