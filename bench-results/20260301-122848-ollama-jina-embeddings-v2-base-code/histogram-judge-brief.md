## Content Quality

1. **opus/baseline** — Most comprehensive: covers bucket creation functions
   (LinearBuckets, ExponentialBuckets), core observation, bucket
   limiting/resolution reduction, validation, and iteration with accurate line
   references throughout. Minor excess detail but nothing incorrect.

2. **sonnet/baseline** — Strong coverage of classic and native paths, hot/cold
   scheme, TSDB iteration layer, and validation. Includes `addToBucket` and
   `validateCount` with line references. Slightly less organized but very
   thorough.

3. **opus/mcp-full** — Clean, well-structured explanation of both bucket systems
   with accurate code snippets for the key math (Frexp decomposition, schema
   branching). Covers cumulative conversion and delta-encoded iteration. Concise
   without sacrificing correctness.

4. **opus/mcp-only** — Comparable to opus/mcp-full with good coverage of both
   systems, hot/cold swap, and iteration types. Slightly more verbose on the
   native bucket routing but accurate throughout.

5. **sonnet/mcp-full** — Accurate and well-organized with clear sections. Covers
   findBucket, observe, hot/cold dispatch, and cumulative write. Slightly less
   detail on iteration and validation than top answers.

6. **sonnet/mcp-only** — Solid coverage of the core observation path with good
   detail on the double-buffering scheme and implicit +Inf bucket. Misses
   iteration and validation; narrower scope than others.

## Efficiency

The MCP-only runs (sonnet at $0.14/19s, opus at $0.13/22s) are 7-20× cheaper
than baselines while delivering answers of comparable quality. MCP-full runs add
~$0.07-0.08 for cache-read tokens with minimal quality gain over MCP-only. The
baselines are dramatically more expensive (sonnet baseline at $2.62 is an
outlier, opus baseline at $1.01 with 181K input tokens), making them poor value
propositions despite slightly more comprehensive answers.

## Verdict

**Winner: opus/mcp-only**
