---
name: add-benchmark
description:
  Add a new SWE benchmark task from a real GitHub bug-fix. Use when the user provides a GitHub
  issue or PR URL and wants to add it to the bench-swe pipeline.
argument-hint: <github-issue-or-pr-url> <language> <difficulty>
disable-model-invocation: true
---

# Add SWE Benchmark

Add a new benchmark task to the bench-swe pipeline from a real GitHub bug-fix.
The human provides the GitHub issue or PR URL; the agent handles extraction,
validation, and file creation.

## Arguments

- **url** (required): GitHub issue or PR URL
  (e.g. `https://github.com/gorilla/mux/issues/534` or
  `https://github.com/gorilla/mux/pull/585`)
- **language** (required): One of: go, python, typescript, javascript, rust,
  ruby, java, c, cpp, php, csharp
- **difficulty** (required): easy, medium, or hard

## Steps

1. Dispatch the `task-curator` agent with the provided arguments. The agent
   will:
   - Validate inputs (URL, language, difficulty)
   - Resolve the fix PR (from issue or directly)
   - Clone the repo, extract base/fix commits, and generate the gold patch
   - Determine the test command from repo conventions
   - Write task JSON to `bench-swe/tasks/{language}/` and patch to
     `bench-swe/patches/`
   - Run 6 inline verification checks (patch applies, files match, no leaks,
     difficulty calibration, schema completeness, no test files in patch)
   - Fix any issues found during verification

2. Report the result including:
   - Task ID, repo, issue URL
   - Files and lines changed
   - Verification table
