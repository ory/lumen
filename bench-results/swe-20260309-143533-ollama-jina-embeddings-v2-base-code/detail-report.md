# SWE-Bench Detail Report

Generated: 2026-03-09 13:56 UTC

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
| **baseline** | 130.7s | 362 | 2860033 | 9951 | $0.4016 |
| **with-lumen** | 204.3s | 460 | 3153752 | 17254 | $0.4759 |

### baseline

**Rating: Good**

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
index 8c27b9f..f86cf02 100644
--- a/router_test.go
+++ b/router_test.go
@@ -1265,6 +1265,40 @@ func BenchmarkRouterGooglePlusAPI(b *testing.B) {
 	benchmarkRouterRoutes(b, googlePlusAPI)
 }
 
+// Issue #1526 - Trailing slash routing regression with wildcard routes
+func TestRouterTrailingSlashWildcard(t *testing.T) {
+	e := New()
+	r := e.router
+
+	// Routes
+	r.Add(http.MethodGet, "/articles", handlerHelper("a", 1))
+	r.Add(http.MethodGet, "/articles/*", handlerHelper("b", 2))
+
+	c := e.NewContext(nil, nil).(*context)
+
+	// Test /articles
+	r.Find(http.MethodGet, "/articles", c)
+	c.handler(c)
+	assert.Equal(t, 1, c.Get("a"))
+	assert.Equal(t, "/articles", c.Get("path"))
+
+	// Test /articles/
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/articles/", c)
+	c.handler(c)
+	assert.Equal(t, 2, c.Get("b"))
+	assert.Equal(t, "/articles/*", c.Get("path"))
+	assert.Equal(t, "", c.Param("*"))
+
+	// Test /articles/whatever
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/articles/whatever", c)
+	c.handler(c)
+	assert.Equal(t, 2, c.Get("b"))
+	assert.Equal(t, "/articles/*", c.Get("path"))
+	assert.Equal(t, "whatever", c.Param("*"))
+}
+
 func (n *node) printTree(pfx string, tail bool) {
 	p := prefix(tail, pfx, "└── ", "├── ")
 	fmt.Printf("%s%s, %p: type=%d, parent=%p, handler=%v, pnames=%v\n", p, n.prefix, n, n.kind, n.parent, n.methodHandler, n.pnames)
```

### with-lumen

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
index 8c27b9f..cb628df 100644
--- a/router_test.go
+++ b/router_test.go
@@ -942,6 +942,78 @@ func TestRouterPriorityNotFound(t *testing.T) {
 	assert.Equal(t, http.StatusNotFound, he.Code)
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
 func TestRouterParamNames(t *testing.T) {
 	e := New()
 	r := e.router
```

---

## go-hard [go / hard]

**Issue:** Decoder overrides defaults with null values

> The decoder overrides the default values of a nested struct with zero-initialised values when the YAML node is null or contains only comments.
> 
> Given the following Go types:
> 
> ```go
> type Default struct {
>     Key string `yaml:"key"`
> }
> 
> type Test struct {
>     First    string  `yaml:"first"`
>     Defaults Default `yaml:"second"`
> }
> ```
> 
> And the following YAML input where the nested struct key is present but has no actual values (only comments):
> 
> ```yaml
> first: "Test"
> second:
>   # Just want to document the fields of the nested struct
> #  key: "Value from YAML"
> ```
> 
> When decoding into a pre-initialized struct with default values:
> 
> ```go
> test := Test{Defaults: Default{Key: "My default Value"}}
> d := yaml.NewDecoder(strings.NewReader(yml))
> err := d.Decode(&test)
> ```
> 
> The decoder zeroes out the nested struct, losing the pre-set default value. After decoding, `test.Defaults.Key` is an empty string instead of preserving the original default value "My default Value".
> 
> The standard library yaml.v3 preserves the default values in this scenario. The go-yaml decoder should also preserve struct defaults when the corresponding YAML node is null or contains no actual values.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 273.6s | 570 | 4045782 | 22166 | $0.5876 |
| **with-lumen** | 179.4s | 340 | 2456903 | 19632 | $0.4146 |

### baseline

**Rating: Good**

```diff
diff --git a/decode.go b/decode.go
index 72af5e2..e26d5c0 100644
--- a/decode.go
+++ b/decode.go
@@ -990,6 +990,22 @@ func (d *Decoder) createDecodedNewValue(
 		}
 	}
 	if node.Type() == ast.NullType {
+		// Preserve the default value if provided
+		dereferencedVal := defaultVal
+		for dereferencedVal.Kind() == reflect.Ptr {
+			if dereferencedVal.IsNil() {
+				return reflect.Zero(typ), nil
+			}
+			dereferencedVal = dereferencedVal.Elem()
+		}
+		// Get the dereferenced target type for comparison
+		targetType := typ
+		for targetType.Kind() == reflect.Ptr {
+			targetType = targetType.Elem()
+		}
+		if dereferencedVal.IsValid() && dereferencedVal.Type().AssignableTo(targetType) {
+			return dereferencedVal, nil
+		}
 		return reflect.Zero(typ), nil
 	}
 	newValue := d.createDecodableValue(typ)
diff --git a/decode_test.go b/decode_test.go
index f0b0c08..18f05c2 100644
--- a/decode_test.go
+++ b/decode_test.go
@@ -3027,3 +3027,48 @@ func TestMapKeyCustomUnmarshaler(t *testing.T) {
 		t.Fatalf("expected to have value \"value\", but got %q", val)
 	}
 }
+
+type NestedStruct struct {
+	Key string `yaml:"key"`
+}
+
+type TestStruct struct {
+	First    string       `yaml:"first"`
+	Defaults NestedStruct `yaml:"second"`
+}
+
+func TestPreserveDefaultValuesWithNullNestedStruct(t *testing.T) {
+	yml := `first: "Test"
+second:
+  # Just want to document the fields of the nested struct
+#  key: "Value from YAML"
+`
+
+	test := TestStruct{Defaults: NestedStruct{Key: "My default Value"}}
+	d := yaml.NewDecoder(strings.NewReader(yml))
+	err := d.Decode(&test)
+	if err != nil {
+		t.Fatalf("failed to decode: %v", err)
+	}
+
+	if test.Defaults.Key != "My default Value" {
+		t.Errorf("expected default value 'My default Value', got '%s'", test.Defaults.Key)
+	}
+}
+
+func TestPreserveDefaultValuesWithExplicitNull(t *testing.T) {
+	yml := `first: "Test"
+second: null
+`
+
+	test := TestStruct{Defaults: NestedStruct{Key: "My default Value"}}
+	d := yaml.NewDecoder(strings.NewReader(yml))
+	err := d.Decode(&test)
+	if err != nil {
+		t.Fatalf("failed to decode: %v", err)
+	}
+
+	if test.Defaults.Key != "My default Value" {
+		t.Errorf("expected default value 'My default Value', got '%s'", test.Defaults.Key)
+	}
+}
```

### with-lumen

**Rating: Good**

```diff
diff --git a/decode.go b/decode.go
index 72af5e2..197b01a 100644
--- a/decode.go
+++ b/decode.go
@@ -990,7 +990,14 @@ func (d *Decoder) createDecodedNewValue(
 		}
 	}
 	if node.Type() == ast.NullType {
-		return reflect.Zero(typ), nil
+		newValue := d.createDecodableValue(typ)
+		for defaultVal.Kind() == reflect.Ptr {
+			defaultVal = defaultVal.Elem()
+		}
+		if defaultVal.IsValid() && defaultVal.Type().AssignableTo(newValue.Type()) {
+			newValue.Set(defaultVal)
+		}
+		return newValue, nil
 	}
 	newValue := d.createDecodableValue(typ)
 	for defaultVal.Kind() == reflect.Ptr {
diff --git a/test_issue.go b/test_issue.go
new file mode 100644
index 0000000..76f1d5f
--- /dev/null
+++ b/test_issue.go
@@ -0,0 +1,37 @@
+package yaml
+
+import (
+	"strings"
+	"testing"
+)
+
+func TestDecoder_NullNestedStructDefaultValues(t *testing.T) {
+	type Default struct {
+		Key string `yaml:"key"`
+	}
+
+	type Test struct {
+		First    string  `yaml:"first"`
+		Defaults Default `yaml:"second"`
+	}
+
+	test := Test{Defaults: Default{Key: "My default Value"}}
+	yml := `first: "Test"
+second:
+  # Just want to document the fields of the nested struct
+#  key: "Value from YAML"
+`
+	d := NewDecoder(strings.NewReader(yml))
+	err := d.Decode(&test)
+	if err != nil {
+		t.Fatalf("Decode failed: %v", err)
+	}
+
+	if test.First != "Test" {
+		t.Fatalf("test.First should be 'Test', got '%s'", test.First)
+	}
+
+	if test.Defaults.Key != "My default Value" {
+		t.Fatalf("test.Defaults.Key should be 'My default Value', got '%s'", test.Defaults.Key)
+	}
+}
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
| **baseline** | 169.1s | 458 | 3320752 | 12694 | $0.4692 |
| **with-lumen** | 141.8s | 242 | 1390848 | 11780 | $0.2485 |

### baseline

**Rating: Good**

```diff
diff --git a/mux_test.go b/mux_test.go
index 2d8d2b3..a81cc15 100644
--- a/mux_test.go
+++ b/mux_test.go
@@ -216,6 +216,26 @@ func TestHost(t *testing.T) {
 			hostTemplate: `{v-1:[a-z]{3}}.{v-2:[a-z]{3}}.{v-3:[a-z]{3}}`,
 			shouldMatch:  true,
 		},
+		{
+			title:        "Host route with wildcard pattern, match request with port",
+			route:        new(Route).Host("{subdomain:.+}.localhost"),
+			request:      newRequestHost("GET", "/", "test.localhost:8080"),
+			vars:         map[string]string{"subdomain": "test"},
+			host:         "test.localhost",
+			path:         "",
+			hostTemplate: `{subdomain:.+}.localhost`,
+			shouldMatch:  true,
+		},
+		{
+			title:        "Host route with wildcard pattern and port, match request with port",
+			route:        new(Route).Host("{subdomain:.+}.localhost:{port:.*}"),
+			request:      newRequestHost("GET", "/", "test.localhost:8080"),
+			vars:         map[string]string{"subdomain": "test", "port": "8080"},
+			host:         "test.localhost:8080",
+			path:         "",
+			hostTemplate: `{subdomain:.+}.localhost:{port:.*}`,
+			shouldMatch:  true,
+		},
 	}
 	for _, test := range tests {
 		t.Run(test.title, func(t *testing.T) {
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
index 2d8d2b3..529ca4e 100644
--- a/mux_test.go
+++ b/mux_test.go
@@ -225,6 +225,40 @@ func TestHost(t *testing.T) {
 	}
 }
 
+func TestHostWildcardPortMismatch(t *testing.T) {
+	tests := []routeTest{
+		{
+			title:        "Wildcard host without port, request with port",
+			route:        new(Route).Host("{subdomain:.+}.localhost").Path("/"),
+			request:      newRequestHost("GET", "/", "test.localhost:8080"),
+			vars:         map[string]string{"subdomain": "test"},
+			path:         "/",
+			shouldMatch:  true,
+		},
+		{
+			title:        "Wildcard host without port, request without port",
+			route:        new(Route).Host("{subdomain:.+}.localhost").Path("/"),
+			request:      newRequestHost("GET", "/", "test.localhost"),
+			vars:         map[string]string{"subdomain": "test"},
+			path:         "/",
+			shouldMatch:  true,
+		},
+		{
+			title:        "Wildcard host with port, request with matching port",
+			route:        new(Route).Host("{subdomain:.+}.localhost:{port:.*}").Path("/"),
+			request:      newRequestHost("GET", "/", "test.localhost:8080"),
+			vars:         map[string]string{"subdomain": "test", "port": "8080"},
+			path:         "/",
+			shouldMatch:  true,
+		},
+	}
+	for _, test := range tests {
+		t.Run(test.title, func(t *testing.T) {
+			testRoute(t, test)
+		})
+	}
+}
+
 func TestPath(t *testing.T) {
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


