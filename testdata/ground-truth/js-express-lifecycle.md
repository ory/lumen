<!-- Source: https://expressjs.com/en/guide/using-middleware.html -->
<!-- Source: https://expressjs.com/en/guide/error-handling.html -->
<!-- Validated against fixtures: 2026-03-05 -->

## Reference Documentation

Express is a minimal Node.js web framework built on middleware composition. The
request/response lifecycle flows through a chain of middleware functions, each
receiving `(req, res, next)`. The Router manages route matching and dispatches
to handler stacks. Error-handling middleware uses a 4-parameter signature
`(err, req, res, next)` and must be defined after all other middleware. Sub-apps
can be mounted with `app.use('/path', subApp)`, which automatically manages
prototype chains and error propagation.

## Key Types in Fixtures

**express-express.js — Main entry point:**
- `createApplication` — factory function that creates the app
- `exports.Router` — re-exported from external `router` package
- `exports.Route` — re-exported from external `router` package
- `exports.json`, `exports.raw`, `exports.text`, `exports.urlencoded` — re-exported from `body-parser`
- `exports.static` — re-exported from `serve-static`

**express-application.js — App prototype:**
- `app.init` — initializes settings
- `app.defaultConfiguration` — sets default settings (etag, json spaces, etc.)
- `app.handle` — main request dispatch entry point
- `app.use` — registers middleware with optional path prefix
- `app.route` — creates a new Route for a path
- `app.render` — renders a view template
- `app.engine` — registers a template engine
- `app.param` — registers parameter middleware
- `app.set`, `app.get`, `app.enable`, `app.disable` — settings management
- `app.listen` — creates HTTP server

**express-request.js — Request prototype:**
- `req.get` / `req.header` — get request header
- `req.accepts`, `req.acceptsEncodings`, `req.acceptsCharsets`, `req.acceptsLanguages` — content negotiation
- `req.range` — parse Range header
- `req.is` — check Content-Type
- Getters: `req.protocol`, `req.secure`, `req.ip`, `req.ips`, `req.hostname`, `req.subdomains`, `req.path`, `req.fresh`, `req.stale`, `req.xhr`, `req.query`

**express-response.js — Response prototype:**
- `res.status` — set status code
- `res.send` — send response body (auto-detects type)
- `res.json` — send JSON response
- `res.jsonp` — send JSONP response
- `res.sendFile` — send file
- `res.download` — send file as attachment
- `res.render` — render view template
- `res.redirect` — redirect to URL
- `res.set` / `res.header` — set response header
- `res.get` — get response header
- `res.append` — append to header
- `res.cookie`, `res.clearCookie` — cookie management
- `res.location` — set Location header
- `res.type` / `res.contentType` — set Content-Type
- `res.format` — content negotiation response
- `res.vary` — set Vary header
- `res.links` — set Link header

**express-view.js — View class:**
- `View` — constructor that resolves template path and loads engine
- `View.prototype.lookup` — finds template file
- `View.prototype.resolve` — resolves file path
- `View.prototype.render` — renders template via engine

**express-utils.js — Utilities:**
- `exports.etag`, `exports.wetag` — ETag generation
- `exports.compileETag`, `exports.compileQueryParser`, `exports.compileTrust` — setting compilers
- `exports.normalizeType`, `exports.normalizeTypes` — MIME type normalization

## Required Facts

1. The Express app is a function `(req, res, next)` created by `createApplication()` in express-express.js — it calls `app.handle(req, res, next)`.
2. `app.handle()` sets up cross-references (`req.res = res`, `res.req = req`), sets `req.app = app`, initializes `res.locals`, and delegates to `router.handle()`.
3. `app.handle()` explicitly sets request/response prototypes via `Object.setPrototypeOf(req, this.request)` and `Object.setPrototypeOf(res, this.response)`.
4. The Router is lazily initialized on first access via a getter — it auto-registers `query` parser and `expressInit` middleware.
5. `app.use()` accepts an optional path prefix (defaults to `/`), flattens middleware arrays, and delegates to `this.router.use()`.
6. When mounting a sub-application via `app.use('/path', subApp)`, Express detects it by checking `fn.handle && fn.set`, sets `mountpath` and `parent`, emits a `mount` event, and wraps it to restore prototype chains on entry/exit.
7. HTTP method handlers (`app.get`, `app.post`, etc.) create a Route via `this.router.route(path)` and register handlers on it. `app.get` with one argument returns a setting value instead.
8. Standard middleware has 3 parameters `(req, res, next)`. Error-handling middleware has 4 parameters `(err, req, res, next)`.
9. Middleware calls `next()` to pass control to the next function. Calling `next(err)` passes to the next error-handling middleware.
10. `res.send()` auto-detects content type (string→HTML, object→JSON, Buffer→binary), sets Content-Length, generates ETag if enabled, and handles 304 Not Modified.
11. `res.render()` merges `res.locals` into options, delegates to `app.render()`, and defaults to sending the result via `res.send(str)` if no callback provided.
12. View engines are dynamically required and must export an `__express` function with signature `(path, options, callback)`.
13. `View.prototype.lookup()` searches through `this.root` (array of paths) and tries both direct file and `index` file resolutions.
14. Many response methods are chainable (return `this`): `res.status()`, `res.set()`, `res.cookie()`, `res.location()`, `res.vary()`, `res.type()`.
15. `req.query` is lazily parsed on first access using the configured `query parser fn` setting.

## Hallucination Traps

- The `Router` class is NOT defined in the fixtures — it is imported from the external `router` package.
- `express.static` is NOT defined in the fixtures — it is re-exported from `serve-static`.
- Body parser middleware (`json`, `raw`, `text`, `urlencoded`) is NOT defined in the fixtures — re-exported from `body-parser`.
- The `query` and `expressInit` built-in middleware implementations are NOT in the fixtures — only their registration in the router is shown.
- There is NO `next('route')` skip pattern demonstrated in the fixture code.
- There is NO async/await or Promise rejection handling shown in the fixtures (Express 5+ feature).
- The `express-express.js` app factory mixes in `EventEmitter` from Node.js, but the EventEmitter itself is NOT defined in Express fixtures.
