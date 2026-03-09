# SWE-Bench Detail Report

Generated: 2026-03-09 09:59 UTC

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
| **with-lumen** | — | — | — | — | — |

### baseline

```diff
diff --git a/router.go b/router.go
index 15a3398..71ec93b 100644
--- a/router.go
+++ b/router.go
@@ -351,6 +351,10 @@ func (r *Router) Find(method, path string, c Context) {
 			if search == "" && (nn == nil || cn.parent == nil || cn.ppath != "") {
 				break
 			}
+			// Handle special case of trailing slash route with existing any route (see #1526)
+			if search == "" && path[len(path)-1] == '/' && cn.findChildByKind(akind) != nil {
+				goto Any
+			}
 		}
 
 		// Attempt to go back up the tree on no matching prefix or no remaining search
diff --git a/router_test.go b/router_test.go
index 8c27b9f..6229323 100644
--- a/router_test.go
+++ b/router_test.go
@@ -1249,6 +1249,46 @@ func benchmarkRouterRoutes(b *testing.B, routes []*Route) {
 	}
 }
 
+// Test for issue: trailing slash routing regression with wildcard routes
+// Before v4.1.15, defining routes /articles and /articles/* would correctly handle
+// requests to /articles, /articles/, and /articles/whatever.
+// Since v4.1.15, /articles/ no longer matched /articles/* unless explicitly registered.
+func TestRouterTrailingSlashWithWildcardRegression(t *testing.T) {
+	e := New()
+	r := e.router
+	handler := func(c Context) error {
+		c.Set("path", c.Path())
+		return nil
+	}
+
+	// Routes
+	r.Add(http.MethodGet, "/articles", handler)
+	r.Add(http.MethodGet, "/products", handler)
+	r.Add(http.MethodGet, "/articles/*", handler)
+
+	c := e.NewContext(nil, nil).(*context)
+
+	// /articles should match /articles
+	r.Find(http.MethodGet, "/articles", c)
+	c.handler(c)
+	assert.Equal(t, "/articles", c.Get("path"))
+
+	// /articles/ should match /articles/* (this was the failing case)
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/articles/", c)
+	assert.NotNil(t, c.handler, "handler should not be nil for /articles/")
+	c.handler(c)
+	assert.Equal(t, "/articles/*", c.Get("path"))
+	assert.Equal(t, "", c.Param("*"))
+
+	// /articles/something should match /articles/*
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/articles/something", c)
+	c.handler(c)
+	assert.Equal(t, "/articles/*", c.Get("path"))
+	assert.Equal(t, "something", c.Param("*"))
+}
+
 func BenchmarkRouterStaticRoutes(b *testing.B) {
 	benchmarkRouterRoutes(b, staticRoutes)
 }
diff --git a/router_test.go.new b/router_test.go.new
new file mode 100644
index 0000000..392084e
--- /dev/null
+++ b/router_test.go.new
@@ -0,0 +1,67 @@
+package echo
+
+import (
+	"net/http"
+	"testing"
+
+	"github.com/stretchr/testify/assert"
+)
+
+func TestRouterTrailingSlashWithWildcard(t *testing.T) {
+	e := New()
+	r := e.router
+
+	// Routes
+	r.Add(http.MethodGet, "/articles", func(c Context) error {
+		c.Set("path", c.Path())
+		return nil
+	})
+	r.Add(http.MethodGet, "/articles/*", func(c Context) error {
+		c.Set("path", c.Path())
+		return nil
+	})
+
+	c := e.NewContext(nil, nil).(*context)
+
+	// Test 1: /articles should match /articles
+	r.Find(http.MethodGet, "/articles", c)
+	assert.NotNil(t, c.handler)
+	c.handler(c)
+	assert.Equal(t, "/articles", c.Get("path"))
+
+	// Test 2: /articles/ should match /articles/*
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/articles/", c)
+	assert.NotNil(t, c.handler, "handler should not be nil for /articles/")
+	c.handler(c)
+	assert.Equal(t, "/articles/*", c.Get("path"), "/articles/ should match /articles/*")
+	assert.Equal(t, "", c.Param("*"))
+
+	// Test 3: /articles/whatever should match /articles/*
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/articles/whatever", c)
+	assert.NotNil(t, c.handler)
+	c.handler(c)
+	assert.Equal(t, "/articles/*", c.Get("path"))
+	assert.Equal(t, "whatever", c.Param("*"))
+}
+	p := prefix(tail, pfx, "└── ", "├── ")
+	fmt.Printf("%s%s, %p: type=%d, parent=%p, handler=%v, pnames=%v\n", p, n.prefix, n, n.kind, n.parent, n.methodHandler, n.pnames)
+
+	children := n.children
+	l := len(children)
+	p = prefix(tail, pfx, "    ", "│   ")
+	for i := 0; i < l-1; i++ {
+		children[i].printTree(p, false)
+	}
+	if l > 0 {
+		children[l-1].printTree(p, true)
+	}
+}
+
+func prefix(tail bool, p, on, off string) string {
+	if tail {
+		return fmt.Sprintf("%s%s", p, on)
+	}
+	return fmt.Sprintf("%s%s", p, off)
+}
diff --git a/temp_test.go b/temp_test.go
new file mode 100644
index 0000000..b620866
--- /dev/null
+++ b/temp_test.go
@@ -0,0 +1,47 @@
+package echo
+
+import (
+	"net/http"
+	"testing"
+
+	"github.com/stretchr/testify/assert"
+)
+
+func TestRouterTrailingSlashWithWildcard(t *testing.T) {
+	e := New()
+	r := e.router
+
+	// Routes
+	r.Add(http.MethodGet, "/articles", func(c Context) error {
+		c.Set("path", c.Path())
+		return nil
+	})
+	r.Add(http.MethodGet, "/articles/*", func(c Context) error {
+		c.Set("path", c.Path())
+		return nil
+	})
+
+	c := e.NewContext(nil, nil).(*context)
+
+	// Test 1: /articles should match /articles
+	r.Find(http.MethodGet, "/articles", c)
+	assert.NotNil(t, c.handler)
+	c.handler(c)
+	assert.Equal(t, "/articles", c.Get("path"))
+
+	// Test 2: /articles/ should match /articles/*
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/articles/", c)
+	assert.NotNil(t, c.handler, "handler should not be nil for /articles/")
+	c.handler(c)
+	assert.Equal(t, "/articles/*", c.Get("path"), "/articles/ should match /articles/*")
+	assert.Equal(t, "", c.Param("*"))
+
+	// Test 3: /articles/whatever should match /articles/*
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/articles/whatever", c)
+	assert.NotNil(t, c.handler)
+	c.handler(c)
+	assert.Equal(t, "/articles/*", c.Get("path"))
+	assert.Equal(t, "whatever", c.Param("*"))
+}
diff --git a/temp_test2.go b/temp_test2.go
new file mode 100644
index 0000000..04bd306
--- /dev/null
+++ b/temp_test2.go
@@ -0,0 +1,36 @@
+package echo
+
+import (
+	"net/http"
+	"testing"
+
+	"github.com/stretchr/testify/assert"
+)
+
+func TestRouterTrailingSlashWithWildcardUsers(t *testing.T) {
+	e := New()
+	r := e.router
+
+	// Routes - exact case from issue
+	r.Add(http.MethodGet, "/users", func(c Context) error {
+		c.Set("path", c.Path())
+		return nil
+	})
+	r.Add(http.MethodGet, "/users/*", func(c Context) error {
+		c.Set("path", c.Path())
+		return nil
+	})
+
+	c := e.NewContext(nil, nil).(*context)
+
+	// Test: /users/ should match /users/*
+	r.Find(http.MethodGet, "/users/", c)
+	t.Logf("Handler: %v", c.handler)
+	t.Logf("Path: %v", c.Get("path"))
+	assert.NotNil(t, c.handler, "handler should not be nil for /users/")
+	if c.handler != nil {
+		c.handler(c)
+		assert.Equal(t, "/users/*", c.Get("path"), "/users/ should match /users/*")
+		assert.Equal(t, "", c.Param("*"))
+	}
+}
```

### with-lumen

```diff
diff --git a/detailed_test.go b/detailed_test.go
new file mode 100644
index 0000000..c938783
--- /dev/null
+++ b/detailed_test.go
@@ -0,0 +1,34 @@
+package echo
+
+import (
+	"fmt"
+	"net/http"
+	"testing"
+)
+
+func TestWildcardParamCapture(t *testing.T) {
+	e := New()
+	r := e.router
+
+	// Test wildcard param capturing
+	var lastPath string
+	var lastParam string
+	
+	handler := func(c Context) error {
+		lastPath = c.Path()
+		lastParam = c.Param("*")
+		return nil
+	}
+	
+	r.Add(http.MethodGet, "/articles/*", handler)
+
+	// Test 1: /articles/ should capture empty param
+	c := e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/articles/", c)
+	fmt.Printf("Path: /articles/ -> param='%s' (should be '')\n", c.Param("*"))
+	
+	// Test 2: /articles/foo should capture "foo"
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/articles/foo", c)
+	fmt.Printf("Path: /articles/foo -> param='%s' (should be 'foo')\n", c.Param("*"))
+}
diff --git a/router_test.go b/router_test.go
index 8c27b9f..9adbfb2 100644
--- a/router_test.go
+++ b/router_test.go
@@ -1286,3 +1286,86 @@ func prefix(tail bool, p, on, off string) string {
 	}
 	return fmt.Sprintf("%s%s", p, off)
 }
+
+// Test for trailing slash regression with wildcard routes (Issue from v4.1.15)
+// Routes /articles and /articles/* should both handle /articles/
+func TestTrailingSlashWildcard(t *testing.T) {
+	// Test case 1: /articles and /articles/* registered in order
+	e := New()
+	r := e.router
+
+	// Route without trailing slash and wildcard route
+	r.Add(http.MethodGet, "/articles", func(Context) error {
+		return nil
+	})
+	r.Add(http.MethodGet, "/articles/*", func(Context) error {
+		return nil
+	})
+
+	c := e.NewContext(nil, nil).(*context)
+
+	// /articles (exact match)
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/articles", c)
+	assert.NotNil(t, c.handler, "Expected to find handler for /articles")
+
+	// /articles/ should match /articles/* with empty wildcard
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/articles/", c)
+	assert.NotNil(t, c.handler, "Expected to find handler for /articles/")
+	assert.Equal(t, "", c.Param("*"), "Wildcard param should be empty for /articles/")
+
+	// /articles/something should match /articles/*
+	c = e.NewContext(nil, nil).(*context)
+	r.Find(http.MethodGet, "/articles/something", c)
+	assert.NotNil(t, c.handler, "Expected to find handler for /articles/something")
+	assert.Equal(t, "something", c.Param("*"), "Wildcard param should be 'something'")
+
+	// Test case 2: /articles/* registered first
+	e2 := New()
+	r2 := e2.router
+	r2.Add(http.MethodGet, "/articles/*", func(Context) error {
+		return nil
+	})
+	r2.Add(http.MethodGet, "/articles", func(Context) error {
+		return nil
+	})
+
+	c2 := e2.NewContext(nil, nil).(*context)
+	r2.Find(http.MethodGet, "/articles/", c2)
+	assert.NotNil(t, c2.handler, "Expected to find handler for /articles/ (reverse order)")
+	assert.Equal(t, "", c2.Param("*"), "Wildcard param should be empty for /articles/ (reverse order)")
+}
+
+func TestTrailingSlashWildcardDebug(t *testing.T) {
+	e := New()
+	r := e.router
+
+	// Register both /articles and /articles/* - exact scenario from issue
+	handler := func(Context) error { return nil }
+	r.Add(http.MethodGet, "/articles", handler)
+	r.Add(http.MethodGet, "/articles/*", handler)
+
+	// Test all paths from the issue
+	testCases := []struct {
+		path        string
+		shouldMatch bool
+		expectedWildcard string
+	}{
+		{"/articles", true, ""},
+		{"/articles/", true, ""},        // This is the key test - trailing slash should match /articles/*
+		{"/articles/foo", true, "foo"},
+		{"/articles/foo/bar", true, "foo/bar"},
+	}
+
+	for _, tc := range testCases {
+		c := e.NewContext(nil, nil).(*context)
+		r.Find(http.MethodGet, tc.path, c)
+		if tc.shouldMatch {
+			assert.NotNil(t, c.handler, "Expected handler for path: "+tc.path)
+			assert.Equal(t, tc.expectedWildcard, c.Param("*"), "Wildcard param mismatch for path: "+tc.path)
+		} else {
+			assert.Nil(t, c.handler, "Unexpected handler for path: "+tc.path)
+		}
+	}
+}
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
| **baseline** | 173.1s | 290 | 1940387 | 10344 | $0.2991 |
| **with-lumen** | — | — | — | — | — |

### baseline

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

### with-lumen

**Rating: Poor**

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
| **baseline** | 60.2s | 130 | 564539 | 4473 | $0.1317 |
| **with-lumen** | 121.2s | 234 | 1439169 | 9762 | $0.2691 |

### baseline

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

### with-lumen

```diff
diff --git a/mux_test.go b/mux_test.go
index 2d8d2b3..0570f08 100644
--- a/mux_test.go
+++ b/mux_test.go
@@ -216,6 +216,16 @@ func TestHost(t *testing.T) {
 			hostTemplate: `{v-1:[a-z]{3}}.{v-2:[a-z]{3}}.{v-3:[a-z]{3}}`,
 			shouldMatch:  true,
 		},
+		{
+			title:        "Host route with pattern and no port, match with request that has port",
+			route:        new(Route).Host("{subdomain:.+}.localhost"),
+			request:      newRequestHost("GET", "/", "test.localhost:8080"),
+			vars:         map[string]string{"subdomain": "test"},
+			host:         "test.localhost",
+			path:         "",
+			hostTemplate: `{subdomain:.+}.localhost`,
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

---

## php-easy [php / easy]

**Issue:** Custom route binding for backed enums doesn't work

> We're using a custom route binding for a backed enum in our Laravel application. To avoid having to create a new backed enum (string) for plural words, we've been using a custom route binding to resolve the enum from plural words.
> 
> For single words, it works fine. But for plural words, it doesn't work anymore, giving the following error:
> 
> ```
> Object of class App\Enums\YourEnumHere could not be converted to string
> ```
> 
> Steps to reproduce:
> 
> 1. Create a backed enum:
> ```php
> namespace App\Enums;
> 
> enum Fruit:string
> {
>     case APPLE = 'apple';
>     case BANANA = 'banana';
>     case PEAR = 'pear';
> 
>     public static function fromPlural(string $plural): self
>     {
>         return match ($plural) {
>             'apples' => self::APPLE,
>             'bananas' => self::BANANA,
>             'pears' => self::PEAR,
>         };
>     }
> }
> ```
> 
> 2. Add custom route binding to the AppServiceProvider.php:
> ```php
> Route::bind('fruits', fn(string $x) => Fruit::fromPlural($x));
> ```
> 
> 3. Add routes:
> ```php
> Route::get('shop/get/{fruit}/{id}', function (App\Enums\Fruit $fruit, string $id) {
>     dd($fruit, $id); // IT WORKS!!!
> });
> 
> Route::get('shop/list/{fruits}', function (App\Enums\Fruit $fruits) {
>     dd($fruits); // ERROR: Object of class App\Enums\Fruit could not be converted to string
> });
> ```
> 
> Possible solution: Do not implicitly resolve backed enums when there is a custom route binding.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | — | — | — | — | — |
| **with-lumen** | — | — | — | — | — |

### baseline

### with-lumen

**Rating: Poor**

---

## php-hard [php / hard]

**Issue:** Collection is hydrated as empty if first row contains nulls from a leftJoin

> When using a DQL query with a LEFT JOIN and a SQL filter that causes the first row of the result set to contain null values for the joined entity, the collection on the owning side is incorrectly hydrated as empty, even though subsequent rows contain valid data for the association.
> 
> Version affected: 2.11.3
> 
> Steps to reproduce:
> 
> 1. Define two entities with a OneToMany/ManyToOne relationship (e.g. User has many Addresses).
> 2. Enable a SQL filter that excludes some of the joined rows (e.g. a soft-delete filter).
> 3. Execute a DQL query that LEFT JOINs the related entity.
> 4. Ensure the query result ordering is such that the first row returned has NULL values for the joined entity columns (because the filter excluded that particular related entity).
> 5. Hydrate the result.
> 
> Current behavior:
> The collection is hydrated as empty. All related entities are missing from the collection, even those that are not excluded by the filter and appear in subsequent rows of the result set.
> 
> Expected behavior:
> The collection should be properly hydrated with all matching related entities. Only the filtered-out entities should be absent; the remaining valid entities should still appear in the collection.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 295.3s | 554 | 4762715 | 30676 | $0.7139 |
| **with-lumen** | — | — | — | — | — |

### baseline

```diff
diff --git a/lib/Doctrine/ORM/Internal/Hydration/ObjectHydrator.php b/lib/Doctrine/ORM/Internal/Hydration/ObjectHydrator.php
index 4f21a578a..20865f108 100644
--- a/lib/Doctrine/ORM/Internal/Hydration/ObjectHydrator.php
+++ b/lib/Doctrine/ORM/Internal/Hydration/ObjectHydrator.php
@@ -44,6 +44,9 @@ class ObjectHydrator extends AbstractHydrator
     /** @var mixed[] */
     private $initializedCollections = [];
 
+    /** @var array<string, PersistentCollection> */
+    private $uninitializedCollections = [];
+
     /** @var mixed[] */
     private $existingCollections = [];
 
@@ -112,10 +115,11 @@ class ObjectHydrator extends AbstractHydrator
 
         parent::cleanup();
 
-        $this->identifierMap          =
-        $this->initializedCollections =
-        $this->existingCollections    =
-        $this->resultPointers         = [];
+        $this->identifierMap            =
+        $this->initializedCollections   =
+        $this->uninitializedCollections =
+        $this->existingCollections      =
+        $this->resultPointers           = [];
 
         if ($eagerLoad) {
             $this->_uow->triggerEagerLoads();
@@ -126,10 +130,11 @@ class ObjectHydrator extends AbstractHydrator
 
     protected function cleanupAfterRowIteration(): void
     {
-        $this->identifierMap          =
-        $this->initializedCollections =
-        $this->existingCollections    =
-        $this->resultPointers         = [];
+        $this->identifierMap            =
+        $this->initializedCollections   =
+        $this->uninitializedCollections =
+        $this->existingCollections      =
+        $this->resultPointers           = [];
     }
 
     /**
@@ -148,6 +153,12 @@ class ObjectHydrator extends AbstractHydrator
             $coll->takeSnapshot();
         }
 
+        foreach ($this->uninitializedCollections as $coll) {
+            if (! $coll->isInitialized()) {
+                $coll->setInitialized(true);
+            }
+        }
+
         return $result;
     }
 
@@ -411,8 +422,8 @@ class ObjectHydrator extends AbstractHydrator
                         }
                     } elseif (! $reflFieldValue) {
                         $this->initRelatedCollection($parentObject, $parentClass, $relationField, $parentAlias);
-                    } elseif ($reflFieldValue instanceof PersistentCollection && $reflFieldValue->isInitialized() === false) {
-                        $reflFieldValue->setInitialized(true);
+                    } elseif ($reflFieldValue instanceof PersistentCollection && $reflFieldValue->isInitialized() === false && ! isset($this->uninitializedCollections[$oid . $relationField])) {
+                        $this->uninitializedCollections[$oid . $relationField] = $reflFieldValue;
                     }
                 } else {
                     // PATH B: Single-valued association
diff --git a/tests/Doctrine/Tests/ORM/Functional/Ticket/GH9807Test.php b/tests/Doctrine/Tests/ORM/Functional/Ticket/GH9807Test.php
new file mode 100644
index 000000000..40d19d410
--- /dev/null
+++ b/tests/Doctrine/Tests/ORM/Functional/Ticket/GH9807Test.php
@@ -0,0 +1,129 @@
+<?php
+
+declare(strict_types=1);
+
+namespace Doctrine\Tests\ORM\Functional\Ticket;
+
+use Doctrine\Common\Collections\Collection;
+use Doctrine\ORM\Internal\Hydration\ObjectHydrator;
+use Doctrine\ORM\Mapping as ORM;
+use Doctrine\ORM\Mapping\Column;
+use Doctrine\ORM\Mapping\Entity;
+use Doctrine\ORM\Mapping\GeneratedValue;
+use Doctrine\ORM\Mapping\Id;
+use Doctrine\ORM\Query\ResultSetMapping;
+use Doctrine\Tests\Mocks\ArrayResultFactory;
+use Doctrine\Tests\OrmFunctionalTestCase;
+
+final class GH9807Test extends OrmFunctionalTestCase
+{
+    protected function setUp(): void
+    {
+        parent::setUp();
+
+        $this->createSchemaForModels(GH9807Main::class, GH9807Join::class);
+    }
+
+    public function testHydrateJoinedCollectionWithFirstNullishRow(): void
+    {
+        $rsm = new ResultSetMapping();
+        $rsm->addEntityResult(GH9807Main::class, 'm');
+        $rsm->addJoinedEntityResult(GH9807Join::class, 'j', 'm', 'joins');
+
+        $rsm->addFieldResult('m', 'id_0', 'id');
+        $rsm->addFieldResult('j', 'id_1', 'id');
+        $rsm->addFieldResult('j', 'value_2', 'value');
+
+        $hydrator = new ObjectHydrator($this->_em);
+
+        $uow = $this->_em->getUnitOfWork();
+
+        $uow->createEntity(
+            GH9807Main::class,
+            ['id' => 1]
+        );
+
+        $resultSet = [
+            [
+                'id_0' => 1,
+                'id_1' => null,
+                'value_2' => null,
+            ],
+            [
+                'id_0' => 1,
+                'id_1' => 1,
+                'value_2' => '2',
+            ],
+            [
+                'id_0' => 1,
+                'id_1' => 2,
+                'value_2' => '2',
+            ],
+        ];
+
+        $stmt = ArrayResultFactory::createFromArray($resultSet);
+
+        /** @var GH9807Main[] $result */
+        $result = $hydrator->hydrateAll($stmt, $rsm);
+
+        self::assertInstanceOf(GH9807Main::class, $result[0]);
+        self::assertCount(2, $result[0]->getJoins());
+    }
+}
+
+/**
+ * @Entity
+ */
+class GH9807Main
+{
+    /**
+     * @var int
+     * @Column(type="integer")
+     * @Id
+     * @GeneratedValue
+     */
+    private $id;
+
+    /**
+     * @ORM\OneToMany(targetEntity="GH9807Join", mappedBy="main", cascade={"all"})
+     *
+     * @var Collection<int, GH9807Join>
+     */
+    private $joins;
+
+    /**
+     * @return Collection<int, GH9807Join>
+     */
+    public function getJoins(): Collection
+    {
+        return $this->joins;
+    }
+}
+
+/**
+ * @Entity
+ */
+class GH9807Join
+{
+    /**
+     * @var int
+     * @Column(type="integer")
+     * @Id
+     * @GeneratedValue
+     */
+    private $id;
+
+    /**
+     * @ORM\ManyToOne(targetEntity="GH9807Main", inversedBy="joins")
+     *
+     * @var GH9807Main
+     */
+    private $main;
+
+    /**
+     * @ORM\Column(type="string", nullable=false)
+     *
+     * @var string
+     */
+    private $value;
+}
diff --git a/tests/Doctrine/Tests/ORM/Hydration/ObjectHydratorTest.php b/tests/Doctrine/Tests/ORM/Hydration/ObjectHydratorTest.php
index 28aff5d8f..3aa2ed41e 100644
--- a/tests/Doctrine/Tests/ORM/Hydration/ObjectHydratorTest.php
+++ b/tests/Doctrine/Tests/ORM/Hydration/ObjectHydratorTest.php
@@ -2004,4 +2004,72 @@ class ObjectHydratorTest extends HydrationTestCase
         self::assertCount(1, $result[0]->collection);
         self::assertInstanceOf(SimpleEntity::class, $result[0]->collection[0]);
     }
+
+    /**
+     * Test for issue GH-9807: Collection is hydrated as empty if first row contains nulls from a leftJoin
+     *
+     * When the first row of a LEFT JOIN result has NULL values for the joined entity
+     * (e.g., due to a SQL filter), but subsequent rows have valid data, the collection
+     * should still be populated with the valid entities from those rows.
+     *
+     * SELECT u, a
+     *   FROM User u
+     *   LEFT JOIN u.articles a
+     */
+    public function testLeftJoinWithFirstRowNullAndSubsequentValidRows(): void
+    {
+        $rsm = new ResultSetMapping();
+        $rsm->addEntityResult(CmsUser::class, 'u');
+        $rsm->addJoinedEntityResult(CmsArticle::class, 'a', 'u', 'articles');
+        $rsm->addFieldResult('u', 'u__id', 'id');
+        $rsm->addFieldResult('u', 'u__name', 'name');
+        $rsm->addFieldResult('a', 'a__id', 'id');
+        $rsm->addFieldResult('a', 'a__topic', 'topic');
+
+        // Faked result set: first row has NULL article (filtered out), second and third rows have valid articles
+        $resultSet = [
+            // Row 1: User exists but article is NULL (filtered by SQL filter)
+            [
+                'u__id' => '1',
+                'u__name' => 'romanb',
+                'a__id' => null,
+                'a__topic' => null,
+            ],
+            // Row 2: Same user with valid article
+            [
+                'u__id' => '1',
+                'u__name' => 'romanb',
+                'a__id' => '1',
+                'a__topic' => 'First Article',
+            ],
+            // Row 3: Same user with another valid article
+            [
+                'u__id' => '1',
+                'u__name' => 'romanb',
+                'a__id' => '2',
+                'a__topic' => 'Second Article',
+            ],
+        ];
+
+        $stmt     = ArrayResultFactory::createFromArray($resultSet);
+        $hydrator = new ObjectHydrator($this->entityManager);
+        $result   = $hydrator->hydrateAll($stmt, $rsm, [Query::HINT_FORCE_PARTIAL_LOAD => true]);
+
+        // We should have only one user in the result
+        self::assertCount(1, $result);
+
+        $user = $result[0];
+        self::assertInstanceOf(CmsUser::class, $user);
+        self::assertEquals('romanb', $user->name);
+
+        // The user should have 2 articles (not empty!)
+        // This is the critical test - without the fix, the collection would be empty
+        self::assertInstanceOf(PersistentCollection::class, $user->articles);
+        self::assertCount(2, $user->articles, 'Articles collection should contain 2 articles from rows 2 and 3, not be empty');
+
+        // Verify the articles are correct
+        $articles = $user->articles->toArray();
+        self::assertEquals('First Article', $articles[0]->topic);
+        self::assertEquals('Second Article', $articles[1]->topic);
+    }
 }
```

### with-lumen

```diff
diff --git a/tests/Doctrine/Tests/ORM/Hydration/ObjectHydratorTest.php b/tests/Doctrine/Tests/ORM/Hydration/ObjectHydratorTest.php
index 28aff5d8f..a073fe945 100644
--- a/tests/Doctrine/Tests/ORM/Hydration/ObjectHydratorTest.php
+++ b/tests/Doctrine/Tests/ORM/Hydration/ObjectHydratorTest.php
@@ -2004,4 +2004,140 @@ class ObjectHydratorTest extends HydrationTestCase
         self::assertCount(1, $result[0]->collection);
         self::assertInstanceOf(SimpleEntity::class, $result[0]->collection[0]);
     }
+
+    /**
+     * Test for LEFT JOIN collection hydration when first row contains nulls.
+     *
+     * This reproduces the issue where a collection is hydrated as empty when
+     * the first row of the result set contains NULL values for the joined entity
+     * (e.g., due to a SQL filter), even though subsequent rows contain valid data
+     * for the same parent entity.
+     *
+     * SELECT PARTIAL u.{id, status}, PARTIAL a.{id, topic}
+     *   FROM Doctrine\Tests\Models\CMS\CmsUser u
+     *   LEFT JOIN u.articles a
+     */
+    public function testCollectionHydrationWithNullFirstRow(): void
+    {
+        $rsm = new ResultSetMapping();
+        $rsm->addEntityResult(CmsUser::class, 'u');
+        $rsm->addJoinedEntityResult(
+            CmsArticle::class,
+            'a',
+            'u',
+            'articles'
+        );
+        $rsm->addFieldResult('u', 'u__id', 'id');
+        $rsm->addFieldResult('u', 'u__status', 'status');
+        $rsm->addFieldResult('a', 'a__id', 'id');
+        $rsm->addFieldResult('a', 'a__topic', 'topic');
+
+        // Faked result set simulating a filter that excludes the first related entity
+        // but includes others in subsequent rows for the same parent
+        $resultSet = [
+            // First row: user with NULL article (filtered out)
+            [
+                'u__id' => '1',
+                'u__status' => 'developer',
+                'a__id' => null,
+                'a__topic' => null,
+            ],
+            // Second row: same user with valid article
+            [
+                'u__id' => '1',
+                'u__status' => 'developer',
+                'a__id' => '1',
+                'a__topic' => 'Doctrine',
+            ],
+            // Third row: same user with another valid article
+            [
+                'u__id' => '1',
+                'u__status' => 'developer',
+                'a__id' => '2',
+                'a__topic' => 'DDD',
+            ],
+        ];
+
+        $stmt     = ArrayResultFactory::createFromArray($resultSet);
+        $hydrator = new ObjectHydrator($this->entityManager);
+        $result   = $hydrator->hydrateAll($stmt, $rsm, [Query::HINT_FORCE_PARTIAL_LOAD => true]);
+
+        self::assertCount(1, $result);
+        self::assertInstanceOf(CmsUser::class, $result[0]);
+        self::assertEquals('developer', $result[0]->status);
+
+        // The collection should contain both articles, not be empty
+        self::assertInstanceOf(PersistentCollection::class, $result[0]->articles);
+        self::assertCount(2, $result[0]->articles, 'Collection should contain 2 articles, not be empty');
+
+        // Verify the articles are the correct ones
+        $articles = $result[0]->articles->toArray();
+        self::assertEquals('Doctrine', $articles[0]->topic);
+        self::assertEquals('DDD', $articles[1]->topic);
+    }
+
+    /**
+     * Test LEFT JOIN collection hydration when first row of result has NULL
+     * for a different parent than subsequent rows.
+     *
+     * This is another variant where multiple parents are present and the first
+     * row has nulls while subsequent rows (possibly with a different parent)
+     * have valid data.
+     */
+    public function testCollectionHydrationWithNullFirstRowMultipleParents(): void
+    {
+        $rsm = new ResultSetMapping();
+        $rsm->addEntityResult(CmsUser::class, 'u');
+        $rsm->addJoinedEntityResult(
+            CmsArticle::class,
+            'a',
+            'u',
+            'articles'
+        );
+        $rsm->addFieldResult('u', 'u__id', 'id');
+        $rsm->addFieldResult('u', 'u__status', 'status');
+        $rsm->addFieldResult('a', 'a__id', 'id');
+        $rsm->addFieldResult('a', 'a__topic', 'topic');
+
+        // Result set with multiple users, where first user has initial NULL article
+        $resultSet = [
+            // First user's first row: NULL article
+            [
+                'u__id' => '1',
+                'u__status' => 'developer',
+                'a__id' => null,
+                'a__topic' => null,
+            ],
+            // First user's second row: valid article
+            [
+                'u__id' => '1',
+                'u__status' => 'developer',
+                'a__id' => '1',
+                'a__topic' => 'Doctrine',
+            ],
+            // Second user's first row: valid article
+            [
+                'u__id' => '2',
+                'u__status' => 'developer',
+                'a__id' => '2',
+                'a__topic' => 'DDD',
+            ],
+        ];
+
+        $stmt     = ArrayResultFactory::createFromArray($resultSet);
+        $hydrator = new ObjectHydrator($this->entityManager);
+        $result   = $hydrator->hydrateAll($stmt, $rsm, [Query::HINT_FORCE_PARTIAL_LOAD => true]);
+
+        self::assertCount(2, $result);
+
+        // First user should have 1 article
+        self::assertInstanceOf(CmsUser::class, $result[0]);
+        self::assertCount(1, $result[0]->articles, 'First user should have 1 article');
+        self::assertEquals('Doctrine', $result[0]->articles[0]->topic);
+
+        // Second user should have 1 article
+        self::assertInstanceOf(CmsUser::class, $result[1]);
+        self::assertCount(1, $result[1]->articles, 'Second user should have 1 article');
+        self::assertEquals('DDD', $result[1]->articles[0]->topic);
+    }
 }
```

---

## php-medium [php / medium]

**Issue:** [Symfony/forms] Double validation on fields

> ### Symfony version(s) affected
> 
> 7.4.0 7.4.6
> 
> ### Description
> 
> Im having issues with my forms getting duplicate "NotBlank" (probably other constraints as well) validation errors when posting a form.
> 
> I can reproduce the same issue in two different projects (both are Pimcore v12.3.3 projects) but to me it does not feel like the issues is related to Pimcore since no Pimcore code is involved regarding the forms.
> 
> ### How to reproduce
> 
> 1. Create a TestController with a indexAction
> 2. Index action should generate a simple form with a name field with a "NotBlank" constraint
> 3. Add a view which simply prints the form
> 4. Visit the controllers URL and submit an empty form
> 
> ### Additional Context
> 
> HTML result after POST shows each validation error appearing twice:
> ```html
> <span class="invalid-feedback d-block mb-1">
>   <span class="d-block">
>     <span class="form-error-icon badge badge-danger text-uppercase">Error</span>
>     <span class="form-error-message">This value should not be blank.</span>
>   </span>
>   <span class="d-block">
>     <span class="form-error-icon badge badge-danger text-uppercase">Error</span>
>     <span class="form-error-message">This value should not be blank.</span>
>   </span>
> </span>
> ```
> 
> Expected: Each constraint violation should only be reported once.
> Actual: Each constraint violation message appears twice in the rendered form.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 128.6s | 322 | 1832858 | 11570 | $0.2865 |
| **with-lumen** | — | — | — | — | — |

### baseline

```diff
diff --git a/src/Symfony/Component/Form/Extension/Validator/ValidatorExtension.php b/src/Symfony/Component/Form/Extension/Validator/ValidatorExtension.php
index bfad8074fc..a934033efd 100644
--- a/src/Symfony/Component/Form/Extension/Validator/ValidatorExtension.php
+++ b/src/Symfony/Component/Form/Extension/Validator/ValidatorExtension.php
@@ -44,7 +44,18 @@ class ValidatorExtension extends AbstractExtension
         // the DIC, where the XML file is loaded automatically. Thus the following
         // code must be kept synchronized with validation.xml
 
-        $metadata->addConstraint(new Form());
+        // Check if Form constraint is already registered to prevent duplicates
+        $hasFormConstraint = false;
+        foreach ($metadata->getConstraints() as $constraint) {
+            if ($constraint instanceof Form) {
+                $hasFormConstraint = true;
+                break;
+            }
+        }
+
+        if (!$hasFormConstraint) {
+            $metadata->addConstraint(new Form());
+        }
         $metadata->addConstraint(new Traverse(false));
 
         $this->validator = $validator;
diff --git a/src/Symfony/Component/Form/Tests/Extension/Validator/ValidatorExtensionTest.php b/src/Symfony/Component/Form/Tests/Extension/Validator/ValidatorExtensionTest.php
index b721a11530..6f9e626329 100644
--- a/src/Symfony/Component/Form/Tests/Extension/Validator/ValidatorExtensionTest.php
+++ b/src/Symfony/Component/Form/Tests/Extension/Validator/ValidatorExtensionTest.php
@@ -46,4 +46,24 @@ class ValidatorExtensionTest extends TestCase
         $this->assertSame(TraversalStrategy::NONE, $metadata->getTraversalStrategy());
         $this->assertCount(0, $metadata->getPropertyMetadata('children'));
     }
+
+    public function testMultipleExtensionInstantiationDoesNotDuplicateConstraints()
+    {
+        $metadata = new ClassMetadata(Form::class);
+
+        $metadataFactory = new FakeMetadataFactory();
+        $metadataFactory->addMetadata($metadata);
+
+        $validator = Validation::createValidatorBuilder()
+            ->setMetadataFactory($metadataFactory)
+            ->getValidator();
+
+        // Create multiple extensions to simulate the constraint being registered multiple times
+        $extension1 = new ValidatorExtension($validator, false);
+        $extension2 = new ValidatorExtension($validator, false);
+
+        // The Form constraint should still only be added once
+        $this->assertCount(1, $metadata->getConstraints());
+        $this->assertInstanceOf(FormConstraint::class, $metadata->getConstraints()[0]);
+    }
 }
```

### with-lumen

**Rating: INVALID (lumen not used)**


