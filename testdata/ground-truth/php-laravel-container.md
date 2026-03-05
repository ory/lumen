<!-- Source: https://laravel.com/docs/12.x/container -->
<!-- Validated against fixtures: 2026-03-05 -->

## Reference Documentation

Laravel's service container is a dependency injection container that manages
class dependencies and performs automatic injection. The `Container` class
provides binding (`bind`, `singleton`, `scoped`, `instance`), contextual binding
(`when()->needs()->give()`), and automatic resolution via PHP Reflection. The
`build()` method uses `ReflectionClass` and `ReflectionParameter` to inspect
constructor parameters and recursively resolve dependencies. `ServiceProvider`
is the primary mechanism for registering bindings via `register()` and `boot()`.

## Key Types in Fixtures

**Container.php:**
- `Container` — main IoC container, implements `ArrayAccess` and `ContainerContract`
- `Container::bind` — register a binding (abstract → concrete)
- `Container::singleton` — register a shared binding
- `Container::scoped` — register a scoped binding (reset per lifecycle)
- `Container::instance` — register an existing instance
- `Container::make` — resolve a type from the container
- `Container::build` — instantiate a concrete class using Reflection
- `Container::when` — begin contextual binding definition
- `Container::extend` — extend a resolved type with closure
- `Container::alias` — register an alias for an abstract type
- `Container::tag` — tag bindings for grouped resolution
- `Container::tagged` — resolve all bindings with a tag
- `Container::bound` — check if a binding exists
- `Container::resolved` — check if a type has been resolved
- `Container::isShared` — check if binding is shared (singleton)
- `Container::bindIf` — bind only if not already bound
- `Container::singletonIf` — singleton only if not already bound
- `Container::scopedIf` — scoped only if not already bound
- `Container::addContextualBinding` — add a contextual binding directly
- `Container::bindMethod` — bind a method call
- `Container::callMethodBinding` — invoke a method binding
- `Container::rebinding` — register a rebinding callback
- `Container::resolveDependencies` — resolve constructor dependencies via Reflection
- `Container::resolveFromAttribute` — resolve from PHP attributes

**ContextualBindingBuilder.php:**
- `ContextualBindingBuilder` — builder for contextual bindings, implements `ContextualBindingBuilderContract`
- `ContextualBindingBuilder::__construct` — takes Container and concrete class
- `ContextualBindingBuilder::needs` — specify the abstract dependency
- `ContextualBindingBuilder::give` — provide the implementation
- `ContextualBindingBuilder::giveTagged` — provide all tagged implementations
- `ContextualBindingBuilder::giveConfig` — provide a config value

**ServiceProvider.php:**
- `ServiceProvider` — abstract class for registering container bindings
- `ServiceProvider::register` — register bindings (default empty implementation)
- `ServiceProvider::booting` — register a booting callback
- `ServiceProvider::booted` — register a booted callback
- `ServiceProvider::callBootingCallbacks` — invoke booting callbacks
- `ServiceProvider::callBootedCallbacks` — invoke booted callbacks
- `ServiceProvider::commands` — register Artisan commands
- `ServiceProvider::mergeConfigFrom` — merge package config
- `ServiceProvider::loadRoutesFrom` — load route files
- `ServiceProvider::loadViewsFrom` — load view templates
- `ServiceProvider::loadMigrationsFrom` — load migration files
- `ServiceProvider::publishes` — register publishable assets

## Required Facts

1. `Container` implements `ArrayAccess` and `ContainerContract` (Container.php).
2. `Container::bind($abstract, $concrete, $shared)` registers a binding — when `$concrete` is null, it defaults to `$abstract`; when `$shared` is true, it acts like `singleton`.
3. `Container::singleton($abstract, $concrete)` calls `bind()` with `$shared = true`.
4. `Container::make($abstract, $parameters)` is the primary resolution method — it calls `resolve()` internally.
5. `Container::build($concrete)` uses `ReflectionClass` to inspect the constructor, then calls `resolveDependencies()` to recursively resolve each parameter.
6. `Container::build()` throws `BindingResolutionException` if the class is not instantiable (e.g., abstract class or interface).
7. `Container::resolveDependencies()` iterates `ReflectionParameter` instances and resolves each based on type hints.
8. `Container::when($concrete)` returns a `ContextualBindingBuilder` that provides a fluent API: `->needs($abstract)->give($implementation)`.
9. `ContextualBindingBuilder` stores the container, concrete class(es), and needs/give chain, calling `container->addContextualBinding()` when `give()` is called.
10. `ContextualBindingBuilder::giveTagged($tag)` resolves all bindings tagged with `$tag` for the contextual dependency.
11. `ContextualBindingBuilder::giveConfig($key, $default)` provides a configuration value for the contextual dependency.
12. `ServiceProvider` is abstract and defines `register()` with an empty default — subclasses override it to register bindings.
13. `ServiceProvider` documents `$bindings` and `$singletons` properties for declarative binding registration.
14. `Container::extend($abstract, Closure)` allows modifying a resolved instance after creation.
15. `Container::instance($abstract, $instance)` registers an already-constructed object as a shared binding.

## Hallucination Traps

- The `ContainerContract` interface is NOT defined in the fixtures — only referenced as an implemented interface.
- The `Application` class is NOT in the fixtures — `Container` is the base; Application extends it but is not present.
- The `Facade` class is NOT in the fixtures.
- There is NO `boot()` method defined on `ServiceProvider` in the fixtures — only `booting()` and `booted()` callback registration.
- The `$bindings` and `$singletons` properties on ServiceProvider are only documented in PHPDoc comments, not defined as actual class properties in the fixture.
- There is NO `ContextualBindingBuilderContract` definition in the fixtures — only referenced.
- `Container::getClosure()` is protected, not public — it creates closures for deferred resolution.
- There is NO `Illuminate\Foundation\Application` in the fixtures.
