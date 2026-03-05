<!-- Source: https://guides.rubyonrails.org/rails_on_rack.html -->
<!-- Validated against fixtures: 2026-03-05 -->

## Reference Documentation

Rails applications are Rack applications. The middleware stack is assembled by
`Rails::Engine` (and its subclass `Rails::Application`) through a three-stage
process: get the default stack, merge configured middleware, and build the final
Rack application by composing middleware around an endpoint. Each middleware
wraps the next in reverse order using `inject`. Controllers expose actions as
Rack endpoints via `ActionController::Metal.action(name)`, which returns a
lambda accepting a Rack env hash. The `ActionController::MiddlewareStack`
extends `ActionDispatch::MiddlewareStack` with per-action filtering via
`only`/`except` options.

## Key Types in Fixtures

**application.rb:**
- `Rails::Application` — main Rails app class, extends Engine
- `DefaultMiddlewareStack` — autoloaded class that builds the default middleware stack

**engine.rb:**
- `Rails::Engine` — base engine class, extends Railtie
- `Engine#app` — builds/caches the Rack application with middleware
- `Engine#call` — implements Rack interface: `call(env)`
- `Engine#endpoint` — returns the Rack endpoint (defaults to routes)
- `Engine#build_request` — wraps env in ActionDispatch::Request
- `Engine#default_middleware_stack` — returns new ActionDispatch::MiddlewareStack

**metal.rb:**
- `ActionController::Metal` — minimal controller base class, extends AbstractController::Base
- `ActionController::MiddlewareStack` — extends ActionDispatch::MiddlewareStack with action filtering
- `Metal.action` — class method returning a Rack endpoint lambda
- `Metal#dispatch` — dispatches action to controller instance

**base.rb:**
- `ActionController::Base` — full-featured controller, extends Metal

**Other fixture files:**
- `concern.rb` — `ActiveSupport::Concern` module
- `callbacks.rb` — `ActiveRecord::Callbacks` module
- `associations.rb` — `ActiveRecord::Associations` module
- `relation.rb` — `ActiveRecord::Relation` class
- `validations.rb` — `ActiveRecord::Validations` module
- `cache.rb` — `ActiveSupport::Cache` module
- `notifications.rb` — `ActiveSupport::Notifications` module
- `sinatra-base.rb` — `Sinatra::Base` class
- `sinatra-main.rb` — `Sinatra` module

## Required Facts

1. `Rails::Application` inherits from `Rails::Engine` (application.rb), which inherits from `Railtie` (engine.rb).
2. `Engine#app` assembles the Rack application in three stages: (1) get `default_middleware_stack`, (2) merge with `build_middleware.merge_into(stack)`, (3) `build(endpoint)` to compose the final app.
3. `Engine#app` is thread-safe — it uses `@app_build_lock.synchronize` with double-checked locking.
4. `Engine#call(env)` implements the Rack interface: it calls `build_request(env)` then delegates to `app.call(req.env)`.
5. `Engine#endpoint` returns `self.class.endpoint || routes` — defaulting to the routing RouteSet.
6. `Engine#build_request(env)` merges `env_config` into the env hash, creates `ActionDispatch::Request.new(env)`, and sets `req.routes` and `req.engine_script_name`.
7. `Application#build_middleware` prepends `config.app_middleware` before Engine's middleware via `config.app_middleware + super`.
8. `DefaultMiddlewareStack` is autoloaded from `rails/application/default_middleware_stack` (application.rb line 66).
9. Middleware is composed bottom-up: `middlewares.reverse.inject(app)` wraps each middleware around the previous one (metal.rb).
10. `ActionController::MiddlewareStack` extends `ActionDispatch::MiddlewareStack` with `only`/`except` action filtering (metal.rb).
11. `MiddlewareStack#build(action, app, &block)` accepts either an app argument or a block as the innermost endpoint and filters middleware by action name via `middleware.valid?(action)`.
12. `Metal.action(name)` returns a lambda `{ |env| ... }` that creates `ActionDispatch::Request`, makes a response, and calls `dispatch(name, req, res)` — wrapping it with middleware_stack if any middleware exists.
13. `Metal` has a class attribute `middleware_stack` defaulting to a new `ActionController::MiddlewareStack`.
14. `Engine` delegates `:middleware, :root, :paths` to `:config`.
15. Custom middleware can be added via engine initializers (`app.middleware.use`), engine class configuration (`middleware.use`), or application config (`config.middleware.use`).

## Hallucination Traps

- `Rack::Builder` is NOT defined or used in the fixtures — Rails uses `ActionDispatch::MiddlewareStack` instead.
- The full `ActionDispatch::MiddlewareStack` class implementation is NOT in the fixtures — only references to its methods (`merge_into`, `build`, `use`).
- Individual middleware classes (e.g., `ActionDispatch::Static`, `ActionDispatch::Cookies`, `Rack::ETag`, `Rack::Sendfile`) are NOT in the fixtures.
- `ActionDispatch::Routing::RouteSet` is referenced as the default endpoint but NOT defined in fixtures.
- `ActionDispatch::Session::CookieStore` is referenced in comments but NOT implemented in fixtures.
- Methods like `insert_before`, `insert_after`, `swap`, `delete` on MiddlewareStack are NOT demonstrated in fixture code.
- There is NO `Rack::Handler` implementation in the fixtures.
- There is NO `ActionDispatch::Request` class definition in the fixtures — only its instantiation.
