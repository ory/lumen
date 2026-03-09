# SWE-Bench Detail Report

Generated: 2026-03-09 00:32 UTC

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
| **baseline run1** | 148.5s | 27 | 702170 | 8287 | $0.7711 |
| **baseline run2** | 48.6s | 9 | 132453 | 2886 | $0.2283 |
| **baseline run3** | 237.6s | 37 | 1088686 | 14141 | $1.1268 |
| **baseline run4** | 225.9s | 38 | 1096696 | 13330 | $1.0968 |
| **baseline run5** | 52.5s | 16 | 305219 | 2571 | $0.3193 |
| **with-lumen run1** | 224.9s | 26 | 871950 | 13560 | $1.0715 |
| **with-lumen run2** | 169.7s | 18 | 430331 | 9789 | $0.9062 |
| **with-lumen run3** | 74.4s | 10 | 171895 | 3892 | $0.2975 |
| **with-lumen run4** | 57.1s | 6 | 71358 | 3149 | $0.2013 |
| **with-lumen run5** | 49.9s | 8 | 117376 | 2756 | $0.2186 |

### baseline run1

**Rating: Poor**

```diff
diff --git a/middleware.go b/middleware.go
index cf2b26d..917eede 100644
--- a/middleware.go
+++ b/middleware.go
@@ -59,18 +59,12 @@ func getAllMethodsForRoute(r *Router, req *http.Request) ([]string, error) {
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
+		if route.regexp.path != nil && route.regexp.path.Match(req, &RouteMatch{}) {
+			methods, err := route.GetMethods()
+			if err != nil {
+				return err
 			}
+			allMethods = append(allMethods, methods...)
 		}
 		return nil
 	})
```

### baseline run2

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

### baseline run3

**Rating: Poor**

```diff
diff --git a/middleware.go b/middleware.go
index cf2b26d..03506f1 100644
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
+		if route.regexp.path != nil && route.regexp.path.Match(req, &RouteMatch{}) {
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

### baseline run4

**Rating: Poor**

```diff
diff --git a/middleware.go b/middleware.go
index cf2b26d..03506f1 100644
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
+		if route.regexp.path != nil && route.regexp.path.Match(req, &RouteMatch{}) {
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

### baseline run5

**Rating: Good**

```diff
diff --git a/middleware.go b/middleware.go
index cf2b26d..7ce3173 100644
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
+		routeMatch := &RouteMatch{}
+		if route.Match(req, routeMatch) || routeMatch.MatchErr == ErrMethodMismatch {
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

### with-lumen run1

**Rating: Poor**

```diff
diff --git a/middleware.go b/middleware.go
index cf2b26d..03506f1 100644
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
+		if route.regexp.path != nil && route.regexp.path.Match(req, &RouteMatch{}) {
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

### with-lumen run2

**Rating: Poor**

```diff
diff --git a/middleware.go b/middleware.go
index cf2b26d..917eede 100644
--- a/middleware.go
+++ b/middleware.go
@@ -59,18 +59,12 @@ func getAllMethodsForRoute(r *Router, req *http.Request) ([]string, error) {
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
+		if route.regexp.path != nil && route.regexp.path.Match(req, &RouteMatch{}) {
+			methods, err := route.GetMethods()
+			if err != nil {
+				return err
 			}
+			allMethods = append(allMethods, methods...)
 		}
 		return nil
 	})
```

### with-lumen run3

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
```

### with-lumen run4

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

### with-lumen run5

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

---

## go-hard [go / hard]

**Issue:** terraform plan -refresh-only does not detect output value changes

> When using `terraform plan -refresh-only`, changes to root output values are not detected. Even when output values have been modified externally (e.g. through state manipulation or drift), the refresh-only plan reports no changes and does not mark the plan as applyable.
> 
> Steps to reproduce:
> 1. Create a Terraform configuration with root output values
> 2. Change the output values externally (e.g. through state manipulation so that the prior state and refreshed state differ)
> 3. Run `terraform plan -refresh-only`
> 4. Observe that the plan does not detect the output value changes and incorrectly reports that no changes are needed, even though the values clearly differ between the prior state and the refreshed state
> 
> Expected behavior: The refresh-only plan should detect that root output values have changed and mark the plan as applyable so the updated values can be persisted.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline run1** | 91.5s | 6 | 71127 | 1278 | $0.4045 |
| **baseline run2** | 111.6s | 6 | 72097 | 1185 | $0.3852 |
| **baseline run3** | 14.9s | 5 | 47712 | 597 | $0.1075 |
| **baseline run4** | 12.2s | 5 | 47714 | 580 | $0.1071 |
| **baseline run5** | 14.3s | 5 | 47735 | 601 | $0.1077 |
| **with-lumen run1** | 14.8s | 8 | 68639 | 738 | $0.1282 |
| **with-lumen run2** | 14.7s | 8 | 68646 | 717 | $0.1278 |
| **with-lumen run3** | — | — | — | — | — |
| **with-lumen run4** | 12.9s | 5 | 48420 | 599 | $0.1101 |
| **with-lumen run5** | 12.6s | 5 | 48954 | 606 | $0.1120 |

### baseline run1

**Rating: Poor**

```diff
diff --git a/internal/states/state_equal.go b/internal/states/state_equal.go
index 97414311c..4112c58cd 100644
--- a/internal/states/state_equal.go
+++ b/internal/states/state_equal.go
@@ -102,7 +102,7 @@ func (s *State) RootOutputValuesEqual(s2 *State) bool {
 	}
 
 	for k, v1 := range s2.RootOutputValues {
-		v2, ok := s2.RootOutputValues[k]
+		v2, ok := s.RootOutputValues[k]
 		if !ok || !v1.Equal(v2) {
 			return false
 		}
```

### baseline run2

**Rating: Poor**

```diff
diff --git a/internal/states/state_equal.go b/internal/states/state_equal.go
index 97414311c..4112c58cd 100644
--- a/internal/states/state_equal.go
+++ b/internal/states/state_equal.go
@@ -102,7 +102,7 @@ func (s *State) RootOutputValuesEqual(s2 *State) bool {
 	}
 
 	for k, v1 := range s2.RootOutputValues {
-		v2, ok := s2.RootOutputValues[k]
+		v2, ok := s.RootOutputValues[k]
 		if !ok || !v1.Equal(v2) {
 			return false
 		}
```

### baseline run3

**Rating: Perfect**

```diff
diff --git a/internal/states/state_equal.go b/internal/states/state_equal.go
index 97414311c..6ed37ff25 100644
--- a/internal/states/state_equal.go
+++ b/internal/states/state_equal.go
@@ -101,7 +101,7 @@ func (s *State) RootOutputValuesEqual(s2 *State) bool {
 		return false
 	}
 
-	for k, v1 := range s2.RootOutputValues {
+	for k, v1 := range s.RootOutputValues {
 		v2, ok := s2.RootOutputValues[k]
 		if !ok || !v1.Equal(v2) {
 			return false
```

### baseline run4

**Rating: Perfect**

```diff
diff --git a/internal/states/state_equal.go b/internal/states/state_equal.go
index 97414311c..6ed37ff25 100644
--- a/internal/states/state_equal.go
+++ b/internal/states/state_equal.go
@@ -101,7 +101,7 @@ func (s *State) RootOutputValuesEqual(s2 *State) bool {
 		return false
 	}
 
-	for k, v1 := range s2.RootOutputValues {
+	for k, v1 := range s.RootOutputValues {
 		v2, ok := s2.RootOutputValues[k]
 		if !ok || !v1.Equal(v2) {
 			return false
```

### baseline run5

**Rating: Perfect**

```diff
diff --git a/internal/states/state_equal.go b/internal/states/state_equal.go
index 97414311c..6ed37ff25 100644
--- a/internal/states/state_equal.go
+++ b/internal/states/state_equal.go
@@ -101,7 +101,7 @@ func (s *State) RootOutputValuesEqual(s2 *State) bool {
 		return false
 	}
 
-	for k, v1 := range s2.RootOutputValues {
+	for k, v1 := range s.RootOutputValues {
 		v2, ok := s2.RootOutputValues[k]
 		if !ok || !v1.Equal(v2) {
 			return false
```

### with-lumen run1

**Rating: Perfect**

```diff
diff --git a/internal/states/state_equal.go b/internal/states/state_equal.go
index 97414311c..6ed37ff25 100644
--- a/internal/states/state_equal.go
+++ b/internal/states/state_equal.go
@@ -101,7 +101,7 @@ func (s *State) RootOutputValuesEqual(s2 *State) bool {
 		return false
 	}
 
-	for k, v1 := range s2.RootOutputValues {
+	for k, v1 := range s.RootOutputValues {
 		v2, ok := s2.RootOutputValues[k]
 		if !ok || !v1.Equal(v2) {
 			return false
```

### with-lumen run2

**Rating: Perfect**

```diff
diff --git a/internal/states/state_equal.go b/internal/states/state_equal.go
index 97414311c..6ed37ff25 100644
--- a/internal/states/state_equal.go
+++ b/internal/states/state_equal.go
@@ -101,7 +101,7 @@ func (s *State) RootOutputValuesEqual(s2 *State) bool {
 		return false
 	}
 
-	for k, v1 := range s2.RootOutputValues {
+	for k, v1 := range s.RootOutputValues {
 		v2, ok := s2.RootOutputValues[k]
 		if !ok || !v1.Equal(v2) {
 			return false
```

### with-lumen run3

**Rating: Poor**

### with-lumen run4

**Rating: Perfect**

```diff
diff --git a/internal/states/state_equal.go b/internal/states/state_equal.go
index 97414311c..6ed37ff25 100644
--- a/internal/states/state_equal.go
+++ b/internal/states/state_equal.go
@@ -101,7 +101,7 @@ func (s *State) RootOutputValuesEqual(s2 *State) bool {
 		return false
 	}
 
-	for k, v1 := range s2.RootOutputValues {
+	for k, v1 := range s.RootOutputValues {
 		v2, ok := s2.RootOutputValues[k]
 		if !ok || !v1.Equal(v2) {
 			return false
```

### with-lumen run5

**Rating: Perfect**

```diff
diff --git a/internal/states/state_equal.go b/internal/states/state_equal.go
index 97414311c..6ed37ff25 100644
--- a/internal/states/state_equal.go
+++ b/internal/states/state_equal.go
@@ -101,7 +101,7 @@ func (s *State) RootOutputValuesEqual(s2 *State) bool {
 		return false
 	}
 
-	for k, v1 := range s2.RootOutputValues {
+	for k, v1 := range s.RootOutputValues {
 		v2, ok := s2.RootOutputValues[k]
 		if !ok || !v1.Equal(v2) {
 			return false
```

---

## go-medium [go / medium]

**Issue:** host matching but variables are empty

> When using wildcard host matching with gorilla/mux, the route matches correctly but the extracted variables (mux.Vars) are empty when the request includes a port number and the route template does not specify a port.
> 
> Reproduction:
> ```go
> r := mux.NewRouter()
> r.Host("{subdomain:.+}.localhost").Subrouter().Path("/").HandlerFunc(Handler2)
> r.Host("{subdomain:.+}.localhost:{port:.*}").Subrouter().Path("/").HandlerFunc(Handler1)
> http.Handle("/", r)
> ```
> 
> Running `curl localhost:8080 -H 'Host: test.localhost:8080'` matches Handler2 (the route without port) but `mux.Vars(r)` returns an empty map `map[]`. Running `curl localhost:8080 -H 'Host: test.localhost'` (no port) correctly returns `map[subdomain:test]`.
> 
> The `subdomain` variable should never be empty when the route is matching. When the order of handler registration is switched (or rolling back to v1.7.1), it works as expected.
> 
> Expected behavior: When a wildcard host route matches a request, `mux.Vars()` should always return the correct captured variables regardless of whether a port is present in the request and absent from the route template.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline run1** | 28.5s | 6 | 95347 | 1472 | $0.2414 |
| **baseline run2** | 20.7s | 6 | 76077 | 906 | $0.1566 |
| **baseline run3** | 22.8s | 6 | 76044 | 944 | $0.1575 |
| **baseline run4** | 29.2s | 8 | 125651 | 1389 | $0.1958 |
| **baseline run5** | 23.0s | 6 | 75837 | 1024 | $0.1593 |
| **with-lumen run1** | 22.6s | 6 | 76627 | 890 | $0.1574 |
| **with-lumen run2** | 23.6s | 6 | 76548 | 985 | $0.1600 |
| **with-lumen run3** | 22.2s | 6 | 76400 | 954 | $0.1587 |
| **with-lumen run4** | 24.3s | 6 | 76598 | 1126 | $0.1641 |
| **with-lumen run5** | 24.7s | 6 | 76509 | 1039 | $0.1614 |

### baseline run1

**Rating: Good**

```diff
diff --git a/regexp.go b/regexp.go
index 96dd94a..efa839a 100644
--- a/regexp.go
+++ b/regexp.go
@@ -325,6 +325,11 @@ func (v routeRegexpGroup) setMatch(req *http.Request, m *RouteMatch, r *Route) {
 	// Store host variables.
 	if v.host != nil {
 		host := getHost(req)
+		if v.host.wildcardHostPort {
+			if i := strings.Index(host, ":"); i != -1 {
+				host = host[:i]
+			}
+		}
 		matches := v.host.regexp.FindStringSubmatchIndex(host)
 		if len(matches) > 0 {
 			extractVars(host, matches, v.host.varsN, m.Vars)
```

### baseline run2

**Rating: Good**

```diff
diff --git a/regexp.go b/regexp.go
index 96dd94a..efa839a 100644
--- a/regexp.go
+++ b/regexp.go
@@ -325,6 +325,11 @@ func (v routeRegexpGroup) setMatch(req *http.Request, m *RouteMatch, r *Route) {
 	// Store host variables.
 	if v.host != nil {
 		host := getHost(req)
+		if v.host.wildcardHostPort {
+			if i := strings.Index(host, ":"); i != -1 {
+				host = host[:i]
+			}
+		}
 		matches := v.host.regexp.FindStringSubmatchIndex(host)
 		if len(matches) > 0 {
 			extractVars(host, matches, v.host.varsN, m.Vars)
```

### baseline run3

**Rating: Good**

```diff
diff --git a/regexp.go b/regexp.go
index 96dd94a..efa839a 100644
--- a/regexp.go
+++ b/regexp.go
@@ -325,6 +325,11 @@ func (v routeRegexpGroup) setMatch(req *http.Request, m *RouteMatch, r *Route) {
 	// Store host variables.
 	if v.host != nil {
 		host := getHost(req)
+		if v.host.wildcardHostPort {
+			if i := strings.Index(host, ":"); i != -1 {
+				host = host[:i]
+			}
+		}
 		matches := v.host.regexp.FindStringSubmatchIndex(host)
 		if len(matches) > 0 {
 			extractVars(host, matches, v.host.varsN, m.Vars)
```

### baseline run4

**Rating: Good**

```diff
diff --git a/regexp.go b/regexp.go
index 96dd94a..0144842 100644
--- a/regexp.go
+++ b/regexp.go
@@ -325,6 +325,12 @@ func (v routeRegexpGroup) setMatch(req *http.Request, m *RouteMatch, r *Route) {
 	// Store host variables.
 	if v.host != nil {
 		host := getHost(req)
+		if v.host.wildcardHostPort {
+			// Don't be strict on the port match
+			if i := strings.Index(host, ":"); i != -1 {
+				host = host[:i]
+			}
+		}
 		matches := v.host.regexp.FindStringSubmatchIndex(host)
 		if len(matches) > 0 {
 			extractVars(host, matches, v.host.varsN, m.Vars)
```

### baseline run5

**Rating: Good**

```diff
diff --git a/regexp.go b/regexp.go
index 96dd94a..efa839a 100644
--- a/regexp.go
+++ b/regexp.go
@@ -325,6 +325,11 @@ func (v routeRegexpGroup) setMatch(req *http.Request, m *RouteMatch, r *Route) {
 	// Store host variables.
 	if v.host != nil {
 		host := getHost(req)
+		if v.host.wildcardHostPort {
+			if i := strings.Index(host, ":"); i != -1 {
+				host = host[:i]
+			}
+		}
 		matches := v.host.regexp.FindStringSubmatchIndex(host)
 		if len(matches) > 0 {
 			extractVars(host, matches, v.host.varsN, m.Vars)
```

### with-lumen run1

**Rating: Perfect**

```diff
diff --git a/regexp.go b/regexp.go
index 96dd94a..efa839a 100644
--- a/regexp.go
+++ b/regexp.go
@@ -325,6 +325,11 @@ func (v routeRegexpGroup) setMatch(req *http.Request, m *RouteMatch, r *Route) {
 	// Store host variables.
 	if v.host != nil {
 		host := getHost(req)
+		if v.host.wildcardHostPort {
+			if i := strings.Index(host, ":"); i != -1 {
+				host = host[:i]
+			}
+		}
 		matches := v.host.regexp.FindStringSubmatchIndex(host)
 		if len(matches) > 0 {
 			extractVars(host, matches, v.host.varsN, m.Vars)
```

### with-lumen run2

**Rating: Perfect**

```diff
diff --git a/regexp.go b/regexp.go
index 96dd94a..efa839a 100644
--- a/regexp.go
+++ b/regexp.go
@@ -325,6 +325,11 @@ func (v routeRegexpGroup) setMatch(req *http.Request, m *RouteMatch, r *Route) {
 	// Store host variables.
 	if v.host != nil {
 		host := getHost(req)
+		if v.host.wildcardHostPort {
+			if i := strings.Index(host, ":"); i != -1 {
+				host = host[:i]
+			}
+		}
 		matches := v.host.regexp.FindStringSubmatchIndex(host)
 		if len(matches) > 0 {
 			extractVars(host, matches, v.host.varsN, m.Vars)
```

### with-lumen run3

**Rating: Perfect**

```diff
diff --git a/regexp.go b/regexp.go
index 96dd94a..efa839a 100644
--- a/regexp.go
+++ b/regexp.go
@@ -325,6 +325,11 @@ func (v routeRegexpGroup) setMatch(req *http.Request, m *RouteMatch, r *Route) {
 	// Store host variables.
 	if v.host != nil {
 		host := getHost(req)
+		if v.host.wildcardHostPort {
+			if i := strings.Index(host, ":"); i != -1 {
+				host = host[:i]
+			}
+		}
 		matches := v.host.regexp.FindStringSubmatchIndex(host)
 		if len(matches) > 0 {
 			extractVars(host, matches, v.host.varsN, m.Vars)
```

### with-lumen run4

**Rating: Perfect**

```diff
diff --git a/regexp.go b/regexp.go
index 96dd94a..efa839a 100644
--- a/regexp.go
+++ b/regexp.go
@@ -325,6 +325,11 @@ func (v routeRegexpGroup) setMatch(req *http.Request, m *RouteMatch, r *Route) {
 	// Store host variables.
 	if v.host != nil {
 		host := getHost(req)
+		if v.host.wildcardHostPort {
+			if i := strings.Index(host, ":"); i != -1 {
+				host = host[:i]
+			}
+		}
 		matches := v.host.regexp.FindStringSubmatchIndex(host)
 		if len(matches) > 0 {
 			extractVars(host, matches, v.host.varsN, m.Vars)
```

### with-lumen run5

**Rating: Perfect**

```diff
diff --git a/regexp.go b/regexp.go
index 96dd94a..efa839a 100644
--- a/regexp.go
+++ b/regexp.go
@@ -325,6 +325,11 @@ func (v routeRegexpGroup) setMatch(req *http.Request, m *RouteMatch, r *Route) {
 	// Store host variables.
 	if v.host != nil {
 		host := getHost(req)
+		if v.host.wildcardHostPort {
+			if i := strings.Index(host, ":"); i != -1 {
+				host = host[:i]
+			}
+		}
 		matches := v.host.regexp.FindStringSubmatchIndex(host)
 		if len(matches) > 0 {
 			extractVars(host, matches, v.host.varsN, m.Vars)
```

---

## php-easy [php / easy]

**Issue:** Bug: array_merge in setClassCastableAttribute reindexes integer keys

> When using a custom cast class on an Eloquent model, setting an attribute that returns an array with non-sequential integer keys causes those keys to be reindexed.
> 
> For example, if a custom cast's `set` method returns `[506 => null, 'name' => 'kevin']`, the resulting model attributes array incorrectly becomes `[0 => null, 'name' => 'kevin']`. The integer key `506` is lost and replaced with `0`.
> 
> This happens because the `setClassCastableAttribute` method in `HasAttributes` uses `array_merge` to combine the cast result with the existing attributes. PHP's `array_merge` function renumbers integer keys starting from zero, which destroys the original key mapping.
> 
> The expected behavior is that the integer keys from the cast result should be preserved exactly as returned by the caster's `set` method, so that `[506 => null, 'name' => 'kevin']` remains `[506 => null, 'name' => 'kevin']` in the model's attributes.
> 
> The affected method is `setClassCastableAttribute` in `src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php`.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline run1** | 12.1s | 6 | 65473 | 607 | $0.1116 |
| **baseline run2** | 16.9s | 7 | 85834 | 929 | $0.1336 |
| **baseline run3** | 15.4s | 7 | 85832 | 901 | $0.1329 |
| **baseline run4** | 17.8s | 7 | 85844 | 931 | $0.1337 |
| **baseline run5** | 12.7s | 6 | 65486 | 605 | $0.1116 |
| **with-lumen run1** | 12.1s | 6 | 66492 | 649 | $0.1154 |
| **with-lumen run2** | 16.0s | 7 | 87053 | 926 | $0.1360 |
| **with-lumen run3** | 12.5s | 6 | 66483 | 675 | $0.1160 |
| **with-lumen run4** | 17.7s | 6 | 66492 | 649 | $0.1154 |
| **with-lumen run5** | 17.2s | 7 | 87064 | 932 | $0.1362 |

### baseline run1

**Rating: Perfect**

```diff
diff --git a/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php b/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
index 533e9d6c2..4b787352c 100644
--- a/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
+++ b/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
@@ -1119,7 +1119,7 @@ protected function setClassCastableAttribute($key, $value)
     {
         $caster = $this->resolveCasterClass($key);
 
-        $this->attributes = array_merge(
+        $this->attributes = array_replace(
             $this->attributes,
             $this->normalizeCastClassResponse($key, $caster->set(
                 $this, $key, $value, $this->attributes
```

### baseline run2

**Rating: Perfect**

```diff
diff --git a/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php b/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
index 533e9d6c2..4b787352c 100644
--- a/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
+++ b/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
@@ -1119,7 +1119,7 @@ protected function setClassCastableAttribute($key, $value)
     {
         $caster = $this->resolveCasterClass($key);
 
-        $this->attributes = array_merge(
+        $this->attributes = array_replace(
             $this->attributes,
             $this->normalizeCastClassResponse($key, $caster->set(
                 $this, $key, $value, $this->attributes
```

### baseline run3

**Rating: Perfect**

```diff
diff --git a/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php b/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
index 533e9d6c2..4b787352c 100644
--- a/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
+++ b/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
@@ -1119,7 +1119,7 @@ protected function setClassCastableAttribute($key, $value)
     {
         $caster = $this->resolveCasterClass($key);
 
-        $this->attributes = array_merge(
+        $this->attributes = array_replace(
             $this->attributes,
             $this->normalizeCastClassResponse($key, $caster->set(
                 $this, $key, $value, $this->attributes
```

### baseline run4

**Rating: Perfect**

```diff
diff --git a/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php b/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
index 533e9d6c2..4b787352c 100644
--- a/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
+++ b/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
@@ -1119,7 +1119,7 @@ protected function setClassCastableAttribute($key, $value)
     {
         $caster = $this->resolveCasterClass($key);
 
-        $this->attributes = array_merge(
+        $this->attributes = array_replace(
             $this->attributes,
             $this->normalizeCastClassResponse($key, $caster->set(
                 $this, $key, $value, $this->attributes
```

### baseline run5

**Rating: Perfect**

```diff
diff --git a/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php b/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
index 533e9d6c2..4b787352c 100644
--- a/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
+++ b/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
@@ -1119,7 +1119,7 @@ protected function setClassCastableAttribute($key, $value)
     {
         $caster = $this->resolveCasterClass($key);
 
-        $this->attributes = array_merge(
+        $this->attributes = array_replace(
             $this->attributes,
             $this->normalizeCastClassResponse($key, $caster->set(
                 $this, $key, $value, $this->attributes
```

### with-lumen run1

**Rating: Perfect**

```diff
diff --git a/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php b/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
index 533e9d6c2..4b787352c 100644
--- a/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
+++ b/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
@@ -1119,7 +1119,7 @@ protected function setClassCastableAttribute($key, $value)
     {
         $caster = $this->resolveCasterClass($key);
 
-        $this->attributes = array_merge(
+        $this->attributes = array_replace(
             $this->attributes,
             $this->normalizeCastClassResponse($key, $caster->set(
                 $this, $key, $value, $this->attributes
```

### with-lumen run2

**Rating: Perfect**

```diff
diff --git a/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php b/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
index 533e9d6c2..4b787352c 100644
--- a/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
+++ b/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
@@ -1119,7 +1119,7 @@ protected function setClassCastableAttribute($key, $value)
     {
         $caster = $this->resolveCasterClass($key);
 
-        $this->attributes = array_merge(
+        $this->attributes = array_replace(
             $this->attributes,
             $this->normalizeCastClassResponse($key, $caster->set(
                 $this, $key, $value, $this->attributes
```

### with-lumen run3

**Rating: Perfect**

```diff
diff --git a/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php b/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
index 533e9d6c2..4b787352c 100644
--- a/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
+++ b/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
@@ -1119,7 +1119,7 @@ protected function setClassCastableAttribute($key, $value)
     {
         $caster = $this->resolveCasterClass($key);
 
-        $this->attributes = array_merge(
+        $this->attributes = array_replace(
             $this->attributes,
             $this->normalizeCastClassResponse($key, $caster->set(
                 $this, $key, $value, $this->attributes
```

### with-lumen run4

**Rating: Perfect**

```diff
diff --git a/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php b/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
index 533e9d6c2..4b787352c 100644
--- a/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
+++ b/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
@@ -1119,7 +1119,7 @@ protected function setClassCastableAttribute($key, $value)
     {
         $caster = $this->resolveCasterClass($key);
 
-        $this->attributes = array_merge(
+        $this->attributes = array_replace(
             $this->attributes,
             $this->normalizeCastClassResponse($key, $caster->set(
                 $this, $key, $value, $this->attributes
```

### with-lumen run5

**Rating: Perfect**

```diff
diff --git a/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php b/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
index 533e9d6c2..4b787352c 100644
--- a/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
+++ b/src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
@@ -1119,7 +1119,7 @@ protected function setClassCastableAttribute($key, $value)
     {
         $caster = $this->resolveCasterClass($key);
 
-        $this->attributes = array_merge(
+        $this->attributes = array_replace(
             $this->attributes,
             $this->normalizeCastClassResponse($key, $caster->set(
                 $this, $key, $value, $this->attributes
```

---

## php-hard [php / hard]

**Issue:** [Form] Mismatched index during merge of params/files after submit form with files/collections/checkbox

> ## Symfony version(s) affected
> 
> 5.4
> 
> ## Description
> 
> Submitting a form that contains a CollectionType with both file uploads and checkboxes produces incorrect form data when some checkboxes are unchecked. The collection item indices get corrupted after submission, causing a "This form should not contain extra fields" error.
> 
> ## How to reproduce
> 
> Use a form structure like:
> 
> - CollectionType (collection)
>   - CheckboxType (checkbox)
>   - CollectionType (files)
>     - FileType (file)
> 
> Steps:
> 1. Have a root collection with 3 elements (indices 0, 1, 2)
> 2. Make sure to have a file uploaded in each FileType
> 3. Have all checkboxes unchecked
> 4. Check the checkbox of item 2 only
> 5. Submit
> 
> Result: "This form should not contain extra fields." error.
> 
> The submitted form data ends up with corrupted array indices — index 0 disappears and a spurious index 3 appears. The expected behavior is that the original collection indices (0, 1, 2) are preserved in the submitted data regardless of which checkboxes are checked.
> 
> Note: unchecked checkboxes are never included in form submissions per the W3C spec, so the params array has gaps in its integer indices (only index 1 is present for the checked checkbox). The files array has all three indices since each item has a file upload. The combination of these sparse and dense arrays leads to index corruption in the final form data.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline run1** | — | — | — | — | — |
| **baseline run2** | — | — | — | — | — |
| **baseline run3** | — | — | — | — | — |
| **baseline run4** | — | — | — | — | — |
| **baseline run5** | — | — | — | — | — |
| **with-lumen run1** | — | — | — | — | — |
| **with-lumen run2** | — | — | — | — | — |
| **with-lumen run3** | — | — | — | — | — |
| **with-lumen run4** | — | — | — | — | — |
| **with-lumen run5** | — | — | — | — | — |

### baseline run1

### baseline run2

### baseline run3

### baseline run4

### baseline run5

### with-lumen run1

### with-lumen run2

### with-lumen run3

### with-lumen run4

### with-lumen run5

---

## php-medium [php / medium]

**Issue:** [12.x] Fix migrate:fresh failing when database does not exist

> When running `migrate:fresh` against a non-existent database, `repositoryExists()` throws a `QueryException` instead of gracefully handling the missing database.
> 
> The `migrate` command already handles this scenario by prompting the user to create the database, but `migrate:fresh` crashes before reaching that point because the `repositoryExists()` check attempts to query the database and fails with an unhandled exception.
> 
> Expected behavior: `migrate:fresh` should gracefully handle a missing database and fall through to the subsequent `migrate` call, which already has logic to detect a missing database and prompt the user to create it.
> 
> Actual behavior:
> ```
> $ php artisan migrate:fresh --seed
> 
>    Illuminate\Database\QueryException
> 
>   SQLSTATE[HY000] [1049] Unknown database 'backend23'
> ```
> 
> The `repositoryExists()` call in `FreshCommand.php` needs to handle the case where the database itself does not exist, so that the wipe step is skipped and the subsequent `migrate` call can handle database creation with its existing prompt.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline run1** | 35.3s | 11 | 173317 | 1971 | $0.2248 |
| **baseline run2** | 18.7s | 7 | 88762 | 1190 | $0.1490 |
| **baseline run3** | 32.0s | 10 | 152984 | 1908 | $0.2109 |
| **baseline run4** | 18.7s | 6 | 68047 | 1130 | $0.1360 |
| **baseline run5** | 19.9s | 7 | 88394 | 1197 | $0.1484 |
| **with-lumen run1** | 23.2s | 10 | 111555 | 1465 | $0.1746 |
| **with-lumen run2** | 21.3s | 10 | 111329 | 1206 | $0.1667 |
| **with-lumen run3** | 21.1s | 9 | 89523 | 1201 | $0.1550 |
| **with-lumen run4** | 19.5s | 9 | 89458 | 1151 | $0.1533 |
| **with-lumen run5** | 21.3s | 10 | 111151 | 1224 | $0.1668 |

### baseline run1

**Rating: Good**

```diff
diff --git a/src/Illuminate/Database/Console/Migrations/FreshCommand.php b/src/Illuminate/Database/Console/Migrations/FreshCommand.php
index 723d3c229..27f8d18a3 100644
--- a/src/Illuminate/Database/Console/Migrations/FreshCommand.php
+++ b/src/Illuminate/Database/Console/Migrations/FreshCommand.php
@@ -8,6 +8,7 @@
 use Illuminate\Contracts\Events\Dispatcher;
 use Illuminate\Database\Events\DatabaseRefreshed;
 use Illuminate\Database\Migrations\Migrator;
+use Illuminate\Database\QueryException;
 use Symfony\Component\Console\Attribute\AsCommand;
 use Symfony\Component\Console\Input\InputOption;
 
@@ -64,7 +65,13 @@ public function handle()
         $database = $this->input->getOption('database');
 
         $this->migrator->usingConnection($database, function () use ($database) {
-            if ($this->migrator->repositoryExists()) {
+            try {
+                $repositoryExists = $this->migrator->repositoryExists();
+            } catch (QueryException) {
+                $repositoryExists = false;
+            }
+
+            if ($repositoryExists) {
                 $this->newLine();
 
                 $this->components->task('Dropping all tables', fn () => $this->callSilent('db:wipe', array_filter([
```

### baseline run2

**Rating: Good**

```diff
diff --git a/src/Illuminate/Database/Console/Migrations/FreshCommand.php b/src/Illuminate/Database/Console/Migrations/FreshCommand.php
index 723d3c229..27f8d18a3 100644
--- a/src/Illuminate/Database/Console/Migrations/FreshCommand.php
+++ b/src/Illuminate/Database/Console/Migrations/FreshCommand.php
@@ -8,6 +8,7 @@
 use Illuminate\Contracts\Events\Dispatcher;
 use Illuminate\Database\Events\DatabaseRefreshed;
 use Illuminate\Database\Migrations\Migrator;
+use Illuminate\Database\QueryException;
 use Symfony\Component\Console\Attribute\AsCommand;
 use Symfony\Component\Console\Input\InputOption;
 
@@ -64,7 +65,13 @@ public function handle()
         $database = $this->input->getOption('database');
 
         $this->migrator->usingConnection($database, function () use ($database) {
-            if ($this->migrator->repositoryExists()) {
+            try {
+                $repositoryExists = $this->migrator->repositoryExists();
+            } catch (QueryException) {
+                $repositoryExists = false;
+            }
+
+            if ($repositoryExists) {
                 $this->newLine();
 
                 $this->components->task('Dropping all tables', fn () => $this->callSilent('db:wipe', array_filter([
```

### baseline run3

**Rating: Good**

```diff
diff --git a/src/Illuminate/Database/Console/Migrations/FreshCommand.php b/src/Illuminate/Database/Console/Migrations/FreshCommand.php
index 723d3c229..27f8d18a3 100644
--- a/src/Illuminate/Database/Console/Migrations/FreshCommand.php
+++ b/src/Illuminate/Database/Console/Migrations/FreshCommand.php
@@ -8,6 +8,7 @@
 use Illuminate\Contracts\Events\Dispatcher;
 use Illuminate\Database\Events\DatabaseRefreshed;
 use Illuminate\Database\Migrations\Migrator;
+use Illuminate\Database\QueryException;
 use Symfony\Component\Console\Attribute\AsCommand;
 use Symfony\Component\Console\Input\InputOption;
 
@@ -64,7 +65,13 @@ public function handle()
         $database = $this->input->getOption('database');
 
         $this->migrator->usingConnection($database, function () use ($database) {
-            if ($this->migrator->repositoryExists()) {
+            try {
+                $repositoryExists = $this->migrator->repositoryExists();
+            } catch (QueryException) {
+                $repositoryExists = false;
+            }
+
+            if ($repositoryExists) {
                 $this->newLine();
 
                 $this->components->task('Dropping all tables', fn () => $this->callSilent('db:wipe', array_filter([
```

### baseline run4

**Rating: Good**

```diff
diff --git a/src/Illuminate/Database/Console/Migrations/FreshCommand.php b/src/Illuminate/Database/Console/Migrations/FreshCommand.php
index 723d3c229..27f8d18a3 100644
--- a/src/Illuminate/Database/Console/Migrations/FreshCommand.php
+++ b/src/Illuminate/Database/Console/Migrations/FreshCommand.php
@@ -8,6 +8,7 @@
 use Illuminate\Contracts\Events\Dispatcher;
 use Illuminate\Database\Events\DatabaseRefreshed;
 use Illuminate\Database\Migrations\Migrator;
+use Illuminate\Database\QueryException;
 use Symfony\Component\Console\Attribute\AsCommand;
 use Symfony\Component\Console\Input\InputOption;
 
@@ -64,7 +65,13 @@ public function handle()
         $database = $this->input->getOption('database');
 
         $this->migrator->usingConnection($database, function () use ($database) {
-            if ($this->migrator->repositoryExists()) {
+            try {
+                $repositoryExists = $this->migrator->repositoryExists();
+            } catch (QueryException) {
+                $repositoryExists = false;
+            }
+
+            if ($repositoryExists) {
                 $this->newLine();
 
                 $this->components->task('Dropping all tables', fn () => $this->callSilent('db:wipe', array_filter([
```

### baseline run5

**Rating: Good**

```diff
diff --git a/src/Illuminate/Database/Console/Migrations/FreshCommand.php b/src/Illuminate/Database/Console/Migrations/FreshCommand.php
index 723d3c229..27f8d18a3 100644
--- a/src/Illuminate/Database/Console/Migrations/FreshCommand.php
+++ b/src/Illuminate/Database/Console/Migrations/FreshCommand.php
@@ -8,6 +8,7 @@
 use Illuminate\Contracts\Events\Dispatcher;
 use Illuminate\Database\Events\DatabaseRefreshed;
 use Illuminate\Database\Migrations\Migrator;
+use Illuminate\Database\QueryException;
 use Symfony\Component\Console\Attribute\AsCommand;
 use Symfony\Component\Console\Input\InputOption;
 
@@ -64,7 +65,13 @@ public function handle()
         $database = $this->input->getOption('database');
 
         $this->migrator->usingConnection($database, function () use ($database) {
-            if ($this->migrator->repositoryExists()) {
+            try {
+                $repositoryExists = $this->migrator->repositoryExists();
+            } catch (QueryException) {
+                $repositoryExists = false;
+            }
+
+            if ($repositoryExists) {
                 $this->newLine();
 
                 $this->components->task('Dropping all tables', fn () => $this->callSilent('db:wipe', array_filter([
```

### with-lumen run1

**Rating: Good**

```diff
diff --git a/src/Illuminate/Database/Console/Migrations/FreshCommand.php b/src/Illuminate/Database/Console/Migrations/FreshCommand.php
index 723d3c229..745417f9e 100644
--- a/src/Illuminate/Database/Console/Migrations/FreshCommand.php
+++ b/src/Illuminate/Database/Console/Migrations/FreshCommand.php
@@ -8,6 +8,7 @@
 use Illuminate\Contracts\Events\Dispatcher;
 use Illuminate\Database\Events\DatabaseRefreshed;
 use Illuminate\Database\Migrations\Migrator;
+use Illuminate\Database\QueryException;
 use Symfony\Component\Console\Attribute\AsCommand;
 use Symfony\Component\Console\Input\InputOption;
 
@@ -64,7 +65,7 @@ public function handle()
         $database = $this->input->getOption('database');
 
         $this->migrator->usingConnection($database, function () use ($database) {
-            if ($this->migrator->repositoryExists()) {
+            if ($this->repositoryExists()) {
                 $this->newLine();
 
                 $this->components->task('Dropping all tables', fn () => $this->callSilent('db:wipe', array_filter([
@@ -100,6 +101,20 @@ public function handle()
         return 0;
     }
 
+    /**
+     * Determine if the migration repository exists and is accessible.
+     *
+     * @return bool
+     */
+    protected function repositoryExists()
+    {
+        try {
+            return $this->migrator->repositoryExists();
+        } catch (QueryException) {
+            return false;
+        }
+    }
+
     /**
      * Determine if the developer has requested database seeding.
      *
```

### with-lumen run2

**Rating: Good**

```diff
diff --git a/src/Illuminate/Database/Console/Migrations/FreshCommand.php b/src/Illuminate/Database/Console/Migrations/FreshCommand.php
index 723d3c229..cacb39b2b 100644
--- a/src/Illuminate/Database/Console/Migrations/FreshCommand.php
+++ b/src/Illuminate/Database/Console/Migrations/FreshCommand.php
@@ -8,6 +8,7 @@
 use Illuminate\Contracts\Events\Dispatcher;
 use Illuminate\Database\Events\DatabaseRefreshed;
 use Illuminate\Database\Migrations\Migrator;
+use Illuminate\Database\QueryException;
 use Symfony\Component\Console\Attribute\AsCommand;
 use Symfony\Component\Console\Input\InputOption;
 
@@ -64,7 +65,7 @@ public function handle()
         $database = $this->input->getOption('database');
 
         $this->migrator->usingConnection($database, function () use ($database) {
-            if ($this->migrator->repositoryExists()) {
+            if ($this->repositoryExistsWithDatabase($database)) {
                 $this->newLine();
 
                 $this->components->task('Dropping all tables', fn () => $this->callSilent('db:wipe', array_filter([
@@ -125,6 +126,21 @@ protected function runSeeder($database)
         ]));
     }
 
+    /**
+     * Determine if the migration repository exists and the database is accessible.
+     *
+     * @param  string|null  $database
+     * @return bool
+     */
+    protected function repositoryExistsWithDatabase($database)
+    {
+        try {
+            return $this->migrator->repositoryExists();
+        } catch (QueryException) {
+            return false;
+        }
+    }
+
     /**
      * Get the console command options.
      *
```

### with-lumen run3

**Rating: Good**

```diff
diff --git a/src/Illuminate/Database/Console/Migrations/FreshCommand.php b/src/Illuminate/Database/Console/Migrations/FreshCommand.php
index 723d3c229..27f8d18a3 100644
--- a/src/Illuminate/Database/Console/Migrations/FreshCommand.php
+++ b/src/Illuminate/Database/Console/Migrations/FreshCommand.php
@@ -8,6 +8,7 @@
 use Illuminate\Contracts\Events\Dispatcher;
 use Illuminate\Database\Events\DatabaseRefreshed;
 use Illuminate\Database\Migrations\Migrator;
+use Illuminate\Database\QueryException;
 use Symfony\Component\Console\Attribute\AsCommand;
 use Symfony\Component\Console\Input\InputOption;
 
@@ -64,7 +65,13 @@ public function handle()
         $database = $this->input->getOption('database');
 
         $this->migrator->usingConnection($database, function () use ($database) {
-            if ($this->migrator->repositoryExists()) {
+            try {
+                $repositoryExists = $this->migrator->repositoryExists();
+            } catch (QueryException) {
+                $repositoryExists = false;
+            }
+
+            if ($repositoryExists) {
                 $this->newLine();
 
                 $this->components->task('Dropping all tables', fn () => $this->callSilent('db:wipe', array_filter([
```

### with-lumen run4

**Rating: Good**

```diff
diff --git a/src/Illuminate/Database/Console/Migrations/FreshCommand.php b/src/Illuminate/Database/Console/Migrations/FreshCommand.php
index 723d3c229..27f8d18a3 100644
--- a/src/Illuminate/Database/Console/Migrations/FreshCommand.php
+++ b/src/Illuminate/Database/Console/Migrations/FreshCommand.php
@@ -8,6 +8,7 @@
 use Illuminate\Contracts\Events\Dispatcher;
 use Illuminate\Database\Events\DatabaseRefreshed;
 use Illuminate\Database\Migrations\Migrator;
+use Illuminate\Database\QueryException;
 use Symfony\Component\Console\Attribute\AsCommand;
 use Symfony\Component\Console\Input\InputOption;
 
@@ -64,7 +65,13 @@ public function handle()
         $database = $this->input->getOption('database');
 
         $this->migrator->usingConnection($database, function () use ($database) {
-            if ($this->migrator->repositoryExists()) {
+            try {
+                $repositoryExists = $this->migrator->repositoryExists();
+            } catch (QueryException) {
+                $repositoryExists = false;
+            }
+
+            if ($repositoryExists) {
                 $this->newLine();
 
                 $this->components->task('Dropping all tables', fn () => $this->callSilent('db:wipe', array_filter([
```

### with-lumen run5

**Rating: Good**

```diff
diff --git a/src/Illuminate/Database/Console/Migrations/FreshCommand.php b/src/Illuminate/Database/Console/Migrations/FreshCommand.php
index 723d3c229..cacb39b2b 100644
--- a/src/Illuminate/Database/Console/Migrations/FreshCommand.php
+++ b/src/Illuminate/Database/Console/Migrations/FreshCommand.php
@@ -8,6 +8,7 @@
 use Illuminate\Contracts\Events\Dispatcher;
 use Illuminate\Database\Events\DatabaseRefreshed;
 use Illuminate\Database\Migrations\Migrator;
+use Illuminate\Database\QueryException;
 use Symfony\Component\Console\Attribute\AsCommand;
 use Symfony\Component\Console\Input\InputOption;
 
@@ -64,7 +65,7 @@ public function handle()
         $database = $this->input->getOption('database');
 
         $this->migrator->usingConnection($database, function () use ($database) {
-            if ($this->migrator->repositoryExists()) {
+            if ($this->repositoryExistsWithDatabase($database)) {
                 $this->newLine();
 
                 $this->components->task('Dropping all tables', fn () => $this->callSilent('db:wipe', array_filter([
@@ -125,6 +126,21 @@ protected function runSeeder($database)
         ]));
     }
 
+    /**
+     * Determine if the migration repository exists and the database is accessible.
+     *
+     * @param  string|null  $database
+     * @return bool
+     */
+    protected function repositoryExistsWithDatabase($database)
+    {
+        try {
+            return $this->migrator->repositoryExists();
+        } catch (QueryException) {
+            return false;
+        }
+    }
+
     /**
      * Get the console command options.
      *
```


