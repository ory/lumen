# SWE-Bench Detail Report

Generated: 2026-03-27 18:08 UTC

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
| **baseline** | 246.1s | 439 | 4126087 | 21286 | $0.6342 |
| **with-lumen** | 50.9s | 126 | 663146 | 3862 | $0.1533 |

### baseline

**Rating: Good**

```diff
diff --git a/pkgs/shelf_router/lib/src/router.dart b/pkgs/shelf_router/lib/src/router.dart
index 720d3c3..8613f01 100644
--- a/pkgs/shelf_router/lib/src/router.dart
+++ b/pkgs/shelf_router/lib/src/router.dart
@@ -58,11 +58,12 @@ extension RouterParams on Request {
   }
 }
 
-/// Middleware to remove body from request.
+/// Middleware to remove body from HEAD requests.
+///
+/// Per RFC 9110 Section 9.3.2, HEAD and GET responses should be identical
+/// except for the body. The content-length header should reflect the size of
+/// the body that WOULD have been returned by a GET request.
 final _removeBody = createMiddleware(responseHandler: (r) {
-  if (r.headers.containsKey('content-length')) {
-    r = r.change(headers: {'content-length': '0'});
-  }
   return r.change(body: <int>[]);
 });
 
diff --git a/pkgs/shelf_router/test/router_test.dart b/pkgs/shelf_router/test/router_test.dart
index fdffa51..cb19c9c 100644
--- a/pkgs/shelf_router/test/router_test.dart
+++ b/pkgs/shelf_router/test/router_test.dart
@@ -204,4 +204,24 @@ void main() {
     final b2 = await Router.routeNotFound.readAsString();
     expect(b2, b1);
   });
+
+  test('HEAD request should preserve content-length header per RFC 9110', () async {
+    var app = Router();
+
+    app.get('/hello', (Request request) {
+      // Explicitly set content-length header
+      return Response.ok('hello-world',
+          headers: {'content-length': '11'});
+    });
+
+    // Create a request with HEAD method
+    final headRequest = Request('HEAD', Uri.parse('http://localhost/hello'));
+    final headResponse = await app(headRequest);
+
+    // Per RFC 9110 Section 9.3.2, the content-length should be preserved
+    expect(headResponse.headers['content-length'], '11');
+
+    // Verify that the body is empty
+    expect(await headResponse.readAsString(), isEmpty);
+  });
 }
```

### with-lumen

**Rating: Good**

```diff
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
 
```


