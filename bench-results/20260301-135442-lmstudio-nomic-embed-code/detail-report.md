# Benchmark Detail Report

Generated: 2026-03-01 13:16 UTC | Results: `20260301-135442`

---

## label-matcher [easy]

**Question:** What label matcher types are available and how is a Matcher
created? Show the type definitions and constructor.

### Metrics

| Model      | Scenario | Duration | Input Tok | Cache Read | Cache Created | Output Tok | Cost (USD) |
| ---------- | -------- | -------- | --------- | ---------- | ------------- | ---------- | ---------- |
| **sonnet** | baseline | 20.7s    | 27763     | 28104      | 0             | 727        | $0.24109   |
| **sonnet** | mcp-only | 10.9s    | 25075     | 0          | 0             | 759        | $0.14435   |
| **sonnet** | mcp-full | 13.1s    | 44008     | 42156      | 0             | 769        | $0.26034   |
| **opus**   | baseline | 14.9s    | 44898     | 42345      | 0             | 712        | $0.26346   |
| **opus**   | mcp-only | 10.4s    | 17478     | 0          | 0             | 576        | $0.10179   |
| **opus**   | mcp-full | 15.9s    | 46124     | 42345      | 0             | 674        | $0.26864   |

### Answer: `sonnet` / `baseline`

### Label Matcher Types — `labels_matcher.go`

**MatchType enum** (lines 21–30):

```go
type MatchType int

const (
    MatchEqual     MatchType = iota  // =
    MatchNotEqual                    // !=
    MatchRegexp                      // =~
    MatchNotRegexp                   // !~
)
```

Four matching strategies: exact equality, inequality, regex match, and negated
regex match.

---

**Matcher struct** (lines 46–53):

```go
type Matcher struct {
    Type  MatchType
    Name  string
    Value string

    re *FastRegexMatcher  // compiled regex, set only for regexp types
}
```

---

**Constructors** (lines 55–79):

```go
// Primary constructor — compiles regex if needed.
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

// Panic variant — for tests only.
func MustNewMatcher(mt MatchType, name, val string) *Matcher {
    m, err := NewMatcher(mt, name, val)
    if err != nil {
        panic(err)
    }
    return m
}
```

`NewMatcher` is the standard constructor. It lazily compiles a
`FastRegexMatcher` only when the type is `MatchRegexp` or `MatchNotRegexp`.
`MustNewMatcher` is a test-only convenience wrapper that panics instead of
returning an error.

### Answer: `sonnet` / `mcp-only`

Here's the full picture from `labels_matcher.go`:

---

### Match Types

There are four `MatchType` constants defined as an `iota` enum:

```go
// labels_matcher.go:21-22
type MatchType int

// labels_matcher.go:26-37
const (
    MatchEqual     MatchType = iota  // "="
    MatchNotEqual                    // "!="
    MatchRegexp                      // "=~"
    MatchNotRegexp                   // "!~"
)

var matchTypeToStr = [...]string{
    MatchEqual:     "=",
    MatchNotEqual:  "!=",
    MatchRegexp:    "=~",
    MatchNotRegexp: "!~",
}
```

---

### Matcher Type

```go
// labels_matcher.go:46-53
type Matcher struct {
    Type  MatchType
    Name  string
    Value string

    re *FastRegexMatcher  // populated only for regexp types
}
```

---

### Constructor

```go
// labels_matcher.go:55-70
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

Key behavior: for `MatchRegexp` and `MatchNotRegexp`, `NewMatcher` compiles the
value string into a `FastRegexMatcher` at construction time and stores it in the
unexported `re` field. For equality types (`MatchEqual`/`MatchNotEqual`), `re`
is left nil. There's also `MustNewMatcher` (`labels_matcher.go:72-79`) which
panics on error, intended only for tests.

### Answer: `sonnet` / `mcp-full`

Here's a summary of the label matcher types and how a `Matcher` is created:

---

### `MatchType` — `labels_matcher.go:21-30`

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

Four types, corresponding to the classic PromQL label matching operators.

---

### `Matcher` struct — `labels_matcher.go:46-53`

```go
type Matcher struct {
    Type  MatchType
    Name  string
    Value string

    re *FastRegexMatcher  // populated only for regexp types
}
```

---

### `NewMatcher` constructor — `labels_matcher.go:55-70`

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

For `MatchRegexp`/`MatchNotRegexp`, it compiles the pattern into a
`FastRegexMatcher` at construction time and returns an error on invalid regex.
For `MatchEqual`/`MatchNotEqual`, no regex is compiled and construction is
infallible.

There's also `MustNewMatcher` (`labels_matcher.go:72-79`) which panics on error
— intended only for tests.

### Answer: `opus` / `baseline`

Both `matcher.go` and `labels_matcher.go` are identical files. Here's the answer
from `labels_matcher.go`:

### MatchType enum (line 22-30)

```go
type MatchType int

const (
    MatchEqual     MatchType = iota  // "="
    MatchNotEqual                     // "!="
    MatchRegexp                       // "=~"
    MatchNotRegexp                    // "!~"
)
```

Four match types using `iota`: exact equality, not-equal, regex match, and
negated regex match.

### Matcher struct (line 47-53)

```go
type Matcher struct {
    Type  MatchType
    Name  string
    Value string
    re    *FastRegexMatcher  // unexported; only set for regex types
}
```

### Constructor — `NewMatcher` (line 56-70)

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

It sets the basic fields, then **only compiles a `FastRegexMatcher`** when the
type is `MatchRegexp` or `MatchNotRegexp`. For equality types, `re` stays nil.

There's also a convenience wrapper `MustNewMatcher` (line 73) that panics on
error, intended for tests.

### Answer: `opus` / `mcp-only`

There are **4 label matcher types**, defined as a `MatchType` enum in
`labels_matcher.go`:

**Type definition** (`labels_matcher.go:21-22`):

```go
// MatchType is an enum for label matching types.
type MatchType int
```

**Enum values** (`labels_matcher.go:26-29`): | Constant | Operator | |---|---| |
`MatchEqual` | `=` | | `MatchNotEqual` | `!=` | | `MatchRegexp` | `=~` | |
`MatchNotRegexp` | `!~` |

**Matcher struct** (`labels_matcher.go:46-53`):

```go
type Matcher struct {
    Type  MatchType
    Name  string
    Value string
    re    *FastRegexMatcher
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

The constructor sets the basic fields, and for regex-based match types
(`MatchRegexp`, `MatchNotRegexp`), it compiles the value into a
`FastRegexMatcher`. There's also a `MustNewMatcher` (`labels_matcher.go:72-79`)
convenience wrapper that panics on error, intended for tests.

### Answer: `opus` / `mcp-full`

It looks like `labels_matcher.go` and `matcher.go` are duplicates. Here's what's
defined in `labels_matcher.go:14-70`:

### MatchType enum

```go
type MatchType int

const (
    MatchEqual     MatchType = iota  // "="
    MatchNotEqual                     // "!="
    MatchRegexp                       // "=~"
    MatchNotRegexp                    // "!~"
)
```

Four match types using `iota`: exact equality, not-equal, regex match, and
negated regex match. Each has a string representation (`=`, `!=`, `=~`, `!~`)
via the `matchTypeToStr` array and the `String()` method.

### Matcher struct

```go
type Matcher struct {
    Type  MatchType
    Name  string
    Value string
    re    *FastRegexMatcher  // unexported, set only for regex types
}
```

### Constructor — `NewMatcher`

```go
func NewMatcher(t MatchType, n, v string) (*Matcher, error)
```

Creates a `Matcher` with the given type, label name, and value. If the type is
`MatchRegexp` or `MatchNotRegexp`, it compiles a `FastRegexMatcher` from the
value string (returning an error if the regex is invalid). For equality types,
no regex is compiled.

There's also a convenience wrapper `MustNewMatcher` (line 73) that panics on
error, intended for tests only.

### Full Judge Analysis (Opus 4.6)

## Content Quality

All six answers are substantively correct and cover the same core material: the
`MatchType` enum with four values, the `Matcher` struct, and the `NewMatcher`
constructor. The differences are minor. Ranking:

**1. sonnet / mcp-only** — Most complete answer. Includes the `matchTypeToStr`
array which other answers omit, provides accurate line references (21-22, 26-37,
46-53, 55-70, 72-79), shows well-formatted code blocks, and gives a clear
explanation of the regex compilation behavior. The line references are precise
and consistent.

**2. opus / mcp-full** — Equally correct, mentions the `matchTypeToStr` array
and `String()` method (unique detail), and notes the duplicate file issue
(`labels_matcher.go` / `matcher.go`). Slightly less code shown inline
(summarizes the constructor signature rather than showing full body), but the
explanation is clear and accurate. Good line references.

**3. sonnet / mcp-full** — Clean, accurate, precise line references. Doesn't
mention `matchTypeToStr` but otherwise covers everything well. The explanation
of infallible vs fallible construction is a nice touch.

**4. opus / mcp-only** — Uses a table format for the enum values which is a nice
touch for readability. Accurate line references, correct code. Slightly less
detailed explanation than the top answers.

**5. sonnet / baseline** — Correct and complete, includes `MustNewMatcher` with
full code. Line references are present but slightly less precise (e.g., "lines
21–30" vs exact ranges). The code formatting uses a slightly different style for
`NewMatcher` (single-line struct literal vs multi-line), which may or may not
match the source exactly.

**6. opus / baseline** — Correct content, mentions the duplicate file
observation. Line references are single-line (e.g., "line 22-30", "line 56-70")
which is fine. Slightly terser than the others. The code shown for the
constructor uses a condensed struct literal that may not match source formatting
exactly.

All answers are close in quality — the spread is narrow. The main
differentiators are inclusion of `matchTypeToStr`, precision of line references,
and clarity of explanation.

## Efficiency Analysis

| Run                   | Duration  | Total Input Tok | Output Tok | Cost       |
| --------------------- | --------- | --------------- | ---------- | ---------- |
| sonnet / baseline     | 20.7s     | ~56K            | 727        | $0.241     |
| **sonnet / mcp-only** | **10.9s** | **25K**         | **759**    | **$0.144** |
| sonnet / mcp-full     | 13.1s     | ~86K            | 769        | $0.260     |
| opus / baseline       | 14.9s     | ~87K            | 712        | $0.263     |
| **opus / mcp-only**   | **10.4s** | **17.5K**       | **576**    | **$0.102** |
| opus / mcp-full       | 15.9s     | ~88K            | 674        | $0.269     |

**Key observations:**

- **mcp-only is the clear efficiency winner** for both models. Both
  `sonnet/mcp-only` and `opus/mcp-only` are the fastest, cheapest, and
  lowest-token runs in their respective model tiers. The semantic search index
  returned precisely what was needed without requiring extensive file reading or
  exploration.

- **opus/mcp-only is the cheapest run overall** at $0.102 — less than half the
  cost of the baseline and mcp-full variants, while producing a top-4 quality
  answer.

- **mcp-full and baseline are surprisingly similar in cost** (~$0.26 each for
  both models). The mcp-full runs gain the semantic index but still carry the
  overhead of the full MCP toolset, negating the token savings.

- **Cache reads dominate input tokens** for baseline and mcp-full runs (28K-42K
  cached tokens), suggesting repeated context loading. The mcp-only runs avoid
  this entirely.

- **sonnet/baseline is the slowest** at 20.7s despite being a cheaper model,
  likely due to more exploration rounds needed without semantic search.

**Recommendation:** **sonnet / mcp-only** offers the best quality-to-cost
tradeoff — ranked #1 in content quality at $0.144 (second cheapest overall). If
cost is the primary concern, **opus / mcp-only** at $0.102 delivers strong
quality at the lowest price. The baseline and mcp-full variants offer no
meaningful quality advantage to justify their ~2x cost premium.

---

## histogram [medium]

**Question:** How does histogram bucket counting work? Show me the relevant
function signatures.

### Metrics

| Model      | Scenario | Duration | Input Tok | Cache Read | Cache Created | Output Tok | Cost (USD) |
| ---------- | -------- | -------- | --------- | ---------- | ------------- | ---------- | ---------- |
| **sonnet** | baseline | 103.7s   | 42625     | 42156      | 0             | 1022       | $1.55734   |
| **sonnet** | mcp-only | 12.4s    | 16741     | 0          | 0             | 634        | $0.09956   |
| **sonnet** | mcp-full | 13.3s    | 28888     | 28104      | 0             | 571        | $0.17277   |
| **opus**   | baseline | 36.1s    | 122700    | 84690      | 0             | 1639       | $0.69682   |
| **opus**   | mcp-only | 20.4s    | 20398     | 0          | 0             | 885        | $0.12412   |
| **opus**   | mcp-full | 17.9s    | 32614     | 28230      | 0             | 776        | $0.19658   |

### Answer: `sonnet` / `baseline`

Here's how histogram bucket counting works in this codebase:

## Core Iterator Pattern

Bucket counting is driven by two iterator types in `histogram.go`:

**`regularBucketIterator`** — yields individual bucket counts:

```go
func newRegularBucketIterator(spans []Span, buckets []int64, schema int32, positive bool, customValues []float64) regularBucketIterator
func (r *regularBucketIterator) Next() bool
```

Each call to `Next()` advances through spans and accumulates delta-encoded
counts:

```go
r.currCount += r.buckets[r.bucketsIdx]
```

**`cumulativeBucketIterator`** — yields running totals:

```go
func (c *cumulativeBucketIterator) Next() bool
func (c *cumulativeBucketIterator) At() Bucket[uint64]
```

Accumulates a rolling sum across all buckets:

```go
c.currCount += c.h.PositiveBuckets[c.posBucketsIdx]
c.currCumulativeCount += uint64(c.currCount)
```

## Entry Points

```go
func (h *Histogram) PositiveBucketIterator() BucketIterator[uint64]
func (h *Histogram) NegativeBucketIterator() BucketIterator[uint64]
func (h *Histogram) CumulativeBucketIterator() BucketIterator[uint64]
func (h *Histogram) ZeroBucket() Bucket[uint64]
```

## Prometheus Client Side (`prom_histogram.go`)

**Finding which bucket a value falls into:**

```go
func (h *histogram) findBucket(v float64) int
// Linear search for n < 35 buckets, binary search (sort.SearchFloat64s) for n >= 35
```

**Adding to a bucket atomically:**

```go
func addToBucket(buckets *sync.Map, key int, increment int64) bool
// Fast path: existing bucket → atomic add
// Slow path: LoadOrStore to create new bucket
```

**Bucket limit enforcement (compression strategies):**

```go
func (h *histogram) limitBuckets(counts *histogramCounts, value float64, bucket int)
func (h *histogram) maybeWidenZeroBucket(hot, cold *histogramCounts) bool
func (h *histogram) doubleBucketWidth(hot, cold *histogramCounts)
```

## Storage Format

Buckets are **delta-encoded**: `PositiveBuckets []int64` stores differences
between consecutive counts, not absolute values. `makeBuckets()` converts from a
sparse `sync.Map` to this compact spans+deltas format. Validation in
`Validate()` checks that
`sum(NegativeBuckets) + ZeroCount + sum(PositiveBuckets) == Count`.

### Answer: `sonnet` / `mcp-only`

The core of histogram bucket counting lives in `prom_histogram.go:652`.

### How it works

**`histogramCounts.observe(v float64, bucket int, doSparse bool)`** —
`prom_histogram.go:652`

This method handles all bucket counting for a single observation:

1. **Fixed-width (classic) buckets** — if the pre-computed `bucket` index falls
   within the bucket array, it atomically increments that slot:

   ```go
   if bucket < len(hc.buckets) {
       atomic.AddUint64(&hc.buckets[bucket], 1)
   }
   ```

2. **Sparse (native) buckets** — when `doSparse` is true, the value is placed
   into a logarithmic bucket using a schema-dependent key:
   - For `schema > 0`: binary searches the `nativeHistogramBounds` table using
     `math.Frexp` to get the fractional exponent, then maps to a key.
   - For `schema <= 0`: uses the raw exponent, offset-shifted by the schema.
   - Infinities are handled by clamping to `±MaxFloat64` and incrementing the
     key by 1.
   - The key routes to positive buckets, negative buckets, or the zero bucket:
     ```go
     case v > zeroThreshold:  addToBucket(&hc.nativeHistogramBucketsPositive, key, 1)
     case v < -zeroThreshold: addToBucket(&hc.nativeHistogramBucketsNegative, key, 1)
     default:                 atomic.AddUint64(&hc.nativeHistogramZeroBucket, 1)
     ```

3. **Count is incremented last** (atomically) as the signal that the observation
   is complete — this is an intentional ordering guarantee.

**Validation** — `histogram.go:470` — after the fact, `Validate` sums all bucket
counts and checks they match `h.Count`, returning `ErrHistogramCountMismatch` or
`ErrHistogramCountNotBigEnough` if not.

### Answer: `sonnet` / `mcp-full`

The core of histogram bucket counting is in `histogramCounts.observe`. Here are
the relevant function signatures and their roles:

**Primary counting method** — `prom_histogram.go:652`

```go
func (hc *histogramCounts) observe(v float64, bucket int, doSparse bool)
```

This handles both classic and native (sparse) histograms:

- **Classic buckets**: `atomic.AddUint64(&hc.buckets[bucket], 1)` — increments
  the pre-computed bucket index
- **Native/sparse buckets**: computes a bucket `key` using `math.Frexp` +
  schema-based scaling, then routes the value to positive, negative, or zero
  buckets

**Key logic for native histogram key computation:**

- For `schema > 0`: binary searches `nativeHistogramBounds` (fractional bounds)
  then offsets by exponent
- For `schema <= 0`: uses the raw exponent, adjusted by the schema's offset and
  shift

**Validation** — `histogram.go:470`

```go
// (method on Histogram, no explicit signature in chunk)
// sums bucket deltas and checks against h.Count
```

Bucket counts are stored as delta-encoded `int64` slices (`PositiveBuckets`,
`NegativeBuckets`), and validation reconstructs the total by summing deltas and
comparing to `h.Count`.

**Supporting type** — `histogram.go:481`

```go
type regularBucketIterator struct {
    baseBucketIterator[uint64, int64]
}
```

Used for iterating over classic fixed-width buckets.

In short: classic bucket index is pre-computed by the caller and passed in;
native histogram bucket keys are computed on the fly from the value's
floating-point representation and the active schema.

### Answer: `opus` / `baseline`

Those helper functions (`checkHistogramBuckets`, `checkHistogramSpans`,
`checkHistogramCustomBounds`) are called but not defined in the fixture files —
they're likely in a shared utility file in the actual Prometheus codebase. But
the bucket counting logic is fully visible in the fixture code. Here's how it
works:

## How Histogram Bucket Counting Works

The core counting mechanism lives in `Histogram.Validate()`
(`histogram.go:426`). It works by accumulating bucket counts from positive,
negative, and zero buckets, then verifying they match `h.Count`.

### The counting flow:

1. **Delta-encoded buckets** — `PositiveBuckets` and `NegativeBuckets` are
   delta-encoded `[]int64` slices. The first element is an absolute count; each
   subsequent element is a delta relative to the previous one. See
   `histogram.go:70-73`.

2. **Accumulation** — The `regularBucketIterator.Next()`
   (`histogram.go:496-528`) decodes deltas into running counts:

   ```go
   r.currCount += r.buckets[r.bucketsIdx]  // line 524
   ```

   Each call to `Next()` adds the next delta to `currCount`, reconstructing the
   absolute count for that bucket.

3. **Cumulative counting** — The `cumulativeBucketIterator.Next()`
   (`histogram.go:550-607`) builds a running cumulative total:

   ```go
   c.currCount += c.h.PositiveBuckets[c.posBucketsIdx]   // line 590 — decode delta
   c.currCumulativeCount += uint64(c.currCount)            // line 591 — accumulate
   ```

4. **Validation** — `Validate()` (`histogram.go:426-479`) calls
   `checkHistogramBuckets` for both positive and negative sides to sum all
   bucket observations into `nCount` and `pCount`, then checks:
   ```go
   sumOfBuckets := nCount + pCount + h.ZeroCount  // line 467
   ```

   - If `Sum` is NaN: `sumOfBuckets <= h.Count` (lower bound, since NaN
     observations increment Count but not buckets)
   - Otherwise: `sumOfBuckets == h.Count` (exact match required)

### Key function signatures:

| Signature                                                               | Location           | Purpose                                   |
| ----------------------------------------------------------------------- | ------------------ | ----------------------------------------- |
| `func (h *Histogram) Validate() error`                                  | `histogram.go:426` | Validates bucket counts match total Count |
| `func (h *Histogram) PositiveBucketIterator() BucketIterator[uint64]`   | `histogram.go:216` | Iterates positive buckets ascending       |
| `func (h *Histogram) NegativeBucketIterator() BucketIterator[uint64]`   | `histogram.go:223` | Iterates negative buckets descending      |
| `func (h *Histogram) CumulativeBucketIterator() BucketIterator[uint64]` | `histogram.go:232` | Cumulative view of positive buckets       |
| `func (r *regularBucketIterator) Next() bool`                           | `histogram.go:496` | Decodes delta-encoded bucket counts       |
| `func (c *cumulativeBucketIterator) Next() bool`                        | `histogram.go:550` | Accumulates cumulative bucket counts      |
| `func (h *Histogram) ZeroBucket() Bucket[uint64]`                       | `histogram.go:201` | Returns the zero bucket                   |

### Spans control which buckets exist

The `Span` struct (`histogram.go:83-89`) maps the sparse bucket layout. Each
span has an `Offset` (gap from previous span or starting index) and `Length`
(number of consecutive buckets). The iterators walk spans to determine which
bucket indices are populated, skipping gaps between spans.

### Answer: `opus` / `mcp-only`

Histogram bucket counting works through a two-level system: **classic fixed
buckets** and **native sparse buckets**.

## Core Observation Flow

**`histogramCounts.observe`** (`prom_histogram.go:652-706`) is the main entry
point. When a value is observed:

1. It increments the classic fixed bucket via
   `atomic.AddUint64(&hc.buckets[bucket], 1)`
2. If native (sparse) histograms are enabled, it computes a bucket key from the
   value using `math.Frexp` and the schema, then routes to positive, negative,
   or zero buckets
3. Increments count last as a "completion signal"

**`histogram.observe`** (`prom_histogram.go:899-912`) wraps this, selecting the
hot counts (double-buffered) and calling `limitBuckets` afterward.

## Key Function Signatures

```go
// Main observe - routes to correct bucket, handles sparse + classic
func (hc *histogramCounts) observe(v float64, bucket int, doSparse bool)

// Outer observe - picks hot counts, triggers bucket limiting
func (h *histogram) observe(v float64, bucket int)

// Atomically adds to a sparse bucket (sync.Map); returns true if bucket was created
func addToBucket(buckets *sync.Map, key int, increment int64) bool

// Limits bucket count via reset, zero-bucket widening, or resolution halving
func (h *histogram) limitBuckets(counts *histogramCounts, value float64, bucket int)

// Validates bucket population sum matches the count field
func validateCount(sum float64, count uint64, negativeBuckets, positiveBuckets map[int]int64, zeroBucket uint64) error

// Transfers counts from cold→hot buckets and resets cold to 0
func addAndReset(hotBuckets *sync.Map, bucketNumber *uint32) func(k, v interface{}) bool
```

## Bucket Key Calculation (inside `histogramCounts.observe`)

For a given value `v` and schema:

- **Positive schema** (`schema > 0`): uses precomputed bounds table —
  `key = sort.SearchFloat64s(bounds, frac) + (exp-1)*len(bounds)`
- **Zero/negative schema**: `key = exp` (adjusted by `frac == 0.5`), then
  right-shifted by `-schema`
- **Infinity**: pretends `v = MaxFloat64`, then increments key by 1
- Values within the zero threshold go to `nativeHistogramZeroBucket` instead

## Bucket Limiting Strategy (`limitBuckets`, `prom_histogram.go:914-958`)

When `nativeHistogramMaxBuckets` is exceeded, three strategies are tried in
order:

1. `maybeReset` — full reset if enough time has passed
2. `maybeWidenZeroBucket` — absorb near-zero buckets into the zero bucket
3. `doubleBucketWidth` — halve the schema (reduce resolution)

### Answer: `opus` / `mcp-full`

Here's how histogram bucket counting works in this codebase:

**Core observation flow:**

1. **`histogram.observe`** (`prom_histogram.go:899-912`) — entry point.
   Increments the hot counter atomically, delegates to
   `histogramCounts.observe`, then calls `limitBuckets` if sparse buckets are
   enabled.

2. **`histogramCounts.observe`** (`prom_histogram.go:652-706`) — the main bucket
   counting logic:
   - Increments the classic `buckets[bucket]` counter atomically
   - For native (sparse) histograms, computes a bucket key from the value using
     `math.Frexp` and the schema:
     - **Positive schema** (`schema > 0`): binary-searches
       `nativeHistogramBounds` to find the sub-bucket, then combines with the
       exponent
     - **Zero/negative schema**: uses the exponent directly, shifted by schema
   - Routes the value to `nativeHistogramBucketsPositive`,
     `nativeHistogramBucketsNegative`, or `nativeHistogramZeroBucket` based on
     sign vs. zero threshold
   - Increments `count` last as a completion signal

3. **`addToBucket`** (`prom_histogram.go:1532-1549`) — concurrent-safe sparse
   bucket increment using `sync.Map`. Fast path loads existing bucket; slow path
   allocates a new `*int64` via `LoadOrStore`. Returns `true` if a new bucket
   was created (to track `nativeHistogramBucketsNumber`).

**Bucket limiting:**

4. **`histogram.limitBuckets`** (`prom_histogram.go:914-958`) — when
   `nativeHistogramMaxBuckets` is exceeded, tries strategies in order: reset,
   widen zero bucket, or double bucket width (halve resolution).

**Validation:**

5. **`validateCount`** (`prom_histogram.go:1871-1889`) — sums all bucket
   populations and checks they match `count` (or `<=` if sum is NaN).

6. **`Histogram.Validate`** (`histogram.go:470-479`) — similar check on the
   model-layer `Histogram` struct.

**Iteration:**

7. **`cumulativeBucketIterator`** (`histogram.go:530-548`) — iterates buckets
   producing cumulative counts, tracking empty gaps between spans.
8. **`regularBucketIterator`** / `newRegularBucketIterator`
   (`histogram.go:481-494`) — iterates non-cumulative buckets using spans and
   delta-encoded bucket counts.

### Full Judge Analysis (Opus 4.6)

Now I have full ground truth. Here's my evaluation:

---

## Content Quality

### Ranking: 1st through 6th

**1. opus / mcp-full** — The most well-organized and accurate answer. It
correctly identifies the full observation flow from `histogram.observe`
(line 899) → `histogramCounts.observe` (line 655) → `addToBucket` (line 1534),
with accurate line numbers throughout. It covers the classic vs. native bucket
distinction, bucket limiting strategies, validation via both `validateCount` and
`Histogram.Validate`, and the iteration layer (`regularBucketIterator`,
`cumulativeBucketIterator`). Every file:line reference I checked was correct or
off by only the comment-vs-signature difference (e.g., saying 652 for the
comment block when the `func` line is 655 — a minor quibble). The answer is
concise and well-structured without unnecessary padding.

**2. opus / mcp-only** — Excellent coverage with accurate function signatures
and line numbers. It uniquely includes the `validateCount` signature (line 1871)
and `addAndReset`, which are genuinely relevant. The bucket key calculation
explanation is detailed and correct (schema > 0 uses binary search on bounds,
schema <= 0 uses exponent). The bucket limiting strategy section accurately
describes the three-step cascade (reset → widen zero → double width). Line
references are accurate. Slightly less well-organized than opus/mcp-full but
covers more ground.

**3. opus / baseline** — Strong on the `histogram.go` side (iterators, Validate,
Spans), with mostly accurate line references. The table of function signatures
is a nice touch and references are correct (e.g., `ZeroBucket` at 201,
`PositiveBucketIterator` at 216, `regularBucketIterator.Next` at 496). However,
it completely misses the `prom_histogram.go` side — there's no mention of
`histogramCounts.observe`, `addToBucket`, `findBucket`, or `limitBuckets`. The
opening sentence about helper functions being "called but not defined" is
slightly confused. Still, what it covers is accurate and well-explained (delta
encoding, spans, cumulative counting).

**4. sonnet / baseline** — Covers both files, which is good. The
`regularBucketIterator` and `cumulativeBucketIterator` descriptions are correct,
and the `prom_histogram.go` section identifies `findBucket`, `addToBucket`, and
`limitBuckets`. However, line numbers are entirely absent (no file:line
references at all), and the `findBucket` description is slightly imprecise — it
says "linear search for n < 35" but the actual code uses a more nuanced
early-exit pattern. The claim about `makeBuckets()` converting from `sync.Map`
to spans+deltas format is plausible but wasn't verified. The entry point
signatures are correct.

**5. sonnet / mcp-only** — Focused and accurate on `histogramCounts.observe`
with a correct line reference (652 for the comment block). The explanation of
classic vs. sparse bucket routing is correct, and the code snippets match the
source. However, it's notably incomplete — it doesn't cover the iteration layer
(`regularBucketIterator`, `cumulativeBucketIterator`) at all, and barely touches
validation. The `Validate` reference says "histogram.go:470" which is within the
function body, not the signature at line 426. It's a good focused answer but
doesn't fully address the "show me relevant function signatures" part of the
question.

**6. sonnet / mcp-full** — The shortest answer and it shows. It correctly
identifies `histogramCounts.observe` at line 652 and the core logic, but then
gets vague. The "Validation" section says "no explicit signature in chunk" which
is wrong — `Validate` has a clear signature at line 426. The
`regularBucketIterator` struct reference is correct but incomplete (no mention
of `Next()` or `cumulativeBucketIterator`). It misses `addToBucket`,
`limitBuckets`, `findBucket`, and the full observation flow. The concluding
sentence is useful but the answer feels rushed.

---

## Efficiency Analysis

| Scenario        | Duration | Input Tokens | Output Tokens | Cost  |
| --------------- | -------- | ------------ | ------------- | ----- |
| sonnet/baseline | 103.7s   | 42,625       | 1,022         | $1.56 |
| sonnet/mcp-only | 12.4s    | 16,741       | 634           | $0.10 |
| sonnet/mcp-full | 13.3s    | 28,888       | 571           | $0.17 |
| opus/baseline   | 36.1s    | 122,700      | 1,639         | $0.70 |
| opus/mcp-only   | 20.4s    | 20,398       | 885           | $0.12 |
| opus/mcp-full   | 17.9s    | 32,614       | 776           | $0.20 |

**Key observations:**

- **Baseline is dramatically more expensive.** Sonnet/baseline costs 15.6x more
  than sonnet/mcp-only and takes 8.4x longer. Opus/baseline costs 5.6x more than
  opus/mcp-only. The baseline approach reads raw files into context, burning
  tokens on irrelevant code.

- **MCP-only vs MCP-full:** MCP-only is consistently cheaper (~40% less) than
  MCP-full with similar or better quality for opus. The cache read tokens in
  MCP-full suggest it's loading additional context (conversation history or
  CLAUDE.md) that doesn't proportionally improve results.

- **Opus dominates quality-adjusted efficiency.** Opus/mcp-only at $0.12
  produces a top-2 answer (better than sonnet/baseline at $1.56). Opus/mcp-full
  at $0.20 produces the best answer overall.

- **Surprising finding:** Sonnet/baseline is the most expensive run ($1.56) yet
  ranks 4th in quality. The 103.7s runtime and 42K input tokens suggest it's
  doing extensive file reading without the semantic search to guide it
  efficiently.

**Recommended best tradeoff:** **opus/mcp-full** at $0.20 — produces the highest
quality answer with accurate line references, complete coverage of both files,
and good organization. If cost is the primary constraint, **opus/mcp-only** at
$0.12 is nearly as good. The baseline approaches offer no advantage at any price
point.

---

## tsdb-compaction [hard]

**Question:** How does TSDB compaction work end-to-end? Explain the Compactor
interface, LeveledCompactor, and how the DB triggers compaction. Show relevant
types, interfaces, and key method signatures.

### Metrics

| Model      | Scenario | Duration | Input Tok | Cache Read | Cache Created | Output Tok | Cost (USD) |
| ---------- | -------- | -------- | --------- | ---------- | ------------- | ---------- | ---------- |
| **sonnet** | baseline | 120.4s   | 30099     | 28104      | 0             | 1954       | $1.69490   |
| **sonnet** | mcp-only | 87.8s    | 305968    | 0          | 0             | 4529       | $1.64306   |
| **sonnet** | mcp-full | 45.2s    | 80082     | 56208      | 0             | 2328       | $0.48671   |
| **opus**   | baseline | 189.8s   | 32841     | 28230      | 0             | 1950       | $2.61725   |
| **opus**   | mcp-only | 53.2s    | 70935     | 0          | 0             | 2570       | $0.41892   |
| **opus**   | mcp-full | 93.2s    | 33914     | 28230      | 0             | 1611       | $0.64117   |

### Answer: `sonnet` / `baseline`

Here's a complete breakdown of TSDB compaction end-to-end:

---

## Compactor Interface

`compact.go:52-77`

```go
type Compactor interface {
    Plan(dir string) ([]string, error)
    Write(dest string, b BlockReader, mint, maxt int64, base *BlockMeta) ([]ulid.ULID, error)
    Compact(dest string, dirs []string, open []*Block) ([]ulid.ULID, error)
}
```

- **`Plan`** — scans a directory and returns sets of block paths that are safe
  to compact concurrently.
- **`Write`** — persists a single `BlockReader` as a new on-disk block (used for
  head flushing, not multi-block merges).
- **`Compact`** — merges multiple existing blocks into one new block.

---

## LeveledCompactor

`compact.go:79-93`

```go
type LeveledCompactor struct {
    metrics                     *CompactorMetrics
    logger                      *slog.Logger
    ranges                      []int64   // time range per compaction level, e.g. [2h, 6h, 24h, ...]
    chunkPool                   chunkenc.Pool
    ctx                         context.Context
    maxBlockChunkSegmentSize    int64
    mergeFunc                   storage.VerticalChunkSeriesMergeFunc
    blockExcludeFunc            BlockExcludeFilterFunc
    postingsEncoder             index.PostingsEncoder
    enableOverlappingCompaction bool
    // ...
}
```

The `ranges` slice defines the compaction level boundaries. A block at level N
spans `ranges[N]` milliseconds.

### Key method flow

**`Plan(dir)`** `compact.go:248`

1. Reads all block dirs and loads their `BlockMeta`.
2. Runs `blockExcludeFunc` to skip user-excluded blocks.
3. Calls internal `plan()` → `selectDirs()` to find blocks that together fill a
   complete time range for a level, or `selectOverlappingDirs()` for overlapping
   blocks.
4. Returns the chosen directory paths.

**`Compact(dest, dirs, open)`** → **`CompactWithBlockPopulator`**
`compact.go:489`

1. Opens (or reuses from `open`) each source block.
2. Allocates a fresh `ulid.ULID` for the output block.
3. Calls `CompactBlockMetas(uid, metas...)` to produce merged metadata:
   - Combines all `Sources` ULIDs.
   - Increments `Compaction.Level`.
   - Records all source blocks as `Compaction.Parents`.
   - Time range spans `min(mint)…max(maxt)`.
4. Calls `write()` to merge series/chunks/index to disk.
5. If the resulting block has 0 samples: marks source dirs `Deletable`, returns
   empty slice.

**`Write(dest, b, mint, maxt, base)`** `compact.go:586` Wraps a single
`BlockReader` into a level-1 block. Used when flushing the in-memory head to
disk.

---

## CompactorMetrics

`compact.go:95-103`

```go
type CompactorMetrics struct {
    Ran               prometheus.Counter    // total compactions run
    PopulatingBlocks  prometheus.Gauge      // 1 while writing a block
    OverlappingBlocks prometheus.Counter
    Duration          prometheus.Histogram
    ChunkSize         prometheus.Histogram
    ChunkSamples      prometheus.Histogram
    ChunkRange        prometheus.Histogram
}
```

---

## How DB triggers compaction

`db.go:283` — the `DB` struct holds:

```go
type DB struct {
    dir       string
    compactor Compactor
    compactc  chan struct{}   // signal: "please compact"
    stopc     chan struct{}
    cmtx      sync.Mutex     // serialises compactions and deletions
    blocks    []*Block        // currently open on-disk blocks
    head      *Head
    // ...
}
```

A background goroutine (started at `Open`) waits on `compactc`. When signalled
it calls **`compact()`**, which orchestrates:

1. **Flush head** — if `head.compactable()` (enough data accumulated), calls
   `compactor.Write()` to persist it as a new level-0 block, then reloads
   blocks.
2. **`compactBlocks()`** `db.go:1721` — loop:

   ```go
   for {
       if db.head.compactable() && !db.waitingForCompactionDelay() {
           // head needs flushing — abort block compactions and go back to step 1
           return nil
       }
       plan, err := db.compactor.Plan(db.dir)
       if len(plan) == 0 { break }

       uids, err := db.compactor.Compact(db.dir, plan, db.blocks)
       db.reloadBlocks()   // opens new block, closes/deletes old ones
   }
   ```

   Keeps compacting until `Plan` returns nothing.

3. **Delete old blocks** — `reloadBlocks()` GCs any blocks flagged `Deletable`.

---

## End-to-end summary

```
WAL/Head accumulates samples
         │
         ▼  (head.compactable())
  compactor.Write()  →  level-0 block on disk
         │
         ▼  (Plan finds N level-0 blocks filling a 2h range)
  compactor.Compact() →  level-1 block  (sources marked Deletable)
         │
         ▼  (Plan finds N level-1 blocks filling a 6h range)
  compactor.Compact() →  level-2 block
         │
        ...
         ▼  reloadBlocks() deletes source blocks
```

Each compaction level doubles (or multiplies by the `ranges` config) the time
span, keeping the total block count bounded as data ages.

### Answer: `sonnet` / `mcp-only`

Now I have a complete picture of the end-to-end compaction flow. Here's the full
explanation:

---

## TSDB Compaction End-to-End

### 1. The `Compactor` Interface (`compact.go:52-77`)

```go
type Compactor interface {
    Plan(dir string) ([]string, error)
    Write(dest string, b BlockReader, mint, maxt int64, base *BlockMeta) ([]ulid.ULID, error)
    Compact(dest string, dirs []string, open []*Block) ([]ulid.ULID, error)
}
```

Three methods with distinct roles:

- **`Plan`** — decides _which_ blocks to compact (returns directories)
- **`Write`** — flushes a single in-memory source (e.g., the Head) to a new
  on-disk block
- **`Compact`** — merges multiple existing on-disk blocks into one

---

### 2. `LeveledCompactor` (`compact.go:79-93`)

```go
type LeveledCompactor struct {
    ranges    []int64   // e.g. [2h, 12h, 48h, ...] — the compaction levels
    metrics   *CompactorMetrics
    chunkPool chunkenc.Pool
    mergeFunc storage.VerticalChunkSeriesMergeFunc
    enableOverlappingCompaction bool
    // ...
}
```

The `ranges` slice defines **compaction levels**. Blocks at level N span
`ranges[N]` milliseconds; when enough of them accumulate, they're merged into a
level N+1 block.

#### `Plan` (`compact.go:248-277`) — selecting candidates

```go
func (c *LeveledCompactor) Plan(dir string) ([]string, error)
```

Reads all block metadata under `dir`, then calls the internal
`plan(dms []dirMeta)` which (in priority order):

1. **Overlapping blocks** (`selectOverlappingDirs`) — if any blocks have
   overlapping time ranges, they're merged first.
2. **Level compaction** (`selectDirs`) — splits blocks into buckets that span
   each range level; picks the first bucket at any level that has `>1` block
   covering the full range interval.
3. **Tombstone cleanup** — if a block has `>5%` deleted series and spans at
   least half the configured range, it's rewritten alone to reclaim space.

The most-recent block is always excluded from consideration (gives a window for
incremental backup).

#### `Compact` (`compact.go:483`, `489-577`)

```go
func (c *LeveledCompactor) Compact(dest string, dirs []string, open []*Block) ([]ulid.ULID, error)
// delegates to:
func (c *LeveledCompactor) CompactWithBlockPopulator(dest string, dirs []string, open []*Block, blockPopulator BlockPopulator) ([]ulid.ULID, error)
```

- Opens each source block (reusing already-open ones to avoid double memory
  mapping)
- Calls `CompactBlockMetas` to build the merged `BlockMeta` (compaction level =
  max(sources)+1, parent ULIDs recorded)
- Calls internal `write()` to materialize the result

#### `Write` (`compact.go:586-636`)

```go
func (c *LeveledCompactor) Write(dest string, b BlockReader, mint, maxt int64, base *BlockMeta) ([]ulid.ULID, error)
```

Used specifically to flush the **in-memory Head** to disk. Sets compaction level
= 1.

#### Internal `write()` (`compact.go:657-769`)

Both `Write` and `Compact` converge here:

1. Creates a temp directory `<ulid>.tmp.for-creation`
2. Opens a `ChunkWriter` and an `index.Writer`
3. Calls `blockPopulator.PopulateBlock()` — iterates all series (via
   `MergeChunkSeriesSet` for multi-block merges), writes chunks and index
   entries
4. Writes `meta.json` and an empty tombstones file
5. **Atomically renames** the temp dir to the final block directory
   (`fileutil.Replace`)

---

### 3. How `DB` Triggers Compaction

#### Trigger: `dbAppender.Commit` (`db.go:1360-1372`)

Every time an appender commits, it checks if the Head is compactable:

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

`compactc` is a `chan struct{}` that signals the background loop.

#### Background loop: `DB.run` (`db.go:1175-1243`)

```go
func (db *DB) run(ctx context.Context) {
    for {
        select {
        case <-time.After(db.opts.BlockReloadInterval):
            db.reloadBlocks()
            db.compactc <- struct{}{}  // periodic trigger
            // stale series compaction check...
        case <-db.compactc:
            if db.autoCompact {
                db.Compact(ctx)  // full compaction cycle
            }
        }
    }
}
```

Two sources trigger `compactc`: appender commits (reactive) and the periodic
reload interval (proactive).

#### `DB.Compact` — the orchestrator (`db.go:1410-1506`)

```go
func (db *DB) Compact(ctx context.Context) error {
    db.cmtx.Lock()  // serializes all compaction
    // 1. Flush Head blocks (highest priority)
    for db.head.compactable() {
        rh := NewRangeHeadWithIsolationDisabled(db.head, mint, maxt-1)
        db.head.WaitForAppendersOverlapping(rh.MaxTime())
        db.compactHead(rh)       // Write() + reloadBlocks() + truncateMemory()
        lastBlockMaxt = maxt
    }
    db.head.truncateWAL(lastBlockMaxt)
    // 2. Flush OOO Head (if out-of-order writes enabled)
    db.compactOOOHead(ctx)
    // 3. Compact on-disk blocks
    return db.compactBlocks()
}
```

#### `DB.compactBlocks` — the Plan→Compact→Reload loop (`db.go:1719-1763`)

```go
func (db *DB) compactBlocks() error {
    for {
        if db.head.compactable() { break }  // yield to Head compaction
        plan, _ := db.compactor.Plan(db.dir)
        if len(plan) == 0 { break }
        uids, _ := db.compactor.Compact(db.dir, plan, db.blocks)
        db.reloadBlocks()  // swaps db.blocks, deletes source blocks
    }
}
```

---

### End-to-End Flow Summary

```
Appender.Commit()
    └─ head.compactable()? ──yes──▶ compactc ◀──── periodic ticker
                                        │
                                   DB.run loop
                                        │
                                   DB.Compact(ctx)
                                    ├─ compactHead(RangeHead)
                                    │     └─ compactor.Write()  ← Head→Block
                                    │           └─ write() [temp dir → atomic rename]
                                    │     └─ reloadBlocks()
                                    │     └─ head.truncateMemory()
                                    │     └─ head.truncateWAL()
                                    ├─ compactOOOHead()
                                    │     └─ compactor.Write() × N ranges
                                    │     └─ reloadBlocks()
                                    └─ compactBlocks()
                                          loop:
                                          compactor.Plan(dir)   ← selectDirs/selectOverlapping
                                          compactor.Compact(dir, plan, blocks)
                                               └─ write() [merge N blocks → 1]
                                          reloadBlocks()        ← swaps blocks, deletes parents
```

**Key design points:**

- `cmtx` mutex serializes all compaction; `compactc` channel is non-blocking
  (drops signal if one is already pending)
- Head compaction (WAL→block) always takes priority over on-disk block merging
- `reloadBlocks()` handles both loading new blocks and deleting source blocks
  (using parent ULID tracking in `meta.json`) — crash-safe because parent ULIDs
  are checked on every reload
- Blocks are written to a `.tmp.for-creation` directory then atomically renamed,
  so partial writes are never visible

### Answer: `sonnet` / `mcp-full`

Here's the end-to-end explanation of TSDB compaction:

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

Three operations:

- **`Plan`** — scans a directory and returns which block dirs are ready to
  compact
- **`Write`** — persists a single `BlockReader` (e.g. the in-memory Head) as a
  new on-disk block; used for head flush
- **`Compact`** — merges multiple existing on-disk block dirs into one new block

---

### 2. `LeveledCompactor` (`compact.go:79-93`)

```go
type LeveledCompactor struct {
    ranges      []int64   // e.g. [2h, 6h, 24h, ...] — hierarchical time windows
    chunkPool   chunkenc.Pool
    mergeFunc   storage.VerticalChunkSeriesMergeFunc
    enableOverlappingCompaction bool
    // ...
}
```

Created via `NewLeveledCompactor` / `NewLeveledCompactorWithOptions`. The
`ranges` slice is the key: each level is typically 3-4× the previous, e.g.
`[7200000, 21600000, 86400000]` ms.

#### `Plan` (`compact.go:248-328`)

`Plan` → `plan(dms)` does three things in priority order:

1. **Overlapping blocks first** (`selectOverlappingDirs`): If any blocks have
   overlapping time ranges, return them all — fixes data integrity issues
   immediately.
2. **Level-based compaction** (`selectDirs`): Sorts blocks by `MinTime`,
   partitions them into windows of each range size, and returns a group that
   fills or precedes the high-water mark of that level. This is the "leveled"
   part — small blocks get merged into medium, medium into large.
3. **Tombstone cleanup**: If a large block has >5% tombstones (deleted series),
   return it solo for rewriting without the deleted data.

#### `Compact` → `CompactWithBlockPopulator` (`compact.go:483-577`)

```go
func (c *LeveledCompactor) Compact(dest string, dirs []string, open []*Block) ([]ulid.ULID, error)
```

For each dir: opens (or reuses already-open) `*Block`, collects `BlockMeta`s.
Calls `CompactBlockMetas` to compute merged metadata, then calls
`c.write(dest, meta, ...)` which:

- Iterates merged series via `BlockPopulator.PopulateBlock`
- Writes chunks, index, and tombstones files
- If result has 0 samples: marks source dirs `Deletable` instead of writing

#### `Write` (`compact.go:586-636`)

```go
func (c *LeveledCompactor) Write(dest string, b BlockReader, mint, maxt int64, base *BlockMeta) ([]ulid.ULID, error)
```

Used exclusively for head→disk flush. Sets `Compaction.Level = 1` (brand-new
block). Same `c.write` path underneath.

---

### 3. How `DB` Triggers Compaction

#### Background loop (`db.go:1175-1243`)

```go
func (db *DB) run(ctx context.Context) {
    // ...
    case <-db.compactc:       // signal channel
        if db.autoCompact {
            db.Compact(ctx)   // main entry point
        }
    // ...
}
```

The `compactc` channel is signalled on a `BlockReloadInterval` timer tick, and
also explicitly when the head becomes compactable.

#### `DB.Compact` (`db.go:1410-1506`) — the orchestrator

```
DB.Compact(ctx)
  │
  ├─ Loop while head.compactable():
  │    compactHead(RangeHead)  ← flush head chunk range to disk block
  │         └─ compactor.Write(...)
  │         └─ reloadBlocks()
  │         └─ head.truncateMemory(maxt)
  │
  ├─ head.truncateWAL(lastBlockMaxt)
  │
  ├─ compactOOOHead(ctx)       ← out-of-order head flush (if any)
  │
  └─ compactBlocks()           ← merge on-disk blocks
```

#### `DB.compactBlocks` (`db.go:1719-1763`)

```go
func (db *DB) compactBlocks() error {
    for {
        plan, _ := db.compactor.Plan(db.dir)   // what to merge?
        if len(plan) == 0 { break }

        uids, _ := db.compactor.Compact(db.dir, plan, db.blocks)  // merge
        db.reloadBlocks()                      // reload + GC old blocks
    }
}
```

Loops until `Plan` returns nothing. Each iteration: plan → compact → reload. Old
blocks are deleted on `reloadBlocks` if their parent info shows they've been
superseded.

---

### Summary Flow

```
Head in-memory
    │  (head.compactable() = true when 1.5× chunkRange old)
    ▼
compactor.Write() → Level-1 block on disk
    │
    ▼  (LeveledCompactor.Plan finds ≥2 blocks fitting a range window)
compactor.Compact() → merged Level-2+ block
    │
    ▼  (repeat until Plan returns nil)
DB.reloadBlocks() → delete superseded blocks
```

The `ranges` config (e.g. `[2h, 6h, 24h, 72h]`) creates a pyramid: 3 × 2h blocks
→ 1 × 6h block → 4 × 6h → 1 × 24h, etc.

### Answer: `opus` / `baseline`

## TSDB Compaction End-to-End

### 1. The `Compactor` Interface (`compact.go:54-77`)

```go
type Compactor interface {
    Plan(dir string) ([]string, error)                                              // decide what blocks to compact
    Write(dest string, b BlockReader, mint, maxt int64, base *BlockMeta) ([]ulid.ULID, error) // flush head → disk block
    Compact(dest string, dirs []string, open []*Block) ([]ulid.ULID, error)          // merge existing on-disk blocks
}
```

Three responsibilities: **Plan** which blocks need merging, **Write** the
in-memory head to a new level-1 block, and **Compact** multiple on-disk blocks
into one.

---

### 2. `LeveledCompactor` (`compact.go:80-93`)

```go
type LeveledCompactor struct {
    metrics                     *CompactorMetrics
    logger                      *slog.Logger
    ranges                      []int64            // e.g. [2h, 6h, 18h, 54h] — exponential tiers
    chunkPool                   chunkenc.Pool
    ctx                         context.Context
    mergeFunc                   storage.VerticalChunkSeriesMergeFunc
    enableOverlappingCompaction bool
    // ...
}
```

The `ranges` slice (built by `ExponentialBlockRanges`) defines the leveled
hierarchy. Blocks are promoted through increasing time-span tiers.

**Key methods:**

| Method                                    | What it does                                                                                                                                           |
| ----------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `Plan(dir)`                               | Reads `meta.json` from each block subdir, calls internal `plan()`                                                                                      |
| `plan(dms)`                               | 1) Check for overlapping blocks → vertical compaction. 2) Otherwise `selectDirs` for level promotion. 3) Check for >5% tombstones → cleanup compaction |
| `selectDirs(ds)`                          | For each range tier, bucket blocks by `splitByRange`. Pick the first group that fills a complete interval                                              |
| `Compact(dest, dirs, open)`               | Delegates to `CompactWithBlockPopulator` — opens/reuses source blocks, calls `CompactBlockMetas` (union meta, `level = max+1`), then `write()`         |
| `Write(dest, b, mint, maxt, base)`        | Builds a fresh `BlockMeta{Level:1}`, calls internal `write()` — used for head flush                                                                    |
| `write(dest, meta, populator, blocks...)` | Creates `<uid>.tmp-for-creation/`, populates chunks+index via `BlockPopulator`, writes `meta.json`, atomic renames to final location                   |

`CompactBlockMetas` merges metadata: `MinTime`/`MaxTime` are the union,
`Level = max(all) + 1`, `Sources` is the deduplicated union of all source ULIDs,
`Parents` records each input as a `BlockDesc`.

---

### 3. How the DB Triggers Compaction

**Signal path:** `Appender.Commit()` checks `head.compactable()` and sends on
`db.compactc`:

```go
func (a dbAppender) Commit() error {
    err := a.Appender.Commit()
    if a.db.head.compactable() {
        select { case a.db.compactc <- struct{}{}: default: }
    }
    return err
}
```

**Head is compactable** when its time range exceeds 1.5× the chunk range
(`head.go:1795`):

```go
func (h *Head) compactable() bool {
    return h.MaxTime()-h.MinTime() > h.chunkRange.Load()/2*3
}
```

**The event loop** `db.run()` also fires on a periodic timer
(`BlockReloadInterval`, ~1 min):

```go
select {
case <-time.After(db.opts.BlockReloadInterval):
    db.reloadBlocks()
    db.compactc <- struct{}{}
case <-db.compactc:
    if db.autoCompact { db.Compact(ctx) }
case <-db.stopc:
    return
}
```

**`DB.Compact(ctx)`** (`db.go:1414-1506`) runs three phases under `cmtx` lock:

1. **Phase 1 — Head compaction loop:** While `head.compactable()`, create a
   `RangeHead(mint, maxt-1)`, wait for overlapping appenders to finish, then
   `compactor.Write()` to flush head to a level-1 block. Reload blocks, truncate
   memory.

2. **Phase 2 — OOO head compaction:** If phase 1 ran, also compact the
   out-of-order head by windowing it into block-sized chunks and calling
   `compactor.Write()` for each.

3. **Phase 3 — On-disk block merging:** Loop calling `compactor.Plan()` then
   `compactor.Compact()`. Aborts early if the head becomes compactable again
   (head has priority).

---

### End-to-End Flow

```
Appender.Commit()
  └─ head.compactable()? ──yes──► compactc channel
                                       │
                                       ▼
                         db.run() → db.Compact(ctx)
                         ├── Phase 1: Head → disk
                         │   └── compactor.Write(dir, RangeHead, mint, maxt)
                         │       └── write() → tmp dir → PopulateBlock → atomic rename
                         │       └── reloadBlocks() + truncateMemory()
                         ├── Phase 2: OOO head → disk (same Write path, per time window)
                         └── Phase 3: Block merging loop
                             ├── compactor.Plan(dir)
                             │   └── overlapping? → vertical compaction
                             │   └── else selectDirs → leveled promotion
                             └── compactor.Compact(dir, plan, blocks)
                                 └── CompactBlockMetas (level++) → write() → new merged block
```

### Answer: `opus` / `mcp-only`

Here's the end-to-end explanation of TSDB compaction:

---

## 1. The `Compactor` Interface (`compact.go:52-77`)

The top-level abstraction with three methods:

```go
type Compactor interface {
    Plan(dir string) ([]string, error)
    Write(dest string, b BlockReader, mint, maxt int64, base *BlockMeta) ([]ulid.ULID, error)
    Compact(dest string, dirs []string, open []*Block) ([]ulid.ULID, error)
}
```

- **`Plan`** — scans the data directory and returns block directories eligible
  for compaction
- **`Write`** — persists a single `BlockReader` (e.g. the in-memory Head) to a
  new on-disk block
- **`Compact`** — merges multiple existing on-disk blocks into one new block

## 2. `LeveledCompactor` — the implementation (`compact.go:79-93`)

```go
type LeveledCompactor struct {
    ranges                      []int64          // time ranges per compaction level
    chunkPool                   chunkenc.Pool
    mergeFunc                   storage.VerticalChunkSeriesMergeFunc
    enableOverlappingCompaction bool
    blockExcludeFunc            BlockExcludeFilterFunc
    // ... metrics, logger, postings encoder, etc.
}
```

The `ranges` slice is the key parameter — it defines the leveled compaction
strategy (e.g. `[2h, 6h, 18h, 54h]`). Blocks get progressively merged into
larger time ranges.

### Planning (`compact.go:248-328`)

`Plan` → reads block metadata from disk → delegates to `plan(dms)` which applies
three strategies in priority order:

1. **Overlapping blocks** (`selectOverlappingDirs`, `compact.go:369-394`) — if
   `enableOverlappingCompaction` is true, any blocks with overlapping time
   ranges are selected first (vertical compaction).

2. **Leveled range compaction** (`selectDirs`, `compact.go:330-367`) — iterates
   through `ranges[1:]`, calls `splitByRange` to group blocks into aligned time
   windows, and picks groups with >1 block that either span the full range or
   are older than the newest block.

3. **Tombstone cleanup** — blocks with >5% tombstones (or 100% deleted series)
   are compacted to reclaim space.

The most recent block is always excluded from leveled compaction to give backup
tools a window.

### Compacting (`compact.go:483-584`)

`Compact` delegates to `CompactWithBlockPopulator`:

1. Opens each block directory (reusing already-open `*Block` when possible)
2. Merges metadata via `CompactBlockMetas` (union of time ranges, incremented
   compaction level)
3. Calls `write()` which creates a temp dir, opens chunk + index writers, calls
   `BlockPopulator.PopulateBlock` to merge all series, then atomically renames
   into place
4. If the result has 0 samples, source blocks are marked `Deletable` and no
   block is produced

### Writing head to block (`compact.go:586-636`)

`Write` creates a level-1 block from a `BlockReader` (typically a `RangeHead`),
sets up parent info in the compaction metadata, and delegates to the same
`write()` internal method.

## 3. How the DB Triggers Compaction

### The `DB` struct (`db.go:281-342`)

Key fields:

```go
type DB struct {
    compactor      Compactor
    compactc       chan struct{}   // signal channel
    autoCompact    bool
    compactCancel  context.CancelFunc
    cmtx           sync.Mutex     // serializes compactions and deletions
    // ...
}
```

### Initialization (`db.go:990-1059`)

During `open()`, either `opts.NewCompactorFunc` is called or a default
`LeveledCompactor` is created via `NewLeveledCompactorWithOptions`.

### Three trigger paths:

**a) Appender commit** (`db.go:1360-1372`)

```go
func (a dbAppender) Commit() error {
    err := a.Appender.Commit()
    if a.db.head.compactable() {
        select {
        case a.db.compactc <- struct{}{}:
        default:
        }
    }
    return err
}
```

Every time samples are committed, if the head is compactable, a signal is sent
on `compactc`.

**b) Periodic reload** (`db.go:1175-1243`)

The `DB.run` goroutine fires on `BlockReloadInterval` ticks — it reloads blocks,
then sends a compaction signal. It also checks for stale series ratio and can
trigger `CompactStaleHead` early.

**c) `db.compactc` consumer** (also in `DB.run`)

```go
case <-db.compactc:
    if db.autoCompact {
        if err := db.Compact(ctx); err != nil {
            backoff = exponential(backoff, 1*time.Second, 1*time.Minute)
        }
    }
```

### `DB.Compact` — the main orchestrator (`db.go:1410-1506`)

Runs in three phases:

1. **Head compaction loop** — while `head.compactable()` is true (head has
   enough data beyond chunkRange/2):
   - Optionally waits for `CompactionDelay`
   - Creates a `RangeHead` with `[mint, maxt-1]`
   - Waits for overlapping appenders to finish
   - Calls `db.compactHead(rh)` which uses `compactor.Write()` to persist the
     head range to a block, reloads blocks, then truncates head memory

2. **OOO head compaction** — if the head was compacted, also compacts
   out-of-order data via `db.compactOOOHead(ctx)`

3. **Block compaction** (`db.compactBlocks`, `db.go:1719-1763`) — loops calling
   `compactor.Plan()` → `compactor.Compact()` → `reloadBlocks()` until no more
   plans exist. Aborts early if the head becomes compactable again (head takes
   priority).

### `compactHead` (`db.go:1634-1662`)

```go
func (db *DB) compactHead(head *RangeHead) error {
    uids, err := db.compactor.Write(db.dir, head, head.MinTime(), head.BlockMaxTime(), nil)
    // ... reload blocks, truncate head memory, rebuild symbol table
}
```

---

## End-to-end flow summary

```
Samples appended → dbAppender.Commit()
    → head.compactable()? → signal compactc
        → DB.run picks up signal
            → DB.Compact()
                1. Head → compactHead → compactor.Write → new level-1 block on disk
                2. OOO head → compactOOOHead
                3. compactBlocks loop:
                    → compactor.Plan (overlapping? leveled? tombstones?)
                    → compactor.Compact → merge blocks → new higher-level block
                    → reloadBlocks (cleans up old blocks via parent info)
```

### Answer: `opus` / `mcp-full`

## TSDB Compaction: End-to-End

### The `Compactor` Interface

**`compact.go:52-77`** — Three core methods:

```go
type Compactor interface {
    Plan(dir string) ([]string, error)
    Write(dest string, b BlockReader, mint, maxt int64, base *BlockMeta) ([]ulid.ULID, error)
    Compact(dest string, dirs []string, open []*Block) ([]ulid.ULID, error)
}
```

- **`Plan`** — scans block directories and returns a set that should be
  compacted together
- **`Write`** — persists an in-memory block (typically the Head) to disk as a
  new block
- **`Compact`** — merges multiple on-disk blocks into fewer, larger blocks

---

### `LeveledCompactor`

**`compact.go:79-93`** — The production implementation:

```go
type LeveledCompactor struct {
    metrics                     *CompactorMetrics
    logger                      *slog.Logger
    ranges                      []int64           // exponential time ranges for levels
    chunkPool                   chunkenc.Pool
    ctx                         context.Context
    mergeFunc                   storage.VerticalChunkSeriesMergeFunc
    blockExcludeFunc            BlockExcludeFilterFunc
    enableOverlappingCompaction bool
    // ... other fields
}
```

**Key methods:**

| Method                             | Purpose                                                                                                                                                            |
| ---------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `Plan(dir)`                        | Reads block metadata, delegates to internal `plan()`                                                                                                               |
| `plan(dms)`                        | Checks for overlapping blocks first (vertical compaction), then falls back to leveled (horizontal) compaction. Also handles tombstone cleanup (>5% deleted series) |
| `selectDirs(ds)`                   | Picks blocks for leveled compaction based on `ranges`                                                                                                              |
| `selectOverlappingDirs(ds)`        | Detects and selects time-overlapping blocks for vertical compaction                                                                                                |
| `Compact(dest, dirs, open)`        | Merges selected blocks into new ones                                                                                                                               |
| `Write(dest, b, mint, maxt, base)` | Persists a single block (typically Head → disk)                                                                                                                    |

---

### How the DB Triggers Compaction

The `DB` struct (`db.go:281-344`) holds a `compactc chan struct{}` channel and
runs a background goroutine:

**Trigger sources** (all non-blocking sends to `compactc`):

1. **Appender commit** (`db.go:1360-1402`) — after every `Commit()`, checks
   `head.compactable()` and signals
2. **Periodic timer** in `db.run()` (`db.go:1175-1243`) — fires every
   `BlockReloadInterval` (~1 min)
3. **Stale series** — if stale ratio exceeds threshold, calls
   `CompactStaleHead()` immediately

**The background loop** (`db.run`):

```
select {
case <-timer:      → reloadBlocks(), then send to compactc
case <-compactc:   → call db.Compact(ctx)
case <-stopc:      → exit
}
```

---

### `DB.Compact()` — The Four Phases

**`db.go:1414-1506`** — holds `cmtx` lock for the entire duration:

```
Phase 1: COMPACT HEAD
  Loop while head.compactable():
    → Create RangeHead for time window
    → Wait for overlapping appenders
    → compactor.Write() to persist head block to disk

Phase 2: TRUNCATE WAL
  → head.truncateWAL(lastBlockMaxt)

Phase 3: COMPACT OUT-OF-ORDER HEAD
  → db.compactOOOHead(ctx)

Phase 4: COMPACT EXISTING BLOCKS (db.compactBlocks)
  Loop:
    → compactor.Plan(db.dir)     // get plan
    → compactor.Compact(plan)    // merge blocks
    → db.reloadBlocks()          // update in-memory state
    (abort early if head becomes compactable again — head takes priority)
```

---

### Flow Diagram

```
Appender.Commit() ──┐
Periodic timer ─────┤──→ compactc channel ──→ db.run() ──→ DB.Compact()
Stale series ───────┘                                          │
                                                    ┌──────────┼──────────┐
                                                    ▼          ▼          ▼
                                              compactHead  compactOOO  compactBlocks
                                              (Write)      (OOO head) (Plan→Compact)
                                                    │          │          │
                                                    └──────────┴──────────┘
                                                               │
                                                        reloadBlocks()
```

**Synchronization:** `cmtx` mutex ensures compactions and deletions never run
concurrently. The `compactc` channel is buffered/non-blocking so triggers never
block the appender hot path. An optional `CompactionDelay` prevents aggressive
compaction during rapid ingestion bursts.

### Full Judge Analysis (Opus 4.6)

## Content Quality

### Ranking: Best to Worst

**1. opus / mcp-only**

This answer is the most complete and precise. It correctly identifies all three
`Compactor` interface methods with accurate signatures, provides the
`LeveledCompactor` struct with correct fields, and details the planning strategy
(overlapping → leveled → tombstone cleanup) with correct priority ordering. The
trigger path coverage is excellent — it identifies all three trigger sources
(appender commit, periodic timer, stale series) and accurately describes the
four phases of `DB.Compact()` including OOO head compaction. Line references
(`compact.go:52-77`, `compact.go:79-93`, `db.go:1360-1402`, `db.go:1175-1243`,
`db.go:1414-1506`, `db.go:1719-1763`) are consistently precise. The
`compactHead` code snippet is a nice concrete addition. It achieved this with
the lowest cost of any run ($0.42), making excellent use of semantic search
without wasted effort.

**2. sonnet / mcp-only**

The most detailed answer overall. It covers everything opus/mcp-only does and
adds more: the internal `write()` method flow (`compact.go:657-769`) with temp
directory naming convention (`.tmp.for-creation`), atomic rename details, the
`head.compactable()` condition, and crash-safety properties. The end-to-end flow
diagram is the most detailed of all answers. Line references are precise and
numerous. The "key design points" section at the end adds valuable architectural
insight (cmtx serialization, head priority, parent ULID tracking). The slight
knock is verbosity — it's longer than needed, and some details (like the exact
temp dir naming) are implementation minutiae rather than architectural
understanding. But content-wise it's extremely strong.

**3. opus / baseline**

Impressively complete for a baseline (no MCP tools) run. Covers the interface,
`LeveledCompactor` struct, planning strategies, `CompactBlockMetas` behavior,
all trigger paths, and the three-phase `Compact` flow. Includes the
`head.compactable()` threshold formula (`chunkRange/2*3`), which is a useful
detail most others omit. The table format for `LeveledCompactor` methods is
clean. The flow diagram is clear. Line references are present but slightly less
precise than the MCP-assisted answers (e.g., `compact.go:54-77` vs the correct
`52-77`). The `ExponentialBlockRanges` mention is a good detail. Minor: it says
"three phases" but there are actually four (WAL truncation is a separate step).

**4. sonnet / mcp-full**

Correct and well-structured but noticeably less detailed than the top three. It
covers the core interface, `LeveledCompactor`, planning strategies (overlapping
→ leveled → tombstone), and the DB trigger/compaction flow. The table format for
key methods is clean. However, it omits: the `dbAppender.Commit` trigger code,
the `head.compactable()` condition, OOO head details, WAL truncation, and
crash-safety discussion. The flow diagram is the simplest. Line references are
present (`compact.go:248-328`, `db.go:1719-1763`) but fewer. It feels like a
competent summary rather than a deep dive.

**5. opus / mcp-full**

Similar depth to sonnet/mcp-full but with a cleaner structure. Correctly
identifies all four phases of `DB.Compact()` (including WAL truncation as phase
2, which others sometimes miss). The trigger sources table is well-organized.
However, the `LeveledCompactor` method descriptions are more superficial —
`selectDirs` and `selectOverlappingDirs` get one-line descriptions rather than
the algorithmic detail other answers provide. The flow diagram is simple but
effective. Mentions `CompactionDelay` which is a nice detail. Line references
are accurate. Overall correct but less illuminating than the top three.

**6. sonnet / baseline**

Correct on fundamentals but has the most minor inaccuracies and gaps. The
`LeveledCompactor` struct listing includes a `ctx context.Context` field and
`maxBlockChunkSegmentSize` which are correct but the line reference
(`compact.go:79-93`) doesn't fully match the expanded listing. The
`compactBlocks` code shows a `waitingForCompactionDelay()` check that may not
exist with that exact name. The flow is mostly right but less precise — it says
"each compaction level doubles (or multiplies by the ranges config)" which is
vague. Missing OOO head compaction entirely. The `CompactorMetrics` section,
while correct, is low-value information that displaces more important details.
The flow diagram is clear but oversimplified.

---

## Efficiency Analysis

| Scenario            | Duration  | Input Tok  | Cache Read | Output Tok | Cost      |
| ------------------- | --------- | ---------- | ---------- | ---------- | --------- |
| sonnet/baseline     | 120.4s    | 30,099     | 28,104     | 1,954      | $1.69     |
| sonnet/mcp-only     | 87.8s     | 305,968    | 0          | 4,529      | $1.64     |
| **sonnet/mcp-full** | **45.2s** | **80,082** | **56,208** | **2,328**  | **$0.49** |
| opus/baseline       | 189.8s    | 32,841     | 28,230     | 1,950      | $2.62     |
| **opus/mcp-only**   | **53.2s** | **70,935** | **0**      | **2,570**  | **$0.42** |
| opus/mcp-full       | 93.2s     | 33,914     | 28,230     | 1,611      | $0.64     |

**Key observations:**

- **opus/mcp-only is the clear winner on quality-to-cost ratio.** Ranked #1 in
  quality at the lowest cost ($0.42) and second-fastest time (53.2s). It used
  semantic search effectively to find exactly what it needed without bloating
  the context.

- **sonnet/mcp-only has a massive input token anomaly** (305,968 tokens with
  zero cache reads). This suggests it read enormous amounts of source code
  during its search, which paradoxically produced the second-best answer but at
  4× the input tokens of comparable runs. The zero cache reads explain why it's
  expensive despite Sonnet's lower per-token rate.

- **Baseline runs are consistently the most expensive** ($1.69 and $2.62). They
  relied on pre-cached knowledge (28K cache reads each) rather than live code
  search, yet produced lower-quality answers. The opus/baseline is the most
  expensive run overall at $2.62 — nearly 6× the cost of opus/mcp-only for a
  worse result.

- **Cache reads correlate with less precision.** The baseline and mcp-full runs
  with high cache reads (28K+) tend to have vaguer line references, suggesting
  cached context provides general knowledge but not the pinpoint accuracy of
  fresh semantic search.

- **sonnet/mcp-full is the speed champion** (45.2s) at a reasonable cost
  ($0.49), but its quality is mid-tier (#4). Good for quick overviews, not for
  deep dives.

**Recommendation:** **opus/mcp-only** offers the best overall tradeoff — highest
quality, lowest cost, fast execution. For budget-conscious scenarios where
slightly less depth is acceptable, **sonnet/mcp-full** at $0.49 and 45s is a
reasonable alternative. The baseline approach (no tools) should be avoided —
it's the most expensive and least accurate pattern across both models.

---

## promql-engine [very-hard]

**Question:** How does PromQL query evaluation work? Explain the evaluation
engine, how functions are registered and called, how the AST nodes are
evaluated, and how alert and recording rules trigger evaluations. Show key
interfaces, types, and function signatures.

### Metrics

| Model      | Scenario | Duration | Input Tok | Cache Read | Cache Created | Output Tok | Cost (USD) |
| ---------- | -------- | -------- | --------- | ---------- | ------------- | ---------- | ---------- |
| **sonnet** | baseline | 139.3s   | 33957     | 28104      | 0             | 3218       | $0.82284   |
| **sonnet** | mcp-only | 96.2s    | 241469    | 0          | 0             | 5708       | $1.35004   |
| **sonnet** | mcp-full | 115.6s   | 617837    | 337248     | 0             | 6126       | $3.41096   |
| **opus**   | baseline | 273.2s   | 33552     | 28230      | 0             | 2602       | $7.15514   |
| **opus**   | mcp-only | 71.9s    | 114751    | 0          | 0             | 3856       | $0.67015   |
| **opus**   | mcp-full | 161.8s   | 34782     | 28230      | 0             | 3546       | $1.82648   |

### Answer: `sonnet` / `baseline`

Here's a comprehensive breakdown of how PromQL query evaluation works in this
codebase:

---

## PromQL Query Evaluation

### Query Engine Interface

`engine.go:124-156` — top-level public interfaces:

```go
type QueryEngine interface {
    NewInstantQuery(ctx context.Context, q storage.Queryable, opts QueryOpts, qs string, ts time.Time) (Query, error)
    NewRangeQuery(ctx context.Context, q storage.Queryable, opts QueryOpts, qs string, start, end time.Time, interval time.Duration) (Query, error)
}

type Query interface {
    Exec(ctx context.Context) *Result
    Close()
    Statement() parser.Statement
    Stats() *stats.Statistics
    Cancel()
    String() string
}
```

---

### Core Evaluator Struct

`engine.go:1138-1152` — the internal evaluator created per query execution:

```go
type evaluator struct {
    startTimestamp int64
    endTimestamp   int64
    interval       int64
    maxSamples               int
    currentSamples           int
    logger                   *slog.Logger
    lookbackDelta            time.Duration
    samplesStats             *stats.QuerySamples
    noStepSubqueryIntervalFn func(rangeMillis int64) int64
    querier                  storage.Querier
}
```

---

### AST Node Types

`ast.go:26-280` — the expression interface and its implementations:

```go
type Expr interface {
    Node
    Type() ValueType
    PromQLExpr()
}
```

Key concrete types:

| Type                             | Purpose                                                      |
| -------------------------------- | ------------------------------------------------------------ |
| `AggregateExpr`                  | `sum`, `count`, `topk`, etc. with grouping                   |
| `BinaryExpr`                     | `+`, `-`, `*`, `/`, `==`, `and`, `or`, `unless`              |
| `Call`                           | Function invocation (`rate(...)`, `histogram_quantile(...)`) |
| `VectorSelector`                 | Metric selector with label matchers                          |
| `MatrixSelector`                 | Selector with time range (`[5m]`)                            |
| `SubqueryExpr`                   | Nested subquery with step                                    |
| `StepInvariantExpr`              | Wraps exprs whose result doesn't change per step             |
| `NumberLiteral`, `StringLiteral` | Constants                                                    |

---

### Top-Level Evaluation: `eval()`

`engine.go:1904-1927` — routes each AST node to a handler via type switch:

```go
func (ev *evaluator) eval(ctx context.Context, expr parser.Expr) (parser.Value, annotations.Annotations) {
    if err := contextDone(ctx, "expression evaluation"); err != nil {
        ev.error(err)
    }
    // OpenTelemetry tracing per node type
    ctx, span := otel.Tracer("").Start(ctx, ...)
    defer span.End()

    switch e := expr.(type) {
    case *parser.AggregateExpr:   ...
    case *parser.Call:            ...
    case *parser.BinaryExpr:      ...
    case *parser.MatrixSelector:  ...
    case *parser.VectorSelector:  ...
    case *parser.StepInvariantExpr: ...
    // ...
    }
}
```

---

### Range Evaluation Loop: `rangeEval()`

`engine.go:1390-1567` — the core loop that drives step-by-step evaluation:

```go
func (ev *evaluator) rangeEval(
    ctx context.Context,
    matching *parser.VectorMatching,
    funcCall func([]Vector, Matrix, [][]EvalSeriesHelper, *EvalNodeHelper) (Vector, annotations.Annotations),
    exprs ...parser.Expr,
) (Matrix, annotations.Annotations) {
    // 1. Pre-evaluate all sub-expressions into matrices
    for i, e := range exprs {
        matrixes[i] = ev.eval(ctx, e).(Matrix)
    }
    // 2. For each timestamp, gather per-step vectors and invoke funcCall
    for ts := ev.startTimestamp; ts <= ev.endTimestamp; ts += ev.interval {
        for i := range exprs {
            vectors[i] = ev.gatherVector(ts, matrixes[i], ...)
        }
        enh.Ts = ts
        result, ws := funcCall(vectors, nil, bufHelpers, enh)
        // 3. Accumulate results into output series
    }
}
```

This is what all node evaluations ultimately delegate to — binary ops, function
calls, aggregations all use `rangeEval` with a custom `funcCall` closure.

---

### Function Registration

`functions.go:2153-2237` — all PromQL functions registered in a single map:

```go
type FunctionCall func(
    vectorVals []Vector,
    matrixVals Matrix,
    args parser.Expressions,
    enh *EvalNodeHelper,
) (Vector, annotations.Annotations)

var FunctionCalls = map[string]FunctionCall{
    "abs":              funcAbs,
    "rate":             funcRate,
    "increase":         funcIncrease,
    "histogram_quantile": funcHistogramQuantile,
    "avg_over_time":    funcAvgOverTime,
    "label_replace":    funcLabelReplace,
    // ~50 total entries
}
```

When `eval()` encounters a `*parser.Call` node, it looks up `e.Func.Name` in
`FunctionCalls` and passes it as the `funcCall` to `rangeEval`.

---

### Binary Expression Evaluation

`engine.go:2288-2327` — dispatches on type combination and operator:

```go
case *parser.BinaryExpr:
    switch lt, rt := e.LHS.Type(), e.RHS.Type(); {
    case lt == ValueTypeScalar && rt == ValueTypeScalar:
        // scalarBinop(op, lhs, rhs)
    case lt == ValueTypeVector && rt == ValueTypeVector:
        switch e.Op {
        case LAND:  ev.VectorAnd(...)
        case LOR:   ev.VectorOr(...)
        default:    ev.VectorBinop(...)
        }
    case lt == ValueTypeVector && rt == ValueTypeScalar:
        ev.VectorscalarBinop(op, v[0], Scalar{V: v[1][0].F}, false, ...)
    case lt == ValueTypeScalar && rt == ValueTypeVector:
        ev.VectorscalarBinop(op, v[1], Scalar{V: v[0][0].F}, true, ...)
    }
```

`VectorBinop` uses label-signature matching (from `VectorMatching`) to pair
samples, then calls `doBinOp`. Cardinality modes: `CardOneToOne`,
`CardManyToOne`, `CardOneToMany` (set operators use `CardManyToMany`).

---

### AST Visitor / Walk

`ast.go:336-399` — used for analysis, type-checking, and query planning (not
evaluation):

```go
type Visitor interface {
    Visit(node Node, path []Node) (w Visitor, err error)
}

func Walk(v Visitor, node Node, path []Node) error {
    if v, err = v.Visit(node, path); v == nil || err != nil {
        return err
    }
    for e := range ChildrenIter(node) {
        Walk(v, e, append(path, node))
    }
    v.Visit(nil, nil) // post-order signal
    return err
}
```

`ChildrenIter` returns child nodes for each concrete type. This drives the
type-checker, label analyzer, and query planner — **not** the actual evaluation
(which uses the recursive `eval()` switch).

---

### Recording Rules

`recording.go:84-122`:

```go
func (rule *RecordingRule) Eval(
    ctx context.Context,
    queryOffset time.Duration,
    ts time.Time,
    query QueryFunc,   // func(ctx, expr string, ts time.Time) (Vector, error)
    _ *url.URL,
    limit int,
) (promql.Vector, error) {
    vector, err := query(ctx, rule.vector.String(), ts.Add(-queryOffset))
    // Relabel: override metric name and extra labels
    for i := range vector {
        lb.Reset(vector[i].Metric)
        lb.Set(labels.MetricName, rule.name)
        rule.labels.Range(func(l labels.Label) { lb.Set(l.Name, l.Value) })
        vector[i].Metric = lb.Labels()
    }
    return vector, nil
}
```

---

### Alerting Rules

`alerting.go:380-399`:

```go
func (r *AlertingRule) Eval(
    ctx context.Context,
    queryOffset time.Duration,
    ts time.Time,
    query QueryFunc,
    externalURL *url.URL,
    limit int,
) (promql.Vector, error) {
    res, err := query(ctx, r.vector.String(), ts.Add(-queryOffset))
    // Drive alert state machine per result sample:
    // StateInactive → StatePending → StateFiring
    // based on holdDuration (the `for:` clause)
}
```

Both rule types share the `QueryFunc` signature:
`func(ctx context.Context, qs string, t time.Time) (promql.Vector, error)`. The
rule group scheduler calls `Eval` periodically and the result flows back into
the TSDB (recording) or alert manager (alerting).

---

### End-to-End Flow Summary

```
Query string
    │
    ▼ parser.ParseExpr()
AST (Expr tree)
    │
    ▼ Walk() — type-check, label analysis
Validated AST
    │
    ▼ evaluator.eval()  ← recursive, type-switch dispatch
    │
    ├── VectorSelector  → storage.Querier.Select() → series scan
    ├── MatrixSelector  → series scan over time range
    ├── AggregateExpr   → rangeEval() + aggregation callback
    ├── BinaryExpr      → rangeEval() + VectorBinop / scalarBinop
    ├── Call            → rangeEval() + FunctionCalls[name]
    └── StepInvariantExpr → eval once, broadcast across steps
         │
         ▼
    Matrix (series × timestamps)
         │
         ▼
    Result{Value, Warnings, Err}
         │
    ┌────┴────────────┐
    ▼                 ▼
RecordingRule    AlertingRule
(write to TSDB)  (fire/resolve alerts)
```

### Answer: `sonnet` / `mcp-only`

Here is a comprehensive explanation of how PromQL query evaluation works in this
codebase (Prometheus).

---

## PromQL Query Evaluation: End-to-End

### 1. Key Interfaces and Types

**`QueryEngine` interface** (`engine.go:124`):

```go
type QueryEngine interface {
    NewInstantQuery(ctx context.Context, q storage.Queryable, opts QueryOpts,
        qs string, ts time.Time) (Query, error)
    NewRangeQuery(ctx context.Context, q storage.Queryable, opts QueryOpts,
        qs string, start, end time.Time, interval time.Duration) (Query, error)
}
```

This is the public interface that callers use. `*Engine` satisfies it, but
anything else can be plugged in (for mocking/wrapping).

**`QueryOpts` interface** (`engine.go:182`):

```go
type QueryOpts interface {
    EnablePerStepStats() bool
    LookbackDelta() time.Duration
}
```

**`evaluator` struct** (`engine.go:1134`):

```go
type evaluator struct {
    startTimestamp int64  // ms
    endTimestamp   int64  // ms
    interval       int64  // ms
    maxSamples     int
    currentSamples int
    lookbackDelta  time.Duration
    querier        storage.Querier
    // ... stats, logger, flags
}
```

This is the internal workhorse. It is not exposed; it is created fresh per
`Exec` call.

---

### 2. Query Lifecycle

#### Step 1 — Create a query object

`Engine.NewInstantQuery` / `Engine.NewRangeQuery` (`engine.go:530`,
`engine.go:550`):

1. Calls `ng.parser.ParseExpr(qs)` to produce a `parser.Expr` AST.
2. Calls `PreprocessExpr(expr, ...)` to annotate the tree (e.g., wraps constants
   in `StepInvariantExpr`).
3. Wraps everything in a `query` struct containing a `parser.EvalStmt` (start,
   end, interval, lookbackDelta).

#### Step 2 — Execute

`query.Exec(ctx)` (`engine.go:247`) → `ng.exec(ctx, q)` (`engine.go:669`):

```
exec()
  → switch q.Statement().(type)
      case *parser.EvalStmt → ng.execEvalStmt(ctx, q, s)
```

#### Step 3 — Prepare and dispatch

`execEvalStmt` (`engine.go:772`):

1. Opens a `storage.Querier` for the time range `[mint, maxt]`.
2. Calls `ng.populateSeries(...)` — walks the AST and attaches matching series
   from storage to each `VectorSelector`/`MatrixSelector` node.
3. Builds an `evaluator` struct.
4. Calls `evaluator.Eval(ctx, s.Expr)`.
5. Post-processes the returned `Matrix` into `Vector`, `Scalar`, or `Matrix`
   depending on `s.Expr.Type()`.

---

### 3. AST Evaluation: The `eval` Switch

`evaluator.Eval` (`engine.go:1189`) is the public entry point; it calls
`ev.eval(ctx, expr)` and optionally runs `cleanupMetricLabels` when delayed name
removal is on.

`ev.eval` (`engine.go:1904`) is a giant type switch on `parser.Expr`:

| AST node                                          | Evaluation                                                                                                                                                 |
| ------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `*parser.AggregateExpr`                           | Recursively evals sub-expression, then calls `rangeEvalAgg` or `rangeEval` with aggregation logic.                                                         |
| `*parser.Call`                                    | Looks up `FunctionCalls[e.Func.Name]` and dispatches. Range-vector functions iterate series step-by-step; instant-vector functions go through `rangeEval`. |
| `*parser.BinaryExpr`                              | Scalar–scalar: `rangeEval` + `scalarBinop`. Vector–vector: `VectorAnd/Or/Unless/Binop`. Vector–scalar: `VectorscalarBinop`.                                |
| `*parser.VectorSelector`                          | Calls `ev.evalSeries(...)` which extracts float/histogram samples per step.                                                                                |
| `*parser.MatrixSelector`                          | Calls `ev.matrixSelector(...)` which fills a `Matrix` with windowed samples.                                                                               |
| `*parser.SubqueryExpr`                            | Spawns a nested `evaluator` for the sub-range, copies results.                                                                                             |
| `*parser.StepInvariantExpr`                       | Evaluates once, then replicates across all steps.                                                                                                          |
| `*parser.UnaryExpr`                               | Negates float values; drops metric name.                                                                                                                   |
| `*parser.ParenExpr`                               | Transparent pass-through.                                                                                                                                  |
| `*parser.NumberLiteral` / `*parser.StringLiteral` | Returns constant value.                                                                                                                                    |

#### `rangeEval` — the step loop (`engine.go:1384`)

Most nodes ultimately go through `rangeEval`:

```go
func (ev *evaluator) rangeEval(
    ctx context.Context,
    matching *parser.VectorMatching,
    funcCall func([]Vector, Matrix, [][]EvalSeriesHelper, *EvalNodeHelper) (Vector, annotations.Annotations),
    exprs ...parser.Expr,
) (Matrix, annotations.Annotations)
```

It:

1. Evaluates each sub-expression's full time range into a `Matrix`.
2. Loops `for ts := startTimestamp; ts <= endTimestamp; ts += interval`.
3. At each step, slices out the per-step `Vector` from each matrix
   (`gatherVector`).
4. Calls `funcCall(vectors, ..., enh)` — the actual operation (binary op,
   aggregation, built-in function, etc.).
5. Accumulates results into output series by hash.

---

### 4. Function Registration and Dispatch

**`FunctionCall` type** (`functions.go:39`):

```go
type FunctionCall func(
    vectorVals []Vector,
    matrixVals Matrix,
    args parser.Expressions,
    enh *EvalNodeHelper,
) (Vector, annotations.Annotations)
```

**`FunctionCalls` map** (`functions.go:2152`) — a static
`map[string]FunctionCall`:

```go
var FunctionCalls = map[string]FunctionCall{
    "abs":              funcAbs,
    "rate":             funcRate,
    "histogram_quantile": funcHistogramQuantile,
    "avg_over_time":    funcAvgOverTime,
    "label_replace":    nil,  // handled specially in eval()
    // ~80+ entries
}
```

**Dispatch** in `eval()` for `*parser.Call` (`engine.go:1956`):

```go
case *parser.Call:
    call := FunctionCalls[e.Func.Name]
    // Special cases first:
    switch e.Func.Name {
    case "label_replace": return ev.evalLabelReplace(...)
    case "label_join":    return ev.evalLabelJoin(...)
    case "info":          return ev.evalInfo(...)
    }
    // Range-vector argument path: iterate series manually, call call(...)
    if matrixArg { /* per-series loop */ }
    // Instant-vector path:
    return ev.rangeEval(ctx, nil, func(v []Vector, ...) {
        return call(v, nil, e.Args, enh)
    }, e.Args...)
```

Range-vector functions (e.g. `rate`, `avg_over_time`) receive a pre-filled
`inMatrix[0]` with the window's samples for each series at each step.

---

### 5. Alert and Recording Rules

Rules don't use the engine directly — they go through a `QueryFunc`:

```go
// rules/manager.go (inferred from usage)
type QueryFunc func(ctx context.Context, q string, t time.Time) (promql.Vector, error)
```

#### Recording rules (`recording.go:84`)

```go
func (rule *RecordingRule) Eval(
    ctx context.Context,
    queryOffset time.Duration,
    ts time.Time,
    query QueryFunc,
    _ *url.URL,
    limit int,
) (promql.Vector, error) {
    vector, err := query(ctx, rule.vector.String(), ts.Add(-queryOffset))
    // Rewrite metric name + labels on each sample
    // Check for duplicate labelsets
    return vector, nil
}
```

`QueryFunc` is typically `EngineQueryFunc` which internally calls:

```
engine.NewInstantQuery(...) → Query.Exec(ctx)
```

#### Alerting rules

Same signature:
`AlertingRule.Eval(ctx, queryOffset, ts, query, externalURL, limit)`. It calls
`query(...)` to get a `Vector`, then compares results against alert states to
fire/resolve alerts.

#### Rule group scheduling

A `Group` runs on a ticker (its `interval`). On each tick it iterates its rules
in order and calls `rule.Eval(...)`. This is entirely outside the engine — the
engine only knows how to evaluate a single `EvalStmt`; rules/alerting owns the
scheduling loop.

---

### Summary: The Call Chain

```
RuleGroup tick
  └─ rule.Eval(ctx, queryOffset, ts, queryFunc, ...)
       └─ queryFunc(ctx, exprString, ts)          ← EngineQueryFunc
            └─ engine.NewInstantQuery(...)
                 └─ query.Exec(ctx)
                      └─ ng.exec(ctx, q)
                           └─ ng.execEvalStmt(ctx, q, stmt)
                                ├─ populateSeries()     ← attach storage series to AST
                                └─ evaluator.Eval(ctx, expr)
                                     └─ ev.eval(ctx, expr)   ← big type switch
                                          ├─ AggregateExpr → rangeEvalAgg
                                          ├─ Call → FunctionCalls[name](...)
                                          ├─ BinaryExpr → VectorBinop / scalarBinop
                                          ├─ VectorSelector → evalSeries
                                          └─ MatrixSelector → matrixSelector
```

The engine is stateless between queries — all mutable state lives in the
short-lived `evaluator` struct. Each query gets a fresh evaluator with its own
sample budget (`maxSamples`/`currentSamples`) and stats counters.

### Answer: `sonnet` / `mcp-full`

Now I have a thorough picture. Here's the full breakdown:

---

## PromQL Query Evaluation

### 1. Entry Point: `QueryEngine` Interface

`engine.go:124`

```go
type QueryEngine interface {
    NewInstantQuery(ctx, q storage.Queryable, opts QueryOpts, qs string, ts time.Time) (Query, error)
    NewRangeQuery(ctx, q storage.Queryable, opts QueryOpts, qs string, start, end time.Time, interval time.Duration) (Query, error)
}
```

The concrete `*Engine` implements this. Queries are created via these two
methods, then executed by calling `query.Exec(ctx)`.

---

### 2. Engine Execution: `exec` → `execEvalStmt`

`engine.go:669` — `Engine.exec` is the main driver:

- Applies a timeout via `context.WithTimeout`
- Queues the query through `queueActive` (concurrency limiting)
- Logs the query on completion

`engine.go:824` — `Engine.execEvalStmt`:

- For **instant queries**: creates an `evaluator` with
  `startTimestamp == endTimestamp`, calls `evaluator.Eval`, then converts the
  resulting `Matrix` into a `Vector` or `Scalar`
- For **range queries**: creates an `evaluator` spanning the full time range and
  calls `evaluator.Eval` directly, returning a `Matrix`

---

### 3. The `evaluator` Struct

`engine.go:1134`

```go
type evaluator struct {
    startTimestamp int64
    endTimestamp   int64
    interval       int64         // step in ms
    maxSamples     int
    lookbackDelta  time.Duration
    querier        storage.Querier
    // ...
}
```

The public entry point is:

`engine.go:1189`

```go
func (ev *evaluator) Eval(ctx context.Context, expr parser.Expr) (v parser.Value, ws annotations.Annotations, err error)
```

It wraps `ev.eval(ctx, expr)` with panic recovery and optional label cleanup.

---

### 4. AST Node Dispatch: `eval` switch

`engine.go:1904` — `eval` is a recursive type-switch over `parser.Expr`:

| AST Node                                            | Action                                                                                             |
| --------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| `*parser.AggregateExpr`                             | Recursively evals the sub-expression, then calls `rangeEvalAgg`                                    |
| `*parser.Call`                                      | Looks up the `FunctionCall` in `FunctionCalls` map, dispatches to `rangeEval` or range-matrix path |
| `*parser.BinaryExpr`                                | Routes to `VectorAnd/Or/Unless` or arithmetic via `rangeEval` with `scalarBinop`                   |
| `*parser.UnaryExpr`                                 | Evals sub-expression, negates floats and histograms                                                |
| `*parser.ParenExpr`                                 | Transparently delegates: `ev.eval(ctx, e.Expr)`                                                    |
| `*parser.MatrixSelector` / `*parser.VectorSelector` | Fetches from `storage.Querier`, returns time series data                                           |
| `*parser.SubqueryExpr`                              | Creates a child `evaluator` with adjusted time range                                               |
| `*parser.StepInvariantExpr`                         | Evaluates once, duplicates result across all steps                                                 |

---

### 5. Function Registration & Dispatch

`functions.go:60`

```go
type FunctionCall func(vectorVals []Vector, matrixVals Matrix, args parser.Expressions, enh *EvalNodeHelper) (Vector, annotations.Annotations)
```

`functions.go:2152`

```go
var FunctionCalls = map[string]FunctionCall{
    "abs":                funcAbs,
    "rate":               funcRate,
    "histogram_quantile": funcHistogramQuantile,
    "avg_over_time":      funcAvgOverTime,
    // ~60+ entries
    "label_replace":      nil,  // handled specially
    "label_join":         nil,  // handled specially
    "info":               nil,  // handled specially
}
```

**Dispatch at `engine.go:1962`:**

```go
case *parser.Call:
    call := FunctionCalls[e.Func.Name]  // look up implementation

    // Special cases first
    switch e.Func.Name {
    case "label_replace": return ev.evalLabelReplace(ctx, e.Args)
    case "label_join":    return ev.evalLabelJoin(ctx, e.Args)
    case "info":          return ev.evalInfo(ctx, e.Args)
    }

    if !matrixArg {
        // Instant-vector functions: each step gets a Vector
        return ev.rangeEval(ctx, nil, func(v []Vector, ..., enh *EvalNodeHelper) (Vector, annotations.Annotations) {
            return call(v, nil, e.Args, enh)
        }, e.Args...)
    }
    // Range-vector functions (rate, avg_over_time, etc.):
    // iterate over the MatrixSelector's series, pass windows to call()
```

`EvalNodeHelper` (`engine.go:1210`) is a per-node cache passed to every
`FunctionCall` — holds the current timestamp, a reusable output `Vector`, and
caches for histogram/binary matching state.

---

### 6. Alert & Recording Rules

Both rule types share the same `QueryFunc` signature (defined implicitly via
usage in `recording.go:85` and `alerting.go:382`):

```go
type QueryFunc func(ctx context.Context, q string, t time.Time) (promql.Vector, error)
```

**Recording rules** (`recording.go:84`):

```go
func (rule *RecordingRule) Eval(ctx, queryOffset, ts, query QueryFunc, _, limit) (promql.Vector, error) {
    vector, err := query(ctx, rule.vector.String(), ts.Add(-queryOffset))
    // Override __name__ label and rule labels on each sample
    // Check for duplicate labelsets
    return vector, nil
}
```

Results are written back as new time series.

**Alerting rules** (`alerting.go:382`):

```go
func (r *AlertingRule) Eval(ctx, queryOffset, ts, query QueryFunc, externalURL, limit) (promql.Vector, error) {
    res, err := query(ctx, r.vector.String(), ts.Add(-queryOffset))
    // For each result sample → create/update Alert{State: StatePending}
    // For existing active alerts no longer in results → mark StateInactive
    // Transition: StatePending → StateFiring when ts - ActiveAt >= holdDuration
    // Keep resolved alerts for resolvedRetention (15 min) for Alertmanager reliability
    return vec, nil  // vec contains ALERTS / ALERTS_FOR_STATE series
}
```

State machine: `StateUnknown` → `StatePending` → `StateFiring` → `StateInactive`

---

### Summary Flow

```
query.Exec(ctx)
  └─ Engine.exec()
       └─ Engine.execEvalStmt()
            └─ evaluator.Eval(expr)
                 └─ evaluator.eval(expr)  ← recursive AST dispatch
                      ├─ *AggregateExpr  → rangeEvalAgg()
                      ├─ *Call           → FunctionCalls[name](vectors, matrix, args, enh)
                      ├─ *BinaryExpr     → rangeEval() + VectorAnd/Or/scalarBinop
                      ├─ *VectorSelector → storage.Querier.Select()
                      └─ *MatrixSelector → storage.Querier.Select() (with range)

Rules (evaluated by rule manager on each interval):
  AlertingRule.Eval(ts, QueryFunc) → StatePending → StateFiring
  RecordingRule.Eval(ts, QueryFunc) → write new series
```

### Answer: `opus` / `baseline`

## How PromQL Query Evaluation Works

### 1. Evaluation Engine

The engine is built around three key types:

**`QueryEngine` interface** (`engine.go:125`) — the public API:

```go
type QueryEngine interface {
    NewInstantQuery(ctx context.Context, q storage.Queryable, opts QueryOpts, qs string, ts time.Time) (Query, error)
    NewRangeQuery(ctx context.Context, q storage.Queryable, opts QueryOpts, qs string, start, end time.Time, interval time.Duration) (Query, error)
}
```

**`Engine` struct** (`engine.go:344`) — the concrete implementation holding
config (timeout, max samples, lookback delta, parser, etc.).

**`evaluator` struct** (`engine.go:1138`) — the internal workhorse created
per-query:

```go
type evaluator struct {
    startTimestamp, endTimestamp, interval int64  // milliseconds
    maxSamples, currentSamples            int
    lookbackDelta                         time.Duration
    querier                               storage.Querier
    // ...
}
```

**Execution flow:**

1. `Engine.NewInstantQuery` / `NewRangeQuery` — parses the expression string,
   validates, wraps step-invariant subtrees in `StepInvariantExpr`, returns a
   `*query`
2. `query.Exec(ctx)` → `Engine.exec(ctx, q)` → `Engine.execEvalStmt(ctx, q, s)`
3. `execEvalStmt` computes time bounds, acquires a `storage.Querier`, calls
   `populateSeries(...)` which walks the AST and calls `Querier.Select(...)` for
   every `VectorSelector`, then creates an `evaluator` and calls
   `evaluator.Eval(ctx, expr)`

---

### 2. AST Node Types

All expressions implement `Expr` (`ast.go:77`):

```go
type Expr interface {
    Node
    Type() ValueType
    PromQLExpr()
}
```

Key node types:

| Node                              | Example                          | Returns             |
| --------------------------------- | -------------------------------- | ------------------- |
| `VectorSelector`                  | `http_requests_total{job="api"}` | `ValueTypeVector`   |
| `MatrixSelector`                  | `metric[5m]`                     | `ValueTypeMatrix`   |
| `Call`                            | `rate(metric[5m])`               | `e.Func.ReturnType` |
| `AggregateExpr`                   | `sum by (job) (...)`             | `ValueTypeVector`   |
| `BinaryExpr`                      | `a + b`, `a > b`                 | vector or scalar    |
| `SubqueryExpr`                    | `expr[1h:5m]`                    | `ValueTypeMatrix`   |
| `StepInvariantExpr`               | (engine-inserted optimization)   | same as child       |
| `NumberLiteral` / `StringLiteral` | `42`, `"hello"`                  | scalar / string     |

The `Call` node holds a `*Function` descriptor and `Expressions` args:

```go
type Call struct {
    Func     *Function
    Args     Expressions
    PosRange posrange.PositionRange
}
```

---

### 3. Recursive Evaluation — `evaluator.eval()`

The core dispatch is `evaluator.eval(ctx, expr)` (`engine.go:1904`), a large
`switch` on expression type:

- **`*AggregateExpr`** — evaluates the inner expression, then calls
  `rangeEvalAgg(...)` to apply `sum`, `avg`, `topk`, etc. at each timestamp step
- **`*Call`** — looks up `FunctionCalls[e.Func.Name]`, then:
  - If no matrix arg: uses `rangeEval(...)` with a closure calling the function
    at each step
  - If has matrix arg: iterates per-series, per-step calling the function
  - Special-cased: `timestamp`, `label_replace`, `label_join`, `info`
- **`*BinaryExpr`** — dispatches on LHS/RHS types (scalar-scalar, vector-vector
  with AND/OR/UNLESS/arithmetic, vector-scalar)
- **`*VectorSelector`** — expands the `SeriesSet`, calls `evalSeries(...)` to
  read data points
- **`*MatrixSelector`** — calls `matrixSelector(...)`, only valid for instant
  queries
- **`*SubqueryExpr`** — creates a nested `evaluator` with finer interval and
  recurses
- **`*StepInvariantExpr`** — evaluates once at `startTimestamp`, replicates
  across all steps

`rangeEval` (`engine.go:1390`) is the generic step-by-step evaluator: it
iterates over every timestamp from start to end, gathers input vectors via
`gatherVector(...)`, calls the provided function, and accumulates results into a
`Matrix`.

---

### 4. Function Registry

**`FunctionCall` type** (`functions.go:60`):

```go
type FunctionCall func(vectorVals []Vector, matrixVals Matrix, args parser.Expressions, enh *EvalNodeHelper) (Vector, annotations.Annotations)
```

**`FunctionCalls` map** (`functions.go:2153`) — the global registry of ~60
functions:

```go
var FunctionCalls = map[string]FunctionCall{
    "abs":       funcAbs,
    "rate":      funcRate,
    "increase":  funcIncrease,
    "irate":     funcIrate,
    "delta":     funcDelta,
    "histogram_quantile": funcHistogramQuantile,
    "label_replace": nil,  // handled specially in eval
    "label_join":    nil,  // handled specially in eval
    // ...
}
```

The engine resolves calls with `call := FunctionCalls[e.Func.Name]`. Functions
receive an `EvalNodeHelper` (`engine.go:1211`) that provides the current
timestamp (`enh.Ts`), a pre-allocated output `Vector` (`enh.Out`), and various
caches (histogram buckets, label builders, binary matching structures).

---

### 5. Alert and Recording Rules

Both rule types call the same `QueryFunc` to trigger evaluation:

```go
type QueryFunc func(ctx context.Context, qs string, t time.Time) (promql.Vector, error)
```

**`RecordingRule.Eval`** (`recording.go:85`):

- Calls `query(ctx, rule.vector.String(), ts)` to evaluate the PromQL expression
- Rewrites metric names and labels on each sample (sets `MetricName` to the rule
  name, applies rule labels)
- Checks for duplicate labelsets
- Returns the vector to be written to storage

**`AlertingRule.Eval`** (`alerting.go:382`):

- Calls `query(ctx, r.vector.String(), ts)` to evaluate the expression
- For each result sample: expands label/annotation templates, builds alert
  labels, creates or updates the alert in `r.active` map
- Manages the alert state machine:
  - Expression fires → `StatePending` (sets `ActiveAt = ts`)
  - `holdDuration` elapses → `StateFiring` (sets `FiredAt`)
  - Expression stops firing → `StateInactive` (sets `ResolvedAt`); kept 15 min
  - If `keepFiringFor > 0`, a firing alert continues for that duration after
    resolution
- Emits `ALERTS{}` and `ALERTS_FOR_STATE{}` sample points

**Alert states** (`alerting.go:54`):

```go
const (
    StateUnknown  AlertState = iota
    StateInactive
    StatePending
    StateFiring
)
```

---

### End-to-End Path

```
Query string "rate(http_requests_total[5m])"
    │
    ▼ Engine.NewInstantQuery → parse → PreprocessExpr
    │
    ▼ query.Exec → Engine.execEvalStmt
    │   populateSeries: walk AST, Querier.Select for each VectorSelector
    │   create evaluator{start, end, interval}
    │
    ▼ evaluator.eval (recursive)
    │   Call("rate") → FunctionCalls["rate"] = funcRate
    │   MatrixSelector → read range data from storage
    │   rangeEval: for each step, call funcRate(vectors, matrix, args, enh)
    │
    ▼ Matrix result → convert to Vector/Scalar/Matrix per query type
    │
    ▼ Rules layer (optional):
       RecordingRule.Eval → query() → rewrite labels → write to storage
       AlertingRule.Eval  → query() → manage Pending→Firing→Inactive FSM
```

### Answer: `opus` / `mcp-only`

Now I have a comprehensive picture. Here's how PromQL query evaluation works in
this codebase:

---

## PromQL Query Evaluation Architecture

### 1. The Engine — Entry Point

The `Engine` struct (`engine.go:343-361`) is the central coordinator. It manages
query lifetime, concurrency, timeouts, and metrics.

```go
// engine.go:343
type Engine struct {
    logger                   *slog.Logger
    metrics                  *engineMetrics
    timeout                  time.Duration
    maxSamplesPerQuery       int
    activeQueryTracker       QueryTracker
    lookbackDelta            time.Duration
    enableDelayedNameRemoval bool
    parser                   parser.Parser
    // ...
}
```

It exposes queries through the **`QueryEngine` interface**
(`engine.go:124-128`):

```go
type QueryEngine interface {
    NewInstantQuery(ctx context.Context, q storage.Queryable, opts QueryOpts, qs string, ts time.Time) (Query, error)
    NewRangeQuery(ctx context.Context, q storage.Queryable, opts QueryOpts, qs string, start, end time.Time, interval time.Duration) (Query, error)
}
```

Both methods create a `query` struct internally via `Engine.newQuery`
(`engine.go:574-599`), which wraps a `parser.EvalStmt` containing the parsed
expression, time range, and interval.

### 2. Query Execution Flow

The **`Query` interface** (`engine.go:141-156`) is what callers interact with:

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

Execution follows this path:

1. **`query.Exec`** (`engine.go:247-256`) — calls `ng.exec(ctx, q)`
2. **`Engine.exec`** (`engine.go:669-750`) — sets up timeout, logging, tracing,
   queues the query via `ActiveQueryTracker`, then dispatches on statement type:

   ```go
   switch s := q.Statement().(type) {
   case *parser.EvalStmt:
       return ng.execEvalStmt(ctx, q, s)
   case parser.TestStmt:
       return nil, nil, s(ctx)
   }
   ```

   Note the comment: _"Alert and record statements are not handled by the
   Engine."_

3. **`Engine.execEvalStmt`** (`engine.go:772-889`) — the core execution:
   - Creates a `Querier` for the required time range
   - Populates series data from storage (`populateSeries`)
   - Creates an `evaluator` struct
   - For **instant queries** (start == end): creates evaluator with a single
     step
   - For **range queries**: creates evaluator spanning start→end with the given
     interval
   - Calls `evaluator.Eval(ctx, expr)`
   - Converts the result based on expression type (Vector, Scalar, Matrix)

### 3. The Evaluator — AST Tree Walker

The `evaluator` struct (`engine.go:1134-1152`) is the recursive AST visitor:

```go
type evaluator struct {
    startTimestamp int64
    endTimestamp   int64
    interval       int64
    maxSamples     int
    currentSamples int
    lookbackDelta  time.Duration
    // ...
}
```

**`evaluator.Eval`** (`engine.go:1189-1197`) is the public entry point with
panic recovery:

```go
func (ev *evaluator) Eval(ctx context.Context, expr parser.Expr) (v parser.Value, ws annotations.Annotations, err error) {
    defer ev.recover(expr, &ws, &err)
    v, ws = ev.eval(ctx, expr)
    // ...
}
```

**Error handling uses panics** — `evaluator.error()` (`engine.go:1159`) panics,
and `evaluator.recover()` (`engine.go:1164-1187`) catches them, converting
runtime panics, typed errors, and warning-bearing errors into return values.

### 4. AST Node Evaluation — The Big Switch

**`evaluator.eval`** (`engine.go:1904+`) is the recursive workhorse. It
type-switches on `parser.Expr` nodes:

| AST Node                    | Handling                                                                                                                                                                        |
| --------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `*parser.AggregateExpr`     | Sorts grouping labels, delegates to `rangeEvalAgg` (or `rangeEval` for COUNT_VALUES)                                                                                            |
| `*parser.Call`              | Looks up function in `FunctionCalls` map, handles matrix vs. instant args, special-cases `timestamp`, `label_replace`, `label_join`, `info`                                     |
| `*parser.ParenExpr`         | Recursively evaluates inner expression                                                                                                                                          |
| `*parser.UnaryExpr`         | Evaluates inner, negates floats/histograms for SUB                                                                                                                              |
| `*parser.BinaryExpr`        | Dispatches on operand types (scalar×scalar, vector×vector, vector×scalar) using `rangeEval` with appropriate binary op functions (`VectorAnd`, `VectorOr`, `VectorBinop`, etc.) |
| `*parser.NumberLiteral`     | Returns constant value via `rangeEval`                                                                                                                                          |
| `*parser.StringLiteral`     | Returns `String{V: e.Val}` directly                                                                                                                                             |
| `*parser.VectorSelector`    | Expands series set, calls `evalSeries` to read from storage iterators                                                                                                           |
| `*parser.MatrixSelector`    | Calls `matrixSelector` — only valid for instant evaluation                                                                                                                      |
| `*parser.SubqueryExpr`      | Creates a new child `evaluator` with adjusted time range and recursively evaluates                                                                                              |
| `*parser.StepInvariantExpr` | Evaluates once at start time, then duplicates the result across all steps (optimization for expressions that don't change per step)                                             |

### 5. Function Registration and Calling

Functions are registered in a **global map** (`functions.go:2152-2237`):

```go
var FunctionCalls = map[string]FunctionCall{
    "abs":               funcAbs,
    "rate":              funcRate,
    "histogram_quantile": funcHistogramQuantile,
    // ~80+ functions total
}
```

The function signature type (`functions.go:39-60`):

```go
type FunctionCall func(
    vectorVals []Vector,
    matrixVals Matrix,
    args parser.Expressions,
    enh *EvalNodeHelper,
) (Vector, annotations.Annotations)
```

When `eval` hits a `*parser.Call` node, it:

1. Looks up the function: `call := FunctionCalls[e.Func.Name]`
2. Checks if any argument is a `MatrixSelector` (range vector functions like
   `rate`)
3. For **instant vector functions**: wraps in `rangeEval` which gathers vectors
   at each timestamp and calls the function per-step
4. For **range vector functions**: iterates over series, builds a sliding window
   of samples per step, and calls the function with the matrix slice
5. Special functions (`label_replace`, `label_join`, `info`) are handled
   directly without the map

### 6. The `rangeEval` Step Loop

`rangeEval` (`engine.go:1431+`) is the core iteration mechanism for range
queries. For each timestamp from start to end:

1. Gathers input vectors from pre-evaluated matrices
2. Optionally computes join signatures for binary operations (`VectorMatching`)
3. Calls the provided function with the gathered vectors
4. Collects outputs into a result Matrix

### 7. Step-Invariant Optimization

Before evaluation, `preprocessExprHelper` (`engine.go:4246-4341`) walks the AST
and wraps sub-expressions that are step-invariant (e.g., a `VectorSelector` with
`@ start()`) in `StepInvariantExpr` nodes. These are evaluated once and
duplicated across all steps, avoiding redundant computation.

### 8. Alert and Recording Rules

Rules live **outside** the PromQL engine. The engine explicitly does _not_
handle them (`engine.go:673`: _"Alert and record statements are not handled by
the Engine"_).

**`RecordingRule`** (`recording.go:37-54`):

```go
type RecordingRule struct {
    name   string
    vector parser.Expr
    labels labels.Labels
    health *atomic.String
    evaluationTimestamp *atomic.Time
    // ...
}
```

**`RecordingRule.Eval`** (`recording.go:84-122`):

```go
func (rule *RecordingRule) Eval(ctx context.Context, queryOffset time.Duration,
    ts time.Time, query QueryFunc, _ *url.URL, limit int) (promql.Vector, error)
```

It works by:

1. Calling `query(ctx, rule.vector.String(), ts.Add(-queryOffset))` — this is a
   `QueryFunc` that executes the PromQL expression through the engine
2. Overriding metric names and labels on the result vector
3. Checking for duplicate label sets
4. Enforcing series limits

The `QueryFunc` callback is the bridge: rule groups provide a function that
creates and executes a PromQL instant query through the engine, closing the
loop. Rules are evaluated on a schedule by a rule group manager (not in this
file set), which calls `rule.Eval()` at each evaluation interval, passing the
current timestamp and the query function.

### Summary Flow

```
User/API → Engine.NewInstantQuery/NewRangeQuery → query.Exec
  → Engine.exec → Engine.execEvalStmt
    → evaluator.Eval → evaluator.eval (recursive AST walk)
      ├── AggregateExpr  → rangeEvalAgg → aggregation()
      ├── Call           → FunctionCalls[name] via rangeEval
      ├── BinaryExpr     → rangeEval with VectorBinop/VectorAnd/etc.
      ├── VectorSelector → evalSeries (reads from storage)
      ├── MatrixSelector → matrixSelector (reads sliding windows)
      ├── SubqueryExpr   → new child evaluator, recursive eval
      └── StepInvariantExpr → eval once, duplicate across steps
    → convert result (Vector/Scalar/Matrix)

Rules (outside engine):
  RuleGroup.Eval() → RecordingRule.Eval(queryFunc) → queryFunc → Engine instant query
```

### Answer: `opus` / `mcp-full`

## PromQL Query Evaluation — Deep Dive

The Prometheus codebase in your testdata fixtures implements a full PromQL
evaluation engine. Here's how it all fits together.

---

### 1. The Evaluation Engine

The core is the `Engine` struct in `engine.go:345-361`:

```go
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

**Query creation** — two entry points:

| Method            | Signature (`engine.go`)                                                                               | Purpose                                  |
| ----------------- | ----------------------------------------------------------------------------------------------------- | ---------------------------------------- |
| `NewInstantQuery` | `(ctx, Queryable, QueryOpts, qs string, ts time.Time) (Query, error)`                                 | Single-timestamp evaluation              |
| `NewRangeQuery`   | `(ctx, Queryable, QueryOpts, qs string, start, end time.Time, interval time.Duration) (Query, error)` | Evaluation over a time range with a step |

Both parse the PromQL string and produce a `*query` object implementing the
`Query` interface (`engine.go:143-156`):

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

**Execution flow:** `Exec()` → `ng.exec()` → `ng.execEvalStmt()` → creates an
`evaluator` → `evaluator.Eval(expr)`.

The internal `evaluator` struct (`engine.go:1138-1152`) holds the time range,
sample budget, and storage querier:

```go
type evaluator struct {
    startTimestamp int64
    endTimestamp   int64
    interval       int64
    maxSamples     int
    currentSamples int
    lookbackDelta  time.Duration
    querier        storage.Querier
    // ...
}
```

Key evaluator methods:

| Method                                         | Location         | Role                                                          |
| ---------------------------------------------- | ---------------- | ------------------------------------------------------------- |
| `Eval(ctx, expr)`                              | `engine.go:1189` | Top-level entry with panic recovery                           |
| `eval(ctx, expr)`                              | `engine.go:1905` | Recursive AST dispatch (big `switch`)                         |
| `rangeEval(ctx, matching, funcCall, exprs...)` | `engine.go:1390` | Steps through all timestamps, calling `funcCall` at each step |
| `rangeEvalAgg(ctx, aggExpr, ...)`              | `engine.go:1569` | Specialized range eval for aggregations                       |
| `matrixSelector(ctx, node)`                    | `engine.go:2637` | Evaluates `[5m]` range selectors                              |
| `vectorSelectorSingle(it, offset, ts)`         | `engine.go:2536` | Evaluates an instant selector at one timestamp                |

---

### 2. Function Registration and Calling

Functions are defined by the `FunctionCall` type in `functions.go:39-60`:

```go
type FunctionCall func(
    vectorVals []Vector,
    matrixVals Matrix,
    args parser.Expressions,
    enh *EvalNodeHelper,
) (Vector, annotations.Annotations)
```

All 85+ built-in functions are registered in a single map
(`functions.go:2153-2237`):

```go
var FunctionCalls = map[string]FunctionCall{
    "abs":                  funcAbs,
    "rate":                 funcRate,
    "histogram_quantile":   funcHistogramQuantile,
    "sum_over_time":        funcSumOverTime,
    "label_replace":        funcLabelReplace,
    "sort":                 funcSort,
    // ... 80+ more entries
}
```

**Lookup during evaluation** happens in `eval()` at `engine.go:1962`:

```go
case *parser.Call:
    call := FunctionCalls[e.Func.Name]
```

Then `rangeEval` is called with this `call` function, stepping it across every
timestamp in the query range.

**Safety sets** restrict certain functions:

- `AtModifierUnsafeFunctions` (`functions.go:2244`) — functions like `time()`,
  `timestamp()` that vary with eval time, cannot use `@ <timestamp>` modifier
- `AnchoredSafeFunctions` (`functions.go:2257`) — `rate`, `increase`, `delta`,
  etc. that support anchored evaluation
- `SmoothedSafeFunctions` (`functions.go:2268`) — `rate`, `increase`, `delta`
  supporting smoothed mode

---

### 3. AST Nodes

The AST is defined in `ast.go:26-85` with three core interfaces:

```go
type Node interface {
    String() string
    Pretty(level int) string
    PositionRange() posrange.PositionRange
}

type Statement interface { Node; PromQLStmt() }
type Expr interface     { Node; Type() ValueType; PromQLExpr() }
```

**Key expression types:**

| Type                | `ast.go` lines | Example PromQL             | Key Fields                                               |
| ------------------- | -------------- | -------------------------- | -------------------------------------------------------- |
| `VectorSelector`    | 206-235        | `http_requests{job="api"}` | `Name`, `LabelMatchers`, `Offset`, `Timestamp`, `Series` |
| `MatrixSelector`    | 132-139        | `http_requests[5m]`        | `VectorSelector`, `Range`                                |
| `Call`              | 124-129        | `rate(...)`                | `Func *Function`, `Args Expressions`                     |
| `AggregateExpr`     | 91-98          | `sum by (job) (...)`       | `Op`, `Expr`, `Param`, `Grouping`, `Without`             |
| `BinaryExpr`        | 100-111        | `a + b`, `a > bool b`      | `Op`, `LHS`, `RHS`, `VectorMatching`, `ReturnBool`       |
| `SubqueryExpr`      | 142-160        | `(...)[5m:1m]`             | `Expr`, `Range`, `Step`, `Offset`                        |
| `UnaryExpr`         | 184-190        | `-x`                       | `Op`, `Expr`                                             |
| `NumberLiteral`     | 163-168        | `42`                       | `Val float64`                                            |
| `StringLiteral`     | 178-181        | `"foo"`                    | `Val string`                                             |
| `ParenExpr`         | 171-175        | `(expr)`                   | `Expr`                                                   |
| `StepInvariantExpr` | 195-197        | _(internal)_               | `Expr` — optimization marker                             |

The `eval()` method (`engine.go:1925-2331`) is a big type switch over all of
these, dispatching to the appropriate evaluation logic for each node kind.

**Binary operation matching** uses `VectorMatching` (`ast.go:309-323`):

```go
type VectorMatching struct {
    Card           VectorMatchCardinality  // one-to-one, many-to-one, etc.
    MatchingLabels []string
    On             bool   // on(...) vs ignoring(...)
    Include        []string
    FillValues     VectorMatchFillValues
}
```

AST traversal utilities are at `ast.go:336-399`: `Walk`, `Inspect`, `Children`,
`ChildrenIter`.

---

### 4. Alert and Recording Rules

Both rule types call the engine through a callback:

```go
type QueryFunc func(ctx context.Context, qs string, ts time.Time) (promql.Vector, error)
```

#### AlertingRule (`alerting.go:116-157`)

```go
type AlertingRule struct {
    name         string
    vector       parser.Expr      // the PromQL condition
    holdDuration time.Duration    // pending → firing delay
    keepFiringFor time.Duration   // keeps firing after resolution
    active       map[uint64]*Alert
    // ...
}
```

`Eval` method (`alerting.go:382`):

```go
func (r *AlertingRule) Eval(ctx context.Context, queryOffset time.Duration,
    ts time.Time, query QueryFunc, externalURL *url.URL, limit int) (promql.Vector, error)
```

**Evaluation steps:**

1. Calls `query(ctx, r.vector.String(), ts.Add(-queryOffset))` — runs the PromQL
   expression as an instant query
2. For each result sample, creates or updates an `Alert` in `r.active`
3. **State machine transitions:**
   - New match → `StatePending` (sets `ActiveAt`)
   - Pending for ≥ `holdDuration` → `StateFiring`
   - Condition gone → `StateInactive` (or kept alive by `keepFiringFor`)
4. Returns a vector with the `ALERTS` metric for downstream consumers
   (Alertmanager)

The `Alert` struct (`alerting.go:84-100`):

```go
type Alert struct {
    State       AlertState
    Labels      labels.Labels
    Annotations labels.Labels
    Value       float64
    ActiveAt    time.Time
    FiredAt     time.Time
    ResolvedAt  time.Time
    // ...
}
```

#### RecordingRule (`recording.go:38-54`)

```go
type RecordingRule struct {
    name   string
    vector parser.Expr   // the PromQL expression to record
    labels labels.Labels  // labels to add to results
    // ...
}
```

`Eval` method (`recording.go:85-122`):

```go
func (rule *RecordingRule) Eval(ctx context.Context, queryOffset time.Duration,
    ts time.Time, query QueryFunc, _ *url.URL, limit int) (promql.Vector, error)
```

**Evaluation steps:**

1. Calls `query(ctx, rule.vector.String(), ts.Add(-queryOffset))`
2. Overrides the metric `__name__` label to the rule's name
3. Merges in the rule's additional labels
4. Checks for duplicate labelsets (returns error if found)
5. Returns the vector for ingestion into TSDB as a new time series

---

### End-to-End Flow

```
PromQL string
    ↓  parser.Parse()
AST (Expr tree)
    ↓  Engine.NewInstantQuery / NewRangeQuery
query object
    ↓  query.Exec()
Engine.exec() → execEvalStmt()
    ↓  creates evaluator, pre-loads series via populateSeries()
evaluator.Eval(expr)
    ↓  eval() — recursive switch on node type
    ├─ VectorSelector → vectorSelectorSingle()    → reads TSDB
    ├─ MatrixSelector → matrixSelector()           → reads TSDB range
    ├─ Call           → FunctionCalls[name] via rangeEval()
    ├─ AggregateExpr  → rangeEvalAgg()
    ├─ BinaryExpr     → rangeEval() with binary op + VectorMatching
    └─ SubqueryExpr   → evalSubquery() (nested evaluator)
    ↓
Result { Value (Vector|Matrix|Scalar), Warnings, Err }

Rules layer (on a timer):
    Group.Eval() → for each Rule:
        rule.Eval(ctx, offset, ts, queryFunc, ...)
            → queryFunc wraps Engine.NewInstantQuery().Exec()
            → AlertingRule: state machine (pending/firing/inactive)
            → RecordingRule: rename metric, write back to TSDB
```

### Full Judge Analysis (Opus 4.6)

## Content Quality

**Ranking: opus/mcp-full > opus/mcp-only > sonnet/mcp-only > opus/baseline >
sonnet/mcp-full > sonnet/baseline**

**1. opus/mcp-full** — The most comprehensive and precise answer. It correctly
identifies the Engine struct at `engine.go:345-361`, the evaluator at
`engine.go:1138-1152`, and provides accurate line references throughout.
Uniquely covers `VectorMatching` struct details (`ast.go:309-323`), safety sets
(`AtModifierUnsafeFunctions`, `AnchoredSafeFunctions`, `SmoothedSafeFunctions`),
the `Alert` struct (`alerting.go:84-100`), and a detailed table of AST node
types with line ranges. The evaluator method table (listing `rangeEval`,
`rangeEvalAgg`, `matrixSelector`, `vectorSelectorSingle` with locations) is a
standout. Coverage of the rules layer is thorough, showing both struct
definitions and eval method internals. File/line references are consistently
specific and appear well-grounded in actual code reads.

**2. opus/mcp-only** — Very strong, nearly matching mcp-full in correctness and
structure. Covers the Engine struct, evaluator, AST dispatch, function
registration, and rules thoroughly. Notable for explaining the panic-based error
handling pattern (`evaluator.error()` panics, `evaluator.recover()` catches),
the `StepInvariantExpr` preprocessing optimization (`preprocessExprHelper` at
`engine.go:4246-4341`), and the explicit engine comment about not handling
alert/recording statements. Slightly less detailed on AST node types and missing
some of the struct-level detail that mcp-full provides (e.g., `VectorMatching`,
`Alert` struct). Line references are precise.

**3. sonnet/mcp-only** — Excellent completeness and organization. Correctly
traces the full execution path from `query.Exec` through `execEvalStmt` to
`evaluator.eval`. The explanation of `rangeEval` mechanics (pre-evaluate
sub-expressions into matrices, loop timestamps, gather per-step vectors, call
funcCall) is the clearest of all answers. Correctly identifies special-cased
functions (`label_replace`, `label_join`, `info`) with `nil` entries in the map.
The call chain summary at the end is detailed and accurate. Slightly less
precise on some line numbers compared to opus variants, and the rules section is
thinner (correctly describes `QueryFunc` but infers the type signature rather
than citing a specific location).

**4. opus/baseline** — Solid and accurate despite having no tool access.
Correctly describes all major components: Engine, evaluator, AST nodes, function
registry, binary expression dispatch, and both rule types. The function
registration section is well-done, noting the `EvalNodeHelper` purpose. The
alert state machine description is detailed (including `keepFiringFor` and the
15-minute resolved retention). Line references are present but some appear
interpolated from general knowledge rather than verified reads. Missing some
specifics like the `Engine` struct fields and the preprocessing optimization.

**5. sonnet/mcp-full** — Correct in all major points but surprisingly concise
given it had full tool access. Covers the same ground as others but with less
depth — the AST dispatch table is shorter, the function registration section is
adequate but not exceptional, and the rules section is the briefest of the six
(correctly describes the flow but lacks struct definitions). The
`EvalNodeHelper` mention is good. The "Summary Flow" diagram is clean but
simpler than others. Given the high token/cost usage, the information density is
disappointing.

**6. sonnet/baseline** — Accurate and well-structured but the weakest overall.
Covers all required topics and the end-to-end flow diagram is good. However, it
presents some information less precisely — the `FunctionCall` type signature
location is given as `functions.go:2153-2237` (which is actually the map, not
the type), and the binary expression section uses pseudocode that doesn't
perfectly match the actual code structure. The AST visitor/walk section is a
nice inclusion but less relevant to the evaluation question. Line references are
the least reliable of the six.

---

## Efficiency Analysis

| Scenario        | Duration | Total Input | Cost  | Quality Rank |
| --------------- | -------- | ----------- | ----- | ------------ |
| opus/mcp-only   | 71.9s    | 114.7K      | $0.67 | 2nd          |
| sonnet/baseline | 139.3s   | 62.1K       | $0.82 | 6th          |
| sonnet/mcp-only | 96.2s    | 241.5K      | $1.35 | 3rd          |
| opus/mcp-full   | 161.8s   | 63.0K       | $1.83 | 1st          |
| sonnet/mcp-full | 115.6s   | 955.1K      | $3.41 | 5th          |
| opus/baseline   | 273.2s   | 61.8K       | $7.16 | 4th          |

**Best efficiency: opus/mcp-only** — Fastest wall-clock time (71.9s), lowest
cost ($0.67), and second-best quality. This is the clear winner for
quality-to-cost ratio. It found the right information quickly via semantic
search without needing to read entire files, and Opus's reasoning produced a
thorough, well-organized answer.

**Worst efficiency: opus/baseline** — By far the most expensive ($7.16) and
slowest (273.2s), yet ranked only 4th in quality. Without tools, Opus spent
heavily on reasoning tokens to reconstruct information from training data. The
4x cost premium over opus/mcp-only for inferior output makes this the worst
value proposition.

**Surprising findings:**

1. **sonnet/mcp-full is an anti-pattern.** It consumed 955K total input tokens
   and cost $3.41 — the second most expensive — yet produced the 5th-ranked
   answer. It appears to have read far too many files without effectively
   synthesizing the information. More tool access made Sonnet _less_ efficient
   here.

2. **opus/mcp-only vs opus/mcp-full** — mcp-only was faster (71.9s vs 161.8s)
   and cheaper ($0.67 vs $1.83) but produced slightly less detailed output. The
   mcp-full answer's additional detail (struct definitions, safety sets, AST
   line ranges) may justify the 2.7x cost increase depending on use case.

3. **Cache reads matter.** The baseline and mcp-full runs show ~28K cache read
   tokens, indicating prompt caching. The mcp-only runs show 0 cache reads,
   suggesting different prompt structures. Despite this, opus/mcp-only was still
   cheapest overall.

4. **Sonnet baseline outperformed sonnet/mcp-full on cost-per-quality.** At
   $0.82 for 6th place vs $3.41 for 5th place, the baseline was 4x cheaper for
   comparable quality — suggesting that for Sonnet, heavy file reading actually
   degraded the cost-effectiveness.

**Recommendation:** **opus/mcp-only** is the optimal quality-to-cost tradeoff —
highest quality tier at the lowest cost. For maximum quality regardless of cost,
**opus/mcp-full** at $1.83 is still very reasonable. Avoid the sonnet/mcp-full
configuration, which combines high cost with mediocre output.

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
| **sonnet** | baseline | 94.1s    | 31591     | 28104      | 0             | 1704       | $0.72614   |
| **sonnet** | mcp-only | 118.8s   | 212217    | 0          | 0             | 6100       | $1.21358   |
| **sonnet** | mcp-full | 99.4s    | 278960    | 168624     | 0             | 5388       | $1.61381   |
| **opus**   | baseline | 161.1s   | 32031     | 28230      | 0             | 2140       | $2.57384   |
| **opus**   | mcp-only | 82.4s    | 103593    | 0          | 0             | 3915       | $0.61584   |
| **opus**   | mcp-full | 150.7s   | 32237     | 28230      | 0             | 2258       | $3.20482   |

### Answer: `sonnet` / `baseline`

Here's how Prometheus metrics scraping and collection works in this codebase:

---

## Key Types

### Registry (`prom_registry.go:260-267`)

The central hub. Holds all registered collectors, validates descriptor
uniqueness, and orchestrates `Gather()`.

```go
type Registry struct {
    collectorsByID   map[uint64]Collector  // hash → collector
    descIDs          map[uint64]struct{}   // known descriptor hashes
    dimHashesByName  map[string]uint64     // name → dimension hash (consistency check)
}
```

**Registration flow:** `Register()` calls `Describe()` on the collector,
validates each descriptor (unique ID, consistent label names), then stores the
mapping.

**Gather flow:** `Gather()` sends each collector a channel, calls `Collect()`,
reads metrics off the channel, validates consistency via `processMetric()`, and
returns sorted `[]*dto.MetricFamily`.

---

### Counter (`prom_counter.go:103-110`)

Only ever increases. Uses **dual-track atomic storage** to avoid floating-point
CAS loops in the hot path:

```go
type counter struct {
    valBits uint64  // float64 bits — for Add(non-integer)
    valInt  uint64  // integer fast path — for Inc()
}
```

`Inc()` → `atomic.AddUint64(&valInt, 1)` (fast)  
`Add(f)` → CAS loop on `valBits`  
`Write()` → sums both: `float64(valInt) + math.Float64frombits(valBits)`

`Add` panics on negative values (`prom_counter.go:127-128`) — that's a Gauge
responsibility.

---

### Gauge (`prom_gauge.go:106-134`)

Arbitrary up/down. `Set()` is a single atomic store; `Add()`/`Sub()` use a CAS
loop:

```go
func (g *gauge) Add(val float64) {
    for {
        old := atomic.LoadUint64(&g.valBits)
        new := math.Float64bits(math.Float64frombits(old) + val)
        if atomic.CompareAndSwapUint64(&g.valBits, old, new) {
            return
        }
    }
}
```

---

### ScrapePool (`scrape.go:84-116`)

One pool per scrape job (e.g., one per `job_name` in `prometheus.yml`). Manages
an HTTP client and a map of per-target `scrapeLoop`s:

```go
type scrapePool struct {
    appendable    storage.Appendable
    loops         map[uint64]loop       // targetHash → loop
    activeTargets map[uint64]*Target
    client        *http.Client
    config        *config.ScrapeConfig
}
```

`reload()` diffs the old and new target lists: stops loops for removed targets,
creates loops for new ones — no restart needed for unchanged targets.

---

### ScrapeLoop (`scrape.go:1154-1299`)

One goroutine per target. Runs forever:

```
wait scrape offset → ticker → scrapeAndReport() → wait → …
```

On shutdown: emits stale markers for all tracked series.

---

## Parser Interface (`textparse_interface.go:28-214`)

Format is negotiated via `Content-Type`:

| Content-Type                      | Format                  |
| --------------------------------- | ----------------------- |
| `application/openmetrics-text`    | OpenMetrics             |
| `application/vnd.google.protobuf` | Protobuf binary         |
| `text/plain`                      | Classic Prometheus text |

The parser is a cursor — call `Next()` to advance, then call the appropriate
method for the current `Entry` kind (`EntryType`, `EntryHelp`, `EntrySeries`,
`EntryHistogram`, etc.).

---

## Full Data Flow: Scrape → Storage

```
[HTTP GET /metrics]
        │
        ▼ raw bytes
[textparse.Parser]   ← detects format, iterates entry by entry
        │
        ▼ (labels.Labels, timestamp, value)
[sampleMutator]      ← applies metric_relabel_configs, honor_labels
        │
        ▼
[Validation]         ← sample/label limits, UTF-8, name length
        │
        ▼
[storage.Appender]
  .Append(ref, labels, t, v)    → returns SeriesRef (cached for next scrape)
  .AppendExemplar(...)
  .AppendHistogram(...)
  .Commit()                     → writes batch to TSDB
        │
        ▼
[report()]           ← writes up{}, scrape_duration_seconds, scrape_samples_post_metric_relabeling
```

The **`SeriesRef`** is the key optimization: after the first scrape, the loop
caches `ref → labels` mappings. Subsequent scrapes pass the ref directly,
skipping label hashing for known series.

---

## Registry `processMetric()` — Consistency Validation

Each metric arriving on the collect channel is checked:

1. Does its descriptor exist in the registry?
2. Do label dimensions match what was registered (via `dimHashesByName`)?
3. Is the metric name globally unique within a `MetricFamily`?

Violations return errors (or panic in strict mode), preventing corrupted
exposition.

---

## Stale Markers (`model_value.go:20-34`)

When a target disappears or a series is no longer exported, the loop appends a
special signaling NaN (`0x7ff0000000000002`) rather than deleting the series.
The TSDB uses this to close the series cleanly and stop returning it in range
queries.

### Answer: `sonnet` / `mcp-only`

Here's a complete walkthrough of how Prometheus metrics scraping and collection
works across this codebase.

---

## Overview: Two Separate Systems

The fixture contains two distinct subsystems:

1. **The scrape engine** (`scrape.go`, `manager.go`) — polls remote targets over
   HTTP and stores time-series data.
2. **The client library** (`prom_gauge.go`, `prom_registry.go`) — lets Go code
   instrument itself and expose metrics.

They are connected at the `/metrics` HTTP endpoint: the client library serves
it; the scrape engine fetches it.

---

## 1. The Scrape Manager (`manager.go`)

`Manager` is the top-level orchestrator. It holds one `scrapePool` per
configured job.

```
Manager
 ├── scrapePools map[jobName]*scrapePool
 └── scrapeConfigs map[jobName]*ScrapeConfig
```

**`ApplyConfig`** (`manager.go:278`) is called on every config reload:

- Reads all `ScrapeConfig` objects from the new config
- For each job, either creates a new `scrapePool`, reloads an existing one, or
  tears down a removed one
- Pools are reloaded concurrently, bounded by `runtime.GOMAXPROCS(0)` via a
  buffered channel semaphore

**`scrapePool.Sync`** (`scrape.go:388`) is called when service discovery
delivers updated target groups:

- Iterates `[]*targetgroup.Group` → calls `TargetsFromGroup` to build `*Target`
  list
- Drops targets with no labels
- Calls `sp.sync(all)` to reconcile running loops against the new target set

**`scrapePool.sync`** (`scrape.go:494`) is the inner reconciliation loop:

- For targets not yet running: creates a new `scrapeLoop` and starts it with
  `go l.run(nil)`
- For targets no longer present: calls `l.stop()` to drain the old loop
- Uses `map[uint64]loop` keyed by the target's label-set hash

---

## 2. The Scrape Loop: One Per Target (`scrape.go`)

```go
// scrape.go:83
type scrapePool struct {
    loops         map[uint64]loop       // hash → scrapeLoop
    activeTargets map[uint64]*Target
    appendable    storage.Appendable
    ...
}

// scrape.go:822
type scrapeLoop struct {
    interval  time.Duration
    timeout   time.Duration
    scraper   scraper              // ← fetches HTTP
    appendable storage.Appendable  // ← writes time-series
    cache     *scrapeCache         // ← series ref cache
    ...
}
```

**`scrapeLoop.run`** (`scrape.go:1234`) is the per-target event loop:

```
run()
  │
  ├── wait for initial jitter offset (scraper.offset())
  │
  └── for { select {
        case <-ticker.C:
          scrapeAndReport(last, scrapeTime, errc)
        case <-ctx.Done():
          break → endOfRunStaleness()
      }}
```

The ticker fires at `sl.interval`. `AlignScrapeTimestamps` optionally snaps
timestamps to wall-clock interval boundaries (with 1% tolerance) to avoid jitter
accumulating over time.

---

## 3. Scrape Execution: HTTP Fetch + Parse + Append

**`scrapeAndReport`** (`scrape.go:1308`) orchestrates one full scrape cycle:

```
scrapeAndReport()
  │
  ├── app := sl.appender()              // get storage.Appender
  │
  ├── sl.scraper.scrape(ctx)            // HTTP GET /metrics
  │   └── targetScraper.scrape()
  │       └── http.Client.Do(req)       // returns *http.Response
  │
  ├── body → []byte buf
  │
  ├── app.append(buf, contentType, ts)  // parse + write
  │
  ├── app.Commit()  (or Rollback on error)
  │
  └── sl.report(...)  // write scrape meta-metrics (_up, _duration_seconds, etc.)
```

**`targetScraper`** (`scrape.go:685`):

```go
type targetScraper struct {
    *Target
    client  *http.Client
    req     *http.Request   // reused across scrapes
    gzipr   *gzip.Reader    // for Accept-Encoding: gzip
    ...
}
```

Sets `Accept`, `Accept-Encoding`, `User-Agent`, and
`X-Prometheus-Scrape-Timeout-Seconds` headers. The `req` is lazily created on
first scrape and reused.

---

## 4. Parsing the Text Exposition Format

**`scrapeLoopAppender.append`** (`scrape.go:1562`) drives parsing:

```go
p, err := textparse.New(b, contentType, sl.symbolTable, textparse.ParserOptions{...})

for {
    et, err := p.Next()          // advance to next entry
    switch et {
    case textparse.EntryType:    // # TYPE foo counter
    case textparse.EntryHelp:    // # HELP foo ...
    case textparse.EntryComment:
    case textparse.EntrySeries:  // foo{label="val"} 1.23 timestamp
        p.Labels(&lset)
        _, ts, val = p.Series()
        ref, err = app.Append(ref, lset, t, val)
    case textparse.EntryHistogram:
        ref, err = app.AppendHistogram(ref, lset, t, h, nil)
    }
}
```

The `Parser` interface (`textparse_interface.go:63`) is the abstraction over
both Prometheus text format and OpenMetrics:

```go
type Parser interface {
    Next() (Entry, error)       // advance, returns entry type
    Series() ([]byte, *int64, float64)  // metric name bytes, optional timestamp, value
    Labels(l *labels.Labels)    // write labels into caller's struct
    Exemplar(l *exemplar.Exemplar) bool
    StartTimestamp() int64      // for created-timestamp / CT zero ingestion
}
```

Entry types: `EntrySeries=1`, `EntryType=2`, `EntryHelp=3`, `EntryComment=4`,
`EntryHistogram=5`.

---

## 5. Series Cache (`scrapeCache`)

`scrapeCache` avoids repeated label lookups across scrapes:

```go
// scrape.go:815
type cacheEntry struct {
    ref      storage.SeriesRef  // TSDB series ID
    lastIter uint64             // iteration counter for staleness
    hash     uint64             // label set hash
    lset     labels.Labels
}
```

- **Cache hit**: re-use `storage.SeriesRef` directly in `app.Append(ref, ...)`,
  avoiding a label-set lookup in TSDB
- **Cache miss**: TSDB returns a new ref → stored via
  `cache.addRef(met, ref, lset, hash)`
- **Staleness tracking**: `trackStaleness(ref, ce)` records which series were
  seen this iteration; `forEachStale` finds series not seen in the current
  scrape and appends `StaleNaN` markers so TSDB knows the series ended

---

## 6. Counter and Gauge Internal Storage (Client Library)

**`gauge`** (`prom_gauge.go:90`):

```go
type gauge struct {
    valBits uint64       // float64 bits stored as uint64 for atomic ops
    selfCollector         // implements Collector by collecting itself
    desc       *Desc
    labelPairs []*dto.LabelPair
}
```

`Set` is a direct atomic store:

```go
func (g *gauge) Set(val float64) {
    atomic.StoreUint64(&g.valBits, math.Float64bits(val))
}
```

`Add` uses a CAS loop because there's no native atomic float64 add:

```go
func (g *gauge) Add(val float64) {
    for {
        oldBits := atomic.LoadUint64(&g.valBits)
        newBits := math.Float64bits(math.Float64frombits(oldBits) + val)
        if atomic.CompareAndSwapUint64(&g.valBits, oldBits, newBits) {
            return
        }
    }
}
```

`Inc()` just calls `Add(1)`. No mutex, no allocation on the hot path.

---

## 7. Registry: Descriptor Validation + Gather

**`Registry.Register`** (`prom_registry.go:269`):

- Collector calls `Describe(descChan)` → Registry reads all `*Desc` objects
- Validates: no duplicate desc ID (fqName + const labels), consistent dimension
  hash per name
- Computes `collectorID = XOR of all desc.id values` — uniquely identifies the
  collector
- Stores in `r.collectorsByID[collectorID]`

**`Registry.Gather`** (`prom_registry.go:411`) — called when `/metrics` is
scraped:

```
Gather()
  │
  ├── enqueue all collectors into checkedCollectors/uncheckedCollectors channels
  │
  ├── spawn goroutine workers (bounded by goroutineBudget = num collectors)
  │   └── collectWorker():
  │       └── collector.Collect(metricChan)  // each metric → channel
  │
  ├── drain metricChan:
  │   └── processMetric(metric, metricFamiliesByName, ...)
  │       ├── metric.Write(dtoMetric)  // serialize to dto.Metric proto
  │       ├── look up or create MetricFamily by desc.fqName
  │       ├── set MetricFamily.Type (GAUGE/COUNTER/SUMMARY/HISTOGRAM/UNTYPED)
  │       └── append dtoMetric to MetricFamily.Metric
  │
  └── return sorted []*dto.MetricFamily
```

`processMetric` (`prom_registry.go:688`) determines the metric type by
inspecting which field is set on the `dto.Metric`:

```go
switch {
case dtoMetric.Gauge != nil:     metricFamily.Type = dto.MetricType_GAUGE
case dtoMetric.Counter != nil:   metricFamily.Type = dto.MetricType_COUNTER
case dtoMetric.Histogram != nil: metricFamily.Type = dto.MetricType_HISTOGRAM
...
}
```

---

## Data Flow: End to End

```
Go process (client library)
  gauge.Add(1)
    └── atomic CAS on valBits
          │
HTTP GET /metrics ◄─────────── scrapeLoop (every interval)
  Registry.Gather()
    └── Collector.Collect(ch)
          └── gauge.Write(dto) → encodes float64 to proto
  text/plain response body
          │
targetScraper.scrape() ────────► []byte buffer
          │
textparse.Parser.Next()
  EntrySeries → p.Labels(), p.Series()
          │
scrapeCache lookup (ref hit or miss)
          │
storage.Appender.Append(ref, lset, timestamp, value)
  app.Commit() ──────────────► TSDB / WAL
```

**Key design choices visible in the code:**

- Atomic float64 via `uint64` + CAS avoids mutex on every metric update
- `scrapeCache` keeps TSDB series refs alive across scrapes to avoid label-set
  resolution overhead on every tick
- `Registry.Gather` uses a goroutine-per-collector pattern with channel
  buffering so slow collectors don't block others
- `collectorID = XOR(desc IDs)` lets duplicate-registration detection be O(1)
  per descriptor

### Answer: `sonnet` / `mcp-full`

Here is a comprehensive explanation of the full pipeline, from scrape
coordination down to in-memory storage.

---

## Prometheus Metrics: Data Flow from Scrape to Storage

### 1. The Scrape Manager — Top-Level Coordinator

**`manager.go`** — The `Manager` is the top of the hierarchy. It receives
target-group updates from service discovery and drives everything else.

```
tsets channel  →  Manager.Run()  →  Manager.reloader()  →  Manager.reload()
```

- **`Manager.Run`** (`manager.go:158`) — Listens on a `tsets <-chan` for target
  groups from service discovery. On each update it fires `triggerReload`.
- **`Manager.reloader`** — Throttles reloads using `DiscoveryReloadInterval`,
  then calls `reload()`.
- **`Manager.reload`** (`manager.go:211`) — For each named target set, creates a
  `scrapePool` if one doesn't already exist, then calls `sp.Sync(groups)` in
  parallel goroutines.

```go
// manager.go:211 — one scrapePool per job, Sync'd in parallel
go func(sp *scrapePool, groups []*targetgroup.Group) {
    sp.Sync(groups)
    wg.Done()
}(m.scrapePools[setName], groups)
```

---

### 2. The Scrape Pool — Per-Job Loop Manager

**`scrapePool`** (`scrape.go:83`) manages one scrape job (e.g., all targets for
`job="node-exporter"`):

```go
type scrapePool struct {
    appendable   storage.Appendable    // TSDB writer
    config       *config.ScrapeConfig
    client       *http.Client
    loops        map[uint64]loop        // one loop per target hash
    activeTargets map[uint64]*Target
    ...
}
```

**`scrapePool.sync`** (`scrape.go:436`) deduplicates targets and reconciles the
live set:

- For **new** targets: creates a `targetScraper` + `scrapeLoop` and calls
  `go l.run(nil)`.
- For **gone** targets: calls `l.stop()` to drain them gracefully.
- For **unchanged** targets: no-ops (loop keeps running at its own tick).

---

### 3. The Scrape Loop — Per-Target Ticker

**`scrapeLoop`** (`scrape.go:822`) runs the actual interval ticker for one
target.

**`scrapeLoop.run`** (`scrape.go:1234`):

```go
ticker := time.NewTicker(sl.interval)
for {
    // ...
    last = sl.scrapeAndReport(last, scrapeTime, errc)
    <-ticker.C
}
```

**`scrapeLoop.scrapeAndReport`** (`scrape.go:1308`):

1. Creates an `Appender` from the TSDB.
2. Calls `sl.scraper.scrape(ctx)` — an HTTP GET to the target's `/metrics`
   endpoint.
3. Reads the response body into a `bytes.Buffer`.
4. Passes the raw bytes to `app.append(b, contentType, ts)`.
5. Defers `app.Commit()` (or `Rollback()` on error) and metric reporting.

---

### 4. The Text Parser — Raw Bytes → Typed Entries

**`textparse_interface.go`** defines the `Parser` interface and the `New`
factory.

**`textparse.New`** (`textparse_interface.go:166`) dispatches on `Content-Type`:
| Content-Type | Parser | |---|---| | `text/plain` | `NewPromParser` —
Prometheus text format | | `application/openmetrics-text` |
`NewOpenMetricsParser` | | `application/vnd.google.protobuf` |
`NewProtobufParser` |

The `Parser` interface is a pull-based iterator:

```go
type Parser interface {
    Next() (Entry, error)      // advance: EntryType, EntryHelp, EntrySeries, EntryHistogram...
    Series() ([]byte, *int64, float64)   // name bytes, optional ts, float value
    Histogram() ([]byte, *int64, *histogram.Histogram, *histogram.FloatHistogram)
    Labels(l *labels.Labels)   // writes label set for current entry
    Help() ([]byte, []byte)
    Type() ([]byte, model.MetricType)
    Exemplar(l *exemplar.Exemplar) bool
}
```

**`scrapeLoopAppender.append`** (`scrape.go:1562`) drives this loop:

```go
p, _ := textparse.New(b, contentType, sl.symbolTable, opts)
for {
    et, err := p.Next()
    switch et {
    case textparse.EntrySeries:
        met, parsedTs, val := p.Series()
        p.Labels(&lset)
        ref, err = app.Append(ref, lset, t, val)
    case textparse.EntryHistogram:
        met, parsedTs, h, fh := p.Histogram()
        ref, err = app.AppendHistogram(ref, lset, t, h, nil)
    case textparse.EntryType, textparse.EntryHelp:
        // metadata only, no sample appended
    }
}
```

Each `EntrySeries` or `EntryHistogram` becomes a call to the storage `Appender`.

---

### 5. Counter & Gauge — Atomic In-Memory Values

The client library stores live values in two types.

**`counter`** (`prom_counter.go:103`):

```go
type counter struct {
    valBits uint64   // float64 bits for non-integer adds (CAS loop)
    valInt  uint64   // fast path: exact integers via atomic.AddUint64

    selfCollector
    desc       *Desc
    labelPairs []*dto.LabelPair
    exemplar   atomic.Value
    createdTs  *timestamppb.Timestamp
}
```

- `Inc()` — `atomic.AddUint64(&c.valInt, 1)` (hot path, no CAS).
- `Add(v float64)` — if `v` is a whole number, uses `valInt`; otherwise
  CAS-loops on `valBits`.
- `get()` — returns `float64(valInt) + float64frombits(valBits)` (reads both
  atomically, adds).
- `Write(out *dto.Metric)` — calls `get()` and populates a protobuf
  `dto.Metric`.

**`gauge`** (`prom_gauge.go:90`):

```go
type gauge struct {
    valBits uint64   // single CAS float64 field
    selfCollector
    desc       *Desc
    labelPairs []*dto.LabelPair
}
```

- `Inc()` / `Dec()` — call `Add(±1)`.
- `Add(v)` — CAS loop on `valBits` only (no integer fast path, since gauges go
  up and down).
- `Set(v)` — `atomic.StoreUint64(&g.valBits, math.Float64bits(v))`.

---

### 6. The Registry — Collector Coordination & Gather

**`Registry`** (`prom_registry.go:260`):

```go
type Registry struct {
    mtx             sync.RWMutex
    collectorsByID  map[uint64]Collector  // keyed by XOR of desc IDs
    descIDs         map[uint64]struct{}   // all known descriptor hashes
    dimHashesByName map[string]uint64     // fqName → label-set hash (for consistency)
    uncheckedCollectors []Collector       // collectors with no Desc (skipped at registration)
}
```

**Registration** (`prom_registry.go:270`): calls `c.Describe(descChan)` in a
goroutine, checks that:

- Each `Desc` is valid and unique.
- Label names and help strings for the same `fqName` are consistent with what's
  already registered.
- `collectorID` = XOR of all desc IDs.

**`Registry.Gather`** (`prom_registry.go:412`) — the exposition path:

1. Drains all collectors into `checkedCollectors` / `uncheckedCollectors`
   channels.
2. Spawns `collectWorker` goroutines (one per collector, up to budget) that call
   `collector.Collect(metricChan)`.
3. Main loop reads from `metricChan` and calls **`processMetric`** for each
   metric.
4. `processMetric` calls `metric.Write(dtoMetric)` (which reads the atomic
   value), groups by `fqName` into
   `metricFamiliesByName map[string]*dto.MetricFamily`, validates type
   consistency and label uniqueness.
5. Returns `internal.NormalizeMetricFamilies(metricFamiliesByName)` — sorted by
   name.

**`MetricFamily`** (protobuf `dto.MetricFamily`) is the wire format:

```
MetricFamily {
  name: "http_requests_total"
  type: COUNTER
  metric: [
    Metric { label:[...], counter: { value: 42 } },
    Metric { label:[...], counter: { value: 17 } },
  ]
}
```

---

### Full Data Flow Summary

```
Service Discovery
      │ tsets channel
      ▼
Manager.Run() → reload() → scrapePool.Sync()
                                  │
                    ┌─────────────┤  one loop per target
                    ▼             ▼
             scrapeLoop.run()  [ticker every interval]
                    │
                    ▼
             scraper.scrape()   ← HTTP GET /metrics
                    │ []byte body
                    ▼
             textparse.New()    ← picks parser from Content-Type
                    │
                    ▼
             Parser.Next() loop
               EntrySeries → app.Append(labels, ts, float64)
               EntryHistogram → app.AppendHistogram(...)
                    │
                    ▼
             app.Commit()       → TSDB WAL + head block

   (separately, for exposition to scrapers of this process)

counter.Inc() / gauge.Set()
      │ atomic store
      ▼
Registry.Gather()
      │ collector.Collect() → metric.Write() → dto.Metric
      ▼
[]*dto.MetricFamily  →  text/protobuf encoder  →  HTTP /metrics response
```

The two halves are independent: the scrape pipeline reads remote metrics into
TSDB storage, while the registry/counter/gauge machinery manages this process's
own instrumentation exposed via `/metrics`.

### Answer: `opus` / `baseline`

Here's how Prometheus metrics scraping and collection works in this codebase:

## Data Flow Overview

```
Discovery → Manager.Run() → scrapePool.Sync() → scrapeLoop.run() [ticker]
  → HTTP GET target → parse text → Appender.Append() → TSDB Head (in-memory)
```

## 1. Scrape Manager Coordination

**`Manager`** (`manager.go:135`) is the top-level coordinator. It holds a map of
`scrapePool`s (one per job) and receives target group updates from service
discovery:

- `Run(tsets)` receives target groups, calls `updateTsets()`, signals
  `triggerReload`
- `reloader()` goroutine waits on a ticker (5s) and calls `reload()`
- `reload()` creates missing `scrapePool`s, then concurrently calls
  `sp.Sync(groups)` on each

**`scrapePool`** (`scrape.go:84`) owns the HTTP client, a shared `SymbolTable`,
and a `map[uint64]loop` keyed by target hash. `Sync()` converts target groups
into `*Target` instances, deduplicates, then `sync()` starts new `scrapeLoop`s
(`go l.run(nil)`) for new targets and stops loops for disappeared ones.

**`scrapeLoop`** (`scrape.go:822`) is the per-target actor. It runs a ticker at
`scrapeInterval` and on each tick calls `scrapeAndReport()`:

1. Acquire an `Appender` via `sl.appender()`
2. `sl.scraper.scrape(ctx)` — HTTP GET to the target
3. `sl.scraper.readResponse()` — stream body into buffer
4. `app.append(b, contentType, ts)` — parse and store
5. `app.Commit()` (or `Rollback()` on error)
6. `sl.report()` — write internal scrape health metrics

## 2. Text Format Parsing

**`Parser`** interface (`textparse_interface.go:29`) defines `Next()`,
`Series()`, `Histogram()`, `Help()`, `Type()`, etc.

The factory `New()` selects parser by Content-Type:

- `"text/plain"` → `PromParser`
- `"application/openmetrics-text"` → `OpenMetricsParser`
- `"application/vnd.google.protobuf"` → `ProtobufParser`

The parse drive loop in `scrapeLoopAppender.append` calls `p.Next()` repeatedly:

| Entry            | Action                                         |
| ---------------- | ---------------------------------------------- |
| `EntryType`      | `scrapeCache.setType()`                        |
| `EntryHelp`      | `scrapeCache.setHelp()`                        |
| `EntrySeries`    | `p.Series()` → `app.Append(ref, lset, t, val)` |
| `EntryHistogram` | `p.Histogram()` → `app.AppendHistogram(...)`   |

After parsing, `updateStaleMarkers()` writes `StaleNaN` (`0x7ff0000000000002`)
for any series present in the previous scrape but absent in the current one.

## 3. Counters and Gauges Internally

**`counter`** (`prom_counter.go:103`) uses a dual-variable design for
performance:

```go
type counter struct {
    valBits uint64   // float64 bits for fractional increments (CAS loop)
    valInt  uint64   // integer accumulator for whole-number increments
    // ...
}
```

- `Inc()` → `atomic.AddUint64(&c.valInt, 1)` — single atomic add, no CAS
- `Add(v)` — if `v` is an exact integer, uses `atomic.AddUint64` on `valInt`;
  otherwise CAS-loops on `valBits`
- `get()` → `Float64frombits(valBits) + float64(valInt)` — sums both at read
  time
- Panics if `v < 0` (counters never decrease)

**`gauge`** (`prom_gauge.go:90`) is simpler — a single `uint64` holding float64
bits:

```go
type gauge struct {
    valBits uint64
    // ...
}
```

- `Set(val)` → `atomic.StoreUint64(&g.valBits, Float64bits(val))`
- `Add(val)` → CAS loop: load, compute, swap
- `Inc()`/`Dec()` → `Add(1)`/`Add(-1)`

Both embed `selfCollector`, so they implement both `Metric` and `Collector`
interfaces — `Describe` sends their `*Desc`, `Collect` sends themselves.

## 4. Registry and Metric Families

**`Registry`** (`prom_registry.go:260`):

```go
type Registry struct {
    collectorsByID      map[uint64]Collector    // XOR of desc IDs → Collector
    descIDs             map[uint64]struct{}      // all registered desc IDs
    dimHashesByName     map[string]uint64        // fqName → hash(labelNames+help)
    uncheckedCollectors []Collector              // collectors with no descriptors
}
```

**Registration** validates: desc uniqueness, label/help consistency per
`fqName`, no collector ID collisions. Collectors with no descriptors are stored
as "unchecked."

**`Gather()`** produces `[]*dto.MetricFamily`:

1. Fan out `Collect(ch)` calls across all collectors via worker goroutines
2. Drain channels, calling `processMetric()` for each metric
3. `processMetric()` builds `metricFamiliesByName`, type-checks each metric
   against its family, deduplicates via xxhash of name+labels
4. Returns `NormalizeMetricFamilies()` — lexicographically sorted

## 5. In-Memory Storage (TSDB Head)

**`Head`** (`head.go:68`) is the write destination:

```go
type Head struct {
    series   *stripeSeries       // hash-sharded map: labels hash → memSeries
    postings *index.MemPostings  // inverted index: label=value → []SeriesRef
    wal      *wlog.WL            // write-ahead log
}
```

The **`scrapeCache`** (`scrape.go:887`) accelerates repeated appends by caching
the mapping from raw metric bytes to `SeriesRef` (a `uint64` opaque handle). On
subsequent scrapes, the ref is reused to skip label lookup:

```go
type scrapeCache struct {
    series     map[string]*cacheEntry       // raw metric → ref + labels
    seriesCur  map[SeriesRef]*cacheEntry    // present this scrape
    seriesPrev map[SeriesRef]*cacheEntry    // present last scrape
    metadata   map[string]*metaEntry        // family name → type/help/unit
}
```

After each scrape, `iterDone(true)` swaps `seriesCur`/`seriesPrev`. Any ref in
`seriesPrev` but not `seriesCur` gets a `StaleNaN` marker appended, signaling
downstream that the series has disappeared.

### Answer: `opus` / `mcp-only`

Here's a detailed explanation of how Prometheus metrics scraping and collection
works in this codebase, covering the full data flow from scrape to in-memory
storage.

---

## 1. Scrape Manager: Top-Level Coordination

The `Manager` (`manager.go:133-156`) is the entry point. It maintains a map of
**scrape pools**, one per scrape job configuration:

```go
type Manager struct {
    scrapeConfigs map[string]*config.ScrapeConfig
    scrapePools   map[string]*scrapePool
    targetSets    map[string][]*targetgroup.Group
    triggerReload chan struct{}
    // ...
}
```

**Data flow starts in `Manager.Run`** (`manager.go:158-179`): it receives target
group updates from service discovery on a channel, saves them, and triggers a
reload via `triggerReload`. The `reloader` goroutine then calls `Manager.reload`
(`manager.go:211-245`), which:

1. Creates new `scrapePool` instances for any new job names
2. Calls `sp.Sync(groups)` in parallel goroutines to synchronize targets

`ApplyConfig` (`manager.go:278-344`) handles configuration changes — it reloads
existing pools with updated configs and stops pools for removed jobs, all
concurrently with a goroutine budget capped at `GOMAXPROCS`.

---

## 2. Scrape Pools and Loops

A `scrapePool` (`scrape.go:83-116`) manages all targets for a single job:

```go
type scrapePool struct {
    loops         map[uint64]loop           // one loop per target, keyed by hash
    activeTargets map[uint64]*Target
    client        *http.Client
    // ...
}
```

When `Sync` runs, it diffs current targets against known ones — creates new
loops for new targets, stops loops for disappeared targets
(`scrape.go:494-525`). Each new loop is launched via `go l.run(nil)`.

The **`scrapeLoop`** (`scrape.go:822-881`) is where the actual periodic scraping
happens. Key fields:

```go
type scrapeLoop struct {
    scraper   scraper          // does the HTTP call
    cache     *scrapeCache     // caches series refs between scrapes
    interval  time.Duration
    timeout   time.Duration
    appendable storage.Appendable  // where samples go (TSDB)
    // ...
}
```

---

## 3. The Scrape Cycle

**`scrapeLoop.run`** (`scrape.go:1234-1299`) is the main ticker loop. It:

1. Waits an initial random offset (to spread load across HA pairs)
2. Enters a `for` loop driven by `time.NewTicker(sl.interval)`
3. Aligns scrape timestamps to avoid jitter
4. Calls `sl.scrapeAndReport(last, scrapeTime, errc)` each tick

**`scrapeAndReport`** (`scrape.go:1308-1432`) orchestrates a single scrape:

1. Calls `sl.scraper.scrape(ctx)` → HTTP GET to target endpoint
2. Reads the response body via `sl.scraper.readResponse()` into a pooled buffer
3. Calls `app.append(b, contentType, appendTime)` to parse and store
4. Calls `sl.report(app, ...)` to write automatic metrics
   (`scrape_duration_seconds`, `scrape_samples_scraped`, etc.)
5. On success: `app.Commit()`. On failure: `app.Rollback()` and retries with an
   empty scrape (for stale markers)

The `scraper` interface (`scrape.go:677-683`) is implemented by `targetScraper`,
which does a simple `http.Client.Do` with Prometheus-specific headers
(`scrape.go:735-752`).

---

## 4. Text Format Parsing

The `Parser` interface (`textparse_interface.go:28-87`) defines a streaming
parser. Key methods:

- `Next() (Entry, error)` — advances to next entry (type info, help, comment,
  series, histogram)
- `Series() ([]byte, *int64, float64)` — returns metric name bytes, optional
  timestamp, and value
- `Histogram()` — returns native histogram data
- `Type()`, `Help()`, `Unit()` — return metadata entries
- `Labels(*labels.Labels)` — writes parsed labels into the provided struct

The `New` function (`textparse_interface.go:158-201`) selects the right parser
based on `Content-Type`:

| Content-Type                      | Parser                                   |
| --------------------------------- | ---------------------------------------- |
| `text/plain`                      | `NewPromParser` (Prometheus text format) |
| `application/openmetrics-text`    | `NewOpenMetricsParser`                   |
| `application/vnd.google.protobuf` | `NewProtobufParser`                      |

**Inside `scrapeLoopAppender.append`** (`scrape.go:1562-1624`), parsing drives a
loop:

```go
p, _ := textparse.New(b, contentType, sl.symbolTable, opts)
for {
    et, err := p.Next()
    switch et {
    case EntryType:   sl.cache.setType(p.Type())
    case EntryHelp:   sl.cache.setHelp(p.Help())
    case EntryComment: continue
    case EntryHistogram: // handle histogram
    default:          // handle float series
    }
    met, parsedTimestamp, val = p.Series()
    // label mutation, relabeling, deduplication via cache
    ref, err = app.Append(ref, lset, t, val)
    sl.cache.addRef(met, ref, lset, hash)
}
```

The cache (`scrapeCache`) maps raw metric byte keys to `storage.SeriesRef` so
that on subsequent scrapes, Prometheus can fast-path append by ref instead of
doing a full label lookup.

---

## 5. Counter and Gauge Internals (Client Library Side)

### Counter (`prom_counter.go:103-120`)

```go
type counter struct {
    valBits uint64    // float64 bits for fractional adds
    valInt  uint64    // exact integer accumulator
    selfCollector
    desc       *Desc
    labelPairs []*dto.LabelPair
}
```

- **`Inc()`** (`prom_counter.go:151`): `atomic.AddUint64(&c.valInt, 1)` —
  lock-free integer increment
- **`Add(v)`** (`prom_counter.go:126-144`): If `v` is a whole number, uses
  `atomic.AddUint64` on `valInt`. Otherwise uses CAS loop on `valBits` to
  atomically add float64
- **`get()`** (`prom_counter.go:155-159`): Returns
  `float64(valInt) + Float64frombits(valBits)` — the split avoids floating-point
  precision loss for integer increments
- Panics if `Add` is called with a negative value — counters only go up

### Gauge (`prom_gauge.go:90-100`)

```go
type gauge struct {
    valBits uint64    // float64 bits
    selfCollector
    desc       *Desc
    labelPairs []*dto.LabelPair
}
```

- **`Add(val)`** (`prom_gauge.go:122-130`): CAS loop —
  `LoadUint64 → Float64frombits → add → Float64bits → CompareAndSwap`
- **`Inc()`**: calls `Add(1)`
- **`Set(v)`**: `atomic.StoreUint64(&g.valBits, math.Float64bits(v))`
- Unlike counters, gauges can go up and down, and only use `valBits` (no integer
  optimization needed)

Both embed `selfCollector`, which implements the `Collector` interface by
returning the metric itself from `Collect()`.

---

## 6. Registry and MetricFamily Management

The `Registry` (`prom_registry.go:253-267`) is the central coordinator on the
client/exposition side:

```go
type Registry struct {
    collectorsByID      map[uint64]Collector   // hash of desc IDs → collector
    descIDs             map[uint64]struct{}     // all registered descriptor IDs
    dimHashesByName     map[string]uint64       // fqName → dimension hash (for consistency checks)
    uncheckedCollectors []Collector
}
```

### Registration (`Registry.Register`, `prom_registry.go:269-324`)

1. Spawns a goroutine calling `c.Describe(descChan)` to collect all `Desc` from
   the collector
2. Validates each descriptor: must be valid, unique desc ID, consistent label
   names / help text
3. Computes a `collectorID` by XOR-ing all desc IDs
4. Checks for duplicate collectors (same collectorID = already registered →
   `AlreadyRegisteredError`)
5. Stores in `collectorsByID` and records `descIDs`

### Gathering (`Registry.Gather`, `prom_registry.go:411-479`)

1. Spawns worker goroutines that call `collector.Collect(metricChan)` for each
   registered collector
2. Drains the channel, calling `processMetric()` for each received `Metric`
3. `processMetric` (`prom_registry.go:619-725`):
   - Calls `metric.Write(dtoMetric)` to serialize the metric into a protobuf
     `dto.Metric`
   - Groups metrics into `map[string]*dto.MetricFamily` by fully-qualified name
   - Infers type from the dto (Gauge/Counter/Summary/Histogram/Untyped)
   - Runs `checkMetricConsistency` — hashes name + sorted labels + optional
     timestamp to detect duplicates
   - Appends the metric to `metricFamily.Metric`
4. Returns a sorted slice of `*dto.MetricFamily` for exposition

The `Gatherers` type (`prom_registry.go:727-803`) merges results from multiple
registries, deduplicating by family name and validating type/help consistency.

---

## End-to-End Data Flow Summary

```
Target HTTP endpoint
        │
        ▼
  targetScraper.scrape()          ─── HTTP GET, read response body
        │
        ▼
  textparse.New() → Parser        ─── Select parser by Content-Type
        │
        ▼
  Parser.Next() loop              ─── Stream entries: type/help/series/histogram
        │
        ▼
  scrapeCache.get(met)            ─── Fast-path: reuse SeriesRef from last scrape
        │
        ▼
  sampleMutator(lset)             ─── Apply target labels + metric relabeling
        │
        ▼
  storage.Appender.Append()       ─── Write to TSDB (by ref or new series)
        │
        ▼
  app.Commit()                    ─── Atomically commit all samples
        │
        ▼
  sl.report()                     ─── Write scrape_duration_seconds, up, etc.
```

The scrape manager coordinates many of these loops concurrently — one
`scrapeLoop` goroutine per active target, all writing into the shared TSDB
appender with series ref caching to minimize lock contention.

### Answer: `opus` / `mcp-full`

## Prometheus Metrics Scraping & Collection — Architecture Walkthrough

The fixture project at `testdata/fixtures/go/` contains real Prometheus source
code. Here's how the full pipeline works:

---

### 1. Scrape Manager Coordination

**Key type:** `Manager` (`manager.go:135-156`)

```go
type Manager struct {
    scrapeConfigs map[string]*config.ScrapeConfig
    scrapePools   map[string]*scrapePool   // one pool per job
    targetSets    map[string][]*targetgroup.Group
    triggerReload chan struct{}
    // ...
}
```

The `Manager.Run()` method listens for target-group updates from service
discovery. On each update it calls `updateTsets` and signals `triggerReload`. A
background `reloader()` goroutine debounces these signals on a 5-second tick and
calls `reload()`, which:

- Creates new `scrapePool` instances for new jobs
- Calls `sp.Sync(groups)` on each pool in parallel via goroutines
- `ApplyConfig` handles config changes, concurrently reloading affected pools
  (bounded by `GOMAXPROCS`)

Each **`scrapePool`** (`scrape.go:84-525`) manages one job's targets. `Sync()`
diffs current vs. desired targets by `t.hash()` — new targets get a
`scrapeLoop`, disappeared ones get stopped.

Each **`scrapeLoop`** runs in its own goroutine, ticking at `sl.interval` and
calling `scrapeAndReport()` each cycle.

---

### 2. Text Format Parsing

**Key type:** `Parser` interface (`textparse_interface.go`)

```go
type Parser interface {
    Next() (Entry, error)           // advance to next token
    Series() ([]byte, *int64, float64)
    Histogram() ([]byte, *int64, *histogram.Histogram, *histogram.FloatHistogram)
    Help() ([]byte, []byte)
    Type() ([]byte, model.MetricType)
    Labels(l *labels.Labels)
    // ...
}
```

Parser implementation is selected by `Content-Type`:

- `text/plain` → `PromParser`
- `application/openmetrics-text` → `OpenMetricsParser`
- `application/vnd.google.protobuf` → `ProtobufParser`

Entry types are `EntryType`, `EntryHelp`, `EntrySeries`, `EntryHistogram`, etc.
The scrape loop's `append()` method (`scrape.go:1620-1869`) drives the parser in
a `for` loop:

```go
for {
    et, err = p.Next()
    switch et {
    case textparse.EntryType:  sl.cache.setType(p.Type())
    case textparse.EntryHelp:  sl.cache.setHelp(p.Help())
    case textparse.EntrySeries:
        met, ts, val = p.Series()
        // → label mutation → cache lookup → appender.Append(ref, lset, t, val)
    }
}
```

---

### 3. Counter & Gauge Internals

**Counter** (`prom_counter.go:103-181`) uses a **two-variable atomic design**:

```go
type counter struct {
    valBits uint64   // atomic float64 bits — for fractional Add()
    valInt  uint64   // atomic integer    — for Inc() (single atomic op, no CAS)
    // ...
}
```

- `Inc()` → `atomic.AddUint64(&c.valInt, 1)` — zero-cost, no CAS loop
- `Add(v)` with exact integer → same `AddUint64` fast path
- `Add(v)` with fractional → CAS loop on `valBits`
- `get()` returns `float64(valInt) + Float64frombits(valBits)`

**Gauge** (`prom_gauge.go:90-139`) is simpler — single `uint64` field:

```go
type gauge struct {
    valBits uint64   // atomic float64 bits
    // ...
}
```

- `Set(val)` → `atomic.StoreUint64` (one instruction)
- `Add(val)` / `Sub(val)` → CAS loop on `valBits`
- `Inc()` / `Dec()` → delegate to `Add(±1)`

Both implement `Metric` via `Write(*dto.Metric)` and `Collector` via the
embedded `selfCollector`.

---

### 4. Registry & Metric Families

**Key type:** `Registry` (`prom_registry.go:260-267`)

```go
type Registry struct {
    collectorsByID      map[uint64]Collector  // key = XOR of all desc IDs
    descIDs             map[uint64]struct{}   // every desc ID ever registered
    dimHashesByName     map[string]uint64     // fqName → label dimension hash
    uncheckedCollectors []Collector
    // ...
}
```

**Registration** (`Register`, lines 270-363):

1. Calls `c.Describe(descChan)` to get `*Desc` objects
2. Validates: unique IDs, consistent label dimensions per `fqName`
3. Computes `collectorID = XOR(desc IDs)` as dedup key
4. Collectors with no descs → `uncheckedCollectors` (no validation)

**Gathering** (`Gather`, lines 412-559) — fan-out/fan-in:

1. All collectors go into buffered channels
2. N worker goroutines call `collector.Collect(metricChan)`
3. Main goroutine drains channels, calling `processMetric()` on each
4. `processMetric` calls `metric.Write(dtoMetric)`, looks up/creates
   `MetricFamily` by `fqName`, detects suffix collisions, deduplicates via
   xxhash, and appends to `metricFamily.Metric`
5. Returns lexicographically sorted `[]*dto.MetricFamily`

---

### 5. End-to-End Data Flow

```
Service Discovery
    │  map[string][]*targetgroup.Group
    ▼
Manager.Run → updateTsets → triggerReload
    │
    ▼
Manager.reload() → scrapePool.Sync()
    │  per target
    ▼
scrapeLoop.run()  →  tick every interval
    │
    ▼
targetScraper.scrape()  →  HTTP GET /metrics  →  []byte
    │
    ▼
textparse.New(body, contentType) → Parser
    │  for each p.Next()
    ▼
p.Series() → (metricName, timestamp, float64)
p.Labels(&lset)
    │
    ▼
sampleMutator(lset)           // relabeling
scrapeCache.get(met)          // ref lookup
    │
    ▼
appender.Append(ref, lset, t, val)   →  TSDB head
scrapeCache.addRef(met, ref, ...)     // cache for next scrape
    │
    ▼
updateStaleMarkers()   // StaleNaN (0x7ff0000000000002) for missing series
cache.iterDone(true)   // swap seriesCur↔seriesPrev for next cycle
app.Commit()           // commit TSDB transaction
```

The **scrapeCache** (`scrape.go:887-935`) is the critical per-loop structure: it
maps raw metric bytes → `{SeriesRef, lset, hash, lastIter}` to avoid re-parsing
labels and re-looking up TSDB references on every scrape. It uses a ping-pong
`seriesCur`/`seriesPrev` swap to detect disappeared series and emit staleness
markers.

### Full Judge Analysis (Opus 4.6)

## Content Quality Ranking

### 1. opus / mcp-full (Best)

Highly accurate and well-structured. Covers all four requested areas (scrape
manager, parsing, counter/gauge internals, registry) with correct details. The
two-variable atomic design for counters is correctly explained with the
`valInt`/`valBits` split and the fast-path rationale. The `scrapeCache`
ping-pong mechanism (`seriesCur`/`seriesPrev`) and stale marker emission are
precisely described. File references like `manager.go:135-156`,
`scrape.go:84-525`, `prom_counter.go:103-181` appear accurate. The end-to-end
data flow diagram is the most complete of all answers, showing the stale marker
and cache swap steps. The mention that this is fixture code in
`testdata/fixtures/go/` shows proper codebase awareness. Used tools effectively
to read actual source.

### 2. sonnet / mcp-full

Equally comprehensive and correct. Excellent structural organization with
numbered sections. Accurately describes `Manager.Run` → `reload` → `Sync` flow,
the `Parser` interface with Content-Type dispatch, counter dual-track atomics,
and the `Registry.Gather` fan-out/fan-in pattern. The `processMetric`
description (lines 619-725) with type inference from dto fields is a nice
detail. Provides both the scrape-side and exposition-side data flows in the
final diagram, which is a valuable distinction no other answer makes as clearly.
Line references are specific and appear accurate. Slightly more verbose than
opus/mcp-full but no less correct.

### 3. opus / mcp-only

Concise and accurate. Covers all key areas with correct technical detail. The
`scrapeCache` description with `seriesCur`/`seriesPrev` swap and
`iterDone(true)` is correct. Counter dual-variable design is well-explained. The
TSDB Head section with `stripeSeries` and `MemPostings` goes slightly beyond
what others cover, which is useful context. File/line references are present but
slightly less precise than the mcp-full variants. The stale marker value
`0x7ff0000000000002` is correctly cited. Good efficiency — extracted the right
information without excessive exploration.

### 4. sonnet / mcp-only

Very thorough — the longest answer. Correctly describes all components with good
code snippets. The `scrapePool.sync` description (lines 494-525) with the
reconciliation logic is well done. The `processMetric` type-detection switch
statement is a useful concrete detail. The final data flow showing both halves
(scrape pipeline and client exposition) as independent systems is insightful.
However, some line references may be approximate rather than verified (e.g.,
`scrape.go:1562` for `append`). Slightly over-verbose for the information
density, but no factual errors detected.

### 5. sonnet / baseline

Surprisingly good for having no tool access to the actual codebase. The counter
dual-track atomic explanation with `valBits`/`valInt` is correct. The
`ScrapePool` and `ScrapeLoop` descriptions are accurate. The `SeriesRef` caching
optimization is correctly identified as a key design choice. The stale marker
NaN value is correct. However, line references like `prom_counter.go:127-128`
and `scrape.go:84-116` cannot be verified against the fixture and may be from
Prometheus upstream rather than this specific codebase. Missing some details
about `Manager.reload` concurrency and `processMetric` internals that
tool-assisted answers caught.

### 6. opus / baseline (Worst of the set, still decent)

Correct on all major points. Good coverage of the `Manager` → `scrapePool` →
`scrapeLoop` hierarchy. Counter and gauge explanations are accurate. The TSDB
Head mention (`head.go:68` with `stripeSeries` and `MemPostings`) adds useful
context. However, it's the most terse of the answers, with less detail on the
parsing loop internals and `processMetric` validation logic. The `scrapeCache`
description is correct but briefer than others. Line references are present but
sparse.

---

## Efficiency Analysis

| Scenario          | Duration | Total Input Tok | Cost  | Quality Rank |
| ----------------- | -------- | --------------- | ----- | ------------ |
| opus / mcp-only   | 82.4s    | 103,593         | $0.62 | 3rd          |
| sonnet / baseline | 94.1s    | 31,591          | $0.73 | 5th          |
| sonnet / mcp-full | 99.4s    | 278,960         | $1.61 | 2nd          |
| sonnet / mcp-only | 118.8s   | 212,217         | $1.21 | 4th          |
| opus / mcp-full   | 150.7s   | 32,237          | $3.20 | 1st          |
| opus / baseline   | 161.1s   | 32,031          | $2.57 | 6th          |

**Surprising findings:**

- **opus/mcp-only is the clear efficiency winner** — fastest wall time (82.4s),
  lowest cost ($0.62), and 3rd in quality. It hit a massive cache read of 0
  tokens but kept input tokens moderate, suggesting it found the right files
  quickly via semantic search without over-exploring.
- **opus/mcp-full is the most expensive** ($3.20) despite having 28K cache read
  tokens. The quality is top-ranked but the 5x cost premium over opus/mcp-only
  for a marginal quality improvement is hard to justify.
- **sonnet/baseline is remarkably cheap** ($0.73) and fast (94.1s) for 5th-place
  quality — it relied on parametric knowledge of Prometheus internals, which is
  largely correct but less grounded in the actual fixture files.
- **sonnet/mcp-only and sonnet/mcp-full consumed far more input tokens** (212K
  and 279K respectively) than their opus counterparts, suggesting sonnet
  explored more files or received more verbose tool results. Despite this,
  sonnet/mcp-full's cache hit (168K of 279K) kept its cost reasonable.
- **opus/baseline is the worst value** — $2.57 for the lowest-ranked answer,
  with no tool usage to verify claims against the actual codebase.

**Recommendation:** **opus/mcp-only** offers the best quality-to-cost tradeoff
at $0.62 for a top-3 answer with verified file references. For maximum quality
regardless of cost, opus/mcp-full is the pick, but at 5x the price. The sonnet
variants occupy an awkward middle ground — more expensive than opus/mcp-only
with lower quality, largely due to token-heavy exploration.
