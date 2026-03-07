# SWE-Bench Detail Report

Generated: 2026-03-06 18:04 UTC

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
| **baseline** | 110.9s | 194 | 1009459 | 8541 | $0.2250 |
| **mcp-full** | 93.0s | 146 | 759992 | 8060 | $0.1550 |

### baseline

**Rating: Good**

```diff
diff --git a/middleware.go b/middleware.go
index cf2b26d..df628a0 100644
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
+		match := &RouteMatch{}
+		if route.Match(req, match) || match.MatchErr == ErrMethodMismatch {
+			methods, err := route.GetMethods()
+			if err != nil {
+				return err
 			}
+
+			allMethods = append(allMethods, methods...)
 		}
 		return nil
 	})
diff --git a/middleware_test.go b/middleware_test.go
index 27647af..c4d671d 100644
--- a/middleware_test.go
+++ b/middleware_test.go
@@ -478,6 +478,47 @@ func TestCORSMethodMiddleware(t *testing.T) {
 	}
 }
 
+func TestCORSMethodMiddlewareSubrouter(t *testing.T) {
+	testCases := []struct {
+		name                                    string
+		registerRoutes                          func(r *Router)
+		requestMethod                           string
+		requestPath                             string
+		expectedAccessControlAllowMethodsHeader string
+	}{
+		{
+			name: "subrouter with substring paths",
+			registerRoutes: func(r *Router) {
+				subrouter := r.PathPrefix("/test").Subrouter()
+				subrouter.HandleFunc("/hello", stringHandler("a")).Methods(http.MethodGet, http.MethodOptions, http.MethodPost)
+				subrouter.HandleFunc("/hello/{name}", stringHandler("b")).Methods(http.MethodGet, http.MethodOptions)
+				subrouter.Use(CORSMethodMiddleware(subrouter))
+			},
+			requestMethod:                           "OPTIONS",
+			requestPath:                             "/test/hello/world",
+			expectedAccessControlAllowMethodsHeader: "GET,OPTIONS",
+		},
+	}
+
+	for _, tt := range testCases {
+		t.Run(tt.name, func(t *testing.T) {
+			router := NewRouter()
+
+			tt.registerRoutes(router)
+
+			rw := NewRecorder()
+			req := newRequest(tt.requestMethod, tt.requestPath)
+
+			router.ServeHTTP(rw, req)
+
+			actualMethodsHeader := rw.Header().Get("Access-Control-Allow-Methods")
+			if actualMethodsHeader != tt.expectedAccessControlAllowMethodsHeader {
+				t.Fatalf("Expected Access-Control-Allow-Methods to equal %s but got %s", tt.expectedAccessControlAllowMethodsHeader, actualMethodsHeader)
+			}
+		})
+	}
+}
+
 func TestMiddlewareOnMultiSubrouter(t *testing.T) {
 	first := "first"
 	second := "second"
```

### mcp-full

**Rating: Good**

```diff
diff --git a/middleware.go b/middleware.go
index cf2b26d..03e14e5 100644
--- a/middleware.go
+++ b/middleware.go
@@ -59,19 +59,29 @@ func getAllMethodsForRoute(r *Router, req *http.Request) ([]string, error) {
 	var allMethods []string
 
 	err := r.Walk(func(route *Route, _ *Router, _ []*Route) error {
+		// Check if route matches the request by checking all matchers except method matcher
+		match := &RouteMatch{}
+		routeMatches := true
+
 		for _, m := range route.matchers {
-			if _, ok := m.(*routeRegexp); ok {
-				if m.Match(req, &RouteMatch{}) {
-					methods, err := route.GetMethods()
-					if err != nil {
-						return err
-					}
+			// Skip method matcher as we want to get methods for all matching routes
+			if _, ok := m.(methodMatcher); ok {
+				continue
+			}
 
-					allMethods = append(allMethods, methods...)
-				}
+			if !m.Match(req, match) {
+				routeMatches = false
 				break
 			}
 		}
+
+		if routeMatches {
+			methods, err := route.GetMethods()
+			if err == nil {
+				allMethods = append(allMethods, methods...)
+			}
+		}
+
 		return nil
 	})
 
diff --git a/middleware_test.go b/middleware_test.go
index 27647af..8e885b9 100644
--- a/middleware_test.go
+++ b/middleware_test.go
@@ -449,6 +449,24 @@ func TestCORSMethodMiddleware(t *testing.T) {
 			expectedAccessControlAllowMethodsHeader: "GET,PUT,PATCH,OPTIONS",
 			expectedResponse:                        "b",
 		},
+		{
+			name: "subrouter does not set methods from other subroute paths",
+			registerRoutes: func(r *Router) {
+				subrouter := r.PathPrefix("/test").Subrouter()
+				subrouter.HandleFunc("/hello", stringHandler("a")).Methods(http.MethodGet, http.MethodOptions, http.MethodPost)
+				subrouter.HandleFunc("/hello/{name}", stringHandler("b")).Methods(http.MethodGet, http.MethodOptions)
+				subrouter.Use(CORSMethodMiddleware(subrouter))
+			},
+			requestMethod: "OPTIONS",
+			requestPath:   "/test/hello/name",
+			requestHeader: http.Header{
+				"Access-Control-Request-Method":  []string{"GET"},
+				"Access-Control-Request-Headers": []string{"Authorization"},
+				"Origin":                         []string{"http://example.com"},
+			},
+			expectedAccessControlAllowMethodsHeader: "GET,OPTIONS",
+			expectedResponse:                        "b",
+		},
 	}
 
 	for _, tt := range testCases {
```


