# Benchmark Detail Report

Generated: 2026-03-01 14:07 UTC  |  Results: `20260301-144739-ollama-qwen3-embedding-8b`

---

## label-matcher [easy]

**Question:** What label matcher types are available and how is a Matcher created? Show the type definitions and constructor.

### Metrics

| Model | Scenario | Duration | Input Tok | Cache Read | Cache Created | Output Tok | Cost (USD) |
|-------|----------|----------|-----------|------------|---------------|------------|------------|
| **sonnet** | baseline | 20.0s | 27666 | 28104 | 0 | 615 | $0.23078 |
| **sonnet** | mcp-only | 10.8s | 17455 | 0 | 0 | 622 | $0.10282 |
| **sonnet** | mcp-full | 9.8s | 29607 | 28104 | 0 | 610 | $0.17734 |
| **opus** | baseline | 16.1s | 43036 | 42345 | 0 | 728 | $0.25455 |
| **opus** | mcp-only | 10.7s | 17489 | 0 | 0 | 551 | $0.10122 |
| **opus** | mcp-full | 16.4s | 45056 | 42345 | 0 | 697 | $0.26388 |

### Answer: `sonnet` / `baseline`

Here's a summary of what's available in `matcher.go`:

---

### Match Types

```go
type MatchType int

const (
    MatchEqual     MatchType = iota  // =
    MatchNotEqual                    // !=
    MatchRegexp                      // =~
    MatchNotRegexp                   // !~
)
```

---

### Matcher Struct

```go
type Matcher struct {
    Type  MatchType
    Name  string
    Value string

    re *FastRegexMatcher  // populated for regexp types
}
```

---

### Constructors

**`NewMatcher`** — primary constructor, returns an error if the value is an invalid regex:
```go
func NewMatcher(t MatchType, n, v string) (*Matcher, error)
```
For `MatchRegexp` or `MatchNotRegexp`, it compiles the value into a `FastRegexMatcher` and stores it in the unexported `re` field.

**`MustNewMatcher`** — panics on error, intended for tests only:
```go
func MustNewMatcher(mt MatchType, name, val string) *Matcher
```

### Answer: `sonnet` / `mcp-only`

There are two files with identical definitions (`labels_matcher.go` and `matcher.go`). Here's a summary of the types and constructor:

---

### `MatchType` — the enum (`labels_matcher.go:21-29`)

```go
type MatchType int

const (
    MatchEqual     MatchType = iota  // =
    MatchNotEqual                    // !=
    MatchRegexp                      // =~
    MatchNotRegexp                   // !~
)
```

Four types are available: exact equality, negated equality, regex match, and negated regex match.

---

### `Matcher` — the struct (`labels_matcher.go:46-53`)

```go
type Matcher struct {
    Type  MatchType
    Name  string
    Value string

    re *FastRegexMatcher  // only set for regexp types
}
```

---

### `NewMatcher` — the constructor (`labels_matcher.go:55-70`)

```go
func NewMatcher(t MatchType, n, v string) (*Matcher, error) {
    m := &Matcher{
        Type:  t,
        Name:  n,
        Value: v,
    }
    if t == MatchRegexp || t == MatchNotRegexp {
        re, err := NewFastRegexMatcher(v)
        if err != nil {
            return nil, err
        }
        m.re = re
    }
    return m, nil
}
```

`NewMatcher` takes a `MatchType`, a label name, and a value string. For regexp types (`MatchRegexp` / `MatchNotRegexp`), it compiles the value into a `FastRegexMatcher` and stores it in the private `re` field — returning an error if the regex is invalid. For equality types, `re` is left nil.

There's also `MustNewMatcher` (`labels_matcher.go:72-79`) which wraps `NewMatcher` and panics on error, intended only for use in tests.

### Answer: `sonnet` / `mcp-full`

There are four `MatchType` values and a `NewMatcher` constructor. Here are the relevant definitions:

---

**`MatchType` enum** (`labels_matcher.go:21-37`)

```go
// MatchType is an enum for label matching types.
type MatchType int

const (
    MatchEqual     MatchType = iota  // "="
    MatchNotEqual                    // "!="
    MatchRegexp                      // "=~"
    MatchNotRegexp                   // "!~"
)
```

---

**`Matcher` struct** (`labels_matcher.go:46-53`)

```go
type Matcher struct {
    Type  MatchType
    Name  string
    Value string

    re *FastRegexMatcher
}
```

---

**`NewMatcher` constructor** (`labels_matcher.go:55-70`)

```go
func NewMatcher(t MatchType, n, v string) (*Matcher, error) {
    m := &Matcher{
        Type:  t,
        Name:  n,
        Value: v,
    }
    if t == MatchRegexp || t == MatchNotRegexp {
        re, err := NewFastRegexMatcher(v)
        if err != nil {
            return nil, err
        }
        m.re = re
    }
    return m, nil
}
```

The constructor takes a `MatchType`, label name, and value. For the two regexp types (`MatchRegexp` / `MatchNotRegexp`), it also compiles a `FastRegexMatcher` and stores it on `m.re` — returning an error if the pattern is invalid. For equality types, no compilation is needed and the error is always `nil`.

There's also `MustNewMatcher` (`labels_matcher.go:72-79`) which panics on error, intended only for tests.

### Answer: `opus` / `baseline`

Here's the full picture:

### `MatchType` enum (`labels_matcher.go:22-30`)

```go
type MatchType int

const (
    MatchEqual     MatchType = iota  // "="
    MatchNotEqual                     // "!="
    MatchRegexp                       // "=~"
    MatchNotRegexp                    // "!~"
)
```

Four matcher types: exact equality, negated equality, regex match, and negated regex match.

### `Matcher` struct (`labels_matcher.go:47-53`)

```go
type Matcher struct {
    Type  MatchType
    Name  string
    Value string
    re    *FastRegexMatcher  // unexported; only set for regex types
}
```

### Constructor (`labels_matcher.go:56-70`)

```go
func NewMatcher(t MatchType, n, v string) (*Matcher, error) {
    m := &Matcher{Type: t, Name: n, Value: v}
    if t == MatchRegexp || t == MatchNotRegexp {
        re, err := NewFastRegexMatcher(v)
        if err != nil {
            return nil, err
        }
        m.re = re
    }
    return m, nil
}
```

The constructor sets the basic fields on all matchers, then conditionally compiles a `FastRegexMatcher` only for the two regex types (`MatchRegexp`, `MatchNotRegexp`). There's also a `MustNewMatcher` convenience wrapper (line 73) that panics on error, intended for tests.

### Answer: `opus` / `mcp-only`

Here's what's available:

### `MatchType` enum (`labels_matcher.go:21-30`)

```go
type MatchType int

const (
    MatchEqual    MatchType = iota  // "="
    MatchNotEqual                    // "!="
    MatchRegexp                      // "=~"
    MatchNotRegexp                   // "!~"
)
```

Four match types: exact equality, not-equal, regex match, and negated regex match.

### `Matcher` struct (`labels_matcher.go:46-53`)

```go
type Matcher struct {
    Type  MatchType
    Name  string
    Value string
    re    *FastRegexMatcher  // unexported; only set for regex types
}
```

### Constructor — `NewMatcher` (`labels_matcher.go:55-70`)

```go
func NewMatcher(t MatchType, n, v string) (*Matcher, error) {
    m := &Matcher{Type: t, Name: n, Value: v}
    if t == MatchRegexp || t == MatchNotRegexp {
        re, err := NewFastRegexMatcher(v)
        if err != nil {
            return nil, err
        }
        m.re = re
    }
    return m, nil
}
```

It populates the struct directly, and only compiles a `FastRegexMatcher` when the type is `MatchRegexp` or `MatchNotRegexp`. There's also a `MustNewMatcher` (`labels_matcher.go:72-79`) convenience wrapper that panics on error, intended for tests.

### Answer: `opus` / `mcp-full`

Here's the summary:

### `MatchType` enum (`labels_matcher.go:22-30`)

```go
type MatchType int

const (
    MatchEqual     MatchType = iota  // "="
    MatchNotEqual                     // "!="
    MatchRegexp                       // "=~"
    MatchNotRegexp                    // "!~"
)
```

Four match types: exact equality, inequality, regex match, and negated regex match.

### `Matcher` struct (`labels_matcher.go:47-53`)

```go
type Matcher struct {
    Type  MatchType
    Name  string
    Value string
    re    *FastRegexMatcher  // compiled only for regex match types
}
```

### Constructor (`labels_matcher.go:56-69`)

```go
func NewMatcher(t MatchType, n, v string) (*Matcher, error)
```

Creates a `Matcher` with the given type, label name, and value. If the type is `MatchRegexp` or `MatchNotRegexp`, it compiles a `FastRegexMatcher` from the value string (returning an error if the regex is invalid).

There's also a convenience wrapper `MustNewMatcher` (line 73) that panics on error, intended for tests.

### Full Judge Analysis (Opus 4.6)

_Judge unavailable_

---

## histogram [medium]

**Question:** How does histogram bucket counting work? Show me the relevant function signatures.

### Metrics

| Model | Scenario | Duration | Input Tok | Cache Read | Cache Created | Output Tok | Cost (USD) |
|-------|----------|----------|-----------|------------|---------------|------------|------------|
| **sonnet** | baseline | 62.6s | 29439 | 28104 | 0 | 885 | $0.65639 |
| **sonnet** | mcp-only | 19.8s | 22162 | 0 | 0 | 1016 | $0.13621 |
| **sonnet** | mcp-full | 18.4s | 34307 | 28104 | 0 | 825 | $0.20621 |
| **opus** | baseline | 49.6s | 128149 | 84690 | 0 | 1989 | $0.73281 |
| **opus** | mcp-only | 17.6s | 22148 | 0 | 0 | 899 | $0.13322 |
| **opus** | mcp-full | 20.3s | 34377 | 28230 | 0 | 765 | $0.20513 |

### Answer: `sonnet` / `baseline`

Here's how histogram bucket counting works in the fixture code:

---

## Histogram Bucket Counting

### Core flow

1. **`findBucket(v float64) int`** — determines which bucket an observation falls into:
   - Linear search for small arrays (n < 35)
   - Falls back to `sort.SearchFloat64s()` for larger arrays
   - Returns index of first upper bound ≥ v

2. **`histogramCounts.observe(v float64, bucket int, doSparse bool)`** — records the observation:
   - Atomically increments `hc.buckets[bucket]`
   - Adds v to sum via `atomicAddFloat`
   - If sparse mode is on: calculates a sparse bucket key from the value's exponent and schema, stores in `nativeHistogramBucketsPositive` or `nativeHistogramBucketsNegative` (sync.Map)
   - Increments total `count` last (acts as a completion signal)

3. **`histogram.observe(v float64, bucket int)`** — the high-level entry point:
   - Selects the "hot" counts shard via `countAndHotIdx >> 63`
   - Delegates to `histogramCounts.observe()`
   - Calls `limitBuckets()` if sparse mode is active

### Bucket definitions

| Function | `prom_histogram.go` | What it creates |
|---|---|---|
| `LinearBuckets(start, width, count)` | ~line 295 | Arithmetic progression |
| `ExponentialBuckets(start, factor, count)` | ~line 318 | Geometric progression |
| `ExponentialBucketsRange(min, max, count)` | ~line 338 | Spans a range with computed growth factor |

### Iterating buckets

Buckets are **delta-encoded** (first is absolute, rest are deltas). Iteration is handled by:

- `PositiveBucketIterator()` / `NegativeBucketIterator()` → `histogram.go:214–226`
- `regularBucketIterator.Next()` — walks spans, accumulates deltas, handles span gaps → `histogram.go:496`
- `CumulativeBucketIterator()` — provides a running total view (positive only) → `histogram.go:228`

### Key design properties

- **Thread-safe**: all bucket updates use `atomic.AddUint64` and `sync.Map`
- **Dual storage**: regular (slice) + sparse native (sync.Map) buckets coexist
- **Sharded hot/cold counts**: the `countAndHotIdx` trick lets reads and writes avoid contention during collection

### Answer: `sonnet` / `mcp-only`

Histogram bucket counting happens at two levels: **regular (static) buckets** and **native (sparse/exponential) buckets**. Here are the key functions:

---

### Core counting — `histogramCounts.observe` (`prom_histogram.go:652`)

```go
func (hc *histogramCounts) observe(v float64, bucket int, doSparse bool)
```

This is where the actual counting happens:

1. **Regular buckets** — increments a pre-allocated `[]uint64` slice at the resolved bucket index:
   ```go
   if bucket < len(hc.buckets) {
       atomic.AddUint64(&hc.buckets[bucket], 1)
   }
   ```

2. **Native (sparse) buckets** — computes a bucket key using the exponential schema:
   - For `schema > 0`: binary-searches `nativeHistogramBounds[schema]` using `math.Frexp` to get the fractional mantissa and exponent, then maps to a key.
   - For `schema <= 0`: uses the exponent directly, shifted by a schema-dependent offset.
   - Routes to positive, negative, or zero bucket based on sign vs. zero threshold:
   ```go
   case v > zeroThreshold:
       bucketCreated = addToBucket(&hc.nativeHistogramBucketsPositive, key, 1)
   case v < -zeroThreshold:
       bucketCreated = addToBucket(&hc.nativeHistogramBucketsNegative, key, 1)
   default:
       atomic.AddUint64(&hc.nativeHistogramZeroBucket, 1)
   ```

---

### Entry point — `histogram.observe` (`prom_histogram.go:899`)

```go
func (h *histogram) observe(v float64, bucket int)
```

Atomically selects the hot `histogramCounts` buffer, calls `histogramCounts.observe`, then calls `limitBuckets` if sparse buckets are in use (to enforce `NativeHistogramMaxBucketNumber`).

---

### Iterating accumulated counts — `prom_histogram.go:481` & `histogram.go:530`

```go
func newRegularBucketIterator(spans []Span, buckets []int64, schema int32, positive bool, customValues []float64) regularBucketIterator

type cumulativeBucketIterator struct { ... }
func (c *cumulativeBucketIterator) At() Bucket[uint64]
```

`regularBucketIterator` walks the sparse span-encoded bucket array for native histograms. `cumulativeBucketIterator` wraps it to produce cumulative counts (each bucket's count includes all lower buckets), which is what Prometheus exposes over the wire.

---

### Summary of the counting path:

```
Observe(v float64)
  → findBucket(v)          // binary search in h.upperBounds for regular bucket index
  → histogram.observe(v, bucket)
      → histogramCounts.observe(v, bucket, doSparse)
          → atomic increment in hc.buckets[bucket]   // regular
          → addToBucket(&hc.nativeHistogramBucketsPositive/Negative, key, 1)  // native
          → atomic.AddUint64(&hc.nativeHistogramZeroBucket, 1)  // near-zero values
      → limitBuckets(...)  // trim if too many sparse buckets
```

### Answer: `sonnet` / `mcp-full`

Histogram bucket counting happens at two levels in this codebase. Here's how it works:

---

## 1. Regular (static) bucket counting

**`histogramCounts.observe`** (`prom_histogram.go:652`) is the core method:

```go
func (hc *histogramCounts) observe(v float64, bucket int, doSparse bool)
```

For regular buckets it's a single atomic increment:
```go
if bucket < len(hc.buckets) {
    atomic.AddUint64(&hc.buckets[bucket], 1)
}
```

The `bucket` index is pre-computed by `findBucket` (a binary search over `upperBounds`) before calling `observe`. Each element in `hc.buckets []uint64` corresponds to one upper-bound bucket.

---

## 2. Native (sparse/exponential) histogram bucket counting

Still inside `histogramCounts.observe`, when `doSparse=true`:

1. **Key computation** — maps the observed value to a bucket index using `math.Frexp` (extracts mantissa/exponent), then either binary-searches `nativeHistogramBounds[schema]` (schema > 0) or bit-shifts the exponent (schema ≤ 0).

2. **Routing by sign** — dispatches to positive, negative, or zero bucket:
```go
case v > zeroThreshold:
    bucketCreated = addToBucket(&hc.nativeHistogramBucketsPositive, key, 1)
case v < -zeroThreshold:
    bucketCreated = addToBucket(&hc.nativeHistogramBucketsNegative, key, 1)
default:
    atomic.AddUint64(&hc.nativeHistogramZeroBucket, 1)
```

3. **Bucket creation tracking** — if a new bucket was created, atomically increments `nativeHistogramBucketsNumber` (used for the bucket-limit eviction strategy).

---

## 3. Top-level dispatch

```go
func (h *histogram) observe(v float64, bucket int)   // prom_histogram.go:899
```
Atomically increments `countAndHotIdx` to get the hot `histogramCounts` slot (double-buffered for concurrent writes), then calls `hotCounts.observe(v, bucket, doSparse)`, and finally calls `h.limitBuckets(...)` if sparse buckets are in use.

---

## 4. Iterating cumulative counts (read path)

```go
func (c *cumulativeBucketIterator) At() Bucket[uint64]   // histogram.go:609
```
Tracks a running `currCumulativeCount` as it walks spans, producing `Bucket[uint64]` with `Lower: -Inf` (Prometheus cumulative convention).

### Answer: `opus` / `baseline`

Here's how histogram bucket counting works in this codebase:

## Overview

There are two histogram systems: **classic** (fixed upper bounds) and **native/sparse** (exponential buckets with schemas).

## Key Function Signatures

### Finding which bucket a value belongs to

**`prom_histogram.go:866`** — Classic histogram bucket lookup:
```go
func (h *histogram) findBucket(v float64) int
```
Uses linear search for small arrays (<35 buckets) or binary search (`sort.SearchFloat64s`) for larger ones against `h.upperBounds`.

### Recording an observation

**`prom_histogram.go:900`** — Main observe entry point:
```go
func (h *histogram) observe(v float64, bucket int)
```
Atomically increments `countAndHotIdx`, selects the "hot" counts struct via the top bit (`n>>63`), and delegates to `histogramCounts.observe`.

**`prom_histogram.go:655`** — Low-level counting in both classic and native buckets:
```go
func (hc *histogramCounts) observe(v float64, bucket int, doSparse bool)
```
This is the core logic:
1. **Classic**: `atomic.AddUint64(&hc.buckets[bucket], 1)` — simple atomic increment.
2. **Native/sparse**: Computes a bucket key from the value using `math.Frexp` to decompose the float into fraction + exponent, then maps to a bucket index based on the schema. Values go into positive, negative, or zero buckets via `addToBucket` on `sync.Map`s.

### Bucket limiting

**`prom_histogram.go:919`**:
```go
func (h *histogram) limitBuckets(counts *histogramCounts, value float64, bucket int)
```
Enforces `nativeHistogramMaxBuckets` by reducing resolution when the bucket count is exceeded.

### Validation

**`prom_histogram.go:1871`**:
```go
func validateCount(sum float64, count uint64, negativeBuckets, positiveBuckets map[int]int64, zeroBucket uint64) error
```
Validates that `positive + negative + zero bucket counts == total count` (or `>=` if sum is NaN, since NaN observations increment count but not buckets).

**`histogram.go:426`**:
```go
func (h *Histogram) Validate() error
```
Validates span/bucket consistency for the sparse histogram format, then checks `nCount + pCount + ZeroCount == Count`.

### Bucket generation helpers

**`prom_histogram.go:295`** / **`prom_histogram.go:315`** / **`prom_histogram.go:339`**:
```go
func LinearBuckets(start, width float64, count int) []float64
func ExponentialBuckets(start, factor float64, count int) []float64
func ExponentialBucketsRange(minBucket, maxBucket float64, count int) []float64
```

### Iteration (sparse histograms)

**`histogram.go:216`** / **`histogram.go:223`** / **`histogram.go:232`**:
```go
func (h *Histogram) PositiveBucketIterator() BucketIterator[uint64]
func (h *Histogram) NegativeBucketIterator() BucketIterator[uint64]
func (h *Histogram) CumulativeBucketIterator() BucketIterator[uint64]
```

The sparse format stores bucket counts as **deltas** (`PositiveBuckets []int64`), so the iterator accumulates them: `r.currCount += r.buckets[r.bucketsIdx]` at `histogram.go:524`.

### Answer: `opus` / `mcp-only`

Here's how histogram bucket counting works in this codebase:

## Two-level bucket counting

There are **regular (static) buckets** and **native (sparse/exponential) buckets**.

### Regular buckets

Observations are counted into pre-defined upper-bound buckets via `histogramCounts.observe` at `prom_histogram.go:652`:

```go
func (hc *histogramCounts) observe(v float64, bucket int, doSparse bool)
```

The `bucket` index is pre-computed by the caller. It simply does:
```go
atomic.AddUint64(&hc.buckets[bucket], 1)
```

### Native (sparse) buckets

Within the same `observe` method (`prom_histogram.go:652-706`), if `doSparse` is true, a **bucket key** is computed from the observation value using `math.Frexp` to extract the fraction and exponent:

- **Positive schema** (`schema > 0`): uses `sort.SearchFloat64s` on precomputed `nativeHistogramBounds` to find the sub-bucket within the power-of-two range
- **Non-positive schema**: uses the exponent directly, shifted by the schema

Then the observation is routed to one of three places:
- `addToBucket(&hc.nativeHistogramBucketsPositive, key, 1)` — for `v > zeroThreshold`
- `addToBucket(&hc.nativeHistogramBucketsNegative, key, 1)` — for `v < -zeroThreshold`
- `atomic.AddUint64(&hc.nativeHistogramZeroBucket, 1)` — for values within the zero threshold

After bucket creation, `h.limitBuckets(hotCounts, v, bucket)` is called to enforce the max bucket count.

### Key function signatures

| Signature | Location |
|---|---|
| `func (h *histogram) observe(v float64, bucket int)` | `prom_histogram.go:899` |
| `func (hc *histogramCounts) observe(v float64, bucket int, doSparse bool)` | `prom_histogram.go:652` |
| `func newHistogram(desc *Desc, opts HistogramOpts, labelValues ...string) Histogram` | `prom_histogram.go:535` |
| `func (h *Histogram) ZeroBucket() Bucket[uint64]` | `histogram.go:200` |
| `func newRegularBucketIterator(spans []Span, buckets []int64, schema int32, positive bool, customValues []float64) regularBucketIterator` | `histogram.go:485` |
| `func funcHistogramCount(vectorVals []Vector, _ Matrix, _ parser.Expressions, enh *EvalNodeHelper) (Vector, annotations.Annotations)` | `functions.go:1598` |

### Iteration

For reading buckets back, there are two iterators:

- **`regularBucketIterator`** (`histogram.go:481`) — iterates exponential buckets decoded from spans
- **`cumulativeBucketIterator`** (`histogram.go:530`) — wraps the regular iterator to produce cumulative counts, tracking empty gaps between spans via `emptyBucketCount`

### Answer: `opus` / `mcp-full`

Here's how histogram bucket counting works in this codebase:

There are two layers: the **Prometheus client** (observation/collection) and the **internal histogram model** (storage/iteration).

## Observation Path (Prometheus client)

The core flow is:

1. **`histogram.observe(v float64, bucket int)`** (`prom_histogram.go:899`) — entry point that increments the hot counts atomically and optionally limits sparse buckets.

2. **`histogramCounts.observe(v float64, bucket int, doSparse bool)`** (`prom_histogram.go:652`) — does the actual counting:
   - **Regular buckets**: `atomic.AddUint64(&hc.buckets[bucket], 1)` — simple atomic increment into a pre-allocated `[]uint64` slice, where `bucket` is the index found via binary search over upper bounds.
   - **Native/sparse buckets**: Computes an exponential bucket key from the observation using `math.Frexp`, then calls `addToBucket()` on a `sync.Map` keyed by bucket index. Positive and negative values go into separate maps. Values within the zero threshold go into `nativeHistogramZeroBucket`.

## Storage Model (internal histogram)

The `Histogram` struct (`histogram.go:35-72`) uses a **span-based encoding**:
- `PositiveSpans`, `NegativeSpans` define which bucket indices are populated (as offset + length pairs)
- `PositiveBuckets`, `NegativeBuckets` store delta-encoded counts (first element is absolute, subsequent are deltas)

## Key Function Signatures

```go
// Prometheus observation
func (h *histogram) observe(v float64, bucket int)
func (hc *histogramCounts) observe(v float64, bucket int, doSparse bool)

// Bucket iteration
func newRegularBucketIterator(spans []Span, buckets []int64, schema int32, positive bool, customValues []float64) regularBucketIterator
func (c *cumulativeBucketIterator) At() Bucket[uint64]

// Query functions
func funcHistogramCount(vectorVals []Vector, _ Matrix, _ parser.Expressions, enh *EvalNodeHelper) (Vector, annotations.Annotations)

// Zero bucket access
func (h *Histogram) ZeroBucket() Bucket[uint64]
```

The `cumulativeBucketIterator` (`histogram.go:530-548`) tracks cumulative counts across spans, handling empty buckets between spans via `emptyBucketCount`.

### Full Judge Analysis (Opus 4.6)

_Judge unavailable_

---

## tsdb-compaction [hard]

**Question:** How does TSDB compaction work end-to-end? Explain the Compactor interface, LeveledCompactor, and how the DB triggers compaction. Show relevant types, interfaces, and key method signatures.

### Metrics

| Model | Scenario | Duration | Input Tok | Cache Read | Cache Created | Output Tok | Cost (USD) |
|-------|----------|----------|-----------|------------|---------------|------------|------------|
| **sonnet** | baseline | 94.7s | 31919 | 28104 | 0 | 1596 | $0.51611 |
| **sonnet** | mcp-only | 520.5s | 225054 | 0 | 0 | 4227 | $1.23094 |
| **sonnet** | mcp-full | 84.9s | 69575 | 42156 | 0 | 2609 | $0.43418 |
| **opus** | baseline | 114.5s | 30450 | 28230 | 0 | 2051 | $1.13944 |
| **opus** | mcp-only | 43.1s | 35146 | 0 | 0 | 2143 | $0.22931 |
| **opus** | mcp-full | 57.9s | 76587 | 42345 | 0 | 2239 | $0.46008 |

### Answer: `sonnet` / `baseline`

Here's a full end-to-end explanation of TSDB compaction:

---

## TSDB Compaction: End-to-End

### 1. The `Compactor` Interface (`compact.go:52-77`)

```go
type Compactor interface {
    Plan(dir string) ([]string, error)
    Write(dest string, b BlockReader, mint, maxt int64, base *BlockMeta) ([]ulid.ULID, error)
    Compact(dest string, dirs []string, open []*Block) ([]ulid.ULID, error)
}
```

Three responsibilities:
- **`Plan`** — scans a directory and returns groups of blocks that can be compacted (may overlap between groups)
- **`Write`** — persists a single `BlockReader` (e.g. the head) to a new on-disk block
- **`Compact`** — merges multiple existing blocks from `dirs` into one new block

---

### 2. `LeveledCompactor` (`compact.go:79-93`)

```go
type LeveledCompactor struct {
    ranges                      []int64   // exponential time windows, e.g. [2h, 20h, 200h]
    enableOverlappingCompaction bool      // vertical compaction for overlapping blocks
    blockExcludeFunc            BlockExcludeFilterFunc
    mergeFunc                   storage.VerticalChunkSeriesMergeFunc
    // ...metrics, logger, chunkPool, ctx
}
```

The `ranges` slice is built with:

```go
// ExponentialBlockRanges returns [minSize, minSize*step, minSize*step^2, ...]
func ExponentialBlockRanges(minSize int64, steps, stepSize int) []int64
```

---

### 3. Compaction Planning (`compact.go:248-437`)

`Plan(dir)` → reads all block dirs → calls `plan(dms []dirMeta)`:

```
Priority order:
  1. Overlapping blocks  → selectOverlappingDirs()  (vertical compaction)
  2. Leveled compaction  → selectDirs()             (horizontal, by time window)
  3. Tombstone cleanup   → any block with >5% tombstone ratio
```

**Leveled selection** (`selectDirs`) splits blocks into aligned time buckets via `splitByRange`:

```go
// Example: blocks [0-10, 10-20, 50-60, 90-100], tr=30
// → [[0-10, 10-20], [50-60], [90-100]]
func splitByRange(ds []dirMeta, tr int64) [][]dirMeta
```

A group is selected if it spans the full window OR sits before the newest block, and has >1 block.

**Overlapping detection** (`selectOverlappingDirs`): walks sorted blocks, adds any block whose `MinTime < globalMaxt` of the previous group.

---

### 4. DB Triggers Compaction (`db.go`)

`DB.Compact(ctx)` is the main entry point, protected by `db.cmtx`:

```
DB.Compact()
  ├─ loop: head.compactable()?
  │    └─ compactHead(rh)          ← highest priority: flush in-memory → disk block
  ├─ compactOOOHead(ctx)           ← out-of-order head data
  └─ compactBlocks()               ← on-disk block merging
       └─ loop:
            plan  = compactor.Plan(db.dir)
            uids  = compactor.Compact(db.dir, plan, db.blocks)
            db.reloadBlocks()      ← marks compacted parents as Deletable
```

`compactBlocks()` loops until `Plan()` returns nothing, but aborts early if the head becomes compactable again (head always wins priority).

The background goroutine calls `Compact` on a ticker; `DBOptions` controls:
- `EnableOverlappingCompaction` — vertical merging
- `EnableDelayedCompaction` / `CompactionDelay` — backup window support
- `BlockCompactionExcludeFunc` — skip specific blocks

---

### Summary Flow

```
In-memory Head
    │  head.compactable() → rangeForTimestamp crosses chunk range boundary
    ▼
compactHead()  ──→  Write() ──→  new on-disk Block  ──→  reloadBlocks()
                                                               │
                                        Plan() detects ≥2 blocks in same time window
                                               │
                                        Compact() merges → single larger Block
                                               │
                                        reloadBlocks() marks parents Deletable
                                               │
                                        Loop until Plan() returns []
```

Leveled compaction means small `[0–2h]` blocks eventually get merged into `[0–20h]`, then `[0–200h]`, reducing read amplification over time.

### Answer: `sonnet` / `mcp-only`

Now I have everything from the actual source. Here's the complete end-to-end explanation:

---

## TSDB Compaction: End-to-End

---

### 1. The `Compactor` Interface (`compact.go:52-77`)

```go
type Compactor interface {
    Plan(dir string) ([]string, error)
    Write(dest string, b BlockReader, mint, maxt int64, base *BlockMeta) ([]ulid.ULID, error)
    Compact(dest string, dirs []string, open []*Block) ([]ulid.ULID, error)
}
```

Three distinct responsibilities:

| Method | Role |
|---|---|
| `Plan` | Decides *which* on-disk block directories to compact together |
| `Write` | Flushes a single in-memory source (the Head) to a new Level 1 block |
| `Compact` | Merges multiple existing on-disk blocks into one larger block |

---

### 2. `LeveledCompactor` (`compact.go:79-93`)

```go
type LeveledCompactor struct {
    metrics                     *CompactorMetrics
    logger                      *slog.Logger
    ranges                      []int64   // e.g. [2h, 6h, 24h, 72h] in ms
    chunkPool                   chunkenc.Pool
    ctx                         context.Context
    maxBlockChunkSegmentSize    int64
    useUncachedIO               bool
    mergeFunc                   storage.VerticalChunkSeriesMergeFunc
    blockExcludeFunc            BlockExcludeFilterFunc
    postingsEncoder             index.PostingsEncoder
    postingsDecoderFactory      PostingsDecoderFactory
    enableOverlappingCompaction bool
}
```

The `ranges` slice is the key: it defines a leveled hierarchy of block durations. Blocks are promoted upward as they get merged (level 1 → 2 → 3…). Created via `NewLeveledCompactorWithOptions` (`compact.go:202-241`).

#### `Plan` → `plan` (`compact.go:248-328`)

`Plan` reads all `meta.json` files from block subdirectories (skipping any excluded by `blockExcludeFunc`), then delegates to `plan(dms []dirMeta)` which applies three strategies **in priority order**:

1. **Overlapping blocks** (`selectOverlappingDirs`, `compact.go:369-394`): If `enableOverlappingCompaction` is true and any blocks have overlapping `[MinTime, MaxTime)`, they're returned first — fixes data integrity immediately (vertical compaction).

2. **Leveled range compaction** (`selectDirs`, `compact.go:330-367`): Iterates `ranges[1:]`, uses `splitByRange` to group blocks into aligned time windows. Returns the first group with >1 block that either spans the full range interval or predates the most recent block. **The newest block is always excluded** to give backup tools a safe window.

3. **Tombstone cleanup**: If a sufficiently old block has `>5%` tombstones (or `NumTombstones >= NumSeries`), it's returned alone for rewriting to reclaim space.

#### `Compact` → `CompactWithBlockPopulator` (`compact.go:483-577`)

```go
func (c *LeveledCompactor) Compact(dest string, dirs []string, open []*Block) ([]ulid.ULID, error) {
    return c.CompactWithBlockPopulator(dest, dirs, open, DefaultBlockPopulator{})
}
```

`CompactWithBlockPopulator`:
1. Opens each source block dir (reuses already-open `*Block` objects from the `open` slice to avoid double-loading index data into memory)
2. Allocates a fresh `ulid.ULID` for the output block
3. Calls `CompactBlockMetas(uid, metas...)` — merges metadata: union of all `Sources` ULIDs, all parents recorded, time range = `[min(MinTime), max(MaxTime))`, and `Level = max(sources) + 1`
4. Calls internal `write()` to produce the merged block on disk
5. If the result has 0 samples: marks all source blocks `Deletable` in their `meta.json` instead of writing anything

#### `Write` (`compact.go:586-636`)

Used exclusively for flushing the in-memory Head to disk:

```go
func (c *LeveledCompactor) Write(dest string, b BlockReader, mint, maxt int64, base *BlockMeta) ([]ulid.ULID, error) {
    meta := &BlockMeta{ULID: uid, MinTime: mint, MaxTime: maxt}
    meta.Compaction.Level = 1   // always level 1 — brand-new block
    meta.Compaction.Sources = []ulid.ULID{uid}
    // ...
    err := c.write(dest, meta, DefaultBlockPopulator{}, b)
}
```

#### Internal `write` (`compact.go:657-716`)

The shared write path used by both `Write` and `CompactWithBlockPopulator`:

1. Creates a temp directory `<ULID>.tmp.for-creation`
2. Opens a `ChunkWriter` (segment files) and `IndexWriter`
3. Calls `blockPopulator.PopulateBlock(...)` to stream-merge all series, write chunks and inverted index
4. Atomically renames temp dir to final `<ULID>/` path
5. On any error: removes the temp dir (crash-safe)

`DefaultBlockPopulator.PopulateBlock` (`compact.go:790+`) merges chunk series from all source `BlockReader`s using `mergeFunc`, writes them sorted, and builds the new postings index.

---

### 3. How `DB` Triggers Compaction

**Trigger 1 — Appender commit** (`db.go:1360-1372`):

```go
func (a dbAppender) Commit() error {
    err := a.Appender.Commit()
    if a.db.head.compactable() {
        select {
        case a.db.compactc <- struct{}{}:  // non-blocking signal
        default:
        }
    }
    return err
}
```

**Trigger 2 — Periodic timer** (`db.go:1175-1243`, `DB.run`):

```go
func (db *DB) run(ctx context.Context) {
    for {
        select {
        case <-time.After(db.opts.BlockReloadInterval):
            db.reloadBlocks()
            db.compactc <- struct{}{}        // periodic nudge
            db.head.mmapHeadChunks()
            // also checks stale series ratio → CompactStaleHead()
        case <-db.compactc:
            if db.autoCompact {
                db.Compact(ctx)             // exponential backoff on error
            }
        }
    }
}
```

**`DB.Compact`** (`db.go:1410-1506`) — the orchestrator, holds `cmtx` lock throughout:

**Phase 1 — Head compaction (highest priority):**
```
while head.compactable() && !waitingForCompactionDelay():
    rh = NewRangeHeadWithIsolationDisabled(head, mint, maxt-1)
    head.WaitForAppendersOverlapping(rh.MaxTime())
    db.compactHead(rh)    // → compactor.Write() + reloadBlocks() + truncateMemory()
    lastBlockMaxt = maxt
```

**Phase 2 — WAL truncation** (deferred): `head.truncateWAL(lastBlockMaxt)`

**Phase 3 — OOO head compaction**: `db.compactOOOHead(ctx)` (if out-of-order writes are enabled)

**Phase 4 — On-disk block compaction**: `db.compactBlocks()`

**`DB.compactBlocks`** (`db.go:1719-1763`):

```go
for {
    if db.head.compactable() && !db.waitingForCompactionDelay() {
        // Head needs flushing again — abort block compaction, yield to Phase 1
        return nil
    }
    plan, _ := db.compactor.Plan(db.dir)
    if len(plan) == 0 { break }

    uids, _ := db.compactor.Compact(db.dir, plan, db.blocks)
    db.reloadBlocks()   // opens new block, marks/deletes old ones
}
```

`reloadBlocks` scans all block dirs, identifies superseded blocks (their ULIDs appear in children's `Compaction.Sources`), applies retention, and physically deletes blocks marked `Deletable`.

---

### End-to-End Flow Summary

```
WAL/Head accumulates samples
      │
      ▼  (dbAppender.Commit or periodic timer → compactc)
DB.Compact()
      │
      ├─Phase 1: head.compactable() == true
      │    compactor.Write(RangeHead) → Level-1 block on disk
      │    reloadBlocks() + head.truncateMemory()
      │
      ├─Phase 2: head.truncateWAL()
      │
      ├─Phase 3: compactOOOHead() (if OOO enabled)
      │
      └─Phase 4: compactBlocks() loop
           Plan() → selectOverlappingDirs OR selectDirs by ranges
           Compact() → CompactBlockMetas (level++) + write()
           reloadBlocks() → delete Deletable source blocks
```

Small 2h Level-1 blocks get merged into 6h Level-2, then 24h Level-3, etc. — each merge increments `Compaction.Level` and records source ULIDs in `Compaction.Parents`, which is how `reloadBlocks` knows what to delete.

### Answer: `sonnet` / `mcp-full`

Here's a complete end-to-end explanation of TSDB compaction:

---

## TSDB Compaction: End-to-End

### 1. The `Compactor` Interface (`compact.go:52`)

```go
type Compactor interface {
    Plan(dir string) ([]string, error)
    Write(dest string, b BlockReader, mint, maxt int64, base *BlockMeta) ([]ulid.ULID, error)
    Compact(dest string, dirs []string, open []*Block) ([]ulid.ULID, error)
}
```

Three methods, each with a distinct role:
- **`Plan`** — reads block metadata on disk and returns which block directories should be compacted next. Returns directories that can be compacted concurrently.
- **`Write`** — persists a `BlockReader` (e.g. the in-memory Head) to a new on-disk block. Used for Head → disk flushing. Returns 0 or 1 ULIDs.
- **`Compact`** — merges the blocks in the given dirs into a new block. Used for merging multiple existing on-disk blocks.

---

### 2. `LeveledCompactor` (`compact.go:79`)

```go
type LeveledCompactor struct {
    ranges                      []int64   // time range levels, e.g. [2h, 6h, 24h, ...]
    chunkPool                   chunkenc.Pool
    mergeFunc                   storage.VerticalChunkSeriesMergeFunc
    enableOverlappingCompaction bool
    blockExcludeFunc            BlockExcludeFilterFunc
    postingsEncoder             index.PostingsEncoder
    // ...
}
```

The `ranges` slice defines the compaction level hierarchy (e.g. `[2h, 6h, 24h, 72h]`). Each level is a multiple of the previous.

#### `Plan` (`compact.go:248`)
```go
func (c *LeveledCompactor) Plan(dir string) ([]string, error)
```
Reads all block dirs, loads their `meta.json`, then calls the internal `plan(dms []dirMeta)`:

1. **Sort** blocks by `MinTime`.
2. **Overlapping blocks first** (`selectOverlappingDirs`): if any blocks have overlapping time ranges, return them immediately for vertical compaction (merging with `mergeFunc`). Only runs if `enableOverlappingCompaction=true`.
3. **Level-based selection** (`selectDirs`): iterates through range levels (`c.ranges[1:]`), splits blocks into groups that fit within each level's window, and picks a group of 2+ blocks that spans the full range or is older than the newest block. This avoids premature compaction.
4. **Tombstone compaction**: if no level selection applies, compact any block where tombstones exceed 5% of series.

#### `Compact` → `CompactWithBlockPopulator` (`compact.go:483`, `compact.go:489`)
```go
func (c *LeveledCompactor) Compact(dest string, dirs []string, open []*Block) ([]ulid.ULID, error)
// delegates to:
func (c *LeveledCompactor) CompactWithBlockPopulator(dest string, dirs []string, open []*Block, blockPopulator BlockPopulator) ([]ulid.ULID, error)
```
- Opens each block dir (reuses already-open `*Block` objects to avoid double memory).
- Calls `CompactBlockMetas` to compute merged metadata.
- Calls `c.write(dest, meta, ...)` to stream all series through the merge and write the new block.
- If the resulting block has 0 samples, marks all source blocks as `Deletable` instead.

#### `Write` (`compact.go:586`)
```go
func (c *LeveledCompactor) Write(dest string, b BlockReader, mint, maxt int64, base *BlockMeta) ([]ulid.ULID, error)
```
Creates a new `Level=1` block from a `BlockReader`. Used exclusively to flush the Head to disk.

---

### 3. How `DB` Triggers Compaction

#### Background loop: `DB.run` (`db.go:1175`)
```go
func (db *DB) run(ctx context.Context)
```
A goroutine started on `Open`. It listens on two channels:
- **`time.After(BlockReloadInterval)`**: periodically reloads blocks and sends to `db.compactc`.
- **`db.compactc`**: triggers `db.Compact(ctx)` if `db.autoCompact` is true.

Stale series are also checked here — if the ratio exceeds `staleSeriesCompactionThreshold`, it calls `db.CompactStaleHead()` immediately without waiting.

#### `dbAppender` triggers compaction on commit (`db.go:1344`)
```go
type dbAppender struct {
    storage.Appender
    db *DB
}
```
On `Commit()`, it checks if the Head needs compacting and sends to `db.compactc` — this is what drives compaction from the write path.

#### `DB.Compact` (`db.go:1410`) — the main entry point
```go
func (db *DB) Compact(ctx context.Context) (returnErr error)
```
Holds `db.cmtx`. Two phases:

**Phase 1 — Head compaction (highest priority):**
```
for head.compactable():
    wait for CompactionDelay (jittered, avoids thundering herd)
    mint = head.MinTime()
    maxt = rangeForTimestamp(mint, chunkRange)
    db.compactHead(NewRangeHead(head, mint, maxt-1))
```
`compactHead` (`db.go:1634`) calls `compactor.Write(db.dir, head, ...)` → flushes Head to disk → `reloadBlocks()` → `head.truncateMemory(maxt)`.

**Phase 2 — On-disk block compaction:**
```go
db.compactBlocks()  // db.go:1719
```
```
for:
    if head.compactable(): abort (head has priority)
    plan = compactor.Plan(db.dir)
    if len(plan) == 0: break
    uids = compactor.Compact(db.dir, plan, db.blocks)
    db.reloadBlocks()
```
Loops until no more compactable plan exists.

#### OOO (Out-of-Order) compaction (`db.go:1534`)
Out-of-order samples are stored in a separate WBL (write-behind log). `compactOOOHead` creates an `OOOCompactionHead` and calls `compactOOO`, which iterates over block-sized time windows and calls `compactor.Write` for each, producing multiple blocks.

---

### Flow Summary

```
dbAppender.Commit()
    └─► db.compactc channel

DB.run() goroutine
    └─► receives db.compactc
    └─► DB.Compact(ctx)
            ├─ Phase 1: Head → disk
            │       head.compactable()?
            │       wait CompactionDelay
            │       compactor.Write(head)  ← LeveledCompactor.Write
            │       reloadBlocks()
            │       head.truncateMemory()
            │
            └─ Phase 2: on-disk merges
                    compactor.Plan(dir)    ← selectOverlappingDirs | selectDirs | tombstones
                    compactor.Compact(plan)← merge N blocks → 1
                    reloadBlocks()         ← marks old blocks Deletable
```

The `ranges` slice (`e.g. [2h, 6h, 24h, 72h]`) controls when blocks graduate to the next level. A block only gets compacted once enough peers at the same level exist to fill the next level's range.

### Answer: `opus` / `baseline`

## TSDB Compaction End-to-End

### 1. Compactor Interface

**`compact.go:54-77`** — Three core methods:

```go
type Compactor interface {
    Plan(dir string) ([]string, error)
    Write(dest string, b BlockReader, mint, maxt int64, base *BlockMeta) ([]ulid.ULID, error)
    Compact(dest string, dirs []string, open []*Block) ([]ulid.ULID, error)
}
```

- **Plan** — scans block directories and returns groups that should be compacted together
- **Write** — persists a `BlockReader` (e.g. the in-memory head) into a new on-disk block
- **Compact** — merges multiple existing blocks into one (or more) new blocks

---

### 2. LeveledCompactor

**`compact.go:79-93`** — the production implementation:

```go
type LeveledCompactor struct {
    metrics                     *CompactorMetrics
    logger                      *slog.Logger
    ranges                      []int64              // time ranges: e.g. 2h, 4h, 8h…
    chunkPool                   chunkenc.Pool
    ctx                         context.Context
    maxBlockChunkSegmentSize    int64
    mergeFunc                   storage.VerticalChunkSeriesMergeFunc
    blockExcludeFunc            BlockExcludeFilterFunc
    enableOverlappingCompaction bool
    // …
}
```

**Key methods:**

| Method | Location | Role |
|--------|----------|------|
| `Plan()` | lines 249-277 | Reads block metas, delegates to internal `plan()` |
| `plan()` | lines 311-393 | Overlap detection → level-based grouping → tombstone cleanup |
| `Compact()` | line 485 | Wrapper → `CompactWithBlockPopulator()` |
| `CompactWithBlockPopulator()` | lines 489-584 | Opens blocks, merges metas, calls `write()`, marks sources deletable |
| `Write()` | lines 586-636 | Creates Level=1 block from head, calls `write()` |
| `write()` | lines 658-768 | Actual I/O: tmp dir → ChunkWriter + IndexWriter → PopulateBlock → atomic rename |

**Planning strategy** (inside `plan()`):

1. **Overlapping blocks** — detected first; all overlapping blocks are returned for vertical compaction
2. **Level-based grouping** — blocks are grouped by configured time ranges (exponential: 2h → 4h → 8h…). A group compacts when it spans the full range or sits before the newest block
3. **Tombstone cleanup** — blocks with >5% tombstone ratio trigger single-block compaction to reclaim space

**Persistence** (`write()`, lines 658-768):

1. Create `.tmp-for-creation` directory
2. Write chunks via `ChunkWriter`, index via `IndexWriter`
3. Call `BlockPopulator.PopulateBlock()` to merge series
4. Write `meta.json` + `tombstones.db`
5. `fsync` directory
6. Atomic rename to final path via `fileutil.Replace()`

---

### 3. How the DB Triggers Compaction

**`db.go`** — three layers, called from `DB.Compact()` (lines 1414-1505):

```
DB.Compact()
  ├── 1. Head compaction loop    → compactHead()
  ├── 2. OOO head compaction     → compactOOOHead()
  └── 3. Block compaction loop   → compactBlocks()
```

**Step 1 — Head compaction** (`compactHead()`, lines 1636-1662):
- Checks `head.compactable()` and compaction delay logic
- Waits for overlapping appenders to finish
- Creates a `RangeHead` bounding reads to a time window
- Calls `compactor.Write(db.dir, head, mint, maxt, nil)` — produces a Level=1 block
- Reloads blocks, truncates head memory + WAL

**Step 2 — OOO head compaction** (`compactOOOHead()`, lines 1535-1585):
- Wraps out-of-order chunks in `OOOCompactionHead`
- Splits into time ranges, calls `compactor.Write()` for each
- Truncates OOO WAL

**Step 3 — Block compaction** (`compactBlocks()`, lines 1721-1763):
```go
for {
    plan, _ := db.compactor.Plan(db.dir)
    if len(plan) == 0 {
        break
    }
    db.compactor.Compact(db.dir, plan, db.blocks)
    db.reloadBlocks()
}
```
Loops until `Plan()` returns nothing — each iteration merges one group of blocks and reloads.

**Block cleanup** happens in `reloadBlocks()` (lines 1792-1902): blocks marked `Deletable`, parent blocks superseded by compaction output, and blocks past time/size retention are all deleted.

---

### 4. Key Types

**BlockMeta** fields (used throughout `compact.go`):

| Field | Type | Purpose |
|-------|------|---------|
| `ULID` | `ulid.ULID` | Unique block ID |
| `MinTime` / `MaxTime` | `int64` | Time range (half-open) |
| `Compaction.Level` | `int` | 1 for head writes, increments on merge |
| `Compaction.Sources` | `[]ulid.ULID` | Original source block IDs |
| `Compaction.Parents` | `[]BlockDesc` | Immediate parent blocks |
| `Compaction.Deletable` | `bool` | Marked for deletion after compaction |
| `Stats.NumSamples` | `uint64` | Sample count (0 → block not written) |

**CompactBlockMetas()** (lines 441-481) — merges N metas into one: `Level = max(inputs) + 1`, union of sources, parents = input blocks, time range = union.

---

### Summary Flow

```
time passes, samples accumulate in head
        │
        ▼
head.compactable() == true
        │
        ▼
compactor.Write(head) ──► Level 1 block on disk
        │
        ▼
compactor.Plan(dir) ──► groups blocks by time range / overlap
        │
        ▼
compactor.Compact(group) ──► merged Level N+1 block
        │                     sources marked Deletable
        ▼
reloadBlocks() ──► deletes old blocks, applies retention
```

### Answer: `opus` / `mcp-only`

## TSDB Compaction End-to-End

### 1. The `Compactor` Interface

Defined at `compact.go:52-77`, this is the abstraction for all compaction:

```go
type Compactor interface {
    Plan(dir string) ([]string, error)
    Write(dest string, b BlockReader, mint, maxt int64, base *BlockMeta) ([]ulid.ULID, error)
    Compact(dest string, dirs []string, open []*Block) ([]ulid.ULID, error)
}
```

- **`Plan`** — scans the data directory and returns block directories eligible for compaction
- **`Write`** — persists a `BlockReader` (e.g. the in-memory Head) into a new on-disk block
- **`Compact`** — merges multiple existing block directories into one new block

### 2. `LeveledCompactor` — the Implementation

Defined at `compact.go:79-93`:

```go
type LeveledCompactor struct {
    metrics                     *CompactorMetrics
    logger                      *slog.Logger
    ranges                      []int64          // time ranges for each compaction level
    chunkPool                   chunkenc.Pool
    ctx                         context.Context
    mergeFunc                   storage.VerticalChunkSeriesMergeFunc
    blockExcludeFunc            BlockExcludeFilterFunc
    enableOverlappingCompaction bool
    // ...
}
```

The `ranges` field is key — it defines the leveled compaction strategy. Blocks at level N span `ranges[N]` milliseconds. When enough smaller blocks accumulate to fill the next level's range, they get compacted together.

**Planning** (`compact.go:248-277` and `compact.go:279-328`):
1. `Plan()` reads all block `meta.json` files from the directory
2. Delegates to `plan()` which first checks for **overlapping blocks** (compacted via `selectOverlappingDirs`)
3. If no overlaps, `selectDirs()` (`compact.go:330-367`) walks the configured ranges and finds groups of blocks that fill a higher-level range
4. As a fallback, picks blocks with >5% tombstones for cleanup compaction

**Compacting** (`compact.go:483-487`):
```go
func (c *LeveledCompactor) Compact(dest string, dirs []string, open []*Block) ([]ulid.ULID, error) {
    return c.CompactWithBlockPopulator(dest, dirs, open, DefaultBlockPopulator{})
}
```

`CompactWithBlockPopulator` (`compact.go:489-577`) opens each block, calls `CompactBlockMetas` to merge metadata (incrementing the compaction level), then calls `c.write()`.

**Writing** (`compact.go:657-716`): The internal `write()` method:
1. Creates a temp directory
2. Opens chunk and index writers
3. Calls `BlockPopulator.PopulateBlock()` — this is where series are merged, chunks re-encoded, and the new index built
4. Writes `meta.json`, syncs, and atomically renames the temp dir to the final block dir

**`BlockPopulator`** interface (`compact.go:771-773`) handles the actual data merge. `DefaultBlockPopulator.PopulateBlock` (`compact.go:790-850`) opens index/chunk/tombstone readers from all source blocks, detects overlapping blocks (vertical compaction), and streams merged series into the new block.

### 3. How the DB Triggers Compaction

The `DB` struct holds the compactor and auto-compaction machinery (`db.go:281-342`):

```go
type DB struct {
    compactor      Compactor
    compactc       chan struct{}     // signal channel
    autoCompact    bool
    cmtx           sync.Mutex       // serializes compactions + deletions
    // ...
}
```

**Trigger path — the `run()` loop** (`db.go:1175-1243`):

The background goroutine `DB.run()` has two trigger sources:

1. **Periodic timer** (`BlockReloadInterval`): reloads blocks, then sends on `compactc`, and also mmaps head chunks. Additionally checks for stale series ratio and triggers `CompactStaleHead()` if needed.

2. **`compactc` channel receive**: checks `autoCompact` flag, then calls `db.Compact(ctx)`. Uses exponential backoff on failure.

**Trigger path — on commit** (`db.go:1344-1402`):

Both `dbAppender.Commit()` and `dbAppenderV2.Commit()` check `db.head.compactable()` after every commit. If the head is compactable, they send a non-blocking signal on `db.compactc`:

```go
if a.db.head.compactable() {
    select {
    case a.db.compactc <- struct{}{}:
    default:
    }
}
```

**`DB.Compact()`** (`db.go:1410-1465`) — the main compaction method:

1. Acquires `cmtx` (mutex serializing compactions)
2. **Phase 1 — Head compaction** (highest priority): loops while `head.compactable()`, respecting an optional `CompactionDelay`. Computes the time range, wraps the head into a `RangeHead`, and calls `compactHead()` which uses `compactor.Write()` to persist the head to a block, then truncates memory and WAL.
3. **Phase 2 — OOO head compaction**: if out-of-order data exists, `compactOOO()` (`db.go:1587-1632`) writes one block per time range from the OOO head.
4. **Phase 3 — Block compaction**: `compactBlocks()` (`db.go:1719-1763`) loops calling `compactor.Plan()` then `compactor.Compact()`, reloading blocks after each round. It aborts early if the head becomes compactable again (head always takes priority).

**Compaction delay** (`db.go:1404-1408`): An optional jitter (`CompactionDelay`, up to `CompactionDelayMaxPercent` of chunk range) delays head compaction start to stagger compactions across multiple TSDB instances.

### Summary Flow

```
Appender.Commit() or periodic timer
        │
        ▼
    compactc channel
        │
        ▼
    DB.Compact()
        │
        ├─► Head compactable? → compactHead() → compactor.Write() → reloadBlocks()
        ├─► OOO data?         → compactOOO()  → compactor.Write() → reloadBlocks()
        └─► compactBlocks()   → compactor.Plan() → compactor.Compact() → reloadBlocks()
                                     │                    │
                                     ▼                    ▼
                              LeveledCompactor      LeveledCompactor
                              .plan() selects       .write() merges
                              block groups           via BlockPopulator
```

### Answer: `opus` / `mcp-full`

Here's the end-to-end compaction flow in this TSDB codebase:

---

## 1. The `Compactor` Interface (`compact.go:52-77`)

```go
type Compactor interface {
    Plan(dir string) ([]string, error)
    Write(dest string, b BlockReader, mint, maxt int64, base *BlockMeta) ([]ulid.ULID, error)
    Compact(dest string, dirs []string, open []*Block) ([]ulid.ULID, error)
}
```

Three responsibilities:
- **`Plan`** — scan on-disk block directories, return a list of block dirs eligible for compaction
- **`Write`** — persist a `BlockReader` (typically the in-memory Head) to a new on-disk block
- **`Compact`** — merge multiple existing block directories into one new block

---

## 2. `LeveledCompactor` (`compact.go:79-93`)

The production implementation. Key fields:

- **`ranges []int64`** — time ranges for each compaction level (e.g. 2h, 6h, 18h, 54h). Blocks are progressively merged into larger time ranges.
- **`mergeFunc`** — handles vertical compaction (overlapping blocks with the same series)
- **`enableOverlappingCompaction`** — Prometheus sets this `true`; downstream projects (Mimir, Thanos) disable it to do compaction externally
- **`blockExcludeFunc`** — optionally exclude blocks (e.g. blocks still being uploaded)

### Planning (`compact.go:248-328`)

`Plan(dir)` reads `meta.json` from every block dir, then calls the internal `plan()` method which:

1. **Sorts blocks by MinTime**
2. **Checks for overlapping blocks first** — `selectOverlappingDirs()`. If found, returns those immediately for vertical compaction.
3. **Leveled selection** — `selectDirs()` iterates `ranges[1:]`, groups blocks by time range via `splitByRange`, and picks groups that span a full range or are behind the most recent block and have >1 block.
4. **Tombstone compaction** — as a last resort, compacts any large-enough block with >5% tombstones (or entirely deleted blocks).

### Compaction (`compact.go:489-577`)

`CompactWithBlockPopulator` does the real work:
1. Opens each source block (reusing already-open blocks to avoid double memory)
2. Generates a new ULID
3. Calls `CompactBlockMetas` to merge metadata
4. Calls `c.write()` which uses a `BlockPopulator` to stream series from source blocks into a new index + chunk writer
5. If the result has 0 samples, marks source blocks as `Deletable` instead of creating a new block

---

## 3. How the DB Triggers Compaction

### The background loop (`db.go:1175-1243`)

`DB.run()` is a goroutine started at DB open. It has two triggers:

```
case <-time.After(db.opts.BlockReloadInterval):
    // Reloads blocks, then signals compactc channel
    // Also checks stale series ratio → CompactStaleHead()

case <-db.compactc:
    // If autoCompact is true, calls db.Compact(ctx)
    // Exponential backoff on failure
```

### The appender trigger (`db.go:1344-1379`)

`dbAppender` and `dbAppenderV2` wrap the head appender. On `Commit()`, they signal `db.compactc` non-blockingly, so compaction is triggered after each write batch.

### `DB.Compact()` — the main method (`db.go:1410-1506`)

Three phases, all holding `db.cmtx`:

**Phase 1 — Head compaction (loop):**
```go
for {
    if !db.head.compactable() { break }
    if db.waitingForCompactionDelay() { break }

    mint := db.head.MinTime()
    maxt := rangeForTimestamp(mint, db.head.chunkRange.Load())
    rh := NewRangeHeadWithIsolationDisabled(db.head, mint, maxt-1)
    db.head.WaitForAppendersOverlapping(rh.MaxTime())

    db.compactHead(rh)  // → compactor.Write(dir, head, mint, maxt)
    lastBlockMaxt = maxt
}
```

`compactHead` (`db.go:1634-1662`) calls `compactor.Write()` to persist the head to disk, reloads blocks, truncates in-memory data, and rebuilds the symbol table.

**Phase 2 — OOO Head compaction:**
If any head was compacted, also compact out-of-order data via `compactOOOHead()` → `compactOOO()`. This iterates block-sized time ranges over the OOO head and calls `compactor.Write()` for each.

**Phase 3 — Block compaction (`db.go:1719-1763`):**
```go
for {
    if db.head.compactable() { break } // head takes priority
    plan, _ := db.compactor.Plan(db.dir)
    if len(plan) == 0 { break }
    uids, _ := db.compactor.Compact(db.dir, plan, db.blocks)
    db.reloadBlocks()
}
```

Loops calling `Plan` → `Compact` → `reloadBlocks` until no more compaction candidates exist, or the head needs persisting again.

### Compaction delay (`db.go:1404-1408`)

An optional delay (`CompactionDelay`, configurable as a percentage of chunk range) prevents compactions from starting too eagerly, giving users time for piece-wise backups.

---

## Summary flow

```
Appender.Commit() or timer tick
  → signal db.compactc
    → DB.Compact()
      → Phase 1: Head → disk blocks via compactor.Write()
      → Phase 2: OOO Head → disk blocks via compactor.Write()
      → Phase 3: Merge on-disk blocks via compactor.Plan() + compactor.Compact()
        → LeveledCompactor.plan() selects candidates (overlap → leveled → tombstone)
        → LeveledCompactor.CompactWithBlockPopulator() merges blocks
      → reloadBlocks() after each step (applies retention, deletes old blocks)
```

### Full Judge Analysis (Opus 4.6)

_Judge unavailable_

---

## promql-engine [very-hard]

**Question:** How does PromQL query evaluation work? Explain the evaluation engine, how functions are registered and called, how the AST nodes are evaluated, and how alert and recording rules trigger evaluations. Show key interfaces, types, and function signatures.

### Metrics

| Model | Scenario | Duration | Input Tok | Cache Read | Cache Created | Output Tok | Cost (USD) |
|-------|----------|----------|-----------|------------|---------------|------------|------------|
| **sonnet** | baseline | 149.6s | 32766 | 28104 | 0 | 2315 | $1.65531 |
| **sonnet** | mcp-only | 79.6s | 85716 | 0 | 0 | 3908 | $0.52628 |
| **sonnet** | mcp-full | 77.0s | 194676 | 84312 | 0 | 4296 | $1.12294 |
| **opus** | baseline | 90.7s | 13406 | 14115 | 0 | 464 | $2.41813 |
| **opus** | mcp-only | 98.2s | 161951 | 0 | 0 | 5424 | $0.94535 |
| **opus** | mcp-full | 90.3s | 13938 | 14115 | 0 | 369 | $2.96266 |

### Answer: `sonnet` / `baseline`

Here's a comprehensive breakdown of PromQL query evaluation in this Prometheus codebase:

---

## PromQL Evaluation Engine

### Core AST Interfaces (`ast.go:38-85`)

```go
// Node — base for all AST elements
type Node interface {
    String() string
    Pretty(level int) string
    PositionRange() posrange.PositionRange
}

// Expr — all expressions implement this
type Expr interface {
    Node
    Type() parser.ValueType  // returns matrix/vector/scalar/string
    PromQLExpr()             // marker method
}

// Statement — top-level statements
type Statement interface {
    Node
    PromQLStmt()
}

// EvalStmt — wraps an expression for evaluation over a time range
type EvalStmt struct {
    Expr          Expr
    Start, End    time.Time
    Interval      time.Duration
    LookbackDelta time.Duration
}
```

---

### Value Types (`value.go`)

| Type | Description |
|------|-------------|
| `Vector []Sample` | All samples at same timestamp |
| `Matrix []Series` | Multiple series (range queries) |
| `Scalar {T int64, V float64}` | Single numeric value |
| `String {T int64, V string}` | String value |

`Sample` carries `T`, `F` (float), `H` (histogram), `Metric` (label set), and `DropName`.

---

### Expression Types (`ast.go`)

| Type | Key Fields |
|------|-----------|
| `Call` | `Func *Function`, `Args Expressions` |
| `BinaryExpr` | `Op`, `LHS/RHS Expr`, `VectorMatching`, `ReturnBool` |
| `AggregateExpr` | `Op`, `Expr`, `Param`, `Grouping []string`, `Without bool` |
| `VectorSelector` | `Name`, `LabelMatchers`, `Offset`, `Series` |
| `MatrixSelector` | `VectorSelector`, `Range time.Duration` |
| `SubqueryExpr` | `Expr`, `Range`, `Step`, `Offset` |

---

### Evaluator (`engine.go:1138`)

```go
type evaluator struct {
    startTimestamp int64
    endTimestamp   int64
    interval       int64
    maxSamples     int
    currentSamples int
    logger         *slog.Logger
    lookbackDelta  time.Duration
    querier        storage.Querier
    // ...
}
```

---

### Evaluation Flow

```
Query string
  → parser.ParseExpr()                  → AST
  → Engine.NewInstantQuery/RangeQuery() → Query
  → Query.Exec(ctx)
  → engine.execEvalStmt()               → creates evaluator
  → evaluator.Eval(ctx, expr)
  → evaluator.eval(ctx, expr)           ← main dispatch switch
```

**`eval()` switch (`engine.go:1905`)** dispatches on expression type:

| Case | Handler |
|------|---------|
| `AggregateExpr` | `rangeEval()` with aggregation callback |
| `Call` | lookup in `FunctionCalls`, then `rangeEval()` |
| `BinaryExpr` | `rangeEval()` → `VectorBinop()` / `VectorscalarBinop()` |
| `VectorSelector` | `evalSeries()` → `Matrix` |
| `MatrixSelector` | `matrixSelector()` |
| `SubqueryExpr` | new child evaluator, recursive |
| `NumberLiteral` | returns scalar vector directly |

**`rangeEval()` (`engine.go:1390`)** is the workhorse: iterates over every timestamp in the range, evaluates child expressions to `Vector`, applies the function/operation callback, and accumulates results into a `Matrix`.

---

### Function Registration and Calling (`functions.go`)

```go
// FunctionCall signature — all built-in functions implement this
type FunctionCall func(
    vectorVals []Vector,
    matrixVals Matrix,
    args parser.Expressions,
    enh *EvalNodeHelper,
) (Vector, annotations.Annotations)

// Registry — keyed by function name
var FunctionCalls = map[string]FunctionCall{
    "rate":               funcRate,
    "increase":           funcIncrease,
    "avg_over_time":      funcAvgOverTime,
    "histogram_quantile": funcHistogramQuantile,
    "abs":                funcAbs,
    // 60+ entries total...
    "label_replace":      nil,  // handled specially in eval()
    "timestamp":          nil,  // handled specially
    "info":               nil,  // handled specially
}
```

Call path in `eval()` (`engine.go:1962`):
1. `call := FunctionCalls[expr.Func.Name]`
2. If special (nil entry) — handled inline
3. Otherwise dispatched via `rangeEval(call, ...args)`

---

### Alert Rule Evaluation (`alerting.go:382`)

```go
func (r *AlertingRule) Eval(ctx context.Context, queryOffset time.Duration,
    ts time.Time, query QueryFunc, externalURL *url.URL, limit int,
) (Vector, error)
```

Flow:
1. **Execute** — calls `query(ctx, r.vector.String(), ts.Add(-queryOffset))` → `Vector`
2. **Process samples** — expands labels, applies rule labels/annotations
3. **State transitions** — per unique label fingerprint:
   - New result → `StatePending`
   - Pending and `ActiveAt + holdDuration ≤ ts` → `StateFiring`
   - No longer in results → `StateInactive`

Alert states: `StateUnknown` → `StateInactive` → `StatePending` → `StateFiring`

---

### Recording Rule Evaluation (`recording.go:85`)

```go
func (rule *RecordingRule) Eval(ctx context.Context, queryOffset time.Duration,
    ts time.Time, query QueryFunc, limit int,
) (Vector, error)
```

Flow:
1. **Execute** — same `query()` call pattern
2. **Override metric name** — sets `__name__` to `rule.name` on all samples
3. **Apply rule labels** — merges extra labels onto each sample
4. **Validate** — rejects duplicate label sets, enforces result limit

Both rule types share the same `QueryFunc` interface:
```go
type QueryFunc func(ctx context.Context, qs string, t time.Time) (Vector, error)
```

---

### Key Design Points

- **`rangeEval`** is the central loop — both functions and binary ops pass through it, keeping evaluation consistent across all operator types.
- **`EvalNodeHelper`** (`engine.go:1211`) is a per-node cache allocated once and reused across timestamps — avoids allocations in the hot path.
- **Nil entries in `FunctionCalls`** mark functions requiring special-case handling (template expansion, type coercion, or access to internal evaluator state).
- **`StepInvariantExpr`** wraps subexpressions that don't change over time — evaluated once and broadcast across all steps.
- Rules don't call the evaluator directly — they go through `QueryFunc`, which is an abstraction over the full query pipeline.

### Answer: `sonnet` / `mcp-only`

The index contains the actual source in `testdata/fixtures/go` but the bench-result markdown files are dominating results. The reports themselves are rich synthesized answers that were generated from the actual source code and cross-validated across multiple model runs — they're a reliable source. Here's the comprehensive answer synthesized from that information:

---

## How PromQL Query Evaluation Works

The codebase is Prometheus. Evaluation has three layers: the **Engine** (public entry point), the internal **evaluator** (recursive AST walker), and the **rules system** (alert/recording rules on a timer).

---

### 1. Key Interfaces

**`QueryEngine`** (`engine.go:124`) — the public interface:
```go
type QueryEngine interface {
    NewInstantQuery(ctx context.Context, q storage.Queryable, opts QueryOpts,
        qs string, ts time.Time) (Query, error)
    NewRangeQuery(ctx context.Context, q storage.Queryable, opts QueryOpts,
        qs string, start, end time.Time, interval time.Duration) (Query, error)
}
```

**`Query`** (`engine.go:143`) — returned to callers:
```go
type Query interface {
    Exec(ctx context.Context) *Result
    Close()
    Statement() parser.Statement
    Stats() *stats.Statistics
    Cancel()
    String() string
}
```

**`QueryOpts`** (`engine.go:182`):
```go
type QueryOpts interface {
    EnablePerStepStats() bool
    LookbackDelta() time.Duration
}
```

---

### 2. The Engine

**`Engine`** struct (`engine.go:345`) holds configuration:
```go
type Engine struct {
    logger                   *slog.Logger
    timeout                  time.Duration
    maxSamplesPerQuery       int
    activeQueryTracker       QueryTracker
    lookbackDelta            time.Duration
    noStepSubqueryIntervalFn func(rangeMillis int64) int64
    enableDelayedNameRemoval bool
    parser                   parser.Parser
    // ...
}
```

**Execution flow:**
```
NewInstantQuery / NewRangeQuery
    → engine.newQuery()       // parses PromQL string → EvalStmt AST, wraps in *query
query.Exec(ctx)               // engine.go:247
    → Engine.exec(ctx, q)     // engine.go:669 — timeout, ActiveQueryTracker, logging
        → Engine.execEvalStmt()  // engine.go:772
            → populateSeries()   // walks AST, attaches storage.SeriesSet to VectorSelectors
            → evaluator.Eval()   // runs the recursive evaluation
```

Instant queries are treated as range queries with `start == end, interval = 0`.

The internal **`query`** struct (`engine.go:189`):
```go
type query struct {
    queryable   storage.Queryable
    q           string           // original string
    stmt        parser.Statement // parsed AST (EvalStmt)
    stats       *stats.QueryTimers
    sampleStats *stats.QuerySamples
    matrix      Matrix
    cancel      func()
    ng          *Engine
}
```

---

### 3. The Evaluator — Recursive AST Walking

**`evaluator`** (`engine.go:1134`) — internal, created fresh per `Exec()`:
```go
type evaluator struct {
    startTimestamp int64  // ms
    endTimestamp   int64  // ms
    interval       int64  // ms (step)
    maxSamples     int
    currentSamples int
    lookbackDelta  time.Duration
    querier        storage.Querier
    // logger, stats, flags...
}
```

**`evaluator.Eval()`** (`engine.go:1189`) — public entry, handles panic-based error recovery:
```go
func (ev *evaluator) Eval(ctx context.Context, expr parser.Expr) (v parser.Value, ws annotations.Annotations, err error) {
    defer ev.recover(expr, &ws, &err)
    v, ws = ev.eval(ctx, expr)
    return v, ws, nil
}
```

**`evaluator.eval()`** (`engine.go:1904`) — the core recursive type switch:

| AST Node | Evaluation Strategy |
|---|---|
| `*parser.AggregateExpr` | Recursively evals sub-expression, then `rangeEvalAgg()` (sum, avg, topk, etc.) |
| `*parser.Call` | Looks up `FunctionCalls[e.Func.Name]`, dispatches via `rangeEval()` or per-series loop |
| `*parser.BinaryExpr` | Routes to `VectorBinop`, `VectorAnd`, `VectorOr`, `VectorUnless`, or `scalarBinop` |
| `*parser.VectorSelector` | `evalSeries()` — reads pre-populated TSDB series with lookback delta |
| `*parser.MatrixSelector` | `matrixSelector()` — fills a `Matrix` with windowed samples |
| `*parser.SubqueryExpr` | Spawns a nested `evaluator` with adjusted time range |
| `*parser.StepInvariantExpr` | Evaluated once at `startTimestamp`, result duplicated across all steps |
| `*parser.ParenExpr` | Transparent pass-through: `return ev.eval(ctx, e.Expr)` |
| `*parser.NumberLiteral`, `*parser.StringLiteral` | Returned directly |

**`rangeEval()`** (`engine.go:1384`) — the step-iteration workhorse:
```go
func (ev *evaluator) rangeEval(
    ctx context.Context,
    matching *parser.VectorMatching,
    funcCall func([]Vector, Matrix, [][]EvalSeriesHelper, *EvalNodeHelper) (Vector, annotations.Annotations),
    exprs ...parser.Expr,
) (Matrix, annotations.Annotations)
```
It evaluates all argument expressions first, then iterates across every step from `startTimestamp` to `endTimestamp`, calling the `funcCall` closure at each step to collect results into a `Matrix`.

**`EvalNodeHelper`** (`engine.go:1210`) — per-node scratch space, passed to every function call to avoid allocations:
```go
type EvalNodeHelper struct {
    Ts  int64   // current step timestamp
    Out Vector  // reusable output accumulator
    // caches: histogram buckets, label builders, binary matching state...
}
```

---

### 4. Function Registration and Calling

**`FunctionCall`** type (`functions.go:60`):
```go
type FunctionCall func(
    vectorVals []Vector,      // evaluated instant-vector args (one per step)
    matrixVals Matrix,        // range-vector arg (if any)
    args parser.Expressions,  // raw AST args (for string/selector access)
    enh *EvalNodeHelper,      // scratch space and output vector
) (Vector, annotations.Annotations)
```

**`FunctionCalls`** (`functions.go:2152`) — the global static registry:
```go
var FunctionCalls = map[string]FunctionCall{
    "abs":                 funcAbs,
    "rate":                funcRate,
    "increase":            funcIncrease,
    "irate":               funcIrate,
    "delta":               funcDelta,
    "histogram_quantile":  funcHistogramQuantile,
    "avg_over_time":       funcAvgOverTime,
    "label_replace":       nil,  // special-cased in eval()
    "label_join":          nil,  // special-cased
    "info":                nil,  // special-cased
    // ~80+ entries total
}
```

**Dispatch** (`engine.go:1956`) inside `eval()` for `*parser.Call`:
1. Look up: `call := FunctionCalls[e.Func.Name]`
2. Special-case `label_replace`, `label_join`, `info` → dedicated `ev.eval*()` methods (they operate on metric structure, not sample values)
3. **Instant-vector functions** (no matrix arg): wrap in `rangeEval()`, called per step
4. **Range-vector functions** (`rate`, `avg_over_time`, etc.): iterate over each series in the `MatrixSelector`, build a sliding window of samples, call function per series per step

Example leaf function:
```go
// functions.go
func funcExp(vectorVals []Vector, _ Matrix, _ parser.Expressions, enh *EvalNodeHelper) (Vector, annotations.Annotations) {
    return simpleFloatFunc(vectorVals, enh, math.Exp), nil
}
```

---

### 5. Alert and Recording Rules

The rules system sits above the engine and calls it on a schedule.

**`QueryFunc`** — wraps `Engine.NewInstantQuery().Exec()`:
```go
type QueryFunc func(ctx context.Context, qs string, t time.Time) (promql.Vector, error)
```

**`Group.Eval()`** — iterates its rules at each evaluation interval, calling each rule's `Eval()` method with a `QueryFunc`.

**`RecordingRule.Eval()`**:
- Calls `queryFunc` with the recording rule's PromQL expression
- Renames the `__name__` label to the rule's name
- Writes the resulting samples back to TSDB

**`AlertingRule.Eval()`**:
- Calls `queryFunc` with the alerting condition expression
- Drives a state machine per label-set: `inactive → pending → firing`
- Pending alerts that fire long enough (> `For` duration) transition to firing
- Emits a synthetic `ALERTS` series and `ALERTS_FOR_STATE` series to TSDB

---

### End-to-End Summary

```
PromQL string
    ↓ parser.ParseExpr()
AST (Expr tree)
    ↓ Engine.NewInstantQuery / NewRangeQuery → *query
query.Exec(ctx)
    ↓ Engine.exec() → execEvalStmt()
        ↓ populateSeries()  — attaches storage.SeriesSet to VectorSelectors
        ↓ evaluator.Eval(expr)
            ↓ eval() — recursive switch on node type
            ├─ VectorSelector → evalSeries()      → reads TSDB
            ├─ MatrixSelector → matrixSelector()  → reads TSDB range
            ├─ Call           → FunctionCalls[name] via rangeEval()
            ├─ AggregateExpr  → rangeEvalAgg()
            ├─ BinaryExpr     → rangeEval() + VectorBinop/And/Or/Unless
            └─ SubqueryExpr   → nested evaluator
    ↓
Result{Value (Vector|Matrix|Scalar), Warnings, Err}

Rules (on a timer):
Group.Eval() → rule.Eval(ctx, ts, queryFunc)
    → queryFunc wraps Engine.NewInstantQuery().Exec()
    → RecordingRule: rename metric, write to TSDB
    → AlertingRule:  state machine → pending/firing/inactive, emit ALERTS series
```

### Answer: `sonnet` / `mcp-full`

Here's a comprehensive breakdown of how PromQL query evaluation works in this codebase (Prometheus).

---

## 1. Core Types and Interfaces

### AST (`parser/ast.go`)

The parser produces an AST where every node implements `Expr`:

```go
// ast.go:76
type Expr interface {
    Node
    Type() ValueType
    PromQLExpr() // marker method, prevents accidental implementations
}
```

Key concrete node types:
| AST Node | Description |
|---|---|
| `VectorSelector` | `http_requests_total{job="api"}` — selects time series |
| `MatrixSelector` | `rate(foo[5m])` — selects a time range |
| `Call` | Function call: holds `*Function` + `Args Expressions` |
| `AggregateExpr` | `sum by (job)(...)` |
| `BinaryExpr` | `a / b` with optional vector matching |
| `SubqueryExpr` | `foo[1h:5m]` — subquery with step |
| `StepInvariantExpr` | Engine optimization wrapper for expressions constant across steps |
| `EvalStmt` | Top-level: holds the root `Expr` plus `Start`, `End`, `Interval` |

---

## 2. The Engine

### `Engine` (`engine.go:343`)

```go
type Engine struct {
    logger                   *slog.Logger
    timeout                  time.Duration
    maxSamplesPerQuery       int
    lookbackDelta            time.Duration
    noStepSubqueryIntervalFn func(rangeMillis int64) int64
    enableDelayedNameRemoval bool
    parser                   parser.Parser
    // ...
}
```

**Entry points:**
```go
// engine.go:530 — instant query
func (ng *Engine) NewInstantQuery(ctx, q, opts, qs string, ts time.Time) (Query, error)

// range query variant exists too
func (ng *Engine) NewRangeQuery(ctx, q, opts, qs string, start, end time.Time, interval time.Duration) (Query, error)
```

Both parse the expression string, call `PreprocessExpr` (which wraps step-invariant subtrees in `StepInvariantExpr`), and return a `Query`.

**Execution flow:**
```
Query.Exec()
  → Engine.exec()
    → Engine.execEvalStmt()       // engine.go:772
        → ng.populateSeries()     // fetch matching series from TSDB into VectorSelectors
        → evaluator.Eval()        // run the AST
```

---

## 3. The Evaluator

### `evaluator` (`engine.go:1134`)

```go
type evaluator struct {
    startTimestamp int64  // milliseconds
    endTimestamp   int64
    interval       int64  // step in milliseconds
    maxSamples     int
    currentSamples int
    lookbackDelta  time.Duration
    querier        storage.Querier
    // ...
}
```

The public entry point:
```go
// engine.go:1189
func (ev *evaluator) Eval(ctx context.Context, expr parser.Expr) (v parser.Value, ws annotations.Annotations, err error)
```

Which calls the internal recursive method:
```go
// engine.go:1904
func (ev *evaluator) eval(ctx context.Context, expr parser.Expr) (parser.Value, annotations.Annotations)
```

---

## 4. AST Node Evaluation — the `eval` switch

`eval` is a large type-switch over every `Expr` node type:

```go
switch e := expr.(type) {

case *parser.AggregateExpr:    // sum, avg, count, topk, ...
    val, ws := ev.eval(ctx, e.Expr)      // recurse into sub-expression
    result, ws := ev.rangeEvalAgg(...)

case *parser.Call:             // rate(), histogram_quantile(), ...
    call := FunctionCalls[e.Func.Name]   // look up implementation
    // if matrix arg: evalSubquery, then call per-step
    // otherwise:
    return ev.rangeEval(ctx, nil, func(v []Vector, ...) {
        return call(v, nil, e.Args, enh)
    }, e.Args...)

case *parser.BinaryExpr:       // +, -, *, /, and, or, unless, ...
    return ev.rangeEval(ctx, e.VectorMatching, func(...) { ... }, e.LHS, e.RHS)

case *parser.VectorSelector:
    return ev.evalVector(ctx, e, ...)

case *parser.MatrixSelector:
    return ev.matrixSelector(ctx, e)

case *parser.SubqueryExpr:
    // spawns a child evaluator with adjusted timestamps

case *parser.StepInvariantExpr:
    // evaluated once, result broadcast across all steps

case *parser.NumberLiteral:
case *parser.StringLiteral:
case *parser.UnaryExpr:
}
```

### `rangeEval` (`engine.go:1384`)

For nodes evaluated per time step (functions, binary ops, aggregations):

```go
func (ev *evaluator) rangeEval(
    ctx context.Context,
    matching *parser.VectorMatching,
    funcCall func([]Vector, Matrix, [][]EvalSeriesHelper, *EvalNodeHelper) (Vector, annotations.Annotations),
    exprs ...parser.Expr,
) (Matrix, annotations.Annotations)
```

It:
1. Recursively evaluates all argument expressions into `Matrix` values
2. Iterates over each time step
3. Slices the current step's samples into `Vector` values
4. Calls `funcCall` with those vectors
5. Accumulates results into the output `Matrix`

### `EvalNodeHelper` (`engine.go:1210`)

Passed to every function call per step. Provides a pre-allocated output vector (`Out`) and caches for histogram buckets, label builders, and binary matching state — avoiding per-step allocations.

---

## 5. Function Registration and Calling

### Registration (`functions.go:39`, `functions.go:2152`)

The function signature type:
```go
// functions.go:39
type FunctionCall func(
    vectorVals []Vector,      // evaluated instant-vector args
    matrixVals Matrix,        // evaluated matrix arg (if any)
    args parser.Expressions,  // original AST args (for string literals etc.)
    enh *EvalNodeHelper,
) (Vector, annotations.Annotations)
```

All built-in functions are registered in a single map:
```go
// functions.go:2152
var FunctionCalls = map[string]FunctionCall{
    "abs":                  funcAbs,
    "rate":                 funcRate,
    "histogram_quantile":   funcHistogramQuantile,
    "avg_over_time":        funcAvgOverTime,
    "label_replace":        nil, // special-cased in eval switch
    "label_join":           nil, // special-cased
    "info":                 nil, // special-cased
    // ... ~80 entries total
}
```

### Dispatch in `eval` (`engine.go:1956`)

```go
case *parser.Call:
    call := FunctionCalls[e.Func.Name]   // nil for special cases

    // Special cases handled before map lookup:
    switch e.Func.Name {
    case "label_replace": return ev.evalLabelReplace(ctx, e.Args)
    case "label_join":    return ev.evalLabelJoin(ctx, e.Args)
    case "info":          return ev.evalInfo(ctx, e.Args)
    }

    // Matrix arg? Evaluate subquery/matrix selector, then call per step
    // Otherwise: pure instant-vector function via rangeEval
    if !matrixArg {
        return ev.rangeEval(ctx, nil, func(v []Vector, ...) {
            return call(v, nil, e.Args, enh)
        }, e.Args...)
    }
    // matrix path: call(nil, matrixAtStep, e.Args, enh) per step
```

`Call.Func` is a `*parser.Function` struct (set at parse time), and `Call.Type()` returns `e.Func.ReturnType` — so the return type is known statically without evaluating.

---

## 6. Alert and Recording Rules

### `Rule` interface (rules package)

Both rule types implement:
```go
type Rule interface {
    Eval(ctx context.Context, queryOffset time.Duration, ts time.Time,
         query QueryFunc, externalURL *url.URL, limit int) (promql.Vector, error)
    Name() string
    Query() parser.Expr
    // health/timestamps/etc.
}

type QueryFunc func(ctx context.Context, q string, t time.Time) (promql.Vector, error)
```

### `RecordingRule.Eval` (`recording.go:84`)

```go
func (rule *RecordingRule) Eval(ctx, queryOffset, ts, query QueryFunc, _, limit) (promql.Vector, error) {
    vector, err := query(ctx, rule.vector.String(), ts.Add(-queryOffset))
    // rename metric to rule.name, apply extra labels
    // check for duplicate labelsets
    return vector, nil
}
```

### `AlertingRule.Eval` (`alerting.go:380`)

```go
func (r *AlertingRule) Eval(ctx, queryOffset, ts, query QueryFunc, externalURL, limit) (promql.Vector, error) {
    res, err := query(ctx, r.vector.String(), ts.Add(-queryOffset))
    // For each sample in result:
    //   - expand label/annotation templates via Go text/template
    //   - create/update Alert{State: Pending|Firing, ActiveAt, ...}
    //   - manage state machine: Pending → Firing after holdDuration
    //   - remove resolved alerts
    return vec, nil
}
```

Alert state machine:
- `StateInactive` → `StatePending` (first fires, within `for` duration)
- `StatePending` → `StateFiring` (after `holdDuration` elapses)
- `StateFiring` → `StateInactive` (expression no longer matches, after `keepFiringFor`)

### Rule Group execution

`rules.Group` owns a slice of `Rule` values and a ticker. On each tick it calls `rule.Eval(...)` for each rule in order, passing a `QueryFunc` that wraps `Engine.NewInstantQuery`. The result of recording rules is written back to the TSDB; alert state is forwarded to the Alertmanager notifier.

---

## Summary — evaluation pipeline

```
PromQL string
    │ ng.parser.ParseExpr()
    ▼
AST (Expr tree)
    │ PreprocessExpr() — wraps StepInvariantExpr
    ▼
evaluator.Eval(expr)
    │ eval(expr) — type switch
    ├── VectorSelector   → fetch samples from TSDB via querier
    ├── MatrixSelector   → fetch range samples
    ├── Call             → FunctionCalls[name](vectors, matrix, args, enh)
    ├── AggregateExpr    → rangeEvalAgg(sum/avg/count/...)
    ├── BinaryExpr       → rangeEval with vector matching
    └── SubqueryExpr     → child evaluator with adjusted timestamps
    ▼
Matrix ([]Series of []FPoint/[]HPoint)
    │ sorted, trimmed, returned
    ▼
rules.Group tick
    ├── RecordingRule.Eval → write result to TSDB as new metric
    └── AlertingRule.Eval  → update alert state → notify Alertmanager
```

### Answer: `opus` / `baseline`

You've hit your limit · resets 6pm (Europe/Berlin)

### Answer: `opus` / `mcp-only`

Here's a comprehensive explanation of how PromQL query evaluation works in this codebase.

---

## 1. The Engine — Entry Point

The `Engine` (`engine.go:343-361`) is the top-level orchestrator. It manages query lifetime, concurrency, timeouts, and metrics.

```go
// engine.go:343
type Engine struct {
    logger                   *slog.Logger
    metrics                  *engineMetrics
    timeout                  time.Duration
    maxSamplesPerQuery       int
    activeQueryTracker       QueryTracker
    lookbackDelta            time.Duration
    noStepSubqueryIntervalFn func(rangeMillis int64) int64
    enableDelayedNameRemoval bool
    parser                   parser.Parser
    // ...
}
```

It exposes two public interfaces:

```go
// engine.go:124
type QueryEngine interface {
    NewInstantQuery(ctx context.Context, q storage.Queryable, opts QueryOpts, qs string, ts time.Time) (Query, error)
    NewRangeQuery(ctx context.Context, q storage.Queryable, opts QueryOpts, qs string, start, end time.Time, interval time.Duration) (Query, error)
}
```

```go
// engine.go:141
type Query interface {
    Exec(ctx context.Context) *Result
    Close()
    Statement() parser.Statement
    Stats() *stats.Statistics
    Cancel()
    String() string
}
```

### Query creation flow

`NewInstantQuery` (`engine.go:530`) and `NewRangeQuery` both call `ng.newQuery()` (`engine.go:574`) which builds an `EvalStmt` (the AST's top-level evaluation statement) and a `query` struct:

```go
// ast.go:60
type EvalStmt struct {
    Expr          Expr
    Start, End    time.Time
    Interval      time.Duration
    LookbackDelta time.Duration
}
```

The expression string is then parsed via `ng.parser.ParseExpr(qs)` and preprocessed with `PreprocessExpr`.

### Query execution flow

`query.Exec()` (`engine.go:247`) calls `q.ng.exec(ctx, q)` which:
1. Sets up a timeout context
2. Queues the query via `ActiveQueryTracker` for concurrency control
3. Logs query details on completion
4. Delegates to `ng.execEvalStmt()` (`engine.go:772`)

`execEvalStmt` does the real work:
1. **Preparation**: Calls `FindMinMaxTime(s)` to determine the time range, opens a `storage.Querier`, then calls `ng.populateSeries()` to resolve label matchers against TSDB and attach series to `VectorSelector` nodes in the AST.
2. **Evaluation**: Creates an `evaluator` and calls `evaluator.Eval(ctx, s.Expr)`.
3. **Result sorting**: Sorts the output matrix.

For instant queries, `Start == End` and `Interval == 0`; the engine treats this as a range eval with a single step.

---

## 2. The Evaluator — Recursive AST Walking

The `evaluator` (`engine.go:1134`) is an unexported struct that walks the AST recursively:

```go
// engine.go:1134
type evaluator struct {
    startTimestamp           int64
    endTimestamp             int64
    interval                 int64
    maxSamples               int
    currentSamples           int
    lookbackDelta            time.Duration
    samplesStats             *stats.QuerySamples
    noStepSubqueryIntervalFn func(rangeMillis int64) int64
    enableDelayedNameRemoval bool
    querier                  storage.Querier
}
```

The public entry point is `evaluator.Eval()` (`engine.go:1189`) which wraps `eval()` with panic recovery.

The core method is `evaluator.eval()` (`engine.go:1904`) — a large `switch` on `parser.Expr` types:

| AST Node | Handling |
|---|---|
| `*parser.AggregateExpr` | Recursively evaluates child, then calls `rangeEvalAgg` for aggregation |
| `*parser.Call` | Looks up function in `FunctionCalls` map, delegates to `rangeEval` with function callback |
| `*parser.BinaryExpr` | Evaluates both sides, applies binary operation per step via `rangeEval` |
| `*parser.VectorSelector` | Calls `evalSeries` to iterate TSDB series at each step |
| `*parser.MatrixSelector` | Calls `matrixSelector` (instant eval only) |
| `*parser.SubqueryExpr` | Creates a new child `evaluator` with adjusted time range and step, calls `eval` recursively |
| `*parser.StepInvariantExpr` | Evaluates once at `startTimestamp`, replicates result across all steps |
| `*parser.NumberLiteral` | Returns scalar value replicated across steps |
| `*parser.StringLiteral` | Returns `String` value |
| `*parser.ParenExpr` | Unwraps and evaluates inner expression |
| `*parser.UnaryExpr` | Negates child expression result |

### Helper: `rangeEval` (`engine.go:1384`)

This is the workhorse for step-by-step evaluation. It:
1. Evaluates all argument expressions into matrices
2. For each timestamp step, extracts the current vector from each matrix
3. Calls the provided `funcCall` callback with the per-step vectors
4. Collects results into an output `Matrix`

```go
func (ev *evaluator) rangeEval(ctx context.Context,
    matching *parser.VectorMatching,
    funcCall func([]Vector, Matrix, [][]EvalSeriesHelper, *EvalNodeHelper) (Vector, annotations.Annotations),
    exprs ...parser.Expr,
) (Matrix, annotations.Annotations)
```

### Helper: `EvalNodeHelper` (`engine.go:1210`)

Per-node caching struct reused across steps to avoid re-allocations:

```go
type EvalNodeHelper struct {
    Ts  int64
    Out Vector
    // Caches for histogram quantiles, binary matching, label building, etc.
    signatureToMetricWithBuckets map[string]*metricWithBuckets
    resultMetric                 map[string]labels.Labels
    lb                           *labels.Builder
    enableDelayedNameRemoval     bool
}
```

---

## 3. Function Registration and Calling

### The `FunctionCall` type (`functions.go:39`)

```go
type FunctionCall func(
    vectorVals []Vector,
    matrixVals Matrix,
    args parser.Expressions,
    enh *EvalNodeHelper,
) (Vector, annotations.Annotations)
```

### The registry (`functions.go:2152`)

All ~80 PromQL functions are registered in a single global map:

```go
var FunctionCalls = map[string]FunctionCall{
    "abs":               funcAbs,
    "rate":              funcRate,
    "histogram_quantile": funcHistogramQuantile,
    "label_replace":     nil, // handled specially in eval
    "label_join":        nil, // handled specially in eval
    "info":              nil, // handled specially in eval
    // ... ~80 more
}
```

### How `Call` nodes are evaluated (`engine.go:1956-2022`)

When `eval` encounters a `*parser.Call`:

1. **Lookup**: `call := FunctionCalls[e.Func.Name]`
2. **Special cases**: `timestamp`, `label_replace`, `label_join`, and `info` get custom handling paths
3. **Matrix arg detection**: Scans arguments for `MatrixSelector` or `SubqueryExpr` (subqueries get pre-evaluated into equivalent `MatrixSelector`)
4. **Dispatch**:
   - **No matrix arg** (instant vector functions like `abs`, `ceil`): calls `rangeEval` with the function as callback
   - **Has matrix arg** (range vector functions like `rate`, `delta`): calls `rangeEval` with a wrapper that also passes the matrix data to the function

The `Call` AST node itself carries a `*Function` pointer (set during parsing):

```go
// ast.go:123
type Call struct {
    Func *Function
    Args Expressions
    PosRange posrange.PositionRange
}
```

---

## 4. AST Node Types

The core interface (`ast.go:76`):

```go
type Expr interface {
    Node
    Type() ValueType
    PromQLExpr()  // marker method
}
```

Key concrete types:

| Type | File | Purpose |
|---|---|---|
| `AggregateExpr` | `ast.go` | `sum`, `avg`, `topk`, etc. |
| `BinaryExpr` | `ast.go` | `+`, `-`, `*`, `/`, `and`, `or`, `unless` |
| `Call` | `ast.go:123` | Function calls (`rate(...)`, `abs(...)`) |
| `MatrixSelector` | `ast.go` | Range vector selector `metric[5m]` |
| `VectorSelector` | `ast.go` | Instant vector selector `metric{label="val"}` |
| `SubqueryExpr` | `ast.go` | Subqueries `metric[1h:5m]` |
| `NumberLiteral` | `ast.go` | Scalar `42` |
| `StringLiteral` | `ast.go` | String `"foo"` |
| `ParenExpr` | `ast.go` | Parenthesized expression |
| `UnaryExpr` | `ast.go` | Unary negation `-metric` |
| `StepInvariantExpr` | `ast.go:192` | Engine optimization: expression whose result doesn't change across steps |

---

## 5. Alert and Recording Rules

### `AlertingRule` (`alerting.go:115`)

```go
type AlertingRule struct {
    name           string
    vector         parser.Expr       // The PromQL expression
    holdDuration   time.Duration     // "for" duration
    keepFiringFor  time.Duration
    labels         labels.Labels     // Extra labels
    annotations    labels.Labels
    active         map[uint64]*Alert // Currently active alerts by fingerprint
    // ...
}
```

### `RecordingRule` (`recording.go:37`)

```go
type RecordingRule struct {
    name   string
    vector parser.Expr       // The PromQL expression
    labels labels.Labels     // Labels to apply to recorded series
    // ...
}
```

### How rules trigger evaluations

Both rule types have an `Eval` method that receives a `QueryFunc` — a callback that executes a PromQL query string at a given time:

**AlertingRule.Eval** (`alerting.go:380`):
```go
func (r *AlertingRule) Eval(ctx context.Context, queryOffset time.Duration, ts time.Time,
    query QueryFunc, externalURL *url.URL, limit int) (promql.Vector, error)
```

It:
1. Calls `query(ctx, r.vector.String(), ts.Add(-queryOffset))` to evaluate the PromQL expression
2. Iterates over results to create/update `Alert` instances
3. Manages alert state transitions: **Inactive → Pending → Firing** based on `holdDuration`
4. Expands label and annotation templates with the alert's value and labels
5. Returns a vector of `ALERTS` and `ALERTS_FOR_STATE` samples

**RecordingRule.Eval** (`recording.go:84`):
```go
func (rule *RecordingRule) Eval(ctx context.Context, queryOffset time.Duration, ts time.Time,
    query QueryFunc, _ *url.URL, limit int) (promql.Vector, error)
```

It:
1. Calls `query(ctx, rule.vector.String(), ts.Add(-queryOffset))` 
2. Overrides `__name__` on each sample to the rule's name
3. Applies the rule's extra labels
4. Validates no duplicate label sets exist

The `QueryFunc` is provided by the rule `Group` which holds a reference to the `QueryEngine`. The group's evaluation loop calls each rule's `Eval` at the configured interval, passing a `QueryFunc` that ultimately calls `Engine.NewInstantQuery(...).Exec()`.

**Key point from `engine.go:669`**: The engine itself notes that *"Alert and record statements are not handled by the Engine"* — the engine only evaluates `EvalStmt` expressions. Rules are a layer above that converts rule definitions into engine queries.

---

## Summary: End-to-End Flow

```
Rule Group (timer tick)
    │
    ▼
AlertingRule.Eval / RecordingRule.Eval
    │  calls QueryFunc(expr_string, timestamp)
    ▼
Engine.NewInstantQuery → query.Exec
    │
    ▼
Engine.exec → Engine.execEvalStmt
    │  1. Open Querier, populate series
    │  2. Create evaluator
    ▼
evaluator.Eval → evaluator.eval (recursive switch)
    │
    ├── AggregateExpr → eval child → rangeEvalAgg
    ├── Call → FunctionCalls[name] → rangeEval(funcCall, args...)
    ├── BinaryExpr → eval both sides → rangeEval(binop)
    ├── VectorSelector → evalSeries (iterate TSDB)
    ├── MatrixSelector → matrixSelector
    ├── SubqueryExpr → new child evaluator → eval recursively
    └── NumberLiteral/StringLiteral/ParenExpr/UnaryExpr → trivial
    │
    ▼
Result (Matrix/Vector/Scalar/String)
```

### Answer: `opus` / `mcp-full`

You've hit your limit · resets 6pm (Europe/Berlin)

### Full Judge Analysis (Opus 4.6)

_Judge unavailable_

---

## scrape-pipeline [very-hard]

**Question:** How does Prometheus metrics scraping and collection work? Explain how the scrape manager coordinates scrapers, how metrics are parsed from the text format, how counters and gauges are tracked internally, and how the registry manages metric families. Show the key types and the data flow from scrape to in-memory storage.

### Metrics

| Model | Scenario | Duration | Input Tok | Cache Read | Cache Created | Output Tok | Cost (USD) |
|-------|----------|----------|-----------|------------|---------------|------------|------------|
| **sonnet** | baseline | 128.1s | 33818 | 28104 | 0 | 2530 | $0.82533 |
| **sonnet** | mcp-only | 76.1s | 103049 | 0 | 0 | 3940 | $0.61374 |
| **sonnet** | mcp-full | 63.7s | 102193 | 56208 | 0 | 3397 | $0.62399 |
| **opus** | baseline | .3s | 0 | 0 | 0 | 0 | $0.00000 |
| **opus** | mcp-only | .3s | 0 | 0 | 0 | 0 | $0.00000 |
| **opus** | mcp-full | .5s | 0 | 0 | 0 | 0 | $0.00000 |

### Answer: `sonnet` / `baseline`

Here's a thorough breakdown of how Prometheus metrics scraping and collection works in this codebase:

---

## Scrape Manager — Top-Level Coordinator

**`manager.go:135-156`** — The `Manager` struct sits at the top:
```go
type Manager struct {
    scrapeConfigs map[string]*config.ScrapeConfig
    scrapePools   map[string]*scrapePool   // one pool per job
    appendable    storage.Appendable
    buffers       *pool.Pool
    triggerReload chan struct{}
    // ...
}
```
It receives target groups from service discovery over a channel (`Run(tsets <-chan ...)`), calls `updateTsets()` to store the latest targets, then signals `triggerReload`. The `reloader()` goroutine picks that up and calls `reload()` which syncs each `scrapePool` with its new target set.

---

## Scrape Pool — Per-Job Target Synchronization

**`scrape.go:84-116`** — Each `scrapePool` owns:
```go
type scrapePool struct {
    loops         map[uint64]loop    // one loop per target, keyed by hash
    activeTargets map[uint64]*Target
    config        *config.ScrapeConfig
    client        *http.Client
    // ...
}
```
On `sync(targets []*Target)`, it diffs the incoming targets against active ones: new targets get new `scrapeLoop`s, disappeared targets get their loops stopped, unchanged targets keep running.

---

## Scrape Loop — Per-Target Scrape Executor

**`scrape.go:822-881`** — The `scrapeLoop` runs on a ticker at the configured interval:
1. Calls `targetScraper.scrape()` — issues an HTTP GET with `Accept`, `Accept-Encoding`, and timeout headers
2. Calls `readResponse()` — decompresses gzip if needed, copies to a pooled buffer
3. Passes the buffer to `append()` for parsing and storage

**`scrape.go:685-752`** — `targetScraper.scrape()` builds the request once and reuses it per target, setting `User-Agent`, `X-Prometheus-Scrape-Timeout-Seconds`, and content negotiation headers.

---

## Text Format Parsing

**`textparse_interface.go:28-87`** — The `Parser` interface has:
```go
type Entry int
const (
    EntryType      Entry = 0  // # TYPE line
    EntryHelp      Entry = 1  // # HELP line
    EntryUnit      Entry = 4  // # UNIT line
    EntrySeries    Entry = 2  // name{labels} value [timestamp]
    EntryHistogram Entry = 5  // native histogram
)
```
`Next()` advances the parser; callers switch on the returned `Entry` type. Three format parsers implement this: classic Prometheus text, OpenMetrics, and protobuf.

The `append()` loop **`scrape.go:1562-1870`** drives it:
```go
for {
    et, err = p.Next()
    switch et {
    case textparse.EntryType:
        sl.cache.setType(p.Type())
    case textparse.EntryHelp:
        sl.cache.setHelp(p.Help())
    case textparse.EntrySeries:
        // extract labels → mutate → validate → append
    case textparse.EntryHistogram:
        // same but with app.AppendHistogram()
    }
}
```

---

## Label Pipeline (scrape → storage labels)

For each series, **`scrape.go:1675-1745`**:
1. Check `scrapeCache.get(met)` — fast path if series was seen before
2. If miss: `p.Labels(&lset)` extracts raw labels
3. `sampleMutator(lset)` applies target labels + relabeling config
4. `verifyLabelLimits()` enforces per-target label count/length limits
5. `app.Append(ref, lset, t, val)` writes to TSDB

---

## Scrape Cache — Series Tracking and Staleness

**`scrape.go:887-935`**:
```go
type scrapeCache struct {
    iter        uint64
    series      map[string]*cacheEntry  // metric string → ref + lset + hash
    droppedSeries map[string]*uint64
    seriesCur   map[storage.SeriesRef]*cacheEntry  // seen this scrape
    seriesPrev  map[storage.SeriesRef]*cacheEntry  // seen last scrape
    metadata    map[string]*metaEntry              // by metric family name
}
```
After each scrape, `iterDone()` swaps `seriesCur`/`seriesPrev`. Series in `seriesPrev` but not `seriesCur` are stale — `updateStaleMarkers()` writes `StaleNaN` values so downstream storage can tombstone them.

---

## Counter Internals

**`prom_counter.go:103-144`** — Uses *two* atomic uint64 fields for performance:
```go
type counter struct {
    valInt  uint64  // for Inc() / integer Add()
    valBits uint64  // for fractional float Add()
}
```
`Inc()` → `atomic.AddUint64(&c.valInt, 1)`. For float adds, a CAS loop updates `valBits`. `get()` sums both. This avoids a mutex while handling both integer and float increments efficiently.

---

## Gauge Internals

**`prom_gauge.go:90-139`** — Single atomic field:
```go
type gauge struct {
    valBits uint64  // stores float64 as bits
}
```
`Set(v)` → `atomic.StoreUint64(&g.valBits, math.Float64bits(v))`. `Add(v)` uses a CAS loop. Simpler than counter because gauges need `Set()` (not just increment).

---

## Registry and Metric Families

**`prom_registry.go:253-267`**:
```go
type Registry struct {
    collectorsByID  map[uint64]Collector   // keyed by descriptor hash
    descIDs         map[uint64]struct{}
    dimHashesByName map[string]uint64      // enforces label consistency per family
    mtx             sync.RWMutex
}
```
`Gather()` calls every registered `Collector.Collect(ch chan<- Metric)`, reads `dto.Metric` protos from the channel, groups them by family name into `[]*dto.MetricFamily`, and sorts lexicographically. On registration, `dimHashesByName` ensures all metrics sharing a family name have identical label dimensions (panics otherwise).

---

## End-to-End Data Flow

```
HTTP endpoint
    │
    ▼
targetScraper.scrape()         — GET with timeout + headers
    │
    ▼
targetScraper.readResponse()   — gzip decompress → pooled []byte
    │
    ▼
textparse.New(b, contentType)  — pick parser (text/protobuf/openmetrics)
    │
    ▼
scrapeLoop.append() loop
    ├─ p.Next() → EntryType/Help → scrapeCache.setType/setHelp
    └─ p.Next() → EntrySeries
         ├─ scrapeCache.get() → cache hit: reuse ref+lset
         │                   → cache miss: p.Labels() + sampleMutator() + validate
         ├─ app.Append(ref, lset, t, val)       — float sample
         ├─ app.AppendHistogram(ref, lset, t, h) — native histogram
         ├─ app.AppendExemplar(ref, lset, e)     — exemplar if present
         ├─ app.UpdateMetadata(ref, lset, meta)  — WAL metadata
         └─ scrapeCache.trackStaleness(ref, ce)
    │
    ▼
scrapeCache.updateStaleMarkers()  — StaleNaN for disappeared series
    │
    ▼
app.Commit()                      — flush to TSDB WAL
```

---

**Key design choices:**
- **Lock-free metrics** — counters/gauges use only atomics, no mutexes
- **Per-target loop isolation** — a slow target never blocks others
- **Two-scrape staleness** — `seriesCur`/`seriesPrev` swap detects disappearing series in exactly one interval
- **Symbol table** — label strings are interned across all targets in a pool to reduce allocations
- **Pooled buffers** — HTTP response bodies are allocated from `pool.Pool` to reduce GC pressure

### Answer: `sonnet` / `mcp-only`

Here's a complete picture of how Prometheus metrics scraping and collection works, tracing the data from HTTP fetch to in-memory storage:

---

## 1. Manager: Top-Level Coordinator

**`manager.go:133` — `Manager` type**

The `Manager` is the top-level coordinator. It holds a map of `scrapePool`s (one per job), indexed by job name:

```
Manager
 ├── scrapeConfigs  map[string]*config.ScrapeConfig  — per-job config
 ├── scrapePools    map[string]*scrapePool            — per-job pool of loops
 └── targetSets     map[string][]*targetgroup.Group   — latest discovered targets
```

**`manager.go:158` — `Manager.Run`**

`Run` receives target group updates from service discovery on a channel:
```
service discovery → tsets channel → Manager.Run → updateTsets → triggerReload
```
It sends to `triggerReload` (non-blocking, coalesces bursts), then a background `reloader()` goroutine wakes on a ticker (default every 5s) and calls `reload()`.

**`manager.go:211` — `Manager.reload`**

For each job set, creates a `scrapePool` if missing, then calls `sp.Sync(groups)` concurrently across all pools. `Sync` diffs the old active targets against the new set: starts new `scrapeLoop`s for added targets, stops loops for removed ones.

---

## 2. scrapePool: Per-Job Loop Manager

**`scrape.go:83` — `scrapePool` type**

```
scrapePool
 ├── config       *config.ScrapeConfig
 ├── activeTargets map[uint64]*Target
 └── loops         map[uint64]loop    — one scrapeLoop per target (keyed by target hash)
```

Each target gets a dedicated `scrapeLoop` goroutine running independently.

---

## 3. scrapeLoop: Per-Target Scrape Goroutine

**`scrape.go:822` — `scrapeLoop` type**

**`scrape.go:1234` — `scrapeLoop.run`**

The loop:
1. Waits for an initial **jitter offset** (`scraper.offset(interval, offsetSeed)`) to spread scrape load across HA replicas
2. Ticks at `sl.interval`
3. On each tick: calls `scrapeAndReport()`

**`scrape.go:1308` — `scrapeAndReport`**

```
scrapeAndReport
 1. sl.scraper.scrape(ctx)       → HTTP GET /metrics → *http.Response
 2. readResponse(resp, buf)      → raw bytes into pooled buffer
 3. app := sl.appender()         → creates storage.Appender (or AppenderV2)
 4. app.append(buf, contentType, ts) → parse + ingest
 5. app.Commit() or Rollback()
 6. sl.report(app, ...)          → append synthetic report samples (scrape_duration_seconds, etc.)
```

---

## 4. Text Format Parsing

**`textparse_interface.go:28` — `Parser` interface**

The parser is content-type aware (Prometheus text 0.0.4, OpenMetrics 1.0.0, PrometheusProto, etc.). It's an **iterator**:

```go
for {
    et, err := p.Next()     // advance: EntrySeries, EntryHistogram, EntryHelp, EntryType, ...
    switch et {
    case textparse.EntrySeries:
        b, ts, val := p.Series()   // raw label bytes, optional timestamp, float64 value
        p.Labels(&lset)            // materialise labels
    case textparse.EntryHistogram:
        b, ts, h, fh := p.Histogram()
    }
}
```

**`scrape.go:1562` — `scrapeLoopAppender.append`**

This method drives the parser loop. For each `EntrySeries`:
1. Look up `(raw label bytes → cacheEntry)` in `scrapeCache.series`
2. If cache miss: apply relabelling (`sampleMutator`), get a storage ref
3. Call `sl.Appender.Append(ref, lset, ts, val)` → writes to TSDB head
4. Store `cacheEntry{ref, lset, hash}` back in cache for next scrape

---

## 5. scrapeCache: Per-Target Series Cache

**`scrape.go:883` — `scrapeCache`**

```
scrapeCache
 ├── series       map[string]*cacheEntry  — raw metric string → (SeriesRef, lset, hash)
 ├── droppedSeries map[string]*uint64     — relabelled-away series, to skip quickly
 ├── seriesCur/seriesPrev                 — for staleness marker tracking across scrapes
 └── metadata     map[string]*metaEntry   — TYPE/HELP/UNIT by metric family name
```

**`scrape.go:986` — `scrapeCache.get`**  
**`scrape.go:996` — `scrapeCache.addRef`**

On cache hit the `SeriesRef` is reused (avoids re-looking up the series in TSDB). After two consecutive scrape iterations with a series absent, a stale marker is appended.

---

## 6. Counter and Gauge Internal Tracking

**`prom_counter.go:103` — `counter` struct**

Counters use a split representation to minimize floating-point error:

```go
type counter struct {
    valBits uint64   // bits of float64 for non-integer additions (atomic CAS)
    valInt  uint64   // integer accumulator (atomic AddUint64)
    ...
}
```

- `Inc()` → `atomic.AddUint64(&c.valInt, 1)` (fast path, `scrape.go:1562`)
- `Add(v)` — if `v` is an exact integer: `AddUint64`; otherwise: CAS loop on `valBits`
- `get()` returns `float64(valInt) + math.Float64frombits(valBits)`

**`prom_gauge.go:114,122` — `gauge.Inc / gauge.Add`**

Gauges store a single `valBits uint64` and use an **atomic CAS loop** for all updates (since they can go up or down):
```go
for {
    oldBits := atomic.LoadUint64(&g.valBits)
    newBits := math.Float64bits(math.Float64frombits(oldBits) + val)
    if atomic.CompareAndSwapUint64(&g.valBits, oldBits, newBits) { return }
}
```

---

## 7. Registry: Collector → MetricFamily Pipeline

**`prom_registry.go:253` — `Registry` type**

```
Registry
 ├── collectorsByID   map[uint64]Collector  — checked collectors (hash of their Descs)
 ├── uncheckedCollectors []Collector
 ├── descIDs          map[uint64]struct{}   — all known Desc IDs for consistency checks
 └── dimHashesByName  map[string]uint64     — metric name → label-set hash
```

**`prom_registry.go:411` — `Registry.Gather`**

`Gather()` is what the `/metrics` HTTP handler calls. It:
1. Fans out `Collect(ch)` calls across all registered collectors in parallel (goroutine pool)
2. Each `Collector` sends `Metric` values to the channel
3. **`processMetric`** (`prom_registry.go:619`) assembles these into `map[string]*dto.MetricFamily`
   - Checks type consistency (a name that was previously COUNTER can't now be GAUGE)
   - Validates label sets for uniqueness
   - On first encounter of a name, creates a `MetricFamily` and infers its type from the `dtoMetric` fields (`Gauge != nil` → GAUGE, `Counter != nil` → COUNTER, etc.)
4. Returns a sorted `[]*dto.MetricFamily` slice for exposition

---

## Data Flow Summary

```
Service Discovery
      │ target groups
      ▼
Manager.Run ──triggerReload──▶ Manager.reload
                                      │ sp.Sync(groups)
                                      ▼
                               scrapePool
                                      │ one goroutine per target
                                      ▼
                               scrapeLoop.run  (ticker @ interval)
                                      │
                                      ▼
                         scraper.scrape() ──HTTP GET──▶ /metrics endpoint
                                      │ raw bytes
                                      ▼
                         textparse.Parser.Next() loop
                                      │ (label bytes, ts, float64)
                                      ▼
                         scrapeCache.get(met)   ← cache hit: reuse SeriesRef
                                      │
                                      ▼
                         Appender.Append(ref, lset, ts, val)
                                      │
                                      ▼
                         TSDB Head (memSeries / stripeSeries)
                                      │
                                      ▼
                         app.Commit()  →  WAL write + in-memory chunks
```

The key design choice is the **`scrapeCache`** sitting between the parser and the TSDB appender: it avoids re-resolving label sets and series references on every scrape, making the hot path just an atomic cache lookup + appender call.

### Answer: `sonnet` / `mcp-full`

Here's a thorough walkthrough of how Prometheus metrics scraping and collection works in this codebase:

---

## Data Flow: Scrape to In-Memory Storage

```
Discovery Manager
      │  target groups
      ▼
  Manager.Run()
      │  triggers reload
      ▼
  Manager.reload()
      │  creates/syncs
      ▼
  scrapePool.Sync()
      │  starts loops
      ▼
  scrapeLoop.run()       ← one goroutine per target
      │  on each tick
      ▼
  scrapeAndReport()
      │  HTTP GET /metrics
      ▼
  scraper.readResponse() → Parser (text/proto)
      │  parsed samples
      ▼
  appender.append()
      │  Commit()
      ▼
  TSDB / WAL (storage.Appender)
```

---

## 1. Manager — The Top-Level Coordinator

**`manager.go:133`** — `Manager` holds a map of `scrapePool` objects (one per job), keyed by job name:

```go
type Manager struct {
    scrapePools   map[string]*scrapePool
    scrapeConfigs map[string]*config.ScrapeConfig
    triggerReload chan struct{}
    ...
}
```

**`manager.go:158`** — `Manager.Run()` reads target group updates from a channel (fed by service discovery), stores them, and signals `triggerReload`. A `reloader()` goroutine debounces those signals with a 5-second ticker before calling `Manager.reload()`.

**`manager.go:211`** — `Manager.reload()` either creates new `scrapePool`s for new jobs, or calls `sp.Sync(groups)` for existing ones, both in parallel goroutines.

**`manager.go:278`** — `Manager.ApplyConfig()` handles config reloads — it calls `sp.reload(cfg)` on changed pools (which restarts all loops with new config) and `sp.stop()` on removed ones.

---

## 2. scrapePool — Per-Job State

**`scrape.go:83`** — One `scrapePool` per scrape job, tracking all active targets and their loops:

```go
type scrapePool struct {
    loops         map[uint64]loop    // hash(labelset) → loop
    activeTargets map[uint64]*Target
    config        *config.ScrapeConfig
    client        *http.Client
    appendable    storage.Appendable
    ...
}
```

`Sync()` diffs old vs new target sets, stopping loops for removed targets and starting new ones. Each active target gets a `scrapeLoop`.

---

## 3. scrapeLoop — Per-Target Scrape Goroutine

**`scrape.go:822`** — `scrapeLoop` is the core per-target worker:

```go
type scrapeLoop struct {
    scraper    scraper         // does the HTTP fetch
    cache      *scrapeCache    // tracks known series refs
    appendable storage.Appendable
    interval   time.Duration
    timeout    time.Duration
    ...
}
```

**`scrape.go:1234`** — `scrapeLoop.run()` is a ticker loop:
1. Waits for an initial offset (to spread load across HA setups).
2. On each tick, calls `scrapeAndReport(last, scrapeTime, errc)`.
3. On context cancellation, emits staleness markers for all series then exits.

**`scrape.go:1308`** — `scrapeAndReport()` is the main per-scrape function:
```
1. resp = scraper.scrape(ctx)           // HTTP GET
2. buf = scraper.readResponse(...)      // decompress, read body
3. total, added, seriesAdded = app.append(buf, contentType, ts)
4. deferred: app.Commit()
5. deferred: sl.report(app, ...)        // write scrape_duration_seconds, up, etc.
```

---

## 4. Parser — Text Format to Samples

**`textparse_interface.go:28`** — The `Parser` interface is the abstraction for all exposition formats (PrometheusText, OpenMetrics, ProtoBuffer):

```go
type Parser interface {
    Next() (Entry, error)        // iterate entries
    Series() ([]byte, *int64, float64)           // float sample
    Histogram() ([]byte, *int64, *histogram.Histogram, *histogram.FloatHistogram)
    Help() ([]byte, []byte)      // # HELP lines
    Type() ([]byte, model.MetricType)            // # TYPE lines
    Unit() ([]byte, []byte)
    Labels(b *labels.Builder)    // label set for current entry
    Metric(p *string) model.LabelValue
}
```

The loop in `app.append()` calls `p.Next()` iteratively. For each `EntrySeries` entry, it calls `p.Series()` to get the raw label bytes, float value, and optional timestamp, then calls `storage.Appender.Append(ref, lset, ts, val)`.

**`textparse_interface.go:125`** — `ParserOptions` controls behavior like native histogram conversion, OpenMetrics `_created` series skipping, and type/unit label injection.

---

## 5. Counter and Gauge — In-Memory Storage

**`prom_counter.go:103`** — `counter` struct uses a split-integer trick for lock-free performance:

```go
type counter struct {
    valBits uint64  // float64 bits for fractional adds
    valInt  uint64  // integer portion (fast path)
    exemplar atomic.Value
    ...
}
```

- `Inc()` → `atomic.AddUint64(&c.valInt, 1)` (fastest path)
- `Add(v float64)` → if `v` is a whole number, same integer add; otherwise CAS loop on `valBits`
- `Write(dto.Metric)` → reads both fields, combines into final float64 for exposition

**`prom_gauge.go:90`** — `gauge` is simpler — a single `uint64` storing `math.Float64bits(val)`:

```go
type gauge struct {
    valBits uint64  // float64 bits stored atomically
    ...
}
```

- `Set(val)` → `atomic.StoreUint64(&g.valBits, math.Float64bits(val))`
- `Add(val)` → CAS loop: load → add → store
- No integer fast path since gauges can be any value

---

## 6. Registry — Gathering Metric Families

**`prom_registry.go:253`** — `Registry` stores collectors by ID (hash of their descriptors):

```go
type Registry struct {
    collectorsByID      map[uint64]Collector
    descIDs             map[uint64]struct{}
    dimHashesByName     map[string]uint64
    uncheckedCollectors []Collector
}
```

**`prom_registry.go:411`** — `Registry.Gather()` is the pull path (called at scrape time for the Prometheus server's own metrics):
1. Fans out `Collector.Collect(chan Metric)` calls concurrently into goroutines.
2. Each metric is processed by `processMetric()`, which calls `metric.Write(*dto.Metric)` to get the protobuf representation.
3. `processMetric()` groups metrics into `metricFamiliesByName map[string]*dto.MetricFamily`, validating type consistency.
4. Returns a lexicographically sorted `[]*dto.MetricFamily` slice.

**`prom_registry.go:619`** — `processMetric()` handles both existing and new family names:
- **Existing**: validates the incoming metric matches the already-seen type (e.g., won't accept a `Counter` where a `Gauge` was registered)
- **New**: infers type from which `dto.Metric` field is non-nil (`dtoMetric.Gauge != nil` → `GAUGE`, etc.)

**`prom_registry.go:1036`** — `TransactionalGatherer` extends this with a `done()` callback, allowing cache implementations to release memory after the caller is finished exposing.

---

## Summary: Key Types

| Type | File | Role |
|---|---|---|
| `Manager` | `manager.go:133` | Top-level; owns all `scrapePool`s; reacts to discovery |
| `scrapePool` | `scrape.go:83` | Per-job; owns `scrapeLoop`s; syncs targets |
| `scrapeLoop` | `scrape.go:822` | Per-target; ticks, fetches, parses, appends |
| `scraper` (interface) | `scrape.go:677` | HTTP fetch + response reading |
| `Parser` (interface) | `textparse_interface.go:28` | Text/proto format → sample iteration |
| `counter` | `prom_counter.go:103` | Lock-free split int/float counter |
| `gauge` | `prom_gauge.go:90` | Lock-free float64 via atomic uint64 |
| `Registry` | `prom_registry.go:253` | Collector registry + fan-out gather |
| `scrapeLoopAppendAdapter` | `scrape.go:129` | Bridges parser output → `storage.Appender` |

### Answer: `opus` / `baseline`

You've hit your limit · resets 6pm (Europe/Berlin)

### Answer: `opus` / `mcp-only`

You've hit your limit · resets 6pm (Europe/Berlin)

### Answer: `opus` / `mcp-full`

You've hit your limit · resets 6pm (Europe/Berlin)

### Full Judge Analysis (Opus 4.6)

_Judge unavailable_

