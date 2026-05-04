# SWE-Bench Detail Report

Generated: 2026-05-04 22:09 UTC

---

## dart-hard [dart]

**Issue:** receiveTimeout fires during active downloads even when bytes are being received

> Package: dio
> Version: 5.3.3
> Operating system: Android
> Flutter version: 3.13.6
> Dart version: 3.1.3
> 
> I expected the download to finish without any issue since bytes are constantly received. But a `TimeoutException` is thrown after `receiveTimeout` milliseconds even when the download is progressing.
> 
> Steps to reproduce:
> 1. Configure Dio with a `receiveTimeout` that is shorter than the total download duration (e.g. 3 seconds timeout for a 10 second download)
> 2. Start a download using `Dio.download()`
> 3. Monitor `onReceiveProgress` — bytes are arriving continuously
> 4. After `receiveTimeout` elapses, the download is cancelled with a `DioException` of type `receiveTimeout`
> 
> Expected behavior:
> The `receiveTimeout` should measure the gap between received byte events, not the total download duration. If bytes are arriving continuously, the timeout should keep resetting. Per the documentation: "timeout during data transfer of each bytes and not the overall timing during the receiving."
> 
> Actual behavior:
> The download terminates with a `TimeoutException` after the `receiveTimeout` period regardless of whether bytes are being received. The `onReceiveProgress` callback shows steady byte delivery right up until the timeout kills the transfer.
> 
> This makes `receiveTimeout` unusable for large file downloads — any file that takes longer to download than the timeout period will fail, even on a fast connection with no stalls.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 402.1s | 2816 | 439568 | 21364 | $0.7414 |
| **with-lumen** | 210.1s | 18 | 309897 | 12892 | $0.4139 |

### baseline

**Rating: Poor**

```diff
diff --git a/dio/lib/src/adapters/browser_adapter.dart b/dio/lib/src/adapters/browser_adapter.dart
index a4601ca..49d2095 100644
--- a/dio/lib/src/adapters/browser_adapter.dart
+++ b/dio/lib/src/adapters/browser_adapter.dart
@@ -183,6 +183,11 @@ class BrowserHttpClientAdapter implements HttpClientAdapter {
             StackTrace.current,
           );
           xhr.abort();
+        } else {
+          downloadStopwatch
+            ..stop()
+            ..reset()
+            ..start();
         }
       }
 
diff --git a/dio/lib/src/adapters/io_adapter.dart b/dio/lib/src/adapters/io_adapter.dart
index 5c7d94e..f8d8a91 100644
--- a/dio/lib/src/adapters/io_adapter.dart
+++ b/dio/lib/src/adapters/io_adapter.dart
@@ -220,6 +220,8 @@ class IOHttpClientAdapter implements HttpClientAdapter {
             );
             responseStream.detachSocket().then((socket) => socket.destroy());
           } else {
+            stopwatch.reset();
+            stopwatch.start();
             sink.add(Uint8List.fromList(data));
           }
         },
diff --git a/dio/lib/src/dio/dio_for_native.dart b/dio/lib/src/dio/dio_for_native.dart
index d16fa41..5e821db 100644
--- a/dio/lib/src/dio/dio_for_native.dart
+++ b/dio/lib/src/dio/dio_for_native.dart
@@ -124,9 +124,32 @@ class DioForNative with DioMixin implements Dio {
       }
     }
 
+    final receiveTimeout = response.requestOptions.receiveTimeout;
+    Timer? receiveTimer;
     late StreamSubscription subscription;
+
+    void scheduleReceiveTimeout() {
+      receiveTimer?.cancel();
+      if (receiveTimeout != null && receiveTimeout > Duration.zero) {
+        receiveTimer = Timer(receiveTimeout, () async {
+          await subscription.cancel();
+          await closeAndDelete();
+          if (!completer.isCompleted) {
+            completer.completeError(
+              DioException.receiveTimeout(
+                timeout: receiveTimeout,
+                requestOptions: response.requestOptions,
+              ),
+            );
+          }
+        });
+      }
+    }
+
+    scheduleReceiveTimeout();
     subscription = stream.listen(
       (data) {
+        scheduleReceiveTimeout();
         subscription.pause();
         // Write file asynchronously
         asyncWrite = raf.writeFrom(data).then((result) {
@@ -153,6 +176,7 @@ class DioForNative with DioMixin implements Dio {
         });
       },
       onDone: () async {
+        receiveTimer?.cancel();
         try {
           await asyncWrite;
           closed = true;
@@ -165,6 +189,7 @@ class DioForNative with DioMixin implements Dio {
         }
       },
       onError: (e) async {
+        receiveTimer?.cancel();
         try {
           await closeAndDelete();
         } finally {
@@ -176,28 +201,11 @@ class DioForNative with DioMixin implements Dio {
       cancelOnError: true,
     );
     cancelToken?.whenCancel.then((_) async {
+      receiveTimer?.cancel();
       await subscription.cancel();
       await closeAndDelete();
     });
 
-    final timeout = response.requestOptions.receiveTimeout;
-    if (timeout != null) {
-      future = future.timeout(timeout).catchError(
-        (dynamic e, StackTrace s) async {
-          await subscription.cancel();
-          await closeAndDelete();
-          if (e is TimeoutException) {
-            throw DioException.receiveTimeout(
-              timeout: timeout,
-              requestOptions: response.requestOptions,
-              error: e,
-            );
-          } else {
-            throw e;
-          }
-        },
-      );
-    }
     return DioMixin.listenCancelForAsyncTask(cancelToken, future);
   }
 }
```

### with-lumen

**Rating: Good**

```diff
diff --git a/dio/lib/src/adapters/io_adapter.dart b/dio/lib/src/adapters/io_adapter.dart
index 5c7d94e..9b530fd 100644
--- a/dio/lib/src/adapters/io_adapter.dart
+++ b/dio/lib/src/adapters/io_adapter.dart
@@ -220,6 +220,9 @@ class IOHttpClientAdapter implements HttpClientAdapter {
             );
             responseStream.detachSocket().then((socket) => socket.destroy());
           } else {
+            // Reset so the next chunk measures the gap since this one.
+            stopwatch.reset();
+            stopwatch.start();
             sink.add(Uint8List.fromList(data));
           }
         },
diff --git a/dio/lib/src/dio/dio_for_native.dart b/dio/lib/src/dio/dio_for_native.dart
index d16fa41..819e63e 100644
--- a/dio/lib/src/dio/dio_for_native.dart
+++ b/dio/lib/src/dio/dio_for_native.dart
@@ -124,9 +124,32 @@ class DioForNative with DioMixin implements Dio {
       }
     }
 
+    final timeout = response.requestOptions.receiveTimeout;
+    Timer? receiveTimer;
+
+    void resetReceiveTimer() {
+      receiveTimer?.cancel();
+      if (timeout != null && timeout > Duration.zero) {
+        receiveTimer = Timer(timeout, () async {
+          await subscription.cancel();
+          await closeAndDelete();
+          if (!completer.isCompleted) {
+            completer.completeError(
+              DioException.receiveTimeout(
+                timeout: timeout,
+                requestOptions: response.requestOptions,
+              ),
+            );
+          }
+        });
+      }
+    }
+
     late StreamSubscription subscription;
     subscription = stream.listen(
       (data) {
+        // Each arriving chunk resets the inter-byte timer.
+        resetReceiveTimer();
         subscription.pause();
         // Write file asynchronously
         asyncWrite = raf.writeFrom(data).then((result) {
@@ -153,6 +176,7 @@ class DioForNative with DioMixin implements Dio {
         });
       },
       onDone: () async {
+        receiveTimer?.cancel();
         try {
           await asyncWrite;
           closed = true;
@@ -165,6 +189,7 @@ class DioForNative with DioMixin implements Dio {
         }
       },
       onError: (e) async {
+        receiveTimer?.cancel();
         try {
           await closeAndDelete();
         } finally {
@@ -176,28 +201,14 @@ class DioForNative with DioMixin implements Dio {
       cancelOnError: true,
     );
     cancelToken?.whenCancel.then((_) async {
+      receiveTimer?.cancel();
       await subscription.cancel();
       await closeAndDelete();
     });
 
-    final timeout = response.requestOptions.receiveTimeout;
-    if (timeout != null) {
-      future = future.timeout(timeout).catchError(
-        (dynamic e, StackTrace s) async {
-          await subscription.cancel();
-          await closeAndDelete();
-          if (e is TimeoutException) {
-            throw DioException.receiveTimeout(
-              timeout: timeout,
-              requestOptions: response.requestOptions,
-              error: e,
-            );
-          } else {
-            throw e;
-          }
-        },
-      );
-    }
+    // Start the timer to catch the case where no first byte ever arrives.
+    resetReceiveTimer();
+
     return DioMixin.listenCancelForAsyncTask(cancelToken, future);
   }
 }
```

---

## python-hard [python]

**Issue:** New 'default' behaviour in Click 8.3.x is broken for negative boolean flags

> The behaviour of `default=...` for flag values changed in Click 8.3.0, as documented in the release notes. The new behaviour is broken for negative flags, unless I am holding it wrong.
> 
> Here's a minimal repro:
> 
> ```python
> import click
> 
> @click.command('foo')
> @click.option('--without-xyz', 'enable_xyz',
>               help="Disable xyz", flag_value=False, default=True, show_default=True)
> def foo(enable_xyz):
>     print(f'enable_xyz = {enable_xyz}')
> 
> foo()
> ```
> 
> Expected (Click 8.2.x):
> - `./foo.py` → `enable_xyz = True`
> - `./foo.py --without-xyz` → `enable_xyz = False`
> 
> Actual (Click 8.3.0+):
> - `./foo.py` → `enable_xyz = False`
> - `./foo.py --without-xyz` → `enable_xyz = False`
> 
> So the default value of `True` is being silently replaced by the `flag_value` of `False`, making the flag completely useless — it always returns `False` regardless of whether it was passed or not.
> 
> This is a regression from Click 8.2.x where `default=True` was respected as a literal Python value for boolean flags.
> 
> Questions:
> 1. Is the `default=True` special case truly necessary?
> 2. Should it produce an explicit error rather than a silent behavioral change?
> 3. What is the canonical approach for negative options that default to "off"?
> 4. Would improved documentation help?
> 
> Environment: Python 3.10, Click 8.3.x

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | — | — | — | — | — |
| **with-lumen** | — | — | — | — | — |

### baseline

### with-lumen

**Rating: INVALID (lumen not used)**


