# bench-swe — SWE-Bench Benchmarks for Lumen

Measures whether Lumen semantic search reduces token cost, time, and maintains
quality when Claude fixes real bugs.

## Quick Start

```bash
cd bench-swe
go build -o bench-swe .

# Run all tasks (requires Ollama + Claude CLI)
./bench-swe run

# Run single language
./bench-swe run --language go

# Skip preflight checks (faster iteration)
./bench-swe run --skip-preflight --language go

# Compare two runs
./bench-swe compare ../bench-results/swe-RUN1/ ../bench-results/swe-RUN2/

# Analyze chunker effectiveness
./bench-swe analyze ../bench-results/swe-RUN1/
```

## Scenarios

| Scenario       | MCP Tools                         | Built-in Tools                     |
| -------------- | --------------------------------- | ---------------------------------- |
| **baseline**   | None                              | All (Read, Edit, Write, Grep, ...) |
| **with-lumen** | `semantic_search`, `index_status` | All                                |

## Metrics

Per task x scenario:

- **Cost** (USD) — from Claude API
- **Duration** (ms)
- **Tokens** — input, cache read, cache created, output
- **Quality** — Poor / Good / Perfect (judged by Opus 4.6)

## Adding Tasks

### Task JSON format

Create `tasks/{language}/hard.json`:

```json
{
  "id": "go-hard",
  "language": "go",
  "repo": "https://github.com/owner/repo",
  "base_commit": "commit-before-fix",
  "fix_commit": "commit-with-fix",
  "issue_url": "https://github.com/owner/repo/issues/123",
  "issue_title": "Short description",
  "issue_body": "Full issue description...",
  "gold_patch_file": "patches/go-hard.patch",
  "expected_files": ["file.go"],
  "setup_commands": [],
  "test_command": "go test ./...",
  "timeout_s": 300
}
```

### Gold patch

Save the actual fix as a unified diff in `patches/`:

```bash
cd /tmp && git clone REPO && cd REPO
git diff BASE_COMMIT FIX_COMMIT > /path/to/bench-swe/patches/lang-hard.patch
```

### Source repos by language

| Language   | Repos                                           |
| ---------- | ----------------------------------------------- |
| Go         | gorilla/mux, spf13/cobra, gin-gonic/gin         |
| TypeScript | microsoft/TypeScript, denoland/deno_std         |
| JavaScript | expressjs/express, webpack/webpack              |
| Python     | SWE-bench dataset (django, scikit-learn, flask) |
| Rust       | BurntSushi/ripgrep, tokio-rs/tokio              |
| Ruby       | rails/rails, jekyll/jekyll                      |
| Java       | spring-projects/spring-boot, google/guava       |
| C          | redis/redis, curl/curl                          |
| C++        | nlohmann/json, opencv/opencv                    |
| PHP        | laravel/framework, symfony/symfony              |
| Markdown   | Any repo with README/CONTRIBUTING issues        |
| JSON       | npm packages with schema bugs                   |
| YAML       | Repos with GitHub Actions workflow bugs         |

### Curation checklist

- [ ] Real GitHub issue with clear problem statement
- [ ] Merged PR with the fix (= gold patch)
- [ ] Base commit is parent of fix commit
- [ ] Gold patch applies cleanly: `git checkout BASE && git apply PATCH`
- [ ] Issue body is self-contained
- [ ] Fix is reasonably deterministic
- [ ] Grep score < 50%: `./bench-swe validate` passes without REJECT
  - Issue body must not name the files or functions changed in the patch
  - Prefer issues that describe user-visible symptoms, not internal code
    locations

## Chunker Analysis

After running benchmarks:

```bash
./bench-swe analyze ../bench-results/swe-RUN1/
```

This parses raw Claude conversations to find:

- **Hit rate**: % of gold-patch files that Lumen's search returned
- **Noise rate**: % of search results not in the gold patch
- **Missed files**: What the chunker failed to surface

Recommendations must be **general** (apply to ANY codebase), never specific to
test repos.

## Results

Results are stored in `bench-results/swe-YYYYMMDD-HHMMSS-backend-model/` with:

- `*-raw.jsonl` — Full Claude stream output
- `*-patch.diff` — Claude's generated patch
- `*-metrics.json` — Cost, time, tokens
- `*-judge.json` — Quality rating
- `*-judge.md` — Judge explanation
- `summary-report.md` — Aggregate tables
- `detail-report.md` — Full patches and analysis
- `chunker-analysis.md` — Chunker effectiveness report
