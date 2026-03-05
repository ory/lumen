<!-- Source: https://docs.rs/grep-searcher/latest/grep_searcher/ -->
<!-- Validated against fixtures: 2026-03-05 -->

## Reference Documentation

ripgrep's search pipeline is built on three core abstractions from separate
crates: the `Matcher` trait (grep-matcher) defines pattern matching, the
`Searcher` struct (grep-searcher) manages reading bytes and finding matches
line-by-line, and `Sink` (grep-searcher) defines how callers receive results.
The `Printer` types (grep-printer: `Standard`, `Summary`, `JSON`) implement
`Sink` to format output. The `SearchWorker` (in the CLI crate) composes these
into a complete pipeline: matcher + searcher + printer, then drives search
across files.

## Key Types in Fixtures

**rg-searcher-lib.rs — Searcher crate public API:**
- `Searcher` — principal search type, reads bytes and invokes Matcher
- `SearcherBuilder` — builder for configuring Searcher
- `Sink` — trait for receiving search results
- `SinkMatch` — match data passed to Sink
- `SinkContext` — context lines passed to Sink
- `SinkContextKind` — enum for context types (Before, After, Other)
- `SinkFinish` — summary data when search completes
- `SinkError` — trait for Sink error types
- `BinaryDetection` — configuration for binary file handling
- `Encoding` — character encoding configuration
- `MmapChoice` — memory-mapped file access configuration
- `ConfigError` — searcher configuration error

**rg-matcher-lib.rs — Matcher crate:**
- `Matcher` — trait defining pattern matching operations
- `Match` — struct representing a match span (start, end)
- `LineTerminator` — configurable line ending
- `ByteSet` — set of bytes (for line terminators)
- `Captures` — trait for capture group access
- `NoCaptures` — no-op Captures implementation
- `NoError` — no-op error type
- `LineMatchKind` — enum (Confirmed or Candidate match)

**rg-printer-lib.rs — Printer crate:**
- `Standard` — human-readable output printer
- `StandardBuilder` — builder for Standard printer
- `StandardSink` — Sink implementation for Standard
- `Summary` — aggregate result printer
- `SummaryBuilder` — builder for Summary
- `SummarySink` — Sink implementation for Summary
- `SummaryKind` — enum for summary output formats
- `JSON` — machine-readable JSON output printer
- `JSONBuilder` — builder for JSON printer
- `JSONSink` — Sink implementation for JSON
- `Stats` — search statistics (matches, lines, bytes, etc.)
- `PathPrinter` — file path printer
- `PathPrinterBuilder` — builder for PathPrinter

**rg-search.rs — CLI search worker:**
- `SearchWorker` — composes matcher + searcher + printer for file search
- `SearchWorkerBuilder` — builder for constructing SearchWorker
- `SearchResult` — result of a single file search (has_match, stats)
- `PatternMatcher` — enum wrapping different matcher implementations
- `Printer` — enum wrapping Standard/Summary/JSON printers

**rg-main.rs — CLI entry point:**
- `main` — entry point calling `run()`

**rg-haystack.rs — File reading internals:**
- Internal searcher glue for reading from files/streams

## Required Facts

1. The `Matcher` trait (rg-matcher-lib.rs) defines pattern matching with methods like `find`, `find_iter`, and associated types for captures and errors.
2. `Match` struct has `start` and `end` fields representing byte offsets of a match.
3. `LineMatchKind` enum has `Confirmed` and `Candidate` variants — `Candidate` means the matcher found a potential match that needs further verification.
4. The `Searcher` struct (rg-searcher-lib.rs) consumes bytes from a source, applies a `Matcher`, and reports results to a `Sink`.
5. `SearcherBuilder` configures the Searcher with options like binary detection, encoding, line terminator, and context lines.
6. The `Sink` trait defines callbacks for search results: methods for match lines, context lines, search begin/finish, and error handling.
7. `SinkMatch` carries the matched bytes and line information to the Sink.
8. `SinkFinish` provides summary data when a search of a single source completes.
9. The `Standard` printer (rg-printer-lib.rs) produces human-readable output with line numbers, file paths, and color.
10. The `Summary` printer shows aggregate results (count of matches, files matched, etc.) with `SummaryKind` variants.
11. The `JSON` printer produces machine-readable JSON Lines format for structured output.
12. `SearchWorkerBuilder` (rg-search.rs) creates a `SearchWorker` that composes a `PatternMatcher`, `Searcher`, and `Printer`.
13. `SearchWorkerBuilder::build()` takes a `WriteColor` writer and returns a configured `SearchWorker`.
14. `SearchResult` has a `has_match()` method and an optional `stats()` method returning `Stats`.
15. `PatternMatcher` is an enum that wraps different matcher implementations used by the CLI.
16. `Printer` is an enum wrapping `Standard`, `Summary`, and `JSON` printer variants.
17. The crate documentation in rg-searcher-lib.rs shows a minimal example: `Searcher::new().search_slice(&matcher, bytes, UTF8(|lnum, line| { ... }))` — demonstrating the Searcher/Matcher/Sink composition.
18. `Stats` (rg-printer-lib.rs) tracks search statistics like matched lines and byte counts.

## Hallucination Traps

- `WalkParallel` (the parallel file walker) is NOT defined in the fixtures — the walker code is in a separate crate not included.
- There is NO `Grep` type in the fixtures — the question previously mentioned "Grep" but the actual types are `Searcher`, `Matcher`, and `Sink`.
- The regex engine internals are NOT in the fixtures — only the `Matcher` trait interface.
- There is NO `ignore` crate walker implementation in rg-ignore-lib.rs — only the crate-level documentation and re-exports.
- `SearchWorker` and `SearchWorkerBuilder` are `pub(crate)` (crate-private), NOT public API.
- The `Sink` trait is from the searcher crate, NOT from the printer crate — printers implement Sink but don't define it.
- There is NO `RegexMatcher` struct defined in the fixtures — only the `Matcher` trait.
- `rg-haystack.rs` contains internal searcher glue code, NOT a public `Haystack` type.
