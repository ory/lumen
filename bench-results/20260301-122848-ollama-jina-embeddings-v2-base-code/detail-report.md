# Benchmark Detail Report

Generated: 2026-03-01 11:49 UTC | Results: `20260301-122848`

---

## label-matcher [easy]

**Question:** What label matcher types are available and how is a Matcher
created? Show the type definitions and constructor.

### Metrics

| Model      | Scenario | Duration | Input Tok | Cache Read | Cache Created | Output Tok | Cost (USD) |
| ---------- | -------- | -------- | --------- | ---------- | ------------- | ---------- | ---------- |
| **sonnet** | baseline | 26.6s    | 29344     | 28104      | 0             | 667        | $0.26694   |
| **sonnet** | mcp-only | 12.3s    | 18929     | 0          | 0             | 715        | $0.11252   |
| **sonnet** | mcp-full | 9.4s     | 29935     | 28104      | 0             | 574        | $0.17808   |
| **opus**   | baseline | 17.2s    | 46482     | 42345      | 0             | 537        | $0.26701   |
| **opus**   | mcp-only | 11.7s    | 18953     | 0          | 0             | 504        | $0.10736   |
| **opus**   | mcp-full | 12.7s    | 31142     | 28230      | 0             | 559        | $0.18380   |

### Answer: `sonnet` / `baseline`

Here's the breakdown from `matcher.go`:

---

**Match types** — a typed `int` enum with 4 values:

```go
// matcher.go:21-30
type MatchType int

const (
    MatchEqual     MatchType = iota  // =
    MatchNotEqual                    // !=
    MatchRegexp                      // =~
    MatchNotRegexp                   // !~
)
```

**`Matcher` struct** (`matcher.go:47-53`):

```go
type Matcher struct {
    Type  MatchType
    Name  string
    Value string

    re *FastRegexMatcher  // non-nil only for regexp types
}
```

**Two constructors:**

`NewMatcher` (`matcher.go:55-70`) — normal constructor, returns
`(*Matcher, error)`. For regexp types it compiles the pattern into a
`FastRegexMatcher`; for equality types `re` stays nil.

```go
func NewMatcher(t MatchType, n, v string) (*Matcher, error)
```

`MustNewMatcher` (`matcher.go:72-79`) — test-only variant that panics instead of
returning an error:

```go
func MustNewMatcher(mt MatchType, name, val string) *Matcher
```

The key design point: regex compilation happens eagerly at construction time, so
`Matches(s string)` never errors at call time.

### Answer: `sonnet` / `mcp-only`

There are two files with identical definitions (`labels_matcher.go` and
`matcher.go`). Here's what's available:

---

## `MatchType` — enum for label matching

**File:** `labels_matcher.go:21-29`

```go
// MatchType is an enum for label matching types.
type MatchType int

const (
    MatchEqual     MatchType = iota  // =
    MatchNotEqual                    // !=
    MatchRegexp                      // =~
    MatchNotRegexp                   // !~
)
```

Four types, matching Prometheus label selector syntax.

---

## `Matcher` — the matcher struct

**File:** `labels_matcher.go:46-53`

```go
// Matcher models the matching of a label.
type Matcher struct {
    Type  MatchType
    Name  string
    Value string

    re *FastRegexMatcher  // only set for regexp types
}
```

---

## Constructor: `NewMatcher`

**File:** `labels_matcher.go:55-70`

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

For `MatchRegexp` / `MatchNotRegexp`, the value string is compiled into a
`FastRegexMatcher` at construction time. Returns an error if the regex is
invalid.

There's also a panic-on-error variant for tests:

**File:** `labels_matcher.go:72-79`

```go
// MustNewMatcher panics on error - only for use in tests!
func MustNewMatcher(mt MatchType, name, val string) *Matcher {
    m, err := NewMatcher(mt, name, val)
    if err != nil {
        panic(err)
    }
    return m
}
```

---

**Note:** The definitions appear in two files — `labels_matcher.go` and
`matcher.go` — with identical content, suggesting one may be a copy or the
fixture contains duplicate files.

### Answer: `sonnet` / `mcp-full`

There are four label matcher types, defined as an `iota` enum in
`labels_matcher.go`:

**`MatchType` enum** (`labels_matcher.go:21-29`)

```go
type MatchType int

const (
    MatchEqual     MatchType = iota  // =
    MatchNotEqual                    // !=
    MatchRegexp                      // =~
    MatchNotRegexp                   // !~
)
```

**`Matcher` struct** (`labels_matcher.go:46-53`)

```go
type Matcher struct {
    Type  MatchType
    Name  string
    Value string

    re *FastRegexMatcher
}
```

**Constructor** (`labels_matcher.go:55-70`)

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

Key points:

- `NewMatcher(t, name, value)` is the standard constructor; returns an error if
  the value is an invalid regex for `MatchRegexp`/`MatchNotRegexp`.
- For the two regex types (`=~`, `!~`), the value is compiled into a
  `FastRegexMatcher` and cached in the unexported `re` field.
- `MustNewMatcher` is a panic-on-error convenience wrapper, explicitly
  documented as test-only.

### Answer: `opus` / `baseline`

Here's what's available:

### `MatchType` enum (line 22)

```go
type MatchType int

const (
    MatchEqual     MatchType = iota  // "="
    MatchNotEqual                     // "!="
    MatchRegexp                       // "=~"
    MatchNotRegexp                    // "!~"
)
```

Four match types: exact equality, negated equality, regex match, and negated
regex match.

### `Matcher` struct (line 47)

```go
type Matcher struct {
    Type  MatchType
    Name  string
    Value string
    re    *FastRegexMatcher  // unexported, set only for regex types
}
```

### Constructor — `NewMatcher` (line 56)

```go
func NewMatcher(t MatchType, n, v string) (*Matcher, error)
```

Creates a `Matcher` with the given type, label name, and value. If the type is
`MatchRegexp` or `MatchNotRegexp`, it compiles a `FastRegexMatcher` from the
value string (returning an error if the regex is invalid). For equality types,
no regex is compiled.

There's also `MustNewMatcher` (line 73) which wraps `NewMatcher` and panics on
error — intended for tests only.

### Answer: `opus` / `mcp-only`

Here are the label matcher types and constructor from `labels_matcher.go`:

**`MatchType` enum** (`labels_matcher.go:21-22`):

```go
type MatchType int
```

With four constants (`labels_matcher.go:26-29`):

```go
MatchEqual    MatchType = iota  // "="
MatchNotEqual                    // "!="
MatchRegexp                      // "=~"
MatchNotRegexp                   // "!~"
```

**`Matcher` struct** (`labels_matcher.go:46-53`):

```go
type Matcher struct {
    Type  MatchType
    Name  string
    Value string
    re    *FastRegexMatcher  // unexported, set for regex types
}
```

**Constructor** (`labels_matcher.go:55-70`):

```go
func NewMatcher(t MatchType, n, v string) (*Matcher, error)
```

It builds a `Matcher` with the given type, name, and value. If the type is
`MatchRegexp` or `MatchNotRegexp`, it compiles a `FastRegexMatcher` from the
value string, returning an error if the regex is invalid.

There's also a test helper (`labels_matcher.go:72-79`):

```go
func MustNewMatcher(mt MatchType, name, val string) *Matcher
```

which panics on error instead of returning it.

### Answer: `opus` / `mcp-full`

Here are the label matcher types and constructor from `labels_matcher.go`:

**`MatchType` enum** (`labels_matcher.go:21-22`):

```go
type MatchType int

const (
    MatchEqual    MatchType = iota  // "="
    MatchNotEqual                   // "!="
    MatchRegexp                     // "=~"
    MatchNotRegexp                  // "!~"
)
```

**`Matcher` struct** (`labels_matcher.go:46-53`):

```go
type Matcher struct {
    Type  MatchType
    Name  string
    Value string
    re    *FastRegexMatcher  // only set for regex match types
}
```

**Constructor** (`labels_matcher.go:55-70`):

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

There are **4 match types**: exact equality (`=`), not-equal (`!=`), regex
(`=~`), and negated regex (`!~`). The constructor compiles a `FastRegexMatcher`
only for the two regex variants. There's also a `MustNewMatcher` helper
(`labels_matcher.go:72-79`) that panics on error, intended for tests.

### Full Judge Analysis (Opus 4.6)

Now I have the ground truth. Both files are identical, in
`testdata/fixtures/go/`. Here's my evaluation:

---

## Content Quality

**Ranking: 1st (tie) — sonnet/mcp-only, sonnet/mcp-full, opus/mcp-full** **4th
(tie) — sonnet/baseline, opus/baseline, opus/mcp-only**

All six answers are substantively correct — they identify the four `MatchType`
constants, the `Matcher` struct, `NewMatcher`, and `MustNewMatcher` accurately.
The differences are marginal. Let me break it down:

**sonnet/mcp-full** — Correct and complete. References `labels_matcher.go`
(valid file). Line numbers are accurate (21-29 for the type+consts, 46-53 for
struct, 55-70 for constructor, 72-79 for MustNewMatcher). Shows full constructor
source code. Concise, well-organized. Correctly notes `MustNewMatcher` is
test-only. Minor nit: says "21-29" for the const block which actually ends at
line 30, but the type declaration starts at 21 and the last const name is on 29,
so this is defensible.

**sonnet/mcp-only** — Also correct and complete, with accurate line numbers
referencing `labels_matcher.go`. Shows full source for all three definitions.
However, it adds a confusing note about "two files with identical definitions" —
while factually true (both `matcher.go` and `labels_matcher.go` exist with the
same content), the speculation that "one may be a copy or the fixture contains
duplicate files" adds noise without resolution. This is a minor ding.

**opus/mcp-full** — Correct, complete, accurate line references to
`labels_matcher.go`. Shows full constructor source. Clean and concise. Very
similar quality to sonnet/mcp-full.

**sonnet/baseline** — Correct content, but references only `matcher.go` (still a
valid file). Line numbers are accurate. Says "21-30" for the const block (line
30 is `)`), which is fine. Shows signatures rather than full code for the
constructors — slightly less complete than the MCP variants that showed full
source, but the question did ask to "show" the definitions. Good insight about
eager regex compilation.

**opus/baseline** — Correct, references `matcher.go` only. Line numbers accurate
(22, 47, 56, 73). More summary-style — doesn't show the full constructor body.
Solid but less detailed than MCP variants.

**opus/mcp-only** — Correct. References `labels_matcher.go` with accurate line
numbers. However, it oddly splits the `MatchType` line reference (21-22) from
the constants (26-29), which is a slightly awkward presentation. Shows
signatures but not full bodies. Adequate but slightly less polished.

**Overall**: All answers are correct. The key differentiator is whether they
showed the actual source code (as asked). The MCP-full and MCP-only sonnet
variants showed the most code. No answer contains factual errors.

---

## Efficiency Analysis

| Run             | Duration | Input Tok | Output Tok | Cost   |
| --------------- | -------- | --------- | ---------- | ------ |
| sonnet/mcp-only | 12.3s    | 18,929    | 715        | $0.113 |
| sonnet/mcp-full | 9.4s     | 29,935    | 574        | $0.178 |
| opus/mcp-only   | 11.7s    | 18,953    | 504        | $0.107 |
| opus/mcp-full   | 12.7s    | 31,142    | 559        | $0.184 |
| opus/baseline   | 17.2s    | 46,482    | 537        | $0.267 |
| sonnet/baseline | 26.6s    | 29,344    | 667        | $0.267 |

**Most efficient**: **opus/mcp-only** at $0.107 — lowest cost, fast (11.7s), and
the answer is correct and complete. **sonnet/mcp-only** is essentially tied at
$0.113.

**Fastest**: **sonnet/mcp-full** at 9.4s, benefiting from 28K cached tokens.
This is the fastest wall-clock time of all runs.

**Least efficient**: Both baselines tied at $0.267 — 2.5× the cost of the
MCP-only variants. The sonnet/baseline was also the slowest at 26.6s. The
baselines consumed far more input tokens (29-46K) because they had to read full
files rather than getting targeted search results.

**Surprising findings**:

- The MCP-only runs used zero cache reads, yet were still the cheapest. The
  semantic search tool is clearly saving tokens by returning only relevant
  chunks instead of full files.
- sonnet/mcp-full was the fastest despite not being the cheapest — the cache hit
  (28K tokens) likely explains the speed advantage.
- opus/baseline consumed 46K input tokens — nearly 2.5× the MCP-only variants —
  suggesting it read multiple full files to find the answer.

**Best quality-to-cost tradeoff**: **sonnet/mcp-full** — it produced a top-tier
answer (full source code shown, accurate line numbers, concise explanation) at
$0.178 in only 9.4s. If minimizing cost is paramount, **opus/mcp-only** at
$0.107 delivers a correct answer at 40% less cost, though with slightly less
detail in the presentation.

---

## histogram [medium]

**Question:** How does histogram bucket counting work? Show me the relevant
function signatures.

### Metrics

| Model      | Scenario | Duration | Input Tok | Cache Read | Cache Created | Output Tok | Cost (USD) |
| ---------- | -------- | -------- | --------- | ---------- | ------------- | ---------- | ---------- |
| **sonnet** | baseline | 119.6s   | 31571     | 28104      | 0             | 925        | $2.62379   |
| **sonnet** | mcp-only | 19.1s    | 23359     | 0          | 0             | 929        | $0.14002   |
| **sonnet** | mcp-full | 16.3s    | 35499     | 28104      | 0             | 742        | $0.21010   |
| **opus**   | baseline | 53.1s    | 181341    | 98805      | 0             | 2112       | $1.00891   |
| **opus**   | mcp-only | 22.2s    | 20842     | 0          | 0             | 957        | $0.12813   |
| **opus**   | mcp-full | 22.6s    | 33036     | 28230      | 0             | 840        | $0.20030   |

### Answer: `sonnet` / `baseline`

Here's how histogram bucket counting works in this codebase — there are two
parallel systems:

---

## Classic (Conventional) Histogram

**1. Find the bucket index** — `prom_histogram.go:864`

```go
func (h *histogram) findBucket(v float64) int
```

Linear search for `n < 35` upper bounds, binary search (`sort.SearchFloat64s`)
otherwise. Returns an index into `h.upperBounds[]`.

**2. Record the observation** — `prom_histogram.go:766`

```go
func (h *histogram) Observe(v float64)
func (h *histogram) observe(v float64, bucket int)
```

Uses a hot/cold lock-free scheme: the MSB of `countAndHotIdx` selects which
`histogramCounts` is "hot".

**3. Atomic bucket increment** — `prom_histogram.go:655`

```go
func (hc *histogramCounts) observe(v float64, bucket int, doSparse bool)
```

Classic path: `atomic.AddUint64(&hc.buckets[bucket], 1)` — flat `[]uint64`, one
slot per upper bound.

**4. Cumulative counts at scrape time** — `prom_histogram.go:809` Running sum
over `coldCounts.buckets[i]` — stored non-cumulatively, accumulated into
`dto.Bucket.CumulativeCount` only on write.

---

## Native (Sparse/Exponential) Histogram

Inside the same `histogramCounts.observe`, after the classic increment:

- Bucket key computed via `math.Frexp` → binary search into
  `nativeHistogramBounds[schema]` (schema 1–8) or bit-shift formula (schema ≤ 0)
- Stored in a `sync.Map` per sign (positive/negative) via:

```go
func addToBucket(buckets *sync.Map, key int, increment int64) bool  // prom_histogram.go:1534
```

Lock-free with `LoadOrStore` fallback for concurrent bucket creation.

---

## TSDB Data Model (histogram.go)

Buckets are **delta-encoded `int64` slices**, decoded during iteration:

```go
func (r *regularBucketIterator) Next() bool           // decodes: currCount += buckets[bucketsIdx]
func (c *cumulativeBucketIterator) Next() bool         // accumulates across spans
func (h *Histogram) CumulativeBucketIterator() BucketIterator[uint64]
```

Validation ensures
`sum(positiveBuckets) + sum(negativeBuckets) + zeroBucket == count` via:

```go
func validateCount(sum float64, count uint64, negativeBuckets, positiveBuckets map[int]int64, zeroBucket uint64) error  // prom_histogram.go:1871
```

### Answer: `sonnet` / `mcp-only`

Here's how histogram bucket counting works in this codebase:

## Regular (Classic) Bucket Counting

**Finding the bucket** — `prom_histogram.go:864`

```go
func (h *histogram) findBucket(v float64) int
```

Uses a linear search for arrays < 35 elements, binary search
(`sort.SearchFloat64s`) for larger arrays. Returns the index into
`h.upperBounds`, or `len(h.upperBounds)` for the implicit `+Inf` bucket.

**Recording an observation** — `prom_histogram.go:899`

```go
func (h *histogram) observe(v float64, bucket int)
```

Atomically increments `h.countAndHotIdx` to get the hot counts slot
(double-buffered with a hot/cold index), then delegates to
`histogramCounts.observe`.

**The actual count update** — `prom_histogram.go:652`

```go
func (hc *histogramCounts) observe(v float64, bucket int, doSparse bool)
```

- `atomic.AddUint64(&hc.buckets[bucket], 1)` — increments the raw
  (non-cumulative) bucket counter
- `atomicAddFloat(&hc.sumBits, v)` — updates the running sum
- Atomically increments `hc.count` last (acts as a completion signal)

**Serializing to cumulative counts** — `prom_histogram.go:779`

```go
func (h *histogram) Write(out *dto.Metric) error
```

Iterates `h.upperBounds`, accumulating a `cumCount` by summing raw bucket
values. Prometheus exposes buckets as cumulative (`le` label), so this
conversion happens only at read time.

## Native/Sparse Bucket Counting

For native histograms, `histogramCounts.observe` additionally computes a bucket
key using the schema:

- **Schema > 0**:
  `key = sort.SearchFloat64s(bounds, frac) + (exp-1)*len(bounds)`
  (sub-power-of-two divisions)
- **Schema ≤ 0**: `key = (exp + offset) >> -schema` (power-of-two grouping)

The key maps into a `sync.Map` (`nativeHistogramBucketsPositive` /
`nativeHistogramBucketsNegative`) via `addToBucket`. Values near zero fall into
the zero bucket: `atomic.AddUint64(&hc.nativeHistogramZeroBucket, 1)`.

## Key Design Points

- **Double-buffering**: `h.counts[0]` and `h.counts[1]` are swapped via the MSB
  of `h.countAndHotIdx`. `Write` flips the hot index and waits for the cold
  counts to settle (`waitForCooldown`) before reading.
- **Non-cumulative storage, cumulative output**: Raw per-bucket counts are
  stored; cumulation happens in `Write`.
- **`+Inf` bucket is implicit**: Removed from `upperBounds` at construction
  (`prom_histogram.go:588`); the total `count` serves as its value.

### Answer: `sonnet` / `mcp-full`

Here's how histogram bucket counting works in this codebase:

---

## Bucket Lookup: `findBucket`

`prom_histogram.go:864`

```go
func (h *histogram) findBucket(v float64) int
```

- Takes a float64 observation value, returns the index into `h.upperBounds`
- Uses **linear search** for small arrays (< 35 bounds) — benchmarked to be
  faster
- Falls back to **`sort.SearchFloat64s` (binary search)** for larger arrays
- Returns `len(h.upperBounds)` for the implicit `+Inf` bucket

---

## Observation Recording: `histogramCounts.observe`

`prom_histogram.go:652`

```go
func (hc *histogramCounts) observe(v float64, bucket int, doSparse bool)
```

- Increments the regular bucket counter:
  `atomic.AddUint64(&hc.buckets[bucket], 1)`
- If `doSparse=true`, also computes the **native/sparse bucket index** using:
  - For schema > 0:
    `sort.SearchFloat64s(nativeHistogramBounds[schema], frac) + (exp-1)*len(bounds)`
  - For schema ≤ 0: bit-shift arithmetic on the float64 exponent
- Routes the value to `nativeHistogramBucketsPositive`,
  `nativeHistogramBucketsNegative`, or the zero bucket via `atomic.AddUint64`
- Increments total count **last** as a completion signal

---

## Hot/Cold Index Dispatch: `histogram.observe`

`prom_histogram.go:899`

```go
func (h *histogram) observe(v float64, bucket int)
```

- Uses a double-buffer scheme: `countAndHotIdx` packs a hot-index bit (MSB) +
  count (lower 63 bits) into one atomic
- Directs writes to the "hot" `histogramCounts` to avoid write contention during
  `Write()`

---

## Cumulative Bucketing on Read: `histogram.Write`

`prom_histogram.go:779`

```go
func (h *histogram) Write(out *dto.Metric) error
```

- Accumulates `cumCount` across `coldCounts.buckets[i]` in order, producing the
  standard Prometheus **cumulative bucket** output (each bucket count includes
  all observations ≤ its upper bound)

### Answer: `opus` / `baseline`

Here's how histogram bucket counting works in this codebase. There are two
layers: **classic (fixed-boundary) buckets** and **native (sparse/exponential)
buckets**.

## Classic Bucket Counting

Classic histograms use pre-defined upper bounds. The relevant bucket creation
functions are in `prom_histogram.go`:

```go
func LinearBuckets(start, width float64, count int) []float64        // line 295
func ExponentialBuckets(start, factor float64, count int) []float64   // line 315
func ExponentialBucketsRange(minBucket, maxBucket float64, count int) []float64  // line 339
```

- `LinearBuckets` — creates `count` buckets where each boundary = previous +
  `width`
- `ExponentialBuckets` — creates `count` buckets where each boundary = previous
  × `factor`
- `ExponentialBucketsRange` — creates `count` buckets exponentially spaced
  between `min` and `max`

## Native (Sparse) Bucket Observation

The core counting logic is in `histogramCounts.observe`
(`prom_histogram.go:655`):

```go
func (hc *histogramCounts) observe(v float64, bucket int, doSparse bool)
```

This does two things:

1. **Classic**: atomically increments `hc.buckets[bucket]` (pre-determined
   index)
2. **Native/sparse**: computes the bucket key using `math.Frexp(math.Abs(v))` to
   decompose the value into fraction and exponent, then:
   - For **positive schemas** (`schema > 0`): binary searches
     `nativeHistogramBounds[schema]` for sub-exponential bucket placement:
     `key = SearchFloat64s(bounds, frac) + (exp-1)*len(bounds)`
   - For **non-positive schemas** (`schema <= 0`): uses
     `key = (exp + offset) >> -schema` to merge exponential buckets
   - Values within `zeroThreshold` go to the **zero bucket**
     (`atomic.AddUint64(&hc.nativeHistogramZeroBucket, 1)`)
   - Positive/negative values go to separate `sync.Map` stores via
     `addToBucket()`

## Bucket Limiting

When too many native buckets are created, `limitBuckets`
(`prom_histogram.go:919`) kicks in:

```go
func (h *histogram) limitBuckets(counts *histogramCounts, value float64, bucket int)
func (h *histogram) maybeWidenZeroBucket(hot, cold *histogramCounts) bool  // line 1017
func (h *histogram) doubleBucketWidth(hot, cold *histogramCounts)          // line 1087
```

These reduce resolution by either widening the zero bucket to absorb nearby
buckets, or halving the schema (doubling bucket width).

## Validation

Bucket counts are validated via `validateCount` (`prom_histogram.go:1871`):

```go
func validateCount(sum float64, count uint64, negativeBuckets, positiveBuckets map[int]int64, zeroBucket uint64) error
```

And in the data model layer, `Histogram.Validate` (`histogram.go:426`) checks
that `nCount + pCount + ZeroCount == Count` (or `>=` if Sum is NaN).

## Iteration

Bucket iterators traverse the sparse representation (`histogram.go`):

```go
func (h *Histogram) PositiveBucketIterator() BucketIterator[uint64]    // line 216
func (h *Histogram) NegativeBucketIterator() BucketIterator[uint64]    // line 223
func (h *Histogram) CumulativeBucketIterator() BucketIterator[uint64]  // line 232
```

The `regularBucketIterator.Next()` (line 496) walks spans and decodes
delta-encoded bucket counts (`r.currCount += r.buckets[r.bucketsIdx]`), while
`cumulativeBucketIterator.Next()` (line 550) accumulates counts across buckets
for a cumulative view.

### Answer: `opus` / `mcp-only`

Here's how histogram bucket counting works in this codebase:

## Two Bucket Systems

The code supports **two distinct bucket systems**: classic (fixed) buckets and
native (sparse/exponential) buckets.

### Classic Buckets

Defined by fixed upper bounds (e.g.
`DefBuckets = {.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}`). Counting is
straightforward — find which bucket the value falls into and atomically
increment:

```go
// prom_histogram.go:652-706
func (hc *histogramCounts) observe(v float64, bucket int, doSparse bool)
```

The `bucket` parameter is the pre-computed index into `hc.buckets []uint64`. The
count is simply:

```go
atomic.AddUint64(&hc.buckets[bucket], 1)
```

When writing output, counts are made **cumulative** in `histogram.Write`
(`prom_histogram.go:779`):

```go
var cumCount uint64
for i, upperBound := range h.upperBounds {
    cumCount += atomic.LoadUint64(&coldCounts.buckets[i])
    // ...
}
```

### Native (Sparse/Exponential) Buckets

These use a **schema-based logarithmic** bucketing scheme. The key logic is in
`histogramCounts.observe` (`prom_histogram.go:652-706`):

1. **Decompose** the value via `math.Frexp(math.Abs(v))` → fraction + exponent
2. **Compute bucket key** based on schema:
   - **Schema > 0**: binary search in `nativeHistogramBounds[schema]`, then
     `key = SearchResult + (exp-1)*len(bounds)`
   - **Schema ≤ 0**: `key = exp` (adjusted for exact powers of 2), then
     bit-shifted: `key = (key + offset) >> -schema`
3. **Route** to the appropriate map:
   - `v > zeroThreshold` →
     `addToBucket(&hc.nativeHistogramBucketsPositive, key, 1)`
   - `v < -zeroThreshold` →
     `addToBucket(&hc.nativeHistogramBucketsNegative, key, 1)`
   - Otherwise → `atomic.AddUint64(&hc.nativeHistogramZeroBucket, 1)`

### Key Types for Iteration

For the sparse `Histogram` struct (`histogram.go:35-80`), bucket iteration uses:

```go
// histogram.go:481
type regularBucketIterator struct { baseBucketIterator[uint64, int64] }

// histogram.go:530-548
type cumulativeBucketIterator struct { ... }
```

The `cumulativeBucketIterator` walks spans and reconstructs cumulative counts,
handling gaps between spans via `emptyBucketCount`.

### Hot/Cold Swap for Thread Safety

`histogram.Write` (`prom_histogram.go:779`) uses a **hot/cold index swap** —
observations go to the "hot" `histogramCounts`, while `Write` reads from the
"cold" one after swapping via `atomic.AddUint64(&h.countAndHotIdx, 1<<63)`.

### Answer: `opus` / `mcp-full`

Here's how histogram bucket counting works in this codebase:

**There are two bucket systems: classic (regular) and native
(sparse/exponential).**

### Classic Buckets

Classic histograms use pre-defined upper bounds (e.g.
`DefBuckets = {.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}`). On each
observation, the bucket index is found and the count is atomically incremented:

```go
// prom_histogram.go:652
func (hc *histogramCounts) observe(v float64, bucket int, doSparse bool)
```

The key line is `atomic.AddUint64(&hc.buckets[bucket], 1)` — a simple atomic
increment of the bucket matching the observed value.

When writing out metrics (`histogram.Write` at `prom_histogram.go:779`), bucket
counts are converted to **cumulative** counts:

```go
var cumCount uint64
for i, upperBound := range h.upperBounds {
    cumCount += atomic.LoadUint64(&coldCounts.buckets[i])
    // each bucket reports cumCount (sum of all buckets <= upperBound)
}
```

### Native (Exponential) Buckets

Native histograms use a schema-based exponential bucketing system. The bucket
index is computed from the observation value using `math.Frexp`:

```go
frac, exp := math.Frexp(math.Abs(v))
if schema > 0 {
    key = sort.SearchFloat64s(bounds, frac) + (exp-1)*len(bounds)
} else {
    key = exp
    if frac == 0.5 { key-- }
    offset := (1 << -schema) - 1
    key = (key + offset) >> -schema
}
```

Observations are stored in three places based on value:

- `v > zeroThreshold` → `nativeHistogramBucketsPositive` (sync.Map)
- `v < -zeroThreshold` → `nativeHistogramBucketsNegative` (sync.Map)
- otherwise → `nativeHistogramZeroBucket` (atomic uint64)

### Key Types for Iteration

- **`regularBucketIterator`** (`histogram.go:481`) — iterates classic buckets
- **`cumulativeBucketIterator`** (`histogram.go:530`) — iterates native
  histogram spans, decoding delta-encoded bucket counts into cumulative counts,
  handling empty buckets between spans

The `Histogram` struct (`histogram.go:35`) stores native bucket counts as
**delta-encoded** `int64` slices (`PositiveBuckets`, `NegativeBuckets`), with
`Span` structs defining which bucket indices are populated.

### Full Judge Analysis (Opus 4.6)

## Content Quality

### Ranking: Best to Worst

**1. opus / baseline**

The most comprehensive and well-structured answer. It covers all four key
aspects: bucket creation functions (`LinearBuckets`, `ExponentialBuckets`,
`ExponentialBucketsRange`), the core observation logic in
`histogramCounts.observe`, native bucket key computation, bucket
limiting/resolution reduction (`limitBuckets`, `maybeWidenZeroBucket`,
`doubleBucketWidth`), validation (`validateCount`), and iteration. Line
references are precise and the function signatures are accurate. It's the only
answer that covers bucket limiting — a critical part of "how bucket counting
works" that others miss. The 53s runtime and $1.00 cost reflect thorough file
reading, which paid off in coverage.

**2. sonnet / baseline**

Also very thorough, covering classic observation, native/sparse buckets, the
TSDB data model layer (`histogram.go` iterators), and validation. It uniquely
includes the `addToBucket` sync.Map helper and the delta-encoding iteration
model. Line references are mostly accurate. It covers both the runtime counting
layer (`prom_histogram.go`) and the storage/iteration layer (`histogram.go`),
giving a fuller picture. However, it's the most expensive run at $2.62 — the
119.6s runtime and 31K input tokens suggest inefficient exploration.

**3. opus / mcp-full**

Clean, accurate, and well-organized. Covers classic buckets, native exponential
bucketing with the `math.Frexp` decomposition, cumulative count conversion, and
iteration types. The inline code snippets for the native key computation are
precise and readable. It correctly identifies delta-encoding in the `Histogram`
struct. Missing bucket limiting and validation, but what it covers is correct
and concise. Excellent value at $0.20.

**4. opus / mcp-only**

Very similar quality to opus/mcp-full with slightly more detail on the hot/cold
swap mechanism and `cumulativeBucketIterator` behavior. Covers the three-way
routing (positive/negative/zero) clearly. Minor edge: it explains
`emptyBucketCount` in the cumulative iterator, which others skip. At $0.13 it's
the cheapest opus run. Slightly less polished organization than mcp-full.

**5. sonnet / mcp-full**

Accurate and focused. Covers `findBucket`, `histogramCounts.observe`, hot/cold
dispatch, and cumulative bucketing on read. Good structure with clear section
headers. However, it's the shortest answer and omits iteration, validation, and
bucket limiting entirely. The native bucket key computation coverage is thinner
than the opus answers. Adequate but not as deep.

**6. sonnet / mcp-only**

Solid coverage of classic bucket mechanics — the best explanation of the
double-buffer `countAndHotIdx` scheme and the "count incremented last as
completion signal" detail. Good coverage of native bucket key computation.
Includes the important note about `+Inf` bucket being implicit. However, it
omits iteration entirely and has no coverage of the TSDB data model layer. The
"Key Design Points" section adds useful context but doesn't compensate for
missing topics.

---

## Efficiency Analysis

| Scenario          | Duration | Cost  | Quality Rank |
| ----------------- | -------- | ----- | ------------ |
| sonnet / baseline | 119.6s   | $2.62 | 2nd          |
| sonnet / mcp-only | 19.1s    | $0.14 | 6th          |
| sonnet / mcp-full | 16.3s    | $0.21 | 5th          |
| opus / baseline   | 53.1s    | $1.01 | 1st          |
| opus / mcp-only   | 22.2s    | $0.13 | 4th          |
| opus / mcp-full   | 22.6s    | $0.20 | 3rd          |

**Key observations:**

- **Sonnet baseline is an outlier in cost.** At $2.62 it's 12-19x more expensive
  than the MCP runs, largely driven by 31K input tokens and 28K cache reads —
  suggesting it read many large files to find the relevant code. The 119.6s
  runtime confirms extensive file exploration.
- **Opus baseline is far more efficient than sonnet baseline.** Despite reading
  even more tokens (181K input), the cost is only $1.01 thanks to heavy cache
  utilization (98K cache reads). It also finished in half the time (53s vs
  120s). This suggests opus navigated the codebase more efficiently.
- **MCP runs are dramatically cheaper and faster across both models.** The
  semantic search tool let both models find relevant code in ~16-22s at
  $0.13-0.21 — a 5-13x cost reduction vs baseline.
- **Quality gap is smaller than cost gap.** The MCP answers are 80-90% as good
  as baseline answers at 5-10% of the cost. The main loss is coverage of
  secondary topics (bucket limiting, validation).
- **Opus consistently outperforms sonnet at similar cost points.** opus/mcp-only
  ($0.13) produces better answers than sonnet/mcp-only ($0.14). opus/mcp-full
  ($0.20) beats sonnet/mcp-full ($0.21).

**Recommendation:** **opus / mcp-full** offers the best quality-to-cost tradeoff
— ranked 3rd in quality at only $0.20, with a 22.6s runtime. It captures all the
essential mechanics (classic counting, native exponential bucketing, cumulative
conversion, delta-encoded iteration) in a clear, accurate format. If budget
permits, opus/baseline at $1.01 gives the most complete answer but at 5x the
cost for incremental gains in coverage (bucket limiting, validation).

---

## tsdb-compaction [hard]

**Question:** How does TSDB compaction work end-to-end? Explain the Compactor
interface, LeveledCompactor, and how the DB triggers compaction. Show relevant
types, interfaces, and key method signatures.

### Metrics

| Model      | Scenario | Duration | Input Tok | Cache Read | Cache Created | Output Tok | Cost (USD) |
| ---------- | -------- | -------- | --------- | ---------- | ------------- | ---------- | ---------- |
| **sonnet** | baseline | 96.3s    | 30539     | 28104      | 0             | 1877       | $1.13178   |
| **sonnet** | mcp-only | 45.9s    | 49394     | 0          | 0             | 2299       | $0.30444   |
| **sonnet** | mcp-full | 53.1s    | 87711     | 56208      | 0             | 2532       | $0.52996   |
| **opus**   | baseline | 81.4s    | 285260    | 155265     | 0             | 3211       | $1.58421   |
| **opus**   | mcp-only | 49.5s    | 42393     | 0          | 0             | 2241       | $0.26799   |
| **opus**   | mcp-full | 38.3s    | 51437     | 28230      | 0             | 1785       | $0.31593   |

### Answer: `sonnet` / `baseline`

Here's how TSDB compaction works end-to-end:

---

## Compactor Interface (`compact.go:52-77`)

```go
type Compactor interface {
    Plan(dir string) ([]string, error)
    Write(dest string, b BlockReader, mint, maxt int64, base *BlockMeta) ([]ulid.ULID, error)
    Compact(dest string, dirs []string, open []*Block) ([]ulid.ULID, error)
}
```

- **`Plan`** — scans a directory and returns sets of block paths that should be
  compacted together
- **`Write`** — persists a single `BlockReader` (typically the in-memory Head)
  into a new immutable block
- **`Compact`** — merges multiple on-disk block directories into one new block

---

## LeveledCompactor (`compact.go:79-93`)

```go
type LeveledCompactor struct {
    ranges      []int64                              // e.g. [2h, 6h, 24h, 72h]
    chunkPool   chunkenc.Pool
    ctx         context.Context
    mergeFunc   storage.VerticalChunkSeriesMergeFunc
    enableOverlappingCompaction bool
    // + logger, metrics, encoders, etc.
}
```

**`Plan(dir)`** (`compact.go:249-277`):

1. Reads all block subdirectories and their `meta.json` files
2. Optionally filters via `blockExcludeFunc`
3. Delegates to internal `plan()` which:
   - Sorts blocks by `MinTime`
   - Detects overlapping blocks (triggers vertical/overlapping compaction)
   - Selects blocks whose combined span fits within a `ranges[]` boundary
   - Also selects blocks with >5% tombstones for early compaction

**`Compact(dest, dirs, open)`** (`compact.go:485-584`):

1. Opens/reads `BlockMeta` for each directory
2. Calls `CompactBlockMetas()` to compute merged metadata and increment
   `CompactionMeta.Level`
3. Calls internal `write()` to merge all series, chunks, and index into a new
   block
4. If the result has 0 samples, marks source blocks `Deletable` instead of
   writing

**`Write(dest, reader, mint, maxt, base)`** (`compact.go:586-636`):

- Used for flushing the in-memory **Head** to disk as a level-1 block
- Generates a new ULID, sets `Compaction.Level = 1`
- Calls same internal `write()` path as `Compact`

---

## DB Triggers Compaction (`db.go`)

### DB struct (relevant fields)

```go
type DB struct {
    dir       string
    compactor Compactor   // LeveledCompactor
    blocks    []*Block    // currently loaded on-disk blocks
    head      *Head       // in-memory mutable block
    cmtx      sync.Mutex  // serializes compaction runs
    // ...
}
```

### `DB.Compact()` — main entry point (`db.go:1414-1506`)

Called periodically by a background goroutine:

1. **Acquires `cmtx`** to prevent concurrent compaction
2. **Head → disk**: loops while head is compactable (old enough), calls
   `compactHead()` → `compactor.Write()`
3. **OOO head**: calls `compactOOOHead()` for out-of-order samples
4. **On-disk blocks**: calls `compactBlocks()`
5. **WAL truncation**: truncates WAL up to the persisted head range

### `compactBlocks()` (`db.go:1721-1763`)

```
loop:
    plan  = compactor.Plan(db.dir)      // which blocks to merge?
    uids  = compactor.Compact(db.dir, plan, db.blocks)  // merge them
    reloadBlocks()                       // reload block list
until Plan returns nothing
```

### `reloadBlocks()` (`db.go:1795-1902`)

After each compaction:

1. Scans `db.dir` for block directories
2. Marks **parent blocks** as `Deletable` (their ULIDs appear in child's
   `Compaction.Sources`)
3. Applies **retention** — removes blocks outside time/size limits
4. Closes stale `*Block` handles, opens new ones
5. Calls `deleteBlocks()` to physically remove `Deletable` blocks

---

## Key Types

```go
type BlockMeta struct {
    ULID       ulid.ULID
    MinTime    int64
    MaxTime    int64
    Compaction CompactionMeta
    Stats      BlockStats
}

type CompactionMeta struct {
    Level     int          // incremented each merge: 1 → 2 → 3 …
    Sources   []ulid.ULID  // all original level-1 block ULIDs in lineage
    Parents   []BlockDesc  // direct parents (input to this compaction)
    Deletable bool         // safe to delete from disk
    Failed    bool
}

type BlockDesc struct {
    ULID    ulid.ULID
    MinTime int64
    MaxTime int64
}
```

---

## End-to-End Flow Summary

```
[background goroutine]
        │
        ▼
  DB.Compact()
  ├── compactHead()         Head (RAM) ──► Write() ──► level-1 block on disk
  ├── compactOOOHead()      OOO Head  ──► Write() ──► OOO blocks on disk
  └── compactBlocks()
        └── loop:
              Plan(dir)     → picks e.g. three 2h blocks fitting a 6h range
              Compact(...)  → merges them into one level-2 block
              reloadBlocks()→ marks parents Deletable, applies retention,
                              physically deletes old blocks
```

Compaction levels grow as time ranges widen: `2h → 6h → 24h → 72h`, with each
merge incrementing `CompactionMeta.Level` and carrying forward the full
`Sources` lineage for deletion tracking.

### Answer: `sonnet` / `mcp-only`

Here's a complete end-to-end explanation of TSDB compaction:

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

Three methods:

- **`Plan`** — scans a directory and returns sets of block dirs that are ready
  to be compacted (can be called concurrently)
- **`Write`** — persists a single `BlockReader` (e.g., the in-memory Head) to
  disk as a new Level 1 block
- **`Compact`** — merges multiple existing on-disk blocks into one

---

### 2. `LeveledCompactor` (`compact.go:79-93`)

```go
type LeveledCompactor struct {
    ranges                      []int64   // time range boundaries per level
    chunkPool                   chunkenc.Pool
    mergeFunc                   storage.VerticalChunkSeriesMergeFunc
    blockExcludeFunc            BlockExcludeFilterFunc
    enableOverlappingCompaction bool
    // ...metrics, logger, ctx, etc.
}
```

Constructed via `NewLeveledCompactorWithOptions` (`compact.go:202-241`),
defaulting `mergeFunc` to `CompactingChunkSeriesMerger(ChainedSeriesMerge)` and
`maxBlockChunkSegmentSize` to `chunks.DefaultChunkSegmentSize`.

**`Plan` → `plan` (`compact.go:248-328`):**

1. Reads all block dirs, loads their `BlockMeta`
2. First tries `selectOverlappingDirs` — if any blocks have overlapping time
   ranges, compact those first (vertical compaction)
3. Otherwise calls `selectDirs` (`compact.go:330-367`) — groups blocks by the
   configured `ranges` levels; picks a group if it spans a full range interval
   or all blocks in it predate the most recent block
4. Falls back to single-block tombstone rewrite: if `>5%` of series are
   tombstoned in a sufficiently old block, returns that block alone for
   rewriting

**`Compact` → `CompactWithBlockPopulator` (`compact.go:483-584`):**

1. Opens each dir's `Block` (reuses already-open blocks to avoid double-loading
   index)
2. Calls `CompactBlockMetas` to merge metadata (min/max time, sources, level)
3. Calls `c.write(dest, meta, DefaultBlockPopulator{}, blocks...)` to
   stream-merge all series and write the new block
4. If the result has 0 samples, marks source blocks `Deletable` instead of
   writing
5. Returns the new `[]ulid.ULID`

**`Write` (`compact.go:586-636`):** Used for Head→disk flushing. Creates a Level
1 block from a `BlockReader`, delegates to the same internal `c.write` path.

---

### 3. How `DB` Triggers Compaction

#### Background loop (`db.go:1175-1243`, `DB.run`)

```go
func (db *DB) run(ctx context.Context) {
    for {
        select {
        case <-time.After(db.opts.BlockReloadInterval):
            db.reloadBlocks()
            db.compactc <- struct{}{}   // nudge compaction
            db.head.mmapHeadChunks()
            // stale series threshold check → CompactStaleHead()
        case <-db.compactc:
            if db.autoCompact {
                db.Compact(ctx)         // main compaction entry point
            }
        }
    }
}
```

A tick fires every `BlockReloadInterval`, reloads blocks, then signals
`compactc`. The `compactc` handler calls `DB.Compact`.

#### `DB.Compact` (`db.go:1410-1506`) — the orchestrator

```go
func (db *DB) Compact(ctx context.Context) error {
    db.cmtx.Lock()          // serializes with deletions
    // ...
}
```

Three phases, in priority order:

**Phase 1 — Head compaction (highest priority):**

```
while head.compactable() && !waitingForCompactionDelay():
    rh := NewRangeHeadWithIsolationDisabled(head, mint, maxt-1)
    head.WaitForAppendersOverlapping(rh.MaxTime())
    db.compactHead(rh)         // → compactor.Write(db.dir, rh, ...)
    lastBlockMaxt = maxt
```

`head.compactable()` returns true when the head's oldest chunk is at least
`chunkRange/2` behind the current time. `compactHead` calls `compactor.Write`
(persists the range to disk), then `reloadBlocks` + `head.truncateMemory`.

**Phase 2 — OOO Head compaction:**

```
if lastBlockMaxt != MinInt64:
    db.compactOOOHead(ctx)     // writes OOO blocks, truncates WBL/mmap refs
```

Only runs if out-of-order was ever enabled. Uses `NewOOOCompactionHead` to
collect OOO chunks and writes them as separate blocks.

**Phase 3 — On-disk block compaction:**

```
db.compactBlocks()
```

#### `DB.compactBlocks` (`db.go:1719-1763`)

```go
for {
    if head.compactable() && !waitingForCompactionDelay() {
        // Head needs attention again; stop block compaction early
        break
    }
    plan, _ := db.compactor.Plan(db.dir)
    if len(plan) == 0 { break }
    uids, _ := db.compactor.Compact(db.dir, plan, db.blocks)
    db.reloadBlocks()
}
```

Loops: plan → compact → reload, until `Plan` returns nothing or the head becomes
compactable again (Head always wins priority).

---

### Summary: Data Flow

```
WAL append → Head (in-memory)
    │
    │  [head.compactable() + delay elapsed]
    ▼
compactor.Write(Head range) → Level-1 block on disk
    │
    │  [Plan finds ≥2 blocks in same range level]
    ▼
compactor.Compact(dirs) → higher-level block on disk
    │
    ▼
reloadBlocks() → deletes source blocks marked Deletable
                  + applies retention (drops old blocks)
```

Compaction levels are defined by `ranges []int64` (e.g., `[2h, 6h, 24h]`).
Blocks get merged upward through levels until they're either deleted by
retention or reach the max range.

### Answer: `sonnet` / `mcp-full`

Here is a full end-to-end explanation of TSDB compaction.

---

## TSDB Compaction — End-to-End

### 1. The `Compactor` Interface (`compact.go:52-77`)

```go
type Compactor interface {
    Plan(dir string) ([]string, error)
    Write(dest string, b BlockReader, mint, maxt int64, base *BlockMeta) ([]ulid.ULID, error)
    Compact(dest string, dirs []string, open []*Block) ([]ulid.ULID, error)
}
```

Three methods define the contract:

- **`Plan`** — scans a directory and returns sets of block dirs that should be
  compacted together.
- **`Write`** — persists a single `BlockReader` (typically the in-memory Head)
  to disk as a new block.
- **`Compact`** — merges multiple on-disk block directories into a single new
  block.

---

### 2. `LeveledCompactor` (`compact.go:79-93`)

The sole production implementation:

```go
type LeveledCompactor struct {
    ranges                      []int64   // e.g. [2h, 6h, 24h, 72h]
    chunkPool                   chunkenc.Pool
    mergeFunc                   storage.VerticalChunkSeriesMergeFunc
    enableOverlappingCompaction bool
    blockExcludeFunc            BlockExcludeFilterFunc
    postingsEncoder             index.PostingsEncoder
    // ...metrics, logger, ctx
}
```

The `ranges` slice is the key: it defines a hierarchy of block durations. Blocks
are promoted up levels as they get merged.

**Construction** via `NewLeveledCompactorWithOptions` (`compact.go:202-241`) —
applies defaults for merge func, chunk segment size, and postings encoder.

---

### 3. Planning: Which blocks to compact? (`compact.go:279-328`)

`LeveledCompactor.plan(dms []dirMeta)` does the selection:

```
1. Sort blocks by MinTime.
2. selectOverlappingDirs — if overlapping blocks exist AND overlapping compaction is enabled,
   compact those first (vertical compaction).
3. selectDirs — for each range level (e.g. 6h, 24h, 72h), use splitByRange to group
   blocks that fall within an aligned window. Pick the first group with >1 block
   that either spans the full range or is old enough.
4. Tombstone rewrite — if a block has >5% tombstone ratio and is large enough,
   select it alone for rewrite.
```

`selectDirs` (`compact.go:330-367`) iterates `c.ranges[1:]` (skipping the base
level), calls `splitByRange` to bucket blocks into aligned windows, and returns
the first eligible group of 2+ blocks.

`selectOverlappingDirs` (`compact.go:369-394`) finds the first contiguous run of
blocks with overlapping `[MinTime, MaxTime)` — used for vertical compaction of
out-of-order data.

---

### 4. The `DB` struct (`db.go:281-342`)

```go
type DB struct {
    compactor   Compactor
    compactc    chan struct{}   // signals that compaction should run
    cmtx        sync.Mutex     // serialises compaction + deletion
    autoCompact bool
    head        *Head
    // ...
}
```

---

### 5. The Compaction Loop (`db.go:1175-1243`)

`DB.run(ctx)` is the background goroutine:

```
loop:
  - on BlockReloadInterval tick → reload blocks, signal compactc
  - on compactc → call db.Compact(ctx) if autoCompact
  - on stopc → return
```

Stale-series compaction can also trigger `db.CompactStaleHead()` inline if the
ratio of stale series exceeds a threshold.

---

### 6. `DB.Compact` — the main orchestrator (`db.go:1410-1506`)

```go
func (db *DB) Compact(ctx context.Context) error {
    // Phase 1: Flush compactable Head ranges to disk
    for db.head.compactable() {
        mint := db.head.MinTime()
        maxt := rangeForTimestamp(mint, db.head.chunkRange)
        rh := NewRangeHeadWithIsolationDisabled(db.head, mint, maxt-1)
        db.head.WaitForAppendersOverlapping(rh.MaxTime())
        db.compactHead(rh)          // → compactor.Write(...)
        lastBlockMaxt = maxt
    }

    // Phase 2: Truncate WAL up to the last flushed maxt
    db.head.truncateWAL(lastBlockMaxt)

    // Phase 3: If head was flushed, compact OOO head too
    if lastBlockMaxt != math.MinInt64 {
        db.compactOOOHead(ctx)
    }

    // Phase 4: Compact on-disk blocks
    return db.compactBlocks()
}
```

---

### 7. Head → Disk (`db.go:1634-1662`)

```go
func (db *DB) compactHead(head *RangeHead) error {
    uids, err := db.compactor.Write(db.dir, head, head.MinTime(), head.BlockMaxTime(), nil)
    db.reloadBlocks()
    db.head.truncateMemory(head.BlockMaxTime())
    db.head.RebuildSymbolTable(db.logger)
    return err
}
```

`compactor.Write` serialises the `RangeHead` (an in-memory view over the
WAL/mmap chunks) into an immutable on-disk block (chunks + index + tombstones +
meta.json).

---

### 8. On-disk block merging (`db.go:1719-1763`)

```go
func (db *DB) compactBlocks() error {
    for {
        plan, _ := db.compactor.Plan(db.dir)  // ask LeveledCompactor
        if len(plan) == 0 { break }

        uids, _ := db.compactor.Compact(db.dir, plan, db.blocks)
        db.reloadBlocks()
    }
    return nil
}
```

`LeveledCompactor.Compact` (via `CompactWithBlockPopulator`)
(`compact.go:489-584`):

1. Opens each block dir (reuses already-open `*Block` objects to avoid double
   memory).
2. Calls `CompactBlockMetas` to merge the `BlockMeta` structs — increments
   compaction `Level`, union of `Sources`, merged `[MinTime, MaxTime)`.
3. Calls `c.write(dest, meta, blockPopulator, blocks...)` — iterates all series,
   merges chunks (using `mergeFunc` for overlapping series), writes new chunk
   segments + inverted index + tombstones.
4. If the resulting block has 0 samples, marks source blocks `Deletable` instead
   of writing.
5. Returns the new block `ULID`; `DB.reloadBlocks` picks it up and marks old
   blocks for deletion.

---

### Summary flow

```
WAL/Head compactable?
  └─▶ compactor.Write(RangeHead) → new Level-1 block on disk
        └─▶ head.truncateMemory()

compactor.Plan(db.dir)
  └─▶ LeveledCompactor.plan()
        ├─▶ selectOverlappingDirs → vertical compaction (OOO)
        └─▶ selectDirs + splitByRange → pick blocks filling a range window

compactor.Compact(plan)
  └─▶ CompactWithBlockPopulator
        ├─▶ merge chunk series (mergeFunc)
        ├─▶ write new block (chunks + index + meta)
        └─▶ mark sources Deletable

DB.reloadBlocks()
  └─▶ delete Deletable blocks, load new block into db.blocks
```

The leveled design means small 2 h blocks get merged into 6 h, then 24 h, then
72 h blocks — each merge increments `Compaction.Level` in the block's
`meta.json`.

### Answer: `opus` / `baseline`

Here's how TSDB compaction works end-to-end:

---

## The `Compactor` Interface (`compact.go:54-77`)

```go
type Compactor interface {
    Plan(dir string) ([]string, error)
    Write(dest string, b BlockReader, mint, maxt int64, base *BlockMeta) ([]ulid.ULID, error)
    Compact(dest string, dirs []string, open []*Block) ([]ulid.ULID, error)
}
```

Three responsibilities:

- **`Plan`** — scans block directories and returns which ones should be
  compacted together
- **`Write`** — persists a single `BlockReader` (typically the in-memory Head)
  to a new on-disk block
- **`Compact`** — merges multiple existing on-disk blocks into one new block

---

## `LeveledCompactor` (`compact.go:80-93`)

The concrete implementation. Key fields:

| Field                         | Purpose                                                                  |
| ----------------------------- | ------------------------------------------------------------------------ |
| `ranges []int64`              | Exponential time ranges (e.g. 2h, 6h, 18h) that define compaction levels |
| `mergeFunc`                   | Strategy for merging overlapping chunk series (vertical compaction)      |
| `enableOverlappingCompaction` | Whether to merge time-overlapping blocks                                 |
| `blockExcludeFunc`            | Optional filter to skip blocks (used by Mimir/Thanos)                    |

### Planning algorithm (`plan`, line 279)

1. Sort blocks by `MinTime`
2. **First priority**: find overlapping blocks via `selectOverlappingDirs` — if
   any block's `MinTime < previous block's MaxTime`, group them for vertical
   compaction
3. **Second priority**: level-based compaction via `selectDirs` — uses
   `splitByRange` to group blocks into aligned time windows at each range level,
   picks the first group of 2+ blocks that spans a full range or is behind the
   newest block
4. **Third priority**: tombstone cleanup — any block with >5% tombstones or
   fully deleted series

### Compact flow (`Compact`, line 485)

1. Read each block's `meta.json`, reuse already-open `*Block` objects when
   possible
2. Call `CompactBlockMetas` — merges metadata, increments compaction level,
   records parent blocks
3. Call `write` — creates a temp dir, opens chunk/index writers, calls
   `BlockPopulator.PopulateBlock` to iterate all series across input blocks,
   merge them, and write chunks + index entries
4. Atomically rename temp dir to final block dir
5. If the result has 0 samples, mark source blocks as `Deletable`
6. On failure, mark source blocks' compaction as `Failed`

### `PopulateBlock` (`DefaultBlockPopulator`, line 793)

The core merge logic: opens index/chunk/tombstone readers for each input block,
creates `ChunkSeriesSet` iterators, merges symbols, then iterates all series
writing chunks and index entries while accumulating stats into the new block's
`BlockMeta`.

---

## How the DB Triggers Compaction

### The `run` loop (`db.go:1175`)

The DB starts a background goroutine that runs a select loop on two channels:

1. **`BlockReloadInterval` timer** (default 1 min) — reloads blocks from disk,
   then sends on `compactc` to trigger compaction. Also checks stale series
   ratio for early compaction.
2. **`compactc` channel** — when received, calls `db.Compact(ctx)` if
   `autoCompact` is true. Uses exponential backoff on failure.

External triggers (e.g. `dbAppender.Commit`) also send on `compactc` to signal
the head is ready.

### `DB.Compact` (`db.go:1414`) — the main entry point

Three phases executed sequentially:

1. **Head compaction** — loops while `head.compactable()` returns true:
   - Builds a `RangeHead` from `[mint, maxt)` using the head's chunk range
   - Waits for overlapping appenders to finish
   - Calls `compactHead` → `compactor.Write(db.dir, head, ...)` — writes head
     data to a new level-1 block
   - Reloads blocks, truncates head memory and WAL

2. **OOO (out-of-order) head compaction** — if the head was compacted, also
   compact out-of-order data via `compactOOOHead`

3. **Block compaction** via `compactBlocks` (`db.go:1721`):
   - Loops calling `compactor.Plan(db.dir)` until no more plans
   - For each plan, calls `compactor.Compact(db.dir, plan, db.blocks)`
   - Reloads blocks after each compaction
   - Aborts early if the head becomes compactable again (head has priority)

### Control mechanisms

- `DisableCompactions` / `EnableCompactions` (`db.go:2243-2256`) — toggle
  `autoCompact` flag
- `compactCancel` context — allows canceling a running compaction on shutdown
- `cmtx` mutex — ensures compactions and deletions don't run simultaneously
- Optional `CompactionDelay` — delays compaction start to allow backups of fresh
  data

---

**In summary**: the DB's background loop periodically triggers `Compact`, which
first flushes the in-memory Head to level-1 blocks via `Write`, then repeatedly
calls `Plan` + `Compact` to merge small blocks into larger ones following the
exponential range levels — all orchestrated through the `Compactor` interface
with `LeveledCompactor` as the default implementation.

### Answer: `opus` / `mcp-only`

Here's how TSDB compaction works end-to-end:

---

## 1. The `Compactor` Interface

Defined at `compact.go:52-77`, this is the abstraction all compaction goes
through:

```go
type Compactor interface {
    Plan(dir string) ([]string, error)
    Write(dest string, b BlockReader, mint, maxt int64, base *BlockMeta) ([]ulid.ULID, error)
    Compact(dest string, dirs []string, open []*Block) ([]ulid.ULID, error)
}
```

- **`Plan`** — scans the data directory for block directories, reads their
  metadata, and returns a list of block dirs that should be compacted together.
- **`Write`** — persists a `BlockReader` (typically the in-memory Head) into a
  new on-disk block. Used for head compaction.
- **`Compact`** — merges multiple existing on-disk blocks into a single new
  block. Used for block-level compaction.

## 2. `LeveledCompactor` — The Implementation

Defined at `compact.go:79-93`:

```go
type LeveledCompactor struct {
    metrics                     *CompactorMetrics
    logger                      *slog.Logger
    ranges                      []int64          // time ranges for each compaction level
    chunkPool                   chunkenc.Pool
    ctx                         context.Context
    mergeFunc                   storage.VerticalChunkSeriesMergeFunc
    enableOverlappingCompaction bool
    blockExcludeFunc            BlockExcludeFilterFunc
    postingsEncoder             index.PostingsEncoder
    postingsDecoderFactory      PostingsDecoderFactory
    // ...
}
```

**Key `ranges` field**: This is a slice of time ranges (e.g.,
`[2h, 6h, 18h, 54h]`). Blocks are compacted into progressively larger time
ranges — this is the "leveled" part.

### Planning (`compact.go:248-328`)

`Plan` reads block metadata from disk, then `plan` applies three strategies in
priority order:

1. **Overlapping blocks** — `selectOverlappingDirs` finds blocks whose time
   ranges overlap and returns them for vertical compaction.
2. **Leveled compaction** — `selectDirs` groups adjacent blocks that fit within
   the next level's time range. The newest block is excluded to give users a
   backup window.
3. **Tombstone compaction** — blocks with >5% tombstones (or 100% deleted
   series) are compacted to reclaim space.

### Compacting (`compact.go:483-584`)

`Compact` delegates to `CompactWithBlockPopulator`, which:

1. Opens each block directory (reusing already-open blocks when possible)
2. Merges block metadata via `CompactBlockMetas` (increments compaction level,
   unions sources)
3. Calls `c.write()` to create the new block
4. If the result has 0 samples, marks source blocks as `Deletable` instead

### Writing (`compact.go:657-716`)

The internal `write` method:

1. Creates a temp directory (`<ulid>.tmp-for-creation`)
2. Opens chunk and index writers
3. Calls `blockPopulator.PopulateBlock()` (`DefaultBlockPopulator`) to iterate
   all series from the source blocks and write chunks + index entries
4. Writes the `meta.json`, then atomically renames temp dir to final

## 3. How the DB Triggers Compaction

### Trigger 1: Background loop (`db.go:1175-1243`)

`DB.run` is a goroutine started at DB open. It has a `select` loop with two
triggers:

- **Timer tick** (`BlockReloadInterval`): reloads blocks, then sends on
  `db.compactc` channel. Also checks for stale series — if the stale ratio
  exceeds a threshold and the next regular compaction isn't imminent, triggers
  `CompactStaleHead()`.
- **`db.compactc` channel**: if `autoCompact` is true, calls `db.Compact(ctx)`.

### Trigger 2: Appender commit (`db.go:1360-1372`)

Every time an appender commits, it checks `head.compactable()`. If true, it
sends on `db.compactc` (non-blocking) to wake up the background loop.

### The `DB.Compact` orchestration (`db.go:1410-1506`)

This is the main compaction entry point, called under `db.cmtx` lock:

**Phase 1 — Head compaction** (loop):

1. Check `head.compactable()` — head has enough data spanning at least half a
   chunk range
2. Optionally apply compaction delay (`waitingForCompactionDelay`)
3. Create a `RangeHead` bounding `[mint, maxt-1]`
4. Wait for overlapping appenders to finish
5. Call `db.compactHead(rh)` → `db.compactor.Write()` to persist head to a block
6. Truncate WAL, truncate head memory, repeat until head is no longer
   compactable

**Phase 2 — OOO Head compaction** (`db.go:1534-1632`):

- If out-of-order ingestion was enabled, compact `OOOCompactionHead` into blocks
  (one per block-range interval)
- Then truncate OOO WAL and garbage-collect mmap'd chunks

**Phase 3 — Block compaction** (`db.go:1719-1763`):

```go
func (db *DB) compactBlocks() error {
    for {
        if db.head.compactable() && !db.waitingForCompactionDelay() {
            return nil  // abort to persist head first
        }
        plan, _ := db.compactor.Plan(db.dir)
        if len(plan) == 0 { break }
        uids, _ := db.compactor.Compact(db.dir, plan, db.blocks)
        db.reloadBlocks()  // reload + delete old blocks
    }
}
```

Loops calling `Plan` → `Compact` → `reloadBlocks` until no more blocks need
merging. Aborts early if the head becomes compactable again.

### Compactor initialization (`db.go:990-1010`)

At DB open, either a custom `NewCompactorFunc` is used or the default
`NewLeveledCompactorWithOptions` is created with the configured ranges and
options.

## End-to-End Flow Summary

```
Appender.Commit()  ──or──  Timer tick
        │                       │
        └──── compactc chan ────┘
                    │
              DB.Compact()
                    │
        ┌───────────┼──────────────┐
        ▼           ▼              ▼
   Head → Block   OOO Head    Plan+Compact
   (Write)        → Blocks    existing blocks
        │           │              │
        ▼           ▼              ▼
   reloadBlocks  reloadBlocks  reloadBlocks
   truncateWAL   truncateOOO   (deletes old)
```

### Answer: `opus` / `mcp-full`

Here's how TSDB compaction works end-to-end:

## 1. The `Compactor` Interface (`compact.go:52-77`)

```go
type Compactor interface {
    Plan(dir string) ([]string, error)
    Write(dest string, b BlockReader, mint, maxt int64, base *BlockMeta) ([]ulid.ULID, error)
    Compact(dest string, dirs []string, open []*Block) ([]ulid.ULID, error)
}
```

Three responsibilities:

- **Plan** — scan on-disk block directories and return groups that should be
  compacted together
- **Write** — persist a `BlockReader` (e.g. the in-memory head) into a new
  on-disk block
- **Compact** — merge multiple existing on-disk blocks into one new block

## 2. `LeveledCompactor` (`compact.go:79-93`)

The production implementation. Key fields:

```go
type LeveledCompactor struct {
    ranges                      []int64          // time ranges per compaction level
    chunkPool                   chunkenc.Pool
    mergeFunc                   storage.VerticalChunkSeriesMergeFunc
    enableOverlappingCompaction bool
    blockExcludeFunc            BlockExcludeFilterFunc
    // ...
}
```

**Planning** (`compact.go:279-328`): `plan()` sorts blocks by MinTime, then:

1. First picks overlapping blocks (vertical compaction)
2. If none, uses `selectDirs` for level-based compaction (excludes the newest
   block to allow backups)
3. As a fallback, compacts blocks with >5% tombstones

**Compact** (`compact.go:483-487`): Delegates to `CompactWithBlockPopulator`,
which opens each block, merges their metas via `CompactBlockMetas`, then calls
`write()`.

**write** (`compact.go:657+`): Creates a temp directory, writes chunks and index
via `BlockPopulator.PopulateBlock`, then renames the temp dir to the final
ULID-based path. On error, the temp dir is cleaned up.

## 3. How `DB` Triggers Compaction

### DB struct (`db.go:281-342`)

Key compaction-related fields:

```go
type DB struct {
    compactor      Compactor
    compactc       chan struct{}    // signal channel
    autoCompact    bool
    compactCancel  context.CancelFunc
    cmtx           sync.Mutex      // serializes compactions and deletions
    // ...
}
```

### Initialization (`db.go:990+`)

During `open()`, the DB creates a `LeveledCompactor` (or uses
`opts.NewCompactorFunc` if provided):

```go
if opts.NewCompactorFunc != nil {
    db.compactor, err = opts.NewCompactorFunc(ctx, r, l, rngs, db.chunkPool, opts)
} else {
    db.compactor, err = NewLeveledCompactorWithOptions(ctx, r, l, rngs, db.chunkPool, ...)
}
```

### Three trigger paths

**A. Appender commit** (`db.go:1360-1372`): After every `Commit()`, if
`head.compactable()` is true, a signal is sent on `compactc`:

```go
if a.db.head.compactable() {
    select {
    case a.db.compactc <- struct{}{}:
    default:
    }
}
```

**B. Periodic tick** (`db.go:1175-1243`): `DB.run()` is a background goroutine
with two select cases:

- Every `BlockReloadInterval` — reloads blocks, then sends on `compactc`; also
  checks for stale series compaction
- On `<-db.compactc` — if `autoCompact` is true, calls `db.Compact(ctx)` with
  exponential backoff on failure

**C. Stale series** (`db.go:1204+`): If the ratio of stale series exceeds the
threshold and the next regular compaction isn't imminent (within 10 min),
triggers `CompactStaleHead()` immediately.

### `DB.Compact()` — the main flow (`db.go:1410-1506`)

Three phases, in order:

1. **Head compaction** — loops while `head.compactable()`:
   - Optionally delays start (`CompactionDelay`)
   - Creates a `RangeHead` for the compactable time range
   - Waits for overlapping appenders to finish
   - Calls `compactHead()` → `compactor.Write()` to persist head to a block
   - Truncates WAL up to the compacted max time

2. **OOO head compaction** — if head was compacted and out-of-order was ever
   enabled:
   - `compactOOOHead()` → `compactOOO()` creates one block per block-range
     interval from the OOO head
   - Each block is written via `compactor.Write()`, then blocks are reloaded and
     OOO WAL/chunks are truncated

3. **Block compaction** (`db.go:1719-1763`) — `compactBlocks()` loops:
   - Calls `compactor.Plan(db.dir)` to find compactable block groups
   - Calls `compactor.Compact(db.dir, plan, db.blocks)` to merge them
   - Reloads blocks after each compaction (which also applies retention)
   - Breaks early if the head becomes compactable again (head has priority)

### Control

- `DB.EnableCompactions()` / `DB.DisableCompactions()` toggle `autoCompact`
  under `autoCompactMtx`
- `compactCancel` cancels the compactor's context on DB shutdown, aborting
  in-progress compactions

### Full Judge Analysis (Opus 4.6)

## Content Quality

### Ranking: Best to Worst

**1. opus / mcp-full**

The most well-structured and accurate answer. Correctly identifies the
`Compactor` interface with all three methods, the `LeveledCompactor` struct with
key fields, and the three-phase compaction flow in `DB.Compact()`. Uniquely
includes the appender commit trigger path (`db.go:1360-1372`) with the actual
code showing the non-blocking channel send, the compactor initialization code
from `db.go:990+`, and the `EnableCompactions`/`DisableCompactions` control
mechanisms. Line references are precise. The planning algorithm description
(overlapping → leveled → tombstone) is concise yet complete. The only minor
omission is less detail on `PopulateBlock`/write internals, but this is a
reasonable scope choice.

**2. opus / mcp-only**

Nearly as complete as opus/mcp-full. Covers all three trigger paths (appender
commit, timer tick, stale series), includes the compactor initialization, and
provides accurate line references. The end-to-end flow diagram is clear and
useful. Includes good detail on the `write` method internals (temp directory,
atomic rename) and `DefaultBlockPopulator`. The planning section correctly
describes all three priority levels. Very close to opus/mcp-full but slightly
less polished in organization.

**3. sonnet / mcp-full**

Strong answer with accurate interface definitions, good coverage of the planning
algorithm (including `selectDirs` and `selectOverlappingDirs` with line refs),
and correct DB orchestration. Includes the `DB.run` background loop and
`compactBlocks` loop. The `compactHead` section (`db.go:1634-1662`) with WAL
truncation and symbol table rebuild is a nice detail. Slightly weaker than the
opus answers — misses the appender-commit trigger and control mechanisms.

**4. sonnet / baseline**

Impressively detailed for a baseline (no MCP) run. Includes `BlockMeta`,
`CompactionMeta`, and `BlockDesc` type definitions that no other answer provides
— useful context for understanding deletion tracking. The `reloadBlocks`
description (marking parents deletable, applying retention, deleting blocks) is
the most thorough across all answers. However, some line references may be
approximate since it lacked semantic search, and it misses the `DB.run`
background loop trigger mechanism and appender commit trigger.

**5. opus / baseline**

Correct and well-organized with good coverage of the planning algorithm,
`PopulateBlock`/`DefaultBlockPopulator` details, and the background `run` loop.
Mentions `DisableCompactions`/`EnableCompactions`, exponential backoff on
failure, and the appender commit trigger. Line references are reasonable.
However, the answer feels slightly less precise in its code snippets compared to
the MCP-assisted opus answers, and the `CompactWithBlockPopulator` flow
description is less detailed.

**6. sonnet / mcp-only**

Accurate and covers all major components, but slightly less detailed than the
other answers. The `DB.run` loop code snippet is helpful, and the three-phase
breakdown of `DB.Compact` is correct. However, it provides less detail on
`PopulateBlock`, the write internals, and control mechanisms. The planning
algorithm description is adequate but less precise than the top answers. Still a
solid answer — the gap from #5 is small.

---

## Efficiency Analysis

| Scenario          | Duration | Total Input | Cost  | Quality Rank |
| ----------------- | -------- | ----------- | ----- | ------------ |
| opus / mcp-full   | 38.3s    | 51,437      | $0.32 | 1st          |
| sonnet / mcp-only | 45.9s    | 49,394      | $0.30 | 6th          |
| opus / mcp-only   | 49.5s    | 42,393      | $0.27 | 2nd          |
| sonnet / mcp-full | 53.1s    | 87,711      | $0.53 | 3rd          |
| opus / baseline   | 81.4s    | 285,260     | $1.58 | 5th          |
| sonnet / baseline | 96.3s    | 30,539      | $1.13 | 4th          |

**Key observations:**

- **Baseline runs are dramatically more expensive.** Opus/baseline cost 5-6x
  more than opus/mcp variants ($1.58 vs ~$0.30), largely due to massive input
  token counts (285K) from reading entire files. Sonnet/baseline's high cost
  ($1.13) comes from cache read tokens (28K) contributing to the bill.

- **MCP-only is the cheapest option** for both models (~$0.27-0.30), while
  delivering strong quality — especially for opus where it ranked 2nd overall.

- **opus/mcp-full is the best quality-to-cost tradeoff.** It produced the
  top-ranked answer at $0.32 in the fastest time (38.3s). The combination of
  semantic search plus full tool access let opus target exactly the right code
  sections without bloating context.

- **Sonnet/mcp-full used nearly 2x the input tokens of opus/mcp-full** (87K vs
  51K) for a worse result, suggesting opus is more efficient at using search
  results and reading only what's needed.

- **Cache reads on baseline runs** are notable: sonnet/baseline had 28K cache
  reads (likely from prompt caching), and opus/baseline had 155K. These reduce
  cost somewhat but baselines are still far more expensive.

**Recommendation:** **opus/mcp-full** — best answer, fastest runtime, and only
$0.05 more than the cheapest option. If cost is the primary concern,
**opus/mcp-only** delivers nearly equivalent quality at the lowest price point
($0.27).

---

## promql-engine [very-hard]

**Question:** How does PromQL query evaluation work? Explain the evaluation
engine, how functions are registered and called, how the AST nodes are
evaluated, and how alert and recording rules trigger evaluations. Show key
interfaces, types, and function signatures.

### Metrics

| Model      | Scenario | Duration | Input Tok | Cache Read | Cache Created | Output Tok | Cost (USD) |
| ---------- | -------- | -------- | --------- | ---------- | ------------- | ---------- | ---------- |
| **sonnet** | baseline | 133.3s   | 33526     | 28104      | 0             | 2108       | $1.13111   |
| **sonnet** | mcp-only | 85.0s    | 193983    | 0          | 0             | 4701       | $1.08744   |
| **sonnet** | mcp-full | 75.3s    | 158657    | 84312      | 0             | 4247       | $0.94162   |
| **opus**   | baseline | 225.7s   | 33429     | 28230      | 0             | 2866       | $6.14290   |
| **opus**   | mcp-only | 75.5s    | 117776    | 0          | 0             | 4107       | $0.69156   |
| **opus**   | mcp-full | 89.0s    | 232269    | 84690      | 0             | 4661       | $1.32021   |

### Answer: `sonnet` / `baseline`

Here is how PromQL query evaluation works in this Prometheus codebase:

---

## Evaluation Engine

The engine entry point is `Engine.exec()` → `Engine.execEvalStmt()` → creates an
`evaluator`:

**`engine.go:345`** — `Engine` struct holds timeout, max samples, lookback
delta, query tracker/logger, and feature flags.

**`engine.go:773`** — `execEvalStmt` orchestrates evaluation:

- Determines time range from `EvalStmt`
- Creates a `storage.Querier`
- Populates series from storage
- For instant queries (start==end): single-step evaluator
- For range queries: multi-step evaluator

**`engine.go:1138`** — The `evaluator` struct:

```go
type evaluator struct {
    startTimestamp, endTimestamp, interval int64
    maxSamples, currentSamples             int
    lookbackDelta                          time.Duration
    querier                                storage.Querier
    // ... stats, logger, feature flags ...
}
```

**`engine.go:1189`** — `Eval()` is the public entry point; it calls internal
`eval()` and optionally cleans up metric labels.

---

## AST Node Evaluation

**`ast.go:77`** — The `Expr` interface:

```go
type Expr interface {
    Node
    Type() ValueType  // Scalar, Vector, or String
    PromQLExpr()      // marker method
}
```

**`ast.go:62`** — `EvalStmt` wraps the root expression with time bounds:

```go
type EvalStmt struct {
    Expr          Expr
    Start, End    time.Time
    Interval      time.Duration
    LookbackDelta time.Duration
}
```

**`engine.go:1905`** — `eval()` is a type-switch dispatcher over all AST node
types:

| Node type                | What happens                                                                      |
| ------------------------ | --------------------------------------------------------------------------------- |
| `*parser.AggregateExpr`  | sum, avg, min, max, topk, etc. — via `rangeEvalAgg`                               |
| `*parser.Call`           | Lookup in `FunctionCalls` map, dispatch to typed func                             |
| `*parser.BinaryExpr`     | Routes to `VectorBinop`, `VectorAnd`, `VectorOr`, `VectorUnless`, or scalar binop |
| `*parser.VectorSelector` | Reads time series from storage matching label selectors                           |
| `*parser.MatrixSelector` | Range selections like `metric[5m]`                                                |
| `*parser.SubqueryExpr`   | Nested evaluator with adjusted time range                                         |
| Literals                 | `NumberLiteral`, `StringLiteral` — returned directly                              |

**`engine.go:1390`** — `rangeEval()` drives the time loop: for each step from
`startTimestamp` to `endTimestamp`, it calls a `funcCall` closure with evaluated
argument vectors and collects results into a `Matrix`.

---

## Function Registry & Dispatch

**`functions.go:60`** — The function signature type:

```go
type FunctionCall func(
    vectorVals []Vector,
    matrixVals Matrix,
    args parser.Expressions,
    enh *EvalNodeHelper,
) (Vector, annotations.Annotations)
```

**`functions.go:2153`** — The registry is a plain `map[string]FunctionCall`:

```go
var FunctionCalls = map[string]FunctionCall{
    "abs":              funcAbs,
    "rate":             funcRate,
    "histogram_quantile": funcHistogramQuantile,
    // 90+ entries ...
}
```

**`engine.go:1962`** — Dispatch at call sites:

```go
case *parser.Call:
    call := FunctionCalls[e.Func.Name]   // direct map lookup
    // special-cased: label_replace, label_join, info → dedicated eval methods
    // otherwise → rangeEval with the call closure
```

**`engine.go:1210`** — `EvalNodeHelper` is a per-step scratch space to avoid
allocations:

```go
type EvalNodeHelper struct {
    Ts  int64    // current step timestamp
    Out Vector   // output accumulator
    // caches for histograms, quantiles, binary matching, label building ...
}
```

---

## Alert & Recording Rule Evaluation

Both rule types share the same `QueryFunc` hook back into the engine:

```go
type QueryFunc func(ctx context.Context, query string, ts time.Time) (promql.Vector, error)
```

**`alerting.go:116`** — `AlertingRule`:

```go
type AlertingRule struct {
    name         string
    vector       parser.Expr       // the alert condition
    holdDuration time.Duration     // pending → firing threshold
    keepFiringFor time.Duration
    active       map[uint64]*Alert // fingerprint → alert state
}
```

**`alerting.go:382`** — `AlertingRule.Eval()`:

1. Calls `query(ctx, r.vector.String(), ts)` to execute the PromQL expression
2. For each result sample: expands label/annotation templates, tracks active
   alerts
3. Manages state transitions: `Inactive → Pending → Firing`

**`recording.go:38`** — `RecordingRule`:

```go
type RecordingRule struct {
    name   string
    vector parser.Expr    // the recording expression
    labels labels.Labels  // labels to add to output
}
```

**`recording.go:85`** — `RecordingRule.Eval()`:

1. Calls `query(ctx, rule.vector.String(), ts)` to execute the expression
2. Overrides the metric name and merges labels on each result sample
3. Validates for duplicate labelsets and enforces the result `limit`

---

## Key Data Flow

```
Engine.exec()
  └─ execEvalStmt()           ← builds querier, populates series
       └─ evaluator.Eval()    ← panic recovery wrapper
            └─ eval()         ← type-switch dispatch
                 ├─ VectorSelector  → storage.Querier.Select()
                 ├─ MatrixSelector  → storage query with range
                 ├─ Call            → FunctionCalls[name](...)
                 ├─ AggregateExpr   → rangeEvalAgg()
                 └─ BinaryExpr      → VectorBinop / scalarBinop
                      └─ rangeEval() ← drives the time step loop

AlertingRule.Eval() ──┐
RecordingRule.Eval() ─┴─ QueryFunc → Engine → eval() (same pipeline)
```

Rules are evaluated on a schedule by the rule manager. They re-enter the engine
through `QueryFunc`, which serializes the expression back to a string and runs
it through the same `exec` pipeline — they don't bypass evaluation or get a
special code path.

### Answer: `sonnet` / `mcp-only`

Here is a comprehensive explanation of how PromQL query evaluation works in this
codebase (Prometheus).

---

## PromQL Query Evaluation

### 1. Core Interfaces and Types

**`engine.go:124`** — The top-level engine interface:

```go
type QueryEngine interface {
    NewInstantQuery(ctx, q storage.Queryable, opts QueryOpts, qs string, ts time.Time) (Query, error)
    NewRangeQuery(ctx, q storage.Queryable, opts QueryOpts, qs string, start, end time.Time, interval time.Duration) (Query, error)
}
```

**`engine.go:141`** — The `Query` interface returned to callers:

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

**`engine.go:189`** — Internal `query` struct (implements `Query`):

```go
type query struct {
    queryable   storage.Queryable
    q           string           // original query string
    stmt        parser.Statement // parsed AST
    stats       *stats.QueryTimers
    sampleStats *stats.QuerySamples
    matrix      Matrix
    cancel      func()
    ng          *Engine
}
```

---

### 2. AST Node Types

**`ast.go:76`** — The `Expr` interface that all expression nodes implement:

```go
type Expr interface {
    Node
    Type() ValueType
    PromQLExpr() // marker method to prevent accidental implementation
}
```

The concrete AST node types, each with their `PromQLExpr()` marker:

| Type                 | File         | Description                                   |
| -------------------- | ------------ | --------------------------------------------- |
| `*AggregateExpr`     | `ast.go:269` | `sum()`, `avg()`, `count()`, etc.             |
| `*BinaryExpr`        | `ast.go:270` | `+`, `-`, `and`, `or`, comparisons            |
| `*Call`              | `ast.go:271` | Function call — `rate()`, `abs()`, etc.       |
| `*MatrixSelector`    | `ast.go:272` | Range vector like `metric[5m]`                |
| `*SubqueryExpr`      | `ast.go:273` | `metric[1h:5m]`                               |
| `*NumberLiteral`     | `ast.go:274` | `42`                                          |
| `*ParenExpr`         | `ast.go:275` | `(expr)`                                      |
| `*StringLiteral`     | `ast.go:276` | `"foo"`                                       |
| `*UnaryExpr`         | `ast.go:277` | Unary negation `-`                            |
| `*VectorSelector`    | `ast.go:278` | `metric{label="val"}`                         |
| `*StepInvariantExpr` | `ast.go:279` | Wrapper for expressions constant across steps |

**`ast.go:123`** — `Call` specifically:

```go
type Call struct {
    Func *Function   // the function descriptor
    Args Expressions // arguments
}
```

Tree traversal is done via **`ChildrenIter`** (`ast.go:401`), which dispatches
by type using a type switch over all node types.

---

### 3. Query Execution Flow

```
API.query()
  → Engine.NewInstantQuery() / NewRangeQuery()
      → engine.newQuery()        // creates query struct + EvalStmt AST
  → query.Exec(ctx)             // engine.go:247
      → Engine.exec(ctx, q)     // engine.go:669
          → Engine.execEvalStmt()  // engine.go:772
              → populateSeries()   // fetches series from storage
              → evaluator.Eval()   // top-level evaluator entry
                  → evaluator.eval()  // recursive AST walk
```

**`engine.go:247`** — `Exec`:

```go
func (q *query) Exec(ctx context.Context) *Result {
    res, warnings, err := q.ng.exec(ctx, q)
    return &Result{Err: err, Value: res, Warnings: warnings}
}
```

**`engine.go:772`** — `execEvalStmt` creates the `evaluator` for both instant
and range evaluations. For instant queries, `start == end` with `interval=1`;
for range queries, `start` through `end` stepping by `interval`.

---

### 4. The Evaluator

**`engine.go:1134`** — Core evaluator struct:

```go
type evaluator struct {
    startTimestamp int64   // ms
    endTimestamp   int64   // ms
    interval       int64   // ms
    maxSamples     int
    currentSamples int
    lookbackDelta  time.Duration
    querier        storage.Querier
    // ...
}
```

**`engine.go:1189`** — Public entry point (handles panic recovery):

```go
func (ev *evaluator) Eval(ctx context.Context, expr parser.Expr) (v parser.Value, ws annotations.Annotations, err error)
```

**`engine.go:1904`** — The recursive core — `eval()` dispatches by AST node
type:

```go
func (ev *evaluator) eval(ctx context.Context, expr parser.Expr) (parser.Value, annotations.Annotations) {
    switch e := expr.(type) {
    case *parser.AggregateExpr:  // sum, avg, count, topk, ...
    case *parser.Call:           // function calls
    case *parser.BinaryExpr:     // arithmetic, comparisons, set ops
    case *parser.MatrixSelector: // range vectors
    case *parser.SubqueryExpr:   // subqueries — spins up a child evaluator
    case *parser.StepInvariantExpr: // evaluated once, results duplicated across steps
    case *parser.ParenExpr:      // transparent pass-through
    case *parser.UnaryExpr:      // negate floats/histograms
    case *parser.VectorSelector: // instant vector lookups
    case *parser.NumberLiteral, *parser.StringLiteral:
    }
}
```

**`rangeEval`** (`engine.go:1384`) is the workhorse for multi-step evaluation.
It evaluates all argument expressions first, then iterates across all time steps
calling a `funcCall` closure:

```go
func (ev *evaluator) rangeEval(
    ctx context.Context,
    matching *parser.VectorMatching,
    funcCall func([]Vector, Matrix, [][]EvalSeriesHelper, *EvalNodeHelper) (Vector, annotations.Annotations),
    exprs ...parser.Expr,
) (Matrix, annotations.Annotations)
```

**`EvalNodeHelper`** (`engine.go:1210`) is a per-node scratch space passed into
each function call, containing caches and a pre-allocated output `Vector`:

```go
type EvalNodeHelper struct {
    Ts  int64
    Out Vector
    // histogram caches, binary match state, label builders ...
}
```

---

### 5. Function Registration and Calling

**`functions.go:59`** — The `FunctionCall` type signature:

```go
type FunctionCall func(
    vectorVals []Vector,       // evaluated instant vector args (one per step)
    matrixVals Matrix,         // range vector arg (if any)
    args parser.Expressions,   // raw AST args (for string/selector access)
    enh *EvalNodeHelper,       // scratch space and output vector
) (Vector, annotations.Annotations)
```

**`functions.go:2152`** — All built-in functions are registered in a global map:

```go
var FunctionCalls = map[string]FunctionCall{
    "abs":                funcAbs,
    "rate":               funcRate,
    "histogram_quantile": funcHistogramQuantile,
    "label_replace":      funcLabelReplace,
    // ~80 more entries ...
}
```

**`engine.go:1956`** — Dispatch inside `eval()` for `*parser.Call`:

```go
case *parser.Call:
    call := FunctionCalls[e.Func.Name]
    // Special cases: timestamp(), label_replace(), label_join(), info()
    // Matrix arg? → iterate per-series directly
    // No matrix arg? → ev.rangeEval(..., func(...) { call(v, nil, e.Args, enh) }, e.Args...)
```

Functions with a **range vector** (matrix) argument (`rate`, `delta`,
`avg_over_time`, etc.) receive the raw sample window; functions with only
**instant vector** args (`abs`, `ceil`, `exp`, etc.) are called per-step through
`rangeEval`.

Example leaf function (`functions.go:1350`):

```go
func funcExp(vectorVals []Vector, _ Matrix, _ parser.Expressions, enh *EvalNodeHelper) (Vector, annotations.Annotations) {
    return simpleFloatFunc(vectorVals, enh, math.Exp), nil
}
```

---

### 6. Alert and Recording Rules

Rules are _not_ executed through the `Engine.exec` path. Instead they use a
**`QueryFunc`** callback:

```go
// rules/manager.go (inferred from usage)
type QueryFunc func(ctx context.Context, q string, t time.Time) (promql.Vector, error)
```

This function builds and executes an instant query using the engine, but
delivers only the `Vector` result back to the rule evaluator.

**`recording.go:84`** — `RecordingRule.Eval`:

```go
func (rule *RecordingRule) Eval(ctx context.Context, queryOffset time.Duration, ts time.Time, query QueryFunc, _ *url.URL, limit int) (promql.Vector, error) {
    ctx = NewOriginContext(ctx, NewRuleDetail(rule))
    vector, err := query(ctx, rule.vector.String(), ts.Add(-queryOffset))
    // override metric name + labels on each sample
    // check for duplicate labelsets
    return vector, nil
}
```

**`alerting.go:380`** — `AlertingRule.Eval`:

```go
func (r *AlertingRule) Eval(ctx context.Context, queryOffset time.Duration, ts time.Time, query QueryFunc, externalURL *url.URL, limit int) (promql.Vector, error) {
    ctx = NewOriginContext(ctx, NewRuleDetail(r))
    res, err := query(ctx, r.vector.String(), ts.Add(-queryOffset))
    // for each sample in res:
    //   - expand label/annotation templates
    //   - create/update Alert{State: Pending → Firing} keyed by label fingerprint
    //   - track active alerts in r.active map
    return vec, nil
}
```

**Rule scheduling** — A `Group` runs a periodic ticker loop (not shown in
indexed chunks, but architecturally each `Group` owns an `interval` and a
goroutine that calls `evalRules()` on each tick). Each tick calls
`rule.Eval(...)` for every rule in the group sequentially, passing the shared
`QueryFunc`.

---

### Summary: Key Call Chain

```
ticker → Group.evalRules()
           → AlertingRule.Eval(ts, queryFunc)  or  RecordingRule.Eval(ts, queryFunc)
               → queryFunc("rate(http_requests[5m])", ts)
                   → Engine.NewInstantQuery(...)
                   → query.Exec(ctx)
                       → Engine.exec → execEvalStmt
                           → evaluator.Eval(expr)
                               → eval(*Call)
                                   → FunctionCalls["rate"](matrix, enh)
                               → eval(*BinaryExpr) → rangeEval(...)
                               → eval(*AggregateExpr) → rangeEvalAgg(...)
                   → returns promql.Vector
```

### Answer: `sonnet` / `mcp-full`

Here's a comprehensive walkthrough of PromQL query evaluation in this codebase
(Prometheus):

---

## 1. Key Interfaces and Types

**`ast.go` — The AST**

```go
// Expr is the interface all expression nodes implement
type Expr interface {
    Node
    Type() ValueType
    PromQLExpr()  // marker method to prevent accidental implementations
}
```

AST node types (each implements `Expr`): | Node | Description | |---|---| |
`*AggregateExpr` | `sum`, `avg`, `topk`, etc. | | `*Call` | Function call, e.g.
`rate(...)` | | `*BinaryExpr` | `+`, `-`, `and`, `or`, etc. | |
`*VectorSelector` | `http_requests_total{job="api"}` | | `*MatrixSelector` |
`http_requests_total[5m]` | | `*SubqueryExpr` | `expr[range:step]` | |
`*UnaryExpr` | Unary `-` | | `*NumberLiteral` | Scalar constant | |
`*StepInvariantExpr` | Wraps exprs whose value doesn't change per step |

---

## 2. The Engine and Query Lifecycle

**`engine.go`**

```go
// query implements the Query interface
type query struct {
    queryable storage.Queryable
    q         string
    stmt      parser.Statement  // parsed AST
    stats     *stats.QueryTimers
    ng        *Engine
    cancel    func()
}

// Entry point
func (q *query) Exec(ctx context.Context) *Result {
    res, warnings, err := q.ng.exec(ctx, q)
    return &Result{Err: err, Value: res, Warnings: warnings}
}
```

`exec` → `execEvalStmt` → creates an `evaluator` → calls `evaluator.Eval`:

```go
// execEvalStmt — engine.go:772
func (ng *Engine) execEvalStmt(ctx context.Context, query *query, s *parser.EvalStmt) (parser.Value, annotations.Annotations, error) {
    // 1. Open a storage Querier for the time range
    querier, _ := query.queryable.Querier(mint, maxt)
    // 2. Pre-populate series on all selectors (storage lookup)
    ng.populateSeries(ctxPrepare, querier, s)
    // 3. Create evaluator and run
    evaluator := &evaluator{
        startTimestamp: timeMilliseconds(s.Start),
        endTimestamp:   timeMilliseconds(s.End),
        interval:       durationMilliseconds(s.Interval),
        querier:        querier,
        ...
    }
    val, warnings, err := evaluator.Eval(ctxInnerEval, s.Expr)
}
```

---

## 3. The Evaluator — AST Node Dispatch

**`engine.go:1134`** — the evaluator struct:

```go
type evaluator struct {
    startTimestamp int64  // ms
    endTimestamp   int64  // ms
    interval       int64  // ms (step for range queries)
    maxSamples     int
    currentSamples int
    lookbackDelta  time.Duration
    querier        storage.Querier
    ...
}
```

**`engine.go:1189`** — public entry point:

```go
func (ev *evaluator) Eval(ctx context.Context, expr parser.Expr) (v parser.Value, ws annotations.Annotations, err error) {
    defer ev.recover(expr, &ws, &err)
    v, ws = ev.eval(ctx, expr)
    return v, ws, nil
}
```

**`engine.go:1904`** — the core recursive `eval` method dispatches on node type:

```go
func (ev *evaluator) eval(ctx context.Context, expr parser.Expr) (parser.Value, annotations.Annotations) {
    switch e := expr.(type) {

    case *parser.AggregateExpr:
        // Recursively eval child, then aggregate over label groups
        val, ws := ev.eval(ctx, e.Expr)
        result, ws := ev.rangeEvalAgg(ctx, e, sortedGrouping, inputMatrix, fp)

    case *parser.Call:
        // Look up implementation, handle matrix vs instant args
        call := FunctionCalls[e.Func.Name]
        // If no matrix arg: evaluate args, call per step via rangeEval
        // If matrix arg: iterate series with sliding window, call per series
        return ev.rangeEval(ctx, nil, func(v []Vector, ...) (Vector, annotations.Annotations) {
            return call(v, nil, e.Args, enh)
        }, e.Args...)

    case *parser.BinaryExpr:
        // Dispatch by LHS/RHS types (scalar+scalar, vector+vector, etc.)
        return ev.rangeEval(ctx, e.VectorMatching, func(v []Vector, ...) (Vector, annotations.Annotations) {
            return ev.VectorBinop(e.Op, v[0], v[1], ...)
        }, e.LHS, e.RHS)

    case *parser.VectorSelector:
        // Fetch pre-populated series with lookback delta
        mat := ev.evalSeries(ctx, e.Series, e.Offset, false)
        return mat, ws

    case *parser.MatrixSelector:
        return ev.matrixSelector(ctx, e)

    case *parser.SubqueryExpr:
        // Create a nested evaluator with its own step/range, eval recursively
        newEv := &evaluator{interval: durationMilliseconds(e.Step), ...}
        return newEv.eval(ctx, e.Expr)

    case *parser.UnaryExpr:
        val, ws := ev.eval(ctx, e.Expr)
        // negate all float/histogram values

    case *parser.ParenExpr:
        return ev.eval(ctx, e.Expr)  // transparent

    case *parser.NumberLiteral:
        return ev.rangeEval(ctx, nil, func(...) (Vector, ...) {
            return append(enh.Out, Sample{F: e.Val}), nil
        })

    case *parser.StepInvariantExpr:
        // Evaluate only at startTimestamp, replicate across all steps
    }
}
```

**`rangeEval`** (`engine.go:1384`) is the step-loop workhorse — it evaluates
sub-expressions into `Matrix` form, then calls the provided `funcCall` closure
for each timestamp step, collecting results into output series.

---

## 4. Function Registration and Calling

**`functions.go:39`** — the function signature:

```go
// All built-in functions share this type
type FunctionCall func(
    vectorVals []Vector,        // evaluated instant vector args
    matrixVals Matrix,          // evaluated range vector arg (if any)
    args       parser.Expressions, // original AST args (for string literals, etc.)
    enh        *EvalNodeHelper,    // pre-allocated output vector + helpers
) (Vector, annotations.Annotations)
```

**`functions.go:2152`** — the global registry (map from name → implementation):

```go
var FunctionCalls = map[string]FunctionCall{
    "abs":                funcAbs,
    "rate":               funcRate,
    "histogram_quantile": funcHistogramQuantile,
    "label_replace":      funcLabelReplace,
    // ~80+ entries ...
}
```

At call time (`engine.go:1960`):

```go
case *parser.Call:
    call := FunctionCalls[e.Func.Name]  // direct map lookup
```

Special cases (`label_replace`, `label_join`, `info`) bypass `rangeEval`
entirely and get their own evaluator methods because they operate on the metric
structure, not sample values.

---

## 5. Alert and Recording Rules

Both implement the `Rule` interface and are evaluated by a **rule group** on a
ticker loop.

**`QueryFunc`** is the bridge between rules and the engine:

```go
type QueryFunc func(ctx context.Context, q string, t time.Time) (promql.Vector, error)
```

**Recording rule** (`recording.go:84`):

```go
func (rule *RecordingRule) Eval(ctx context.Context, queryOffset time.Duration,
    ts time.Time, query QueryFunc, _ *url.URL, limit int) (promql.Vector, error) {

    // 1. Execute the PromQL expression via QueryFunc
    vector, err := query(ctx, rule.vector.String(), ts.Add(-queryOffset))

    // 2. Override metric name and labels on each result sample
    for i := range vector {
        lb.Set(labels.MetricName, rule.name)
        rule.labels.Range(func(l labels.Label) { lb.Set(l.Name, l.Value) })
        vector[i].Metric = lb.Labels()
    }
    return vector, nil
}
```

**Alerting rule** (`alerting.go:380`):

```go
func (r *AlertingRule) Eval(ctx context.Context, queryOffset time.Duration,
    ts time.Time, query QueryFunc, externalURL *url.URL, limit int) (promql.Vector, error) {

    // 1. Execute PromQL via QueryFunc
    res, err := query(ctx, r.vector.String(), ts.Add(-queryOffset))

    // 2. For each result sample → create/update a pending Alert
    for _, smpl := range res {
        // Template-expand label/annotation values
        // Transition: inactive → Pending → Firing based on holdDuration
    }

    // 3. Expire alerts no longer in result set
    // 4. Return ALERT_FOR_STATE series for active alerts
}
```

**Alert state machine:**

```
inactive ──(expr fires)──▶ Pending ──(holdDuration elapsed)──▶ Firing
Firing   ──(expr stops)──▶ (keepFiringFor countdown) ──▶ inactive
```

The **rule group** (not shown in detail here) runs a ticker at the configured
`interval`, calls `rule.Eval(...)` on each rule in sequence, and for recording
rules writes the resulting vector back to the TSDB.

---

## Summary Flow

```
query.Exec()
  └─ Engine.exec()
       └─ Engine.execEvalStmt()
            ├─ populateSeries()       ← storage lookup, attach Series to selectors
            └─ evaluator.Eval(expr)
                 └─ evaluator.eval()  ← recursive AST dispatch
                      ├─ AggregateExpr → rangeEvalAgg()
                      ├─ Call         → FunctionCalls[name](…) via rangeEval()
                      ├─ BinaryExpr   → VectorBinop() via rangeEval()
                      ├─ VectorSelector → evalSeries() ← storage iterator
                      └─ MatrixSelector → matrixSelector() ← sliding window

Rule ticker (every interval)
  └─ rule.Eval(ts, QueryFunc, …)
       └─ QueryFunc(expr.String(), ts)  ← re-enters Engine
            └─ (same path as above)
```

### Answer: `opus` / `baseline`

## How PromQL Query Evaluation Works

### 1. The Evaluation Engine

The engine lives in `engine.go` (package `promql`). The key interfaces are:

**`QueryEngine` interface** (engine.go:125):

```go
type QueryEngine interface {
    NewInstantQuery(ctx context.Context, q storage.Queryable, opts QueryOpts,
        qs string, ts time.Time) (Query, error)
    NewRangeQuery(ctx context.Context, q storage.Queryable, opts QueryOpts,
        qs string, start, end time.Time, interval time.Duration) (Query, error)
}
```

**`Query` interface** (engine.go:143):

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

The concrete `Engine` struct (engine.go:345) holds configuration like `timeout`,
`maxSamplesPerQuery`, `lookbackDelta`, an `activeQueryTracker`, and a
`parser.Parser`.

**Execution flow:**

1. **Create query** — `NewInstantQuery` or `NewRangeQuery` (engine.go:531/552)
   parses the PromQL string into an AST `EvalStmt`, wraps it in a `*query`.
2. **Execute** — `query.Exec(ctx)` (engine.go:248) calls `ng.exec(ctx, q)`.
3. **`ng.exec`** (engine.go:673) — applies timeout, registers with
   `activeQueryTracker`, dispatches to `ng.execEvalStmt(ctx, q, s)` for
   `*parser.EvalStmt`.
4. **`ng.execEvalStmt`** (engine.go:773) — calls `ng.populateSeries(...)` to
   walk the AST and attach `storage.SeriesSet` to every `VectorSelector`, then
   constructs an internal `evaluator` and calls `evaluator.Eval(ctx, s.Expr)`.

The `evaluator` struct (engine.go:1138) is the per-query worker:

```go
type evaluator struct {
    startTimestamp, endTimestamp, interval int64  // milliseconds
    maxSamples, currentSamples            int
    lookbackDelta                         time.Duration
    querier                               storage.Querier
    // ...
}
```

---

### 2. How Functions Are Registered and Called

**`FunctionCall` type** (functions.go:60):

```go
type FunctionCall func(
    vectorVals []Vector,
    matrixVals Matrix,
    args parser.Expressions,
    enh *EvalNodeHelper,
) (Vector, annotations.Annotations)
```

**Registry** (functions.go:2153) — a package-level map:

```go
var FunctionCalls = map[string]FunctionCall{
    "abs":      funcAbs,
    "rate":     funcRate,
    "increase": funcIncrease,
    "delta":    funcDelta,
    "irate":    funcIrate,
    // ~70 entries total
    "label_replace": nil,  // handled specially in eval()
    "label_join":    nil,  // handled specially
}
```

When a `*parser.Call` node is evaluated (engine.go:1962), the engine looks up
`FunctionCalls[e.Func.Name]`. Three dispatch paths:

1. **Special functions** (`label_replace`, `label_join`, `info`) — branch to
   dedicated methods.
2. **No matrix argument** — uses `ev.rangeEval(...)` which iterates timestamps
   and calls the `FunctionCall` at each step.
3. **Matrix argument present** — iterates over each series in the matrix
   selector, uses a `storage.Buffer` window, and calls the function per series
   per step.

---

### 3. How AST Nodes Are Evaluated

The core recursive dispatch is `evaluator.eval` (engine.go:1905):

```go
func (ev *evaluator) eval(ctx context.Context, expr parser.Expr) (parser.Value, annotations.Annotations)
```

It switches on `expr.(type)`:

| AST Node                    | Behavior                                                                                                                               |
| --------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| `*parser.AggregateExpr`     | Sorts grouping labels, evaluates sub-expression, calls `ev.rangeEvalAgg(...)`                                                          |
| `*parser.Call`              | Looks up function in `FunctionCalls`, evaluates args, dispatches                                                                       |
| `*parser.BinaryExpr`        | Dispatches on operand types (scalar/scalar, vector/vector, vector/scalar) using `VectorBinop`, `VectorAnd`, `VectorOr`, `VectorUnless` |
| `*parser.VectorSelector`    | Expands series set, calls `ev.evalSeries` per series                                                                                   |
| `*parser.MatrixSelector`    | Calls `ev.matrixSelector(ctx, e)` (instant eval only)                                                                                  |
| `*parser.SubqueryExpr`      | Creates a child `evaluator` with adjusted timestamps and recursively evaluates                                                         |
| `*parser.StepInvariantExpr` | Evaluates once at `startTimestamp`, duplicates result across all steps                                                                 |
| `*parser.ParenExpr`         | Pass-through: `return ev.eval(ctx, e.Expr)`                                                                                            |
| `*parser.UnaryExpr`         | Evaluates sub-expr, negates float values                                                                                               |
| `*parser.NumberLiteral`     | Returns constant via `rangeEval`                                                                                                       |
| `*parser.StringLiteral`     | Returns `String` value directly                                                                                                        |

**`rangeEval`** (engine.go:1390) is the step-iteration backbone shared by
functions, binary ops, and aggregations:

```go
func (ev *evaluator) rangeEval(ctx context.Context, matching *parser.VectorMatching,
    funcCall func([]Vector, Matrix, [][]EvalSeriesHelper, *EvalNodeHelper) (Vector, annotations.Annotations),
    exprs ...parser.Expr) (Matrix, annotations.Annotations)
```

It pre-evaluates all sub-expressions into matrices, then iterates
`for ts := startTimestamp; ts <= endTimestamp; ts += interval`, calling
`funcCall` at each step and accumulating output into a `Matrix`.

**`EvalNodeHelper`** (engine.go:1211) is a per-step helper reused across steps
to minimize allocations:

```go
type EvalNodeHelper struct {
    Ts  int64   // current evaluation timestamp
    Out Vector  // pre-allocated output vector
    // histogram caches, label builder, binary-op matching state...
}
```

---

### 4. How Alert and Recording Rules Trigger Evaluations

Both rule types share the same pattern: they call a `QueryFunc` which is a
closure wrapping `Engine.NewInstantQuery(...).Exec(ctx)`.

**`RecordingRule.Eval`** (recording.go:85):

```go
func (rule *RecordingRule) Eval(ctx context.Context, queryOffset time.Duration,
    ts time.Time, query QueryFunc, _ *url.URL, limit int) (promql.Vector, error)
```

1. Calls `query(ctx, rule.vector.String(), ts.Add(-queryOffset))` — instant
   query at the evaluation timestamp.
2. Overrides metric `__name__` with `rule.name`, applies `rule.labels`.
3. Checks for duplicate label sets and enforces `limit`.

**`AlertingRule.Eval`** (alerting.go:382):

```go
func (r *AlertingRule) Eval(ctx context.Context, queryOffset time.Duration,
    ts time.Time, query QueryFunc, externalURL *url.URL, limit int) (promql.Vector, error)
```

1. Same `query(...)` call to evaluate the alerting expression.
2. For each result sample: expands label templates, creates/updates `Alert`
   entries in `r.active` (keyed by fingerprint).
3. State machine transitions:
   - `StatePending → StateFiring` when `holdDuration` has elapsed
   - Handles `keepFiringFor` to keep alerts active after resolution
   - Cleans up resolved alerts after 15-minute retention
4. Emits two synthetic series per active alert: `ALERTS{alertname=...}` and
   `ALERTS_FOR_STATE{alertname=...}`.

**Alert states** (alerting.go:54):

```go
const (
    StateUnknown  AlertState = iota
    StateInactive
    StatePending   // firing but holdDuration not yet elapsed
    StateFiring    // fully active
)
```

---

### Summary of the Data Flow

```
PromQL string
    │  ng.NewInstantQuery / ng.NewRangeQuery
    ▼
parser.ParseExpr → AST (Expr tree)
    │  ng.exec → ng.execEvalStmt
    ▼
ng.populateSeries (attaches storage.Series to VectorSelectors)
    │
    ▼
evaluator.Eval(expr)
    │  recursive eval() switch on node type
    ▼
rangeEval (step iteration) → FunctionCalls[name](...) or binop/agg logic
    │
    ▼
Matrix / Vector / Scalar result
    │
    ▼
Rules call QueryFunc → Engine.NewInstantQuery().Exec()
    ├── RecordingRule: renames metric, applies labels
    └── AlertingRule: manages Alert state machine, emits ALERTS series
```

### Answer: `opus` / `mcp-only`

Now I have a comprehensive picture. Here's how PromQL query evaluation works in
this codebase:

---

## PromQL Query Evaluation

### 1. The Engine — Entry Point

The `Engine` is the central orchestrator. It implements `QueryEngine`
(`engine.go:124-128`):

```go
type QueryEngine interface {
    NewInstantQuery(ctx context.Context, q storage.Queryable, opts QueryOpts, qs string, ts time.Time) (Query, error)
    NewRangeQuery(ctx context.Context, q storage.Queryable, opts QueryOpts, qs string, start, end time.Time, interval time.Duration) (Query, error)
}
```

Both methods parse the PromQL string and create a `query` struct
(`engine.go:189-208`):

```go
type query struct {
    queryable   storage.Queryable
    q           string              // original query string
    stmt        parser.Statement    // parsed AST
    stats       *stats.QueryTimers
    sampleStats *stats.QuerySamples
    matrix      Matrix
    cancel      func()
    ng          *Engine
}
```

The `Query` interface (`engine.go:141-156`) exposes `Exec()`, `Close()`,
`Cancel()`, `Statement()`, and `Stats()`.

Configuration lives in `EngineOpts` (`engine.go:298-341`) — max samples,
timeout, lookback delta, feature flags, etc.

### 2. Query Execution Flow

When `query.Exec()` is called (`engine.go:247-256`), it delegates to
`Engine.exec()`:

```
query.Exec() → Engine.exec() → Engine.execEvalStmt() → evaluator.Eval() → evaluator.eval()
```

**`Engine.exec()`** (`engine.go:669-726`): Sets up context with timeout, queues
the query for concurrency control, logs the query, and delegates to
`execEvalStmt`.

**`Engine.execEvalStmt()`** (`engine.go:772-880`): This is where the real work
begins:

1. Determines the time range (`mint`, `maxt`) from the `EvalStmt`
2. Creates a `Querier` from the storage layer
3. Populates series data via `ng.populateSeries()`
4. Creates an `evaluator` struct
5. For **instant queries** (Start == End): runs as a single-step range
   evaluation
6. For **range queries**: iterates over the `[Start, End]` interval at
   `Interval` steps
7. Converts the raw `Matrix` result to the appropriate return type (`Vector`,
   `Scalar`, or `Matrix`)

### 3. The Evaluator — Recursive AST Walking

The `evaluator` struct (`engine.go:1134-1152`):

```go
type evaluator struct {
    startTimestamp           int64
    endTimestamp             int64
    interval                 int64
    maxSamples               int
    currentSamples           int
    logger                   *slog.Logger
    lookbackDelta            time.Duration
    samplesStats             *stats.QuerySamples
    noStepSubqueryIntervalFn func(rangeMillis int64) int64
    enableDelayedNameRemoval bool
    enableTypeAndUnitLabels  bool
    querier                  storage.Querier
}
```

**`evaluator.Eval()`** (`engine.go:1189-1197`) is the public entry point — it
calls `eval()` and handles panic-based error recovery.

**`evaluator.eval()`** (`engine.go:1904+`) is the core recursive method. It does
a type switch over the AST node types:

| AST Node                    | Evaluation Strategy                                                                                  |
| --------------------------- | ---------------------------------------------------------------------------------------------------- |
| `*parser.AggregateExpr`     | Recursively eval inner expr, then `rangeEvalAgg()` for aggregation (sum, avg, topk, etc.)            |
| `*parser.Call`              | Look up function in `FunctionCalls` map, handle matrix vs instant args, call via `rangeEval()`       |
| `*parser.BinaryExpr`        | Eval both sides, dispatch to `VectorAnd`/`VectorOr`/`VectorUnless`/`VectorBinop`/`VectorscalarBinop` |
| `*parser.VectorSelector`    | Fetch series from storage via `evalSeries()`                                                         |
| `*parser.MatrixSelector`    | Handled within `Call` processing for range vector functions                                          |
| `*parser.NumberLiteral`     | Wrap scalar value in a `Vector` via `rangeEval()`                                                    |
| `*parser.StringLiteral`     | Return `String` value directly                                                                       |
| `*parser.ParenExpr`         | Recurse into inner expression                                                                        |
| `*parser.UnaryExpr`         | Eval inner, negate floats/histograms if SUB                                                          |
| `*parser.SubqueryExpr`      | Replaced with a `MatrixSelector` after evaluation                                                    |
| `*parser.StepInvariantExpr` | Eval at a single timestamp, expand across steps                                                      |

**`rangeEval()`** (`engine.go:1384-1430`) is the workhorse for step-by-step
evaluation. It:

1. Evaluates all sub-expressions into `Matrix` results
2. For each timestamp step, extracts the current `Vector` from each matrix
3. Calls the provided `funcCall` callback with those vectors
4. Accumulates results back into an output `Matrix`

### 4. Function Registration and Calling

Functions are registered as a global map (`functions.go:2152+`):

```go
var FunctionCalls = map[string]FunctionCall{
    "abs":               funcAbs,
    "rate":              funcRate,
    "histogram_quantile": funcHistogramQuantile,
    // ~80+ functions...
}
```

The `FunctionCall` type (`functions.go:39-60`):

```go
type FunctionCall func(
    vectorVals []Vector,
    matrixVals Matrix,
    args parser.Expressions,
    enh *EvalNodeHelper,
) (Vector, annotations.Annotations)
```

When `eval()` encounters a `*parser.Call` node (`engine.go:1956-2077`):

1. Looks up the function: `call := FunctionCalls[e.Func.Name]`
2. Special-cases `timestamp`, `label_replace`, `label_join`, `info`
3. Checks if any argument is a `MatrixSelector` (range vector functions like
   `rate`, `increase`)
4. **Without matrix arg**: calls `rangeEval()` with the function as the step
   callback
5. **With matrix arg**: iterates over each series, builds a sliding window of
   samples per step, and calls the function per-series per-step

### 5. The AST — Node Types

The `Expr` interface (`ast.go:76-85`):

```go
type Expr interface {
    Node
    Type() ValueType
    PromQLExpr()  // marker method — prevents accidental implementation
}
```

The `EvalStmt` wraps an expression with evaluation boundaries (`ast.go:60-72`):

```go
type EvalStmt struct {
    Expr          Expr
    Start, End    time.Time
    Interval      time.Duration
    LookbackDelta time.Duration
}
```

Key concrete node types and their `Type()` returns:

- `AggregateExpr` → `ValueTypeVector`
- `Call` → depends on function definition
- `BinaryExpr` → depends on operand types
- `VectorSelector` → `ValueTypeVector`
- `MatrixSelector` → `ValueTypeMatrix`
- `NumberLiteral` → `ValueTypeScalar`
- `SubqueryExpr` → `ValueTypeMatrix`

Tree traversal uses `ChildrenIter()` (`ast.go:401-450`) which yields child nodes
via a switch over concrete types.

### 6. Alert and Recording Rules

Both rule types delegate to the PromQL engine via a `QueryFunc` callback — they
do **not** use the engine directly. The engine comment confirms this: _"At this
point per query only one EvalStmt is evaluated. Alert and record statements are
not handled by the Engine."_

**`AlertingRule`** (`alerting.go:115-157`):

```go
type AlertingRule struct {
    name           string
    vector         parser.Expr        // the PromQL expression
    holdDuration   time.Duration      // "for" duration
    keepFiringFor  time.Duration
    labels         labels.Labels
    annotations    labels.Labels
    active         map[uint64]*Alert  // currently active alerts by fingerprint
    // ...
}
```

**`AlertingRule.Eval()`** (`alerting.go:380+`):

```go
func (r *AlertingRule) Eval(ctx context.Context, queryOffset time.Duration, ts time.Time,
    query QueryFunc, externalURL *url.URL, limit int) (promql.Vector, error)
```

1. Calls `query(ctx, r.vector.String(), ts.Add(-queryOffset))` — this executes
   the PromQL via the engine
2. For each result sample, expands template labels/annotations
3. Creates or updates `Alert` instances in the `active` map
4. Manages state transitions: Inactive → Pending → Firing (after `holdDuration`
   elapses)
5. Returns a vector of `ALERTS` and `ALERTS_FOR_STATE` samples

**`RecordingRule`** (`recording.go:37-54`):

```go
type RecordingRule struct {
    name   string
    vector parser.Expr
    labels labels.Labels
    // ...
}
```

**`RecordingRule.Eval()`** (`recording.go:84-122`):

```go
func (rule *RecordingRule) Eval(ctx context.Context, queryOffset time.Duration, ts time.Time,
    query QueryFunc, _ *url.URL, limit int) (promql.Vector, error)
```

1. Calls `query(ctx, rule.vector.String(), ts.Add(-queryOffset))`
2. Overrides `__name__` label with the recording rule name
3. Applies any additional labels
4. Checks for duplicate labelsets
5. Returns the relabeled vector (which gets written back to storage)

Both rule types are evaluated periodically by a rule `Group` (referenced in the
API layer at `api.go:1577+`), which iterates over its rules and calls `Eval()`
at each group interval.

### Answer: `opus` / `mcp-full`

Now I have enough to give a comprehensive answer.

---

## How PromQL Query Evaluation Works

The evaluation system has three layers: the **Engine** (entry point), the
**evaluator** (recursive AST walker), and the **rules system** (alert/recording
rules that trigger evaluations on a schedule).

---

### 1. The Engine — Entry Point

The `QueryEngine` interface (`engine.go:124-128`) defines two ways to create
queries:

```go
type QueryEngine interface {
    NewInstantQuery(ctx context.Context, q storage.Queryable, opts QueryOpts, qs string, ts time.Time) (Query, error)
    NewRangeQuery(ctx context.Context, q storage.Queryable, opts QueryOpts, qs string, start, end time.Time, interval time.Duration) (Query, error)
}
```

Both return a `Query` (`engine.go:141-156`):

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

The concrete `query` struct (`engine.go:189-208`) holds the parsed statement,
queryable storage, stats, and a back-pointer to the engine. Calling
`query.Exec()` (`engine.go:247-256`) delegates to `Engine.exec()`.

**Execution flow:**

1. `Engine.exec()` (`engine.go:669`) — sets up timeout/cancellation, manages the
   active query queue, logs the query, then calls `Engine.execEvalStmt()`.
2. `Engine.execEvalStmt()` (`engine.go:772`) — creates a TSDB `Querier` for the
   time range, populates series data from storage, then constructs an
   `evaluator` and calls `evaluator.Eval()`. Instant queries are treated as
   range queries with a single step (`Start == End, Interval == 0`).

---

### 2. The Evaluator — Recursive AST Walking

The `evaluator` struct (`engine.go:1134-1152`) holds the time range, step
interval, sample limits, and a storage querier:

```go
type evaluator struct {
    startTimestamp, endTimestamp int64
    interval                    int64
    maxSamples                  int
    currentSamples              int
    lookbackDelta               time.Duration
    // ...
}
```

`evaluator.Eval()` (`engine.go:1189-1197`) is the public entry point that calls
the internal `eval()` and recovers from panics (the evaluator uses panic-based
error propagation):

```go
func (ev *evaluator) Eval(ctx context.Context, expr parser.Expr) (v parser.Value, ws annotations.Annotations, err error) {
    defer ev.recover(expr, &ws, &err)
    v, ws = ev.eval(ctx, expr)
    // ...
}
```

The core `eval()` method (`engine.go:1904`) is a large `switch` over AST node
types:

| AST Node                    | Evaluation Strategy                                                                                                                                        |
| --------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `*parser.AggregateExpr`     | Recursively evaluates child expr, then applies aggregation (`sum`, `avg`, `topk`, etc.) via `rangeEvalAgg()`                                               |
| `*parser.Call`              | Looks up function in `FunctionCalls` map, handles matrix vs. vector args, delegates to `rangeEval()` with the function as a callback                       |
| `*parser.BinaryExpr`        | Evaluates LHS and RHS, then applies the binary operation (arithmetic, comparison, set ops) with optional vector matching                                   |
| `*parser.VectorSelector`    | Fetches time series from storage for the current time range                                                                                                |
| `*parser.MatrixSelector`    | Fetches range vectors (only valid for instant evaluation)                                                                                                  |
| `*parser.SubqueryExpr`      | Creates a new child `evaluator` with adjusted time range and recursively evaluates                                                                         |
| `*parser.StepInvariantExpr` | Evaluates the inner expression once at the start time, then copies the result across all steps (optimization for expressions with `@ start()` / `@ end()`) |
| `*parser.NumberLiteral`     | Returns a scalar value at each step                                                                                                                        |
| `*parser.StringLiteral`     | Returns a string value                                                                                                                                     |
| `*parser.UnaryExpr`         | Evaluates inner expr then negates                                                                                                                          |
| `*parser.ParenExpr`         | Evaluates inner expr (passthrough)                                                                                                                         |

The `rangeEval()` method (`engine.go:1384`) is the workhorse for stepping
through time: it evaluates child expressions into matrices, then iterates
step-by-step, extracting the vector at each timestamp and calling the provided
`funcCall` callback.

---

### 3. The AST — Expression Types

The `Expr` interface (`ast.go:76-85`) is the root of all expression nodes:

```go
type Expr interface {
    Node
    Type() ValueType
    PromQLExpr()  // marker method — prevents accidental interface satisfaction
}
```

The `EvalStmt` (`ast.go:60-72`) wraps an `Expr` with time boundaries:

```go
type EvalStmt struct {
    Expr          Expr
    Start, End    time.Time
    Interval      time.Duration
    LookbackDelta time.Duration
}
```

The `Call` node (`ast.go:123-129`) represents function calls:

```go
type Call struct {
    Func *Function     // metadata: name, arg types, return type
    Args Expressions   // child expressions
}
```

Each node type implements `PromQLExpr()` as a marker and `Type()` to return its
value type — e.g. `Call.Type()` returns `e.Func.ReturnType` (`ast.go:252`).

---

### 4. Function Registration and Calling

The `FunctionCall` type (`functions.go:60`):

```go
type FunctionCall func(vectorVals []Vector, matrixVals Matrix, args parser.Expressions, enh *EvalNodeHelper) (Vector, annotations.Annotations)
```

All built-in functions are registered in the `FunctionCalls` map
(`functions.go:2152+`):

```go
var FunctionCalls = map[string]FunctionCall{
    "abs":               funcAbs,
    "rate":              funcRate,
    "histogram_quantile": funcHistogramQuantile,
    "label_replace":     ...,
    // ~70+ functions total
}
```

When `eval()` hits a `*parser.Call` node (`engine.go:1956-2022`):

1. It looks up `call := FunctionCalls[e.Func.Name]`
2. Checks if the function takes a matrix argument (range vector)
3. Special-cases `timestamp`, `label_replace`, `label_join`, and `info` with
   dedicated evaluation paths
4. For normal functions, calls `rangeEval()` passing the looked-up function as
   the step callback:
   ```go
   return ev.rangeEval(ctx, nil, func(v []Vector, _ Matrix, ...) (Vector, annotations.Annotations) {
       return call(v, nil, e.Args, enh)
   }, e.Args...)
   ```
5. For matrix-arg functions (like `rate`, `increase`), it iterates over series
   and calls the function per-series with the buffered samples.

---

### 5. Alert and Recording Rules — Triggering Evaluations

Both rule types use a `QueryFunc` to execute PromQL queries against the engine.
They do **not** use the engine directly — the engine comment in `exec()`
explicitly says: _"Alert and record statements are not handled by the Engine."_

**`AlertingRule`** (`alerting.go:115-157`):

```go
type AlertingRule struct {
    name         string
    vector       parser.Expr       // the PromQL expression
    holdDuration time.Duration     // "for" duration
    keepFiringFor time.Duration
    labels, annotations labels.Labels
    active       map[uint64]*Alert // currently active alert instances
    // ...
}
```

`AlertingRule.Eval()` (`alerting.go:380`):

```go
func (r *AlertingRule) Eval(ctx context.Context, queryOffset time.Duration, ts time.Time,
    query QueryFunc, externalURL *url.URL, limit int) (promql.Vector, error)
```

1. Calls `query(ctx, r.vector.String(), ts.Add(-queryOffset))` — executes the
   PromQL expression via the engine
2. For each returned sample, creates or updates an `Alert` in the `active` map
3. Manages alert lifecycle: Pending → Firing (after `holdDuration`) → Resolved
4. Expands label/annotation templates using Go `text/template`
5. Returns a vector of `ALERTS` and `ALERTS_FOR_STATE` samples for TSDB storage

**`RecordingRule`** (`recording.go:37-54`):

```go
type RecordingRule struct {
    name   string
    vector parser.Expr
    labels labels.Labels
    // ...
}
```

`RecordingRule.Eval()` (`recording.go:84`):

```go
func (rule *RecordingRule) Eval(ctx context.Context, queryOffset time.Duration, ts time.Time,
    query QueryFunc, _ *url.URL, limit int) (promql.Vector, error)
```

1. Calls `query(ctx, rule.vector.String(), ts.Add(-queryOffset))`
2. Overrides `__name__` on each result sample to the recording rule's name
3. Applies additional labels
4. Checks for duplicate label sets
5. Returns the vector for TSDB ingestion

Both rule types are evaluated periodically by a `Group` which iterates over its
rules and calls `Eval()` at each evaluation interval, passing in a `QueryFunc`
that wraps `Engine.NewInstantQuery().Exec()`.

### Full Judge Analysis (Opus 4.6)

## Content Quality

**Ranking: opus/mcp-full > opus/mcp-only > sonnet/mcp-full > opus/baseline >
sonnet/mcp-only > sonnet/baseline**

**1. opus/mcp-full** — The most polished and well-structured answer. Correctly
covers all four requested areas (engine, functions, AST, rules) with accurate
file/line references (`engine.go:124-128`, `engine.go:1134-1152`,
`functions.go:2152+`, `alerting.go:380`, etc.). The execution flow is presented
clearly with the right level of detail — it explains _why_ things happen (e.g.,
panic-based error propagation, the `StepInvariantExpr` optimization for
`@ start()`/`@ end()`), not just _what_. Correctly notes that the engine
explicitly disclaims handling alert/recording rules. The `QueryEngine`
interface, `Query` interface, `EvalStmt`, and `FunctionCall` type are all
accurately reproduced. The answer benefits from having both semantic search and
traditional tools to cross-reference findings.

**2. opus/mcp-only** — Nearly as comprehensive as opus/mcp-full. Accurately
identifies all key interfaces (`QueryEngine`, `Query`, `Expr`), the `evaluator`
struct, the `FunctionCalls` registry, and both rule types with correct
signatures. The AST node dispatch table is thorough. One minor issue: the
`MatrixSelector` entry says "Handled within `Call` processing for range vector
functions" which is partially misleading — `matrixSelector` is also called
directly in `eval()` for instant evaluation contexts. Line references are
precise. The `EngineOpts` mention adds useful context not found in some other
answers. Slightly less polished prose than mcp-full but substantively
equivalent.

**3. sonnet/mcp-full** — Covers all four areas competently with good code
snippets. The `eval()` dispatch section is presented as inline Go code with
comments rather than a table, which makes it feel more grounded in the actual
source. Correctly shows the `rangeEval` signature and explains the matrix-arg
vs. non-matrix-arg function dispatch paths. The alert state machine diagram
(`inactive → Pending → Firing`) is a nice touch. File/line references are
present but slightly less precise than the opus answers (e.g., `functions.go:39`
for the FunctionCall type vs. the correct `functions.go:60`). The summary flow
diagram at the end is clean and accurate.

**4. opus/baseline** — Impressive given it had no MCP tools. Covers all four
areas with correct structural understanding. The function registration section
accurately notes that `label_replace` and `label_join` are `nil` in the map and
handled specially. The `rangeEval` signature is correctly reproduced. The rules
section correctly explains the `QueryFunc` closure pattern. However, some line
references may be approximate since they came from baseline knowledge rather
than actual file reads. The answer is slightly more compact than the
MCP-assisted versions but doesn't sacrifice accuracy — a testament to the
model's training data including Prometheus source.

**5. sonnet/mcp-only** — Thorough and well-organized with accurate content.
Covers all areas including the `QueryEngine` interface (which sonnet/baseline
missed). The AST node type table with line references (`ast.go:269-279`) is
helpful. However, the "Rule scheduling" section acknowledges uncertainty ("not
shown in indexed chunks, but architecturally...") which is honest but reveals
incomplete exploration. The `ChildrenIter` mention and the `EvalStmt` struct are
good additions. The summary call chain diagram is accurate and useful.

**6. sonnet/baseline** — The weakest of the six, though still competent. Covers
the core areas but misses the `QueryEngine` and `Query` interfaces entirely,
jumping straight to `Engine.exec()`. The `evaluator` struct snippet is
simplified with `// ...` elisions. The function registry section is accurate.
The rules section correctly identifies `QueryFunc` and the state transitions.
Line references are present but some feel approximate (e.g., `engine.go:345` for
the Engine struct). The data flow diagram at the end is a good summary but the
answer overall feels thinner than the others, particularly on AST node types and
the `rangeEval` mechanism.

## Efficiency Analysis

| Scenario        | Duration | Total Tokens (In+Cache+Out) | Cost  | Quality Rank |
| --------------- | -------- | --------------------------- | ----- | ------------ |
| sonnet/baseline | 133.3s   | 63,738                      | $1.13 | 6th          |
| sonnet/mcp-only | 85.0s    | 198,684                     | $1.09 | 5th          |
| sonnet/mcp-full | 75.3s    | 247,216                     | $0.94 | 3rd          |
| opus/baseline   | 225.7s   | 64,525                      | $6.14 | 4th          |
| opus/mcp-only   | 75.5s    | 121,883                     | $0.69 | 2nd          |
| opus/mcp-full   | 89.0s    | 321,620                     | $1.32 | 1st          |

**Most efficient: opus/mcp-only** — Best cost ($0.69), fastest runtime tied with
sonnet/mcp-full, and second-best quality. Opus with semantic search alone found
the right code quickly without the token overhead of traditional tools. This is
a remarkable result: the most expensive model produced the cheapest run because
MCP search eliminated wasteful exploration.

**Best quality-to-cost: opus/mcp-only** — At $0.69 it's the cheapest run across
all six scenarios while delivering the second-best answer. opus/mcp-full is
marginally better in quality but costs nearly 2x more ($1.32).

**Surprising findings:**

- **opus/baseline is the most expensive by far** ($6.14) — 9x the cost of
  opus/mcp-only for a worse answer. The 225.7s runtime suggests extensive
  context-window-based reasoning to compensate for lack of tools.
- **sonnet/baseline is slower than sonnet/mcp runs** despite using fewer tokens
  — the 133.3s duration vs. 75-85s suggests the baseline approach required more
  sequential reasoning turns.
- **Cache reads dramatically help sonnet** — sonnet/mcp-full had 84K cached
  tokens and was both fastest (75.3s) and cheapest ($0.94) among sonnet runs,
  while producing the best sonnet answer.
- **MCP-only opus used fewer input tokens than MCP-only sonnet** (117K vs 194K)
  — opus found relevant code more efficiently with semantic search, needing
  fewer search iterations.

**Recommendation:** For this type of deep codebase question, **opus/mcp-only**
is the clear winner on cost-efficiency. If absolute quality is paramount and
cost is secondary, opus/mcp-full edges ahead. The baseline approaches are
strictly dominated — they cost more (especially opus) and produce worse results
for this kind of cross-cutting architectural question.

---

## scrape-pipeline [very-hard]

**Question:** How does Prometheus metrics scraping and collection work? Explain
how the scrape manager coordinates scrapers, how metrics are parsed from the
text format, how counters and gauges are tracked internally, and how the
registry manages metric families. Show the key types and the data flow from
scrape to in-memory storage.

### Metrics

| Model      | Scenario | Duration | Input Tok | Cache Read | Cache Created | Output Tok | Cost (USD) |
| ---------- | -------- | -------- | --------- | ---------- | ------------- | ---------- | ---------- |
| **sonnet** | baseline | 120.8s   | 34214     | 28104      | 0             | 2657       | $0.82100   |
| **sonnet** | mcp-only | 66.4s    | 91257     | 0          | 0             | 3951       | $0.55506   |
| **sonnet** | mcp-full | 62.6s    | 112663    | 56208      | 0             | 3663       | $0.68299   |
| **opus**   | baseline | 100.3s   | 32629     | 28230      | 0             | 2103       | $0.65464   |
| **opus**   | mcp-only | 70.8s    | 100358    | 0          | 0             | 3883       | $0.59886   |
| **opus**   | mcp-full | 259.8s   | 35841     | 28230      | 0             | 2462       | $4.39919   |

### Answer: `sonnet` / `baseline`

Here's a detailed walkthrough of how Prometheus metrics scraping and collection
works in this codebase:

---

## Data Flow Overview

```
Discovery Updates
    → Manager.Run() receives target groups
    → Manager.reload() creates/updates scrape pools
    → scrapePool.Sync() creates/updates a scrapeLoop per target
    → scrapeLoop.run() ticks at configured interval
    → scrapeAndReport(): HTTP GET → parse → append to TSDB
```

---

## 1. Scrape Manager Coordination

**`manager.go:135`** — `Manager` is the top-level coordinator:

```go
type Manager struct {
    scrapeConfigs map[string]*config.ScrapeConfig
    scrapePools   map[string]*scrapePool
    targetSets    map[string][]*targetgroup.Group
    triggerReload chan struct{}
    mtxScrape     sync.Mutex
    buffers       *pool.Pool
    // ...
}
```

**Control loop:**

- `Manager.Run()` (line 160) receives target group updates, stores them, signals
  `triggerReload`
- `Manager.reloader()` (line 186) coalesces signals with a 5-second default
  interval, calls `reload()`
- `Manager.reload()` (line 211) creates new `scrapePool`s for new job names;
  calls `sp.Sync(groups)` concurrently for existing ones
- `Manager.ApplyConfig()` (line 279) atomically swaps `scrapeConfigs`, reloads
  all pools concurrently via a buffered channel

---

## 2. Scrape Pool and Loop

**`scrape.go:84`** — `scrapePool` owns all scrapers for one job:

```go
type scrapePool struct {
    loops         map[uint64]loop    // keyed by target hash
    activeTargets map[uint64]*Target
    config        *config.ScrapeConfig
    client        *http.Client
    symbolTable   *labels.SymbolTable
    // ...
}
```

**`scrape.go:822`** — `scrapeLoop` handles a single target:

```go
type scrapeLoop struct {
    scraper      scraper           // HTTP layer
    cache        *scrapeCache      // series ref cache
    appendable   storage.Appendable
    interval     time.Duration
    timeout      time.Duration
    sampleMutator labelsMutator    // relabeling
    // ...
}
```

`scrapeLoop.run()` ticks at `interval + offset`, calls `scrapeAndReport()` each
tick.

---

## 3. Text Format Parsing

**`textparse_interface.go:29`** — `Parser` interface abstracts all exposition
formats:

```go
type Parser interface {
    Next() (Entry, error)        // advances iterator
    Series() ([]byte, *int64, float64)                              // regular sample
    Histogram() ([]byte, *int64, *histogram.Histogram, *histogram.FloatHistogram)
    Labels(l *labels.Labels)     // extracts labels into l
    Help() ([]byte, []byte)
    Type() ([]byte, model.MetricType)
    Unit() ([]byte, []byte)
    Exemplar(l *exemplar.Exemplar) bool
}
```

Entry types: `EntryType=0`, `EntryHelp=1`, `EntrySeries=2`, `EntryComment=3`,
`EntryUnit=4`, `EntryHistogram=5`

`textparse.New()` selects the parser from the HTTP `Content-Type`:

- `application/openmetrics-text` → OpenMetrics parser
- `application/vnd.google.protobuf` → Protobuf parser
- `text/plain` → classic Prometheus text parser

---

## 4. Scrape-to-Storage Pipeline

**`scrape.go:1313`** — `scrapeAndReport()` orchestrates each scrape cycle:

```
1. scraper.scrape()        → HTTP GET with timeout context
2. readResponse()          → decompress (gzip), enforce body limit
3. textparse.New()         → pick parser from Content-Type
4. for p.Next():
   ├─ EntryType/Help/Unit  → cache.setType/setHelp/setUnit (metadata only)
   ├─ EntrySeries:
   │   met, ts, val = p.Series()
   │   p.Labels(&lset)         → extract label set
   │   sampleMutator(lset)     → apply relabeling
   │   app.Append(ref, lset, t, val)   → write to TSDB, get SeriesRef
   │   cache entry: ref → lset  → next scrape reuses ref without re-hashing
   └─ EntryHistogram:
       app.AppendHistogram(ref, lset, t, h, fh)
5. app.Commit()
```

---

## 5. Counter and Gauge Internal Storage

**`prom_counter.go:103`** — Counter uses a split design for performance:

```go
type counter struct {
    valBits uint64  // float64 bits (for non-integer adds)
    valInt  uint64  // integer accumulator (common fast path)
    // ...
}

func (c *counter) Inc() {
    atomic.AddUint64(&c.valInt, 1)   // lock-free integer fast path
}
func (c *counter) Add(v float64) {
    // if v is integer: atomic.AddUint64(&c.valInt, uint64(v))
    // else: CAS loop on c.valBits
}
func (c *counter) get() float64 {
    return math.Float64frombits(atomic.LoadUint64(&c.valBits)) +
           float64(atomic.LoadUint64(&c.valInt))
}
```

**`prom_gauge.go:90`** — Gauge is simpler (single atomic field):

```go
type gauge struct {
    valBits uint64  // float64 bits, single atomic value
}

func (g *gauge) Set(val float64) {
    atomic.StoreUint64(&g.valBits, math.Float64bits(val))
}
func (g *gauge) Add(val float64) {
    for {   // CAS retry loop — no integer fast path needed
        old := atomic.LoadUint64(&g.valBits)
        new := math.Float64bits(math.Float64frombits(old) + val)
        if atomic.CompareAndSwapUint64(&g.valBits, old, new) { return }
    }
}
```

---

## 6. Registry and Metric Families

**`prom_registry.go:260`** — `Registry` groups collectors:

```go
type Registry struct {
    mtx               sync.RWMutex
    collectorsByID    map[uint64]Collector   // XOR of all desc IDs → collector
    descIDs           map[uint64]struct{}    // all seen descriptor IDs
    dimHashesByName   map[string]uint64      // label consistency per name
    uncheckedCollectors []Collector
}
```

**`Registry.Gather()` (line 412)** is the collection entry point:

1. Spawns goroutines (up to `GOMAXPROCS`) — each calls `c.Collect(channel)`
2. Calls `metric.Write(dtoMetric)` per metric → serializes to protobuf DTO
3. Groups `dtoMetric`s into `MetricFamily` by name
4. Validates type consistency (can't mix Counter/Gauge under same name)
5. Hash-based duplicate detection (line 918)
6. Returns `[]*dto.MetricFamily` sorted lexicographically

---

## 7. Series Caching and Staleness

**`scrape.go:887`** — `scrapeCache` tracks series across scrapes:

```go
type scrapeCache struct {
    iter       uint64
    series     map[string]*cacheEntry      // metric text → entry
    seriesCur  map[storage.SeriesRef]*cacheEntry  // refs seen this scrape
    seriesPrev map[storage.SeriesRef]*cacheEntry  // refs seen last scrape
    metadata   map[string]*metaEntry
}

type cacheEntry struct {
    ref      storage.SeriesRef  // TSDB ref — avoids re-hashing labels
    lastIter uint64
    lset     labels.Labels
}
```

Staleness logic (line 937): series in `seriesPrev` but absent from `seriesCur`
get a `StaleNaN` marker appended — signaling TSDB that the series has gone away.

---

## Key Design Decisions

| Decision                                      | Rationale                                                          |
| --------------------------------------------- | ------------------------------------------------------------------ |
| Counter `valInt` + `valBits` split            | `atomic.AddUint64` is faster than CAS; most increments are integer |
| `SeriesRef` caching in `scrapeCache`          | Avoids re-hashing label sets on every scrape tick                  |
| `scrapePool` per job, `scrapeLoop` per target | Clean concurrency boundary; pools reload in parallel               |
| Parser abstraction over Content-Type          | Supports classic text, OpenMetrics, and protobuf with one loop     |
| Staleness via prev/cur ref maps               | O(1) detection of disappeared series without full set comparison   |

### Answer: `sonnet` / `mcp-only`

Here's a comprehensive breakdown of how Prometheus metrics scraping and
collection works in this codebase.

---

## Overview: The Data Flow

```
Discovery Manager
      │ target groups
      ▼
  Manager.Run()
      │ triggers reload
      ▼
  Manager.reload()
      │ creates/syncs
      ▼
  scrapePool.Sync()
      │ starts/stops per-target
      ▼
  scrapeLoop.run()
      │ on interval
      ▼
  targetScraper.scrape()   ← HTTP GET /metrics
      │ raw bytes
      ▼
  scrapeLoopAppender.append()
      │ text parsing via Parser
      ▼
  storage.Appender.Append() ← in-memory storage
```

---

## 1. Manager — Top-Level Coordinator

**`manager.go:133`** — `Manager` holds all scrape pools:

```go
type Manager struct {
    scrapePools  map[string]*scrapePool
    scrapeConfigs map[string]*config.ScrapeConfig
    targetSets   map[string][]*targetgroup.Group
    triggerReload chan struct{}
    // ...
}
```

**`manager.go:158`** — `Manager.Run()` receives target groups from the service
discovery manager and signals a reload:

```go
func (m *Manager) Run(tsets <-chan map[string][]*targetgroup.Group) error {
    go m.reloader()   // background goroutine
    for {
        select {
        case ts := <-tsets:
            m.updateTsets(ts)
            m.triggerReload <- struct{}{}
        }
    }
}
```

**`manager.go:186`** — `reloader()` debounces on a 5-second tick before calling
`reload()`.

**`manager.go:211`** — `reload()` creates missing `scrapePool`s and runs
`sp.Sync(groups)` for each job in parallel:

```go
go func(sp *scrapePool, groups []*targetgroup.Group) {
    sp.Sync(groups)
}(m.scrapePools[setName], groups)
```

---

## 2. scrapePool — Per-Job Manager

**`scrape.go:83`** — `scrapePool` tracks active targets and their loops:

```go
type scrapePool struct {
    activeTargets map[uint64]*Target   // hash → target
    loops         map[uint64]loop      // hash → scrapeLoop (same keys)
    client        *http.Client
    config        *config.ScrapeConfig
    appendable    storage.Appendable
    // ...
}
```

**`scrape.go:388`** — `Sync()` converts raw discovery groups into `Target`
structs, filters dropped ones, then calls the internal `sync()`:

**`scrape.go:436`** — `sync()` is where the actual reconciliation happens:

- For **new targets** (not in `activeTargets`): creates a `targetScraper` +
  `scrapeLoop` and adds them.
- For **disappeared targets**: calls `l.stop()` and removes them.
- Starts `go l.run(nil)` for new loops.

```go
// New target
l := sp.newLoop(scrapeLoopOptions{
    target:  t,
    scraper: &targetScraper{Target: t, client: sp.client, ...},
    cache:   newScrapeCache(sp.metrics),
    interval: targetInterval,
})
sp.activeTargets[hash] = t
sp.loops[hash] = l

// Gone target
l.stop()
delete(sp.loops, hash)
delete(sp.activeTargets, hash)
```

---

## 3. scrapeLoop — Per-Target Goroutine

**`scrape.go:822`** — `scrapeLoop` is the workhorse per target:

```go
type scrapeLoop struct {
    scraper     scraper           // does the actual HTTP fetch
    appendable  storage.Appendable
    cache       *scrapeCache      // tracks seen series for staleness
    interval    time.Duration
    timeout     time.Duration
    // label mutators, relabeling configs, limits...
}
```

The `loop` interface (implemented by `scrapeLoop`) has `run()`, `stop()`, etc.

**`scrape.go:1308`** — `scrapeAndReport()` is called each tick:

1. Calls `sl.scraper.scrape(ctx)` → gets `*http.Response`
2. Reads response body into a buffer:
   `sl.scraper.readResponse(scrapeCtx, resp, buf)`
3. Calls `app.append(b, contentType, appendTime)` to parse + store
4. On `appErr`: rolls back, re-appends empty bytes (triggers stale markers)
5. Calls `sl.report(...)` to write scrape meta-metrics (`up`,
   `scrape_duration_seconds`, etc.)

---

## 4. targetScraper — HTTP Fetcher

**`scrape.go:685`** — `targetScraper` wraps `*Target` with HTTP machinery:

```go
type targetScraper struct {
    *Target
    client               *http.Client
    req                  *http.Request
    timeout              time.Duration
    gzipr                *gzip.Reader
    bodySizeLimit        int64
    acceptHeader         string   // negotiates Protobuf vs text format
    acceptEncodingHeader string
}
```

It issues a GET to the target's `/metrics` endpoint and handles gzip
decompression.

---

## 5. Text Format Parsing

**`textparse_interface.go:63`** — The `Parser` interface drives incremental
parsing:

```go
type Parser interface {
    Next() (Entry, error)    // advance; returns EntryType
    Labels(l *labels.Labels) // write current sample's labels
    Series() []byte          // raw series bytes
    Exemplar(l *exemplar.Exemplar) bool
    StartTimestamp() int64   // created timestamp
}
```

Entry types include `EntryInvalid`, `EntrySeries`, `EntryComment`, `EntryType`,
`EntryHelp`, `EntryHistogram`.

The `scrapeLoopAppender.append()` method (**`scrape.go:1701`**) drives a `for`
loop over `p.Next()`, relabels the series, checks label limits, and dispatches
to the correct `storage.Appender` method:

```go
// Regular float sample
ref, err = app.Append(ref, lset, t, val)

// Native histogram
ref, err = app.AppendHistogram(ref, lset, t, h, nil)
```

---

## 6. Counter and Gauge — Internal Representation

**`prom_counter.go:103`** — `counter` uses a split int/float representation for
lock-free atomics:

```go
type counter struct {
    valBits uint64   // float64 bits for non-integer additions (CAS loop)
    valInt  uint64   // integer part (atomic add, fast path)
    exemplar atomic.Value
    labelPairs []*dto.LabelPair
}
```

**`prom_counter.go:126`** — `Add(v)`:

- If `v` is an exact integer → `atomic.AddUint64(&c.valInt, ival)` (fast path)
- Otherwise → CAS loop on `valBits` using `math.Float64bits`

**`prom_counter.go:151`** — `Inc()`:

```go
func (c *counter) Inc() { atomic.AddUint64(&c.valInt, 1) }
```

**`prom_counter.go:155`** — `get()` combines both:

```go
func (c *counter) get() float64 {
    fval := math.Float64frombits(atomic.LoadUint64(&c.valBits))
    ival := atomic.LoadUint64(&c.valInt)
    return fval + float64(ival)
}
```

**`prom_gauge.go:90`** — `gauge` uses a single `valBits uint64` with a CAS loop
for all operations:

```go
func (g *gauge) Add(val float64) {
    for {
        oldBits := atomic.LoadUint64(&g.valBits)
        newBits := math.Float64bits(math.Float64frombits(oldBits) + val)
        if atomic.CompareAndSwapUint64(&g.valBits, oldBits, newBits) { return }
    }
}
```

---

## 7. Registry — MetricFamily Management

**`prom_registry.go:253`** — `Registry` stores collectors by XOR'd descriptor
ID:

```go
type Registry struct {
    collectorsByID      map[uint64]Collector
    descIDs             map[uint64]struct{}
    dimHashesByName     map[string]uint64   // name → label dimension hash
    uncheckedCollectors []Collector
}
```

**`prom_registry.go:269`** — `Register(c Collector)`:

- Calls `c.Describe(descChan)` to enumerate descriptors
- Validates uniqueness of `descID` (hash of `fqName + constLabels`)
- Checks label name/help consistency via `dimHashesByName`
- XORs all `descID`s → `collectorID` → maps to the collector

**`prom_registry.go:411`** — `Gather()`:

1. Sends all `checkedCollectors` and `uncheckedCollectors` to worker goroutines
2. Workers call `collector.Collect(metricChan)` concurrently (one goroutine per
   collector, budget-limited)
3. Main loop drains `checkedMetricChan` / `uncheckedMetricChan`, calling
   `processMetric()` for each

**`prom_registry.go:619`** — `processMetric()`:

- Calls `metric.Write(dtoMetric)` to serialize to protobuf `dto.Metric`
- Looks up or creates `dto.MetricFamily` by name
- Infers type from which field is set (`dtoMetric.Counter`, `.Gauge`,
  `.Histogram`, etc.)
- Validates consistency with existing family
- Appends `dtoMetric` to `metricFamily.Metric`

Result: a sorted `[]*dto.MetricFamily` slice, ready for HTTP exposition.

---

## Summary Table

| Layer              | Type                        | Role                                           |
| ------------------ | --------------------------- | ---------------------------------------------- |
| `Manager`          | `manager.go:133`            | Receives discovery updates, orchestrates pools |
| `scrapePool`       | `scrape.go:83`              | Per-job: reconciles targets ↔ loops            |
| `scrapeLoop`       | `scrape.go:822`             | Per-target goroutine: ticks on interval        |
| `targetScraper`    | `scrape.go:685`             | Issues HTTP GET, handles gzip                  |
| `Parser`           | `textparse_interface.go:63` | Incremental text/proto format parser           |
| `counter`          | `prom_counter.go:103`       | Lock-free atomic, split int+float              |
| `gauge`            | `prom_gauge.go:90`          | Lock-free atomic CAS on float bits             |
| `Registry`         | `prom_registry.go:253`      | Collector registration + concurrent gather     |
| `dto.MetricFamily` | (protobuf)                  | Wire format grouping name + type + samples     |

### Answer: `sonnet` / `mcp-full`

Here's a comprehensive explanation of how Prometheus metrics scraping and
collection works in this codebase:

---

## Data Flow Overview

```
Discovery Manager
      │ target groups
      ▼
  Manager (manager.go)
      │ per job_name
      ▼
  scrapePool (scrape.go)
      │ per target (hash)
      ▼
  scrapeLoop + targetScraper
      │ HTTP GET /metrics
      ▼
  textparse.Parser (textparse_interface.go)
      │ parsed samples
      ▼
  storage.Appender → TSDB
```

---

## 1. Manager — Top-Level Coordinator

**`manager.go:133`** — `Manager` is the root coordinator:

```go
type Manager struct {
    scrapeConfigs map[string]*config.ScrapeConfig
    scrapePools   map[string]*scrapePool  // one pool per job
    targetSets    map[string][]*targetgroup.Group
    triggerReload chan struct{}
    ...
}
```

**Config reload flow (`manager.go:211`, `278`):**

- `reloader()` runs a ticker (default 5s); on `triggerReload`, calls `reload()`
- `reload()` creates new `scrapePool`s for new jobs, then calls
  `sp.Sync(groups)` on each in parallel goroutines
- `ApplyConfig()` handles configuration changes: stops removed pools, calls
  `sp.reload(cfg)` for changed ones

---

## 2. scrapePool — Per-Job Target Manager

**`scrape.go:83`** — `scrapePool` owns all targets for one job:

```go
type scrapePool struct {
    activeTargets map[uint64]*Target   // hash(labels) → Target
    loops         map[uint64]loop      // same hash → scrapeLoop
    client        *http.Client
    appendable    storage.Appendable
    ...
}
```

**`Sync` → `sync` (`scrape.go:388`, `436`):**

1. `Sync(tgs)` converts raw `targetgroup.Group`s into `*Target`s via
   `TargetsFromGroup`
2. `sync(targets)` deduplicates by label hash, then:
   - **New targets**: creates a `targetScraper` + `scrapeLoop`, stores in maps
   - **Removed targets**: calls `l.stop()` in a goroutine, deletes from maps
   - **Surviving targets**: left running, no restart needed
3. New loops are started immediately via `go l.run(nil)`

---

## 3. scrapeLoop — Per-Target Scrape Engine

**`scrape.go:822`** — `scrapeLoop` is the per-target goroutine:

```go
type scrapeLoop struct {
    interval  time.Duration
    timeout   time.Duration
    scraper   scraper           // the HTTP fetcher
    cache     *scrapeCache      // series ref cache between scrapes
    appendable storage.Appendable
    ...
}
```

**`scrapeAndReport` (`scrape.go:1308`)** is called on each tick:

1. Gets an `Appender` from storage
2. Calls `sl.scraper.scrape(ctx)` — HTTP GET with timeout
3. Reads body bytes into a buffer
4. Calls `app.append(b, contentType, ts)` to parse and store
5. On success: `app.Commit()`. On failure: `app.Rollback()`
6. Reports internal scrape metrics (duration, sample count, etc.)

**`targetScraper` (`scrape.go:685`)** is the HTTP layer:

```go
type targetScraper struct {
    *Target
    client  *http.Client
    req     *http.Request
    timeout time.Duration
    gzipr   *gzip.Reader  // optional decompression
    ...
}
```

---

## 4. Text Format Parsing

**`textparse_interface.go:28`** — `Parser` is the core parsing interface:

```go
type Parser interface {
    Next() (Entry, error)          // advance to next entry
    Series() ([]byte, *int64, float64)  // metric bytes, timestamp, value
    Histogram() ([]byte, *int64, *histogram.Histogram, *histogram.FloatHistogram)
    Labels(l *labels.Labels)       // label set for current sample
    Help() ([]byte, []byte)        // metric name + help text
    Type() ([]byte, model.MetricType)   // metric name + type
    Exemplar(l *exemplar.Exemplar) bool
    ...
}
```

**`scrapeLoopAppender.append` (`scrape.go:1562`)** drives the parse loop:

```go
p, err := textparse.New(b, contentType, sl.symbolTable, textparse.ParserOptions{...})

loop:
for {
    et, err := p.Next()   // EntryType, EntrySeries, EntryHistogram, ...
    switch et {
    case textparse.EntrySeries:
        met, parsedTimestamp, val = p.Series()
        p.Labels(&lset)
        ref, err = app.Append(ref, lset, t, val)
    case textparse.EntryHistogram:
        met, parsedTimestamp, h, fh = p.Histogram()
        ref, err = app.AppendHistogram(ref, lset, t, h, nil)
    }
}
```

The `scrapeCache` tracks series refs between scrapes to avoid label re-lookup on
every interval.

---

## 5. Counter and Gauge Internal Storage

Both use **lock-free atomic operations** on `uint64` bit patterns:

**Counter (`prom_counter.go:126`, `151`):**

```go
type counter struct {
    valInt  uint64  // fast path: integer additions
    valBits uint64  // float path: for non-integer Add()
    ...
}

func (c *counter) Inc() {
    atomic.AddUint64(&c.valInt, 1)   // single atomic instruction
}

func (c *counter) Add(v float64) {
    ival := uint64(v)
    if float64(ival) == v {
        atomic.AddUint64(&c.valInt, ival)  // integer fast path
        return
    }
    // float CAS loop for fractional values
    for {
        oldBits := atomic.LoadUint64(&c.valBits)
        newBits := math.Float64bits(math.Float64frombits(oldBits) + v)
        if atomic.CompareAndSwapUint64(&c.valBits, oldBits, newBits) { return }
    }
}
// get() returns float64(valInt) + valBits-decoded float
```

**Gauge (`prom_gauge.go:90`, `122`):** always uses the CAS float loop since it
can decrease:

```go
type gauge struct {
    valBits uint64  // single float64 stored as bits
}

func (g *gauge) Add(val float64) {
    for {
        oldBits := atomic.LoadUint64(&g.valBits)
        newBits := math.Float64bits(math.Float64frombits(oldBits) + val)
        if atomic.CompareAndSwapUint64(&g.valBits, oldBits, newBits) { return }
    }
}
```

Key difference: counters use a **split int+float scheme** (integer `Inc`/`Add`
is a single `atomic.Add`; only fractional values use CAS), while gauges always
CAS since they support `Sub`/`Dec`.

---

## 6. Registry and MetricFamily Collection

**`prom_registry.go:411`** — `Registry.Gather()` orchestrates collection:

1. Fans all registered `Collector`s out to `checkedCollectors` /
   `uncheckedCollectors` channels
2. Spawns worker goroutines (`collectWorker`) that call
   `collector.Collect(metricChan)`
3. Each `Metric.Write(dtoMetric)` serializes the current value into a protobuf
   `dto.Metric`

**`processMetric` (`prom_registry.go:619`)** merges each metric into a
`MetricFamily`:

- First occurrence for a name creates a new `dto.MetricFamily` with `Name`,
  `Help`, and inferred `Type`
- Subsequent occurrences check consistency (same help text, same type)
- All `dto.Metric` instances get appended to `metricFamily.Metric`
- Duplicate detection via fnv64 hash of `{name, label pairs, timestamp}`

```
Collector.Collect(ch)
    → each Metric.Write(dtoMetric)
        → dtoMetric.Gauge / .Counter / .Histogram → typed value
    → processMetric groups by name into MetricFamily
        → metricFamiliesByName["my_metric"] = &dto.MetricFamily{
              Name:   "my_metric",
              Type:   GAUGE,
              Metric: []*dto.Metric{...},
          }
→ NormalizeMetricFamilies → sorted []*dto.MetricFamily
```

---

## Summary of Key Types

| Type                 | File                        | Role                                                       |
| -------------------- | --------------------------- | ---------------------------------------------------------- |
| `Manager`            | `manager.go:133`            | Top-level coordinator; one `scrapePool` per job            |
| `scrapePool`         | `scrape.go:83`              | Manages target lifecycle; one `scrapeLoop` per target      |
| `scrapeLoop`         | `scrape.go:822`             | Per-target goroutine; drives HTTP fetch + parse + append   |
| `targetScraper`      | `scrape.go:685`             | HTTP client wrapper for one target                         |
| `scrapeLoopAppender` | `scrape.go:1554`            | Wraps `scrapeLoop` + `storage.Appender`; drives parse loop |
| `Parser`             | `textparse_interface.go:28` | Interface for Prometheus/OpenMetrics text format parsing   |
| `counter`            | `prom_counter.go`           | Lock-free atomic counter (split int+float)                 |
| `gauge`              | `prom_gauge.go:90`          | Lock-free atomic gauge (CAS float)                         |
| `Registry`           | `prom_registry.go`          | Collector registry + `Gather()` fan-out                    |
| `dto.MetricFamily`   | (proto)                     | Wire-format grouping of metrics by name+type               |

### Answer: `opus` / `baseline`

## How Prometheus Metrics Scraping & Collection Works

Based on the fixture code in `testdata/fixtures/go/`, here's the full picture:

---

### 1. Scrape Manager — The Coordinator

**`manager.go`** — The `Manager` struct is the top-level orchestrator:

```
Manager
├── scrapeConfigs   map[string]*ScrapeConfig     // job_name → config
├── scrapePools     map[string]*scrapePool        // job_name → pool of loops
├── targetSets      map[string][]*targetgroup.Group
├── buffers         *pool.Pool                    // reusable byte slices
└── triggerReload   chan struct{}                  // signal reload
```

**Flow:**

1. `Run(tsets)` receives target groups from the discovery manager
2. Stores them in `targetSets`, then signals `triggerReload`
3. A background goroutine calls `reload()`, which syncs each pool concurrently
4. `ApplyConfig()` hot-reloads scrape configs and rebuilds affected pools

Each **scrape pool** (`scrape.go`) manages one loop per target:

```go
type scrapePool struct {
    config          *config.ScrapeConfig
    loops           map[uint64]loop          // one per unique target
    activeTargets   map[uint64]*Target
    symbolTable     *labels.SymbolTable      // interns repeated label strings
}
```

`Sync([]*targetgroup.Group)` hashes each target by its labels — new targets get
a loop started, removed targets get stopped.

---

### 2. Parsing the Text Format

**`textparse_interface.go`** defines the `Parser` interface that all format
parsers implement:

```go
type Parser interface {
    Next() (Entry, error)              // advance to next entry
    Series() ([]byte, *int64, float64) // metric name bytes, optional timestamp, value
    Histogram() ([]byte, *int64, *histogram.Histogram, *histogram.FloatHistogram)
    Help() ([]byte, []byte)            // metric name, help text
    Type() ([]byte, model.MetricType)  // metric name, type (counter/gauge/etc.)
    Labels(l *labels.Labels)
    Exemplar(l *exemplar.Exemplar) bool
}
```

Entry types map to lines in the exposition format:

- `EntryType` → `# TYPE http_requests_total counter`
- `EntryHelp` → `# HELP http_requests_total Total requests`
- `EntrySeries` → `http_requests_total{method="GET"} 1234`
- `EntryHistogram` → native histogram samples

Three formats are supported: `text/plain`, `application/openmetrics-text`, and
protobuf. The parser is selected by `extractMediaType()` on the HTTP response.

---

### 3. Counter & Gauge Internals

**`prom_counter.go`** — Counters use a dual-value atomic trick for performance:

```go
type counter struct {
    valBits uint64           // atomic float64 bits (for Add with fractional values)
    valInt  uint64           // atomic uint64 (fast path for Inc)
    desc    *Desc
    exemplar atomic.Value    // *dto.Exemplar
}
```

- **`Inc()`** → `atomic.AddUint64(&c.valInt, 1)` — single instruction, no CAS
  loop
- **`Add(v)`** → if integer-representable, adds to `valInt`; otherwise CAS-loops
  on `valBits`
- **`Write()`** → final value = `float64(valInt) + Float64frombits(valBits)` —
  merges both accumulators
- Panics if `v < 0` (counters only go up)

**`prom_gauge.go`** — Gauges are simpler, just atomic float64:

```go
type gauge struct {
    valBits uint64           // atomic float64 bits
    desc    *Desc
}
```

- **`Set(val)`** → `atomic.StoreUint64(&g.valBits, math.Float64bits(val))`
- **`Add(val)`** → CAS loop: load old bits → convert to float → add → store new
  bits
- `Inc()`, `Dec()`, `Sub()` all delegate to `Add()`

---

### 4. Registry & MetricFamily Management

**`prom_registry.go`** — The `Registry` is the central collection point:

```go
type Registry struct {
    collectorsByID       map[uint64]Collector   // keyed by descriptor hash
    descIDs              map[uint64]struct{}     // all registered descriptor IDs
    dimHashesByName      map[string]uint64       // ensures label consistency per name
    uncheckedCollectors  []Collector             // collectors that skip Describe()
}
```

**Registration** (`Register()`):

- Calls `Describe()` to get all descriptors from the collector
- Validates: no duplicate names, consistent label sets, valid descriptor
- Stores by descriptor ID hash

**Gathering** (`Gather()`):

- Spawns goroutines to call `Collect()` on all registered collectors
  concurrently
- Each collector sends `Metric` values on a channel
- Metrics are grouped into `MetricFamily` structs (one per metric name)
- Validates type consistency and label uniqueness within each family
- Returns `[]*dto.MetricFamily` sorted lexicographically by name

---

### 5. End-to-End Data Flow

```
Discovery Manager
    │  target groups (endpoints to scrape)
    ▼
Scrape Manager (manager.go)
    │  reload() → sync pools concurrently
    ▼
Scrape Pool (scrape.go)
    │  one loop per target, managed by label hash
    ▼
Scrape Loop
    │  HTTP GET → target:port/metrics
    ▼
Text Parser (textparse_interface.go)
    │  Next() loop → Series()/Histogram()/Help()/Type()
    ▼
Relabeling (relabel.go)
    │  replace, keep, drop, hashmod, labelmap, etc.
    ▼
Storage Appender (interface.go)
    │  Append(ref, labels, timestamp, value)
    │  AppendHistogram(ref, labels, timestamp, histogram)
    │  AppendExemplar(ref, labels, exemplar)
    ▼
Commit() → TSDB Head (in-memory + WAL)
```

Key architectural patterns throughout:

- **Object pooling** (`pool.Pool`) to reduce GC pressure on byte buffers
- **Symbol tables** (`labels.SymbolTable`) to intern repeated label strings
  across targets
- **Staleness markers** (`StaleNaN = 0x7ff0000000000002`) appended when a series
  disappears between scrapes
- **SeriesRef** — ephemeral handles returned by `Append()` for fast subsequent
  appends to the same series, avoiding repeated label lookups

### Answer: `opus` / `mcp-only`

Here's a comprehensive walkthrough of how Prometheus metrics scraping and
collection works in this codebase.

---

## 1. Scrape Manager — Top-Level Orchestrator

The entry point is the `Manager` struct in `manager.go:133-156`:

```go
type Manager struct {
    scrapeConfigs map[string]*config.ScrapeConfig
    scrapePools   map[string]*scrapePool
    targetSets    map[string][]*targetgroup.Group
    triggerReload chan struct{}
    // ...
}
```

**Data flow:**

1. **`Manager.Run()`** (`manager.go:158-179`) listens on a channel of target
   group updates from service discovery. When new target sets arrive, it calls
   `updateTsets()` and signals `triggerReload`.

2. **`Manager.reloader()`** (`manager.go:186-209`) runs in a background
   goroutine with a ticker (default 5s). When it receives a reload signal, it
   calls `Manager.reload()`.

3. **`Manager.reload()`** (`manager.go:211-245`) creates a `scrapePool` for each
   scrape config that doesn't have one yet, then calls `sp.Sync(groups)` in
   parallel goroutines to reconcile active targets.

4. **`Manager.ApplyConfig()`** (`manager.go:278-344`) handles config reloads —
   it updates scrape configs, cleans up removed pools, and reloads existing
   pools concurrently (bounded by `GOMAXPROCS`).

## 2. Scrape Pool — Per-Job Target Management

`scrapePool` (`scrape.go:83-116`) manages all targets for a single scrape job:

```go
type scrapePool struct {
    config        *config.ScrapeConfig
    client        *http.Client
    loops         map[uint64]loop           // hash → scrape loop
    activeTargets map[uint64]*Target        // hash → target
    appendable    storage.Appendable        // where samples go
    // ...
}
```

**`scrapePool.sync()`** (`scrape.go:494-525`) is the key reconciliation method.
For each discovered target, it either reuses an existing loop or creates a new
one via `sp.newLoop()`. Old loops not in the new target set are stopped. New
loops are started with `go l.run(nil)`.

## 3. Scrape Loop — The Per-Target Ticker

`scrapeLoop` (`scrape.go:822-881`) is the core scraping engine for a single
target:

```go
type scrapeLoop struct {
    scraper   scraper        // does the HTTP GET
    cache     *scrapeCache   // tracks series across scrapes
    interval  time.Duration
    timeout   time.Duration
    appendable storage.Appendable
    // ...
}
```

**`scrapeLoop.run()`** (`scrape.go:1234-1299`) is the main loop:

1. Waits for an initial offset (to spread scrape load across HA setups)
2. Runs a `time.Ticker` at the configured interval
3. Each tick calls `sl.scrapeAndReport(last, scrapeTime, errc)` which performs
   the HTTP scrape and appends results

The actual HTTP request is done by `targetScraper.scrape()`
(`scrape.go:735-752`) — a simple `GET` with `Accept`, `Accept-Encoding`,
`User-Agent`, and timeout headers.

## 4. Parsing the Text Format

After the HTTP response body is read, `scrapeLoopAppender.append()`
(`scrape.go:1562-1624`) creates a parser:

```go
p, err := textparse.New(b, contentType, sl.symbolTable, textparse.ParserOptions{...})
```

The `Parser` interface (`textparse_interface.go:28-87`) is a streaming parser
that yields entries one at a time:

```go
type Parser interface {
    Series() ([]byte, *int64, float64)
    Histogram() ([]byte, *int64, *histogram.Histogram, *histogram.FloatHistogram)
    Help() ([]byte, []byte)
    Type() ([]byte, model.MetricType)
    Labels(l *labels.Labels)
    Exemplar(l *exemplar.Exemplar) bool
    Next() (Entry, error)
}
```

Entry types: `EntrySeries`, `EntryHistogram`, `EntryType`, `EntryHelp`,
`EntryUnit`, `EntryComment`.

The append loop (`scrape.go:1625-1762`) iterates the parser:

- **`EntryType`/`EntryHelp`/`EntryUnit`** → cached as metadata in `sl.cache`
- **`EntrySeries`** → calls `p.Series()` to get metric name bytes, optional
  timestamp, and float64 value
- **`EntryHistogram`** → calls `p.Histogram()` for native histogram data

For each sample, labels are resolved (from cache or freshly parsed), mutated
through relabeling, and then appended to storage via
`app.Append(ref, lset, t, val)` or `app.AppendHistogram(...)`.

## 5. Counters and Gauges — Internal Tracking

### Counter (`prom_counter.go:103-120`)

```go
type counter struct {
    valBits uint64  // float64 bits for fractional adds
    valInt  uint64  // exact integer accumulator
    desc    *Desc
    labelPairs []*dto.LabelPair
    exemplar   atomic.Value
}
```

- **`Inc()`** (`prom_counter.go:151`): `atomic.AddUint64(&c.valInt, 1)` — fast
  path for integer increments
- **`Add(v)`** (`prom_counter.go:126-144`): If `v` is an exact integer, uses
  `atomic.AddUint64` on `valInt`. Otherwise, CAS loop on `valBits` for float64
  addition. Panics if `v < 0`.
- **`get()`** (`prom_counter.go:155-159`): Returns
  `float64(valInt) + Float64frombits(valBits)` — combines both accumulators

The dual-accumulator design avoids floating-point precision loss for the common
case of integer increments.

### Gauge (`prom_gauge.go:90-100`)

```go
type gauge struct {
    valBits uint64  // float64 bits
    desc    *Desc
    labelPairs []*dto.LabelPair
}
```

- **`Add(val)`** (`prom_gauge.go:122-130`): CAS loop —
  `atomic.CompareAndSwapUint64` with `Float64bits/Float64frombits`
- **`Inc()`/`Dec()`**: Delegate to `Add(1)`/`Add(-1)`
- **`Set(v)`**: Direct `atomic.StoreUint64`

Both use lock-free atomics for thread safety.

## 6. Registry — Managing Metric Families

### Registry struct (`prom_registry.go:253-267`)

```go
type Registry struct {
    mtx                 sync.RWMutex
    collectorsByID      map[uint64]Collector  // XOR of desc IDs → collector
    descIDs             map[uint64]struct{}
    dimHashesByName     map[string]uint64
    uncheckedCollectors []Collector
}
```

### Registration (`prom_registry.go:269-363`)

`Register(c Collector)`:

1. Calls `c.Describe(descChan)` to get all `Desc` descriptors
2. Validates each descriptor (unique fqName + constLabels combo)
3. Checks label name/help string consistency against previously registered
   descriptors
4. Computes `collectorID` as XOR of all desc IDs
5. Stores in `collectorsByID[collectorID] = c`
6. Collectors with no descriptors go into `uncheckedCollectors`

### Gathering (`prom_registry.go:411-560`)

`Registry.Gather()` is the collection pipeline:

1. Launches **concurrent collector workers** — one goroutine per collector, each
   calling `collector.Collect(metricChan)`
2. Drains metrics from two channels (checked and unchecked) in a select loop
3. Each metric goes through **`processMetric()`** (`prom_registry.go:619-725`):
   - Calls `metric.Desc()` to get the descriptor
   - Calls `metric.Write(dtoMetric)` to serialize into protobuf
   - Groups into `metricFamiliesByName` map, auto-detecting type
     (COUNTER/GAUGE/SUMMARY/HISTOGRAM/UNTYPED)
   - Validates consistency — same name must have same help string and type
   - Deduplicates via `metricHashes`
4. Returns `NormalizeMetricFamilies(metricFamiliesByName)` — sorted, validated
   MetricFamily slice

The `Gatherers` type (`prom_registry.go:727-803`) composes multiple `Gatherer`
instances, merging results and deduplicating across registries.

## Summary Data Flow

```
Discovery → Manager.Run() → triggerReload → Manager.reload()
                                                  │
                                    ┌──────────────┘
                                    ▼
                             scrapePool.Sync()
                              │  creates/stops loops
                              ▼
                         scrapeLoop.run()
                          │  ticker every interval
                          ▼
                     targetScraper.scrape()  ← HTTP GET /metrics
                          │
                          ▼
                     textparse.Parser.Next()  ← streaming parse
                          │
                          ▼
                  scrapeLoopAppender.append()
                   │  label mutation, relabeling
                   ▼
              storage.Appender.Append(ref, labels, timestamp, value)
                          │
                          ▼
                   TSDB Head (in-memory)
```

On the client library side (for exposing metrics):

```
Counter/Gauge.Add() → atomic update → Collect(ch) → Write(dto.Metric)
     → Registry.Gather() → processMetric() → MetricFamily protobuf → HTTP exposition
```

### Answer: `opus` / `mcp-full`

## Prometheus Metrics Scraping & Collection — Data Flow

The fixture project at `testdata/fixtures/go` contains excerpts from core
Prometheus packages. Here's how the pieces fit together:

---

### 1. Scrape Manager (`manager.go`) — Top-level Coordinator

The `Manager` struct owns a map of **scrape pools**, one per job name:

```go
type Manager struct {
    scrapeConfigs map[string]*config.ScrapeConfig
    scrapePools   map[string]*scrapePool
    targetSets    map[string][]*targetgroup.Group
    appendable    storage.Appendable
    // ...
}
```

**Flow:** `Run()` receives target groups from service discovery on a channel,
debounces them via `triggerReload`, and a `reloader()` goroutine (throttled to
every ~5s) calls `reload()`. That method creates a `scrapePool` for each new job
and calls `sp.Sync(groups)` to reconcile targets.

`ApplyConfig()` handles config reloads — it calls `sp.reload(cfg)` for changed
pools and stops removed ones, bounded by `GOMAXPROCS` concurrent goroutines.

---

### 2. Scrape Loop (`scrape.go`) — Per-Target Worker

Each discovered target gets a `scrapeLoop` + `targetScraper` pair:

```go
type scrapeLoop struct {
    scraper     scraper          // HTTP client (targetScraper)
    cache       *scrapeCache     // label/ref cache for this target
    interval    time.Duration
    appendable  storage.Appendable
    sampleMutator labelsMutator  // honor_labels + metric relabeling
    // ...
}
```

**`run()`** ticks at the configured interval (with an initial jitter offset
computed via FNV-64a hash to spread HA pairs). Each tick calls
**`scrapeAndReport()`**:

1. `targetScraper.scrape(ctx)` — HTTP GET with `Accept`, gzip, timeout headers
2. `targetScraper.readResponse()` — validates 200, decompresses gzip, enforces
   body size limit
3. `app.append(body, contentType, timestamp)` — the core parse-and-store step
   (see below)
4. On success: `appender.Commit()`; on error: `Rollback()` + emit stale markers

**Staleness tracking** uses two maps swapped each iteration:

```go
type scrapeCache struct {
    series     map[string]*cacheEntry        // metric bytes → (SeriesRef, labels, hash)
    seriesCur  map[storage.SeriesRef]*cacheEntry  // seen this scrape
    seriesPrev map[storage.SeriesRef]*cacheEntry  // seen last scrape
    metadata   map[string]*metaEntry         // family name → (Type, Help, Unit)
}
```

After parsing, anything in `seriesPrev` but not `seriesCur` gets a `StaleNaN`
(`0x7ff0000000000002`) sample appended. Then the maps are swapped for the next
iteration.

**SeriesRef caching** is the key optimization: on the first scrape,
`Append(0, lset, t, v)` returns a `SeriesRef`. On subsequent scrapes, the cached
ref is passed directly, skipping label lookup in TSDB.

---

### 3. Text Format Parsing (`textparse_interface.go`) — Iterator Protocol

```go
type Parser interface {
    Next() (Entry, error)        // EntryType, EntryHelp, EntrySeries, EntryHistogram, ...
    Series() ([]byte, *int64, float64)
    Histogram() ([]byte, *int64, *histogram.Histogram, *histogram.FloatHistogram)
    Help() ([]byte, []byte)
    Type() ([]byte, model.MetricType)
    Labels(l *labels.Labels)
}
```

`New(body, contentType, ...)` dispatches to `PromParser` (text/plain),
`OpenMetricsParser`, or `ProtobufParser`. The parse loop in `append()` calls
`parser.Next()` repeatedly:

- **`EntryType`/`EntryHelp`/`EntryUnit`** → update `scrapeCache.metadata`
- **`EntrySeries`** → extract labels, apply `mutateSampleLabels()`
  (honor_labels + relabeling), then `appender.Append(ref, lset, t, v)`
- **`EntryHistogram`** → similar path via `appender.AppendHistogram()`

---

### 4. Counter (`prom_counter.go`) — Split Atomic Design

```go
type counter struct {
    valBits uint64   // float64 bits, updated via CAS loop for non-integer Add()
    valInt  uint64   // integer accumulator for Inc() and integer Add()
    desc       *Desc
    exemplar   atomic.Value
    labelPairs []*dto.LabelPair
}
```

**Why two fields?** `Inc()` uses a single `atomic.AddUint64(&c.valInt, 1)` — no
CAS retry. Non-integer `Add(v)` uses a CAS loop on `valBits`. `get()` returns
`Float64frombits(valBits) + float64(valInt)`. This keeps the extremely hot
`Inc()` path lock-free with one instruction.

`CounterVec` wraps a `MetricVec` — a hash-map from label values (e.g.,
`"200", "GET"`) to individual `counter` instances.

---

### 5. Gauge (`prom_gauge.go`) — Simple Atomic Float

```go
type gauge struct {
    valBits uint64   // atomic store of float64 bits
    desc       *Desc
    labelPairs []*dto.LabelPair
}
```

- `Set(v)` → `atomic.StoreUint64` (cheapest path, no CAS)
- `Add(v)` → CAS loop (load, compute, swap)
- `Inc()`/`Dec()` → delegate to `Add(±1)`

Simpler than counter because gauges can go both directions — no need for a
separate integer fast path.

---

### 6. Registry (`prom_registry.go`) — Metric Family Management

```go
type Registry struct {
    collectorsByID  map[uint64]Collector   // XOR of desc IDs → Collector
    descIDs         map[uint64]struct{}    // all known desc IDs
    dimHashesByName map[string]uint64      // fqName → hash(label names + help)
    uncheckedCollectors []Collector        // collectors with no Describe output
}
```

**`Register(c)`**: calls `c.Describe(ch)`, XORs all descriptor IDs into a
`collectorID`, checks for collisions in `descIDs` and schema consistency in
`dimHashesByName` (same metric name must always have the same label names + help
string).

**`Gather()`**: spawns worker goroutines that call `c.Collect(ch)` for each
registered collector. Each emitted `dto.Metric` is grouped into
`dto.MetricFamily` by name, checked for type consistency and duplicate label
sets (via xxhash), then returned sorted lexicographically.

---

### End-to-End Data Flow

```
Service Discovery
    │  target groups
    ▼
Manager.Run() → reload() → scrapePool.Sync()
    │                           │
    │                    creates scrapeLoop + targetScraper per target
    │                           │
    ▼                           ▼
scrapeLoop.run()  ──ticker──▶  scrapeAndReport()
                                │
                    ┌───────────┼───────────────┐
                    ▼           ▼               ▼
              HTTP GET    textparse.New()   scrapeCache
              (gzip,      Parser.Next()    get()/addRef()
               timeout)   loop:            trackStaleness()
                          Type→metadata
                          Series→Append()
                          Histogram→AppendHistogram()
                                │
                                ▼
                    forEachStale() → Append(StaleNaN)
                                │
                                ▼
                         appender.Commit()
                                │
                                ▼
                    Head (WAL + stripeSeries + MemPostings)
```

The key insight is the **three-layer caching** strategy: `scrapeCache` caches
`SeriesRef` values to avoid repeated label resolution in TSDB, pools byte
buffers across scrapes to reduce GC pressure, and swaps two maps each iteration
for O(1) staleness detection.

### Full Judge Analysis (Opus 4.6)

## Content Quality

### Ranking: 1st through 6th

**1. opus / mcp-full** — The strongest answer overall. Correctly identifies that
the code lives in `testdata/fixtures/go/` (fixture code, not the main project).
Uniquely covers the staleness tracking mechanism in depth with the
`seriesCur`/`seriesPrev` swap pattern and `StaleNaN` sentinel value. The
SeriesRef caching explanation is precise and contextualized as an optimization.
The three-layer caching insight at the end (SeriesRef, buffer pooling, staleness
maps) shows genuine understanding rather than surface-level enumeration.
File/line references are accurate. The counter explanation correctly identifies
_why_ the split design exists (keeping `Inc()` to a single atomic instruction).
Only weakness: took dramatically longer and cost far more than other runs.

**2. sonnet / mcp-full** — Very complete and well-structured. Covers all four
requested areas (manager coordination, parsing, counter/gauge internals,
registry). The `scrapeLoopAppender.append` code snippet showing the actual parse
loop is a strong addition. The `processMetric` explanation with type inference
from DTO fields is accurate. File references like `scrape.go:1562` and
`prom_registry.go:619` appear precise. The summary table is clean and useful.
Slightly less depth than opus/mcp-full on staleness and caching, but covers the
registry pipeline more explicitly with the `NormalizeMetricFamilies` step.

**3. sonnet / baseline** — Impressively detailed despite having no MCP search
tools. Covers all major areas with accurate code snippets. The counter/gauge
section is particularly good, showing the actual `get()` method combining both
accumulators. The staleness section with `scrapeCache` and
`seriesCur`/`seriesPrev` maps is accurate. The "Key Design Decisions" table at
the end adds value. Minor concern: some line numbers may be approximations since
it couldn't search the actual codebase, but the structural understanding is
solid.

**4. opus / mcp-only** — Thorough and well-organized with accurate detail. The
registration flow in section 6 is the most detailed of all answers, walking
through the XOR descriptor ID computation step by step. Good coverage of the
dual data flow (scrape side vs client library side) in the summary diagram. The
`CounterVec` mention is unique and relevant. Slightly verbose in places, and the
formatting is dense.

**5. sonnet / mcp-only** — Solid coverage of all topics with accurate code
snippets. The `scrapePool.sync()` reconciliation logic is well-explained with
the new/gone target code blocks. Good detail on `targetScraper` fields. However,
it's slightly less precise on some line references compared to mcp-full
variants, and the registry section, while correct, is less detailed on the
validation pipeline. The summary table is clean but adds less insight than other
answers' closing sections.

**6. opus / baseline** — The shortest and least detailed answer. While
everything stated is correct, it's noticeably thinner than the others. The
parsing section lacks the actual parse loop code. The registry section omits
`processMetric` details. The counter/gauge sections are accurate but brief. The
end-to-end flow diagram is good but the answer overall feels like it stopped
short. It does correctly mention FNV-64a hash for HA pair offset and xxhash for
duplicate detection, which are nice specific details other answers miss.

---

## Efficiency Analysis

| Run               | Duration | Input Tok | Output Tok | Cost      | Quality Rank |
| ----------------- | -------- | --------- | ---------- | --------- | ------------ |
| sonnet / mcp-full | 62.6s    | 112,663   | 3,663      | $0.68     | 2nd          |
| sonnet / mcp-only | 66.4s    | 91,257    | 3,951      | $0.56     | 5th          |
| opus / mcp-only   | 70.8s    | 100,358   | 3,883      | $0.60     | 4th          |
| opus / baseline   | 100.3s   | 32,629    | 2,103      | $0.65     | 6th          |
| sonnet / baseline | 120.8s   | 34,214    | 2,657      | $0.82     | 3rd          |
| opus / mcp-full   | 259.8s   | 35,841    | 2,462      | **$4.40** | 1st          |

**Key observations:**

- **opus / mcp-full is a massive outlier on cost** — $4.40 vs $0.56–$0.82 for
  everything else. It took 4x longer than the next slowest run. The low input
  token count (35K) with high cost suggests many sequential tool calls with
  expensive per-turn overhead on Opus. The quality edge over sonnet/mcp-full is
  modest and doesn't justify 6.4x the cost.

- **sonnet / mcp-only is the cheapest run** at $0.56, and the mcp-only runs are
  generally faster than baselines despite higher input tokens — the search tools
  front-load context efficiently.

- **Baseline runs are surprisingly competitive on quality** — sonnet/baseline
  ranked 3rd despite having no code search tools at all. It relied on training
  data / cache reads (28K cached tokens) and still produced accurate, detailed
  output. However, it was the slowest sonnet run.

- **Cache reads correlate with baseline performance** — both baselines show ~28K
  cached tokens, suggesting the model drew heavily on prior context or training
  knowledge about Prometheus internals.

- **MCP search increases input tokens significantly** (91K–113K vs 32K–34K for
  baselines) but this translates to faster wall-clock times for sonnet, likely
  because the model needs fewer reasoning steps when it has source code in
  context.

**Recommendation:** **sonnet / mcp-full** offers the best quality-to-cost
tradeoff — 2nd-ranked quality at $0.68 and the fastest wall time (62.6s). If
budget is tight, **sonnet / mcp-only** at $0.56 is reasonable but sacrifices
some precision. The opus/mcp-full run should be avoided for this type of
question — the quality gain is marginal while the cost is prohibitive.
