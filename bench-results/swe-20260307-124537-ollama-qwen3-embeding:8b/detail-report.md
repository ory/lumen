# SWE-Bench Detail Report

Generated: 2026-03-07 11:45 UTC

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
| **baseline** | — | — | — | — | — |
| **mcp-full** | — | — | — | — | — |

### baseline

### mcp-full


