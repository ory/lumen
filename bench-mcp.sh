#!/usr/bin/env bash
# bench-mcp.sh — benchmark baseline vs lumen MCP across questions and models
set -eufo pipefail

REPO="$(cd "$(dirname "$0")" && pwd)"
FIXTURES_GO="$REPO/testdata/fixtures/go"
FIXTURES_PY="$REPO/testdata/fixtures/python"
FIXTURES_TS="$REPO/testdata/fixtures/ts"
FIXTURES_JAVA="$REPO/testdata/fixtures/java"
FIXTURES_JS="$REPO/testdata/fixtures/js"
FIXTURES_RUBY="$REPO/testdata/fixtures/ruby"
FIXTURES_RUST="$REPO/testdata/fixtures/rust"
FIXTURES_PHP="$REPO/testdata/fixtures/php"
BINARY="$REPO/bin/lumen"

# ── Questions (8 languages × 1 hard question each) ───────────────────────────
QUESTIONS=(
  # Go (Prometheus TSDB fixtures)
  "How does TSDB compaction work end-to-end? Explain the Compactor interface, LeveledCompactor, and how the DB triggers compaction. Show relevant types, interfaces, and key method signatures."
  # Python (Django fixtures)
  "How does the Django QuerySet evaluation and filtering pipeline work? Explain QuerySet chaining, lazy evaluation, the Query class, how lookups and filters are compiled into SQL, and how the Manager ties it all together. Show key classes and method signatures."
  # TypeScript (VSCode base library fixtures)
  "How do Disposable and IDisposable work together with the event Emitter system? Explain the lifecycle management pattern, how listeners are registered and cleaned up, and how events are typed and fired. Show key interfaces and class relationships."
  # Java (Spring PetClinic fixtures)
  "How is the PetClinic domain model structured? Explain the entity hierarchy (Owner, Pet, Visit, Vet), how JPA/Hibernate maps the relationships, and how the repository layer exposes data access. Show key classes, annotations, and method signatures."
  # JavaScript (Express fixtures)
  "How does Express handle the full request/response lifecycle? Explain middleware chaining, how the Router works, how error-handling middleware differs from regular middleware, and how app.use and route mounting compose. Show key function signatures and flow."
  # Ruby (Rails fixtures)
  "How does the Rails middleware stack work? Explain how Rack middleware is assembled, how ActionDispatch integrates, how requests flow through the stack, and how custom middleware is added. Show key classes, modules, and call signatures."
  # Rust (ripgrep fixtures)
  "How does ripgrep's search pipeline work end-to-end? Explain the searcher/matcher/sink architecture, how the Searcher and Printer interact, and how results flow to the output layer. Show key traits, structs, and method signatures."
  # PHP (Laravel fixtures)
  "How does the Laravel service container resolve dependencies? Explain binding, contextual binding, automatic injection, how the container builds concrete classes, and how service providers register bindings. Show key classes, interfaces, and method signatures."
)
Q_SLUGS=(
  "go-tsdb-compaction"
  "py-django-queryset"
  "ts-disposable-events"
  "java-petclinic-domain"
  "js-express-lifecycle"
  "ruby-rails-middleware"
  "rust-ripgrep-pipeline"
  "php-laravel-container"
)
Q_LANG=(
  "go"
  "python"
  "typescript"
  "java"
  "javascript"
  "ruby"
  "rust"
  "php"
)
Q_FIXTURES=(
  "$FIXTURES_GO"
  "$FIXTURES_PY"
  "$FIXTURES_TS"
  "$FIXTURES_JAVA"
  "$FIXTURES_JS"
  "$FIXTURES_RUBY"
  "$FIXTURES_RUST"
  "$FIXTURES_PHP"
)
Q_DIFFICULTY=(
  "hard"       # Go: 32 files, 32K lines, deep cross-file TSDB architecture
  "hard"       # Python: 24 files, 17K lines, complex ORM query pipeline
  "medium"     # TypeScript: 24 files, 18K lines, focused event/disposable pattern
  "easy"       # Java: 20 files, 1.5K lines, small PetClinic domain model
  "medium"     # JavaScript: 20 files, 16K lines, Express middleware lifecycle
  "medium"     # Ruby: 13 files, 10K lines, Rails middleware stack
  "hard"       # Rust: 17 files, 15K lines, complex search pipeline architecture
  "medium"     # PHP: 23 files, 22K lines, well-scoped container pattern
)

# ── Models ────────────────────────────────────────────────────────────────────
MODELS=("haiku")
FILTER_MODELS=()
FILTER_QUESTIONS=()

# ── CLI flags ─────────────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case "$1" in
    --claude-model) FILTER_MODELS+=("$2");    shift 2 ;;
    --question)     FILTER_QUESTIONS+=("$2"); shift 2 ;;
    --embed-model)  EMBED_MODEL="$2";         shift 2 ;;
    *) echo "Unknown arg: $1"; exit 1 ;;
  esac
done

[[ ${#FILTER_MODELS[@]} -gt 0 ]] && MODELS=("${FILTER_MODELS[@]}")

# ── Embedding model / backend ──────────────────────────────────────────────
EMBED_MODEL="${EMBED_MODEL:-ordis/jina-embeddings-v2-base-code}"

# Auto-detect backend: nomic-ai/* → lmstudio, everything else → ollama
case "$EMBED_MODEL" in
  nomic-ai/*) EMBED_BACKEND="lmstudio" ;;
  *)          EMBED_BACKEND="ollama"   ;;
esac

# Model slug for directory name: text after last '/', colons replaced with '-'
MODEL_SLUG="${EMBED_MODEL##*/}"
MODEL_SLUG="${MODEL_SLUG//:/-}"

# Build filtered question index
Q_INDICES=()
for i in "${!Q_SLUGS[@]}"; do
  if [[ ${#FILTER_QUESTIONS[@]} -eq 0 ]]; then
    Q_INDICES+=("$i")
  else
    for fq in "${FILTER_QUESTIONS[@]}"; do
      if [[ "${Q_SLUGS[$i]}" == "$fq" ]]; then
        Q_INDICES+=("$i")
        break
      fi
    done
  fi
done

# ── Build ──────────────────────────────────────────────────────────────────────
echo "Building lumen..."
make build-local

# ── Isolate fixtures ─────────────────────────────────────────────────────────
# Copy fixtures to a temp directory so the evaluated model cannot read ground
# truth files (which live in testdata/ground-truth/ inside the repo).
BENCH_TMPDIR=$(mktemp -d)
echo "Isolating fixtures to $BENCH_TMPDIR ..."
for lang in go python ts java js ruby rust php; do
  cp -r "$REPO/testdata/fixtures/$lang" "$BENCH_TMPDIR/$lang"
done
FIXTURES_GO="$BENCH_TMPDIR/go"
FIXTURES_PY="$BENCH_TMPDIR/python"
FIXTURES_TS="$BENCH_TMPDIR/ts"
FIXTURES_JAVA="$BENCH_TMPDIR/java"
FIXTURES_JS="$BENCH_TMPDIR/js"
FIXTURES_RUBY="$BENCH_TMPDIR/ruby"
FIXTURES_RUST="$BENCH_TMPDIR/rust"
FIXTURES_PHP="$BENCH_TMPDIR/php"
Q_FIXTURES=(
  "$FIXTURES_GO" "$FIXTURES_PY" "$FIXTURES_TS" "$FIXTURES_JAVA"
  "$FIXTURES_JS" "$FIXTURES_RUBY" "$FIXTURES_RUST" "$FIXTURES_PHP"
)

# ── Index ─────────────────────────────────────────────────────────────────────
echo "Indexing fixtures..."
for fx_dir in "$FIXTURES_GO" "$FIXTURES_PY" "$FIXTURES_TS" "$FIXTURES_JAVA" "$FIXTURES_JS" "$FIXTURES_RUBY" "$FIXTURES_RUST" "$FIXTURES_PHP"; do
  LUMEN_BACKEND="$EMBED_BACKEND" LUMEN_EMBED_MODEL="$EMBED_MODEL" \
    bin/lumen index "$fx_dir" 2>&1 | tail -1
done

# ── MCP configs ───────────────────────────────────────────────────────────────
MCP_ENABLED=$(mktemp /tmp/bench-mcp-enabled-XXXXXX).json
MCP_EMPTY=$(mktemp /tmp/bench-mcp-empty-XXXXXX).json
trap 'rm -f "$MCP_ENABLED" "$MCP_EMPTY"; rm -rf "$BENCH_TMPDIR"' EXIT

cat > "$MCP_ENABLED" <<EOF
{"mcpServers":{"lumen":{"command":"$BINARY","args":["stdio"],"env":{"LUMEN_BACKEND":"$EMBED_BACKEND","LUMEN_EMBED_MODEL":"$EMBED_MODEL"}}}}
EOF
echo '{"mcpServers":{}}' > "$MCP_EMPTY"

# ── Results dir ───────────────────────────────────────────────────────────────
RESULTS_DIR="$REPO/bench-results/$(date +%Y%m%d-%H%M%S)-${EMBED_BACKEND}-${MODEL_SLUG}"
mkdir -p "$RESULTS_DIR"

# ── Run one scenario ──────────────────────────────────────────────────────────
run() {
  local mcp_cfg="$1" model="$2" q_idx="$3" scenario="$4" disable_builtin_tools="$5"
  local lang="${Q_LANG[$q_idx]}"
  local fixtures="${Q_FIXTURES[$q_idx]}"
  local slug="${Q_SLUGS[$q_idx]}-${model}-${scenario}"
  local prompt="The ${lang} project is at $fixtures. Answer this question about the code: ${QUESTIONS[$q_idx]}"
  local raw="$RESULTS_DIR/$slug-raw.jsonl"
  local answer_file="$RESULTS_DIR/$slug-answer.txt"

  printf "  %-28s %-12s %-10s ... " "${Q_SLUGS[$q_idx]}" "$model" "$scenario"

  local tools_arg=()
  [[ -n "$disable_builtin_tools" ]] && tools_arg=(--tools "")

  local allowed_tools_arg=()
  [[ "$mcp_cfg" == "$MCP_ENABLED" ]] && allowed_tools_arg=(--allowedTools "mcp__lumen__semantic_search,mcp__lumen__index_status")

  env -u CLAUDECODE claude \
    --output-format stream-json \
    --verbose \
    --model "$model" \
    --effort medium \
    --strict-mcp-config \
    --mcp-config "$mcp_cfg" \
    ${tools_arg[@]:+"${tools_arg[@]}"} \
    ${allowed_tools_arg[@]:+"${allowed_tools_arg[@]}"} \
    -p "$prompt" \
  > "$raw" 2>&1 || true

  # Strip home directory and username from raw snapshots to avoid PII in committed results
  [[ -f "$raw" ]] && sed -i '' -e "s|${HOME}|~|g" -e "s|${USER}|<user>|g" "$raw"

  local result_line
  result_line=$(grep -m1 '"type":"result"' "$raw" || true)
  if [[ -n "$result_line" ]]; then
    local cost duration_ms input_tokens cache_read cache_created output_tokens
    cost=$(echo "$result_line"          | jq -r '.total_cost_usd')
    duration_ms=$(echo "$result_line"   | jq -r '.duration_ms')
    input_tokens=$(echo "$result_line"  | jq -r '.usage.input_tokens // 0')
    cache_read=$(echo "$result_line"    | jq -r '.usage.cache_read_input_tokens // 0')
    cache_created=$(echo "$result_line" | jq -r '.usage.cache_creation_input_tokens // 0')
    output_tokens=$(echo "$result_line" | jq -r '.usage.output_tokens // 0')

    echo "$result_line" | jq -r '.result' | sed -e "s|${HOME}|~|g" -e "s|${USER}|<user>|g" > "$answer_file"
    echo "{\"cost_usd\":$cost,\"duration_ms\":$duration_ms,\"input_tokens\":$input_tokens,\"cache_read\":$cache_read,\"cache_created\":$cache_created,\"output_tokens\":$output_tokens}" \
      > "$RESULTS_DIR/$slug-metrics.json"

    local cost_fmt dur_s
    cost_fmt=$(printf "%.4f" "$cost")
    dur_s=$(echo "scale=1; $duration_ms/1000" | bc)
    printf "done  [%6ss  \$%s  in=%s+%scr  out=%s]\n" \
      "$dur_s" "$cost_fmt" "$input_tokens" "$cache_read" "$output_tokens"
  else
    echo "FAILED (no result event)"
  fi
}

# ── Extract winner from judge brief file ──────────────────────────────────────
extract_winner() {
  local brief_file="$1"
  grep -oE '\*\*Winner: [^*]+' "$brief_file" 2>/dev/null | sed 's/\*\*Winner: //' | tr -d ' \n' || echo "unknown"
}

# ── Extract runner-up from judge brief file ────────────────────────────────────
extract_runner_up() {
  local brief_file="$1"
  grep -oE '\*\*Runner-up: [^*]+' "$brief_file" 2>/dev/null \
    | sed 's/\*\*Runner-up: //' | tr -d ' \n' || echo "---"
}

# ── Run LLM judge for one question ────────────────────────────────────────────
run_judge() {
  local q_idx="$1"
  local slug="${Q_SLUGS[$q_idx]}"
  local question="${QUESTIONS[$q_idx]}"
  local judge_file="$RESULTS_DIR/$slug-judge.md"
  local judge_brief_file="$RESULTS_DIR/$slug-judge-brief.md"

  # Collect answers and metrics
  local all_answers=""
  local metrics_table="| Run | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) |"$'\n'
  metrics_table+="|----|----------|-----------|------------|------------|------------|"$'\n'
  local have_answers=false

  for model in "${MODELS[@]}"; do
    for scenario in baseline solo together; do
      local af="$RESULTS_DIR/${slug}-${model}-${scenario}-answer.txt"
      local mf="$RESULTS_DIR/${slug}-${model}-${scenario}-metrics.json"
      if [[ -f "$af" ]]; then
        have_answers=true
        all_answers+="
Answer [${model} / ${scenario}]:
$(cat "$af")
"
      fi
      if [[ -f "$mf" ]]; then
        local in_tok cr_tok out_tok cost_usd dur_ms cost_fmt dur_s
        in_tok=$(jq -r '.input_tokens'   "$mf")
        cr_tok=$(jq -r '.cache_read'     "$mf")
        out_tok=$(jq -r '.output_tokens' "$mf")
        cost_usd=$(jq -r '.cost_usd'     "$mf")
        dur_ms=$(jq -r '.duration_ms'    "$mf")
        cost_fmt=$(printf "%.5f" "$cost_usd")
        dur_s=$(echo "scale=1; $dur_ms/1000" | bc)
        metrics_table+="| ${model} / ${scenario} | ${dur_s}s | $in_tok | $cr_tok | $out_tok | \$$cost_fmt |"$'\n'
      fi
    done
  done

  if ! $have_answers; then
    return
  fi

  # Build fixture and ground truth context
  local fixture_dir="${Q_FIXTURES[$q_idx]}"
  local fixture_listing=""
  if [[ -d "$fixture_dir" ]]; then
    fixture_listing="## Available Source Files
$(ls -1 "$fixture_dir")
"
  fi

  local ground_truth=""
  local gt_file="$REPO/testdata/ground-truth/${slug}.md"
  if [[ -f "$gt_file" ]]; then
    ground_truth="## Ground Truth
$(cat "$gt_file")
"
  fi

  printf "  Judging %-28s ... " "$slug"

  # Brief verdict for summary (content quality + efficiency)
  env -u CLAUDECODE claude -p --model claude-opus-4-6 --effort medium \
    "You are a judge evaluating AI answers to a codebase question. Be concise.

Question: $question

$all_answers

Metrics:
$metrics_table

$fixture_listing
$ground_truth

Evaluate in three sections:

## Content Quality
For each answer [model/scenario], evaluate against the ground truth and source file listing:
1. **Fact coverage**: How many Required Facts from the ground truth does the answer address? Report as (X/Y).
2. **Accuracy**: Does the answer claim any types, functions, or signatures NOT listed in Key Types? List each fabrication.
3. **Hallucination traps**: Does the answer assert anything listed under Hallucination Traps in the ground truth? List each.
4. **Relationships**: Does the answer correctly identify cross-file architectural connections?
Rank answers from best to worst. One sentence summary per answer. Penalize fabrications heavily.
If no ground truth is provided, rank by correctness, completeness, and use of specific file/line references.

## Efficiency
One or two sentences comparing runtime, token usage, and cost across scenarios. Note which scenario offers the best quality-to-cost tradeoff.

## Verdict
On the very last two lines write exactly:
**Winner: model/scenario**
**Runner-up: model/scenario**
Choose based on answer quality first, then token usage, cost, and runtime. All three efficiency dimensions (tokens, cost, time) matter equally alongside quality.
Example:
**Winner: sonnet/solo**
**Runner-up: sonnet/together**" \
    > "$judge_brief_file" 2>&1 || echo "_Judge unavailable_" > "$judge_brief_file"

  # Detailed analysis for detail report
  env -u CLAUDECODE claude -p --model claude-opus-4-6 --effort medium \
    "You are a judge evaluating AI answers to a question about a codebase.

Question: $question

$all_answers

Metrics:
$metrics_table

$fixture_listing
$ground_truth

Evaluate in two sections:

## Content Quality
Rank the answers from best to worst. For each, write a paragraph covering:
1. **Fact coverage**: How many Required Facts from the ground truth does the answer address? Report as (X/Y).
2. **Accuracy**: Does the answer claim any types, functions, or signatures NOT listed in Key Types? List each fabrication explicitly.
3. **Hallucination traps**: Does the answer assert anything listed under Hallucination Traps in the ground truth? List each instance.
4. **Relationships**: Does the answer correctly identify the cross-file architectural connections described in the ground truth?
5. **File references**: Are file/line references precise and correct?
6. **Approach**: Did it use the right tools/strategy to find information?
Flag any fabricated code or incorrect type/signature claims — cross-check against the ground truth and available source files.
If no ground truth is provided, evaluate by correctness, completeness, and precision of references.

## Efficiency Analysis
Compare runtime, token usage, and cost across all scenarios. Identify which scenarios were most efficient, note any surprising differences, and recommend the best quality-to-cost tradeoff." \
    > "$judge_file" 2>&1 || echo "_Judge unavailable_" > "$judge_file"

  echo "done"
}

# ── Emit aggregate stats table across all questions ───────────────────────────
emit_overall_stats() {
  echo "## Overall: Aggregated by Scenario"
  echo ""
  echo "Totals across all ${#Q_INDICES[@]} questions × ${#MODELS[@]} models."
  echo ""
  echo "| Model | Scenario | Total Time | Total Input Tok | Total Output Tok | Total Cost (USD) |"
  echo "|-------|----------|------------|-----------------|------------------|------------------|"

  for model in "${MODELS[@]}"; do
    for scenario in baseline solo together; do
      local total_dur_ms=0 total_in=0 total_out=0 total_cost_scaled=0 count=0
      for q_idx in "${Q_INDICES[@]}"; do
        local mf="$RESULTS_DIR/${Q_SLUGS[$q_idx]}-${model}-${scenario}-metrics.json"
        if [[ -f "$mf" ]]; then
          total_dur_ms=$(( total_dur_ms + $(jq -r '.duration_ms'    "$mf") ))
          total_in=$(( total_in         + $(jq -r '.input_tokens'   "$mf") ))
          total_out=$(( total_out       + $(jq -r '.output_tokens'  "$mf") ))
          # cost: multiply by 100000 to keep integer arithmetic, divide at end
          local cost_scaled
          cost_scaled=$(jq -r '(.cost_usd * 100000) | round' "$mf")
          total_cost_scaled=$(( total_cost_scaled + cost_scaled ))
          (( count++ )) || true
        fi
      done
      if [[ $count -gt 0 ]]; then
        local dur_s cost_fmt
        dur_s=$(echo "scale=1; $total_dur_ms/1000" | bc)
        cost_fmt=$(printf "%.4f" "$(echo "scale=5; $total_cost_scaled/100000" | bc)")
        echo "| **$model** | $scenario | ${dur_s}s | $total_in | $total_out | \$$cost_fmt |"
      else
        echo "| **$model** | $scenario | — | — | — | — |"
      fi
    done
  done
  echo ""
}

# ── Emit overall algorithm comparison table ───────────────────────────────────
emit_overall_comparison() {
  echo "## Overall: Algorithm Comparison"
  echo ""
  echo "| Question | Language | Difficulty | 🏆 Winner | Runner-up |"
  echo "|----------|----------|------------|-----------|-----------|"

  local wins_baseline=0
  local wins_mcp_only=0
  local wins_mcp_full=0

  for q_idx in "${Q_INDICES[@]}"; do
    local slug="${Q_SLUGS[$q_idx]}"
    local difficulty="${Q_DIFFICULTY[$q_idx]}"
    local brief_file="$RESULTS_DIR/$slug-judge-brief.md"
    local winner="unknown"
    [[ -f "$brief_file" ]] && winner=$(extract_winner "$brief_file")

    # Tally wins per scenario
    local winner_scenario="${winner#*/}"
    if [[ -n "$winner_scenario" && "$winner" != "unknown" ]]; then
      case "$winner_scenario" in
        baseline) (( wins_baseline++ )) || true ;;
        solo) (( wins_mcp_only++ )) || true ;;
        together) (( wins_mcp_full++ )) || true ;;
      esac
    fi

    # Find runner-up: quality-ranked second place from judge brief
    local runner_up="---"
    [[ -f "$brief_file" ]] && runner_up=$(extract_runner_up "$brief_file")

    local lang="${Q_LANG[$q_idx]}"
    echo "| $slug | $lang | $difficulty | $winner | $runner_up |"
  done

  echo ""
  echo "**Scenario Win Counts** (across all ${#Q_INDICES[@]} questions):"
  echo ""
  echo "| Scenario | Wins |"
  echo "|----------|------|"

  local overall_winner_scenario=""
  local overall_winner_count=0

  for scenario in baseline solo together; do
    local wins
    case "$scenario" in
      baseline) wins=$wins_baseline ;;
      solo) wins=$wins_mcp_only ;;
      together) wins=$wins_mcp_full ;;
    esac
    echo "| $scenario | $wins |"
    if (( wins > overall_winner_count )); then
      overall_winner_count=$wins
      overall_winner_scenario=$scenario
    fi
  done

  echo ""
  if [[ -n "$overall_winner_scenario" && $overall_winner_count -gt 0 ]]; then
    echo "**Overall winner: $overall_winner_scenario** — won $overall_winner_count of ${#Q_INDICES[@]} questions."
  else
    echo "**Overall winner: undetermined** (no judge results available)."
  fi
  echo ""
}

# ── Generate reports ───────────────────────────────────────────────────────────
generate_reports() {
  local summary_file="$RESULTS_DIR/summary-report.md"
  local detail_file="$RESULTS_DIR/detail-report.md"

  # ── Summary report ────────────────────────────────────────────────────────
  {
    echo "# Benchmark Summary"
    echo ""
    echo "Generated: $(date -u '+%Y-%m-%d %H:%M UTC')  |  Results: \`$(basename "$RESULTS_DIR")\`"
    echo ""
    echo "| Scenario | Description |"
    echo "|----------|-------------|"
    echo "| **baseline** | All default Claude tools, no MCP |"
    echo "| **solo** | \`semantic_search\` MCP tool only |"
    echo "| **together** | All default tools + MCP |"
    echo ""
    emit_overall_stats
    echo "---"
    echo ""

    for q_idx in "${Q_INDICES[@]}"; do
      local slug="${Q_SLUGS[$q_idx]}"
      local difficulty="${Q_DIFFICULTY[$q_idx]}"
      local question="${QUESTIONS[$q_idx]}"

      local lang="${Q_LANG[$q_idx]}"
      echo "## $slug [$lang / $difficulty]"
      echo ""
      echo "> $question"
      echo ""
      echo "### Time & Tokens"
      echo ""
      echo "| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |"
      echo "|-------|----------|----------|-----------|------------|------------|------------|--------|"

      local judge_brief_file="$RESULTS_DIR/$slug-judge-brief.md"
      local winner=""
      [[ -f "$judge_brief_file" ]] && winner=$(extract_winner "$judge_brief_file")

      for model in "${MODELS[@]}"; do
        for scenario in baseline solo together; do
          local metrics_file="$RESULTS_DIR/${slug}-${model}-${scenario}-metrics.json"
          if [[ -f "$metrics_file" ]]; then
            local in_tok cr_tok out_tok cost_usd dur_ms cost_fmt dur_s badge
            in_tok=$(jq -r '.input_tokens'   "$metrics_file")
            cr_tok=$(jq -r '.cache_read'     "$metrics_file")
            out_tok=$(jq -r '.output_tokens' "$metrics_file")
            cost_usd=$(jq -r '.cost_usd'     "$metrics_file")
            dur_ms=$(jq -r '.duration_ms'    "$metrics_file")
            cost_fmt=$(printf "%.4f" "$cost_usd")
            dur_s=$(echo "scale=1; $dur_ms/1000" | bc)
            local run_key="${model}/${scenario}"
            badge=""
            [[ -n "$winner" && "$winner" == "$run_key" ]] && badge="🏆 Winner"
            echo "| **$model** | $scenario | ${dur_s}s | $in_tok | $cr_tok | $out_tok | \$$cost_fmt | $badge |"
          else
            echo "| **$model** | $scenario | — | — | — | — | — | |"
          fi
        done
      done

      echo ""

      if [[ -f "$judge_brief_file" ]]; then
        echo "### Quality Ranking (Opus 4.6)"
        echo ""
        cat "$judge_brief_file"
        echo ""
      fi

      echo "---"
      echo ""
    done

    emit_overall_comparison
    echo "_Full answers and detailed analysis: \`detail-report.md\`_"
  } > "$summary_file"

  # ── Detail report ─────────────────────────────────────────────────────────
  {
    echo "# Benchmark Detail Report"
    echo ""
    echo "Generated: $(date -u '+%Y-%m-%d %H:%M UTC')  |  Results: \`$(basename "$RESULTS_DIR")\`"
    echo ""

    for q_idx in "${Q_INDICES[@]}"; do
      local slug="${Q_SLUGS[$q_idx]}"
      local difficulty="${Q_DIFFICULTY[$q_idx]}"
      local question="${QUESTIONS[$q_idx]}"

      echo "---"
      echo ""
      local lang="${Q_LANG[$q_idx]}"
      echo "## $slug [$lang / $difficulty]"
      echo ""
      echo "**Question:** $question"
      echo ""

      echo "### Metrics"
      echo ""
      echo "| Model | Scenario | Duration | Input Tok | Cache Read | Cache Created | Output Tok | Cost (USD) |"
      echo "|-------|----------|----------|-----------|------------|---------------|------------|------------|"

      for model in "${MODELS[@]}"; do
        for scenario in baseline solo together; do
          local metrics_file="$RESULTS_DIR/${slug}-${model}-${scenario}-metrics.json"
          if [[ -f "$metrics_file" ]]; then
            local in_tok cr_tok cc_tok out_tok cost_usd dur_ms cost_fmt dur_s
            in_tok=$(jq -r '.input_tokens'    "$metrics_file")
            cr_tok=$(jq -r '.cache_read'      "$metrics_file")
            cc_tok=$(jq -r '.cache_created'   "$metrics_file")
            out_tok=$(jq -r '.output_tokens'  "$metrics_file")
            cost_usd=$(jq -r '.cost_usd'      "$metrics_file")
            dur_ms=$(jq -r '.duration_ms'     "$metrics_file")
            cost_fmt=$(printf "%.5f" "$cost_usd")
            dur_s=$(echo "scale=1; $dur_ms/1000" | bc)
            echo "| **$model** | $scenario | ${dur_s}s | $in_tok | $cr_tok | $cc_tok | $out_tok | \$$cost_fmt |"
          else
            echo "| **$model** | $scenario | — | — | — | — | — | — |"
          fi
        done
      done

      echo ""

      for model in "${MODELS[@]}"; do
        for scenario in baseline solo together; do
          local af="$RESULTS_DIR/${slug}-${model}-${scenario}-answer.txt"
          if [[ -f "$af" ]]; then
            echo "### Answer: \`$model\` / \`$scenario\`"
            echo ""
            cat "$af"
            echo ""
          fi
        done
      done

      local judge_file="$RESULTS_DIR/$slug-judge.md"
      if [[ -f "$judge_file" ]]; then
        echo "### Full Judge Analysis (Opus 4.6)"
        echo ""
        cat "$judge_file"
        echo ""
      fi
    done
  } > "$detail_file"

  echo ""
  echo "Reports written:"
  echo "  Summary : $summary_file"
  echo "  Detail  : $detail_file"
}

# ── Main loop ─────────────────────────────────────────────────────────────────
echo ""
echo "Running benchmarks (${#MODELS[@]} models × ${#Q_INDICES[@]} questions × 3 scenarios)..."
echo ""

for model in "${MODELS[@]}"; do
  echo "── Model: $model ──────────────────────────────────────────"
  for q_idx in "${Q_INDICES[@]}"; do
    _t1=$(mktemp) _t2=$(mktemp) _t3=$(mktemp)
    run "$MCP_EMPTY"   "$model" "$q_idx" "baseline" ""  >"$_t1" 2>&1 & _p1=$!
    run "$MCP_ENABLED" "$model" "$q_idx" "solo"  "1" >"$_t2" 2>&1 & _p2=$!
    run "$MCP_ENABLED" "$model" "$q_idx" "together"  ""  >"$_t3" 2>&1 & _p3=$!
    wait "$_p1" || true; cat "$_t1"; rm -f "$_t1"
    wait "$_p2" || true; cat "$_t2"; rm -f "$_t2"
    wait "$_p3" || true; cat "$_t3"; rm -f "$_t3"
  done
  echo ""
done

echo "── Generating LLM judge reports ──────────────────────────────"
_judge_pids=() _judge_tmps=()
for q_idx in "${Q_INDICES[@]}"; do
  _jt=$(mktemp)
  _judge_tmps+=("$_jt")
  run_judge "$q_idx" >"$_jt" 2>&1 &
  _judge_pids+=($!)
done
for _ji in "${!_judge_pids[@]}"; do
  wait "${_judge_pids[$_ji]}" || true
  cat "${_judge_tmps[$_ji]}"
  rm -f "${_judge_tmps[$_ji]}"
done

generate_reports
