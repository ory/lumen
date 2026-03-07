# SWE-Bench Detail Report

Generated: 2026-03-07 09:53 UTC

---

## go-easy [go / easy]

**Issue:** CORSMethodMiddleware adds allowed methods from wrong routes in subrouters

> When using CORSMethodMiddleware in a subrouter, it can add allowed methods from other routes that shouldn't be present. The issue is that getAllMethodsForRoute uses Walk and matches substrings of paths rather than doing proper route matching.
> 
> Reproduction:
> ```go
> router := mux.NewRouter().StrictSlash(true)
> subrouter := router.PathPrefix("/test").Subrouter()
> subrouter.HandleFunc("/hello", Hello).Methods(http.MethodGet, http.MethodOptions, http.MethodPost)
> subrouter.HandleFunc("/hello/{name}", HelloName).Methods(http.MethodGet, http.MethodOptions)
> subrouter.Use(mux.CORSMethodMiddleware(subrouter))
> ```
> 
> When requesting GET /test/hello/name, the Access-Control-Allow-Methods header returns `GET,OPTIONS,POST,GET,OPTIONS` — it incorrectly includes POST from the /hello route because /hello is a substring of /hello/name.
> 
> Expected: `Access-Control-Allow-Methods: GET,OPTIONS` (only methods from the matching route)
> Actual: `Access-Control-Allow-Methods: GET,OPTIONS,POST,GET,OPTIONS` (includes methods from /hello too)

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 85.4s | 15 | 101329 | 4379 | $0.2682 |
| **mcp-full** | 67.9s | 16 | 126232 | 2987 | $0.2605 |

### baseline

**Rating: Good**

```diff
diff --git a/middleware.go b/middleware.go
index cf2b26d..5e3e398 100644
--- a/middleware.go
+++ b/middleware.go
@@ -59,18 +59,14 @@ func getAllMethodsForRoute(r *Router, req *http.Request) ([]string, error) {
 	var allMethods []string
 
 	err := r.Walk(func(route *Route, _ *Router, _ []*Route) error {
-		for _, m := range route.matchers {
-			if _, ok := m.(*routeRegexp); ok {
-				if m.Match(req, &RouteMatch{}) {
-					methods, err := route.GetMethods()
-					if err != nil {
-						return err
-					}
-
-					allMethods = append(allMethods, methods...)
-				}
-				break
+		var match RouteMatch
+		if route.Match(req, &match) || match.MatchErr == ErrMethodMismatch {
+			methods, err := route.GetMethods()
+			if err != nil {
+				return err
 			}
+
+			allMethods = append(allMethods, methods...)
 		}
 		return nil
 	})
```

### mcp-full

**Rating: Good**

```diff
diff --git a/middleware.go b/middleware.go
index cf2b26d..107a9bc 100644
--- a/middleware.go
+++ b/middleware.go
@@ -59,18 +59,13 @@ func getAllMethodsForRoute(r *Router, req *http.Request) ([]string, error) {
 	var allMethods []string
 
 	err := r.Walk(func(route *Route, _ *Router, _ []*Route) error {
-		for _, m := range route.matchers {
-			if _, ok := m.(*routeRegexp); ok {
-				if m.Match(req, &RouteMatch{}) {
-					methods, err := route.GetMethods()
-					if err != nil {
-						return err
-					}
-
-					allMethods = append(allMethods, methods...)
-				}
-				break
+		routeMatch := &RouteMatch{}
+		if route.Match(req, routeMatch) || routeMatch.MatchErr == ErrMethodMismatch {
+			methods, err := route.GetMethods()
+			if err != nil {
+				return err
 			}
+			allMethods = append(allMethods, methods...)
 		}
 		return nil
 	})
```


