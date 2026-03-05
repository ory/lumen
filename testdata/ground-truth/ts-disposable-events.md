<!-- Source: https://code.visualstudio.com/api/references/vscode-api -->
<!-- Validated against fixtures: 2026-03-05 -->

## Reference Documentation

VSCode's internal architecture uses two complementary patterns for resource
management: `IDisposable` for deterministic cleanup and `Event<T>` / `Emitter<T>`
for type-safe event subscription. Listeners return `IDisposable` so they can be
collected in a `DisposableStore` and cleaned up in bulk. The `Emitter<T>` class
creates events, manages listener arrays, and integrates lifecycle callbacks
(`onWillAddFirstListener`, `onDidRemoveLastListener`) for lazy resource management.

## Key Types in Fixtures

**lifecycle.ts — Disposable pattern:**
- `IDisposable` — interface with single `dispose(): void` method
- `Disposable` — abstract class implementing IDisposable, owns a `_store: DisposableStore`
- `DisposableStore` — collects IDisposable items, disposes all on `.dispose()`
- `MutableDisposable` — holds single swappable IDisposable, auto-disposes on reassignment
- `MandatoryMutableDisposable` — non-optional version of MutableDisposable
- `RefCountedDisposable` — wraps IDisposable with reference counting
- `IReference` — generic interface extending IDisposable with `object: T`
- `ReferenceCollection` — abstract factory for reference-counted objects
- `DisposableMap` — Map that auto-disposes values on removal
- `DisposableSet` — Set that auto-disposes values on removal
- `IDisposableTracker` — debug interface for leak detection
- `DisposableTracker` — tracks disposable lifecycle via stack traces
- `GCBasedDisposableTracker` — uses FinalizationRegistry for GC-time leak detection

**event.ts — Event system:**
- `Event` — generic type alias: callable `(listener, thisArgs?, disposables?) => IDisposable`
- `Emitter` — generic class creating and firing events, exposes `.event` property
- `EmitterOptions` — interface for lifecycle callbacks and leak warning threshold
- `AsyncEmitter` — extends Emitter with `fireAsync()` and `waitUntil` pattern
- `PauseableEmitter` — extends Emitter with `pause()`/`resume()`, queues events while paused
- `DebounceEmitter` — extends PauseableEmitter, auto-pauses and resumes after delay
- `MicrotaskEmitter` — extends Emitter, batches fires into single microtask
- `EventMultiplexer` — combines multiple source Events into one output Event
- `DynamicListEventMultiplexer` — multiplexes from dynamically added/removed items
- `EventBufferer` — buffers events during code blocks, flushes after
- `Relay` — replugable event pipe with dynamic input rewiring
- `EventDeliveryQueue` — handles re-entrant listener iteration safely

## Required Facts

1. `IDisposable` defines a single method `dispose(): void` with no parameters or return value (lifecycle.ts).
2. `Disposable` is an **abstract** class (not concrete) with a protected `_store: DisposableStore` and a `_register()` method for child disposables.
3. `Disposable.None` is a static frozen singleton no-op `{ dispose() {} }`.
4. `DisposableStore` collects IDisposable items in an array and calls `dispose()` on every item when the store itself is disposed.
5. `MutableDisposable` auto-disposes the previous value when a new value is set via the `value` setter.
6. `Event<T>` is a **function type** (callable signature), not a class — subscriptions are made by calling the event directly.
7. Every event subscription returns an `IDisposable` that removes the listener when disposed.
8. `Emitter<T>` lazily creates its `.event` property via `_event ??=` caching pattern.
9. `Emitter` optimizes the single-listener case by storing a `UniqueContainer` directly, converting to array only when a second listener is added.
10. `EmitterOptions` supports lifecycle callbacks: `onWillAddFirstListener`, `onDidAddFirstListener`, `onWillRemoveListener`, `onDidRemoveLastListener`.
11. Listener removal marks the array element as `undefined` (sparse array), with compaction when sparsity exceeds ~50%.
12. `EventDeliveryQueue` makes fire re-entrant-safe — listeners can dispose themselves during fire without corrupting iteration.
13. Errors in individual listeners are caught and reported via `onListenerError` callback; other listeners still fire.
14. `AsyncEmitter.fireAsync()` supports `waitUntil(promise)` pattern — fires complete only after all listener promises resolve.
15. `PauseableEmitter` queues events while paused and flushes all on `resume()`, with optional merge function.
16. `EventMultiplexer` lazily subscribes to source events only when the multiplexer itself has listeners.
17. `Relay` allows dynamically rewiring the input event without disrupting downstream listeners.
18. Transformed events (via `Event.map`, `Event.filter`) require `DisposableStore` to prevent listener leaks on the original event.

## Hallucination Traps

- There is NO class called `EventEmitter` in the fixtures — the class is `Emitter<T>`.
- `Disposable` is **abstract** and cannot be instantiated directly.
- There is NO `.off()` or `.removeListener()` method — cleanup is done exclusively via `IDisposable.dispose()`.
- Events are NOT string-based (no event name strings) — they use generic type parameters `Event<T>`.
- `NodeEventEmitter` and `DOMEventEmitter` are adapter interfaces for interop, NOT the core event system.
- There is NO WeakMap-based listener tracking — `GCBasedDisposableTracker` uses `FinalizationRegistry`.
- `thisArgs` binding is NOT automatic — it must be explicitly passed as the second parameter to the event function.
