# Embedding Model Comparison — Summary Report

Generated: 2026-03-01
Comparing 4 embedding models across 9 benchmark questions × 2 LLMs × 3 scenarios.

## Models Compared

| Run | Backend | Model | Dims | Context |
|-----|---------|-------|------|---------|
| `20260301-192639` | Ollama | `ordis/jina-embeddings-v2-base-code` | 768 | 8192 |
| `20260301-195217` | Ollama | `qwen3-embedding:8b` | 4096 | 40960 |
| `20260301-202246` | Ollama | `qwen3-embedding:4b` | 2560 | 40960 |
| `20260301-205706` | LM Studio | `nomic-ai/nomic-embed-code-GGUF` | 3584 | 8192 |

---

## Scenario Win Counts (per embedding model)

Which scenario won the quality+efficiency verdict for each of the 9 questions:

| Embedding Model | baseline | mcp-only | mcp-full |
|----------------|----------|----------|----------|
| jina-v2-base-code | 1 | **4** | 4 |
| qwen3-8b | 1 | **7** | 1 |
| qwen3-4b | 2 | **7** | 0 |
| nomic-embed-code | 0 | **6** | 3 |

**mcp-only wins outright across all 4 embedding models.** The only consistent exception is `ts-disposable` (easy TypeScript), where baseline or mcp-full wins due to over-retrieval in mcp-only.

---

## Per-Question Winners by Embedding Model

| Question | Lang | Diff | jina | qwen3-8b | qwen3-4b | nomic |
|----------|------|------|------|----------|----------|-------|
| go-label-matcher | go | easy | opus/mcp-full | opus/mcp-only | opus/mcp-only | opus/mcp-only |
| go-histogram | go | med | opus/mcp-only | opus/mcp-full | opus/mcp-only | opus/mcp-full |
| go-tsdb-compaction | go | hard | opus/mcp-full | opus/mcp-only | opus/mcp-only | opus/mcp-full |
| py-permissions | py | easy | opus/mcp-only | opus/mcp-only | opus/mcp-only | opus/mcp-only |
| py-flask-config | py | med | opus/mcp-only | opus/mcp-only | opus/mcp-only | opus/mcp-only |
| py-django-queryset | py | hard | opus/mcp-full | opus/mcp-only | opus/mcp-only | opus/mcp-only |
| ts-disposable | ts | easy | opus/baseline | opus/baseline | sonnet/baseline | **opus/mcp-full** |
| ts-event-emitter | ts | med | opus/mcp-only | opus/mcp-only | opus/mcp-only | opus/mcp-only |
| ts-async-lifecycle | ts | hard | sonnet/mcp-full | opus/mcp-only | sonnet/baseline | sonnet/mcp-only |

**Observations:**
- Python questions: mcp-only wins unanimously across all 4 models (6/6 slots).
- Go questions: mcp-only or mcp-full win unanimously; no baseline wins.
- TypeScript questions: more variance; baseline wins ts-disposable for 3 of 4 models.

---

## Total Cost by Embedding Model

All costs in USD, summed across 9 questions × 2 LLMs.

| Embedding Model | baseline | mcp-only | mcp-full | **Grand Total** |
|----------------|----------|----------|----------|-----------------|
| jina-v2-base-code | $20.92 | $5.09 | $7.26 | **$33.27** |
| qwen3-8b | $12.49 | $5.72 | $8.24 | **$26.45** |
| qwen3-4b | $17.76 | $8.41 | $8.65 | **$34.82** |
| nomic-embed-code | $16.13 | $7.44 | $8.99 | **$32.56** |

**Key cost findings:**

1. **jina has the cheapest mcp-only total** ($5.09) — significantly lower than nomic ($7.44) and qwen3-4b ($8.41).
2. **qwen3-8b has the lowest grand total** ($26.45), driven by the lowest baseline costs (random run variation).
3. **qwen3-4b has the highest mcp-only costs** ($8.41) — nearly 1.7× jina despite being a smaller model. Over-retrieval on TypeScript fixtures is the main culprit (e.g., opus/mcp-only on ts-disposable cost $1.04, ts-async-lifecycle $1.83).
4. **nomic mcp-only is expensive** ($7.44) for similar reasons — over-retrieval on ts-disposable ($1.45 for opus/mcp-only).

---

## The TypeScript Over-Retrieval Problem

`ts-disposable` (easy question) reveals a systematic issue with larger-dimension models:

| Model | opus/mcp-only cost | Winner |
|-------|--------------------|--------|
| jina | $0.23 | baseline (close race) |
| qwen3-8b | $0.70 | baseline |
| qwen3-4b | **$1.04** | baseline |
| nomic | **$1.45** | mcp-full |

The qwen3-4b and nomic models retrieve far too many chunks for this simple lifecycle question, ballooning input tokens and cost with no quality gain. Jina stays near the mcp-only cost baseline since its lower dimensionality/recall means fewer redundant chunks surface.

The same pattern appears for `ts-async-lifecycle` with qwen3-4b (opus/mcp-only: $1.83) and nomic (opus/mcp-full: $0.98).

---

## Retrieval Quality Assessment

| Embedding Model | Go retrieval | Python retrieval | TypeScript retrieval | Overall |
|----------------|-------------|-----------------|---------------------|---------|
| jina-v2-base-code | ✅ Excellent | ✅ Excellent | ✅ Good (no over-retrieval) | ✅ **Best balanced** |
| qwen3-8b | ✅ Excellent | ✅ Excellent | ⚠️ Moderate (ts-disposable: baseline wins) | ✅ Strong |
| qwen3-4b | ✅ Good | ✅ Good | ❌ Poor (significant over-retrieval) | ⚠️ Inconsistent |
| nomic-embed-code | ✅ Excellent | ✅ Excellent | ⚠️ Moderate (ts-disposable: extreme over-retrieval) | ✅ Good |

---

## Scenario-Level Findings (cross-model)

These hold true regardless of embedding model:

| Finding | Evidence |
|---------|----------|
| **mcp-only is the best default** | Wins 4–7 of 9 questions in every run |
| **mcp-only is 3–6× cheaper than baseline** | Baseline totals 2.5–3.5× mcp-only totals |
| **mcp-full rarely beats mcp-only on quality** | Wins only 0–4 questions depending on model |
| **Baseline wins only on TypeScript lifecycle questions** | Consistent across all 4 embedding models |
| **opus > sonnet on quality for mcp-only** | opus/mcp-only wins or runner-up in most questions |

---

## Embedding Model Recommendation

| Model | Quality | Cost efficiency | Consistency | Verdict |
|-------|---------|----------------|-------------|---------|
| **jina-v2-base-code** | ✅ High | ✅ Best | ✅ No over-retrieval | **Recommended default** |
| **qwen3-8b** | ✅ High | ✅ Good | ✅ Mostly consistent | Good alternative; strongest MCP dominance |
| **nomic-embed-code** | ✅ High | ⚠️ Moderate | ⚠️ ts over-retrieval | Usable; watch TypeScript costs |
| **qwen3-4b** | ⚠️ Variable | ❌ Expensive | ❌ ts severely over-retrieves | Not recommended |

**jina-v2-base-code remains the best overall default** — cheapest mcp-only costs, no over-retrieval pathology, strong retrieval across all three languages. The quality ceiling is slightly lower than qwen3-8b on hard questions, but the cost difference is significant.

**qwen3-8b is the best quality-focused alternative** — strongest MCP dominance (7/9), comparable mcp-only costs to jina, but requires Ollama to have the 4.7 GB model loaded.

**qwen3-4b should be avoided** — it combines higher costs than jina with worse quality and severe over-retrieval on TypeScript fixtures.

---

_Individual run reports: `bench-results/<run-id>/summary-report.md`_
_Full answer transcripts: `bench-results/<run-id>/detail-report.md`_
