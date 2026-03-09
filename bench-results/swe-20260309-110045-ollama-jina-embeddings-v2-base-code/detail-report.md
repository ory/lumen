# SWE-Bench Detail Report

Generated: 2026-03-09 10:11 UTC

---

## go-easy [go / easy]

**Issue:** Trailing slash routing regression with wildcard routes

> Before v4.1.15, defining the following routes would correctly handle requests to '/articles', '/articles/', and '/articles/whatever':
> 
>   /articles
>   /articles/*
> 
> Since v4.1.15, a request to '/articles/' no longer matches the '/articles/*' wildcard route. To work around this, users must now add an explicit '/articles/' route:
> 
>   /articles
>   /articles/
>   /articles/*
> 
> This is a regression. The router should match a trailing-slash request (e.g. '/users/') against an existing any-route (e.g. '/users/*') without requiring the user to register a separate trailing-slash route.
> 
> Expected behavior: GET /users/ should match the route /users/* and return an empty wildcard parameter.
> 
> Actual behavior: GET /users/ returns no match (404) unless /users/ is explicitly registered as a separate route.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 215.4s | 610 | 4543234 | 16382 | $0.6055 |
| **with-lumen** | 154.7s | 330 | 2692926 | 12711 | $0.4036 |

### baseline

**Rating: Perfect**

```diff
diff --git a/router.go b/router.go
index 15a3398..ed728d6 100644
--- a/router.go
+++ b/router.go
@@ -355,6 +355,10 @@ func (r *Router) Find(method, path string, c Context) {
 
 		// Attempt to go back up the tree on no matching prefix or no remaining search
 		if l != pl || search == "" {
+			// Handle special case of trailing slash route with existing any route (see #1526)
+			if path[len(path)-1] == '/' && cn.findChildByKind(akind) != nil {
+				goto Any
+			}
 			if nn == nil { // Issue #1348
 				return // Not found
 			}
diff --git a/router_test.go b/router_test.go
index 8c27b9f..bada613 100644
--- a/router_test.go
+++ b/router_test.go
@@ -1286,3 +1286,75 @@ func prefix(tail bool, p, on, off string) string {
 	}
 	return fmt.Sprintf("%s%s", p, off)
 }
+
+// TestRouterMatchAnySlash shall verify finding the best route
+// for any routes with trailing slash requests
+func TestRouterMatchAnySlash(t *testing.T) {
+	e := New()
+	r := e.router
+
+	handler := func(c Context) error {
+		c.Set("path", c.Path())
+		return nil
+	}
+
+	// Routes
+	r.Add(http.MethodGet, "/users", handler)
+	r.Add(http.MethodGet, "/users/*", handler)
+	r.Add(http.MethodGet, "/img/*", handler)
+	r.Add(http.MethodGet, "/img/load", handler)
+	r.Add(http.MethodGet, "/img/load/*", handler)
+	r.Add(http.MethodGet, "/assets/*", handler)
+
+	c := e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/", c)
+	assert.Equal(t, "", c.Param("*"))
+
+	// Test trailing slash request for simple any route (see #1526)
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/users/", c)
+	c.handler(c)
+	assert.Equal(t, "/users/*", c.Get("path"))
+	assert.Equal(t, "", c.Param("*"))
+
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/users/joe", c)
+	c.handler(c)
+	assert.Equal(t, "/users/*", c.Get("path"))
+	assert.Equal(t, "joe", c.Param("*"))
+
+	// Test trailing slash request for nested any route (see #1526)
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/img/load", c)
+	c.handler(c)
+	assert.Equal(t, "/img/load", c.Get("path"))
+	assert.Equal(t, "", c.Param("*"))
+
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/img/load/", c)
+	c.handler(c)
+	assert.Equal(t, "/img/load/*", c.Get("path"))
+	assert.Equal(t, "", c.Param("*"))
+
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/img/load/ben", c)
+	c.handler(c)
+	assert.Equal(t, "/img/load/*", c.Get("path"))
+	assert.Equal(t, "ben", c.Param("*"))
+
+	// Test /assets/* any route
+	// ... without trailing slash must not match
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/assets", c)
+	c.handler(c)
+	assert.Equal(t, nil, c.Get("path"))
+	assert.Equal(t, "", c.Param("*"))
+
+	// ... with trailing slash must match
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/assets/", c)
+	c.handler(c)
+	assert.Equal(t, "/assets/*", c.Get("path"))
+	assert.Equal(t, "", c.Param("*"))
+
+}
```

### with-lumen

**Rating: INVALID (lumen not used)**

```diff
diff --git a/router.go b/router.go
index 15a3398..ed728d6 100644
--- a/router.go
+++ b/router.go
@@ -355,6 +355,10 @@ func (r *Router) Find(method, path string, c Context) {
 
 		// Attempt to go back up the tree on no matching prefix or no remaining search
 		if l != pl || search == "" {
+			// Handle special case of trailing slash route with existing any route (see #1526)
+			if path[len(path)-1] == '/' && cn.findChildByKind(akind) != nil {
+				goto Any
+			}
 			if nn == nil { // Issue #1348
 				return // Not found
 			}
diff --git a/router_test.go b/router_test.go
index 8c27b9f..175a0ac 100644
--- a/router_test.go
+++ b/router_test.go
@@ -1280,6 +1280,78 @@ func (n *node) printTree(pfx string, tail bool) {
 	}
 }
 
+// TestRouterMatchAnySlash shall verify finding the best route
+// for any routes with trailing slash requests
+func TestRouterMatchAnySlash(t *testing.T) {
+	e := New()
+	r := e.router
+
+	handler := func(c Context) error {
+		c.Set("path", c.Path())
+		return nil
+	}
+
+	// Routes
+	r.Add(http.MethodGet, "/users", handler)
+	r.Add(http.MethodGet, "/users/*", handler)
+	r.Add(http.MethodGet, "/img/*", handler)
+	r.Add(http.MethodGet, "/img/load", handler)
+	r.Add(http.MethodGet, "/img/load/*", handler)
+	r.Add(http.MethodGet, "/assets/*", handler)
+
+	c := e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/", c)
+	assert.Equal(t, "", c.Param("*"))
+
+	// Test trailing slash request for simple any route (see #1526)
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/users/", c)
+	c.handler(c)
+	assert.Equal(t, "/users/*", c.Get("path"))
+	assert.Equal(t, "", c.Param("*"))
+
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/users/joe", c)
+	c.handler(c)
+	assert.Equal(t, "/users/*", c.Get("path"))
+	assert.Equal(t, "joe", c.Param("*"))
+
+	// Test trailing slash request for nested any route (see #1526)
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/img/load", c)
+	c.handler(c)
+	assert.Equal(t, "/img/load", c.Get("path"))
+	assert.Equal(t, "", c.Param("*"))
+
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/img/load/", c)
+	c.handler(c)
+	assert.Equal(t, "/img/load/*", c.Get("path"))
+	assert.Equal(t, "", c.Param("*"))
+
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/img/load/ben", c)
+	c.handler(c)
+	assert.Equal(t, "/img/load/*", c.Get("path"))
+	assert.Equal(t, "ben", c.Param("*"))
+
+	// Test /assets/* any route
+	// ... without trailing slash must not match
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/assets", c)
+	c.handler(c)
+	assert.Equal(t, nil, c.Get("path"))
+	assert.Equal(t, "", c.Param("*"))
+
+	// ... with trailing slash must match
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/assets/", c)
+	c.handler(c)
+	assert.Equal(t, "/assets/*", c.Get("path"))
+	assert.Equal(t, "", c.Param("*"))
+
+}
+
 func prefix(tail bool, p, on, off string) string {
 	if tail {
 		return fmt.Sprintf("%s%s", p, on)
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
| **baseline** | 196.8s | 394 | 3516790 | 15778 | $0.5244 |
| **with-lumen** | 76.2s | 154 | 749679 | 5657 | $0.1358 |

### baseline

**Rating: Good**

```diff
diff --git a/internal/states/state_equal.go b/internal/states/state_equal.go
index 97414311c..d0e64c637 100644
--- a/internal/states/state_equal.go
+++ b/internal/states/state_equal.go
@@ -97,11 +97,16 @@ func (s *State) RootOutputValuesEqual(s2 *State) bool {
 		return true
 	}
 
+	if s == nil || s2 == nil {
+		// One is nil but not the other, so they're not equal
+		return false
+	}
+
 	if len(s.RootOutputValues) != len(s2.RootOutputValues) {
 		return false
 	}
 
-	for k, v1 := range s2.RootOutputValues {
+	for k, v1 := range s.RootOutputValues {
 		v2, ok := s2.RootOutputValues[k]
 		if !ok || !v1.Equal(v2) {
 			return false
diff --git a/internal/states/state_test.go b/internal/states/state_test.go
index dae25f6cd..966f101ca 100644
--- a/internal/states/state_test.go
+++ b/internal/states/state_test.go
@@ -1131,3 +1131,89 @@ func mustAbsResourceAddr(s string) addrs.AbsResource {
 	}
 	return addr
 }
+
+// TestRootOutputValuesEqual_ChangedValues verifies that RootOutputValuesEqual
+// correctly detects when output values have changed between two states.
+// This is a regression test for the bug where the method was comparing a state
+// against itself instead of comparing two different states.
+func TestRootOutputValuesEqual_ChangedValues(t *testing.T) {
+	// Create state 1 with an output value
+	state1 := NewState()
+	state1.SetOutputValue(
+		addrs.OutputValue{Name: "output1"}.Absolute(addrs.RootModuleInstance),
+		cty.StringVal("value1"), false,
+	)
+
+	// Create state 2 with the same output key but different value
+	state2 := NewState()
+	state2.SetOutputValue(
+		addrs.OutputValue{Name: "output1"}.Absolute(addrs.RootModuleInstance),
+		cty.StringVal("value2"), false,
+	)
+
+	// The two states should NOT be equal since the output values differ
+	if state1.RootOutputValuesEqual(state2) {
+		t.Errorf("expected RootOutputValuesEqual to return false when values differ, but it returned true")
+	}
+
+	// Also verify the reverse comparison
+	if state2.RootOutputValuesEqual(state1) {
+		t.Errorf("expected RootOutputValuesEqual to return false when values differ (reverse), but it returned true")
+	}
+}
+
+// TestRootOutputValuesEqual_SameValues verifies that RootOutputValuesEqual
+// correctly returns true when output values are the same.
+func TestRootOutputValuesEqual_SameValues(t *testing.T) {
+	// Create state 1 with an output value
+	state1 := NewState()
+	state1.SetOutputValue(
+		addrs.OutputValue{Name: "output1"}.Absolute(addrs.RootModuleInstance),
+		cty.StringVal("value1"), false,
+	)
+
+	// Create state 2 with the same output value
+	state2 := NewState()
+	state2.SetOutputValue(
+		addrs.OutputValue{Name: "output1"}.Absolute(addrs.RootModuleInstance),
+		cty.StringVal("value1"), false,
+	)
+
+	// The two states should be equal since the output values are the same
+	if !state1.RootOutputValuesEqual(state2) {
+		t.Errorf("expected RootOutputValuesEqual to return true when values are the same, but it returned false")
+	}
+
+	// Also verify the reverse comparison
+	if !state2.RootOutputValuesEqual(state1) {
+		t.Errorf("expected RootOutputValuesEqual to return true when values are the same (reverse), but it returned false")
+	}
+}
+
+// TestRootOutputValuesEqual_NilStates verifies that RootOutputValuesEqual
+// correctly handles nil states.
+func TestRootOutputValuesEqual_NilStates(t *testing.T) {
+	var state1 *State
+	var state2 *State
+
+	// Both nil should be equal
+	if !state1.RootOutputValuesEqual(state2) {
+		t.Errorf("expected nil states to be equal")
+	}
+
+	// Create state2 with a value
+	state2 = NewState()
+	state2.SetOutputValue(
+		addrs.OutputValue{Name: "output1"}.Absolute(addrs.RootModuleInstance),
+		cty.StringVal("value1"), false,
+	)
+
+	// One nil, one not nil should not be equal
+	if state1.RootOutputValuesEqual(state2) {
+		t.Errorf("expected nil and non-nil states to not be equal")
+	}
+
+	if state2.RootOutputValuesEqual(state1) {
+		t.Errorf("expected non-nil and nil states to not be equal")
+	}
+}
```

### with-lumen

**Rating: INVALID (lumen not used)**

```diff
diff --git a/internal/states/state_equal.go b/internal/states/state_equal.go
index 97414311c..1b8d2ff3a 100644
--- a/internal/states/state_equal.go
+++ b/internal/states/state_equal.go
@@ -97,11 +97,15 @@ func (s *State) RootOutputValuesEqual(s2 *State) bool {
 		return true
 	}
 
+	if s == nil || s2 == nil {
+		return false
+	}
+
 	if len(s.RootOutputValues) != len(s2.RootOutputValues) {
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
| **baseline** | 141.8s | 386 | 2259617 | 12517 | $0.3344 |
| **with-lumen** | 128.3s | 322 | 2272959 | 9672 | $0.3430 |

### baseline

**Rating: Good**

```diff
diff --git a/mux_test.go b/mux_test.go
index 2d8d2b3..318c8e1 100644
--- a/mux_test.go
+++ b/mux_test.go
@@ -461,6 +461,56 @@ func TestPath(t *testing.T) {
 	}
 }
 
+func TestWildcardHostWithPort(t *testing.T) {
+	// Test for issue where route variables are empty when a request includes a port
+	// and the route template does not specify one.
+	r := NewRouter()
+
+	// Register routes with and without port
+	r.Host("{subdomain:.+}.localhost").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
+		vars := Vars(req)
+		if subdomain, ok := vars["subdomain"]; !ok || subdomain != "test" {
+			w.WriteHeader(http.StatusInternalServerError)
+			w.Write([]byte("subdomain missing or wrong"))
+		} else {
+			w.WriteHeader(http.StatusOK)
+			w.Write([]byte("ok"))
+		}
+	})
+
+	r.Host("{subdomain:.+}.localhost:{port:.*}").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
+		vars := Vars(req)
+		if subdomain, ok := vars["subdomain"]; !ok || subdomain != "test" {
+			w.WriteHeader(http.StatusInternalServerError)
+			w.Write([]byte("subdomain missing or wrong"))
+		} else if port, ok := vars["port"]; !ok || port != "8080" {
+			w.WriteHeader(http.StatusInternalServerError)
+			w.Write([]byte("port missing or wrong"))
+		} else {
+			w.WriteHeader(http.StatusOK)
+			w.Write([]byte("ok"))
+		}
+	})
+
+	// Test case 1: Request without port should match first route
+	req1 := httptest.NewRequest("GET", "/", nil)
+	req1.Host = "test.localhost"
+	rr1 := httptest.NewRecorder()
+	r.ServeHTTP(rr1, req1)
+	if rr1.Code != http.StatusOK {
+		t.Errorf("Request without port failed: expected 200, got %d: %s", rr1.Code, rr1.Body.String())
+	}
+
+	// Test case 2: Request with port should match second route
+	req2 := httptest.NewRequest("GET", "/", nil)
+	req2.Host = "test.localhost:8080"
+	rr2 := httptest.NewRecorder()
+	r.ServeHTTP(rr2, req2)
+	if rr2.Code != http.StatusOK {
+		t.Errorf("Request with port failed: expected 200, got %d: %s", rr2.Code, rr2.Body.String())
+	}
+}
+
 func TestPathPrefix(t *testing.T) {
 	tests := []routeTest{
 		{
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

### with-lumen

**Rating: Good**

```diff
diff --git a/mux_test.go b/mux_test.go
index 2d8d2b3..251fbd1 100644
--- a/mux_test.go
+++ b/mux_test.go
@@ -2924,3 +2924,29 @@ func newRequestHost(method, url, host string) *http.Request {
 	req.Host = host
 	return req
 }
+
+// TestWildcardHostWithPort tests that variables are correctly extracted
+// when a wildcard host route matches a request with a port, even when
+// the route doesn't explicitly specify a port pattern.
+func TestWildcardHostWithPort(t *testing.T) {
+	var vars map[string]string
+	r := NewRouter()
+	// Add routes in specific order to reproduce the issue
+	r.Host("{subdomain:.+}.localhost").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
+		vars = Vars(r)
+	})
+	r.Host("{subdomain:.+}.localhost:{port:.*}").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
+
+	// Request with port should match first route and extract subdomain
+	req := newRequestHost("GET", "/", "test.localhost:8080")
+	rec := NewRecorder()
+	r.ServeHTTP(rec, req)
+
+	if vars == nil {
+		t.Fatal("Handler was not called")
+	}
+
+	if vars["subdomain"] != "test" {
+		t.Errorf("Expected subdomain=test, got %v. All vars: %v", vars["subdomain"], vars)
+	}
+}
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


