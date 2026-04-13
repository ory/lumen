# Kotlin Tree-Sitter Chunker + SWE-Bench Evaluation

## Context

GitHub issue ory/lumen#107 requests Kotlin language support. Lumen already uses
tree-sitter for 12 languages via `go-tree-sitter` and `go-sitter-forest`. Adding
Kotlin follows the established pattern: define query patterns, register
extensions, write tests, and validate with SWE-bench.

The goal is not just adding Kotlin parsing but also establishing a benchmark
evaluation loop: implement the chunker, curate real Kotlin bug-fix tasks, run the
SWE-bench suite, analyze chunker effectiveness, and iterate on query patterns
until search quality is satisfactory.

## Kotlin Tree-Sitter Chunker

### Dependency

- Package: `github.com/alexaandru/go-sitter-forest/kotlin v1.9.4`
- Same pattern as Dart: wrap with `sitter.NewLanguage(sitter_kotlin.GetLanguage())`

### Query Patterns

Based on AST analysis of the Kotlin tree-sitter grammar, these patterns capture
all significant Kotlin declarations:

| # | Construct | Pattern | Kind |
|---|-----------|---------|------|
| 1 | Functions (top-level, suspend, inline, extension) | `(function_declaration (simple_identifier) @name) @decl` | "function" |
| 2 | Classes (data, sealed, abstract, annotation, value, fun interface) | `(class_declaration (type_identifier) @name) @decl` | "type" |
| 3 | Object declarations | `(object_declaration (type_identifier) @name) @decl` | "type" |
| 4 | Companion objects (named) | `(companion_object (type_identifier) @name) @decl` | "type" |
| 5 | Properties (top-level + class members) | `(property_declaration (variable_declaration (simple_identifier) @name)) @decl` | "var" |
| 6 | Type aliases | `(type_alias (type_identifier) @name) @decl` | "type" |
| 7 | Enum entries | `(enum_entry (simple_identifier) @name) @decl` | "var" |

**Note on field labels:** The Kotlin tree-sitter grammar does not define `name:`
field labels on its nodes (unlike Java/C#). All patterns use positional child
matching, which is correct — verified that `simple_identifier` always captures
the function name (not parameters) and `type_identifier` captures the class name.

**Note on `go-sitter-forest` wrapping:** The `Language` field must use
`sitter.NewLanguage(sitter_kotlin.GetLanguage())` because `go-sitter-forest`
packages return a raw `unsafe.Pointer`, not `*sitter.Language`. This is the same
pattern used for Dart.

### Design Decisions

- **Interfaces** and **fun interfaces** use `class_declaration` in Kotlin's
  grammar (same as regular classes), so pattern #2 captures them.
- **Companion objects** without a name (`companion object { }`) have no
  `type_identifier`, so pattern #4 won't match them. Inner functions are still
  chunked and qualified via the enclosing class.
- **Init blocks** (`anonymous_initializer`) are intentionally excluded — they
  lack names and are typically short initialization code.
- **Secondary constructors** are intentionally excluded. They have no name node
  in the AST, and the chunker skips matches without `@name` captures (line 103
  of `treesitter.go`). Secondary constructors are captured as part of the
  enclosing class chunk.
- **Method qualification**: The existing `findEnclosingSymbol` does not handle
  `class_declaration` parents (only Dart-specific types). This affects Kotlin,
  Java, C#, and TypeScript. As part of this task, we add `class_declaration`
  support to `findEnclosingSymbol` in `treesitter.go`:
  - Add `"class_declaration"` to the `class_body` case in `findEnclosingSymbol`
  - When the parent is `class_declaration`, try `ChildByFieldName("name")` first
    (works for Java, C#, TypeScript which have a `name:` field)
  - Fall back to scanning for the first `type_identifier` or `identifier` child
    (works for Kotlin, which has no `name:` field)
  - This enables `User.greet` style qualification for all class-based languages
- **`IndexVersion` bump**: Not required. Adding a new language does not change
  the index format or affect existing indexes. New `.kt`/`.kts` files will
  simply be indexed on next run.

### File Changes

**`internal/chunker/languages.go`:**
- Add import: `sitter_kotlin "github.com/alexaandru/go-sitter-forest/kotlin"`
- Add `kotlin := mustTreeSitterChunker(LanguageDef{...})` with 8 patterns
- Map `.kt` and `.kts` to `kotlin` in the return map

**`internal/chunker/languages.go` (`SupportedExtensions`):**
- Add `.kt` and `.kts` to the slice

**`go.mod`:**
- Add `github.com/alexaandru/go-sitter-forest/kotlin v1.9.4`

## Unit Tests

**`internal/chunker/treesitter_test.go`:**

Add a Kotlin test case following the existing pattern (similar to Java/Dart
tests). Test fixture should cover:

- Top-level function
- Extension function
- Data class with methods and properties
- Sealed class with nested classes
- Interface
- Object declaration
- Companion object (named)
- Enum class with entries
- Type alias
- Suspend function
- Value class
- Fun interface
- Secondary constructor
- Leading comment capture

Verify:
- Correct symbol names and kinds
- Symbol qualification (e.g., `User.greet`, `Singleton.doSomething`)
- Comment capture for documented declarations
- `.kt` and `.kts` in `TestDefaultLanguages_AllExtensionsPresent`

## SWE-Bench Kotlin Tasks

### Task Curation

Curate 3-5 real bug-fix tasks from popular Kotlin OSS repositories:

**Candidate repos:**
- `JetBrains/ktor` — HTTP framework
- `Kotlin/kotlinx.serialization` — Serialization library
- `JetBrains/Exposed` — SQL framework
- `Kotlin/kotlinx.coroutines` — Coroutines library
- `square/okhttp` — HTTP client (has Kotlin source)

**Selection criteria:**
- Merged PR with a clear GitHub issue
- Issue description describes the symptom, not file/function names
- Grep score < 50% (validated with `bench-swe validate`)
- Fix touches `.kt` files (not just config/build files)
- Repo can be cloned and built with reasonable setup commands

### File Structure

```
bench-swe/
  tasks/kotlin/
    hard.json          # Task definition
  patches/
    kotlin-hard.patch  # Gold patch (git diff base..fix)
```

### Task JSON Schema

```json
{
  "id": "kotlin-hard",
  "language": "kotlin",
  "repo": "https://github.com/owner/repo",
  "base_commit": "<sha>",
  "fix_commit": "<sha>",
  "issue_url": "https://github.com/owner/repo/issues/NNN",
  "issue_title": "...",
  "issue_body": "...",
  "gold_patch_file": "patches/kotlin-hard.patch",
  "expected_files": ["src/main/kotlin/..."],
  "setup_commands": ["./gradlew build -x test"],
  "test_command": "./gradlew test --tests ...",
  "timeout_s": 900
}
```

## Evaluation Loop

1. **Build**: `make build-local` to compile with Kotlin support
2. **Unit test**: `go test ./internal/chunker/ -run TestKotlin`
3. **Validate tasks**: `./bench-swe validate bench-swe/tasks/`
4. **Run benchmark**: `./bench-swe run --language kotlin`
5. **Analyze**: `./bench-swe analyze bench-results/swe-<timestamp>/`
6. **Review**: Check chunker-analysis.md for hit rate / noise / missed files
7. **Iterate**: If search quality is poor, refine query patterns and re-run

## Verification

- `go test ./internal/chunker/` — all chunker tests pass including new Kotlin tests
- `golangci-lint run` — zero lint issues
- `make build-local` — binary compiles with CGO
- `./bench-swe validate bench-swe/tasks/` — Kotlin tasks pass validation
- `./bench-swe run --language kotlin` — benchmark completes successfully
- Chunker analysis shows reasonable hit rate for Kotlin tasks
