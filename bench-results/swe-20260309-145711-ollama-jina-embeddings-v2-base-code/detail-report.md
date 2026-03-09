# SWE-Bench Detail Report

Generated: 2026-03-09 14:15 UTC

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
| **baseline** | — | — | — | — | — |
| **with-lumen** | 235.6s | 19 | 398637 | 11285 | $0.6723 |

### baseline

**Rating: Poor**

```diff
diff --git a/repro_test.go b/repro_test.go
new file mode 100644
index 0000000..978ced4
--- /dev/null
+++ b/repro_test.go
@@ -0,0 +1,57 @@
+package echo
+
+import (
+	"net/http"
+	"testing"
+	"fmt"
+)
+
+func TestTrailingSlashWildcardRepro(t *testing.T) {
+	e := New()
+	r := e.router
+	
+	handler := func(name string, val int) func(c Context) error {
+		return func(c Context) error {
+			c.Set(name, val)
+			c.Set("path", c.Path())
+			return nil
+		}
+	}
+
+	r.Add(http.MethodGet, "/users", handler("a", 1))
+	r.Add(http.MethodGet, "/users/new", handler("b", 2))
+	r.Add(http.MethodGet, "/users/:id", handler("c", 3))
+	r.Add(http.MethodGet, "/users/dew", handler("d", 4))
+	r.Add(http.MethodGet, "/users/:id/files", handler("e", 5))
+	r.Add(http.MethodGet, "/users/newsee", handler("f", 6))
+	r.Add(http.MethodGet, "/users/*", handler("g", 7))
+	r.Add(http.MethodGet, "/users/new/*", handler("h", 8))
+	r.Add(http.MethodGet, "/*", handler("i", 9))
+
+	// Dump tree structure
+	var dumpNode func(n *node, indent string)
+	dumpNode = func(n *node, indent string) {
+		hasHandler := false
+		for _, m := range []string{"GET", "POST"} {
+			if n.findHandler(m) != nil {
+				hasHandler = true
+			}
+		}
+		fmt.Printf("%s[%d] prefix=%q label=%q ppath=%q handler=%v\n",
+			indent, n.kind, n.prefix, string(n.label), n.ppath, hasHandler)
+		for _, c := range n.children {
+			dumpNode(c, indent+"  ")
+		}
+	}
+	fmt.Println("=== TREE ===")
+	dumpNode(r.tree, "")
+	fmt.Println("=== FIND /users/ ===")
+	
+	c := e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/users/", c)
+	fmt.Printf("Result: ppath=%q, pnames=%v, pvalues=%v\n", c.path, c.pnames, c.pvalues[:len(c.pnames)])
+	if c.handler != nil {
+		c.handler(c)
+		fmt.Printf("path=%v\n", c.Get("path"))
+	}
+}
```

### with-lumen

**Rating: INVALID (lumen not used)**

```diff
diff --git a/router.go b/router.go
index 15a3398..5c29542 100644
--- a/router.go
+++ b/router.go
@@ -348,7 +348,7 @@ func (r *Router) Find(method, path string, c Context) {
 			// Continue search
 			search = search[l:]
 			// Finish routing if no remaining search and we are on an leaf node
-			if search == "" && (nn == nil || cn.parent == nil || cn.ppath != "") {
+			if search == "" && (nn == nil || cn.parent == nil || cn.ppath != "" || cn.findChildByKind(akind) != nil) {
 				break
 			}
 		}
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
| **baseline** | 238.7s | 27 | 859155 | 12021 | $0.9738 |
| **with-lumen** | 118.5s | 21 | 395414 | 5259 | $0.4909 |

### baseline

**Rating: Good**

```diff
diff --git a/decode.go b/decode.go
index 72af5e2..2da1ebb 100644
--- a/decode.go
+++ b/decode.go
@@ -990,6 +990,9 @@ func (d *Decoder) createDecodedNewValue(
 		}
 	}
 	if node.Type() == ast.NullType {
+		if defaultVal.IsValid() && typ.Kind() != reflect.Ptr {
+			return defaultVal, nil
+		}
 		return reflect.Zero(typ), nil
 	}
 	newValue := d.createDecodableValue(typ)
```

### with-lumen

**Rating: INVALID (lumen not used)**

```diff
diff --git a/decode.go b/decode.go
index 72af5e2..90be39a 100644
--- a/decode.go
+++ b/decode.go
@@ -990,6 +990,9 @@ func (d *Decoder) createDecodedNewValue(
 		}
 	}
 	if node.Type() == ast.NullType {
+		if typ.Kind() != reflect.Ptr && defaultVal.IsValid() {
+			return defaultVal, nil
+		}
 		return reflect.Zero(typ), nil
 	}
 	newValue := d.createDecodableValue(typ)
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
| **baseline** | 24.2s | 6 | 75340 | 829 | $0.1522 |
| **with-lumen** | 29.3s | 6 | 75945 | 1138 | $0.1628 |

### baseline

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

### with-lumen

**Rating: INVALID (lumen not used)**

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


