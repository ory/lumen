# bench-swe — SWE-Bench Benchmarks for Lumen

Measures whether Lumen retrieval reduces tokens and time while maintaining fix
quality on real bugs. Both **Claude Code and Codex** can solve and judge tasks,
through either their native CLI or their official agent SDK.

## Quick Start

```bash
cd bench-swe
go build -o bench-swe .

# Existing CLI login is reused; BM25 needs no Ollama/model download.
./bench-swe run --agent claude --language go
./bench-swe run --agent codex --language go

# Optional official SDK runners (Node.js 18+).
npm ci
./bench-swe run --agent claude --runner sdk --language go
./bench-swe run --agent codex --runner sdk --language go

# Pin a model for reproducible comparisons; otherwise use Claude's haiku
# alias or Codex's built-in default (--ignore-user-config skips user settings).
./bench-swe run --agent codex --model YOUR_MODEL --language go

# CLI end-to-end gate for BOTH providers, independent of the SDK runner.
./bench-swe smoke --agent all
# Also check the SDK path, using the same fixture and assertions.
./bench-swe smoke --agent all --runner sdk

./bench-swe compare ../bench-results/swe-RUN1/ ../bench-results/swe-RUN2/
./bench-swe analyze ../bench-results/swe-RUN1/
./bench-swe extract ../bench-results/swe-RUN1/go-hard-with-lumen-raw.jsonl --search-only
```

`--agent` defaults to `claude`; `--runner` defaults to `cli`. The existing
`--claude-model` option remains supported; `--model` takes precedence.
The judge uses the selected provider and transport: Claude defaults to Sonnet,
Codex to the task model/client default. Override with `--judge-model`, or disable
judging with `--skip-judge`. Codex runs do not require Claude to be installed.

## Authentication and SDKs

`--auth subscription` is the default. Log in with `claude auth login` or
`codex login` first. The harness checks the login method, removes inherited API
key/provider overrides, and reuses the clients' existing credential stores.
It never copies OAuth tokens or creates new credentials. An API-backed login is
rejected before starting a turn. Use `--auth environment` explicitly to inherit
API/provider authentication instead.

Anthropic's [current plan guidance](https://support.claude.com/en/articles/15036540-use-the-claude-agent-sdk-with-your-claude-plan)
states that Claude Agent SDK and `claude -p` can use subscription limits.
Codex supports [cached ChatGPT subscription login](https://learn.chatgpt.com/docs/auth).
The SDK runners use the installed `claude`/`codex` executables so CLI and SDK
comparisons use the same agent runtime. SDK packages are pinned in the lockfile.
Use a current Codex CLI supporting `--ignore-user-config`, `--ignore-rules`,
`--ephemeral`, and JSON MCP tool events (validated with 0.162.1).

The SDK bridge provides structured MCP configuration and event streams; it does
not change the model's reasoning or guarantee a better fix. The Codex TypeScript
SDK itself wraps `codex exec`. Both native event formats are saved unchanged apart
from home-directory redaction and normalized by the Go analysis layer.

## Scenarios and isolation

| Scenario | MCP tools | Built-in tools |
| --- | --- | --- |
| **baseline** | None | Agent's normal tools |
| **with-lumen** | `semantic_search`, `index_status` | Agent's normal tools |

Both transports supply explicit per-run MCP configuration. Claude uses strict MCP
configuration, excludes filesystem settings, and loads Lumen's local plugin for
its hooks. Codex ignores user configuration/rules and supplies an explicit server
and tool allowlist. Its SDK executable adapter adds the same isolation flags while
preserving the existing login store. Managed policies still apply.

Claude uses `acceptEdits` with explicit tool permissions. Codex uses
`workspace-write` with no interactive approvals and network access for task
commands. Each task runs in a disposable checkout. Codex receives explicit Lumen
search guidance because it does not run Claude's hooks; compare retrieval
scenarios within the same provider/model/transport.

## End-to-end checks and failures

Preflight always runs **CLI** baseline and Lumen smoke tests for the selected
provider, even for `--runner sdk`. `--skip-preflight` is available for iteration.
The correct noninteractive commands are `claude --print` and `codex exec --json`;
Codex's `-p` selects a profile, not print mode.

The smoke fixture requires the agent to fix a Go retry-budget bug. With Lumen,
the raw event log must contain a **completed, successful `semantic_search` call**
whose result includes the fixture source file. Mentioning a tool in prose,
listing available tools, an unfinished call, or a tool error cannot pass. The
harness then installs a trusted regression test after the agent exits, checks
that a patch exists, and runs the test. Baseline rejects observed MCP calls.

Smoke tests consume subscription usage and require network access to the agent
providers. They are explicit commands, not part of `go test ./...`.

Nonzero exits, timeouts, failed turns and missing completion events produce
failure metrics and a nonzero benchmark exit. On Unix, timeout cancellation kills
the SDK/CLI process group, including its tools. Failed runs and Lumen runs without
a successful search are excluded from valid aggregates. Raw streams, stderr and
partial patches are retained. Task test pass/fail/timeout status is saved in metrics and the detail report.
Task test output is evidence separate from the LLM
judge; a judge rating is not a substitute for a regression test.

## Metrics

Per task × scenario:

- **Agent, runner, model**, plus a run configuration manifest.
- **Reported cost** when available. Codex reports `n/a`, never a fabricated zero.
  Claude's reported dollar amount is not a subscription invoice.
- **Duration**, using wall time when the client has no duration field.
- **Tokens**: uncached input, cache read, cache creation and output. Codex's cached
  input is subtracted from its inclusive input total to avoid double counting.
- **Quality**: Poor / Good / Perfect from the selected provider's judge.
- **Tool calls** and whether Lumen search succeeded.

Multiple runs use the median by reported cost, or by total tokens when cost is
unavailable. Use matching agents/models/transports when measuring Lumen's effect.

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

This parses raw Claude and Codex conversations to find:

- **Hit rate**: % of gold-patch files that Lumen's search returned
- **Noise rate**: % of search results not in the gold patch
- **Missed files**: What the chunker failed to surface

Recommendations must be **general** (apply to ANY codebase), never specific to
test repos.

## Results

Results are stored in `bench-results/swe-YYYYMMDD-HHMMSS-agent-runner-retrieval/` with:

- `*-raw.jsonl` — Native CLI/SDK event stream
- `*-patch.diff` — Agent's generated patch
- `*-metrics.json` — Agent identity, cost availability, time, tokens, execution errors
- `*-stderr.log` — Client diagnostics
- `run-config.json` — Selected agent/runner/auth/model and retrieval settings
- `*-judge.json` — Quality rating
- `*-judge.md` — Judge explanation
- `summary-report.md` — Aggregate tables
- `detail-report.md` — Full patches and analysis
- `chunker-analysis.md` — Chunker effectiveness report

Run deterministic harness tests with `go test ./...` from `bench-swe/`.
